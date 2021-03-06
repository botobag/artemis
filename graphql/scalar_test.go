/**
 * Copyright (c) 2018, The Artemis Authors.
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

package graphql_test

import (
	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/graphql/ast"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func schemaWithFieldType(ttype graphql.Type) graphql.Schema {
	return graphql.MustNewSchema(&graphql.SchemaConfig{
		Query: graphql.MustNewObject(&graphql.ObjectConfig{
			Name: "Query",
			Fields: graphql.Fields{
				"field": {
					Type: graphql.T(ttype),
				},
			},
		}),
	})
}

var _ = Describe("Scalar", func() {

	// graphql-js/src/type/__tests__/definition-test.js
	Describe("Type System: Scalar types must be serializable", func() {
		It("accepts a Scalar type defining serialize", func() {
			scalar := graphql.MustNewScalar(&graphql.ScalarConfig{
				Name: "SomeScalar",
				ResultCoercer: graphql.ScalarResultCoercerFunc(func(value interface{}) (interface{}, error) {
					return nil, nil
				}),
			})

			Expect(schemaWithFieldType(scalar)).ShouldNot(BeNil())
		})

		It("rejects a Scalar type not defining serializer for result", func() {
			_, err := graphql.NewScalar(&graphql.ScalarConfig{
				Name: "SomeScalar",
			})

			Expect(err).Should(MatchError(
				`SomeScalar must provide ResultCoercer. If this custom Scalar ` +
					`is also used as an input type, ensure InputCoercer is also provided.`))
		})

		It("accepts a Scalar type defining input parser", func() {
			scalar := graphql.MustNewScalar(&graphql.ScalarConfig{
				Name: "SomeScalar",
				ResultCoercer: graphql.ScalarResultCoercerFunc(func(value interface{}) (interface{}, error) {
					return nil, nil
				}),
				InputCoercer: graphql.ScalarInputCoercerFuncs{
					CoerceVariableValueFunc: func(interface{}) (interface{}, error) {
						return nil, nil
					},
					CoerceLiteralValueFunc: func(ast.Value) (interface{}, error) {
						return nil, nil
					},
				},
			})

			Expect(schemaWithFieldType(scalar)).ShouldNot(BeNil())
		})
	})

	It("rejects creating type without name", func() {
		_, err := graphql.NewScalar(&graphql.ScalarConfig{
			Name: "",
		})
		Expect(err).Should(MatchError("Must provide name for Scalar."))

		Expect(func() {
			graphql.MustNewScalar(&graphql.ScalarConfig{})
		}).Should(Panic())
	})
})
