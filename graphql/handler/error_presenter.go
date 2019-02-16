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
	"github.com/botobag/artemis/graphql/ast"
	"github.com/botobag/artemis/graphql/executor"
	"github.com/botobag/artemis/internal/util"
)

// ErrorPresenter presents an error to a http.ResponseWriter.
type ErrorPresenter interface {
	// Write sends the given error to w.
	Write(w http.ResponseWriter, err error)
}

// Errors by DefaultRequestBuilder.Build

// ErrEmptyQuery describes an error when an empty query is not allowed.
type ErrEmptyQuery struct {
	Request *http.Request
}

// Error implements Go's error interface.
func (err ErrEmptyQuery) Error() string {
	return "empty query"
}

// ErrParseQuery describes an invalid GraphQL query document that failed parsing.
type ErrParseQuery struct {
	Request       *http.Request
	ParsedRequest *HTTPRequest
	Err           error
}

// Error implements Go's error interface.
func (err *ErrParseQuery) Error() string {
	return "invalid query: " + err.Err.Error()
}

// ErrPrepare indicates a failure in prepare a PreparedOperation for execution for a query.
type ErrPrepare struct {
	Request       *http.Request
	ParsedRequest *HTTPRequest
	Document      ast.Document
	Errs          graphql.Errors
}

// Error implements Go's error interface.
func (err *ErrPrepare) Error() string {
	var buf util.StringBuilder
	buf.WriteString("cannot prepare executable operation for query because of following error(s): \n")
	for _, e := range err.Errs.Errors {
		buf.WriteRune('\t')
		buf.WriteString(e.Error())
	}
	return buf.String()
}

// DefaultErrorPresenter implements an ErrorPresenter which is default used by HTTP handler when no
// error presenter is provided.
type DefaultErrorPresenter struct {
	// ResultPresenter is used to present ErrPrepare.Errs in an ExecutionResult.
	ResultPresenter ResultPresenter
}

// Write implements ErrorPresenter.
func (presenter DefaultErrorPresenter) Write(w http.ResponseWriter, err error) {
	switch err := err.(type) {
	case ErrEmptyQuery, *ErrParseQuery:
		http.Error(w, err.Error(), http.StatusBadRequest)

	case *ErrPrepare:
		result := make(chan executor.ExecutionResult, 1)
		result <- executor.ExecutionResult{
			Errors: err.Errs,
		}
		presenter.ResultPresenter.Write(w, err.Request, nil, result)
	}
}
