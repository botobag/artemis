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

// graphql-js/src/validation/__tests__/VariablesInAllowedPosition-test.js@8c96dc8
var _ = Describe("Validate: Variables are in allowed positions", func() {
	expectErrors := func(queryStr string) GomegaAssertion {
		return expectValidationErrors(rules.VariablesInAllowedPosition{}, queryStr)
	}

	expectValid := func(queryStr string) {
		expectErrors(queryStr).Should(Equal(graphql.NoErrors()))
	}

	It("Boolean => Boolean", func() {
		expectValid(`
      query Query($booleanArg: Boolean)
      {
        complicatedArgs {
          booleanArgField(booleanArg: $booleanArg)
        }
      }
    `)
	})

	It("Boolean => Boolean within fragment", func() {
		expectValid(`
      fragment booleanArgFrag on ComplicatedArgs {
        booleanArgField(booleanArg: $booleanArg)
      }
      query Query($booleanArg: Boolean)
      {
        complicatedArgs {
          ...booleanArgFrag
        }
      }
    `)

		expectValid(`
      query Query($booleanArg: Boolean)
      {
        complicatedArgs {
          ...booleanArgFrag
        }
      }
      fragment booleanArgFrag on ComplicatedArgs {
        booleanArgField(booleanArg: $booleanArg)
      }
    `)
	})

	It("Boolean! => Boolean", func() {
		expectValid(`
      query Query($nonNullBooleanArg: Boolean!)
      {
        complicatedArgs {
          booleanArgField(booleanArg: $nonNullBooleanArg)
        }
      }
    `)
	})

	It("Boolean! => Boolean within fragment", func() {
		expectValid(`
      fragment booleanArgFrag on ComplicatedArgs {
        booleanArgField(booleanArg: $nonNullBooleanArg)
      }

      query Query($nonNullBooleanArg: Boolean!)
      {
        complicatedArgs {
          ...booleanArgFrag
        }
      }
    `)
	})

	It("[String] => [String]", func() {
		expectValid(`
      query Query($stringListVar: [String])
      {
        complicatedArgs {
          stringListArgField(stringListArg: $stringListVar)
        }
      }
    `)
	})

	It("[String!] => [String]", func() {
		expectValid(`
      query Query($stringListVar: [String!])
      {
        complicatedArgs {
          stringListArgField(stringListArg: $stringListVar)
        }
      }
    `)
	})

	It("String => [String] in item position", func() {
		expectValid(`
      query Query($stringVar: String)
      {
        complicatedArgs {
          stringListArgField(stringListArg: [$stringVar])
        }
      }
    `)
	})

	It("String! => [String] in item position", func() {
		expectValid(`
      query Query($stringVar: String!)
      {
        complicatedArgs {
          stringListArgField(stringListArg: [$stringVar])
        }
      }
    `)
	})

	It("ComplexInput => ComplexInput", func() {
		expectValid(`
      query Query($complexVar: ComplexInput)
      {
        complicatedArgs {
          complexArgField(complexArg: $complexVar)
        }
      }
    `)
	})

	It("ComplexInput => ComplexInput in field position", func() {
		expectValid(`
      query Query($boolVar: Boolean = false)
      {
        complicatedArgs {
          complexArgField(complexArg: {requiredArg: $boolVar})
        }
      }
    `)
	})

	It("Boolean! => Boolean! in directive", func() {
		expectValid(`
      query Query($boolVar: Boolean!)
      {
        dog @include(if: $boolVar)
      }
    `)
	})

	It("Int => Int!", func() {
		expectErrors(`
      query Query($intArg: Int) {
        complicatedArgs {
          nonNullIntArgField(nonNullIntArg: $intArg)
        }
      }
    `).Should(Equal(graphql.ErrorsOf(
			graphql.NewError(
				validator.BadVarPosMessage("intArg", "Int", "Int!"),
				[]graphql.ErrorLocation{
					{Line: 2, Column: 19},
					{Line: 4, Column: 45},
				},
			),
		)))
	})

	It("Int => Int! within fragment", func() {
		expectErrors(`
      fragment nonNullIntArgFieldFrag on ComplicatedArgs {
        nonNullIntArgField(nonNullIntArg: $intArg)
      }

      query Query($intArg: Int) {
        complicatedArgs {
          ...nonNullIntArgFieldFrag
        }
      }
    `).Should(Equal(graphql.ErrorsOf(
			graphql.NewError(
				validator.BadVarPosMessage("intArg", "Int", "Int!"),
				[]graphql.ErrorLocation{
					{Line: 6, Column: 19},
					{Line: 3, Column: 43},
				},
			),
		)))
	})

	It("Int => Int! within nested fragment", func() {
		expectErrors(`
      fragment outerFrag on ComplicatedArgs {
        ...nonNullIntArgFieldFrag
      }

      fragment nonNullIntArgFieldFrag on ComplicatedArgs {
        nonNullIntArgField(nonNullIntArg: $intArg)
      }

      query Query($intArg: Int) {
        complicatedArgs {
          ...outerFrag
        }
      }
    `).Should(Equal(graphql.ErrorsOf(
			graphql.NewError(
				validator.BadVarPosMessage("intArg", "Int", "Int!"),
				[]graphql.ErrorLocation{
					{Line: 10, Column: 19},
					{Line: 7, Column: 43},
				},
			),
		)))
	})

	It("String over Boolean", func() {
		expectErrors(`
      query Query($stringVar: String) {
        complicatedArgs {
          booleanArgField(booleanArg: $stringVar)
        }
      }
    `).Should(Equal(graphql.ErrorsOf(
			graphql.NewError(
				validator.BadVarPosMessage("stringVar", "String", "Boolean"),
				[]graphql.ErrorLocation{
					{Line: 2, Column: 19},
					{Line: 4, Column: 39},
				},
			),
		)))
	})

	It("String => [String]", func() {
		expectErrors(`
      query Query($stringVar: String) {
        complicatedArgs {
          stringListArgField(stringListArg: $stringVar)
        }
      }
    `).Should(Equal(graphql.ErrorsOf(
			graphql.NewError(
				validator.BadVarPosMessage("stringVar", "String", "[String]"),
				[]graphql.ErrorLocation{
					{Line: 2, Column: 19},
					{Line: 4, Column: 45},
				},
			),
		)))
	})

	It("Boolean => Boolean! in directive", func() {
		expectErrors(`
      query Query($boolVar: Boolean) {
        dog @include(if: $boolVar)
      }
    `).Should(Equal(graphql.ErrorsOf(
			graphql.NewError(
				validator.BadVarPosMessage("boolVar", "Boolean", "Boolean!"),
				[]graphql.ErrorLocation{
					{Line: 2, Column: 19},
					{Line: 3, Column: 26},
				},
			),
		)))
	})

	It("String => Boolean! in directive", func() {
		expectErrors(`
      query Query($stringVar: String) {
        dog @include(if: $stringVar)
      }
    `).Should(Equal(graphql.ErrorsOf(
			graphql.NewError(
				validator.BadVarPosMessage("stringVar", "String", "Boolean!"),
				[]graphql.ErrorLocation{
					{Line: 2, Column: 19},
					{Line: 3, Column: 26},
				},
			),
		)))
	})

	It("[String] => [String!]", func() {
		expectErrors(`
      query Query($stringListVar: [String])
      {
        complicatedArgs {
          stringListNonNullArgField(stringListNonNullArg: $stringListVar)
        }
      }
    `).Should(Equal(graphql.ErrorsOf(
			graphql.NewError(
				validator.BadVarPosMessage("stringListVar", "[String]", "[String!]"),
				[]graphql.ErrorLocation{
					{Line: 2, Column: 19},
					{Line: 5, Column: 59},
				},
			),
		)))
	})

	Describe("Allows optional (nullable) variables with default values", func() {
		It("Int => Int! fails when variable provides null default value", func() {
			expectErrors(`
        query Query($intVar: Int = null) {
          complicatedArgs {
            nonNullIntArgField(nonNullIntArg: $intVar)
          }
        }
      `).Should(Equal(graphql.ErrorsOf(
				graphql.NewError(
					validator.BadVarPosMessage("intVar", "Int", "Int!"),
					[]graphql.ErrorLocation{
						{Line: 2, Column: 21},
						{Line: 4, Column: 47},
					},
				),
			)))
		})

		It("Int => Int! when variable provides non-null default value", func() {
			expectValid(`
        query Query($intVar: Int = 1) {
          complicatedArgs {
            nonNullIntArgField(nonNullIntArg: $intVar)
          }
        }`)
		})

		It("Int => Int! when optional argument provides default value", func() {
			expectValid(`
        query Query($intVar: Int) {
          complicatedArgs {
            nonNullFieldWithDefault(nonNullIntArg: $intVar)
          }
        }`)
		})

		It("Boolean => Boolean! in directive with default value with option", func() {
			expectValid(`
        query Query($boolVar: Boolean = false) {
          dog @include(if: $boolVar)
        }`)
		})
	})
})
