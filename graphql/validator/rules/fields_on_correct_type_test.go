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

// graphql-js/src/validation/__tests__/FieldsOnCorrectType-test.js@8c96dc8
var _ = Describe("Validate: Fields on correct type", func() {
	expectErrors := func(queryStr string) GomegaAssertion {
		return expectValidationErrors(rules.FieldsOnCorrectType{}, queryStr)
	}

	expectValid := func(queryStr string) {
		expectErrors(queryStr).Should(Equal(graphql.NoErrors()))
	}

	undefinedField := func(
		fieldName string,
		typeName string,
		suggestedTypes []string,
		suggestedFields []string,
		line uint,
		column uint) error {

		return graphql.NewError(
			validator.UndefinedFieldMessage(fieldName, typeName, suggestedTypes, suggestedFields),
			[]graphql.ErrorLocation{
				{Line: line, Column: column},
			},
		)
	}

	It("Object field selection", func() {
		expectValid(`
      fragment objectFieldSelection on Dog {
        __typename
        name
      }
    `)
	})

	It("Aliased object field selection", func() {
		expectValid(`
      fragment aliasedObjectFieldSelection on Dog {
        tn : __typename
        otherName : name
      }
    `)
	})

	It("Interface field selection", func() {
		expectValid(`
      fragment interfaceFieldSelection on Pet {
        __typename
        name
      }
    `)
	})

	It("Aliased interface field selection", func() {
		expectValid(`
      fragment interfaceFieldSelection on Pet {
        otherName : name
      }
    `)
	})

	It("Lying alias selection", func() {
		expectValid(`
      fragment lyingAliasSelection on Dog {
        name : nickname
      }
    `)
	})

	It("Ignores fields on unknown type", func() {
		expectValid(`
      fragment unknownSelection on UnknownType {
        unknownField
      }
    `)
	})

	It("reports errors when type is known again", func() {
		expectErrors(`
      fragment typeKnownAgain on Pet {
        unknown_pet_field {
          ... on Cat {
            unknown_cat_field
          }
        }
      }
    `).Should(Equal(graphql.ErrorsOf(
			undefinedField("unknown_pet_field", "Pet", nil, nil, 3, 9),
			undefinedField("unknown_cat_field", "Cat", nil, nil, 5, 13),
		)))
	})

	It("Field not defined on fragment", func() {
		expectErrors(`
      fragment fieldNotDefined on Dog {
        meowVolume
      }
    `).Should(Equal(graphql.ErrorsOf(
			undefinedField("meowVolume", "Dog", nil, []string{"barkVolume"}, 3, 9),
		)))
	})

	It("Ignores deeply unknown field", func() {
		expectErrors(`
      fragment deepFieldNotDefined on Dog {
        unknown_field {
          deeper_unknown_field
        }
      }
    `).Should(Equal(graphql.ErrorsOf(undefinedField("unknown_field", "Dog", nil, nil, 3, 9))))
	})

	It("Sub-field not defined", func() {
		expectErrors(`
      fragment subFieldNotDefined on Human {
        pets {
          unknown_field
        }
      }
    `).Should(Equal(graphql.ErrorsOf(undefinedField("unknown_field", "Pet", nil, nil, 4, 11))))
	})

	It("Field not defined on inline fragment", func() {
		expectErrors(`
      fragment fieldNotDefined on Pet {
        ... on Dog {
          meowVolume
        }
      }
    `).Should(Equal(graphql.ErrorsOf(
			undefinedField("meowVolume", "Dog", nil, []string{"barkVolume"}, 4, 11),
		)))
	})

	It("Aliased field target not defined", func() {
		expectErrors(`
      fragment aliasedFieldTargetNotDefined on Dog {
        volume : mooVolume
      }
    `).Should(Equal(graphql.ErrorsOf(
			undefinedField("mooVolume", "Dog", nil, []string{"barkVolume"}, 3, 9),
		)))
	})

	It("Aliased lying field target not defined", func() {
		expectErrors(`
      fragment aliasedLyingFieldTargetNotDefined on Dog {
        barkVolume : kawVolume
      }
    `).Should(Equal(graphql.ErrorsOf(
			undefinedField("kawVolume", "Dog", nil, []string{"barkVolume"}, 3, 9),
		)))
	})

	It("Not defined on interface", func() {
		expectErrors(`
      fragment notDefinedOnInterface on Pet {
        tailLength
      }
    `).Should(Equal(graphql.ErrorsOf(undefinedField("tailLength", "Pet", nil, nil, 3, 9))))
	})

	It("Defined on implementors but not on interface", func() {
		expectErrors(`
      fragment definedOnImplementorsButNotInterface on Pet {
        nickname
      }
    `).Should(Or(
			Equal(graphql.ErrorsOf(
				undefinedField("nickname", "Pet", []string{"Dog", "Cat"}, []string{"name"}, 3, 9),
			)),
			Equal(graphql.ErrorsOf(
				undefinedField("nickname", "Pet", []string{"Cat", "Dog"}, []string{"name"}, 3, 9),
			)),
		))
	})

	It("Meta field selection on union", func() {
		expectValid(`
      fragment directFieldSelectionOnUnion on CatOrDog {
        __typename
      }
    `)
	})

	It("Direct field selection on union", func() {
		expectErrors(`
      fragment directFieldSelectionOnUnion on CatOrDog {
        directField
      }
    `).Should(Equal(graphql.ErrorsOf(undefinedField("directField", "CatOrDog", nil, nil, 3, 9))))
	})

	It("Defined on implementors queried on union", func() {
		expectErrors(`
      fragment definedOnImplementorsQueriedOnUnion on CatOrDog {
        name
      }
    `).Should(Or(
			Equal(graphql.ErrorsOf(
				undefinedField(
					"name",
					"CatOrDog",
					[]string{"Being", "Pet", "Canine", "Dog", "Cat"},
					nil,
					3,
					9,
				),
			)),

			Equal(graphql.ErrorsOf(
				undefinedField(
					"name",
					"CatOrDog",
					[]string{"Being", "Pet", "Canine", "Cat", "Dog"},
					nil,
					3,
					9,
				),
			)),
		))
	})

	It("valid field in inline fragment", func() {
		expectValid(`
      fragment objectFieldSelection on Pet {
        ... on Dog {
          name
        }
        ... {
          name
        }
      }
    `)
	})

	// Also checks graphql/internal/validator/messages_test.go when syncing with
	// FieldsOnCorrectType-test.js from graphql-js.
})
