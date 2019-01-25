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

package dataloader

import (
	"github.com/botobag/artemis/internal/util"
)

// CacheMap defines interfaces required by DataLoader to cache task that loads a value identified by
// key. Note that all methods must be safe for concurrent use by multiple goroutines.
type CacheMap interface {
	// Get returns the value stored in the cache for a key or nil if no value is present.
	Get(key Key) *Task

	// Set caches the task. If the key associated with the given task exists, return the task that was
	// previously set. Otherwise, the given task is added to the cache and return.
	Set(task *Task) *Task

	// Delete deletes cache task for loading value at given key.
	Delete(key Key)

	// Clear reset the cache.
	Clear()
}

//===----------------------------------------------------------------------------------------====//
// DefaultCacheMap
//===----------------------------------------------------------------------------------------====//

// DefaultCacheMap is used when Config.CacheMap is not set. It uses util.SyncMap (equivalent to
// sync.Map in Go 1.9+).
type DefaultCacheMap struct {
	m util.SyncMap
}

var _ CacheMap = (*DefaultCacheMap)(nil)

// Get implements CacheMap.
func (cacheMap *DefaultCacheMap) Get(key Key) *Task {
	task, ok := cacheMap.m.Load(key)
	if !ok {
		return nil
	}
	return task.(*Task)
}

// Set implements CacheMap.
func (cacheMap *DefaultCacheMap) Set(task *Task) *Task {
	t, _ := cacheMap.m.LoadOrStore(task.Key(), task)
	return t.(*Task)
}

// Delete implements CacheMap.
func (cacheMap *DefaultCacheMap) Delete(key Key) {
	cacheMap.m.Delete(key)
}

// Clear implements CacheMap.
func (cacheMap *DefaultCacheMap) Clear() {
	m := &cacheMap.m
	m.Range(func(key, _ interface{}) bool {
		m.Delete(key)
		return true
	})
}

//===----------------------------------------------------------------------------------------====//
// CustomKeyCacheMap
//===----------------------------------------------------------------------------------------====//

// KeyWithCustomCacheKey is a Key that uses custom key for cache.
type KeyWithCustomCacheKey interface {
	Key
	KeyForCache() interface{}
}

// CustomKeyCacheMap wraps a DefaultCacheMap and requires task key to implement
// KeyWithCustomCacheKey to specify cache key.
type CustomKeyCacheMap struct {
	DefaultCacheMap
}

func (cacheMap *CustomKeyCacheMap) cacheKeyFor(key Key) Key {
	return Key(key.(KeyWithCustomCacheKey).KeyForCache())
}

// Get implements CacheMap.
func (cacheMap *CustomKeyCacheMap) Get(key Key) *Task {
	return cacheMap.DefaultCacheMap.Get(cacheMap.cacheKeyFor(key))
}

// Set implements CacheMap.
func (cacheMap *CustomKeyCacheMap) Set(task *Task) *Task {
	t, _ := cacheMap.m.LoadOrStore(cacheMap.cacheKeyFor(task.Key()), task)
	return t.(*Task)
}

// Delete implements CacheMap.
func (cacheMap *CustomKeyCacheMap) Delete(key Key) {
	cacheMap.DefaultCacheMap.Delete(cacheMap.cacheKeyFor(key))
}

//===----------------------------------------------------------------------------------------====//
// NoCacheMap
//===----------------------------------------------------------------------------------------====//

// noCacheMap serves as type for NoCacheMap.
type noCacheMap int

var _ CacheMap = NoCacheMap

// Get implements CacheMap.
func (noCacheMap) Get(key Key) *Task {
	return nil
}

// Set implements CacheMap.
func (noCacheMap) Set(task *Task) *Task {
	return nil
}

// Delete implements CacheMap.
func (noCacheMap) Delete(key Key) {}

// Clear implements CacheMap.
func (noCacheMap) Clear() {}

// NoCacheMap is a placeholder given to Config.CacheMap to disable cache for a DataLoader.
const NoCacheMap noCacheMap = 0
