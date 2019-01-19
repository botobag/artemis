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

// A Future represents an asynchronous computation.
//
// The design is borrowed from Rust's Future [0][1][2].
//
// (Following comments are copied from Rust's Future class with minor modification [3])
//
// A Future is a value that may not have finished computing yet. This kind of "asynchronous value"
// makes it possible for a thread to continue doing useful work while it waits for the value to
// become available.
//
// Futures alone are inert; they must be actively polled to make progress, meaning that each time
// the current task is woken up, it should actively re-poll pending futures that it still has an
// interest in.
//
// The Poll function is not called repeatedly in a tight loop -- instead, it should only be called
// when the future indicates that it is ready to make progress (by calling waker.Wake). If you're
// familiar with the poll(2) or select(2) syscalls on Unix it's worth noting that futures typically
// do *not* suffer the same problems of "all wakeups must poll all events"; they are more like
// epoll(4).
//
// An implementation of Poll should strive to return quickly, and must *never* block. Returning
// quickly prevents unnecessarily clogging up threads or event loops. If it is known ahead of time
// that a call to Poll may end up taking awhile, the work should be offloaded to a thread pool (or
// something similar) to ensure that Poll can return quickly.
//
// [0]: https://doc.rust-lang.org/std/future/index.html
// [1]: http://aturon.github.io/blog/2016/08/11/futures/
// [2]: https://aturon.github.io/blog/2016/09/07/futures-design/
// [3]: Copied from https://github.com/rust-lang/rust/blob/20d694a/src/libcore/future/future.rs#L20
type Future interface {
	// (Following comments are copied from Rust Core Library [0])
	//
	// Poll attempts to resolve the future to a final value, registering the current task for wakeup
	// if the value is not yet available.
	//
	// This function returns a tuple of (PollResult, error):
	//
	//	* ([any value], err): If error value is presented, the future is immediately finished with the
	//    error value.
	//	* (PollResultPending, nil): indicates the future is not ready yet
	//	* ([value other than PollResultPending], nil): indicates the future finished successfully with
	//    a vlaue.
	//
	// Once a future has finished, clients should not poll it again.
	//
	// When a future is not ready yet, Poll returns PollResultPending and stores Waker to be woken
	// once the future can make progress. For example, a future waiting for a socket to become
	// readable would store waker. When a signal arrives elsewhere indicating that the socket is
	// readable, wake.Wake is called and the socket future's task is awoken. Once a task has been
	// woken up, it should attempt to poll the future again, which may or may not produce a final
	// value.
	//
	// Note that on multiple calls to poll, only the most recent Waker passed to poll should be
	// scheduled to receive a wakeup.
	//
	// [0]: https://github.com/rust-lang/rust/blob/20d694a/src/libcore/future/future.rs#L20
	Poll(waker Waker) (PollResult, error)
}
