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
	"log"

	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/graphql/ast"
)

// ExecutionResult contains result from running an Executor.
type ExecutionResult struct {
	Data   *ResultNode
	Errors graphql.Errors
}

// An Executor executes a prepared operation.
type Executor interface {
	// Execute runs an execution provided a context.
	Execute(ctx context.Context, executionCtx *ExecutionContext) ExecutionResult
}

var serialExecutor = SerialExecutor{}

// SelectDefaultExecutor selects
func SelectDefaultExecutor(operationType ast.OperationType) Executor {
	switch operationType {
	case ast.OperationTypeQuery, ast.OperationTypeMutation, ast.OperationTypeSubscription:
		return serialExecutor
	default:
		log.Printf("SelectDefaultExecutor: unsupported operation type%s; Use SerialExecutor for execution.", operationType)
		return serialExecutor
	}
}
