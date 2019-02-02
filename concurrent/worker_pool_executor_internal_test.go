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
		// Fork number of n goroutines to push the tasks to the queue.
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
	var (
		// Mutex that guards accesses to taskMap and numTasks.
		mutex    sync.Mutex
		taskMap  = map[Task]bool{}
		numTasks = int64(len(tasks))
	)

	// Build task map for checking results.
	for _, task := range tasks {
		taskMap[task] = true
	}

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				// Lock mutex to check numTasks.
				mutex.Lock()
				if numTasks == 0 {
					queue.Close()
					mutex.Unlock()
					break
				}
				mutex.Unlock()

				task, err := queue.Poll(0)
				Expect(err).ShouldNot(HaveOccurred())
				if task == nil {
					continue
				}

				// Lock mutex to access taskMap and decrement numTasks.
				mutex.Lock()
				Expect(taskMap).Should(HaveKey(task))
				delete(taskMap, task.(Task))
				numTasks--
				mutex.Unlock()
			}
		}()
	}

	for i := 0; i < numRemovers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				// Select task to be removed randomly.
				task := tasks[rand.Int31n(int32(len(tasks)))]

				// Lock mutex to check numTasks and taskMap to see whether the specified task is removed.
				mutex.Lock()
				if numTasks == 0 {
					queue.Close()
					mutex.Unlock()
					break
				}
				_, exists := taskMap[task.(Task)]
				mutex.Unlock()

				// Remove.
				err := queue.Remove(task)
				if !exists {
					Expect(err).Should(MatchError(ErrElementNotFound))
				} else {
					Expect(err).Should(Or(BeNil(), MatchError(ErrElementNotFound)))

					if err == nil {
						// Successfully removed. Update taskMap and numTasks.
						mutex.Lock()
						Expect(taskMap).Should(HaveKey(task))
						delete(taskMap, task.(Task))
						// Decrement numTasks.
						numTasks--
						mutex.Unlock()
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
