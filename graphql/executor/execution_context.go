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
	"github.com/botobag/artemis/graphql/internal/value"
)

// An ExecutionContext contains data which are required for an Executor to fulfill a request for
// exeuction. The context includes the operation to execute, variables supplied and request-specific
// values, etc..
type ExecutionContext struct {
	// Context for the execution
	ctx context.Context

	// operation being executed.
	operation *PreparedOperation

	// rootValue is the "source" data for the top level field ("root fields").
	rootValue interface{}

	// appContext contains application-specific data which will get passed to all resolve functions.
	appContext interface{}

	// variableValues contains values to the parameters in current query. The values has passed input
	// coercion.
	variableValues graphql.VariableValues
}

// newExecutionContext initializes an ExecutionContext given the operation to execute and the
// request data.
func newExecutionContext(ctx context.Context, operation *PreparedOperation, params *ExecuteParams) (*ExecutionContext, graphql.Errors) {
	// Run input coercion on variable values.
	variableValues, errs := value.CoerceVariableValues(
		operation.Schema(),
		operation.VariableDefinitions(),
		params.VariableValues)
	if errs.HaveOccurred() {
		return nil, errs
	}

	return &ExecutionContext{
		ctx:            ctx,
		operation:      operation,
		rootValue:      params.RootValue,
		appContext:     params.AppContext,
		variableValues: variableValues,
	}, graphql.NoErrors()
}

// Operation returns context.operation.
func (context *ExecutionContext) Operation() *PreparedOperation {
	return context.operation
}

// RootValue returns context.rootValue.
func (context *ExecutionContext) RootValue() interface{} {
	return context.rootValue
}

// AppContext returns context.appContext.
func (context *ExecutionContext) AppContext() interface{} {
	return context.appContext
}

// VariableValues returns context.variableValues.
func (context *ExecutionContext) VariableValues() graphql.VariableValues {
	return context.variableValues
}
