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

package graphql_test

import (
	"encoding/json"
	"errors"

	"github.com/botobag/artemis/graphql"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func newError(message string, args ...interface{}) *graphql.Error {
	e, ok := graphql.NewError(message, args...).(*graphql.Error)
	Expect(ok).Should(BeTrue())
	return e
}

func wrapError(message string, err error) *graphql.Error {
	e, ok := graphql.WrapError(err, message).(*graphql.Error)
	Expect(ok).Should(BeTrue())
	return e
}

func expectSerializationResult(e error, expected string) {
	s, err := json.Marshal(e)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(s).Should(MatchJSON(expected))
}

func expectOutputResult(e error, expected string) {
	Expect(e.Error()).Should(Equal(expected), e.Error())
}

type errWithLocations struct {
	locations []graphql.ErrorLocation
}

// Locations implements graphql.ErrorWithLocations.
func (e *errWithLocations) Locations() []graphql.ErrorLocation {
	return e.locations
}

// Error implements Go's error interface
func (e *errWithLocations) Error() string {
	return "error provided locations"
}

type errWithPath struct {
	path *graphql.ResponsePath
}

// Path implements graphql.ErrorWithPath.
func (e *errWithPath) Path() *graphql.ResponsePath {
	return e.path
}

// Error implements Go's error interface
func (e *errWithPath) Error() string {
	return "error provided path"
}

type errWithExtensions struct {
	extensions graphql.ErrorExtensions
}

// Extensions implements graphql.ErrorWithExtensions.
func (e *errWithExtensions) Extensions() graphql.ErrorExtensions {
	return e.extensions
}

// Error implements Go's error interface
func (e *errWithExtensions) Error() string {
	return "error provided extensions"
}

var (
	_ graphql.ErrorWithLocations  = (*errWithLocations)(nil)
	_ graphql.ErrorWithPath       = (*errWithPath)(nil)
	_ graphql.ErrorWithExtensions = (*errWithExtensions)(nil)
	_ error                       = (*errWithLocations)(nil)
	_ error                       = (*errWithPath)(nil)
	_ error                       = (*errWithExtensions)(nil)
)

var _ = Describe("Error", func() {
	var (
		mockLocation   graphql.ErrorLocation
		mockLocation2  graphql.ErrorLocation
		mockPath       *graphql.ResponsePath
		mockExtensions graphql.ErrorExtensions
	)

	BeforeEach(func() {
		// TODO: Parse a real GraphQL document and retrieve the location from AST node.
		mockLocation = graphql.ErrorLocation{
			Line:   1,
			Column: 3,
		}

		mockLocation2 = graphql.ErrorLocation{
			Line:   2,
			Column: 5,
		}

		mockPath = &graphql.ResponsePath{}
		mockPath.AppendFieldName("path")
		mockPath.AppendIndex(3)
		mockPath.AppendFieldName("to")
		mockPath.AppendFieldName("field")

		mockExtensions = graphql.ErrorExtensions{
			"code": "CAN_NOT_FETCH_BY_ID",
		}
	})

	// graphql-js/src/error/__tests__/GraphQLError-test.js
	It("has a message", func() {
		e := newError("msg")
		Expect(e.Message).Should(Equal("msg"))
	})

	It("serializes to include message", func() {
		e := newError("msg")
		expectSerializationResult(e, `{"message":"msg"}`)
	})

	It("serializes to include message and locations", func() {
		e := newError("msg", mockLocation)
		expectSerializationResult(e, `{"message":"msg","locations":[{"line":1,"column":3}]}`)
	})

	It("serializes to include path", func() {
		e := newError("msg", mockPath)
		Expect(e.Path).Should(Equal(mockPath))
		expectSerializationResult(e, `{"message":"msg","path":["path",3,"to","field"]}`)
		expectOutputResult(e, `msg for response field in the path path[3].to.field`)
	})

	It("can include an underlying error", func() {
		underlyingErr := errors.New("hello")
		e := newError("msg", underlyingErr)
		Expect(e.Err).Should(Equal(underlyingErr))
	})

	It("can include an op and kind", func() {
		const op graphql.Op = "myop"
		e := newError("msg", op, graphql.ErrKindInternal)
		Expect(e.Op).Should(Equal(op))
		Expect(e.Kind).Should(Equal(graphql.ErrKindInternal))

		// But Op and Kind should not be included in serialization.
		expectSerializationResult(e, `{"message":"msg"}`)
		expectOutputResult(e, `myop: msg: internal error`)
	})

	It("can include multiple locations", func() {
		e := newError("msg", []graphql.ErrorLocation{mockLocation, mockLocation2})
		expectSerializationResult(e,
			`{"message":"msg","locations":[{"line":1,"column":3},{"line":2,"column":5}]}`)
		expectOutputResult(e,
			"msg at [{Line:1 Column:3} {Line:2 Column:5}]")
	})

	It("can include extensions", func() {
		e := newError("msg", mockExtensions)
		expectSerializationResult(e,
			`{"message":"msg","extensions":{"code":"CAN_NOT_FETCH_BY_ID"}}`)
		expectOutputResult(e, `msg (additional info: map[code:CAN_NOT_FETCH_BY_ID])`)
	})

	It("pulls locations from underlying error", func() {
		// Create an error with an errWithLocations.
		locations := []graphql.ErrorLocation{
			mockLocation,
			mockLocation2,
		}
		e := newError("error with locations", &errWithLocations{
			locations: locations,
		})
		Expect(e.Locations).Should(Equal(locations))
		expectSerializationResult(e,
			`{"message":"error with locations","locations":[{"line":1,"column":3},{"line":2,"column":5}]}`)
		expectOutputResult(e,
			`error with locations at [{Line:1 Column:3} {Line:2 Column:5}]: error provided locations`)

		// Wrap an error again without given new locations.
		e = wrapError("error wraps an error with locations", e)
		Expect(e.Locations).Should(Equal(locations))
		expectSerializationResult(e,
			`{"message":"error wraps an error with locations","locations":[{"line":1,"column":3},{"line":2,"column":5}]}`)
		expectOutputResult(e,
			`error wraps an error with locations at [{Line:1 Column:3} {Line:2 Column:5}]:
  error with locations: error provided locations`)

		// Wrap an error with custom locations.
		mockLocation3 := graphql.ErrorLocation{
			Line:   10,
			Column: 30,
		}
		e = newError("error wraps with custom locations", e, mockLocation3)
		Expect(e.Locations).Should(Equal([]graphql.ErrorLocation{mockLocation3}))
		expectSerializationResult(e,
			`{"message":"error wraps with custom locations","locations":[{"line":10,"column":30}]}`)

		expectOutputResult(e,
			`error wraps with custom locations at [{Line:10 Column:30}]:
  error wraps an error with locations at [{Line:1 Column:3} {Line:2 Column:5}]:
  error with locations: error provided locations`)
	})

	It("pulls path from underlying error", func() {
		// Create an error with an errWithPath.
		e := newError("error with path", &errWithPath{
			path: mockPath,
		})
		Expect(e.Path).Should(Equal(mockPath))
		expectSerializationResult(e,
			`{"message":"error with path","path":["path",3,"to","field"]}`)
		expectOutputResult(e, `error with path for response field in the path path[3].to.field: error provided path`)

		// Wrap an error again without given new path.
		e = wrapError("error wraps an error with path", e)
		Expect(e.Path).Should(Equal(mockPath))
		expectSerializationResult(e,
			`{"message":"error wraps an error with path","path":["path",3,"to","field"]}`)
		expectOutputResult(e,
			`error wraps an error with path for response field in the path path[3].to.field:
  error with path: error provided path`)

		// Wrap an error with custom path.
		mockPath2 := &graphql.ResponsePath{}
		mockPath2.AppendFieldName("another")
		mockPath2.AppendFieldName("path")
		mockPath2.AppendIndex(10)
		mockPath2.AppendFieldName("to")
		mockPath2.AppendFieldName("field")
		e = newError("error wraps with custom path", e, mockPath2)
		Expect(e.Path).Should(Equal(mockPath2))
		expectSerializationResult(e,
			`{"message":"error wraps with custom path","path":["another","path",10,"to","field"]}`)

		expectOutputResult(e,
			`error wraps with custom path for response field in the path another.path[10].to.field:
  error wraps an error with path for response field in the path path[3].to.field:
  error with path: error provided path`)
	})

	It("pulls extensions from underlying error", func() {
		// Create an error with an errWithExtensions.
		e := newError("error with extensions", &errWithExtensions{
			extensions: mockExtensions,
		})
		Expect(e.Extensions).Should(Equal(mockExtensions))
		expectSerializationResult(e,
			`{"message":"error with extensions","extensions":{"code":"CAN_NOT_FETCH_BY_ID"}}`)
		expectOutputResult(e, `error with extensions (additional info: map[code:CAN_NOT_FETCH_BY_ID]): error provided extensions`)

		// Wrap an error again without given new extensions.
		e = wrapError("error wraps an error with extensions", e)
		Expect(e.Extensions).Should(Equal(mockExtensions))
		expectSerializationResult(e,
			`{"message":"error wraps an error with extensions","extensions":{"code":"CAN_NOT_FETCH_BY_ID"}}`)
		expectOutputResult(e,
			`error wraps an error with extensions (additional info: map[code:CAN_NOT_FETCH_BY_ID]):
  error with extensions: error provided extensions`)

		// Wrap an error with custom extensions.
		mockExtensions2 := graphql.ErrorExtensions{
			"timestamp": "Fri Feb 9 14:33:09 UTC 2018",
		}
		e = newError("error wraps with custom extensions", e, mockExtensions2)
		Expect(e.Extensions).Should(Equal(mockExtensions2))
		expectSerializationResult(e,
			`{"message":"error wraps with custom extensions","extensions":{"timestamp":"Fri Feb 9 14:33:09 UTC 2018"}}`)

		expectOutputResult(e,
			`error wraps with custom extensions (additional info: map[timestamp:Fri Feb 9 14:33:09 UTC 2018]):
  error wraps an error with extensions (additional info: map[code:CAN_NOT_FETCH_BY_ID]):
  error with extensions: error provided extensions`)
	})

	It("pulls kind from underlying error", func() {
		e := newError("error without kind")
		Expect(e.Kind).Should(Equal(graphql.ErrKindOther))
		expectOutputResult(e, `error without kind`)

		// Wrap error without a kind still doesn't have kind.
		e = newError("wrap an error without kind", e)
		Expect(e.Kind).Should(Equal(graphql.ErrKindOther))
		expectOutputResult(e, `wrap an error without kind:
  error without kind`)

		// Wrap error with a kind.
		e = newError("wrap an error with kind", e, graphql.ErrKindCoercion)
		Expect(e.Kind).Should(Equal(graphql.ErrKindCoercion))
		expectOutputResult(e, `wrap an error with kind: coercion error:
  wrap an error without kind:
  error without kind`)

		// Wrap error without given a kind again.
		e = newError("wrap an error without kind #2", e)
		Expect(e.Kind).Should(Equal(graphql.ErrKindCoercion))
		expectOutputResult(e, `wrap an error without kind #2: coercion error:
  wrap an error with kind:
  wrap an error without kind:
  error without kind`)

		// Finally, wrap the error with new kind.
		e = newError("wrap an error with new kind", e, graphql.ErrKindSyntax)
		Expect(e.Kind).Should(Equal(graphql.ErrKindSyntax))
		expectOutputResult(e, `wrap an error with new kind: syntax error:
  wrap an error without kind #2: coercion error:
  wrap an error with kind:
  wrap an error without kind:
  error without kind`)
	})

	It("throws error when building from unknown argument", func() {
		e := graphql.NewError("msg", 1)
		Expect(e).ShouldNot(BeNil())
		Expect(e.Error()).Should(Equal("unknown type int, value 1 in error call"))
	})

	It("wraps error with formatting string", func() {
		e := graphql.WrapErrorf(errors.New("internal error"), "error for type %T", 1)
		Expect(e).ShouldNot(BeNil())
		Expect(e.Error()).Should(Equal("error for type int: internal error"))
	})
})
