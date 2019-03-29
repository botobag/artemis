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

package rules_test

import (
	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/graphql/internal/validator"
	"github.com/botobag/artemis/graphql/validator/rules"
	"github.com/botobag/artemis/internal/testutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// graphql-js/src/validation/__tests__/NoUndefinedVariables-test.js@8c96dc8
var _ = Describe("Validate: No undefined variables", func() {
	expectErrors := func(queryStr string) GomegaAssertion {
		return expectValidationErrors(rules.NoUndefinedVariables{}, queryStr)
	}

	expectValid := func(queryStr string) {
		expectErrors(queryStr).Should(Equal(graphql.NoErrors()))
	}

	undefVar := func(varName string, l1 uint, c1 uint, opName string, l2 uint, c2 uint) error {
		return graphql.NewError(
			validator.UndefinedVarMessage(varName, opName),
			[]graphql.ErrorLocation{
				{Line: l1, Column: c1},
				{Line: l2, Column: c2},
			},
		)
	}

	It("all variables defined", func() {
		expectValid(`
      query Foo($a: String, $b: String, $c: String) {
        field(a: $a, b: $b, c: $c)
      }
    `)
	})

	It("all variables deeply defined", func() {
		expectValid(`
      query Foo($a: String, $b: String, $c: String) {
        field(a: $a) {
          field(b: $b) {
            field(c: $c)
          }
        }
      }
    `)
	})

	It("all variables deeply in inline fragments defined", func() {
		expectValid(`
      query Foo($a: String, $b: String, $c: String) {
        ... on Type {
          field(a: $a) {
            field(b: $b) {
              ... on Type {
                field(c: $c)
              }
            }
          }
        }
      }
    `)
	})

	It("all variables in fragments deeply defined", func() {
		expectValid(`
      query Foo($a: String, $b: String, $c: String) {
        ...FragA
      }
      fragment FragA on Type {
        field(a: $a) {
          ...FragB
        }
      }
      fragment FragB on Type {
        field(b: $b) {
          ...FragC
        }
      }
      fragment FragC on Type {
        field(c: $c)
      }
    `)
	})

	It("variable within single fragment defined in multiple operations", func() {
		expectValid(`
      query Foo($a: String) {
        ...FragA
      }
      query Bar($a: String) {
        ...FragA
      }
      fragment FragA on Type {
        field(a: $a)
      }
    `)
	})

	It("variable within fragments defined in operations", func() {
		expectValid(`
      query Foo($a: String) {
        ...FragA
      }
      query Bar($b: String) {
        ...FragB
      }
      fragment FragA on Type {
        field(a: $a)
      }
      fragment FragB on Type {
        field(b: $b)
      }
    `)
	})

	It("variable within recursive fragment defined", func() {
		expectValid(`
      query Foo($a: String) {
        ...FragA
      }
      fragment FragA on Type {
        field(a: $a) {
          ...FragA
        }
      }
    `)
	})

	It("variable not defined", func() {
		expectErrors(`
      query Foo($a: String, $b: String, $c: String) {
        field(a: $a, b: $b, c: $c, d: $d)
      }
    `).Should(Equal(graphql.ErrorsOf(undefVar("d", 3, 39, "Foo", 2, 7))))
	})

	It("variable not defined by un-named query", func() {
		expectErrors(`
      {
        field(a: $a)
      }
   `).Should(Equal(graphql.ErrorsOf(undefVar("a", 3, 18, "", 2, 7))))
	})

	It("multiple variables not defined", func() {
		expectErrors(`
      query Foo($b: String) {
        field(a: $a, b: $b, c: $c)
      }
    `).Should(Equal(graphql.ErrorsOf(
			undefVar("a", 3, 18, "Foo", 2, 7),
			undefVar("c", 3, 32, "Foo", 2, 7),
		)))
	})

	It("variable in fragment not defined by un-named query", func() {
		expectErrors(`
      {
        ...FragA
      }
      fragment FragA on Type {
        field(a: $a)
      }
    `).Should(Equal(graphql.ErrorsOf(undefVar("a", 6, 18, "", 2, 7))))
	})

	It("variable in fragment not defined by operation", func() {
		expectErrors(`
      query Foo($a: String, $b: String) {
        ...FragA
      }
      fragment FragA on Type {
        field(a: $a) {
          ...FragB
        }
      }
      fragment FragB on Type {
        field(b: $b) {
          ...FragC
        }
      }
      fragment FragC on Type {
        field(c: $c)
      }
    `).Should(Equal(graphql.ErrorsOf(undefVar("c", 16, 18, "Foo", 2, 7))))
	})

	It("multiple variables in fragments not defined", func() {
		expectErrors(`
      query Foo($b: String) {
        ...FragA
      }
      fragment FragA on Type {
        field(a: $a) {
          ...FragB
        }
      }
      fragment FragB on Type {
        field(b: $b) {
          ...FragC
        }
      }
      fragment FragC on Type {
        field(c: $c)
      }
    `).Should(Equal(graphql.ErrorsOf(
			undefVar("a", 6, 18, "Foo", 2, 7),
			undefVar("c", 16, 18, "Foo", 2, 7),
		)))
	})

	It("single variable in fragment not defined by multiple operations", func() {
		expectErrors(`
      query Foo($a: String) {
        ...FragAB
      }
      query Bar($a: String) {
        ...FragAB
      }
      fragment FragAB on Type {
        field(a: $a, b: $b)
      }
    `).Should(Equal(graphql.ErrorsOf(
			undefVar("b", 9, 25, "Foo", 2, 7),
			undefVar("b", 9, 25, "Bar", 5, 7),
		)))
	})

	It("variables in fragment not defined by multiple operations", func() {
		expectErrors(`
      query Foo($b: String) {
        ...FragAB
      }
      query Bar($a: String) {
        ...FragAB
      }
      fragment FragAB on Type {
        field(a: $a, b: $b)
      }
    `).Should(Equal(graphql.ErrorsOf(
			undefVar("a", 9, 18, "Foo", 2, 7),
			undefVar("b", 9, 25, "Bar", 5, 7),
		)))
	})

	It("variable in fragment used by other operation", func() {
		expectErrors(`
      query Foo($b: String) {
        ...FragA
      }
      query Bar($a: String) {
        ...FragB
      }
      fragment FragA on Type {
        field(a: $a)
      }
      fragment FragB on Type {
        field(b: $b)
      }
    `).Should(Equal(graphql.ErrorsOf(
			undefVar("a", 9, 18, "Foo", 2, 7),
			undefVar("b", 12, 18, "Bar", 5, 7),
		)))
	})

	It("multiple undefined variables produce multiple errors", func() {
		expectErrors(`
      query Foo($b: String) {
        ...FragAB
      }
      query Bar($a: String) {
        ...FragAB
      }
      fragment FragAB on Type {
        field1(a: $a, b: $b)
        ...FragC
        field3(a: $a, b: $b)
      }
      fragment FragC on Type {
        field2(c: $c)
      }
    `).Should(testutil.ConsistOfGraphQLErrors(
			// We have the same set of errors but in a different order when compared to graphql-js.
			Equal(undefVar("a", 9, 19, "Foo", 2, 7)),
			Equal(undefVar("a", 11, 19, "Foo", 2, 7)),
			Equal(undefVar("c", 14, 19, "Foo", 2, 7)),
			Equal(undefVar("b", 9, 26, "Bar", 5, 7)),
			Equal(undefVar("b", 11, 26, "Bar", 5, 7)),
			Equal(undefVar("c", 14, 19, "Bar", 5, 7)),
		))
	})
})
