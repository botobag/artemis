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

	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/graphql/ast"
	"github.com/botobag/artemis/graphql/validator"
	// Load standard rules required by specification for validating queries.
	_ "github.com/botobag/artemis/graphql/validator/rules"
)

// PreparedOperation is like "prepared statement" in conventional DBMS. In GraphQL, an Operation [0]
// is an executable definition [1] in GraphQL Document [2]. Operation can be either a (read-only)
// query, or a mutation or subscription. Before executing an operation, executor needs to make some
// "preparations" such as parsing and validation. PreparedOperation allows you to perform these
// static tasks in advance to save the overheads for subsequent repeatedly execution.
//
// Note PreparedOperation is bound to an Executor.
//
// [0]: https://graphql.github.io/graphql-spec/draft/#sec-Language.Operations
// [1]: https://graphql.github.io/graphql-spec/draft/#ExecutableDefinition
// [2]: https://graphql.github.io/graphql-spec/draft/#sec-Language.Document
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

// prepareOptions contains optional settings to set up a PreparedOperation.
type prepareOptions struct {
	// The name of the Operation in the Document to execute.
	OperationName string

	// Rules to be checked on the Document. This could be:
	//
	//  nil (default): use the "standard" rule set (returned by validator.StandardRules()) which are
	//  required to be checked on the Document by GraphQL Specification.
	//
	//  []interface{}{} (empty array): disable validation.
	//
	//  non-empty rule set: validate Document with the specified rules.
	ValidationRules []interface{}

	// Resolver to be used to fields without providing custom resolvers; If not provided,
	// defaultFieldResolver will be used.
	DefaultFieldResolver graphql.FieldResolver
}

// PrepareOption specifies an option to Prepare.
type PrepareOption func(*prepareOptions)

// OperationName specifies the name of the Operation to be executed in the Document.
func OperationName(name string) PrepareOption {
	return func(options *prepareOptions) {
		options.OperationName = name
	}
}

// defaultFieldResolverInstance is the default value to the prepareOptions.DefaultFieldResolver.
var defaultFieldResolverInstance = NewDefaultFieldResolver()

// DefaultFieldResolver specifies resolver to be used for fields that don't provide resolvers. Note
// that if a nil resolver is given, the one created from NewDefaultFieldResolver will be used (which
// is also the default one if no any default field resolver is given).
func DefaultFieldResolver(resolver graphql.FieldResolver) PrepareOption {
	if resolver == nil {
		resolver = defaultFieldResolverInstance
	}
	return func(options *prepareOptions) {
		options.DefaultFieldResolver = resolver
	}
}

// ValidationRules specifies the set of rules to be checked when validating the provided Document.
// The specified rules should meet the requirements for passing to validator.ValidateWithRules. Note
// that if no rule is provided, it is equivalent to WithoutValidation.
func ValidationRules(rules ...interface{}) PrepareOption {
	if len(rules) == 0 {
		return WithoutValidation()
	}

	return func(options *prepareOptions) {
		options.ValidationRules = rules
	}
}

var noValidationRules = []interface{}{}

// WithoutValidation skips validation for the provided Document.
func WithoutValidation() PrepareOption {
	return func(options *prepareOptions) {
		options.ValidationRules = noValidationRules
	}
}

// Prepare creates a PreparedOperation for executing a document.
func Prepare(schema graphql.Schema, document ast.Document, opts ...PrepareOption) (*PreparedOperation, graphql.Errors) {
	var (
		options = prepareOptions{
			DefaultFieldResolver: defaultFieldResolverInstance,
		}
		errs      graphql.Errors
		operation *ast.OperationDefinition
	)

	// Apply options.
	for _, opt := range opts {
		opt(&options)
	}

	// Validate schema and document.
	if options.ValidationRules != nil {
		errs = validator.ValidateWithRules(schema, document, options.ValidationRules...)
	} else {
		// Validate with the "standard" rules by using validator.Validate.
		errs = validator.Validate(schema, document)
	}
	if errs.HaveOccurred() {
		return nil, errs
	}

	// Find the definition for the operation to be executed from document.
	operationName := options.OperationName
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

	return &PreparedOperation{
		schema:               schema,
		document:             document,
		definition:           operation,
		rootType:             rootType,
		fragmentMap:          fragmentMap,
		defaultFieldResolver: options.DefaultFieldResolver,
	}, graphql.NoErrors()
}

// MustPrepare creates a PreparedOperation with Prepare and panics on error.
func MustPrepare(schema graphql.Schema, document ast.Document, opts ...PrepareOption) *PreparedOperation {
	operation, errs := Prepare(schema, document, opts...)
	if errs.HaveOccurred() {
		panic(errs)
	}
	return operation
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

// executeOptions contains parameter to execute a prepared operation.
type executeOptions struct {
	DataLoaderManager graphql.DataLoaderManager
	RootValue         interface{}
	AppContext        interface{}
	VariableValues    map[string]interface{}
}

// ExecuteOption configures execution of a PreparedOperation.
type ExecuteOption func(*executeOptions)

// DataLoaderManager that manages dispatches for data loaders being used during execution; User can
// also tracks DataLoader instances being used during the execution.
func DataLoaderManager(manager graphql.DataLoaderManager) ExecuteOption {
	return func(options *executeOptions) {
		options.DataLoaderManager = manager
	}
}

// RootValue is an initial value corresponding to the root type being executed. Conceptually, an
// initial value represents the “universe” of data available via a GraphQL Service. It is common for
// a GraphQL Service to always use the same initial value for every request.
func RootValue(value interface{}) ExecuteOption {
	return func(options *executeOptions) {
		options.RootValue = value
	}
}

// AppContext is an application-specific data that will get passed to all resolve functions.
func AppContext(ctx interface{}) ExecuteOption {
	return func(options *executeOptions) {
		options.AppContext = ctx
	}
}

// VariableValues contains values for any Variables defined by the Operation.
func VariableValues(variables map[string]interface{}) ExecuteOption {
	return func(options *executeOptions) {
		options.VariableValues = variables
	}
}

// Execute executes the given operation.  ctx specifies deadline and/or cancellation for
// executor, etc..
func (operation *PreparedOperation) Execute(c context.Context, opts ...ExecuteOption) <-chan ExecutionResult {
	var options executeOptions

	// Get options.
	for _, opt := range opts {
		opt(&options)
	}

	// Initialize an ExecutionContext for executing operation.
	ctx, errs := newExecutionContext(c, operation, &options)
	if errs.HaveOccurred() {
		// Create a channel to return the error.
		result := make(chan ExecutionResult, 1)
		result <- ExecutionResult{
			Errors: errs,
		}
		return result
	}

	// Create executor.
	e := newBlockingExecutor()

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
