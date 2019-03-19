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

// graphql-js/src/validation/__tests__/KnownArgumentNames-test.js@8c96dc8
var _ = Describe("Validate: Known argument names", func() {
	expectErrors := func(queryStr string) GomegaAssertion {
		return expectValidationErrors(rules.KnownArgumentNames{}, queryStr)
	}

	expectValid := func(queryStr string) {
		expectErrors(queryStr).Should(Equal(graphql.NoErrors()))
	}

	unknownArg := func(
		argName string,
		fieldName string,
		typeName string,
		suggestedArgs []string,
		line uint,
		column uint) error {

		return graphql.NewError(
			validator.UnknownArgMessage(argName, fieldName, typeName, suggestedArgs),
			[]graphql.ErrorLocation{
				{Line: line, Column: column},
			},
		)
	}

	unknownDirectiveArg := func(
		argName string,
		directiveName string,
		suggestedArgs []string,
		line uint,
		column uint) error {

		return graphql.NewError(
			validator.UnknownDirectiveArgMessage(argName, directiveName, suggestedArgs),
			[]graphql.ErrorLocation{
				{Line: line, Column: column},
			},
		)
	}

	It("single arg is known", func() {
		expectValid(`
      fragment argOnRequiredArg on Dog {
        doesKnowCommand(dogCommand: SIT)
      }
    `)
	})

	It("multiple args are known", func() {
		expectValid(`
      fragment multipleArgs on ComplicatedArgs {
        multipleReqs(req1: 1, req2: 2)
      }
    `)
	})

	It("ignores args of unknown fields", func() {
		expectValid(`
      fragment argOnUnknownField on Dog {
        unknownField(unknownArg: SIT)
      }
    `)
	})

	It("multiple args in reverse order are known", func() {
		expectValid(`
      fragment multipleArgsReverseOrder on ComplicatedArgs {
        multipleReqs(req2: 2, req1: 1)
      }
    `)
	})

	It("no args on optional arg", func() {
		expectValid(`
      fragment noArgOnOptionalArg on Dog {
        isHousetrained
      }
    `)
	})

	It("args are known deeply", func() {
		expectValid(`
      {
        dog {
          doesKnowCommand(dogCommand: SIT)
        }
        human {
          pet {
            ... on Dog {
              doesKnowCommand(dogCommand: SIT)
            }
          }
        }
      }
    `)
	})

	It("directive args are known", func() {
		expectValid(`
      {
        dog @skip(if: true)
      }
    `)
	})

	It("field args are invalid", func() {
		expectErrors(`
      {
        dog @skip(unless: true)
      }
    `).Should(Equal(graphql.ErrorsOf(
			unknownDirectiveArg("unless", "skip", nil, 3, 19),
		)))
	})

	It("misspelled directive args are reported", func() {
		expectErrors(`
      {
        dog @skip(iff: true)
      }
    `).Should(Equal(graphql.ErrorsOf(
			unknownDirectiveArg("iff", "skip", []string{"if"}, 3, 19),
		)))
	})

	It("invalid arg name", func() {
		expectErrors(`
      fragment invalidArgName on Dog {
        doesKnowCommand(unknown: true)
      }
    `).Should(Equal(graphql.ErrorsOf(
			unknownArg("unknown", "doesKnowCommand", "Dog", nil, 3, 25),
		)))
	})

	It("misspelled arg name is reported", func() {
		expectErrors(`
      fragment invalidArgName on Dog {
        doesKnowCommand(dogcommand: true)
      }
    `).Should(Equal(graphql.ErrorsOf(
			unknownArg("dogcommand", "doesKnowCommand", "Dog", []string{"dogCommand"}, 3, 25),
		)))
	})

	It("unknown args amongst known args", func() {
		expectErrors(`
      fragment oneGoodArgOneInvalidArg on Dog {
        doesKnowCommand(whoknows: 1, dogCommand: SIT, unknown: true)
      }
    `).Should(Equal(graphql.ErrorsOf(
			unknownArg("whoknows", "doesKnowCommand", "Dog", nil, 3, 25),
			unknownArg("unknown", "doesKnowCommand", "Dog", nil, 3, 55),
		)))
	})

	It("unknown args deeply", func() {
		expectErrors(`
      {
        dog {
          doesKnowCommand(unknown: true)
        }
        human {
          pet {
            ... on Dog {
              doesKnowCommand(unknown: true)
            }
          }
        }
      }
    `).Should(Equal(graphql.ErrorsOf(
			unknownArg("unknown", "doesKnowCommand", "Dog", nil, 4, 27),
			unknownArg("unknown", "doesKnowCommand", "Dog", nil, 9, 31),
		)))
	})
})
