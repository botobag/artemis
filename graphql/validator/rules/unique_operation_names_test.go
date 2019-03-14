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

// graphql-js/src/validation/__tests__/UniqueOperationNames-test.js@8c96dc8
var _ = Describe("Validate: Unique operation names", func() {
	expectErrors := func(queryStr string) GomegaAssertion {
		return expectValidationErrors(rules.UniqueOperationNames{}, queryStr)
	}

	expectValid := func(queryStr string) {
		expectErrors(queryStr).Should(Equal(graphql.NoErrors()))
	}

	duplicateOp := func(opName string, l1 uint, c1 uint, l2 uint, c2 uint) error {
		return graphql.NewError(validator.DuplicateOperationNameMessage(opName), []graphql.ErrorLocation{
			{Line: l1, Column: c1},
			{Line: l2, Column: c2},
		})
	}

	It("no operations", func() {
		expectValid(`
      fragment fragA on Type {
        field
      }
    `)
	})

	It("one anon operation", func() {
		expectValid(`
      {
        field
      }
    `)
	})

	It("one named operation", func() {
		expectValid(`
      query Foo {
        field
      }
    `)
	})

	It("multiple operations", func() {
		expectValid(`
      query Foo {
        field
      }

      query Bar {
        field
      }
    `)
	})

	It("multiple operations of different types", func() {
		expectValid(`
      query Foo {
        field
      }

      mutation Bar {
        field
      }

      subscription Baz {
        field
      }
    `)
	})

	It("multiple operations of different types", func() {
		expectValid(`
      query Foo {
        ...Foo
      }
      fragment Foo on Type {
        field
      }
    `)
	})

	It("multiple operations of same name", func() {
		expectErrors(`
      query Foo {
        fieldA
      }
      query Foo {
        fieldB
      }
    `).Should(Equal(graphql.ErrorsOf(duplicateOp("Foo", 2, 13, 5, 13))))
	})

	It("multiple ops of same name of different types (mutation)", func() {
		expectErrors(`
      query Foo {
        fieldA
      }
      mutation Foo {
        fieldB
      }
    `).Should(Equal(graphql.ErrorsOf(duplicateOp("Foo", 2, 13, 5, 16))))
	})

	It("multiple ops of same name of different types (subscription)", func() {
		expectErrors(`
      query Foo {
        fieldA
      }
      subscription Foo {
        fieldB
      }
    `).Should(Equal(graphql.ErrorsOf(duplicateOp("Foo", 2, 13, 5, 20))))
	})
})
