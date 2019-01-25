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
	"fmt"
	"log"
	"reflect"
	"sync/atomic"
	"unsafe"

	"github.com/botobag/artemis/concurrent/future"
)

//===----------------------------------------------------------------------------------------====//
// taskResult
//===----------------------------------------------------------------------------------------====//

type taskResultKind int

const (
	// Indicate that the task is waiting for processing.
	taskNotCompleted taskResultKind = iota

	// Indicate that an error has occurred during processing the task.
	taskResultErr

	// Indicate the task successfully loads the requested data.
	taskResultValue
)

// String implements fmt.Stringer to pretty-print taskResultKind.
func (kind taskResultKind) String() string {
	switch kind {
	case taskNotCompleted:
		return "an incompleted"
	case taskResultErr:
		return "an error"
	case taskResultValue:
		return "a value"
	}
	return "unknown"
}

type taskResult struct {
	Kind taskResultKind

	// Value has different meanings depends on Kind:
	//
	// For taskNotCompleted, the Value is an array of future.Waker each of which stores waker for a
	// dataloader.future that depends on the data loaded by this task. When task completes the data
	// loading (i.e., when result kind is changed to either taskResultErr or taskResultValue), it
	// notifies executor via the wakers to re-poll the futures to read the result.
	//
	// For taskResultErr, this contains the error value.
	//
	// For taskResultValue, this contains the loaded value identified by the task key.
	Value interface{}
}

var initialTaskResult = &taskResult{
	Kind:  taskNotCompleted,
	Value: []future.Waker{},
}

//===----------------------------------------------------------------------------------------====//
// resultFuture
//===----------------------------------------------------------------------------------------====//

// resultFuture implements future.Future. A future represents an asynchronous data loading performed
// by a Task. Load method in DataLoader returns a future object.
type resultFuture struct {
	// The task that loads data requested by the future
	task *Task

	// The slot that stores waker for the future
	wakerSlot int
}

var _ future.Future = (*resultFuture)(nil)

// Poll implements future.Future.
func (f *resultFuture) Poll(waker future.Waker) (future.PollResult, error) {
	task := f.task

	for {
		result := task.loadResult()
		switch result.Kind {
		case taskNotCompleted:
			wakers := result.Value.([]future.Waker)
			wakerSlot := f.wakerSlot

			if !reflect.DeepEqual(wakers[wakerSlot], waker) {
				// Update waker.
				wakers[wakerSlot] = waker

				// Create a new taskResult and perform a CAS to make sure the above update was deployed to
				// the up-to-date taskResult.
				swapped := atomic.CompareAndSwapPointer(
					&task.result,
					unsafe.Pointer(result),
					unsafe.Pointer(&taskResult{
						Kind:  taskNotCompleted,
						Value: wakers,
					}))

				// task.result has been changed. Restart loop to reload current result.
				if !swapped {
					break
				}
			}
			return future.PollResultPending, nil

		case taskResultErr:
			return nil, result.Value.(error)

		default:
			return result.Value, nil
		}
	}
}

//===----------------------------------------------------------------------------------------====//
// Task
//===----------------------------------------------------------------------------------------====//

// Task specifies key for BatchLoader to load data and provides storage to write result on
// completion. A task can be completed only once with either Complete or SetError.
type Task struct {
	key Key

	// Queue that contains this task; Could be nil if the task is never placed in a queue (e.g.,
	// created by Prime.)
	parent *taskQueue

	// The result stores the value loaded by the task or the error. Note that it could be accessed
	// simultaneously from different goroutines when result.Kind is a taskNotCompleted. Atomic
	// operations (i.e., atomic.CompareAndSwapPointer) are used when updating this field to order the
	// accesses and avoid data races.
	result/* *taskResult */ unsafe.Pointer

	// The next task in the list
	next *Task
}

func newTask(parent *taskQueue, key Key) *Task {
	return &Task{
		key:    key,
		parent: parent,
		result: unsafe.Pointer(initialTaskResult),
	}
}

// newFuture creates a future.Future that accesses the value loaded by the task.
func (t *Task) newFuture() future.Future {
	for {
		result := t.loadResult()
		switch result.Kind {
		case taskNotCompleted:
			// Allocate slot to store waker for the returnning future.
			curWakers := result.Value.([]future.Waker)
			newWakers := make([]future.Waker, len(curWakers)+1)
			copy(newWakers, curWakers)

			// Initialize the newly created waker slot with future.NopWaker (must be done before published
			// via the following CAS.)
			newWakerSlot := len(curWakers)
			newWakers[newWakerSlot] = future.NopWaker

			// Create a new taskResult with newWakers and perform a CAS.
			swapped := atomic.CompareAndSwapPointer(
				&t.result,
				unsafe.Pointer(result),
				unsafe.Pointer(&taskResult{
					Kind:  taskNotCompleted,
					Value: newWakers,
				}))

			if swapped {
				return &resultFuture{
					task:      t,
					wakerSlot: newWakerSlot,
				}
			}

			// If here, t.result has been changed. Restart loop to reload current result.

		case taskResultErr:
			return future.Err(result.Value.(error))

		case taskResultValue:
			return future.Ready(result.Value)

		default:
			panic("unknown task result kind")
		}
	}
}

func (t *Task) loadResult() *taskResult {
	return (*taskResult)(atomic.LoadPointer(&t.result))
}

// Key returns t.key.
func (t *Task) Key() Key {
	return t.key
}

func (t *Task) complete(newResult *taskResult) error {
	for {
		oldResult := t.loadResult()
		if oldResult.Kind != taskNotCompleted {
			return fmt.Errorf("task was already completed with %s (%+v) but want to accept %s (%+v)",
				oldResult.Kind, oldResult.Value, newResult.Kind, newResult.Value)
		}

		swapped := atomic.CompareAndSwapPointer(
			&t.result,
			unsafe.Pointer(oldResult),
			unsafe.Pointer(newResult),
		)
		if swapped {
			for _, waker := range oldResult.Value.([]future.Waker) {
				if err := waker.Wake(); err != nil {
					log.Printf("[WARN] Waker %T failed to wake executor that waits data keyed %+v to be "+
						"loaded by DataLoader\n", waker, t.Key())
				}
			}
			return nil
		}
	}
}

// Complete the task with the given value.
func (t *Task) Complete(value interface{}) error {
	return t.complete(&taskResult{
		Kind:  taskResultValue,
		Value: value,
	})
}

// SetError completes the task with an error value.
func (t *Task) SetError(err error) error {
	return t.complete(&taskResult{
		Kind:  taskResultErr,
		Value: err,
	})
}

// Completed returns true if the task has been completed (with either a value or an error.)
func (t *Task) Completed() bool {
	return t.loadResult().Kind != taskNotCompleted
}

//===----------------------------------------------------------------------------------------====//
// TaskIterator
//===----------------------------------------------------------------------------------------====//

// TaskList represents a list of Task's stored in a linked list from begin (included) to the end
// (excluded). It provides an iterator to access the TaskList in the list.
type TaskList struct {
	first *Task
	last  *Task
}

// Begin returns an iterator pointing to the first task in the list.
func (tasks *TaskList) Begin() TaskIterator {
	return TaskIterator{tasks.first}
}

// End returns an iterator refers to the pass-to-the-end task in the list.
func (tasks *TaskList) End() TaskIterator {
	if tasks.last != nil {
		return TaskIterator{tasks.last.next}
	}
	return TaskIterator{nil}
}

// Empty returns true if the TaskList doesn't contain any tasks.
func (tasks *TaskList) Empty() bool {
	return tasks.first == nil
}

// push appends a task at the end of the list. This is an internal method make a task list
// externally immutable.
func (tasks *TaskList) push(task *Task) {
	last := tasks.last
	if last == nil {
		tasks.first = task
	} else {
		last.next = task
	}
	tasks.last = task
}

// TaskIterator is used to access Task in a TaskList.
//
// Example:
//
//	for taskIter, taskEnd := tasks.Begin(), tasks.End(); taskIter != taskEnd; taskIter = taskIter.Next() {
//		task := taskIter.Task()
//		...
//	}
type TaskIterator struct {
	// The referring task by this iterator
	*Task
}

// Next returns a TaskIterator that refers to the Task next to the one referred by iter in the list.
// Note that it is an undefined behavior if iter doesn't refer to one of the task in the corresponding
// TaskList.
func (iter TaskIterator) Next() TaskIterator {
	return TaskIterator{iter.Task.next}
}
