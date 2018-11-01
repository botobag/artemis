/**
 * Copyright (c) 2018, The Artemis Authors.
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
	"context"

	"github.com/botobag/artemis/graphql"
)

// SerialExecutor implements a ExecutionRunner which executes selection sets "serially". That is, at
// any given time, only one selection set is executed and every time a selection set is completed,
// its children selection set is picked to get executed first (than the sibling ones). I.e., the
// graph of execution tree is traversed in DFS order.
type SerialExecutor struct {
	impl Common
}

type serialExecutionQueue []*ResultNode

// Push implements ExecutionQueue.
func (queue *serialExecutionQueue) Push(node *ResultNode) {
	*queue = append(*queue, node)
}

// Execute implements Executor.
func (executor SerialExecutor) Execute(ctx context.Context, executionCtx *ExecutionContext) ExecutionResult {
	impl := executor.impl

	// Build root node.
	rootNode, err := impl.BuildRootResultNode(executionCtx)
	if err != nil {
		return ExecutionResult{
			Errors: graphql.ErrorsOf(err.(*graphql.Error)),
		}
	}

	// Allocate top-level result data.
	result := ExecutionResult{
		Data: rootNode,
	}

	// Queue tracks result nodes that need to be fulfilled. Initialize with unresolved child nodes in
	// rootNode.
	var queue serialExecutionQueue
	impl.EnqueueChildNodes(&queue, rootNode)

	for len(queue) > 0 {
		// Pop one node from back.
		var resultNode *ResultNode
		resultNode, queue = queue[len(queue)-1], queue[:len(queue)-1]

		// Execute the node with source value.
		errs := impl.ExecuteNode(ctx, executionCtx, resultNode)
		if errs.HaveOccurred() {
			result.Errors.AppendErrors(errs)
			continue
		}

		// Enqueue any unresolved child nodes to queue.
		impl.EnqueueChildNodes(&queue, resultNode)
	}

	return result
}
