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

// graphql-js/src/validation/__tests__/FragmentsOnCompositeTypes-test.js@8c96dc8
var _ = Describe("Validate: Fragments on composite types", func() {
	expectErrors := func(queryStr string) GomegaAssertion {
		return expectValidationErrors(rules.FragmentsOnCompositeTypes{}, queryStr)
	}

	expectValid := func(queryStr string) {
		expectErrors(queryStr).Should(Equal(graphql.NoErrors()))
	}

	fragmentOnNonComposite := func(
		fragName string,
		typeName string,
		line uint,
		column uint) error {

		return graphql.NewError(
			validator.FragmentOnNonCompositeErrorMessage(fragName, typeName),
			[]graphql.ErrorLocation{
				{Line: line, Column: column},
			},
		)
	}

	It("object is valid fragment type", func() {
		expectValid(`
      fragment validFragment on Dog {
        barks
      }
    `)
	})

	It("interface is valid fragment type", func() {
		expectValid(`
      fragment validFragment on Pet {
        name
      }
    `)
	})

	It("object is valid inline fragment type", func() {
		expectValid(`
      fragment validFragment on Pet {
        ... on Dog {
          barks
        }
      }
    `)
	})

	It("inline fragment without type is valid", func() {
		expectValid(`
      fragment validFragment on Pet {
        ... {
          name
        }
      }
    `)
	})

	It("union is valid fragment type", func() {
		expectValid(`
      fragment validFragment on CatOrDog {
        __typename
      }
    `)
	})

	It("scalar is invalid fragment type", func() {
		expectErrors(`
      fragment scalarFragment on Boolean {
        bad
      }
    `).Should(Equal(graphql.ErrorsOf(
			fragmentOnNonComposite("scalarFragment", "Boolean", 2, 34),
		)))
	})

	It("enum is invalid fragment type", func() {
		expectErrors(`
      fragment scalarFragment on FurColor {
        bad
      }
    `).Should(Equal(graphql.ErrorsOf(
			fragmentOnNonComposite("scalarFragment", "FurColor", 2, 34),
		)))
	})

	It("input object is invalid fragment type", func() {
		expectErrors(`
      fragment inputFragment on ComplexInput {
        stringField
      }
    `).Should(Equal(graphql.ErrorsOf(
			fragmentOnNonComposite("inputFragment", "ComplexInput", 2, 33),
		)))
	})

	It("scalar is invalid inline fragment type", func() {
		expectErrors(`
      fragment invalidFragment on Pet {
        ... on String {
          barks
        }
      }
    `).Should(Equal(graphql.ErrorsOf(
			graphql.NewError(
				validator.InlineFragmentOnNonCompositeErrorMessage("String"),
				[]graphql.ErrorLocation{
					{Line: 3, Column: 16},
				},
			),
		)))
	})
})
