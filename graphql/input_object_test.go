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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("InputObject", func() {
	// graphql-js/src/type/__tests__/definition-test.js
	It("does not mutate passed field definitions", func() {
		fields := graphql.InputFields{
			"field1": graphql.InputFieldDefinition{
				Type: graphql.T(graphql.String()),
			},
			"field2": graphql.InputFieldDefinition{
				Type: graphql.T(graphql.String()),
			},
		}

		testInputObject1 := graphql.MustNewInputObject(&graphql.InputObjectConfig{
			Name:   "Test1",
			Fields: fields,
		})

		testInputObject2 := graphql.MustNewInputObject(&graphql.InputObjectConfig{
			Name:   "Test2",
			Fields: fields,
		})

		Expect(testInputObject1.Fields()).Should(Equal(testInputObject2.Fields()))
		Expect(fields).Should(Equal(graphql.InputFields{
			"field1": graphql.InputFieldDefinition{
				Type: graphql.T(graphql.String()),
			},
			"field2": graphql.InputFieldDefinition{
				Type: graphql.T(graphql.String()),
			},
		}))
	})

	It("accepts creating type without fields", func() {
		Expect(graphql.MustNewInputObject(&graphql.InputObjectConfig{
			Name: "InputObjectWithoutFields1",
		}).Fields()).Should(BeEmpty())

		Expect(graphql.MustNewInputObject(&graphql.InputObjectConfig{
			Name: "InputObjectWithoutFields2",
		}).Fields()).Should(BeEmpty())
	})

	It("rejects creating type without a name", func() {
		_, err := graphql.NewInputObject(&graphql.InputObjectConfig{})
		Expect(err).Should(MatchError("Must provide name for InputObject."))

		Expect(func() {
			graphql.MustNewInputObject(&graphql.InputObjectConfig{})
		}).Should(Panic())
	})

	Describe("having fields", func() {
		It("sets default value to nil", func() {
			object := graphql.MustNewInputObject(&graphql.InputObjectConfig{
				Name: "Test",
				Fields: graphql.InputFields{
					"field": graphql.InputFieldDefinition{
						Type:         graphql.T(graphql.String()),
						DefaultValue: graphql.NilInputFieldDefaultValue,
					},
				},
			})

			Expect(len(object.Fields())).Should(Equal(1))
			field := object.Fields()["field"]
			Expect(field).ShouldNot(BeNil())
			Expect(field.Name()).Should(Equal("field"))
			Expect(field.Type()).Should(Equal(graphql.String()))
			Expect(field.HasDefaultValue()).Should(BeTrue())
			Expect(field.DefaultValue()).Should(BeNil())
		})

		It("defines without default value", func() {
			object := graphql.MustNewInputObject(&graphql.InputObjectConfig{
				Name: "Test",
				Fields: graphql.InputFields{
					"field": graphql.InputFieldDefinition{
						Type: graphql.T(graphql.String()),
					},
				},
			})

			Expect(len(object.Fields())).Should(Equal(1))
			field := object.Fields()["field"]
			Expect(field).ShouldNot(BeNil())
			Expect(field.Name()).Should(Equal("field"))
			Expect(field.Type()).Should(Equal(graphql.String()))
			Expect(field.HasDefaultValue()).Should(BeFalse())
			Expect(field.DefaultValue()).Should(BeNil())
		})
	})
})
