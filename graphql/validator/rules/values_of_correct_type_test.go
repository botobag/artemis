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
	"github.com/onsi/gomega/types"
)

// graphql-js/src/validation/__tests__/ValuesOfCorrectType-test.js@8c96dc8
var _ = Describe("Validate: Values of correct type", func() {
	expectErrors := func(queryStr string) GomegaAssertion {
		return expectValidationErrors(rules.ValuesOfCorrectType{}, queryStr)
	}

	expectValid := func(queryStr string) {
		expectErrors(queryStr).Should(Equal(graphql.NoErrors()))
	}

	badValue := func(
		typeName string,
		valueName string,
		suggestedNames []string,
		line uint,
		column uint) error {

		return graphql.NewError(
			validator.BadValueMessage(typeName, valueName, suggestedNames),
			[]graphql.ErrorLocation{
				{Line: line, Column: column},
			},
		)
	}

	requiredField := func(
		typeName string,
		fieldName string,
		fieldTypeName string,
		line uint,
		column uint) error {

		return graphql.NewError(
			validator.RequiredFieldMessage(typeName, fieldName, fieldTypeName),
			[]graphql.ErrorLocation{
				{Line: line, Column: column},
			},
		)
	}

	unknownField := func(
		typeName string,
		fieldName string,
		suggestedFields []string,
		line uint,
		column uint) error {

		return graphql.NewError(
			validator.UnknownFieldMessage(typeName, fieldName, suggestedFields),
			[]graphql.ErrorLocation{
				{Line: line, Column: column},
			},
		)
	}

	Describe("Valid values", func() {
		It("Good int value", func() {
			expectValid(`
        {
          complicatedArgs {
            intArgField(intArg: 2)
          }
        }
      `)
		})

		It("Good negative int value", func() {
			expectValid(`
        {
          complicatedArgs {
            intArgField(intArg: -2)
          }
        }
      `)
		})

		It("Good boolean value", func() {
			expectValid(`
        {
          complicatedArgs {
            booleanArgField(booleanArg: true)
          }
        }
      `)
		})

		It("Good string value", func() {
			expectValid(`
        {
          complicatedArgs {
            stringArgField(stringArg: "foo")
          }
        }
      `)
		})

		It("Good float value", func() {
			expectValid(`
        {
          complicatedArgs {
            floatArgField(floatArg: 1.1)
          }
        }
      `)
		})

		It("Good negative float value", func() {
			expectValid(`
        {
          complicatedArgs {
            floatArgField(floatArg: -1.1)
          }
        }
      `)
		})

		It("Int into Float", func() {
			expectValid(`
        {
          complicatedArgs {
            floatArgField(floatArg: 1)
          }
        }
      `)
		})

		It("Int into ID", func() {
			expectValid(`
        {
          complicatedArgs {
            idArgField(idArg: 1)
          }
        }
      `)
		})

		It("String into ID", func() {
			expectValid(`
        {
          complicatedArgs {
            idArgField(idArg: "someIdString")
          }
        }
      `)
		})

		It("Good enum value", func() {
			expectValid(`
        {
          dog {
            doesKnowCommand(dogCommand: SIT)
          }
        }
      `)
		})

		It("Enum with undefined value", func() {
			expectValid(`
        {
          complicatedArgs {
            enumArgField(enumArg: UNKNOWN)
          }
        }
      `)
		})

		It("Enum with null value", func() {
			expectValid(`
        {
          complicatedArgs {
            enumArgField(enumArg: NO_FUR)
          }
        }
      `)
		})

		It("null into nullable type", func() {
			expectValid(`
        {
          complicatedArgs {
            intArgField(intArg: null)
          }
        }
      `)

			expectValid(`
        {
          dog(a: null, b: null, c:{ requiredField: true, intField: null }) {
            name
          }
        }
      `)
		})
	})

	Describe("Invalid String values", func() {
		It("Int into String", func() {
			expectErrors(`
        {
          complicatedArgs {
            stringArgField(stringArg: 1)
          }
        }
      `).Should(Equal(graphql.ErrorsOf(badValue("String", "1", nil, 4, 39))))
		})

		It("Float into String", func() {
			expectErrors(`
        {
          complicatedArgs {
            stringArgField(stringArg: 1.0)
          }
        }
      `).Should(Equal(graphql.ErrorsOf(badValue("String", "1.0", nil, 4, 39))))
		})

		It("Boolean into String", func() {
			expectErrors(`
        {
          complicatedArgs {
            stringArgField(stringArg: true)
          }
        }
      `).Should(Equal(graphql.ErrorsOf(badValue("String", "true", nil, 4, 39))))
		})

		It("Unquoted String into String", func() {
			expectErrors(`
        {
          complicatedArgs {
            stringArgField(stringArg: BAR)
          }
        }
      `).Should(Equal(graphql.ErrorsOf(badValue("String", "BAR", nil, 4, 39))))
		})
	})

	Describe("Invalid Int values", func() {
		It("String into Int", func() {
			expectErrors(`
        {
          complicatedArgs {
            intArgField(intArg: "3")
          }
        }
      `).Should(Equal(graphql.ErrorsOf(badValue("Int", `"3"`, nil, 4, 33))))
		})

		It("Big Int into Int", func() {
			expectErrors(`
        {
          complicatedArgs {
            intArgField(intArg: 829384293849283498239482938)
          }
        }
      `).Should(Equal(graphql.ErrorsOf(badValue("Int", "829384293849283498239482938", nil, 4, 33))))
		})

		It("Unquoted String into Int", func() {
			expectErrors(`
        {
          complicatedArgs {
            intArgField(intArg: FOO)
          }
        }
      `).Should(Equal(graphql.ErrorsOf(badValue("Int", "FOO", nil, 4, 33))))
		})

		It("Simple Float into Int", func() {
			expectErrors(`
        {
          complicatedArgs {
            intArgField(intArg: 3.0)
          }
        }
      `).Should(Equal(graphql.ErrorsOf(badValue("Int", "3.0", nil, 4, 33))))
		})

		It("Float into Int", func() {
			expectErrors(`
        {
          complicatedArgs {
            intArgField(intArg: 3.333)
          }
        }
      `).Should(Equal(graphql.ErrorsOf(badValue("Int", "3.333", nil, 4, 33))))
		})
	})

	Describe("Invalid Float values", func() {
		It("String into Float", func() {
			expectErrors(`
        {
          complicatedArgs {
            floatArgField(floatArg: "3.333")
          }
        }
      `).Should(Equal(graphql.ErrorsOf(badValue("Float", `"3.333"`, nil, 4, 37))))
		})

		It("Boolean into Float", func() {
			expectErrors(`
        {
          complicatedArgs {
            floatArgField(floatArg: true)
          }
        }
      `).Should(Equal(graphql.ErrorsOf(badValue("Float", "true", nil, 4, 37))))
		})

		It("Unquoted into Float", func() {
			expectErrors(`
        {
          complicatedArgs {
            floatArgField(floatArg: FOO)
          }
        }
      `).Should(Equal(graphql.ErrorsOf(badValue("Float", "FOO", nil, 4, 37))))
		})
	})

	Describe("Invalid Boolean value", func() {
		It("Int into Boolean", func() {
			expectErrors(`
        {
          complicatedArgs {
            booleanArgField(booleanArg: 2)
          }
        }
      `).Should(Equal(graphql.ErrorsOf(badValue("Boolean", "2", nil, 4, 41))))
		})

		It("Float into Boolean", func() {
			expectErrors(`
        {
          complicatedArgs {
            booleanArgField(booleanArg: 1.0)
          }
        }
      `).Should(Equal(graphql.ErrorsOf(badValue("Boolean", "1.0", nil, 4, 41))))
		})

		It("String into Boolean", func() {
			expectErrors(`
        {
          complicatedArgs {
            booleanArgField(booleanArg: "true")
          }
        }
      `).Should(Equal(graphql.ErrorsOf(badValue("Boolean", `"true"`, nil, 4, 41))))
		})

		It("Unquoted into Boolean", func() {
			expectErrors(`
        {
          complicatedArgs {
            booleanArgField(booleanArg: TRUE)
          }
        }
      `).Should(Equal(graphql.ErrorsOf(badValue("Boolean", "TRUE", nil, 4, 41))))
		})
	})

	Describe("Invalid ID value", func() {
		It("Float into ID", func() {
			expectErrors(`
        {
          complicatedArgs {
            idArgField(idArg: 1.0)
          }
        }
      `).Should(Equal(graphql.ErrorsOf(badValue("ID", "1.0", nil, 4, 31))))
		})

		It("Boolean into ID", func() {
			expectErrors(`
        {
          complicatedArgs {
            idArgField(idArg: true)
          }
        }
      `).Should(Equal(graphql.ErrorsOf(badValue("ID", "true", nil, 4, 31))))
		})

		It("Unquoted into ID", func() {
			expectErrors(`
        {
          complicatedArgs {
            idArgField(idArg: SOMETHING)
          }
        }
      `).Should(Equal(graphql.ErrorsOf(badValue("ID", "SOMETHING", nil, 4, 31))))
		})
	})

	Describe("Invalid Enum value", func() {
		It("Int into Enum", func() {
			expectErrors(`
        {
          dog {
            doesKnowCommand(dogCommand: 2)
          }
        }
      `).Should(Equal(graphql.ErrorsOf(badValue("DogCommand", "2", nil, 4, 41))))
		})

		It("Float into Enum", func() {
			expectErrors(`
        {
          dog {
            doesKnowCommand(dogCommand: 1.0)
          }
        }
      `).Should(Equal(graphql.ErrorsOf(badValue("DogCommand", "1.0", nil, 4, 41))))
		})

		It("String into Enum", func() {
			expectErrors(`
        {
          dog {
            doesKnowCommand(dogCommand: "SIT")
          }
        }
      `).Should(Equal(graphql.ErrorsOf(
				badValue(
					"DogCommand",
					`"SIT"`,
					[]string{"SIT"},
					4,
					41,
				),
			)))
		})

		It("Boolean into Enum", func() {
			expectErrors(`
        {
          dog {
            doesKnowCommand(dogCommand: true)
          }
        }
      `).Should(Equal(graphql.ErrorsOf(badValue("DogCommand", "true", nil, 4, 41))))
		})

		It("Unknown Enum Value into Enum", func() {
			expectErrors(`
        {
          dog {
            doesKnowCommand(dogCommand: JUGGLE)
          }
        }
      `).Should(Equal(graphql.ErrorsOf(badValue("DogCommand", "JUGGLE", nil, 4, 41))))
		})

		It("Different case Enum Value into Enum", func() {
			expectErrors(`
        {
          dog {
            doesKnowCommand(dogCommand: sit)
          }
        }
      `).Should(Equal(graphql.ErrorsOf(
				badValue(
					"DogCommand",
					"sit",
					[]string{"SIT"},
					4,
					41,
				),
			)))
		})
	})

	Describe("Valid List value", func() {
		It("Good list value", func() {
			expectValid(`
        {
          complicatedArgs {
            stringListArgField(stringListArg: ["one", null, "two"])
          }
        }
      `)
		})

		It("Empty list value", func() {
			expectValid(`
        {
          complicatedArgs {
            stringListArgField(stringListArg: [])
          }
        }
      `)
		})

		It("Null value", func() {
			expectValid(`
        {
          complicatedArgs {
            stringListArgField(stringListArg: null)
          }
        }
      `)
		})

		It("Single value into List", func() {
			expectValid(`
        {
          complicatedArgs {
            stringListArgField(stringListArg: "one")
          }
        }
      `)
		})
	})

	Describe("Invalid List value", func() {
		It("Incorrect item type", func() {
			expectErrors(`
        {
          complicatedArgs {
            stringListArgField(stringListArg: ["one", 2])
          }
        }
      `).Should(Equal(graphql.ErrorsOf(badValue("String", "2", nil, 4, 55))))
		})

		It("Single value of incorrect type", func() {
			expectErrors(`
        {
          complicatedArgs {
            stringListArgField(stringListArg: 1)
          }
        }
      `).Should(Equal(graphql.ErrorsOf(badValue("[String]", "1", nil, 4, 47))))
		})
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
		It("Incorrect value type", func() {
			expectErrors(`
        {
          complicatedArgs {
            multipleReqs(req2: "two", req1: "one")
          }
        }
      `).Should(Equal(graphql.ErrorsOf(
				badValue("Int!", `"two"`, nil, 4, 32),
				badValue("Int!", `"one"`, nil, 4, 45),
			)))
		})

		It("Incorrect value and missing argument (ProvidedRequiredArguments)", func() {
			expectErrors(`
        {
          complicatedArgs {
            multipleReqs(req1: "one")
          }
        }
      `).Should(Equal(graphql.ErrorsOf(badValue("Int!", `"one"`, nil, 4, 32))))
		})

		It("Null value", func() {
			expectErrors(`
        {
          complicatedArgs {
            multipleReqs(req1: null)
          }
        }
      `).Should(Equal(graphql.ErrorsOf(badValue("Int!", "null", nil, 4, 32))))
		})
	})

	Describe("Valid input object value", func() {
		It("Optional arg, despite required field in type", func() {
			expectValid(`
        {
          complicatedArgs {
            complexArgField
          }
        }
      `)
		})

		It("Partial object, only required", func() {
			expectValid(`
        {
          complicatedArgs {
            complexArgField(complexArg: { requiredField: true })
          }
        }
      `)
		})

		It("Partial object, required field can be falsey", func() {
			expectValid(`
        {
          complicatedArgs {
            complexArgField(complexArg: { requiredField: false })
          }
        }
      `)
		})

		It("Partial object, including required", func() {
			expectValid(`
        {
          complicatedArgs {
            complexArgField(complexArg: { requiredField: true, intField: 4 })
          }
        }
      `)
		})

		It("Full object", func() {
			expectValid(`
        {
          complicatedArgs {
            complexArgField(complexArg: {
              requiredField: true,
              intField: 4,
              stringField: "foo",
              booleanField: false,
              stringListField: ["one", "two"]
            })
          }
        }
      `)
		})

		It("Full object with fields in different order", func() {
			expectValid(`
        {
          complicatedArgs {
            complexArgField(complexArg: {
              stringListField: ["one", "two"],
              booleanField: false,
              requiredField: true,
              stringField: "foo",
              intField: 4,
            })
          }
        }
      `)
		})
	})

	Describe("Invalid input object value", func() {
		It("Partial object, missing required", func() {
			expectErrors(`
        {
          complicatedArgs {
            complexArgField(complexArg: { intField: 4 })
          }
        }
      `).Should(Equal(graphql.ErrorsOf(
				requiredField("ComplexInput", "requiredField", "Boolean!", 4, 41),
			)))
		})

		It("Partial object, invalid field type", func() {
			expectErrors(`
        {
          complicatedArgs {
            complexArgField(complexArg: {
              stringListField: ["one", 2],
              requiredField: true,
            })
          }
        }
      `).Should(Equal(graphql.ErrorsOf(badValue("String", "2", nil, 5, 40))))
		})

		It("Partial object, null to non-null field", func() {
			expectErrors(`
        {
          complicatedArgs {
            complexArgField(complexArg: {
              requiredField: true,
              nonNullField: null,
            })
          }
        }
      `).Should(Equal(graphql.ErrorsOf(badValue("Boolean!", "null", nil, 6, 29))))
		})

		It("Partial object, unknown field arg", func() {
			suggestedPermutations := [][]string{
				{"nonNullField", "intField", "booleanField"},
				{"nonNullField", "booleanField", "intField"},
				{"intField", "nonNullField", "booleanField"},
				{"intField", "booleanField", "nonNullField"},
				{"booleanField", "intField", "nonNullField"},
				{"booleanField", "nonNullField", "intField"},
			}

			matchers := make([]types.GomegaMatcher, len(suggestedPermutations))
			for i, suggestedFields := range suggestedPermutations {
				matchers[i] = Equal(graphql.ErrorsOf(
					unknownField(
						"ComplexInput",
						"unknownField",
						suggestedFields,
						6,
						15,
					),
				))
			}

			expectErrors(`
        {
          complicatedArgs {
            complexArgField(complexArg: {
              requiredField: true,
              unknownField: "value"
            })
          }
        }
      `).Should(Or(matchers...))
		})

		It("reports original error for custom scalar which throws", func() {
			expectedErrors := expectErrors(`
        {
          invalidArg(arg: 123)
        }
      `)

			expectedErrors.Should(testutil.ConsistOfGraphQLErrors(
				testutil.MatchGraphQLError(
					testutil.MessageEqual("Expected type Invalid, found 123; Invalid scalar is always invalid: 123"),
					testutil.LocationEqual(graphql.ErrorLocation{
						Line:   3,
						Column: 27,
					}),
					testutil.OriginalErrorMatch("Invalid scalar is always invalid: 123"),
				),
			))
		})

		It("allows custom scalar to accept complex literals", func() {
			expectValid(`
        {
          test1: anyArg(arg: 123)
          test2: anyArg(arg: "abc")
          test3: anyArg(arg: [123, "abc"])
          test4: anyArg(arg: {deep: [123, "abc"]})
        }
      `)
		})
	})

	Describe("Directive arguments", func() {
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

		It("with directive with incorrect types", func() {
			expectErrors(`
        {
          dog @include(if: "yes") {
            name @skip(if: ENUM)
          }
        }
      `).Should(Equal(graphql.ErrorsOf(
				badValue("Boolean!", `"yes"`, nil, 3, 28),
				badValue("Boolean!", "ENUM", nil, 4, 28),
			)))
		})
	})

	Describe("Variable default values", func() {
		It("variables with valid default values", func() {
			expectValid(`
        query WithDefaultValues(
          $a: Int = 1,
          $b: String = "ok",
          $c: ComplexInput = { requiredField: true, intField: 3 }
          $d: Int! = 123
        ) {
          dog { name }
        }
      `)
		})

		It("variables with valid default null values", func() {
			expectValid(`
        query WithDefaultValues(
          $a: Int = null,
          $b: String = null,
          $c: ComplexInput = { requiredField: true, intField: null }
        ) {
          dog { name }
        }
      `)
		})

		It("variables with invalid default null values", func() {
			expectErrors(`
        query WithDefaultValues(
          $a: Int! = null,
          $b: String! = null,
          $c: ComplexInput = { requiredField: null, intField: null }
        ) {
          dog { name }
        }
      `).Should(Equal(graphql.ErrorsOf(
				badValue("Int!", "null", nil, 3, 22),
				badValue("String!", "null", nil, 4, 25),
				badValue("Boolean!", "null", nil, 5, 47),
			)))
		})

		It("variables with invalid default values", func() {
			expectErrors(`
        query InvalidDefaultValues(
          $a: Int = "one",
          $b: String = 4,
          $c: ComplexInput = "notverycomplex"
        ) {
          dog { name }
        }
      `).Should(Equal(graphql.ErrorsOf(
				badValue("Int", `"one"`, nil, 3, 21),
				badValue("String", "4", nil, 4, 24),
				badValue("ComplexInput", `"notverycomplex"`, nil, 5, 30),
			)))
		})

		It("variables with complex invalid default values", func() {
			expectErrors(`
        query WithDefaultValues(
          $a: ComplexInput = { requiredField: 123, intField: "abc" }
        ) {
          dog { name }
        }
      `).Should(Equal(graphql.ErrorsOf(
				badValue("Boolean!", "123", nil, 3, 47),
				badValue("Int", `"abc"`, nil, 3, 62),
			)))
		})

		It("complex variables missing required field", func() {
			expectErrors(`
        query MissingRequiredField($a: ComplexInput = {intField: 3}) {
          dog { name }
        }
      `).Should(Equal(graphql.ErrorsOf(
				requiredField("ComplexInput", "requiredField", "Boolean!", 2, 55),
			)))
		})

		It("list variables with invalid item", func() {
			expectErrors(`
        query InvalidItem($a: [String] = ["one", 2]) {
          dog { name }
        }
      `).Should(Equal(graphql.ErrorsOf(badValue("String", "2", nil, 2, 50))))
		})
	})
})
