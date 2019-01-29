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

package dataloader_test

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/botobag/artemis/concurrent/future"
	"github.com/botobag/artemis/dataloader"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// The following tests are derived from:
//
//   https://github.com/facebook/dataloader/blob/420e62f/src/__tests__/dataloader-test.js.
//
// The license (BSD license) is reproduced as follows,
//
// Copyright (c) 2015, Facebook, Inc. All rights reserved.
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
//  * Redistributions of source code must retain the above copyright notice, this
//    list of conditions and the following disclaimer.
//
//  * Redistributions in binary form must reproduce the above copyright notice,
//    this list of conditions and the following disclaimer in the documentation
//    and/or other materials provided with the distribution.
//
//  * Neither the name Facebook nor the names of its contributors may be used to
//    endorse or promote products derived from this software without specific
//    prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR
// ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON
// ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

type batchLoadLogger struct {
	// mutex that guards loadCalls
	loadCallsMutex sync.Mutex

	// keys that have been sent to identityLoader to load data
	loadCalls [][]dataloader.Key
}

func (logger *batchLoadLogger) LoadCalls() [][]dataloader.Key {
	mutex := &logger.loadCallsMutex
	mutex.Lock()
	defer mutex.Unlock()
	return logger.loadCalls
}

func (logger *batchLoadLogger) LogKeys(tasks *dataloader.TaskList) {
	var (
		mutex = &logger.loadCallsMutex

		// Collect keys.
		keys []dataloader.Key
	)

	for taskIter, taskEnd := tasks.Begin(), tasks.End(); taskIter != taskEnd; taskIter = taskIter.Next() {
		keys = append(keys, taskIter.Task.Key())
	}

	// Acquire lock to append keys to loader.loadCalls.
	mutex.Lock()
	logger.loadCalls = append(logger.loadCalls, keys)
	mutex.Unlock()
}

//===----------------------------------------------------------------------------------------====//
// identityLoader
//===----------------------------------------------------------------------------------------====//

// identityBatchLoader implements dataloader.BatchLoader which simply returns key as the loaded
// value. It also logs the batch load keys that sent to the loader.
type identityBatchLoader struct {
	logger batchLoadLogger
}

func (loader *identityBatchLoader) Load(ctx context.Context, tasks *dataloader.TaskList) {
	// Complete task with its key as loaded value.
	for taskIter, taskEnd := tasks.Begin(), tasks.End(); taskIter != taskEnd; taskIter = taskIter.Next() {
		task := taskIter.Task
		task.Complete(task.Key())
	}

	// Log keys for check.
	loader.logger.LogKeys(tasks)
}

func (loader *identityBatchLoader) LoadCalls() [][]dataloader.Key {
	return loader.logger.LoadCalls()
}

// DataLoader that uses identityBatchLoader for batch load.
type identityLoader struct {
	*dataloader.DataLoader
}

func (loader identityLoader) LoadCalls() [][]dataloader.Key {
	return loader.BatchLoader().(*identityBatchLoader).LoadCalls()
}

func newIdentityLoader(config dataloader.Config) identityLoader {
	Expect(config.BatchLoader).Should(BeNil())
	config.BatchLoader = &identityBatchLoader{}

	loader, err := dataloader.New(config)
	Expect(err).ShouldNot(HaveOccurred())

	return identityLoader{loader}
}

//===----------------------------------------------------------------------------------------====//
// evenLoader
//===----------------------------------------------------------------------------------------====//

// evenBatchLoader implements dataloader.BatchLoader which returns key as the loaded value for key
// that is an even number and an error otherwise.
type evenBatchLoader struct {
	logger batchLoadLogger
}

func (loader *evenBatchLoader) Load(ctx context.Context, tasks *dataloader.TaskList) {
	// Complete task with its key as loaded value.
	for taskIter, taskEnd := tasks.Begin(), tasks.End(); taskIter != taskEnd; taskIter = taskIter.Next() {
		task := taskIter.Task
		key := task.Key()
		if key, ok := key.(int); ok && key%2 == 0 {
			task.Complete(key)
		} else {
			task.SetError(fmt.Errorf("Odd: %+v", key))
		}
	}

	// Log keys for check.
	loader.logger.LogKeys(tasks)
}

func (loader *evenBatchLoader) LoadCalls() [][]dataloader.Key {
	return loader.logger.LoadCalls()
}

// DataLoader that uses evenBatchLoader for batch load.
type evenLoader struct {
	*dataloader.DataLoader
}

func (loader evenLoader) LoadCalls() [][]dataloader.Key {
	return loader.BatchLoader().(*evenBatchLoader).LoadCalls()
}

func newEvenLoader(config dataloader.Config) evenLoader {
	Expect(config.BatchLoader).Should(BeNil())
	config.BatchLoader = &evenBatchLoader{}

	loader, err := dataloader.New(config)
	Expect(err).ShouldNot(HaveOccurred())

	return evenLoader{loader}
}

//===----------------------------------------------------------------------------------------====//
// errorLoader
//===----------------------------------------------------------------------------------------====//

// errorBatchLoader implements dataloader.BatchLoader which always resolves loaded value to an
// error.
type errorBatchLoader struct {
	logger batchLoadLogger
}

func (loader *errorBatchLoader) Load(ctx context.Context, tasks *dataloader.TaskList) {
	// Complete task with its key as loaded value.
	for taskIter, taskEnd := tasks.Begin(), tasks.End(); taskIter != taskEnd; taskIter = taskIter.Next() {
		task := taskIter.Task
		key := task.Key()
		task.SetError(fmt.Errorf("Error: %+v", key))
	}

	// Log keys for check.
	loader.logger.LogKeys(tasks)
}

func (loader *errorBatchLoader) LoadCalls() [][]dataloader.Key {
	return loader.logger.LoadCalls()
}

// DataLoader that uses errorBatchLoader for batch load.
type errorLoader struct {
	*dataloader.DataLoader
}

func (loader errorLoader) LoadCalls() [][]dataloader.Key {
	return loader.BatchLoader().(*errorBatchLoader).LoadCalls()
}

func newErrorLoader(config dataloader.Config) errorLoader {
	Expect(config.BatchLoader).Should(BeNil())
	config.BatchLoader = &errorBatchLoader{}

	loader, err := dataloader.New(config)
	Expect(err).ShouldNot(HaveOccurred())

	return errorLoader{loader}
}

//===----------------------------------------------------------------------------------------====//
// cacheInvalidateLoader
//===----------------------------------------------------------------------------------------====//

// cacheInvalidateBatchLoader is an identityBatchLoader but clear loader cache after a batch load.
type cacheInvalidateBatchLoader struct {
	identityBatchLoader
	loader *dataloader.DataLoader
}

func (loader *cacheInvalidateBatchLoader) Load(ctx context.Context, tasks *dataloader.TaskList) {
	loader.identityBatchLoader.Load(ctx, tasks)

	// Reset loader cache.
	loader.loader.ClearAll()
}

// DataLoader that uses cacheInvalidateBatchLoader for batch load.
type cacheInvalidateLoader struct {
	*dataloader.DataLoader
}

func (loader cacheInvalidateLoader) LoadCalls() [][]dataloader.Key {
	return loader.BatchLoader().(*cacheInvalidateBatchLoader).LoadCalls()
}

func newCacheInvalidateLoader(config dataloader.Config) cacheInvalidateLoader {
	batchLoader := &cacheInvalidateBatchLoader{}

	Expect(config.BatchLoader).Should(BeNil())
	config.BatchLoader = batchLoader

	loader, err := dataloader.New(config)
	Expect(err).ShouldNot(HaveOccurred())

	batchLoader.loader = loader

	return cacheInvalidateLoader{loader}
}

//===----------------------------------------------------------------------------------------====//
// chainLoader
//===----------------------------------------------------------------------------------------====//

// chainBatchLoader is an identityBatchLoader but clear loader cache after a batch load.
type chainBatchLoader struct {
	logger     batchLoadLogger
	deepLoader *dataloader.DataLoader
}

func (loader *chainBatchLoader) Load(ctx context.Context, tasks *dataloader.TaskList) {
	// Collect keys.
	var keys []dataloader.Key
	for taskIter, taskEnd := tasks.Begin(), tasks.End(); taskIter != taskEnd; taskIter = taskIter.Next() {
		task := taskIter.Task
		keys = append(keys, task.Key())
	}

	// Call deepLoader to load values.
	deepLoader := loader.deepLoader
	f, err := deepLoader.LoadMany(keys)
	Expect(err).ShouldNot(HaveOccurred())

	// Dispatch tasks.
	go deepLoader.Dispatch(context.Background())

	// Block on f to wait for completion.
	values, err := future.BlockOn(f)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(values).Should(HaveLen(len(keys)))

	// Complete task.
	taskIter := tasks.Begin()
	for _, value := range values.([]interface{}) {
		taskIter.Task.Complete(value)
		taskIter = taskIter.Next()
	}

	// Log keys for check.
	loader.logger.LogKeys(tasks)
}

func (loader *chainBatchLoader) LoadCalls() [][]dataloader.Key {
	return loader.logger.LoadCalls()
}

// DataLoader that uses chainBatchLoader for batch load.
type chainLoader struct {
	*dataloader.DataLoader
}

func (loader chainLoader) LoadCalls() [][]dataloader.Key {
	return loader.BatchLoader().(*chainBatchLoader).LoadCalls()
}

func newChainLoader(deepLoader *dataloader.DataLoader) chainLoader {
	config := dataloader.Config{
		BatchLoader: &chainBatchLoader{
			deepLoader: deepLoader,
		},
	}

	loader, err := dataloader.New(config)
	Expect(err).ShouldNot(HaveOccurred())

	return chainLoader{loader}
}

//===----------------------------------------------------------------------------------------====//
// customKey
//===----------------------------------------------------------------------------------------====//

type customKey struct {
	ID int
}

// KeyForCache implements dataloader.KeyWithCustomCacheKey.
func (k *customKey) KeyForCache() interface{} {
	return fmt.Sprintf("id:%d", k.ID)
}

type customKeyAB struct {
	A int
	B int
}

// KeyForCache implements dataloader.KeyWithCustomCacheKey.
func (k *customKeyAB) KeyForCache() interface{} {
	return fmt.Sprintf("a:%d,b:%d", k.A, k.B)
}

type customKeyBA struct {
	B int
	A int
}

// KeyForCache implements dataloader.KeyWithCustomCacheKey.
func (k *customKeyBA) KeyForCache() interface{} {
	return fmt.Sprintf("a:%d,b:%d", k.A, k.B)
}

//===----------------------------------------------------------------------------------------====//
// SimpleMap
//===----------------------------------------------------------------------------------------====//

// OrderedMap is a Go's map and provides a function to get list of keys in their insertion order in
// addition.
type OrderedMap struct {
	keys   []dataloader.Key
	values map[dataloader.Key]*dataloader.Task
}

func newOrderedMap() *OrderedMap {
	return &OrderedMap{
		values: map[dataloader.Key]*dataloader.Task{},
	}
}

func (o *OrderedMap) Get(key dataloader.Key) *dataloader.Task {
	return o.values[key]
}

func (o *OrderedMap) Set(task *dataloader.Task) *dataloader.Task {
	key := task.Key()
	if _, exists := o.values[key]; !exists {
		o.keys = append(o.keys, key)
	}
	// Follow facebook/dataloader/src/__tests__/dataloader-test.js which always inserts the value to
	// the map.
	o.values[key] = task
	return task
}

func (o *OrderedMap) Delete(key dataloader.Key) {
	// Check the key.
	_, exists := o.values[key]
	if !exists {
		return
	}

	// Removek key from o.keys.
	for i, k := range o.keys {
		if k == key {
			o.keys = append(o.keys[:i], o.keys[i+1:]...)
			break
		}
	}

	// Remove value from o.values.
	delete(o.values, key)
}

func (o *OrderedMap) Keys() []dataloader.Key {
	return o.keys
}

func (o *OrderedMap) Clear() {
	o.keys = nil
	o.values = map[dataloader.Key]*dataloader.Task{}
}

type SimpleMap struct {
	Stash *OrderedMap
}

var _ dataloader.CacheMap = SimpleMap{}

func newSimpleMap() SimpleMap {
	return SimpleMap{
		Stash: newOrderedMap(),
	}
}

// Get implements CacheMap.
func (m SimpleMap) Get(key dataloader.Key) *dataloader.Task {
	return m.Stash.Get(key)
}

// Set implements CacheMap.
func (m SimpleMap) Set(task *dataloader.Task) *dataloader.Task {
	return m.Stash.Set(task)
}

// Delete implements CacheMap.
func (m SimpleMap) Delete(key dataloader.Key) {
	m.Stash.Delete(key)
}

// Clear implements CacheMap.
func (m SimpleMap) Clear() {
	m.Stash.Clear()
}

var _ = Describe("DataLoader: Primary API", func() {
	var idLoader identityLoader

	BeforeEach(func() {
		idLoader = newIdentityLoader(dataloader.Config{})
	})

	It("throws error if batch loader is not given", func() {
		_, err := dataloader.New(dataloader.Config{})
		Expect(err).Should(MatchError("batch loader is required to construct a DataLoader"))
	})

	It("throws error when load with nil key", func() {
		_, err := idLoader.Load(nil)
		Expect(err).Should(HaveOccurred())

		_, err = idLoader.LoadMany([]dataloader.Key{nil})
		Expect(err).Should(HaveOccurred())
	})

	It("throws error if a task is completed value multiple times", func() {
		loader, err := dataloader.New(dataloader.Config{
			BatchLoader: dataloader.BatchLoadFunc(func(ctx context.Context, tasks *dataloader.TaskList) {
				task := tasks.Begin().Task

				// Complete task with its key (first time, should be ok).
				Expect(task.Complete(task.Key())).Should(Succeed())

				// Complete task with its key (second time, should fail).
				Expect(task.Complete(task.Key())).Should(MatchError("task was already completed with a value (1) but want to accept a value (1)"))

				// Complete task with an error value (second time, should fail).
				Expect(task.SetError(errors.New("Error"))).Should(MatchError("task was already completed with a value (1) but want to accept an error (Error)"))
			}),
		})
		Expect(err).ShouldNot(HaveOccurred())

		f, err := loader.Load(1)
		Expect(err).ShouldNot(HaveOccurred())

		go loader.Dispatch(context.Background())
		Expect(future.BlockOn(f)).Should(Equal(1))
	})

	It("throws error if a task is not completed by the supplied batch loader", func() {
		loader, err := dataloader.New(dataloader.Config{
			BatchLoader: dataloader.BatchLoadFunc(func(ctx context.Context, tasks *dataloader.TaskList) {
				// All tasks remains unfinished on return.
				for taskIter, taskEnd := tasks.Begin(), tasks.End(); taskIter != taskEnd; taskIter = taskIter.Next() {
					Expect(taskIter.Task.Completed()).Should(BeFalse())
				}
			}),
		})

		Expect(err).ShouldNot(HaveOccurred())

		f, err := loader.Load(1)
		Expect(err).ShouldNot(HaveOccurred())

		go loader.Dispatch(context.Background())
		_, err = future.BlockOn(f)
		Expect(err).Should(MatchError("dataloader.BatchLoadFunc must complete every given data loading task with either a value or an error but it doesn't complete task that loads data at key 1"))
	})

	It("builds a really really simple data loader", func() {
		f, err := idLoader.Load(1)
		Expect(err).ShouldNot(HaveOccurred())

		go idLoader.Dispatch(context.Background())
		Expect(future.BlockOn(f)).Should(Equal(1))
	})

	It("supports loading multiple keys in one call", func() {
		f, err := idLoader.LoadMany([]dataloader.Key{1, 2})
		Expect(err).ShouldNot(HaveOccurred())
		go idLoader.Dispatch(context.Background())
		Expect(future.BlockOn(f)).Should(Equal([]interface{}{1, 2}))

		f, err = idLoader.LoadMany([]dataloader.Key{})
		Expect(err).ShouldNot(HaveOccurred())
		go idLoader.Dispatch(context.Background())
		Expect(future.BlockOn(f)).Should(BeEmpty())
	})

	It("batches multiple requests", func() {
		f1, err := idLoader.Load(1)
		Expect(err).ShouldNot(HaveOccurred())

		f2, err := idLoader.Load(2)
		Expect(err).ShouldNot(HaveOccurred())

		go idLoader.Dispatch(context.Background())
		Expect(future.BlockOn(future.Join(f1, f2))).Should(Equal([]interface{}{1, 2}))

		Expect(idLoader.LoadCalls()).Should(Equal([][]dataloader.Key{
			{1, 2},
		}))
	})

	It("batches multiple requests with max batch sizes", func() {
		idLoader := newIdentityLoader(dataloader.Config{
			MaxBatchSize: 2,
		})

		f1, err := idLoader.Load(1)
		Expect(err).ShouldNot(HaveOccurred())

		f2, err := idLoader.Load(2)
		Expect(err).ShouldNot(HaveOccurred())

		f3, err := idLoader.Load(3)
		Expect(err).ShouldNot(HaveOccurred())

		go idLoader.Dispatch(context.Background())
		Expect(future.BlockOn(future.Join(f1, f2, f3))).Should(Equal([]interface{}{1, 2, 3}))

		Expect(idLoader.LoadCalls()).Should(Equal([][]dataloader.Key{
			{1, 2},
			{3},
		}))
	})

	It("coalesces identical requests", func() {
		f1a, err := idLoader.Load(1)
		Expect(err).ShouldNot(HaveOccurred())

		f1b, err := idLoader.Load(1)
		Expect(err).ShouldNot(HaveOccurred())

		go idLoader.Dispatch(context.Background())
		Expect(future.BlockOn(future.Join(f1a, f1b))).Should(Equal([]interface{}{1, 1}))

		Expect(idLoader.LoadCalls()).Should(Equal([][]dataloader.Key{
			{1},
		}))
	})

	It("caches repeated requests", func() {
		a, err := idLoader.Load("A")
		Expect(err).ShouldNot(HaveOccurred())

		b, err := idLoader.Load("B")
		Expect(err).ShouldNot(HaveOccurred())

		go idLoader.Dispatch(context.Background())
		Expect(future.BlockOn(future.Join(a, b))).Should(Equal([]interface{}{"A", "B"}))

		Expect(idLoader.LoadCalls()).Should(Equal([][]dataloader.Key{
			{"A", "B"},
		}))

		a2, err := idLoader.Load("A")
		Expect(err).ShouldNot(HaveOccurred())

		c, err := idLoader.Load("C")
		Expect(err).ShouldNot(HaveOccurred())

		go idLoader.Dispatch(context.Background())
		Expect(future.BlockOn(future.Join(a2, c))).Should(Equal([]interface{}{"A", "C"}))

		Expect(idLoader.LoadCalls()).Should(Equal([][]dataloader.Key{
			{"A", "B"},
			{"C"},
		}))

		a3, err := idLoader.Load("A")
		Expect(err).ShouldNot(HaveOccurred())

		b2, err := idLoader.Load("B")
		Expect(err).ShouldNot(HaveOccurred())

		c2, err := idLoader.Load("C")
		Expect(err).ShouldNot(HaveOccurred())

		go idLoader.Dispatch(context.Background())
		Expect(future.BlockOn(future.Join(a3, b2, c2))).Should(Equal([]interface{}{"A", "B", "C"}))

		Expect(idLoader.LoadCalls()).Should(Equal([][]dataloader.Key{
			{"A", "B"},
			{"C"},
		}))
	})

	It("clears single value in loader", func() {
		a, err := idLoader.Load("A")
		Expect(err).ShouldNot(HaveOccurred())

		b, err := idLoader.Load("B")
		Expect(err).ShouldNot(HaveOccurred())

		go idLoader.Dispatch(context.Background())
		Expect(future.BlockOn(future.Join(a, b))).Should(Equal([]interface{}{"A", "B"}))

		Expect(idLoader.LoadCalls()).Should(Equal([][]dataloader.Key{
			{"A", "B"},
		}))

		idLoader.Clear("A")

		a2, err := idLoader.Load("A")
		Expect(err).ShouldNot(HaveOccurred())

		b2, err := idLoader.Load("B")
		Expect(err).ShouldNot(HaveOccurred())

		go idLoader.Dispatch(context.Background())
		Expect(future.BlockOn(future.Join(a2, b2))).Should(Equal([]interface{}{"A", "B"}))

		Expect(idLoader.LoadCalls()).Should(Equal([][]dataloader.Key{
			{"A", "B"},
			{"A"},
		}))
	})

	It("clears all values in loader", func() {
		a, err := idLoader.Load("A")
		Expect(err).ShouldNot(HaveOccurred())

		b, err := idLoader.Load("B")
		Expect(err).ShouldNot(HaveOccurred())

		go idLoader.Dispatch(context.Background())
		Expect(future.BlockOn(future.Join(a, b))).Should(Equal([]interface{}{"A", "B"}))

		Expect(idLoader.LoadCalls()).Should(Equal([][]dataloader.Key{
			{"A", "B"},
		}))

		idLoader.ClearAll()

		a2, err := idLoader.Load("A")
		Expect(err).ShouldNot(HaveOccurred())

		b2, err := idLoader.Load("B")
		Expect(err).ShouldNot(HaveOccurred())

		go idLoader.Dispatch(context.Background())
		Expect(future.BlockOn(future.Join(a2, b2))).Should(Equal([]interface{}{"A", "B"}))

		Expect(idLoader.LoadCalls()).Should(Equal([][]dataloader.Key{
			{"A", "B"},
			{"A", "B"},
		}))
	})

	It("allows priming the cache", func() {
		Expect(idLoader.Prime("A", "A")).Should(Succeed())

		a, err := idLoader.Load("A")
		Expect(err).ShouldNot(HaveOccurred())

		b, err := idLoader.Load("B")
		Expect(err).ShouldNot(HaveOccurred())

		go idLoader.Dispatch(context.Background())
		Expect(future.BlockOn(future.Join(a, b))).Should(Equal([]interface{}{"A", "B"}))

		Expect(idLoader.LoadCalls()).Should(Equal([][]dataloader.Key{
			{"B"},
		}))
	})

	It("does not prime keys that already exist", func() {
		Expect(idLoader.Prime("A", "X")).Should(Succeed())

		a1, err := idLoader.Load("A")
		Expect(err).ShouldNot(HaveOccurred())

		b1, err := idLoader.Load("B")
		Expect(err).ShouldNot(HaveOccurred())

		go idLoader.Dispatch(context.Background())
		Expect(future.BlockOn(a1)).Should(Equal("X"))
		Expect(future.BlockOn(b1)).Should(Equal("B"))

		Expect(idLoader.Prime("A", "Y")).Should(Succeed())
		Expect(idLoader.Prime("B", "Y")).Should(Succeed())

		a2, err := idLoader.Load("A")
		Expect(err).ShouldNot(HaveOccurred())

		b2, err := idLoader.Load("B")
		Expect(err).ShouldNot(HaveOccurred())

		go idLoader.Dispatch(context.Background())
		Expect(future.BlockOn(a2)).Should(Equal("X"))
		Expect(future.BlockOn(b2)).Should(Equal("B"))

		Expect(idLoader.LoadCalls()).Should(Equal([][]dataloader.Key{
			{"B"},
		}))
	})

	It("allows forcefully priming the cache", func() {
		Expect(idLoader.Prime("A", "X")).Should(Succeed())

		a1, err := idLoader.Load("A")
		Expect(err).ShouldNot(HaveOccurred())

		b1, err := idLoader.Load("B")
		Expect(err).ShouldNot(HaveOccurred())

		go idLoader.Dispatch(context.Background())
		Expect(future.BlockOn(a1)).Should(Equal("X"))
		Expect(future.BlockOn(b1)).Should(Equal("B"))

		idLoader.Clear("A")
		idLoader.Clear("B")
		Expect(idLoader.Prime("A", "Y")).Should(Succeed())
		Expect(idLoader.Prime("B", "Y")).Should(Succeed())

		a2, err := idLoader.Load("A")
		Expect(err).ShouldNot(HaveOccurred())

		b2, err := idLoader.Load("B")
		Expect(err).ShouldNot(HaveOccurred())

		go idLoader.Dispatch(context.Background())
		Expect(future.BlockOn(a2)).Should(Equal("Y"))
		Expect(future.BlockOn(b2)).Should(Equal("Y"))

		Expect(idLoader.LoadCalls()).Should(Equal([][]dataloader.Key{
			{"B"},
		}))
	})

	Describe("Represents Errors", func() {
		It("resolves to error to indicate failure", func() {
			evenLoader := newEvenLoader(dataloader.Config{})

			f1, err := evenLoader.Load(1)
			Expect(err).ShouldNot(HaveOccurred())

			go evenLoader.Dispatch(context.Background())
			_, err = future.BlockOn(f1)
			Expect(err).Should(MatchError("Odd: 1"))

			f2, err := evenLoader.Load(2)
			Expect(err).ShouldNot(HaveOccurred())

			go evenLoader.Dispatch(context.Background())
			Expect(future.BlockOn(f2)).Should(Equal(2))

			Expect(evenLoader.LoadCalls()).Should(Equal([][]dataloader.Key{
				{1},
				{2},
			}))
		})

		It("can represent failures and successes simultaneously", func() {
			evenLoader := newEvenLoader(dataloader.Config{})

			f1, err := evenLoader.Load(1)
			Expect(err).ShouldNot(HaveOccurred())

			f2, err := evenLoader.Load(2)
			Expect(err).ShouldNot(HaveOccurred())

			go evenLoader.Dispatch(context.Background())

			_, err = future.BlockOn(f1)
			Expect(err).Should(MatchError("Odd: 1"))

			Expect(future.BlockOn(f2)).Should(Equal(2))

			Expect(evenLoader.LoadCalls()).Should(Equal([][]dataloader.Key{
				{1, 2},
			}))
		})

		It("caches failed fetches", func() {
			errorLoader := newErrorLoader(dataloader.Config{})

			f1, err := errorLoader.Load(1)
			Expect(err).ShouldNot(HaveOccurred())

			go errorLoader.Dispatch(context.Background())
			_, caughtErrorA := future.BlockOn(f1)
			Expect(caughtErrorA).Should(MatchError("Error: 1"))

			f2, err := errorLoader.Load(1)
			Expect(err).ShouldNot(HaveOccurred())

			go errorLoader.Dispatch(context.Background())
			_, caughtErrorB := future.BlockOn(f2)
			Expect(caughtErrorB).Should(MatchError("Error: 1"))

			Expect(errorLoader.LoadCalls()).Should(Equal([][]dataloader.Key{
				{1},
			}))
		})

		It("handles priming the cache with an error", func() {
			idLoader.PrimeError(1, errors.New("Error: 1"))

			f1, err := idLoader.Load(1)
			Expect(err).ShouldNot(HaveOccurred())
			_, caughtErrorA := future.BlockOn(f1)
			Expect(caughtErrorA).Should(MatchError("Error: 1"))

			Expect(idLoader.LoadCalls()).Should(BeEmpty())
		})

		It("can clear values from cache after errors", func() {
			// TODO: Add MapErr operator to future.
		})

		It("propagates error to all loads", func() {
			// TODO
		})

		Describe("Accepts any kind of key", func() {
			It("accepts objects as keys", func() {
				var (
					keyA = &struct{ a int }{}
					keyB = &struct{ b int }{}
				)

				// Fetches as expected
				valueA, err := idLoader.Load(keyA)
				Expect(err).ShouldNot(HaveOccurred())

				valueB, err := idLoader.Load(keyB)
				Expect(err).ShouldNot(HaveOccurred())

				go idLoader.Dispatch(context.Background())
				Expect(future.BlockOn(future.Join(valueA, valueB))).Should(Equal([]interface{}{keyA, keyB}))

				Expect(idLoader.LoadCalls()).Should(Equal([][]dataloader.Key{
					{keyA, keyB},
				}))

				// Caching
				idLoader.Clear(keyA)

				valueA2, err := idLoader.Load(keyA)
				Expect(err).ShouldNot(HaveOccurred())

				valueB2, err := idLoader.Load(keyB)
				Expect(err).ShouldNot(HaveOccurred())

				go idLoader.Dispatch(context.Background())
				Expect(future.BlockOn(future.Join(valueA2, valueB2))).Should(Equal([]interface{}{keyA, keyB}))

				Expect(idLoader.LoadCalls()).Should(Equal([][]dataloader.Key{
					{keyA, keyB},
					{keyA},
				}))
			})
		})

		Describe("Accepts options", func() {
			// Note: mirrors 'batches multiple requests' above.
			It("may disable batching", func() {
				// Set MaxBatchSize to 1 to disable batch.
				idLoader := newIdentityLoader(dataloader.Config{
					MaxBatchSize: 1,
				})

				f1, err := idLoader.Load(1)
				Expect(err).ShouldNot(HaveOccurred())

				f2, err := idLoader.Load(2)
				Expect(err).ShouldNot(HaveOccurred())

				go idLoader.Dispatch(context.Background())
				Expect(future.BlockOn(future.Join(f1, f2))).Should(Equal([]interface{}{1, 2}))

				Expect(idLoader.LoadCalls()).Should(Equal([][]dataloader.Key{
					{1},
					{2},
				}))
			})

			// Note: mirrors 'caches repeated requests' above.
			It("may disable caching", func() {
				// Set CacheMap to NoCacheMap to disable cache.
				idLoader := newIdentityLoader(dataloader.Config{
					CacheMap: dataloader.NoCacheMap,
				})

				a, err := idLoader.Load("A")
				Expect(err).ShouldNot(HaveOccurred())

				b, err := idLoader.Load("B")
				Expect(err).ShouldNot(HaveOccurred())

				go idLoader.Dispatch(context.Background())
				Expect(future.BlockOn(future.Join(a, b))).Should(Equal([]interface{}{"A", "B"}))

				Expect(idLoader.LoadCalls()).Should(Equal([][]dataloader.Key{
					{"A", "B"},
				}))

				a2, err := idLoader.Load("A")
				Expect(err).ShouldNot(HaveOccurred())

				c, err := idLoader.Load("C")
				Expect(err).ShouldNot(HaveOccurred())

				go idLoader.Dispatch(context.Background())
				Expect(future.BlockOn(future.Join(a2, c))).Should(Equal([]interface{}{"A", "C"}))

				Expect(idLoader.LoadCalls()).Should(Equal([][]dataloader.Key{
					{"A", "B"},
					{"A", "C"},
				}))

				a3, err := idLoader.Load("A")
				Expect(err).ShouldNot(HaveOccurred())

				b2, err := idLoader.Load("B")
				Expect(err).ShouldNot(HaveOccurred())

				c2, err := idLoader.Load("C")
				Expect(err).ShouldNot(HaveOccurred())

				go idLoader.Dispatch(context.Background())
				Expect(future.BlockOn(future.Join(a3, b2, c2))).Should(Equal([]interface{}{"A", "B", "C"}))

				Expect(idLoader.LoadCalls()).Should(Equal([][]dataloader.Key{
					{"A", "B"},
					{"A", "C"},
					{"A", "B", "C"},
				}))
			})

			It("Keys are repeated in batch when cache disabled", func() {
				// Set CacheMap to NoCacheMap to disable cache.
				idLoader := newIdentityLoader(dataloader.Config{
					CacheMap: dataloader.NoCacheMap,
				})

				value1, err := idLoader.Load("A")
				Expect(err).ShouldNot(HaveOccurred())

				value2, err := idLoader.Load("C")
				Expect(err).ShouldNot(HaveOccurred())

				value3, err := idLoader.Load("D")
				Expect(err).ShouldNot(HaveOccurred())

				value4, err := idLoader.LoadMany([]dataloader.Key{"C", "D", "A", "A", "B"})
				Expect(err).ShouldNot(HaveOccurred())

				go idLoader.Dispatch(context.Background())
				Expect(future.BlockOn(value1)).Should(Equal("A"))
				Expect(future.BlockOn(value2)).Should(Equal("C"))
				Expect(future.BlockOn(value3)).Should(Equal("D"))
				Expect(future.BlockOn(value4)).Should(Equal([]interface{}{"C", "D", "A", "A", "B"}))

				Expect(idLoader.LoadCalls()).Should(Equal([][]dataloader.Key{
					{"A", "C", "D", "C", "D", "A", "A", "B"},
				}))
			})

			It("Complex cache behavior via clearAll()", func() {
				// This loader clears its cache as soon as a batch function is dispatched.
				idLoader := newCacheInvalidateLoader(dataloader.Config{})

				value1, err := idLoader.Load("A")
				Expect(err).ShouldNot(HaveOccurred())

				value2, err := idLoader.Load("B")
				Expect(err).ShouldNot(HaveOccurred())

				value3, err := idLoader.Load("A")
				Expect(err).ShouldNot(HaveOccurred())

				values1 := future.Join(value1, value2, value3)
				go idLoader.Dispatch(context.Background())
				Expect(future.BlockOn(values1)).Should(Equal([]interface{}{"A", "B", "A"}))

				value1, err = idLoader.Load("A")
				Expect(err).ShouldNot(HaveOccurred())

				value2, err = idLoader.Load("B")
				Expect(err).ShouldNot(HaveOccurred())

				value3, err = idLoader.Load("A")
				Expect(err).ShouldNot(HaveOccurred())

				values2 := future.Join(value1, value2, value3)
				go idLoader.Dispatch(context.Background())
				Expect(future.BlockOn(values2)).Should(Equal([]interface{}{"A", "B", "A"}))

				Expect(idLoader.LoadCalls()).Should(Equal([][]dataloader.Key{
					{"A", "B"},
					{"A", "B"},
				}))
			})
		})

		Describe("Accepts object key in custom cacheKey function", func() {
			It("accepts objects with a complex key", func() {
				idLoader := newIdentityLoader(dataloader.Config{
					CacheMap: &dataloader.CustomKeyCacheMap{},
				})

				var (
					key1 = &customKey{ID: 123}
					key2 = &customKey{ID: 123}
				)
				Expect(key1).ShouldNot(BeIdenticalTo(key2))

				value1, err := idLoader.Load(key1)
				Expect(err).ShouldNot(HaveOccurred())
				go idLoader.Dispatch(context.Background())
				Expect(future.BlockOn(value1)).Should(Equal(key1))

				value2, err := idLoader.Load(key2)
				Expect(err).ShouldNot(HaveOccurred())
				go idLoader.Dispatch(context.Background())
				Expect(future.BlockOn(value2)).Should(Equal(key1))

				Expect(idLoader.LoadCalls()).Should(Equal([][]dataloader.Key{
					{key1},
				}))
			})

			It("clears objects with complex key", func() {
				idLoader := newIdentityLoader(dataloader.Config{
					CacheMap: &dataloader.CustomKeyCacheMap{},
				})

				var (
					key1 = &customKey{ID: 123}
					key2 = &customKey{ID: 123}
				)
				Expect(key1).ShouldNot(BeIdenticalTo(key2))

				value1, err := idLoader.Load(key1)
				Expect(err).ShouldNot(HaveOccurred())
				go idLoader.Dispatch(context.Background())
				Expect(future.BlockOn(value1)).Should(Equal(key1))

				// Clear equivalent object key.
				idLoader.Clear(key2)

				value2, err := idLoader.Load(key1)
				Expect(err).ShouldNot(HaveOccurred())
				go idLoader.Dispatch(context.Background())
				Expect(future.BlockOn(value2)).Should(Equal(key1))

				Expect(idLoader.LoadCalls()).Should(Equal([][]dataloader.Key{
					{key1},
					{key1},
				}))
			})

			It("accepts objects with different order of keys", func() {
				idLoader := newIdentityLoader(dataloader.Config{
					CacheMap: &dataloader.CustomKeyCacheMap{},
				})

				// Fetches as expected

				var (
					keyA = &customKeyAB{A: 123, B: 321}
					keyB = &customKeyBA{B: 321, A: 123}
				)
				Expect(keyA).ShouldNot(Equal(keyB))

				valueA, err := idLoader.Load(keyA)
				Expect(err).ShouldNot(HaveOccurred())

				valueB, err := idLoader.Load(keyB)
				Expect(err).ShouldNot(HaveOccurred())

				go idLoader.Dispatch(context.Background())
				Expect(future.BlockOn(future.Join(valueA, valueB))).Should(Equal([]interface{}{
					keyA,
					keyA,
				}))

				Expect(idLoader.LoadCalls()).Should(Equal([][]dataloader.Key{
					{keyA},
				}))
			})

			It("allows priming the cache with an object key", func() {
				idLoader := newIdentityLoader(dataloader.Config{
					CacheMap: &dataloader.CustomKeyCacheMap{},
				})

				var (
					key1 = &customKey{ID: 123}
					key2 = &customKey{ID: 123}
				)
				Expect(key1).ShouldNot(BeIdenticalTo(key2))

				Expect(idLoader.Prime(key1, key1)).Should(Succeed())

				value1, err := idLoader.Load(key1)
				Expect(err).ShouldNot(HaveOccurred())
				go idLoader.Dispatch(context.Background())
				Expect(future.BlockOn(value1)).Should(Equal(key1))

				value2, err := idLoader.Load(key2)
				Expect(err).ShouldNot(HaveOccurred())
				go idLoader.Dispatch(context.Background())
				Expect(future.BlockOn(value2)).Should(Equal(key1))

				Expect(idLoader.LoadCalls()).Should(BeEmpty())
			})
		})

		Describe("Accepts custom cacheMap instance", func() {
			It("accepts a custom cache map implementation", func() {
				aCustomMap := newSimpleMap()
				idLoader := newIdentityLoader(dataloader.Config{
					CacheMap: aCustomMap,
				})

				// Fetches as expected
				valueA, err := idLoader.Load("a")
				Expect(err).ShouldNot(HaveOccurred())

				valueB1, err := idLoader.Load("b")
				Expect(err).ShouldNot(HaveOccurred())

				go idLoader.Dispatch(context.Background())
				Expect(future.BlockOn(valueA)).Should(Equal("a"))
				Expect(future.BlockOn(valueB1)).Should(Equal("b"))

				Expect(idLoader.LoadCalls()).Should(Equal([][]dataloader.Key{
					{"a", "b"},
				}))
				Expect(aCustomMap.Stash.Keys()).Should(Equal([]dataloader.Key{"a", "b"}))

				valueC, err := idLoader.Load("c")
				Expect(err).ShouldNot(HaveOccurred())

				valueB2, err := idLoader.Load("b")
				Expect(err).ShouldNot(HaveOccurred())

				go idLoader.Dispatch(context.Background())
				Expect(future.BlockOn(valueC)).Should(Equal("c"))
				Expect(future.BlockOn(valueB2)).Should(Equal("b"))

				Expect(idLoader.LoadCalls()).Should(Equal([][]dataloader.Key{
					{"a", "b"},
					{"c"},
				}))
				Expect(aCustomMap.Stash.Keys()).Should(Equal([]dataloader.Key{"a", "b", "c"}))

				// Supports clear
				idLoader.Clear("b")

				valueB3, err := idLoader.Load("b")
				Expect(err).ShouldNot(HaveOccurred())

				go idLoader.Dispatch(context.Background())
				Expect(future.BlockOn(valueB3)).Should(Equal("b"))

				Expect(idLoader.LoadCalls()).Should(Equal([][]dataloader.Key{
					{"a", "b"},
					{"c"},
					{"b"},
				}))
				Expect(aCustomMap.Stash.Keys()).Should(Equal([]dataloader.Key{"a", "c", "b"}))

				// Supports clear all
				idLoader.ClearAll()
				Expect(aCustomMap.Stash.Keys()).Should(BeEmpty())
			})
		})
	})
})

var _ = Describe("It is resilient to job queue ordering", func() {
	It("batches loads occurring within promises", func() {
		// TODO
	})

	It("can call a loader from a loader", func() {
		var (
			deepLoader = newIdentityLoader(dataloader.Config{})
			aLoader    = newChainLoader(deepLoader.DataLoader)
			bLoader    = newChainLoader(deepLoader.DataLoader)
		)

		a1, err := aLoader.Load("A1")
		Expect(err).ShouldNot(HaveOccurred())
		b1, err := bLoader.Load("B1")
		Expect(err).ShouldNot(HaveOccurred())
		a2, err := aLoader.Load("A2")
		Expect(err).ShouldNot(HaveOccurred())
		b2, err := bLoader.Load("B2")
		Expect(err).ShouldNot(HaveOccurred())

		go func() {
			aLoader.Dispatch(context.Background())
			bLoader.Dispatch(context.Background())
		}()

		Expect(future.BlockOn(a1)).Should(Equal("A1"))
		Expect(future.BlockOn(b1)).Should(Equal("B1"))
		Expect(future.BlockOn(a2)).Should(Equal("A2"))
		Expect(future.BlockOn(b2)).Should(Equal("B2"))

		Expect(aLoader.LoadCalls()).Should(Equal([][]dataloader.Key{
			{"A1", "A2"},
		}))
		Expect(bLoader.LoadCalls()).Should(Equal([][]dataloader.Key{
			{"B1", "B2"},
		}))
		Expect(deepLoader.LoadCalls()).Should(Equal([][]dataloader.Key{
			{"A1", "A2"},
			{"B1", "B2"},
		}))
	})
})
