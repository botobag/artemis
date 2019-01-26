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

package future

// A Waker is a handle to "wake up" a Future that was previously polled to a pending. Practically,
// it notifies executor to place the Future back on the queue of ready tasks.
type Waker interface {
	// Wake indicates the associated task is ready to make progress and should be polled again.
	//
	// Executors generally maintain a queue of "ready" tasks; and Wake should place the associated
	// task onto this queue.
	Wake() error
}

// The WakerFunc type is an adapter to allow the use of ordinary functions as Waker.
type WakerFunc func() error

// Wake implements Waker which calls f().
func (f WakerFunc) Wake() error {
	return f()
}

// Type for NopWaker
type nopWaker int

func (nopWaker) Wake() error {
	return nil
}

// NopWaker is a Waker that does nothing. It is useful to be used as an initial value for Waker.
const NopWaker nopWaker = 0
