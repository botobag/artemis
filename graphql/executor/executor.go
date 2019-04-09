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
	"sync"

	"github.com/botobag/artemis/graphql"
)

// Task defines a task to be executed by an executor. Currently, this is an internal interface and
// only this package (executor package) can provide implementations.
type Task interface {
	// run defines operations performed by the task.
	run()
}

// DataLoaderCycle is used to determine when an AsyncValueTask should dispatches. The following
// explains the magic:
//
//  1. It is an unsigned integer starting from 0.
//  2. Every update to cycle counter increments its value by 1.
//  3. The cycle counter is only incremented when DataLoaderManager in current execution dispatches
//     data loaders that have pending batch fetching.
//  4. Base on 3., there's at most one dispatch occurred in each cycle. The executor maintains a
//     DataLoaderCycle which indicates the next cycle that hasn't started dispatching
//     DataLoaderManager. The cycle value can be accessed via DataLoadrCycle method provided by
//     executor. For concurrent executor, the cycle is updated and accessed with atomic primitives.
//  5. Task (currently only AsyncValueTask) that relies on data loaders (for fetching data) also
//     maintains a DataLoadrCycle indicating in which cycle of the data loader dispatch being
//     depended by the task. The value is usually initialized to the value of executor's
//     DataLoadrCycle at the time a task was created.
//  6. tryDispatchDataLoaders is called when a Task want to dispatch data loaders to fetch desired
//     data. See the comments in the function to see how DataLoadrCycle is used by executor and task
//     cooperatively to avoid excessive data loader dispatch.
type DataLoaderCycle uint64

func propagateExecutionError(result *ResultNode) {
	// Impelement "Errors and Non-Nullability". Propagate the field error until a nullable field was
	// encountered.
	//
	// Reference: https://graphql.github.io/graphql-spec/June2018/#sec-Errors-and-Non-Nullability
	for result != nil && result.ShouldRejectNull() {
		result = result.Parent
		result.Kind = ResultKindNil
		result.Value = nil
	}
}

type yieldTaskState int

const (
	// The task is waiting for resumption
	yieldTaskStateWaiting yieldTaskState = iota
	// The task was resumed
	yieldTaskStateResumed
)

type executor struct {
	// Errors that occurred during the execution
	errs graphql.Errors

	// See comments for DataLoaderCycle.
	dataLoaderCycle DataLoaderCycle

	// Mutex that protects concurrent accesses to yieldCond, yieldTasks and resumeTasks.
	yieldMutex sync.Mutex
	yieldCond  sync.Cond
	// Queue of the tasks yielded during task execution
	yieldTasks map[Task]yieldTaskState
}

func newExecutor() *executor {
	e := &executor{
		yieldTasks: map[Task]yieldTaskState{},
	}
	e.yieldCond = sync.Cond{
		L: &e.yieldMutex,
	}
	return e
}

// Dispatch dispatches and schedules Task for running with executor.
func (e *executor) Dispatch(task Task) {
	// Run the specified task.
	task.run()

	// task may generate (yield) other tasks during its processing. Process them before return.

	// Acquire mutex to wait for yielded tasks.
	mutex := &e.yieldMutex
	mutex.Lock()

	// Load yielded tasks.
	yieldTasks := e.yieldTasks

	for {
		hasResumedTask := false

		// Find the first task that has been resumed.
		for task, state := range yieldTasks {
			if state == yieldTaskStateResumed {
				hasResumedTask = true
				// Remove the task from yieldTasks.
				delete(yieldTasks, task)
				// Unlock mutex in prior to running the task.
				mutex.Unlock()
				// Run the task.
				task.run()
				// Re-lock the mutex.
				mutex.Lock()
				// Reload e.yieldTasks.
				yieldTasks = e.yieldTasks
				// Break to restart loop with newly loaded yieldTasks (which may have been changed during
				// task.run.)
				break
			}
		}

		// Stop if all tasks have been processed.
		if len(yieldTasks) <= 0 {
			break
		}

		// When:
		//
		//  1. There're some tasks (checked above), and
		//  2. All tasks are waiting for resumption
		//
		// Block on Cond to wait for signal from Resume.
		if !hasResumedTask {
			e.yieldCond.Wait()
		}
	}

	mutex.Unlock()
}

// Run starts the runner and returns the channel that passing execution result.
func (e *executor) Run(ctx *ExecutionContext) <-chan ExecutionResult {
	resultChan := make(chan ExecutionResult, 1)

	// Start execution by dispatch root tasks.
	result, err := collectAndDispatchRootTasks(ctx, e)
	if err != nil {
		resultChan <- ExecutionResult{
			Errors: graphql.ErrorsOf(err.(*graphql.Error)),
		}
	} else {
		resultChan <- ExecutionResult{
			Data:   result,
			Errors: e.errs,
		}
	}

	return resultChan
}

// Yield pauses the execution of the given task. It is used by tasks (e.g., AsyncValueTask) to
// notify that it is waiting for some resources to complete (i.e., wait for DataLoader to load data)
// and executor can continue processing other tasks. Resume will be called once the Task has made
// progress.
func (e *executor) Yield(task Task) {
	// Acquire e.yieldMutex to place the task into yieldTasks.
	mutex := &e.yieldMutex
	mutex.Lock()
	yieldTasks := e.yieldTasks
	if _, exists := yieldTasks[task]; !exists {
		yieldTasks[task] = yieldTaskStateWaiting
	}
	mutex.Unlock()
}

// Resume resumes the execution of the given task paused by Yield. Typically, implementation should
// re-dispatches the task.
func (e *executor) Resume(task Task) {
	mutex := &e.yieldMutex
	mutex.Lock()
	e.yieldTasks[task] = yieldTaskStateResumed
	mutex.Unlock()

	// Unblock waiters in Dispatch.
	e.yieldCond.Signal()
}

// DataLoaderCycle returns current data loader cycle counter. See comments for DataLoaderCycle type.
func (e *executor) DataLoaderCycle() DataLoaderCycle {
	return e.dataLoaderCycle
}

// IncDataLoaderCycle incremnts data loader cycle counter by one. See comments in
// tryDispatchDataLoaders.
func (e *executor) IncDataLoaderCycle(expected DataLoaderCycle) bool {
	// Tasks are executed serially. Therefore it is safe to increment the counter directly.
	e.dataLoaderCycle++
	return true
}

// AppendError adds an error to the error list of the given result node to indicate a failed field
// execution. It implements error handling described in "Errors and Non-Nullability" [0] which
// propagate the field error until a nullable field was encountered.
//
// [0]: https://graphql.github.io/graphql-spec/June2018/#sec-Errors-and-Non-Nullability
func (e *executor) AppendError(err *graphql.Error, result *ResultNode) {
	// Check parent result node to see whether the field is erroneous. If so, discard the error as per
	// spec.
	result = result.Parent
	if !result.IsNil() {
		e.errs.Append(err)
		propagateExecutionError(result)
	}
}
