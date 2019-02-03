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
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

//===----------------------------------------------------------------------------------------====//
// WorkerPoolExecutorConfig
//===----------------------------------------------------------------------------------------====//

// WorkerPoolExecutorConfig contains options to configure a WorkerPoolExecutor.
type WorkerPoolExecutorConfig struct {
	// The maximum number of workers allowed in pool (required, must be greater than 0)
	MaxPoolSize uint32

	// The minimum number of workers to maintain in pool
	MinPoolSize uint32

	// The maximum time for an idle thread to wait for new task
	KeepAliveTime time.Duration

	// Queue provides storage to store queueing tasks. If not set, a workerPoolTaskQueue will be
	// created and be used.
	Queue Queue
}

// Validate verifies config values.
func (config *WorkerPoolExecutorConfig) Validate() error {
	if config.MaxPoolSize == 0 {
		return errors.New(`WorkerPoolExecutor: MaxPoolSize must be a non-zero value which specifies ` +
			`the maximum number of workers to be created by the executor. If you have no idea, try to ` +
			`set the value to uint32(runtime.GOMAXPROCS(-1)).`)
	}

	if config.MaxPoolSize < config.MinPoolSize {
		return fmt.Errorf(`WorkerPoolExecutor: MaxPoolSize (%d) should be greater than MinPoolSize (%d)`,
			config.MaxPoolSize, config.MinPoolSize)
	}
	return nil
}

//===----------------------------------------------------------------------------------------====//
// workerPoolExecutorState
//===----------------------------------------------------------------------------------------====//

// workerPoolExecutorState contains current state of the WorkerPoolExecutor. It contains the pool
// size and the running state of the WorkerPoolExecutor. It should be updated atomically with CAS.
type workerPoolExecutorState int64

// workerPoolExecutorRunState indicates the running state of WorkerPoolExecutor. It is stored in
// the high 32 bits of workerPoolExecutorState. The low 32 bits in workerPoolExecutorRunState must
// be 0.
type workerPoolExecutorRunState int64

// Enumeration of workerPoolExecutorRunState
const (
	workerPoolExecutorRunStateMask int64 = -4294967296 // 0xffffffff00000000

	// Executor accepts and processes tasks. The constant is the one and the only one in
	// workerPoolExecutorRunState that sets the HSB. This makes workerPoolExecutorState with running
	// state be a negative value and thus enables fast check IsRunning.
	workerPoolExecutorRunStateRunning workerPoolExecutorRunState = workerPoolExecutorRunState(workerPoolExecutorRunStateMask)

	// Shutdown is invoked on Executor. Queued tasks are processed but no new tasks will be accepted.
	workerPoolExecutorRunStateShutdown = 0 // 0x0 << 32

	// There's no tasks in the queue and no new tasks is accepted.
	workerPoolExecutorRunStateTerminated = 4294967296 // 0x1 << 32
)

// RunState reads run state from state word.
func (s workerPoolExecutorState) RunState() workerPoolExecutorRunState {
	return workerPoolExecutorRunState(int64(s) & workerPoolExecutorRunStateMask)
}

// WorkerCount returns number of workers in the pool currently.
func (s workerPoolExecutorState) WorkerCount() uint32 {
	return uint32(s & 0xffffffff)
}

// Load loads state word with atomic.LoadInt64 because it is a lock-free variable. This suppresses
// the errors from Go's race detector. On conventional machines (e.g., x86-64), this is the same as
// dereferencing an int64 pointer. See [0] for more details.
//
// [0]: https://golang.org/doc/articles/race_detector.html#Primitive_unprotected_variable
func (s *workerPoolExecutorState) Load() workerPoolExecutorState {
	return workerPoolExecutorState(atomic.LoadInt64((*int64)(s)))
}

// SetRunState sets the run state.
func (s *workerPoolExecutorState) SetRunState(newRunState workerPoolExecutorRunState) (oldState workerPoolExecutorState) {
	for {
		oldState = *s
		if int64(oldState) >= int64(newRunState) {
			// States are only allowed to transition from RUNNING to SHUTDOWN to TERMINATED.
			return
		}

		newState := makeWorkerPoolExecutorState(newRunState, oldState.WorkerCount())
		if atomic.CompareAndSwapInt64((*int64)(s), int64(oldState), int64(newState)) {
			return
		}
	}
}

// IsRunning returns true if the run state is workerPoolExecutorRunStateRunning.
func (s workerPoolExecutorState) IsRunning() bool {
	return s < 0
}

// IsShutdown returns true if the executor receives an shutdown request.
func (s workerPoolExecutorState) IsShutdown() bool {
	return s >= workerPoolExecutorRunStateShutdown
}

// IsTerminated returns true if the executor is terminated.
func (s workerPoolExecutorState) IsTerminated() bool {
	return s >= workerPoolExecutorRunStateTerminated
}

// CompareAndIncWorkerCount increments the worker count in the given state by 1 with CAS.
func (s *workerPoolExecutorState) CompareAndIncWorkerCount(old workerPoolExecutorState) (done bool) {
	return atomic.CompareAndSwapInt64((*int64)(s), int64(old), int64(old+1))
}

// CompareAndDecWorkerCount decrements the worker count in the given state by 1 with CAS.
func (s *workerPoolExecutorState) CompareAndDecWorkerCount(old workerPoolExecutorState) (done bool) {
	return atomic.CompareAndSwapInt64((*int64)(s), int64(old), int64(old-1))
}

// DecWorkerCount decrement the worker count in the given state by 1. Return the new state after
// decrement.
func (s *workerPoolExecutorState) DecWorkerCount() workerPoolExecutorState {
	return workerPoolExecutorState(atomic.AddInt64((*int64)(s), int64(-1)))
}

// makeWorkerPoolExecutorState creates a workerPoolExecutorState from given run state and worker
// count.
func makeWorkerPoolExecutorState(
	runState workerPoolExecutorRunState,
	workerCount uint32) workerPoolExecutorState {

	return workerPoolExecutorState(int64(runState) | int64(workerCount))
}

//===----------------------------------------------------------------------------------------====//
// workerPoolTask
//===----------------------------------------------------------------------------------------====//

// workerPoolTask implements TaskHandle for Task executed in a WorkerPoolExecutor.
type workerPoolTask struct {
	Task
	executor *WorkerPoolExecutor

	// Lock that guards cond, result, err and next.
	mutex sync.Mutex
	cond  *sync.Cond

	// Return values from calling the Run method in Task; They're guarded by mutex.
	result interface{}
	err    error

	// The next task to this task in the workerPoolTaskQueue
	next *workerPoolTask
}

var (
	_ Task       = (*workerPoolTask)(nil)
	_ TaskHandle = (*workerPoolTask)(nil)
)

// newWorkerPoolTask initialzes task for running in WorkerPoolExecutor.
func newWorkerPoolTask(task Task, executor *WorkerPoolExecutor) *workerPoolTask {
	t := &workerPoolTask{
		Task:     task,
		executor: executor,
	}
	t.cond = sync.NewCond(&t.mutex)
	return t
}

// Cancel implements TaskHandle.
func (task *workerPoolTask) Cancel() error {
	// Request executor to cancel the task.
	if err := task.executor.cancelTask(task); err != nil {
		return err
	}

	// task was successfully cancelled. Set its result to ErrTaskCancelled.
	task.setResult(nil, ErrTaskCancelled)

	return nil
}

// setResult sets the execution result of the task and notifies the waiters blocked in AwaitResult.
func (task *workerPoolTask) setResult(result interface{}, err error) {
	// Lock mutex.
	mutex := &task.mutex
	mutex.Lock()

	task.result = result
	task.err = err

	// Broadcast cond to unblock all waiters.
	task.cond.Broadcast()

	// Set task.cond to nil to indicate that task result now is available.
	task.cond = nil

	// Unlock.
	mutex.Unlock()
}

// hasResult returns true if the task has completed.
func (task *workerPoolTask) hasResult() bool {
	// task.cond is nil'ed on completion.
	return task.cond == nil
}

// Result implements TaskHandle.
func (task *workerPoolTask) AwaitResult(timeout time.Duration) (interface{}, error) {
	// Lock mutex and recheck hasResult.
	mutex := &task.mutex
	mutex.Lock()

	if !task.hasResult() {
		// Block on cond.
		//
		// BUG(zonr): Support timed wait.
		task.cond.Wait()
	}

	// Read result and err.
	result, err := task.result, task.err

	// Unlock mutex before return.
	mutex.Unlock()

	return result, err
}

//===----------------------------------------------------------------------------------------====//
// workerPoolTaskQueue
//===----------------------------------------------------------------------------------------====//

// workerPoolTaskQueue is custom queue to store tasks for execution for WorkerPoolExecutor. The
// queue is essentially a circular linked list which makes use of the "intrusive" link in
// workerPoolTask to optimize footprint.
type workerPoolTaskQueue struct {
	// Tail of linked list; tail.next is the head of linked list.
	//
	// The actual type is *workerPoolTask. "tail" is read in Empty without locking and therefore may
	// cause data races while Push and Poll are writing a new tail, we have to access it with
	// atomic.{Load,Store}Pointer to appease Go's race detector. Access it with loadTail and
	// storeTail.
	tail unsafe.Pointer // *workerPoolTask

	// Lock that guards accesses to tail and pollCond.
	mutex sync.Mutex

	// Condition variable for Poll to wait for Push; If the queue is closed, it will be set to nil.
	pollCond *sync.Cond
}

func newWorkerPoolTaskQueue() *workerPoolTaskQueue {
	queue := &workerPoolTaskQueue{}
	queue.pollCond = sync.NewCond(&queue.mutex)
	return queue
}

func (queue *workerPoolTaskQueue) loadTail() *workerPoolTask {
	return (*workerPoolTask)(atomic.LoadPointer(&queue.tail))
}

func (queue *workerPoolTaskQueue) storeTail(tail *workerPoolTask) {
	atomic.StorePointer(&queue.tail, unsafe.Pointer(tail))
}

// Push implements Queue.
func (queue *workerPoolTaskQueue) Push(element interface{}) error {
	task := element.(*workerPoolTask)

	mutex := &queue.mutex
	mutex.Lock()

	// Disallow new element to be added to queue.
	cond := queue.pollCond
	if cond == nil {
		mutex.Unlock()
		return ErrQueueClosed
	}

	tail := queue.loadTail()
	empty := queue.Empty()

	if empty {
		// task is also the head.
		task.next = task
	} else {
		// Link head node to task.next.
		task.next = tail.next
		// Append task after tail.
		tail.next = task
	}
	// Update queue.tail.
	queue.storeTail(task)

	if empty {
		cond.Signal()
	}

	mutex.Unlock()

	return nil
}

// Poll implements Queue.
func (queue *workerPoolTaskQueue) Poll(timeout time.Duration) (interface{}, error) {
	mutex := &queue.mutex
	mutex.Lock()

	if queue.Empty() {
		cond := queue.pollCond
		if cond != nil {
			// Block on cond to wait for Push. Only do so when the queue is not closed.
			//
			// BUG(zonr): Support timed wait.
			cond.Wait()
		}

		if queue.Empty() {
			// Unlock mutex for return.
			mutex.Unlock()
			return nil, nil
		}
	}

	tail := queue.loadTail()
	head := tail.next

	if tail == head {
		// Become an empty queue.
		queue.storeTail(nil)
	} else {
		// Update head.
		tail.next = head.next
	}

	// Unlock mutex for return.
	mutex.Unlock()

	return head, nil
}

// Remove implements Queue.
func (queue *workerPoolTaskQueue) Remove(element interface{}) error {
	mutex := &queue.mutex
	mutex.Lock()

	task := element.(*workerPoolTask)

	// Search the previous task of the element in the queue.
	var prevTask *workerPoolTask

	if !queue.Empty() {
		tail := queue.loadTail()
		head := tail.next

		// Search from head.
		prevTask = head

		for {
			nextTask := prevTask.next
			if nextTask == task {
				// Re-link.
				prevTask.next = task.next

				if task == tail {
					// The removed task is tail. Update queue.tail as well.
					if tail == head {
						// Queue becomes empty.
						queue.storeTail(nil)
					} else {
						queue.storeTail(prevTask)
					}
				}
				// Help GC.
				task.next = nil

				mutex.Unlock()
				return nil
			}

			// Move to the next task
			prevTask = nextTask
			if prevTask == head {
				break
			}
		}
	}

	mutex.Unlock()

	return ErrElementNotFound
}

// Close implements Queue.
func (queue *workerPoolTaskQueue) Close() {
	mutex := &queue.mutex
	mutex.Lock()
	cond := queue.pollCond
	if cond != nil {
		// Unblock current waiters.
		cond.Broadcast()
		queue.pollCond = nil
	}
	mutex.Unlock()
}

// Empty implements Queue.
func (queue *workerPoolTaskQueue) Empty() bool {
	return queue.loadTail() == nil
}

//===----------------------------------------------------------------------------------------====//
// workerPoolExecutorWorker
//===----------------------------------------------------------------------------------------====//

type workerPoolExecutorWorker struct {
	// Executor that pools this worker
	executor *WorkerPoolExecutor
}

// newWorkerPoolExecutorWorker creates a worker for WorkerPoolExecutor.
func newWorkerPoolExecutorWorker(executor *WorkerPoolExecutor) workerPoolExecutorWorker {
	return workerPoolExecutorWorker{
		executor: executor,
	}
}

// Start creates a goroutine to execute run loop.
func (w workerPoolExecutorWorker) Start(firstTask Task) {
	go w.run(firstTask)
}

// Run implements run loop for worker to execute tasks in the queue.
func (w workerPoolExecutorWorker) run(firstTask Task) {
	task := firstTask

	// The run loop
	for {
		if task == nil {
			// Retrieve one task from executor.
			task = w.executor.pollTask()
			if task == nil {
				// No task to be executed; Terminate the worker.
				break
			}
		}

		// Run task.
		result, err := task.Run()

		// Set the result.
		task.(*workerPoolTask).setResult(result, err)

		// Reset task.
		task = nil
	}

	w.executor.terminateWorker(w)
}

//===----------------------------------------------------------------------------------------====//
// WorkerPoolExecutor
//===----------------------------------------------------------------------------------------====//

// WorkerPoolExecutor runs submitted tasks with one of the pooled workers backed by a goroutine. The
// implementation is heavily influenced by Doug Lea's PooledExecutor [0] which was released into the
// public domain [1].
//
// We avoid using defer, channel and even lock in the critical path to make it perform efficiently.
//
// The pool does not by default preallocate worker goroutines. Instead, a worker is created if
// necessary when a task arrives.
//
// [0]: http://gee.cs.oswego.edu/dl/classes/EDU/oswego/cs/dl/util/concurrent/intro.html
// [1]: http://creativecommons.org/publicdomain/zero/1.0/
type WorkerPoolExecutor struct {
	// A lock-free word that contains pool running state and worker count
	state workerPoolExecutorState

	// Configuration
	config *WorkerPoolExecutorConfig

	// Task queue contains task to be executed
	taskQueue Queue

	// Mutex for guarding workerPool
	mutex sync.Mutex

	// Channels that are used for waiting termination. This is guarded by mutex.
	terminations []chan<- bool
}

// WorkerPoolExecutor implements Executor.
var _ Executor = (*WorkerPoolExecutor)(nil)

// NewWorkerPoolExecutor creates a WorkerPoolExecutor from given config and uses the supplied Queue for
// queuing tasks.
func NewWorkerPoolExecutor(config WorkerPoolExecutorConfig) (*WorkerPoolExecutor, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	taskQueue := config.Queue
	if taskQueue == nil {
		taskQueue = newWorkerPoolTaskQueue()
	}

	return &WorkerPoolExecutor{
		state:     makeWorkerPoolExecutorState(workerPoolExecutorRunStateRunning, 0),
		config:    &config,
		taskQueue: taskQueue,
	}, nil
}

// Shutdown implements Executor.
func (executor *WorkerPoolExecutor) Shutdown() (terminated <-chan bool, err error) {
	mutex := &executor.mutex

	// Hold lock for potential modification on executor.terminations. This also avoids races with
	// signals in tryTerminate.
	mutex.Lock()

	// Create a channel for return which notifies the completion of termination.
	termination := make(chan bool, 1)

	// Transition the state to SHUTDOWN. After that, addWorker and addTask would refuse to any request.
	prevState := executor.state.SetRunState(workerPoolExecutorRunStateShutdown)

	if prevState.IsTerminated() {
		// Executor was already terminated. Fill the returning channel with termination signal.
		termination <- true
	} else {
		// Append a termination to executor.terminations.
		executor.terminations = append(executor.terminations, termination)

		// Transition from RUNNING.
		if prevState.IsRunning() {
			// Close queue. This will also unblock all workers that are waiting for tasks on empty queue.
			executor.taskQueue.Close()
		}
	}

	// Unlock mutex to call tryTerminate.
	mutex.Unlock()

	// Try to advance to TERMINATED.
	executor.tryTerminate()

	// Setup return values.
	return termination, nil
}

// loadState loads current state. See comment for the Load method in workerPoolExecutorState.
func (executor *WorkerPoolExecutor) loadState() workerPoolExecutorState {
	return executor.state.Load()
}

// tryTerminate tries to transition to TERMINATED if the executor is shut down, and there's no task
// in the queue and all workers are terminated.
func (executor *WorkerPoolExecutor) tryTerminate() {
	// Load state.
	state := executor.loadState()

	// Quick return if we have not received shutdown request or is already terminated.
	if !state.IsShutdown() || state.IsTerminated() {
		return
	}

	// Quick return if task queue is not empty.
	if !executor.taskQueue.Empty() {
		return
	}

	// Quick return if there're some workers.
	if state.WorkerCount() > 0 {
		return
	}

	// No workers in the pool.

	// Lock mutex to send termination signal after transition to TERMINATED.
	mutex := &executor.mutex
	mutex.Lock()
	defer mutex.Unlock()

	if !state.IsTerminated() {
		// Transition to TERMINATED. No new worker can be added to the executor after the state was
		// transitioned to SHUTDOWN. We can update state work with trivial assignment.
		executor.state.SetRunState(workerPoolExecutorRunStateTerminated)

		// Send termination signals.
		terminations := executor.terminations
		executor.terminations = nil
		for _, termination := range terminations {
			termination <- true
		}
	}
}

// Submit implements Executor.
//
// On receiving task, and fewer than the number of config.MinPollSize are running, a new thread is
// always created to process the task even if other workers are idly waiting for task.  Otherwise, a
// new thread is created only if there are fewer than the number of config.MaxPollSize and the
// request cannot immediately be queued.
func (executor *WorkerPoolExecutor) Submit(task Task) (TaskHandle, error) {
	// Create task handle for the task.
	handle := newWorkerPoolTask(task, executor)

	// Wrap input task into a workerPoolTask.
	task = handle

	// Load config into local stack.
	config := executor.config

	// Load state.
	state := executor.loadState()

	// Ensure minimum number of workers.
	if state.WorkerCount() < config.MinPoolSize {
		if err := executor.addWorker(task, config.MinPoolSize); err == nil {
			return handle, nil
		}
		// Ignore errors and reload state.
		state = executor.loadState()
	}

	if state.IsRunning() {
		// Try to give the task to existing worker by putting it to the queue. Note that this assumes
		// that there's always a worker in the pool to process it.
		if err := executor.addTask(task); err != nil {
			return nil, err
		}
		return handle, nil
	}

	// Final try by directly requesting a worker to perform the task.
	if err := executor.addWorker(task, config.MaxPoolSize); err != nil {
		return nil, err
	}

	return handle, nil
}

var (
	errRejectWorkerDueToShuttingDown = errors.New("unable to add new worker because executor is shutting down")
	errTooManyWorkers                = errors.New("unable to add new worker because worker pool is full")
	errRejectTaskDueToShuttingDown   = errors.New("unable to execute task because executor is shutting down")
)

// addWorker tries to create a worker to execute the task. limit specifies the bound of pool size.
// An error will be returned if the pool size exceeds the limit after adding the newly created
// worker.
//
// It could fail with the following reasons:
func (executor *WorkerPoolExecutor) addWorker(firstTask Task, limit uint32) error {
	for {
		// Load state.
		state := executor.loadState()
		if state.IsShutdown() {
			return errRejectWorkerDueToShuttingDown
		}

		// Check pool size limit.
		if (state.WorkerCount() + 1) > limit {
			return errTooManyWorkers
		}

		// Atomically increment pool size.
		if executor.state.CompareAndIncWorkerCount(state) {
			break
		}

		// CAS failed. Restart the loop to load new state.
	}

	// Create a new worker and start running with initial task.
	newWorkerPoolExecutorWorker(executor).Start(firstTask)

	return nil
}

// terminateWorker is called upon termination of worker w. It should be called from the goroutine
// that runs w.
func (executor *WorkerPoolExecutor) terminateWorker(w workerPoolExecutorWorker) {
	// Note that worker count should have been decremented (by pollTask).
	state := executor.loadState()

	if state.IsShutdown() {
		// Try to advance to TERMINATED.
		executor.tryTerminate()
	} else {
		// Create a replacement as needed.
		minPoolSize := executor.config.MinPoolSize
		if minPoolSize == 0 && !executor.taskQueue.Empty() {
			minPoolSize = 1
		}
		if minPoolSize < state.WorkerCount() {
			executor.addWorker(nil, minPoolSize)
		}
	}
}

// addTask puts the task in the queue and ensures that there'll be a worker to run the task.
func (executor *WorkerPoolExecutor) addTask(task Task) error {
	taskQueue := executor.taskQueue

	// Put task to the queue.
	if err := taskQueue.Push(task); err != nil {
		return err
	}

	for {
		// The task was successfully enqueued. But during the enqueue, someone may shutdown the executor
		// or there's no worker to execute the task.
		state := executor.loadState()
		if !state.IsRunning() {
			// Try to remove the task from queue.
			if err := executor.taskQueue.Remove(task); err == nil {
				// Successfully remove the task.
				return errRejectTaskDueToShuttingDown
			}
			// Someone took the task from queue.
		} else if state.WorkerCount() == 0 {
			// Executor is running and there's no any worker in current pool. This may happen when
			// config.MinPoolSize is zero. Try to add a worker.
			if err := executor.addWorker(nil, 1); err != nil {
				// Retry.
				continue
			}
		}
		break
	}

	return nil
}

// cancelTask tries to remove the task from the queue to stop its execution.
func (executor *WorkerPoolExecutor) cancelTask(task Task) error {
	if err := executor.taskQueue.Remove(task); err != nil {
		return err
	}

	// Try to advance to tryTerminate.
	executor.tryTerminate()

	return nil
}

// pollTask blocks the calling worker to wait for a task. This could return nil in the following
// case to indicate that no further task could be run:
//
//  1. The executor received a shutdown request and the task queue is empty.
//  2. The worker doesn't get a task within config.KeepAliveTime and current size of worker pool is
//     greater than config.MaxPoolSize.
//
// Note that upon returning nil, the worker count in state word is decremented.
func (executor *WorkerPoolExecutor) pollTask() Task {
	isIdle := false
	// Cache the config and task queue locally.
	taskQueue := executor.taskQueue
	config := executor.config

	for {
		// Reload state.
		state := executor.state.Load()
		noTasks := taskQueue.Empty()

		if state.IsShutdown() && noTasks {
			executor.state.DecWorkerCount()
			return nil
		}

		redundantWorker := state.WorkerCount() > config.MinPoolSize

		if redundantWorker &&
			isIdle &&
			(state.WorkerCount() > 1 || noTasks) {
			// Cause idle worker to die. The check depends on state.WorkerCount. Other workers may also be
			// here. Perform CAS on decrementing worker count before return. This is would limit at most
			// one idle worker to be removed at a time to keep number of config.MinPoolSize workers in the
			// pool.
			if executor.state.CompareAndDecWorkerCount(state) {
				return nil
			}
		}

		// Reset isIdle.
		isIdle = false

		// Determine timeout for polling.
		var timeout time.Duration
		if state.WorkerCount() > config.MinPoolSize {
			timeout = config.KeepAliveTime
		}

		// Poll queue.
		task, err := taskQueue.Poll(timeout)
		if err == ErrQueuePollTimeout {
			isIdle = true
			// Restart loop to reload state and check whether the worker can be killed.
		} else if err != nil {
			// Ignore error and continue polling.
			//
			// FIXME: Is this ok?
		} else if task != nil {
			return task.(Task)
		}
	}
}
