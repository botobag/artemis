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

// graphql-js/src/validation/__tests__/LoneAnonymousOperation-test.js@8c96dc8
var _ = Describe("Validate: Anonymous operation must be alone", func() {
	expectErrors := func(queryStr string) GomegaAssertion {
		return expectValidationErrors(rules.LoneAnonymousOperation{}, queryStr)
	}

	expectValid := func(queryStr string) {
		expectErrors(queryStr).Should(Equal(graphql.NoErrors()))
	}

	anonOperationNotAlone := func(line, column uint) error {
		return graphql.NewError(validator.AnonOperationNotAloneMessage(), []graphql.ErrorLocation{
			{Line: line, Column: column},
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

	It("multiple named operations", func() {
		expectValid(`
      query Foo {
        field
      }

      query Bar {
        field
      }
    `)
	})

	It("anon operation with fragment", func() {
		expectValid(`
      {
        ...Foo
      }
      fragment Foo on Type {
        field
      }
    `)
	})

	It("multiple anon operations", func() {
		expectErrors(`
      {
        fieldA
      }
      {
        fieldB
      }
    `).Should(Equal(graphql.ErrorsOf(
			anonOperationNotAlone(2, 7),
			anonOperationNotAlone(5, 7),
		)))
	})

	It("anon operation with a mutation", func() {
		expectErrors(`
      {
        fieldA
      }
      mutation Foo {
        fieldB
      }
		`).Should(Equal(graphql.ErrorsOf(anonOperationNotAlone(2, 7))))
	})

	It("anon operation with a subscription", func() {
		expectErrors(`
      {
        fieldA
      }
      subscription Foo {
        fieldB
      }
		`).Should(Equal(graphql.ErrorsOf(anonOperationNotAlone(2, 7))))
	})
})
