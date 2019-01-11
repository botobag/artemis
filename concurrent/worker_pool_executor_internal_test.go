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
	"math/rand"
	"sync"
	"sync/atomic"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func newTestTask() Task {
	return newWorkerPoolTask(TaskFunc(func() (interface{}, error) {
		return nil, nil
	}), nil)
}

func produce(queue *workerPoolTaskQueue, n int, tasks []Task, wg *sync.WaitGroup) {
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(workerIndex int) {
			defer wg.Done()
			for taskIndex, task := range tasks {
				if taskIndex%n == workerIndex {
					Expect(queue.Push(task)).Should(Succeed())
				}
			}
		}(i)
	}
}

func consume(queue *workerPoolTaskQueue, n int, numRemovers int, tasks []Task, wg *sync.WaitGroup) {
	// Build task map for checking results.
	taskMap := map[Task]bool{}
	for _, task := range tasks {
		taskMap[task] = true
	}
	// Mutex that guards accesses to taskMap.

	var (
		taskMapMutex sync.Mutex
		numTasks     = int64(len(tasks))
	)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				// Decrement numTasks.
				cur := atomic.LoadInt64(&numTasks)
				if cur <= 0 {
					// All tasks are consumed. Call Close to unblock others that stuck in Poll.
					queue.Close()
					break
				}

				if !atomic.CompareAndSwapInt64(&numTasks, cur, cur-1) {
					// numTasks has been modified by others. Restart the loop to check current value.
					continue
				}

				task, err := queue.Poll(0)
				Expect(err).ShouldNot(HaveOccurred())
				if task == nil {
					continue
				}

				// Lock taskMapMutex.
				taskMapMutex.Lock()
				Expect(taskMap).Should(HaveKey(task))
				delete(taskMap, task.(Task))
				taskMapMutex.Unlock()
			}
		}()
	}

	for i := 0; i < numRemovers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for atomic.LoadInt64(&numTasks) > 0 {
				// Select task to be removed randomly.
				task := tasks[rand.Int31n(int32(len(tasks)))]

				// Check whether the specified task is removed.
				taskMapMutex.Lock()
				_, exists := taskMap[task.(Task)]
				taskMapMutex.Unlock()

				// Remove.
				err := queue.Remove(task)
				if !exists {
					Expect(err).Should(MatchError(ErrElementNotFound))
				} else {
					Expect(err).Should(Or(BeNil(), MatchError(ErrElementNotFound)))

					if err == nil {
						// Successfully removed. Update taskMap.
						taskMapMutex.Lock()
						Expect(taskMap).Should(HaveKey(task))
						delete(taskMap, task.(Task))
						taskMapMutex.Unlock()

						// Decrement numTasks.
						if atomic.AddInt64(&numTasks, -1) == 0 {
							// All tasks are consumed. Call Close to unblock others that stuck in Poll.
							queue.Close()
							break
						}
					} else if err == ErrElementNotFound {
						// Someone has consumed the task.
					}
				}
			}
		}()
	}
}

func testQueue(numProducers int, numConsumers int, numRemovers int, numTasks int) {
	queue := newWorkerPoolTaskQueue()

	// Create number of NumTestTasks tasks.
	const NumTestTasks = 100
	tasks := make([]Task, NumTestTasks)
	for i := 0; i < NumTestTasks; i++ {
		tasks[i] = newTestTask()
	}

	// Create 10 producers to push the tasks.
	var wg sync.WaitGroup
	produce(queue, numProducers, tasks, &wg)

	// Consume tasks.
	consume(queue, numConsumers, numRemovers, tasks, &wg)

	// Block until all tasks was pushed and popped.
	wg.Wait()

	Expect(queue.Empty()).Should(BeTrue())
}

var _ = Describe("workerPoolTaskQueue: default custom queue used by WorkerPoolExecutor", func() {
	It("accepts a task", func() {
		queue := newWorkerPoolTaskQueue()
		task := newTestTask()
		Expect(queue.Empty()).Should(BeTrue())
		Expect(queue.Push(task)).Should(Succeed())
		Expect(queue.Empty()).Should(BeFalse())
		Expect(queue.Poll(0)).Should(Equal(task))
		Expect(queue.Empty()).Should(BeTrue())
	})

	It("accepts multiple producers", func() {
		testQueue(10 /* numProducers */, 1 /*numConsumers */, 0 /* numRemovers */, 100 /* numTasks */)
	})

	It("accepts multiple consumers", func() {
		testQueue(1 /* numProducers */, 10 /*numConsumers */, 0 /* numRemovers */, 100 /* numTasks */)
	})

	It("accepts multiple producers and consumers", func() {
		testQueue(10 /* numProducers */, 10 /*numConsumers */, 0 /* numRemovers */, 100 /* numTasks */)
	})

	Context("removes tasks from queue", func() {
		It("removes tasks that haven't been taken", func() {
			queue := newWorkerPoolTaskQueue()
			task := newTestTask()
			Expect(queue.Push(task)).Should(Succeed())
			Expect(queue.Remove(task)).Should(Succeed())
		})

		It("cannot remove tasks that have been taken", func() {
			queue := newWorkerPoolTaskQueue()
			task := newTestTask()
			Expect(queue.Push(task)).Should(Succeed())
			Expect(queue.Poll(0)).Should(Equal(task))
			Expect(queue.Remove(task)).Should(MatchError(ErrElementNotFound))
		})

		It("can remove elements concurrently with multiple producers and consumers", func() {
			testQueue(10 /* numProducers */, 10 /*numConsumers */, 1 /* numRemovers */, 100 /* numTasks */)
			testQueue(10 /* numProducers */, 10 /*numConsumers */, 10 /* numRemovers */, 100 /* numTasks */)
		})
	})

	It("can close multiple times", func() {
		queue := newWorkerPoolTaskQueue()
		queue.Close()
		queue.Close()
	})

	It("disallows push on closed queue", func() {
		queue := newWorkerPoolTaskQueue()
		queue.Close()
		task := newTestTask()
		Expect(queue.Push(task)).Should(MatchError(ErrQueueClosed))
	})

	It("unblocks poll on empty closed queue", func() {
		queue := newWorkerPoolTaskQueue()
		Expect(queue.Empty()).Should(BeTrue())

		// Use goroutine to poll the empty queue.
		pollStart := make(chan bool, 1)
		pollDone := make(chan bool, 1)
		go func() {
			pollStart <- true
			Expect(queue.Poll(0)).Should(BeNil())
			pollDone <- true
		}()

		// Wait until goroutine starts.
		<-pollStart

		// Close queue.
		queue.Close()

		// Poll in goroutine should be unblocked and return.
		Eventually(pollDone).Should(Receive())

		// Any future Poll on empty queue will immediately return with nil.
		Expect(queue.Poll(0)).Should(BeNil())
	})
})
