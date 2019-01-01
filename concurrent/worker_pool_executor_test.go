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

package concurrent_test

import (
	"runtime"
	"sync/atomic"

	"github.com/botobag/artemis/concurrent"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("WorkerPoolExecutor", func() {
	It("cannot be created with invalid pool size", func() {
		var err error

		_, err = concurrent.NewWorkerPoolExecutor(concurrent.WorkerPoolExecutorConfig{})
		Expect(err.Error()).Should(ContainSubstring("MaxPoolSize must be a non-zero value"))

		_, err = concurrent.NewWorkerPoolExecutor(concurrent.WorkerPoolExecutorConfig{
			MaxPoolSize: 50,
			MinPoolSize: 100,
		})
		Expect(err.Error()).Should(ContainSubstring("MaxPoolSize (50) should be greater than MinPoolSize (100)"))
	})

	It("can execute a task without pool", func() {
		executor, err := concurrent.NewWorkerPoolExecutor(concurrent.WorkerPoolExecutorConfig{
			MinPoolSize: 0,
			MaxPoolSize: uint32(runtime.GOMAXPROCS(-1)),
		})
		Expect(err).ShouldNot(HaveOccurred())

		task := concurrent.TaskFunc(func() (interface{}, error) {
			return "task result", nil
		})
		handle, err := executor.Submit(task)
		Expect(err).ShouldNot(HaveOccurred())

		// Check the execution result.
		Expect(handle.AwaitResult(0)).Should(Equal("task result"))

		Expect(shutdownExecutor(executor)).Should(Succeed())
	})

	It("can execute multiple tasks with pool", func() {
		executor, err := concurrent.NewWorkerPoolExecutor(concurrent.WorkerPoolExecutorConfig{
			MinPoolSize: 4,
			MaxPoolSize: 8,
		})
		Expect(err).ShouldNot(HaveOccurred())

		var x int32
		task := concurrent.TaskFunc(func() (interface{}, error) {
			atomic.AddInt32(&x, 1)
			return nil, nil
		})

		// Execute the task TIMES times.
		const TIMES = 100

		// Dispatch 100 tasks.
		for i := 0; i < TIMES; i++ {
			_, err := executor.Submit(task)
			Expect(err).ShouldNot(HaveOccurred())
		}

		// Shutdown the executor and wait until termination.
		Expect(shutdownExecutor(executor)).Should(Succeed())

		// Check the result.
		Expect(x).Should(Equal(int32(TIMES)))
	})

	It("can cancel a task", func() {
		// Create an executor with pool size 1.
		executor, err := concurrent.NewWorkerPoolExecutor(concurrent.WorkerPoolExecutorConfig{
			MinPoolSize: 1,
			MaxPoolSize: 1,
		})
		Expect(err).ShouldNot(HaveOccurred())

		// Push 2 tasks. The first task will stuck the only worker in the pool and leave the 2nd task in
		// the queue. The removal of 2nd task should succeed.
		stopFirstTask := make(chan bool, 1)
		enterFirstTask := make(chan bool, 1)
		firstTask := concurrent.TaskFunc(func() (interface{}, error) {
			enterFirstTask <- true
			<-stopFirstTask
			return "first task result", nil
		})

		secondTask := concurrent.TaskFunc(func() (interface{}, error) {
			return "second task", nil
		})

		// Push the first task.
		firstTaskHandle, err := executor.Submit(firstTask)
		Expect(err).ShouldNot(HaveOccurred())

		// Wait until the first task is executed.
		<-enterFirstTask

		// We cannot cancel the first task because it is being executed.
		Expect(firstTaskHandle.Cancel()).ShouldNot(Succeed())

		// Push the second task.
		secondTaskHandle, err := executor.Submit(secondTask)
		Expect(err).ShouldNot(HaveOccurred())

		// Cancel the second task.
		Expect(secondTaskHandle.Cancel()).Should(Succeed())

		// Resume first task.
		stopFirstTask <- true

		// Shutdown the executor.
		Expect(shutdownExecutor(executor)).Should(Succeed())

		// Check result.
		Expect(firstTaskHandle.AwaitResult(0)).Should(Equal("first task result"))

		_, secondTaskResult := secondTaskHandle.AwaitResult(0)
		Expect(secondTaskResult).Should(MatchError(concurrent.ErrTaskCancelled))
	})

	It("allows calling shutdown multiple times", func() {
		executor, err := concurrent.NewWorkerPoolExecutor(concurrent.WorkerPoolExecutorConfig{
			MinPoolSize: 0,
			MaxPoolSize: uint32(runtime.GOMAXPROCS(-1)),
		})
		Expect(err).ShouldNot(HaveOccurred())

		// Push some dummy tasks to executor.
		dummyTask := concurrent.TaskFunc(func() (interface{}, error) {
			return "dummy task", nil
		})
		producerDone := make(chan bool, 1)
		go func() {
			for i := 0; i < 100; i++ {
				executor.Submit(dummyTask)
			}
			producerDone <- true
		}()

		const NumShutdownRequests = 10
		terminations := make([]<-chan bool, NumShutdownRequests)
		for i := 0; i < NumShutdownRequests; i++ {
			var err error
			terminations[i], err = executor.Shutdown()
			Expect(err).ShouldNot(HaveOccurred())
		}

		// Block on all terminations.
		for _, termination := range terminations {
			<-termination
		}

		// Wait for producer.
		<-producerDone
	})

	It("allows shutdown after termination", func() {
		executor, err := concurrent.NewWorkerPoolExecutor(concurrent.WorkerPoolExecutorConfig{
			MinPoolSize: 0,
			MaxPoolSize: uint32(runtime.GOMAXPROCS(-1)),
		})
		Expect(err).ShouldNot(HaveOccurred())

		// Shutdown the executor.
		Expect(shutdownExecutor(executor)).Should(Succeed())

		// Shutdown again.
		Expect(shutdownExecutor(executor)).Should(Succeed())
	})

	It("cannot submit task after shutdown", func() {
		executor, err := concurrent.NewWorkerPoolExecutor(concurrent.WorkerPoolExecutorConfig{
			MinPoolSize: 0,
			MaxPoolSize: uint32(runtime.GOMAXPROCS(-1)),
		})
		Expect(err).ShouldNot(HaveOccurred())

		// Push a task which will start execution before shutdown.
		stopTask := make(chan bool, 1)
		enterTask := make(chan bool, 1)
		task := concurrent.TaskFunc(func() (interface{}, error) {
			enterTask <- true
			<-stopTask
			return "task executed before shutdown", nil
		})

		// Push the task.
		taskHandle, err := executor.Submit(task)
		Expect(err).ShouldNot(HaveOccurred())

		// Wait until the task is executed.
		<-enterTask

		// Shutdown the executor.
		terminated, err := executor.Shutdown()
		Expect(err).ShouldNot(HaveOccurred())
		Expect(terminated).ShouldNot(Receive())

		// Push a task which will fail.
		_, err = executor.Submit(concurrent.TaskFunc(func() (interface{}, error) {
			return "task shouldn't be executed", nil
		}))
		Expect(err).Should(HaveOccurred())

		// Finish task.
		stopTask <- true

		// Check result.
		Eventually(terminated).Should(Receive())
		Expect(taskHandle.AwaitResult(0)).Should(Equal("task executed before shutdown"))
	})
})
