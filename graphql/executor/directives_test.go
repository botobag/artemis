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

package executor_test

import (
	"context"

	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/graphql/executor"
	"github.com/botobag/artemis/graphql/parser"
	"github.com/botobag/artemis/graphql/token"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Execute: handles directives", func() {
	// graphql-js/src/execution/__tests__/directives-test.js
	var executeTestQuery func(query string) <-chan executor.ExecutionResult

	BeforeEach(func() {
		schema, err := graphql.NewSchema(&graphql.SchemaConfig{
			Query: graphql.MustNewObject(&graphql.ObjectConfig{
				Name: "TestType",
				Fields: graphql.Fields{
					"a": {
						Type: graphql.T(graphql.String()),
					},
					"b": {
						Type: graphql.T(graphql.String()),
					},
				},
			}),
		})
		Expect(err).ShouldNot(HaveOccurred())

		rootValue := struct {
			A func(ctx context.Context) (interface{}, error)
			B func(ctx context.Context) (interface{}, error)
		}{
			A: func(ctx context.Context) (interface{}, error) {
				return "a", nil
			},
			B: func(ctx context.Context) (interface{}, error) {
				return "b", nil
			},
		}

		executeTestQuery = func(query string) <-chan executor.ExecutionResult {
			document := parser.MustParse(token.NewSource(query))
			return execute(schema, document, executor.RootValue(rootValue))
		}
	})

	Describe("works without directives", func() {
		It("basic query works", func() {
			result := executeTestQuery("{ a, b }")
			Eventually(result).Should(MatchResultInJSON(`{
				"data": {
					"a": "a",
					"b": "b"
				}
			}`))
		})
	})

	Describe("works on scalars", func() {
		It("if true includes scalar", func() {
			result := executeTestQuery("{ a, b @include(if: true) }")
			Eventually(result).Should(MatchResultInJSON(`{
				"data": {
					"a": "a",
					"b": "b"
				}
			}`))
		})

		It("if false omits on scalar", func() {
			result := executeTestQuery("{ a, b @include(if: false) }")
			Eventually(result).Should(MatchResultInJSON(`{
				"data": {
					"a": "a"
				}
			}`))
		})

		It("unless false includes scalar", func() {
			result := executeTestQuery("{ a, b @skip(if: false) }")
			Eventually(result).Should(MatchResultInJSON(`{
				"data": {
					"a": "a",
					"b": "b"
				}
			}`))
		})

		It("unless true omits scalar", func() {
			result := executeTestQuery("{ a, b @skip(if: true) }")
			Eventually(result).Should(MatchResultInJSON(`{
				"data": {
					"a": "a"
				}
			}`))
		})
	})

	Describe("works on fragment spreads", func() {
		It("if false omits fragment spread", func() {
			result := executeTestQuery(`
				query {
					a
					...Frag @include(if: false)
				}
				fragment Frag on TestType {
					b
				}
			`)
			Eventually(result).Should(MatchResultInJSON(`{
				"data": {
					"a": "a"
				}
			}`))
		})

		It("if true includes fragment spread", func() {
			result := executeTestQuery(`
				query {
					a
					...Frag @include(if: true)
				}
				fragment Frag on TestType {
					b
				}
			`)
			Eventually(result).Should(MatchResultInJSON(`{
				"data": {
					"a": "a",
					"b": "b"
				}
			}`))
		})

		It("unless false includes fragment spread", func() {
			result := executeTestQuery(`
				query {
					a
					...Frag @skip(if: false)
				}
				fragment Frag on TestType {
					b
				}
			`)
			Eventually(result).Should(MatchResultInJSON(`{
				"data": {
					"a": "a",
					"b": "b"
				}
			}`))
		})

		It("unless true omits fragment spread", func() {
			result := executeTestQuery(`
				query {
					a
					...Frag @skip(if: true)
				}
				fragment Frag on TestType {
					b
				}
			`)
			Eventually(result).Should(MatchResultInJSON(`{
				"data": {
					"a": "a"
				}
			}`))
		})
	})

	Describe("works on inline fragment", func() {
		It("if false omits inline fragment", func() {
			result := executeTestQuery(`
				query {
					a
					... on TestType @include(if: false) {
						b
					}
				}
			`)
			Eventually(result).Should(MatchResultInJSON(`{
				"data": {
					"a": "a"
				}
			}`))
		})

		It("if true includes inline fragment", func() {
			result := executeTestQuery(`
				query {
					a
					... on TestType @include(if: true) {
						b
					}
				}
			`)
			Eventually(result).Should(MatchResultInJSON(`{
				"data": {
					"a": "a",
					"b": "b"
				}
			}`))
		})

		It("unless false includes inline fragment", func() {
			result := executeTestQuery(`
				query {
					a
					... on TestType @skip(if: false) {
						b
					}
				}
			`)
			Eventually(result).Should(MatchResultInJSON(`{
				"data": {
					"a": "a",
					"b": "b"
				}
			}`))
		})

		It("unless true includes inline fragment", func() {
			result := executeTestQuery(`
				query {
					a
					... on TestType @skip(if: true) {
						b
					}
				}
			`)
			Eventually(result).Should(MatchResultInJSON(`{
				"data": {
					"a": "a"
				}
			}`))
		})
	})

	Describe("works on anonymous inline fragment", func() {
		It("if false omits anonymous inline fragment", func() {
			result := executeTestQuery(`
				query {
					a
					... @include(if: false) {
						b
					}
				}
			`)
			Eventually(result).Should(MatchResultInJSON(`{
				"data": {
					"a": "a"
				}
			}`))
		})

		It("if true includes anonymous inline fragment", func() {
			result := executeTestQuery(`
				query {
					a
					... @include(if: true) {
						b
					}
				}
			`)
			Eventually(result).Should(MatchResultInJSON(`{
				"data": {
					"a": "a",
					"b": "b"
				}
			}`))
		})

		It("unless false includes anonymous inline fragment", func() {
			result := executeTestQuery(`
				query {
					a
					... @skip(if: false) {
						b
					}
				}
			`)
			Eventually(result).Should(MatchResultInJSON(`{
				"data": {
					"a": "a",
					"b": "b"
				}
			}`))
		})

		It("unless true includes anonymous inline fragment", func() {
			result := executeTestQuery(`
				query {
					a
					... @skip(if: true) {
						b
					}
				}
			`)
			Eventually(result).Should(MatchResultInJSON(`{
				"data": {
					"a": "a"
				}
			}`))
		})
	})

	Describe("works with skip and include directives", func() {
		It("include and no skip", func() {
			result := executeTestQuery(`
				{
					a
					b @include(if: true) @skip(if: false)
				}
			`)
			Eventually(result).Should(MatchResultInJSON(`{
				"data": {
					"a": "a",
					"b": "b"
				}
			}`))
		})

		It("include and skip", func() {
			result := executeTestQuery(`
				{
					a
					b @include(if: true) @skip(if: true)
				}
			`)
			Eventually(result).Should(MatchResultInJSON(`{
				"data": {
					"a": "a"
				}
			}`))
		})

		It("no include or skip", func() {
			result := executeTestQuery(`
				{
					a
					b @include(if: false) @skip(if: false)
				}
			`)
			Eventually(result).Should(MatchResultInJSON(`{
				"data": {
					"a": "a"
				}
			}`))
		})
	})
})
