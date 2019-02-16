/**
 * Copyright (c) 2019, The Artemis Authors.
 *
 * Permission to use, copy, modify, and/or distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copies.
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */

package handler

import (
	"errors"
	"sync"
	"unsafe"

	"github.com/botobag/artemis/graphql/executor"

	"github.com/willf/bitset"
)

// OperationCache caches executor.PreparedOperation created from a query to save parsing efforts.
type OperationCache interface {
	// Get looks up operation for the given query.
	Get(query string) (operation *executor.PreparedOperation, ok bool)

	// Add adds an operation that associated with the query to the cache.
	Add(query string, operation *executor.PreparedOperation)
}

type lruEntry struct {
	query     string
	operation *executor.PreparedOperation

	// Next and previous pointers in the doubly-linked list of elements. To simplify the
	// implementation, internally a list l is implemented as a ring, such that &l.root is both the
	// next element of the last list element (l.Back()) and the previous element of the first list
	// element (l.Front()).
	next, prev *lruEntry
}

const sizeOfLRUEntry = unsafe.Sizeof(lruEntry{})

type lruEntryAllocator struct {
	entries []lruEntry
	// Allocated entries will have their corresponding bits set in the bitset.
	allocated bitset.BitSet
}

func newLRUEntryAllocator(maxEntries uint) lruEntryAllocator {
	allocator := lruEntryAllocator{
		entries: make([]lruEntry, maxEntries),
	}

	// Set the most significant bit to trigger the allocation.
	allocator.allocated.Set(maxEntries - 1)

	return allocator
}

// New allocates an entry to store given query and operation. It panics if there's no any entry
// available to allocate.
func (allocator *lruEntryAllocator) New(query string, operation *executor.PreparedOperation) *lruEntry {
	allocated := &allocator.allocated

	// Search allocated to find an unused entry.
	i, found := allocated.NextSet(0)
	if !found {
		panic("LRUOperationCache: no available entry to return")
	}

	// Reserve the entry.
	entry := &allocator.entries[i]
	allocated.Set(i)

	entry.query = query
	entry.operation = operation

	return entry
}

func (allocator *lruEntryAllocator) indexOf(entry *lruEntry) uint {
	entryAddr := uintptr(unsafe.Pointer(entry))
	firstEntryAddr := uintptr(unsafe.Pointer(&allocator.entries[0]))
	return uint((entryAddr - firstEntryAddr) / sizeOfLRUEntry)
}

// Free deallocates the entry. It doesn't free the memory (in fact we're unable to do that.)
// Instead, it marks the entry to be free for later reuse.
func (allocator *lruEntryAllocator) Free(entry *lruEntry) {
	// Clear reference.
	entry.query = ""
	entry.operation = nil
	// Find the index of the given entry from its address.
	i := allocator.indexOf(entry)
	// Unset the bit in allocated.
	allocator.allocated.Clear(i)
}

// lruEvictList is a doubly linked list that maintains eviction list for LRUOperationCache. Its
// implementation mirrors from container/list [0] and only provides operation used by
// LRUOperationCache.
//
// [0]: https://go.googlesource.com/go/+/5bc1fd4/src/container/list/list.go
type lruEvictList struct {
	// Allocator that manages allocation and deallocation for the entry
	allocator lruEntryAllocator

	// sentinel list element, only &root, root.prev, and root.next are used
	root lruEntry

	// current list length excluding (this) sentinel element
	len uint
}

func newLRUEvictList(maxEntries uint) *lruEvictList {
	l := &lruEvictList{
		allocator: newLRUEntryAllocator(maxEntries),
	}
	l.root.next = &l.root
	l.root.prev = &l.root
	return l
}

// Len returns the number of elements of list l.
// The complexity is O(1).
func (l *lruEvictList) Len() uint { return l.len }

// Back returns the last element of list l or nil if the list is empty.
func (l *lruEvictList) Back() *lruEntry {
	if l.len == 0 {
		return nil
	}
	return l.root.prev
}

// insert inserts e after at, increments l.len, and returns e.
func (l *lruEvictList) insert(e, at *lruEntry) *lruEntry {
	n := at.next
	at.next = e
	e.prev = at
	e.next = n
	n.prev = e
	l.len++
	return e
}

// insertEntry is a convenience wrapper that for insert(l.allocator.New(query, operation), at)
func (l *lruEvictList) insertEntry(query string, operation *executor.PreparedOperation, at *lruEntry) *lruEntry {
	return l.insert(l.allocator.New(query, operation), at)
}

// remove removes e from its list, decrements l.len, and notifies allocator to mark it as free.
func (l *lruEvictList) remove(e *lruEntry) {
	e.prev.next = e.next
	e.next.prev = e.prev
	e.next = nil // avoid memory leaks
	e.prev = nil // avoid memory leaks
	l.len--
	l.allocator.Free(e)
}

// move moves e to next to at and returns e.
func (l *lruEvictList) move(e, at *lruEntry) *lruEntry {
	if e == at {
		return e
	}
	e.prev.next = e.next
	e.next.prev = e.prev

	n := at.next
	at.next = e
	e.prev = at
	e.next = n
	n.prev = e

	return e
}

// Remove removes e from l if e is an element of list l.
// The given entry must not be nil.
func (l *lruEvictList) Remove(e *lruEntry) {
	l.remove(e)
}

// PushFront inserts an new entry e with given values at the front of list l and returns e.
func (l *lruEvictList) PushFront(query string, operation *executor.PreparedOperation) *lruEntry {
	return l.insertEntry(query, operation, &l.root)
}

// MoveToFront moves element e to the front of list l.
// If e is not an element of l, the list is not modified.
// The element must not be nil.
func (l *lruEvictList) MoveToFront(e *lruEntry) {
	if l.root.next == e {
		return
	}
	// see comment in List.Remove about initialization of l
	l.move(e, &l.root)
}

// LRUOperationCache is a thread-safe LRU cache that implements OperationCache. It serves as default
// operation cache for LLHandler. Most part of implementation directly derived from groupcache/lru
// [0] with sync.RWLock added to make it safe for concurrent access.
type LRUOperationCache struct {
	// The maximum number of cache operations before an item is evicted. It must be greater than 0.
	maxEntries uint

	// m guards cache and evictList.
	m         sync.Mutex
	cache     map[string]*lruEntry
	evictList *lruEvictList
}

var _ OperationCache = (*LRUOperationCache)(nil)

var errZeroCacheSize = errors.New("LRUOperationCache: must specified a non-zero cache size")

// NewLRUOperationCache creates a new LRUOperationCache with given size.
func NewLRUOperationCache(maxEntries uint) (*LRUOperationCache, error) {
	if maxEntries <= 0 {
		return nil, errZeroCacheSize
	}

	return &LRUOperationCache{
		maxEntries: maxEntries,
		cache:      make(map[string]*lruEntry, maxEntries),
		evictList:  newLRUEvictList(maxEntries),
	}, nil
}

// Get implements OperationCache.
func (c *LRUOperationCache) Get(query string) (operation *executor.PreparedOperation, ok bool) {
	var (
		m         = &c.m
		cache     = c.cache
		evictList = c.evictList
	)

	m.Lock()

	if entry, hit := cache[query]; hit {
		evictList.MoveToFront(entry)
		// Set up return values.
		operation = entry.operation
		ok = true
	}

	m.Unlock()
	return
}

// Add implements OperationCache.
func (c *LRUOperationCache) Add(query string, operation *executor.PreparedOperation) {
	var (
		m         = &c.m
		cache     = c.cache
		evictList = c.evictList
	)

	m.Lock()
	if e, ok := cache[query]; ok {
		evictList.MoveToFront(e)
		e.operation = operation
		m.Unlock()
		return
	}

	if evictList.Len() > c.maxEntries {
		c.removeOldest()
	}
	e := evictList.PushFront(query, operation)
	cache[query] = e

	m.Unlock()
}

// removeOldest removes the oldest entry from the cache.
func (c *LRUOperationCache) removeOldest() {
	var (
		m         = &c.m
		cache     = c.cache
		evictList = c.evictList
	)

	m.Lock()
	e := c.evictList.Back()
	if e != nil {
		key := e.query
		evictList.Remove(e)
		delete(cache, key)
	}
	m.Unlock()
}

// NopOperationCache does nothing.
type NopOperationCache struct{}

var _ OperationCache = NopOperationCache{}

// Get implements OperationCache.
func (NopOperationCache) Get(query string) (operation *executor.PreparedOperation, ok bool) {
	return
}

// Add implements OperationCache.
func (NopOperationCache) Add(query string, operation *executor.PreparedOperation) {}
