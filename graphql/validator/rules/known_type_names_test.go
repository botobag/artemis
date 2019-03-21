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

// graphql-js/src/validation/__tests__/KnownTypeNames-test.js@8c96dc8
var _ = Describe("Validate: Known type names", func() {
	expectErrors := func(queryStr string) GomegaAssertion {
		return expectValidationErrors(rules.KnownTypeNames{}, queryStr)
	}

	expectErrorsWithSchema := func(schema graphql.Schema, queryStr string) GomegaAssertion {
		return expectValidationErrorsWithSchema(schema, rules.KnownTypeNames{}, queryStr)
	}

	expectValid := func(queryStr string) {
		expectErrors(queryStr).Should(Equal(graphql.NoErrors()))
	}

	unknownType := func(
		typeName string,
		suggestedTypes []string,
		line uint,
		column uint) error {

		return graphql.NewError(
			validator.UnknownTypeMessage(typeName, suggestedTypes),
			[]graphql.ErrorLocation{
				{Line: line, Column: column},
			},
		)
	}

	It("known type names are valid", func() {
		expectValid(`
      query Foo($var: String, $required: [String!]!) {
        user(id: 4) {
          pets { ... on Pet { name }, ...PetFields, ... { name } }
        }
      }
      fragment PetFields on Pet {
        name
      }
    `)
	})

	It("unknown type names are invalid", func() {
		expectErrors(`
      query Foo($var: JumbledUpLetters) {
        user(id: 4) {
          name
          pets { ... on Badger { name }, ...PetFields }
        }
      }
      fragment PetFields on Peettt {
        name
      }
    `).Should(Equal(graphql.ErrorsOf(
			unknownType("JumbledUpLetters", nil, 2, 23),
			unknownType("Badger", nil, 5, 25),
			unknownType("Peettt", []string{"Pet"}, 8, 29),
		)))
	})

	It("references to standard scalars that are missing in schema", func() {
		schema := graphql.MustNewSchema(&graphql.SchemaConfig{
			Query: graphql.MustNewObject(&graphql.ObjectConfig{
				Name: "Query",
				Fields: graphql.Fields{
					"foo": {
						Type: graphql.T(graphql.String()),
					},
				},
			}),
		})

		const query = `
      query ($id: ID, $float: Float, $int: Int) {
        __typename
      }
    `
		expectErrorsWithSchema(schema, query).Should(Equal(graphql.ErrorsOf(
			unknownType("ID", nil, 2, 19),
			unknownType("Float", nil, 2, 31),
			unknownType("Int", nil, 2, 44),
		)))
	})
})
