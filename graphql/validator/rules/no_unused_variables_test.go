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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// graphql-js/src/validation/__tests__/NoUnusedVariables-test.js@8c96dc8
var _ = Describe("Validate: No used variables", func() {
	expectErrors := func(queryStr string) GomegaAssertion {
		return expectValidationErrors(rules.NoUnusedVariables{}, queryStr)
	}

	expectValid := func(queryStr string) {
		expectErrors(queryStr).Should(Equal(graphql.NoErrors()))
	}

	unusedVar := func(varName string, opName string, line uint, column uint) error {
		return graphql.NewError(
			validator.UnusedVariableMessage(varName, opName),
			[]graphql.ErrorLocation{
				{Line: line, Column: column},
			},
		)
	}

	It("uses all variables", func() {
		expectValid(`
      query ($a: String, $b: String, $c: String) {
        field(a: $a, b: $b, c: $c)
      }
    `)
	})

	It("uses all variables deeply", func() {
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

	It("uses all variables deeply in inline fragments", func() {
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

	It("uses all variables in fragments", func() {
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

	It("variable used by fragment in multiple operations", func() {
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

	It("variable used by recursive fragment", func() {
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

	It("variable not used", func() {
		expectErrors(`
      query ($a: String, $b: String, $c: String) {
        field(a: $a, b: $b)
      }
    `).Should(Equal(graphql.ErrorsOf(unusedVar("c", "", 2, 38))))
	})

	It("multiple variables not used", func() {
		expectErrors(`
      query Foo($a: String, $b: String, $c: String) {
        field(b: $b)
      }
    `).Should(Equal(graphql.ErrorsOf(
			unusedVar("a", "Foo", 2, 17),
			unusedVar("c", "Foo", 2, 41),
		)))
	})

	It("variable not used in fragments", func() {
		expectErrors(`
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
        field
      }
    `).Should(Equal(graphql.ErrorsOf(unusedVar("c", "Foo", 2, 41))))
	})

	It("multiple variables not used in fragments", func() {
		expectErrors(`
      query Foo($a: String, $b: String, $c: String) {
        ...FragA
      }
      fragment FragA on Type {
        field {
          ...FragB
        }
      }
      fragment FragB on Type {
        field(b: $b) {
          ...FragC
        }
      }
      fragment FragC on Type {
        field
      }
    `).Should(Equal(graphql.ErrorsOf(
			unusedVar("a", "Foo", 2, 17),
			unusedVar("c", "Foo", 2, 41),
		)))
	})

	It("variable not used by unreferenced fragment", func() {
		expectErrors(`
      query Foo($b: String) {
        ...FragA
      }
      fragment FragA on Type {
        field(a: $a)
      }
      fragment FragB on Type {
        field(b: $b)
      }
    `).Should(Equal(graphql.ErrorsOf(unusedVar("b", "Foo", 2, 17))))
	})

	It("variable not used by fragment used by other operation", func() {
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
			unusedVar("b", "Foo", 2, 17),
			unusedVar("a", "Bar", 5, 17),
		)))
	})
})
