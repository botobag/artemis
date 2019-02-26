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
	"fmt"

	"github.com/botobag/artemis/concurrent"
	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/graphql/ast"
)

// PreparedOperation is like "prepared statement" in conventional DBMS. In GraphQL, an Operation [0]
// is an executable definition [1] in GraphQL Document [2]. Operation can be either a (read-only)
// query, or a mutation or subscription. Before executing an operation, executor needs to make some
// "preparations" such as parsing and validation. PreparedOperation allows you to perform these
// static tasks in advance to save the overheads for subsequent repeatedly execution.
//
// Note PreparedOperation is bound to an Executor.
//
// [0]: https://facebook.github.io/graphql/draft/#sec-Language.Operations
// [1]: https://facebook.github.io/graphql/draft/#ExecutableDefinition
// [2]: https://facebook.github.io/graphql/draft/#sec-Language.Document
type PreparedOperation struct {
	// Schema of the type system that is currently executing
	schema graphql.Schema

	// Document that contains definitions for this operation
	document ast.Document

	// Definition of this operation
	definition *ast.OperationDefinition

	// rootType extracts the root type corresponding to the operation in the schema.
	rootType graphql.Object

	// FragmentMap maps name to the fragment definition in the document to speed up lookup when
	// fragment spread during execution.
	fragmentMap map[string]*ast.FragmentDefinition

	// Resolver to be used for resolving field value when the field doesn't provide one.
	defaultFieldResolver graphql.FieldResolver
}

// PrepareParams specifies parameters to Prepare. All data are required except DefaultFieldResolver.
type PrepareParams struct {
	// Schema of the type system that this operation is executing on
	Schema graphql.Schema

	// Document that contains operations to be prepared for execution
	Document ast.Document

	// The name of the Operation in the Document to execute.
	OperationName string

	// Resolver to be used to fields without providing custom resolvers.
	DefaultFieldResolver graphql.FieldResolver
}

// Prepare prepares an operation for execution. It creates a PreparedOperation.
func Prepare(params PrepareParams) (*PreparedOperation, graphql.Errors) {
	var errs graphql.Errors

	schema := params.Schema
	document := params.Document

	// TODO: Validate schema and document.

	// Find the definition for the operation to be executed from document.
	var operation *ast.OperationDefinition

	operationName := params.OperationName
	// Also build map for fragmentMap.
	fragmentMap := map[string]*ast.FragmentDefinition{}

	for _, definition := range document.Definitions {
		switch definition := definition.(type) {
		case *ast.OperationDefinition:
			if len(operationName) == 0 {
				if operation != nil {
					return nil, graphql.ErrorsOf("Must provide operation name if query contains multiple operations.")
				}
				operation = definition
			} else {
				if operationName == definition.Name.Value() {
					operation = definition
				}
			}

		case *ast.FragmentDefinition:
			fragmentMap[definition.Name.Value()] = definition
		}
	}

	if operation == nil {
		if len(operationName) > 0 {
			errs.Emplace(fmt.Sprintf(`Unknown operation named "%s".`, operationName))
			return nil, errs
		}
		errs.Emplace("Must provide an operation.")
		return nil, errs
	}

	// Extract the root operation type.
	var rootType graphql.Object
	switch operation.OperationType() {
	case ast.OperationTypeQuery:
		rootType = schema.Query()
		if rootType == nil {
			return nil, graphql.ErrorsOf(
				"Schema does not define the required query root type.",
				[]graphql.ErrorLocation{graphql.ErrorLocationOfASTNode(operation)})
		}

	case ast.OperationTypeMutation:
		rootType = schema.Mutation()
		if rootType == nil {
			return nil, graphql.ErrorsOf(
				"Schema is not configured for mutations.",
				[]graphql.ErrorLocation{graphql.ErrorLocationOfASTNode(operation)})
		}

	case ast.OperationTypeSubscription:
		rootType = schema.Subscription()
		if rootType == nil {
			return nil, graphql.ErrorsOf(
				"Schema is not configured for subscriptions.",
				[]graphql.ErrorLocation{graphql.ErrorLocationOfASTNode(operation)})
		}

	default:
		return nil, graphql.ErrorsOf(
			"Can only have query, mutation and subscription operations.",
			[]graphql.ErrorLocation{graphql.ErrorLocationOfASTNode(operation)})
	}

	defaultFieldResolver := params.DefaultFieldResolver
	if defaultFieldResolver == nil {
		defaultFieldResolver = &DefaultFieldResolver{
			UnresolvedAsError:   true,
			ScanAnonymousFields: true,
			ScanMethods:         true,
			FieldTagName:        "graphql",
		}
	}

	return &PreparedOperation{
		schema:               schema,
		document:             document,
		definition:           operation,
		rootType:             rootType,
		fragmentMap:          fragmentMap,
		defaultFieldResolver: defaultFieldResolver,
	}, graphql.NoErrors()
}

// Schema returns the type system definition which the operation is based on.
func (operation *PreparedOperation) Schema() graphql.Schema {
	return operation.schema
}

// Document returns the request document.
func (operation *PreparedOperation) Document() ast.Document {
	return operation.document
}

// VariableDefinitions returns the variable definitions describing the variables taken by the
// operation.
func (operation *PreparedOperation) VariableDefinitions() []*ast.VariableDefinition {
	return operation.definition.VariableDefinitions
}

// ExecuteParams specifies parameter to execute a prepared operation.
type ExecuteParams struct {
	// Runner specifies executor to run the execution. If it is not provided, Execute blocks the
	// calling goroutine to complete the execution.
	Runner concurrent.Executor

	// DataLoaderManager that manages dispatch for data loaders being used during execution; User can
	// also tracks DataLoader instances being used during the execution.
	DataLoaderManager graphql.DataLoaderManager

	// RootValue is an initial value corresponding to the root type being executed. Conceptually, an
	// initial value represents the “universe” of data available via a GraphQL Service. It is common
	// for a GraphQL Service to always use the same initial value for every request.
	RootValue interface{}

	// AppContext is an application-specific data that will get passed to all resolve functions.
	AppContext interface{}

	// VariableValues contains values for any Variables defined by the Operation.
	VariableValues map[string]interface{}
}

// Execute executes the given operation.  ctx specifies deadline and/or cancellation for
// executor, etc..
func (operation *PreparedOperation) Execute(c context.Context, params ExecuteParams) <-chan ExecutionResult {
	// Initialize an ExecutionContext for executing operation.
	ctx, errs := newExecutionContext(c, operation, &params)
	if errs.HaveOccurred() {
		// Create a channel to return the error.
		result := make(chan ExecutionResult, 1)
		result <- ExecutionResult{
			Errors: errs,
		}
		return result
	}

	// Create executor.
	var e executor
	if params.Runner == nil {
		e = newBlockingExecutor()
	} else if operation.Type() == ast.OperationTypeMutation {
		e = newSerialExecutor(params.Runner)
	} else {
		e = newParallelExecutor(params.Runner)
	}

	// Run the execution.
	return e.Run(ctx)
}

// RootType returns operation.rootType.
func (operation *PreparedOperation) RootType() graphql.Object {
	return operation.rootType
}

// Definition returns operation.definition.
func (operation *PreparedOperation) Definition() *ast.OperationDefinition {
	return operation.definition
}

// Type returns operation.definition.OperationType().
func (operation *PreparedOperation) Type() ast.OperationType {
	return operation.definition.OperationType()
}

// FragmentDef finds the fragment definition for given name.
func (operation *PreparedOperation) FragmentDef(name string) *ast.FragmentDefinition {
	return operation.fragmentMap[name]
}

// DefaultFieldResolver returns operation.defaultFieldResolver.
func (operation *PreparedOperation) DefaultFieldResolver() graphql.FieldResolver {
	return operation.defaultFieldResolver
}
