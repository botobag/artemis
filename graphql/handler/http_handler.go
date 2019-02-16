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
	"net/http"

	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/graphql/executor"
	"github.com/botobag/artemis/graphql/parser"
	"github.com/botobag/artemis/graphql/token"
)

// httpHandler implements a http.Handler which is based on LLHandler to serve GraphQL queries from
// HTTP requests.
type httpHandler struct {
	*LLHandler

	config httpHandlerConfig

	// The handler for presenting errors occurred during preparation of execution; It doesn't handle
	// errors occurred during execution (in which ResultPresenter is responsible for.)
	errorPresenter ErrorPresenter

	// The handles for building requests and writing responses; If not given, DefaultRequestBuilder
	// and DefaultResultPresenter are used, respectively.
	requestBuilder  RequestBuilder
	resultPresenter ResultPresenter
}

// httpHandlerConfig contains configuration for a httpHandler.
type httpHandlerConfig struct {
	LLConfig

	// Configuration given to DefaultRequestBuilder; It is not applicable if custom
	// RequestBuilder is used.
	defaultRequestBuilderConfig DefaultRequestBuilderConfig

	errorPresenter  ErrorPresenter
	requestBuilder  RequestBuilder
	resultPresenter ResultPresenter
}

// Option configures httpHandler
type Option func(h *httpHandlerConfig)

// MaxBodySize sets the maximum number of bytes to be read from request body for ParseHTTPRequest
// called by DefaultRequestBuilder.
func MaxBodySize(size uint) Option {
	return func(h *httpHandlerConfig) {
		h.defaultRequestBuilderConfig.HTTPRequestParserOptions.MaxBodySize = size
	}
}

// QueryParserOptions provides settings to the parser for GraphQL query.
func QueryParserOptions(options parser.ParseOptions) Option {
	return func(h *httpHandlerConfig) {
		h.defaultRequestBuilderConfig.QueryParserOptions = options
	}
}

// DefaultFieldResolver sets the resolver to be used when a field doesn't provide one.
func DefaultFieldResolver(resolver graphql.FieldResolver) Option {
	return func(h *httpHandlerConfig) {
		h.defaultRequestBuilderConfig.DefaultFieldResolver = resolver
	}
}

// OverrideErrorPresenter overrides default RequestBuilder.
func OverrideErrorPresenter(errorPresenter ErrorPresenter) Option {
	return func(h *httpHandlerConfig) {
		h.errorPresenter = errorPresenter
	}
}

// OverrideRequestBuilder overrides default RequestBuilder.
func OverrideRequestBuilder(requestBuilder RequestBuilder) Option {
	return func(h *httpHandlerConfig) {
		h.requestBuilder = requestBuilder
	}
}

// OverrideResultPresenter overrides default ResultPresenter.
func OverrideResultPresenter(resultPresenter ResultPresenter) Option {
	return func(h *httpHandlerConfig) {
		h.resultPresenter = resultPresenter
	}
}

// OverrideOperationCache that overrides default OperationCache.
func OverrideOperationCache(cache OperationCache) Option {
	return func(h *httpHandlerConfig) {
		h.OperationCache = cache
	}
}

// New creates a net/http.Handler and builds a GraphQL web service to serve queries against the
// schema.
func New(schema *graphql.Schema, opts ...Option) (http.Handler, error) {
	// Apply Options on config.
	config := httpHandlerConfig{
		LLConfig: LLConfig{
			Schema: schema,
		},

		defaultRequestBuilderConfig: DefaultRequestBuilderConfig{
			HTTPRequestParserOptions: ParseHTTPRequestOptions{
				MaxBodySize: 10 << 20, // 10MB
			},
		},
	}
	for _, opt := range opts {
		opt(&config)
	}

	baseHandler, err := NewLLHandler(&config.LLConfig)
	if err != nil {
		return nil, err
	}

	requestBuilder := config.requestBuilder
	if requestBuilder == nil {
		requestBuilder = DefaultRequestBuilder{
			Config: &config.defaultRequestBuilderConfig,
		}
	}

	resultPresenter := config.resultPresenter
	if resultPresenter == nil {
		resultPresenter = DefaultResultPresenter{}
	}

	errorPresenter := config.errorPresenter
	if errorPresenter == nil {
		errorPresenter = DefaultErrorPresenter{
			ResultPresenter: resultPresenter,
		}
	}

	return &httpHandler{
		LLHandler:       baseHandler,
		config:          config,
		errorPresenter:  errorPresenter,
		requestBuilder:  requestBuilder,
		resultPresenter: resultPresenter,
	}, nil
}

// ErrorPresenter implements HTTPHandler which returns h.errorPresenter.
func (h *httpHandler) ErrorPresenter() ErrorPresenter {
	return h.errorPresenter
}

func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Prepare executable operation from r with RequestBuilder.
	req, err := h.requestBuilder.Build(r, h)
	if err != nil {
		// Present error.
		h.errorPresenter.Write(w, err)
		return
	}

	// Serve the request with LLHandler which executes query.
	result := h.Serve(req)

	// Present the result to w.
	h.resultPresenter.Write(w, r, req, result)
}

// RequestBuilder generates a Request to be served by LLHandler from an HTTP request.
type RequestBuilder interface {
	// Build turns a http.Request r into a Request for h. w is used for sending errors.
	Build(r *http.Request, h HTTPHandler) (*Request, error)
}

// DefaultRequestBuilderConfig specifies settings to configure DefaultRequestBuilder.
type DefaultRequestBuilderConfig struct {
	HTTPRequestParserOptions ParseHTTPRequestOptions
	QueryParserOptions       parser.ParseOptions
	DefaultFieldResolver     graphql.FieldResolver
}

// DefaultRequestBuilder implements the default request builder used by HTTP handler to obtain
// a Request object from a http.Request.
type DefaultRequestBuilder struct {
	Config *DefaultRequestBuilderConfig
}

// HTTPHandler provides interfaces to access settings in httpHandler from RequestBuilder.
type HTTPHandler interface {
	// Schema served by this handler
	Schema() *graphql.Schema

	// OperationCache for the parsed queries
	OperationCache() OperationCache
}

// Build implements RequestBuilder.
func (builder DefaultRequestBuilder) Build(r *http.Request, h HTTPHandler) (*Request, error) {
	// Parse query from request parameters.
	parsedReq, err := ParseHTTPRequest(r, &builder.Config.HTTPRequestParserOptions)
	if err != nil {
		return nil, err
	}

	// Empty query is an error.
	if len(parsedReq.Query) == 0 {
		return nil, ErrEmptyQuery{
			Request: r,
		}
	}

	// Try to find the operation that has been prepared for given query before from cache.
	cache := h.OperationCache()
	operation, ok := cache.Get(parsedReq.Query)
	if !ok {
		// Parse query.
		document, err := parser.Parse(token.NewSource(&token.SourceConfig{
			Body: token.SourceBody([]byte(parsedReq.Query)),
		}), builder.Config.QueryParserOptions)

		if err != nil {
			return nil, &ErrParseQuery{
				Request:       r,
				ParsedRequest: parsedReq,
				Err:           err,
			}
		}

		// Prepare operation for executing the query.
		var errs graphql.Errors
		operation, errs = executor.Prepare(executor.PrepareParams{
			Schema:               h.Schema(),
			Document:             document,
			OperationName:        parsedReq.OperationName,
			DefaultFieldResolver: builder.Config.DefaultFieldResolver,
		})
		if errs.HaveOccurred() {
			return nil, &ErrPrepare{
				Request:       r,
				ParsedRequest: parsedReq,
				Document:      document,
				Errs:          errs,
			}
		}

		// Update cache.
		cache.Add(parsedReq.Query, operation)
	}

	return &Request{
		Ctx:       r.Context(),
		Operation: operation,
		Params: &executor.ExecuteParams{
			VariableValues: parsedReq.Variables,
		},
	}, nil
}

// ResultPresenter presents an execution result to a http.ResponseWriter.
type ResultPresenter interface {
	// Write writes an ExecutionResult to w.
	Write(
		w http.ResponseWriter,
		httpRequest *http.Request,
		graphqlRequest *Request,
		result <-chan executor.ExecutionResult)
}

// DefaultResultPresenter implements a ResultPresenter used by HTTP handler to present an
// ExecutionResult.
type DefaultResultPresenter struct{}

func (DefaultResultPresenter) Write(
	w http.ResponseWriter,
	httpRequest *http.Request,
	graphqlRequest *Request,
	result <-chan executor.ExecutionResult) {

	// Serialize result to JSON encoding.
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)

	// Receive result.
	r := <-result
	r.MarshalJSONTo(w)
}
