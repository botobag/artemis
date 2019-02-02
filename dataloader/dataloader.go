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
	"context"
	"errors"
	"sync"

	"github.com/botobag/artemis/concurrent/future"
	"github.com/botobag/artemis/iterator"
)

// Key is an unique identifier of a value loaded by a DataLoader.
type Key interface{}

// Keys specifies a list of keys and provides an iterator over the keys.
type Keys interface {
	Iterator() KeyIterator
}

// KeysWithSize is a Keys with size hint.
type KeysWithSize interface {
	Keys
	Size() int
}

// KeyIterator is an iterator over keys in Keys.
type KeyIterator interface {
	// Next returns the next key in the iteration. It conforms the iterator pattern described in
	// iterator package.
	Next() (Key, error)
}

// keysArray is a return value for KeysFromArray which implements KeysWithSize.
type keysArray struct {
	keys []Key
}

type keysArrayIterator struct {
	keys []Key
	i    int
	size int
}

// Iterator implements Keys.
func (a keysArray) Iterator() KeyIterator {
	return &keysArrayIterator{
		keys: a.keys,
		i:    0,
		size: len(a.keys),
	}
}

// Size implements KeysWithSize.
func (a keysArray) Size() int {
	return len(a.keys)
}

// Next implements KeyIterator.
func (iter *keysArrayIterator) Next() (Key, error) {
	i := iter.i
	if i != iter.size {
		iter.i++
		return iter.keys[i], nil
	}
	return nil, iterator.Done
}

// KeysFromArray creates from an array of Key's.
func KeysFromArray(keys ...Key) KeysWithSize {
	return keysArray{keys}
}

type taskQueue struct {
	// DataLoader that creates and executes the tasks in the queue.
	loader *DataLoader

	// flag to indicate whether the queue has been dispatched; The flag is updated with CAS and only
	// the one who successfully changes the flag can dispatch the queue.
	dispatched bool

	// tasks stored in a linked list
	tasks TaskList
}

func newTaskQueue(loader *DataLoader) *taskQueue {
	return &taskQueue{
		loader: loader,
	}
}

func (queue *taskQueue) Enqueue(key Key) *Task {
	// Create a task.
	task := newTask(queue, key)

	// Try to insert it into cache.
	cacheMap := queue.loader.cacheMap
	if cacheMap != nil {
		cachedTask := cacheMap.Set(task)
		if cachedTask != task {
			// Task for the given key found in cache which has been enqueued. Return the cache one without
			// enqueuing.
			return cachedTask
		}
	}

	// Enqueue the task.
	queue.tasks.push(task)

	return task
}

func (queue *taskQueue) Empty() bool {
	return queue.tasks.Empty()
}

// A DataLoader loads data from a data backend with unique keys such as the id column of a SQL
// table.
type DataLoader struct {
	config *Config

	// Lock that guard accesses to queue
	queueMutex sync.Mutex

	// Queue containing the pending tasks for data loading
	queue *taskQueue

	// cacheMap caches loaded data. It is nil if the cache is disabled.
	cacheMap CacheMap
}

var (
	errMissingBatchLoader = errors.New("batch loader is required to construct a DataLoader")
	errMissingKey         = errors.New("must specify key to identify data to be loaded")
)

// New creates a DataLoader instance from given config.
func New(config Config) (*DataLoader, error) {
	// Check config.
	if config.BatchLoader == nil {
		return nil, errMissingBatchLoader
	}

	// Determine storage for cache.
	cacheMap := config.CacheMap
	if cacheMap == nil {
		// Create a DefaultCacheMap instance.
		cacheMap = &DefaultCacheMap{}
	} else if cacheMap == NoCacheMap {
		cacheMap = nil
	}

	loader := &DataLoader{
		config:   &config,
		cacheMap: cacheMap,
	}
	loader.queue = newTaskQueue(loader)

	return loader, nil
}

// BatchLoader returns loader.config.BatchLoader.
func (loader *DataLoader) BatchLoader() BatchLoader {
	return loader.config.BatchLoader
}

// Load loads a data identified by the key. It returns a Future for the value represented by that
// key.
func (loader *DataLoader) Load(key Key) (future.Future, error) {
	if key == nil {
		return nil, errMissingKey
	}

	// Check cache.
	cacheMap := loader.cacheMap
	if cacheMap != nil {
		task := cacheMap.Get(key)
		if task != nil {
			// Cache hit; Request a Future from task to access loaded value.
			return task.newFuture(), nil
		}
	}

	// Acquire the lock to enqueue the task.
	queueMutex := &loader.queueMutex
	queueMutex.Lock()

	// Enqueue a task.
	task := loader.queue.Enqueue(key)

	// Release lock.
	queueMutex.Unlock()

	// TODO: Check dispatch policy to see whether we should dispatch the queue immediately.

	return task.newFuture(), nil
}

// LoadMany loads collection of data identified by multiple keys. It returns a Future for the values
// represented by those keys.
func (loader *DataLoader) LoadMany(keys Keys) (future.Future, error) {
	var futures []future.Future

	// Pre-allocate futures when size hint is available.
	if keys, ok := keys.(KeysWithSize); ok {
		futures = make([]future.Future, 0, keys.Size())
	}

	keyIter := keys.Iterator()
	for {
		key, err := keyIter.Next()
		if err == iterator.Done {
			break
		} else if err != nil {
			return nil, err
		}

		f, err := loader.Load(key)
		if err != nil {
			return nil, err
		}

		futures = append(futures, f)
	}

	return future.Join(futures...), nil
}

// Dispatch dispatches jobs to load data specified by tasks in current queue as of the time this
// function is called.
func (loader *DataLoader) Dispatch(ctx context.Context) {
	loader.dispatchQueue(ctx, loader.queue)
}

// dispatchQueue tries to dispatch jobs to perform batch load for given queue. Note that the work
// can be performed by the one who successfully "detatches" the queue from the loader.
func (loader *DataLoader) dispatchQueue(ctx context.Context, queue *taskQueue) {
	// Acquire the lock to detatch the queue from the loader.
	queueMutex := &loader.queueMutex
	queueMutex.Lock()

	// Return quickly if someone has dispatched the given queue or the queue is empty.
	if queue != loader.queue || queue.Empty() {
		queueMutex.Unlock()
		return
	}

	// Set the dispatched flag.
	queue.dispatched = true

	// Replace with an empty queue.
	loader.queue = newTaskQueue(loader)
	queueMutex.Unlock()

	// Create jobs.
	maxBatchSize := loader.config.MaxBatchSize
	if maxBatchSize == 0 {
		loader.dispatchQueueBatch(ctx, queue.tasks)
	} else {
		var (
			tasks = queue.tasks
			// tasks will be split into some small sub-lists each of which has at most maxBatchSize tasks.
			// firstTask marks the first task of the sub-list in current batch.
			firstTask = tasks.first
			task      = firstTask
			counter   = maxBatchSize
		)

		for task != nil {
			nextTask := task.next

			counter--
			if counter == 0 {
				// Dispatch one job.
				loader.dispatchQueueBatch(ctx, TaskList{
					first: firstTask,
					last:  task,
				})

				// Reset counter.
				counter = maxBatchSize
				// Next batch starts from nextTask.
				firstTask = nextTask
			}

			// Move to the next task.
			task = nextTask
		}

		// Dispatch the last batch.
		if firstTask != nil {
			loader.dispatchQueueBatch(ctx, TaskList{
				first: firstTask,
			})
		}
	}
}

// dispatchQueue tries to dispatch jobs to perform batch load for given queue. Note that the work
// can be performed by the one who successfully "detatches" the queue from the loader.
func (loader *DataLoader) dispatchQueueBatch(ctx context.Context, tasks TaskList) error {
	job := &BatchLoadJob{
		ctx:   ctx,
		tasks: tasks,
	}

	runner := loader.config.Runner
	if runner == nil {
		// Run the job with current goroutine.
		if _, err := job.Run(); err != nil {
			return err
		}
	} else {
		if _, err := runner.Submit(job); err != nil {
			return err
		}
	}

	return nil
}

// Clear the value for the given key from the cache.
func (loader *DataLoader) Clear(key Key) {
	cacheMap := loader.cacheMap
	if cacheMap != nil {
		cacheMap.Delete(key)
	}
}

// ClearAll clears the entire cache.
func (loader *DataLoader) ClearAll() {
	cacheMap := loader.cacheMap
	if cacheMap != nil {
		cacheMap.Clear()
	}
}

// Prime adds the provided key and value to the cache. If the key already exists, no change is made.
func (loader *DataLoader) Prime(key Key, value interface{}) error {
	cacheMap := loader.cacheMap
	if cacheMap != nil {
		// Create a task.
		task := newTask(nil, key)

		// Complete the task with the value.
		if err := task.Complete(value); err != nil {
			return err
		}

		// Add to the cache.
		cacheMap.Set(task)
	}

	return nil
}

// PrimeError adds the provided key with an error value to the cache. If the key already exists, no
// change is made.
func (loader *DataLoader) PrimeError(key Key, err error) error {
	cacheMap := loader.cacheMap
	if cacheMap != nil {
		// Create a task.
		task := newTask(nil, key)

		// Complete the task with an error value.
		if err := task.SetError(err); err != nil {
			return err
		}

		// Add to the cache.
		cacheMap.Set(task)
	}

	return nil
}
