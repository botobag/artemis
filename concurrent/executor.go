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

// Task represents an instance that can be executed by an Executor.
type Task interface {
	// Run performs actions to complete a Task. The return value would be send to corresponding
	// TaskHandle which can be then accessed via calling Result.
	Run() (interface{}, error)
}

// The TaskFunc type is an adapter to allow the use of ordinary functions as a Task.
type TaskFunc func() (interface{}, error)

// TaskFunc implements Task.
var _ Task = (TaskFunc)(nil)

// Run implements Task. It calls f().
func (f TaskFunc) Run() (interface{}, error) {
	return f()
}

// Error values to be returned from AwaitResult.
var (
	// ErrTaskCancelled indicates the task is cancelled.
	ErrTaskCancelled = errors.New("task was cancelled")
	// ErrTaskAwaitResultTimeout indicates runs out of time to wait for result.
	ErrkAwaitTaskResultTimeout = errors.New("timeout while waiting task result")
)

// TaskHandle tracks progress of a Task and can be used to cancel execution and/or wait for
// completion.
type TaskHandle interface {
	// Cancel tries to cancel execution of the associated task.
	Cancel() error

	// AwaitResult blocks caller until the underlying task completed or timeout. Possible return
	// values are:
	//
	//  1. (nil, ErrTaskCancelled): task was cancelled.
	//  2. (nil, ErrkAwaitTaskResultTimeout)
	//  3. (any, any): the result returned from the Run method of corresponding task.
	AwaitResult(timeout time.Duration) (interface{}, error)
}

// Executor provides interfaces to manage and to execute tasks.
type Executor interface {
	// Shutdown shuts down the executor. Previously submitted tasks are executed but no new tasks will
	// be accepted. It is an no-op if the executor has already shut down. It returns a channel which
	// will receives a notification from the Executor when all remaining tasks have completed after
	// shutdown request.
	Shutdown() (terminated <-chan bool, err error)

	// Submit submits a task for execution. The method only arranges task for execution. The actual
	// execution may occur sometime later.
	Submit(task Task) (TaskHandle, error)
}
