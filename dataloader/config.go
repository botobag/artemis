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

	"github.com/botobag/artemis/concurrent"
)

// BatchLoader loads data identified by keys in task. The result is also written to each task
// through iterator.
type BatchLoader interface {
	Load(ctx context.Context, tasks *TaskList)
}

// The BatchLoadFunc type is an adapter to allow the use of ordinary functions as BatchLoader. If f
// is a function with the appropriate signature, BatchLoadFunc(f) is a BatchLoader that calls f.
type BatchLoadFunc func(ctx context.Context, tasks *TaskList)

// Load implements BatchLoader by simply calling f(keys, taskIter).
func (f BatchLoadFunc) Load(ctx context.Context, tasks *TaskList) {
	f(ctx, tasks)
}

// Config specifies:
//
//  1. The way to fetch data;
//  2. Various configurations for batching;
//  3. Various configurations for cacheing.
type Config struct {
	// (Required) BatchLoader specifies the way to load data in batch from given keys for a
	// DataLoader.
	BatchLoader BatchLoader

	// (Optional) Runner for running the jobs dispatched by the loader to load data.
	Runner concurrent.Executor

	// (Optional) Set the batch size. Default is 0 which means unlimited. Setting it to 1 causes
	// DataLoader to send only one task to its BatchLoader which disables batch load.
	MaxBatchSize uint

	// (Optional) CacheMap specifies cache instance to cache requested and loaded data. 3 possible
	// values can be provided:
	//
	//  1. nil (when CacheMap is not set): cache is enabled and a DefaultCacheMap instance will be
	//     used.
	//  2. NoCacheMap: cache is disabled.
	//  3. Others: Custom cache instance that implements CacheMap interfaces.
	CacheMap CacheMap
}
