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

// graphql-js/src/validation/__tests__/UniqueFragmentNames-test.js@8c96dc8
var _ = Describe("Validate: Unique fragment names", func() {
	expectErrors := func(queryStr string) GomegaAssertion {
		return expectValidationErrors(rules.UniqueFragmentNames{}, queryStr)
	}

	expectValid := func(queryStr string) {
		expectErrors(queryStr).Should(Equal(graphql.NoErrors()))
	}

	duplicateFrag := func(fragName string, l1 uint, c1 uint, l2 uint, c2 uint) error {
		return graphql.NewError(validator.DuplicateFragmentNameMessage(fragName), []graphql.ErrorLocation{
			{Line: l1, Column: c1},
			{Line: l2, Column: c2},
		})
	}

	It("no fragments", func() {
		expectValid(`
      {
        field
      }
    `)
	})

	It("one fragment", func() {
		expectValid(`
      {
        ...fragA
      }

      fragment fragA on Type {
        field
      }
    `)
	})

	It("many fragments", func() {
		expectValid(`
      {
        ...fragA
        ...fragB
        ...fragC
      }
      fragment fragA on Type {
        fieldA
      }
      fragment fragB on Type {
        fieldB
      }
      fragment fragC on Type {
        fieldC
      }
    `)
	})

	It("inline fragments are always unique", func() {
		expectValid(`
      {
        ...on Type {
          fieldA
        }
        ...on Type {
          fieldB
        }
      }
    `)
	})

	It("fragment and operation named the same", func() {
		expectValid(`
      query Foo {
        ...Foo
      }
      fragment Foo on Type {
        field
      }
    `)
	})

	It("fragments named the same", func() {
		expectErrors(`
      {
        ...fragA
      }
      fragment fragA on Type {
        fieldA
      }
      fragment fragA on Type {
        fieldB
      }
    `).Should(Equal(graphql.ErrorsOf(
			duplicateFrag("fragA", 5, 16, 8, 16),
		)))
	})

	It("fragments named the same without being referenced", func() {
		expectErrors(`
      fragment fragA on Type {
        fieldA
      }
      fragment fragA on Type {
        fieldB
      }
    `).Should(Equal(graphql.ErrorsOf(
			duplicateFrag("fragA", 2, 16, 5, 16),
		)))
	})
})
