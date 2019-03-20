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

// graphql-js/src/validation/__tests__/ProvidedRequiredArguments-test.js@8c96dc8
var _ = Describe("Validate: Provided required arguments", func() {
	expectErrors := func(queryStr string) GomegaAssertion {
		return expectValidationErrors(rules.ProvidedRequiredArguments{}, queryStr)
	}

	expectValid := func(queryStr string) {
		expectErrors(queryStr).Should(Equal(graphql.NoErrors()))
	}

	missingFieldArg := func(
		fieldName string,
		argName string,
		typeName string,
		line uint,
		column uint) error {

		return graphql.NewError(
			validator.MissingFieldArgMessage(argName, fieldName, typeName),
			[]graphql.ErrorLocation{
				{Line: line, Column: column},
			},
		)
	}

	missingDirectiveArg := func(
		directiveName string,
		argName string,
		typeName string,
		line uint,
		column uint) error {

		return graphql.NewError(
			validator.MissingDirectiveArgMessage(argName, directiveName, typeName),
			[]graphql.ErrorLocation{
				{Line: line, Column: column},
			},
		)
	}

	It("ignores unknown arguments", func() {
		expectValid(`
      {
        dog {
          isHousetrained(unknownArgument: true)
        }
      }
    `)
	})

	Describe("Valid non-nullable value", func() {
		It("Arg on optional arg", func() {
			expectValid(`
        {
          dog {
            isHousetrained(atOtherHomes: true)
          }
        }
      `)
		})

		It("No Arg on optional arg", func() {
			expectValid(`
        {
          dog {
            isHousetrained
          }
        }
      `)
		})

		It("No arg on non-null field with default", func() {
			expectValid(`
        {
          complicatedArgs {
            nonNullFieldWithDefault
          }
        }
      `)
		})

		It("Multiple args", func() {
			expectValid(`
        {
          complicatedArgs {
            multipleReqs(req1: 1, req2: 2)
          }
        }
      `)
		})

		It("Multiple args reverse order", func() {
			expectValid(`
        {
          complicatedArgs {
            multipleReqs(req2: 2, req1: 1)
          }
        }
      `)
		})

		It("No args on multiple optional", func() {
			expectValid(`
        {
          complicatedArgs {
            multipleOpts
          }
        }
      `)
		})

		It("One arg on multiple optional", func() {
			expectValid(`
        {
          complicatedArgs {
            multipleOpts(opt1: 1)
          }
        }
      `)
		})

		It("Second arg on multiple optional", func() {
			expectValid(`
        {
          complicatedArgs {
            multipleOpts(opt2: 1)
          }
        }
      `)
		})

		It("Multiple reqs on mixedList", func() {
			expectValid(`
        {
          complicatedArgs {
            multipleOptAndReq(req1: 3, req2: 4)
          }
        }
      `)
		})

		It("Multiple reqs and one opt on mixedList", func() {
			expectValid(`
        {
          complicatedArgs {
            multipleOptAndReq(req1: 3, req2: 4, opt1: 5)
          }
        }
      `)
		})

		It("All reqs and opts on mixedList", func() {
			expectValid(`
        {
          complicatedArgs {
            multipleOptAndReq(req1: 3, req2: 4, opt1: 5, opt2: 6)
          }
        }
      `)
		})
	})

	Describe("Invalid non-nullable value", func() {
		It("Missing one non-nullable argument", func() {
			expectErrors(`
        {
          complicatedArgs {
            multipleReqs(req2: 2)
          }
        }
      `).Should(Equal(graphql.ErrorsOf(
				missingFieldArg("multipleReqs", "req1", "Int!", 4, 13),
			)))
		})

		It("Missing multiple non-nullable arguments", func() {
			expectErrors(`
        {
          complicatedArgs {
            multipleReqs
          }
        }
      `).Should(Or(
				Equal(graphql.ErrorsOf(
					missingFieldArg("multipleReqs", "req1", "Int!", 4, 13),
					missingFieldArg("multipleReqs", "req2", "Int!", 4, 13),
				)),
				Equal(graphql.ErrorsOf(
					missingFieldArg("multipleReqs", "req2", "Int!", 4, 13),
					missingFieldArg("multipleReqs", "req1", "Int!", 4, 13),
				)),
			))
		})

		It("Incorrect value and missing argument", func() {
			expectErrors(`
        {
          complicatedArgs {
            multipleReqs(req1: "one")
          }
        }
      `).Should(Equal(graphql.ErrorsOf(
				missingFieldArg("multipleReqs", "req2", "Int!", 4, 13),
			)))
		})
	})

	Describe("Directive arguments", func() {
		It("ignores unknown directives", func() {
			expectValid(`
        {
          dog @unknown
        }
      `)
		})

		It("with directives of valid types", func() {
			expectValid(`
        {
          dog @include(if: true) {
            name
          }
          human @skip(if: false) {
            name
          }
        }
      `)
		})

		It("with directive with missing types", func() {
			expectErrors(`
        {
          dog @include {
            name @skip
          }
        }
      `).Should(Equal(graphql.ErrorsOf(
				missingDirectiveArg("include", "if", "Boolean!", 3, 15),
				missingDirectiveArg("skip", "if", "Boolean!", 4, 18),
			)))
		})
	})
})
