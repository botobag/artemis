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

package executor

import (
	"unsafe"
)

// Number of ResultNode's to be allocated in each chunk for a ResultNodeList with unknown size on
// creation
const fixedResultNodeListChunkSize = 16

// ResultNodeListChunk is element type of ResultNodeList. It carries an array of ResultNode's.
type ResultNodeListChunk struct {
	nodes []ResultNode
	// Point to the chunk in prior to this one in the list. For the first chunk in the list, this
	// points to the last chunk.
	prev *ResultNodeListChunk
	// Point to the chunk next to this one in the list. For the last chunk in the list, this points to
	// the first chunk.
	next *ResultNodeListChunk
}

// Nodes returns the chunk.nodes.
func (chunk *ResultNodeListChunk) Nodes() []ResultNode {
	return chunk.nodes
}

// Prev returns the chunk.prev.
func (chunk *ResultNodeListChunk) Prev() *ResultNodeListChunk {
	return chunk.prev
}

// Next returns the chunk.next.
func (chunk *ResultNodeListChunk) Next() *ResultNodeListChunk {
	return chunk.next
}

// Size returns number of nodes currently in the chunk.
func (chunk *ResultNodeListChunk) Size() int {
	return len(chunk.nodes)
}

// cap returns the available capacity of nodes allocation in the chunk.
func (chunk *ResultNodeListChunk) cap() int {
	return cap(chunk.nodes)
}

// ResultNodeList implements a container for storing list of ResultNode's. Under the hood, it is a
// circular doubly linked list in which each element in the list is a pre-allocated ResultNode chunk
// (i.e., ResultNodeListChunk). Currently it is used for storing ResultNode's for elements in List
// fields.
//
// Unlike []ResultNode, appending new node never triggers reallocation. Its address is fixed after
// creation which is essential if we want to iterate through element values in the List (i.e.,
// ExecuteNodeTask.completeWrappingValue) while processing the already-read value (with
// task.completeNonWrappingValue) in parallel. (Because we need to set the result node list to
// List field's ResultNode.Value before diving into completing its elements.)
//
// The array-like element allows us to compute the index for a ResultNode in the list fast (required
// for error reporting) using their address.
type ResultNodeList struct {
	// The first chunk in the list
	chunks *ResultNodeListChunk
}

// NewResultNodeList creates a ResultNodeList.
func NewResultNodeList() ResultNodeList {
	return NewFixedSizeResultNodeList(fixedResultNodeListChunkSize)
}

// NewFixedSizeResultNodeList creates a ResultNodeList that can store ResultNode no more than the
// given n.
func NewFixedSizeResultNodeList(n int) ResultNodeList {
	firstChunk := &ResultNodeListChunk{
		nodes: make([]ResultNode, 0, n),
	}
	firstChunk.prev = firstChunk
	firstChunk.next = firstChunk
	return ResultNodeList{
		chunks: firstChunk,
	}
}

// Chunks returns the head chunk in the list.
func (list ResultNodeList) Chunks() *ResultNodeListChunk {
	return list.chunks
}

// Empty returns true if the list contains no any result nodes.
func (list ResultNodeList) Empty() bool {
	firstChunk := list.chunks
	// List contains only one chunk and no any node is in use in the chunk.
	return firstChunk.next == firstChunk && firstChunk.Size() == 0
}

// EmplaceBack extends the list by inserting a new ResultNode at the end of list.
func (list ResultNodeList) EmplaceBack(parent *ResultNode, nullable bool) *ResultNode {
	var (
		firstChunk    = list.chunks
		lastChunk     = firstChunk.prev
		lastChunkSize = lastChunk.Size()
	)
	if lastChunkSize >= lastChunk.cap() {
		// We run out of nodes in current chunk. Allocate a new chunk.
		newChunk := &ResultNodeListChunk{
			nodes: make([]ResultNode, 0, fixedResultNodeListChunkSize),
			prev:  lastChunk,
			next:  lastChunk.next,
		}

		// Update link to append the newChunk at the end of list.
		lastChunk.next = newChunk
		firstChunk.prev = newChunk

		// Switch lastChunk to the newly allocated chunk.
		lastChunk = newChunk
		lastChunkSize = 0
	}

	// Extend nodes in lastChunk.
	lastChunk.nodes = lastChunk.nodes[:lastChunkSize+1]
	node := &lastChunk.nodes[lastChunkSize]

	// Fill data.
	node.Parent = parent
	if !nullable {
		node.SetToRejectNull()
	}

	return node
}

// IndexOf returns the index of given node in the list. The first node has index 0. Return -1 if the
// node doesn't belong to the list.
func (list ResultNodeList) IndexOf(node *ResultNode) int {
	var (
		chunk     = list.chunks
		lastChunk = chunk.prev
		nodeAddr  = uintptr(unsafe.Pointer(node))
		listIndex = 0
	)

	for {
		chunkFirstNodeAddr := uintptr(unsafe.Pointer(&chunk.nodes[0]))
		if nodeAddr >= chunkFirstNodeAddr {
			// index of node within chunk
			index := int((nodeAddr - chunkFirstNodeAddr) / sizeOfResultNode)
			if index < chunk.Size() {
				return listIndex + index
			}
		}

		// Advance listIndex by chunk size.
		listIndex += chunk.Size()

		if chunk == lastChunk {
			break
		}
		// Move to the next chunk.
		chunk = chunk.next
	}

	return -1
}
