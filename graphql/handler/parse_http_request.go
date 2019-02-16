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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"net/url"
	"strings"
)

// If the value doesn't contains value for the given key, return an empty string without error.
// If there're multiple values associated with the key, return an error. Otherwise, return the
// single value.
func getOneValue(values url.Values, key string) (string, error) {
	v := values[key]
	switch len(v) {
	case 0:
		return "", nil
	case 1:
		return v[0], nil
	default:
		return "", fmt.Errorf(`multiple values are provided to "%s", but only one expected`, key)
	}
}

// Parse a HTTPRequest from url.Values. r is used for including in error value to provide verbose
// context for debugging.
func parseRequestFromValues(
	r *http.Request,
	options *ParseHTTPRequestOptions,
	values url.Values) (*HTTPRequest, error) {

	var (
		req HTTPRequest
		err error
	)

	if req.Query, err = getOneValue(values, "query"); err != nil {
		return nil, &HTTPRequestParseError{
			Request: r,
			Options: options,
			Err:     err,
		}
	}
	if req.OperationName, err = getOneValue(values, "operationName"); err != nil {
		return nil, &HTTPRequestParseError{
			Request: r,
			Options: options,
			Err:     err,
		}
	}

	variables, err := getOneValue(values, "variables")
	if err != nil {
		return nil, &HTTPRequestParseError{
			Request: r,
			Options: options,
			Err:     err,
		}
	}

	if len(variables) > 0 {
		if err := json.NewDecoder(strings.NewReader(variables)).Decode(&req.Variables); err != nil {
			return nil, &HTTPRequestParseError{
				Request: r,
				Options: options,
				Err:     err,
			}
		}
	}

	return &req, nil
}

// ParseHTTPRequestOptions provides settings to ParseHTTPRequest.
type ParseHTTPRequestOptions struct {
	// Maximum size in bytes to be read when parsing a GraphQL query from HTTP request body. If it is
	// not set, the size is capped at 10MB.
	MaxBodySize uint
}

// HTTPRequest contains result values of ParseHTTPRequest.
type HTTPRequest struct {
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName"`
	Variables     map[string]interface{} `json:"variables"`
}

// HTTPRequestParseError is returned by ParseHTTPRequest when parsing failed.
type HTTPRequestParseError struct {
	Request *http.Request
	Options *ParseHTTPRequestOptions
	Err     error
}

// Error implements Go's error interface.
func (err *HTTPRequestParseError) Error() string {
	return err.Err.Error()
}

var errRequestBodyTooLarge = errors.New("request body is too large")

// ParseHTTPRequest parses a GraphQL request from a http.Request object.
func ParseHTTPRequest(r *http.Request, options *ParseHTTPRequestOptions) (*HTTPRequest, error) {
	switch r.Method {
	case http.MethodGet:
		var (
			// Read query variables in URL from r.Form.
			values = r.Form
			err    error
		)
		if values == nil {
			// If not present, parse them from URL.
			values, err = url.ParseQuery(r.URL.RawQuery)
			if err != nil {
				return nil, &HTTPRequestParseError{
					Request: r,
					Options: options,
					Err:     err,
				}
			}
		}
		return parseRequestFromValues(r, options, values)

	case http.MethodPost:
		// Determine the content-type.
		var contentType = r.Header.Get("Content-Type")
		contentType, _, _ = mime.ParseMediaType(contentType)
		// Ignore error.

		// Quick path: if content-type is application/x-www-form-urlencoded and r.Form has already been
		// populated, use it.
		if contentType == "application/x-www-form-urlencoded" && r.Form != nil {
			return parseRequestFromValues(r, options, r.Form)
		}

		// Read body.
		maxBodySize := options.MaxBodySize
		body, err := ioutil.ReadAll(io.LimitReader(r.Body, int64(maxBodySize+1)))
		if err != nil {
			return nil, &HTTPRequestParseError{
				Request: r,
				Options: options,
				Err:     err,
			}
		}

		// Check the overflow.
		if len(body) > int(maxBodySize) {
			return nil, &HTTPRequestParseError{
				Request: r,
				Options: options,
				Err:     errRequestBodyTooLarge,
			}
		}

		// See https://github.com/graphql/express-graphql/blob/8826952/src/parseBody.js for the
		// supported content-type.
		switch contentType {
		case "application/graphql":
			// The entire body is the query.
			return &HTTPRequest{
				Query: string(body),
			}, nil

		case "application/x-www-form-urlencoded":
			values, err := url.ParseQuery(string(body))
			if err != nil {
				return nil, &HTTPRequestParseError{
					Request: r,
					Options: options,
					Err:     err,
				}
			}

			return parseRequestFromValues(r, options, values)

		case "", "application/json":
			var req HTTPRequest
			if err := json.Unmarshal(body, &req); err != nil {
				return nil, &HTTPRequestParseError{
					Request: r,
					Options: options,
					Err:     err,
				}
			}
			return &req, nil

		default:
			// Return silently without error for unsupported content-type.
			return &HTTPRequest{}, nil
		}

	default:
		// Return silently without error for unsupported method
		return &HTTPRequest{}, nil
	}
}
