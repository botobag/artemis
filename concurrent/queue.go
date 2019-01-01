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

package concurrent

import (
	"errors"
	"time"
)

var (
	// ErrQueueClosed is returned by Push to indicate the queue cannot accept the new element because
	// it is closed.
	ErrQueueClosed = errors.New("queue: closed")

	// ErrQueuePollTimeout is returned by Poll to indicate the poll doesn't find an element within
	// timeout.
	ErrQueuePollTimeout = errors.New("queue: poll timeout")

	// ErrElementNotFound is returned by Remove to indicate the given element is not in the queue.
	ErrElementNotFound = errors.New("queue: given element is not found in the queue")
)

// Queue implements container which stores a collection of objects. Implementation to the interfaces
// should be thread-safe. That is, they need to allow concurrent accesses.
type Queue interface {
	// Add inserts the specified element into this queue. Return nil if the element is successfully
	// inserted. Note that element cannot be nil.
	Push(element interface{}) error

	// Poll pops one element from the head of this queue only if one is available within the timeout.
	Poll(timeout time.Duration) (interface{}, error)

	// Remove removes the given element from queue.
	Remove(element interface{}) error

	// Empty returns true if the queue contains no elements.
	Empty() bool

	// Close stops queue to accept new elements. Elements that are submitted to the queue are still
	// available via Poll. Calls to Push will return ErrQueueClosed. Once the queue becomes empty, any
	// calls to Poll will immediately return with nil.
	Close()
}
