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
	"sync/atomic"

	"github.com/botobag/artemis/concurrent"
	"github.com/botobag/artemis/graphql"
)

// Task defines a task to be executed by an executor. Currently, this is an internal interface and
// only this package (executor package) can provide implementations.
type Task interface {
	// run defines operations performed by the task.
	run()
}

type executor interface {
	// Dispatch dispatches and schedules Task for running with executor.
	Dispatch(task Task)

	// Run starts the runner and returns the channel that passing execution result.
	Run(ctx *ExecutionContext) <-chan ExecutionResult

	// Yield pauses the execution of the given task. It is used by tasks (e.g., AsyncValueTask) to
	// notify that it is waiting for some resources to complete (i.e., wait for DataLoader to load
	// data) and executor can continue processing other tasks. Resume will be called once the Task has
	// made progress.
	Yield(task Task)

	// Resume resumes the execution of the given task paused by Yield. Typically, implementation
	// should re-dispatches the task.
	Resume(task Task)

	// AppendError adds an error to the error list of the given result node to indicate a failed field
	// execution. It implements error handling described in "Errors and Non-Nullability" [0] which
	// propagate the field error until a nullable field was encountered.
	//
	// [0]: https://facebook.github.io/graphql/June2018/#sec-Errors-and-Non-Nullability
	AppendError(err *graphql.Error, result *ResultNode)
}

func propagateExecutionError(result *ResultNode) {
	// Impelement "Errors and Non-Nullability". Propagate the field error until a nullable field was
	// encountered.
	//
	// Reference: https://facebook.github.io/graphql/June2018/#sec-Errors-and-Non-Nullability
	for result != nil && result.IsNonNull() {
		result = result.Parent
		result.Kind = ResultKindNil
		result.Value = nil
	}
}

//===----------------------------------------------------------------------------------------====//
// blockingExecutor
//===----------------------------------------------------------------------------------------====//

type yieldTaskState int

const (
	// The task is waiting for resumption
	yieldTaskStateWaiting yieldTaskState = iota
	// The task was resumed
	yieldTaskStateResumed
)

type blockingExecutor struct {
	// Errors that occurred during the execution
	errs graphql.Errors

	// Mutex that protects concurrent accesses to yieldCond, yieldTasks and resumeTasks.
	yieldMutex sync.Mutex
	yieldCond  sync.Cond
	// Queue of the tasks yielded during task execution
	yieldTasks map[Task]yieldTaskState
}

func newBlockingExecutor() executor {
	e := &blockingExecutor{
		yieldTasks: map[Task]yieldTaskState{},
	}
	e.yieldCond = sync.Cond{
		L: &e.yieldMutex,
	}
	return e
}

// Dispatch implements executor.
func (e *blockingExecutor) Dispatch(task Task) {
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

// Run implements executor.
func (e *blockingExecutor) Run(ctx *ExecutionContext) <-chan ExecutionResult {
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

// Yield implements executor. blockingExecutor can only
func (e *blockingExecutor) Yield(task Task) {
	// Acquire e.yieldMutex to place the task into yieldTasks.
	mutex := &e.yieldMutex
	mutex.Lock()
	yieldTasks := e.yieldTasks
	if _, exists := yieldTasks[task]; !exists {
		yieldTasks[task] = yieldTaskStateWaiting
	}
	mutex.Unlock()
}

// Resume implements executor.
func (e *blockingExecutor) Resume(task Task) {
	mutex := &e.yieldMutex
	mutex.Lock()
	e.yieldTasks[task] = yieldTaskStateResumed
	mutex.Unlock()

	// Unblock waiters in Dispatch.
	e.yieldCond.Signal()
}

func (e *blockingExecutor) AppendError(err *graphql.Error, result *ResultNode) {
	// Check parent result node to see whether the field is erroneous. If so, discard the error as per
	// spec.
	result = result.Parent
	if !result.IsNil() {
		e.errs.Append(err)
		propagateExecutionError(result)
	}
}

//===----------------------------------------------------------------------------------------====//
// concurrentExecutor
//===----------------------------------------------------------------------------------------====//

// concurrentExecutor executes fields concurrently using concurrent.Executor. It serves as a
// based for serialExecutor and parallelExecutor.
type concurrentExecutor struct {
	runner concurrent.Executor

	result     chan *ResultNode
	resultChan chan ExecutionResult

	// taskCounter is a
	taskCounter int64

	// Errors that occurred during the execution
	errs graphql.Errors
	// Mutex that guards writes to errs
	errsMutex sync.Mutex
}

func (e *concurrentExecutor) Init(runner concurrent.Executor) {
	e.result = make(chan *ResultNode, 1)
	e.resultChan = make(chan ExecutionResult, 1)
	e.runner = runner
}

func (e *concurrentExecutor) IncTaskCount() (remainingTasks int64) {
	return atomic.AddInt64(&e.taskCounter, 1)
}

func (e *concurrentExecutor) DecTaskCount() (remainingTasks int64) {
	return atomic.AddInt64(&e.taskCounter, -1)
}

// Yield implements executor.Yield.
func (e *concurrentExecutor) Yield(task Task) {
	// This is tricky.
	//
	// When yielding a task, the task will be removed from the executor queue and task count will be
	// decremented. When the task count reaches 0, executor thinks that all tasks have been processed
	// and will send the result. This IncTaskCount cancels the effect of DecTaskCount in taskFunc. It
	// retains the task count for the yielding task to avoid executor returning the result before its
	// resumption.
	//
	// BUG(zonr): Note that in an extreme condition, the following e.IncTaskCount may be performed
	//            AFTER task was re-dispatched by Resume and completed its execution. This causes
	//            task count being changed to a non-zero value after exeutor sends the result.
	e.IncTaskCount()
}

// AppendError implements executor.AppendError.
func (e *concurrentExecutor) AppendError(err *graphql.Error, result *ResultNode) {
	// Lock is required to append and to propagate the error. Firstly, multiple nodes may generate
	// errors at the same time. Secondly, according to specification, we can add most one error to the
	// error list per field [0].
	mutex := &e.errsMutex
	mutex.Lock()

	// Check parent result node to see whether the field is erroneous. If so, discard the error as per
	// spec.
	result = result.Parent
	if !result.IsNil() {
		e.errs.Append(err)
		propagateExecutionError(result)
	}

	mutex.Unlock()
}

func (e *concurrentExecutor) SendResult() {
	e.resultChan <- ExecutionResult{
		Data:   <-e.result,
		Errors: e.errs,
	}
}

//===----------------------------------------------------------------------------------------====//
// serialExecutor
//===----------------------------------------------------------------------------------------====//

// serialExecutor executes top-level fields one by one.
type serialExecutor struct {
	concurrentExecutor
	rootTasks []Task
}

func newSerialExecutor(runner concurrent.Executor) executor {
	e := &serialExecutor{}
	e.Init(runner)
	return e
}

// Dispatch implements executor.
func (e *serialExecutor) Dispatch(task Task) {
	isTopLevelNode := task.(*ExecuteNodeTask).node.Parent.IsRoot()
	if isTopLevelNode {
		// Top-level fields are executed serially [0].
		//
		// [0]: https://facebook.github.io/graphql/June2018/#sec-Mutation
		e.rootTasks = append(e.rootTasks, task)
	} else {
		e.IncTaskCount()
		// TODO: Error handling
		e.runner.Submit(e.taskFunc(task))
	}
}

// Run implements executor.
func (e *serialExecutor) Run(ctx *ExecutionContext) <-chan ExecutionResult {
	// Collect root tasks with rootTasksDispatcher.
	result, err := collectAndDispatchRootTasks(ctx, e)
	if err != nil {
		e.resultChan <- ExecutionResult{
			Errors: graphql.ErrorsOf(err.(*graphql.Error)),
		}
		return e.resultChan
	}

	e.result <- result

	// Run the first root task.
	e.runOneRootTask()

	return e.resultChan
}

func (e *serialExecutor) runOneRootTask() {
	// Note that this method assumes that it is called without other tasks being executed at the time.
	rootTasks := e.rootTasks

	if len(rootTasks) == 0 {
		e.SendResult()
	} else {
		e.rootTasks = rootTasks[1:]
		e.IncTaskCount()
		// Submit the first root task.
		//
		// TODO: Error handling
		e.runner.Submit(e.taskFunc(rootTasks[0]))
	}
}

func (e *serialExecutor) taskFunc(task Task) concurrent.Task {
	return concurrent.TaskFunc(func() (interface{}, error) {
		// Run the task.
		task.run()

		// Decrement task counter and check the count.
		if e.DecTaskCount() == 0 {
			// One root task has been completed. Execute the next one or write the result.
			e.runOneRootTask()
		}

		return nil, nil
	})
}

// Resume implements executor.
func (e *serialExecutor) Resume(task Task) {
	// Re-dispatch task to runner directly. Not using Dispatch here to avoid incrementing task count
	// (which was retained when the task was yielded.)
	e.runner.Submit(e.taskFunc(task))
}

//===----------------------------------------------------------------------------------------====//
// parallelExecutor
//===----------------------------------------------------------------------------------------====//

type parallelExecutor struct {
	concurrentExecutor

	// hasTasks is set once Dispatch is called. This is used by Run to deal with the case where
	// there's no any fields to execute we should immediately send an empty result. The only valid
	// transition is from 0 to 1. It is accessed with atomic memory primitives.
	hasTasks int32
}

func newParallelExecutor(runner concurrent.Executor) executor {
	e := &parallelExecutor{}
	e.Init(runner)
	return e
}

// Dispatch implements executor.
func (e *parallelExecutor) Dispatch(task Task) {
	atomic.StoreInt32(&e.hasTasks, 1)
	e.IncTaskCount()
	// TODO: Error handling
	e.runner.Submit(e.taskFunc(task))
}

// Run implements executor.
func (e *parallelExecutor) Run(ctx *ExecutionContext) <-chan ExecutionResult {
	// Start execution by dispatch root tasks.
	result, err := collectAndDispatchRootTasks(ctx, e)
	if err != nil {
		e.resultChan <- ExecutionResult{
			Errors: graphql.ErrorsOf(err.(*graphql.Error)),
		}
	} else {
		e.result <- result
		if atomic.LoadInt32(&e.hasTasks) == 0 {
			// No any fields to execute
			e.SendResult()
		}
	}

	return e.resultChan
}

func (e *parallelExecutor) taskFunc(task Task) concurrent.Task {
	return concurrent.TaskFunc(func() (interface{}, error) {
		// Run the task.
		task.run()

		// Decrement task counter.
		if e.DecTaskCount() == 0 {
			// No further tasks for running. Write the result.
			e.SendResult()
		}

		return nil, nil
	})
}

// Resume implements executor.
func (e *parallelExecutor) Resume(task Task) {
	// Re-dispatch task to runner directly. Not using Dispatch here to avoid incrementing task count
	// (which was retained when the task was yielded.)
	e.runner.Submit(e.taskFunc(task))
}
