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
	"fmt"
)

// BatchLoadJob performs a batch load to fetch data required by a list of tasks.
type BatchLoadJob struct {
	ctx context.Context

	// Tasks processed by this job stored in a linked list
	tasks TaskList
}

// Run implements concurrent.Task, allowing a BatchLoadJob to be executed by a concurrent.Executor.
func (job *BatchLoadJob) Run() (interface{}, error) {
	tasks := &job.tasks
	config := tasks.first.parent.loader.config

	// Call BatchLoader to load data.
	config.BatchLoader.Load(job.ctx, tasks)

	// Make sure that all tasks were completed. If not, complete it with an error.
	taskIter := tasks.Iterator()
	for {
		task, done := taskIter.Next()
		if done {
			break
		}

		result := task.loadResult()
		if result.Kind == taskNotCompleted {
			task.SetError(fmt.Errorf("%T must complete every given data loading task with either a "+
				"value or an error but it doesn't complete task that loads data at key %v",
				config.BatchLoader, task.Key()))
		}
	}

	return nil, nil
}
