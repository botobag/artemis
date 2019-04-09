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

package handler

import (
	"context"
	"errors"
	"fmt"

	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/graphql/executor"
)

// LLHandler creates a handler that is suit for serving GraphQL queries against a schema in a
// long-running process. It is useful as a low-level building block for building GraphQL services
// such as GraphQL web services.
type LLHandler struct {
	// Schema served by this handler
	schema graphql.Schema

	// Cache for the parsed query; Could be nil (when the handler was created
	// with config.OperationCache set to NopOperationCache) to disable cache.
	cache OperationCache

	// Middlewares to be applied before executing a Request
	middlewares []RequestMiddleware
}

// LLConfig contains configuration to set up a LLHandler.
type LLConfig struct {
	// Schema to be working on
	Schema graphql.Schema

	// OperationCache caches graphql.PreparedOperation created from a query to save parsing efforts.
	OperationCache OperationCache

	// Middlewares to be applied before executing a Request
	Middlewares []RequestMiddleware
}

var errMissingSchema = errors.New("artemis/handler: must specify a schema")

// NewLLHandler creates a LLHandler from given configuration.
func NewLLHandler(config *LLConfig) (*LLHandler, error) {
	// schema is required.
	schema := config.Schema
	if schema == nil {
		return nil, errMissingSchema
	}

	cache := config.OperationCache
	if cache == nil {
		// Create a LRU cache with 512 entries in maximum by default.
		var err error
		cache, err = NewLRUOperationCache(512)
		if err != nil {
			return nil, err
		}
	} else if _, isNop := cache.(NopOperationCache); isNop {
		cache = nil
	}

	return &LLHandler{
		schema: schema,
		cache:  cache,
	}, nil
}

// Schema returns handler.schema.
func (handler *LLHandler) Schema() graphql.Schema {
	return handler.schema
}

// OperationCache returns handler.cache.
func (handler *LLHandler) OperationCache() OperationCache {
	return handler.cache
}

// Request contains parameter required by Serve.
type Request struct {
	Ctx         context.Context
	Operation   *executor.PreparedOperation
	ExecuteOpts []executor.ExecuteOption
}

// RequestMiddleware applies changes on Request before its operation gets executed. It can be used
// to modify ExecuteParams in Request such as setting root values and/or supplied app-specific
// context.
type RequestMiddleware interface {
	// Apply modifies request. next specifies the next action to do after applying the middleware.
	Apply(request *Request, next *RequestMiddlewareNext)
}

// RequestMiddlewareNext is provided to a RequestMiddleware to specify the next action to do.
type RequestMiddlewareNext struct {
	middlewares []RequestMiddleware

	// The index of middleware to be applied when Next is called.
	nextIndex int

	// The result after applying middlewares
	result interface{} /* Should be either *Request or *executor.ExecutionResult */
}

// Next continues applying the next middleware in the chain.
func (next *RequestMiddlewareNext) Next(request *Request) {
	switch next.result.(type) {
	case *Request:
		panic("calling Next multiple times is not allowed")
	case *executor.ExecutionResult:
		panic("cannot call Next after one of NextError or NextResult is called")
	case nil:
		/* Apply next middleware or return */
	default:
		panic(fmt.Errorf("unexpected result type: %T", next.result))
	}

	middlewares := next.middlewares
	if next.nextIndex >= len(middlewares) {
		// All middlewares has been applied.
		next.result = request
		return
	}

	// Take the next middleware to be applied.
	nextMiddleware := middlewares[next.nextIndex]
	// Increment index.
	next.nextIndex++
	// Apply the middleware.
	nextMiddleware.Apply(request, next)

	if next.result == nil {
		panic(fmt.Errorf(`"%T" must end with one of Next, NextError or NextResult on return`,
			nextMiddleware))
	}
}

// NextError stops applying rest middlewares in the chain and sends an ExecutionResult that includes
// given error.
func (next *RequestMiddlewareNext) NextError(err *graphql.Error) {
	next.NextResult(&executor.ExecutionResult{
		Errors: graphql.ErrorsOf(err),
	})
}

// NextResult stops applying rest middlewares in the chain and sends the result.
func (next *RequestMiddlewareNext) NextResult(result *executor.ExecutionResult) {
	switch next.result.(type) {
	case *Request:
		panic("calling NextError or NextResult is not allowed on returning from Next")

	case *executor.ExecutionResult:
		panic("calling NextError or NextResult multiple times is not allowed")

	case nil:
		next.result = result

	default:
		panic(fmt.Errorf("unexpected result type: %T", next.result))
	}
}

// onReturn returns true when the middleware chain has been applied and we're currently on the
// return path to the root callsite of RequestMiddlewareNext.Next (in most cases, it is returning to
// LLHandler.Serve.)
func (next *RequestMiddlewareNext) onReturn() bool {
	return next.result != nil
}

// Serve executes the operation with given context and parameters. The given request object must not
// be nil.
func (handler *LLHandler) Serve(request *Request) *executor.ExecutionResult {
	middlewares := handler.middlewares
	if len(middlewares) > 0 {
		next := RequestMiddlewareNext{
			middlewares: middlewares,
		}
		// Call Next to apply the middleware chain.
		next.Next(request)

		switch result := next.result.(type) {
		case *Request:
			request = result

		case *executor.ExecutionResult:
			return result

		default:
			panic(fmt.Errorf("unexpected result type: %T", next.result))
		}
	}

	// Apply RequestMiddleware before execution.
	return request.Operation.Execute(request.Ctx, request.ExecuteOpts...)
}
