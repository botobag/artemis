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

var _ = Describe("Object", func() {
	var InterfaceType graphql.Interface

	BeforeEach(func() {
		InterfaceType = graphql.MustNewInterface(&graphql.InterfaceConfig{
			Name: "Interface",
		})
	})

	// graphql-js/src/type/__tests__/definition-test.js
	It("defines an object type with deprecated field", func() {
		TypeWithDeprecatedField := graphql.MustNewObject(&graphql.ObjectConfig{
			Name: "foo",
			Fields: graphql.Fields{
				"bar": graphql.FieldConfig{
					Type: graphql.T(graphql.String()),
					Deprecation: &graphql.Deprecation{
						Reason: "A terrible reason",
					},
				},
			},
		})

		bar := TypeWithDeprecatedField.Fields()["bar"]
		Expect(bar).ShouldNot(BeNil())
		Expect(bar.Type()).Should(Equal(graphql.String()))
		Expect(bar.Deprecation()).Should(Equal(&graphql.Deprecation{
			Reason: "A terrible reason",
		}))
		Expect(bar.Name()).Should(Equal("bar"))
		Expect(bar.Args()).Should(BeEmpty())
	})

	Describe("interfaces must be array", func() {
		It("accepts an Object type with array interfaces", func() {
			objType := graphql.MustNewObject(&graphql.ObjectConfig{
				Name: "SomeObject",
				Interfaces: []graphql.InterfaceTypeDefinition{
					graphql.I(InterfaceType),
				},
				Fields: graphql.Fields{
					"f": graphql.FieldConfig{
						Type: graphql.T(graphql.String()),
					},
				},
			})

			Expect(objType.Interfaces()).Should(Equal([]graphql.Interface{InterfaceType}))
		})

		It("accepts empty interfaces", func() {
			objType := graphql.MustNewObject(&graphql.ObjectConfig{
				Name: "SomeObjectWithoutInterfaces",
			})
			Expect(objType.Interfaces()).Should(BeEmpty())

			objType = graphql.MustNewObject(&graphql.ObjectConfig{
				Name:       "SomeObjectWithEmptyInterfacesSet",
				Interfaces: []graphql.InterfaceTypeDefinition{},
			})
			Expect(objType.Interfaces()).Should(BeEmpty())
		})
	})

	It("does not mutate passed field definitions", func() {
		fields := graphql.Fields{
			"field1": graphql.FieldConfig{
				Type: graphql.T(graphql.String()),
			},
			"field2": graphql.FieldConfig{
				Type: graphql.T(graphql.String()),
				Args: graphql.ArgumentConfigMap{
					"id": graphql.ArgumentConfig{
						Type: graphql.T(graphql.String()),
					},
				},
			},
		}

		testObject1 := graphql.MustNewObject(&graphql.ObjectConfig{
			Name:   "Test1",
			Fields: fields,
		})

		testObject2 := graphql.MustNewObject(&graphql.ObjectConfig{
			Name:   "Test2",
			Fields: fields,
		})

		Expect(testObject1.Fields()).Should(Equal(testObject2.Fields()))
		Expect(fields).Should(Equal(graphql.Fields{
			"field1": graphql.FieldConfig{
				Type: graphql.T(graphql.String()),
			},
			"field2": graphql.FieldConfig{
				Type: graphql.T(graphql.String()),
				Args: graphql.ArgumentConfigMap{
					"id": graphql.ArgumentConfig{
						Type: graphql.T(graphql.String()),
					},
				},
			},
		}))
	})

	It("rejects creating type without name", func() {
		_, err := graphql.NewObject(&graphql.ObjectConfig{
			Name: "",
		})
		Expect(err).Should(MatchError("Must provide name for Object."))

		Expect(func() {
			graphql.MustNewObject(&graphql.ObjectConfig{})
		}).Should(Panic())
	})

	It("accepts creating type without fields", func() {
		objectType := graphql.MustNewObject(&graphql.ObjectConfig{
			Name: "ObjectWithoutFields1",
		})
		Expect(objectType.Fields()).Should(BeEmpty())

		objectType = graphql.MustNewObject(&graphql.ObjectConfig{
			Name: "ObjectWithoutFields2",
		})
		Expect(objectType.Fields()).Should(BeEmpty())
	})

	Describe("having fields", func() {
		It("defines argument with nil default value", func() {
			object := graphql.MustNewObject(&graphql.ObjectConfig{
				Name: "Test",
				Fields: graphql.Fields{
					"field": graphql.FieldConfig{
						Type: graphql.T(graphql.String()),
						Args: graphql.ArgumentConfigMap{
							"id": graphql.ArgumentConfig{
								Type:         graphql.T(graphql.ID()),
								DefaultValue: graphql.NilArgumentDefaultValue,
							},
						},
					},
				},
			})

			Expect(len(object.Fields())).Should(Equal(1))
			field := object.Fields()["field"]
			Expect(field).ShouldNot(BeNil())
			Expect(field.Name()).Should(Equal("field"))
			Expect(field.Type()).Should(Equal(graphql.String()))

			Expect(len(field.Args())).Should(Equal(1))
			arg := &field.Args()[0]
			Expect(arg.Name()).Should(Equal("id"))
			Expect(arg.Description()).Should(Equal(""))
			Expect(arg.Type()).Should(Equal(graphql.ID()))
			Expect(arg.HasDefaultValue()).Should(BeTrue())
			Expect(arg.DefaultValue()).Should(BeNil())
		})

		It("defines argument without default value", func() {
			object := graphql.MustNewObject(&graphql.ObjectConfig{
				Name: "Test",
				Fields: graphql.Fields{
					"field": graphql.FieldConfig{
						Type: graphql.T(graphql.String()),
						Args: graphql.ArgumentConfigMap{
							"id": graphql.ArgumentConfig{
								Type: graphql.T(graphql.ID()),
							},
						},
					},
				},
			})

			Expect(len(object.Fields())).Should(Equal(1))
			field := object.Fields()["field"]
			Expect(field).ShouldNot(BeNil())
			Expect(field.Name()).Should(Equal("field"))
			Expect(field.Type()).Should(Equal(graphql.String()))

			Expect(len(field.Args())).Should(Equal(1))
			arg := &field.Args()[0]
			Expect(arg.Name()).Should(Equal("id"))
			Expect(arg.Description()).Should(Equal(""))
			Expect(arg.Type()).Should(Equal(graphql.ID()))
			Expect(arg.HasDefaultValue()).Should(BeFalse())
			Expect(arg.DefaultValue()).Should(BeNil())
		})
	})
})
