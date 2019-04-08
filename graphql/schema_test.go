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

var _ = Describe("Type System: Schema", func() {
	// graphql-js/src/type/__tests__/schema-test.js@2fcd55e
	Describe("Type Map", func() {
		It("includes interface possible types in the type map", func() {
			SomeInterface := graphql.MustNewInterface(&graphql.InterfaceConfig{
				Name:   "SomeInterface",
				Fields: graphql.Fields{},
			})

			SomeSubType := graphql.MustNewObject(&graphql.ObjectConfig{
				Name: "SomeSubType",
				Interfaces: []graphql.InterfaceTypeDefinition{
					graphql.I(SomeInterface),
				},
			})

			Schema := graphql.MustNewSchema(&graphql.SchemaConfig{
				Query: graphql.MustNewObject(&graphql.ObjectConfig{
					Name: "Query",
					Interfaces: []graphql.InterfaceTypeDefinition{
						graphql.I(SomeInterface),
					},
				}),
				Types: []graphql.Type{SomeSubType},
			})

			Expect(Schema.TypeMap().Lookup("SomeInterface")).Should(Equal(SomeInterface))
			Expect(Schema.TypeMap().Lookup("SomeSubType")).Should(Equal(SomeSubType))
		})

		It("includes nested input objects in the map", func() {
			NestedInputObject := graphql.MustNewInputObject(&graphql.InputObjectConfig{
				Name: "NestedInputObject",
			})

			SomeInputObject := graphql.MustNewInputObject(&graphql.InputObjectConfig{
				Name: "SomeInputObject",
				Fields: graphql.InputFields{
					"nested": {
						Type: graphql.T(NestedInputObject),
					},
				},
			})

			Schema := graphql.MustNewSchema(&graphql.SchemaConfig{
				Query: graphql.MustNewObject(&graphql.ObjectConfig{
					Name: "Query",
					Fields: graphql.Fields{
						"something": {
							Type: graphql.T(graphql.String()),
							Args: graphql.ArgumentConfigMap{
								"input": {
									Type: graphql.T(SomeInputObject),
								},
							},
						},
					},
				}),
			})

			Expect(Schema.TypeMap().Lookup("SomeInputObject")).Should(Equal(SomeInputObject))
			Expect(Schema.TypeMap().Lookup("NestedInputObject")).Should(Equal(NestedInputObject))
		})

		It("includes input types only used in directives", func() {
			directive := graphql.MustNewDirective(&graphql.DirectiveConfig{
				Name: "dir",
				Locations: []graphql.DirectiveLocation{
					graphql.DirectiveLocationObject,
				},
				Args: graphql.ArgumentConfigMap{
					"arg": {
						Type: &graphql.InputObjectConfig{
							Name: "Foo",
						},
					},
					"argList": {
						Type: &graphql.InputObjectConfig{
							Name: "Bar",
						},
					},
				},
			})

			schema := graphql.MustNewSchema(&graphql.SchemaConfig{
				Directives: []graphql.Directive{
					directive,
				},
			})

			Expect(schema.TypeMap().Lookup("Foo")).ShouldNot(BeNil())
			Expect(schema.TypeMap().Lookup("Bar")).ShouldNot(BeNil())
		})
	})

	Describe("A Schema must contain uniquely named types", func() {
		It("rejects a Schema which redefines a built-in type", func() {
			FakeString := graphql.MustNewScalar(&graphql.ScalarConfig{
				Name: "String",
				ResultCoercer: graphql.ScalarResultCoercerFunc(func(value interface{}) (interface{}, error) {
					return nil, nil
				}),
			})

			QueryType := graphql.MustNewObject(&graphql.ObjectConfig{
				Name: "Query",
				Fields: graphql.Fields{
					"normal": {
						Type: graphql.T(graphql.String()),
					},
					"fake": {
						Type: graphql.T(FakeString),
					},
				},
			})

			_, err := graphql.NewSchema(&graphql.SchemaConfig{
				Query: QueryType,
			})
			Expect(err).Should(MatchError(`Schema must contain unique named types but contains multiple types named "String".`))

			Expect(func() {
				graphql.MustNewSchema(&graphql.SchemaConfig{
					Query: QueryType,
				})
			}).Should(Panic())
		})

		It("rejects a Schema which defines an object type twice", func() {
			SameName1 := &graphql.ObjectConfig{
				Name: "SameName",
			}
			SameName2 := &graphql.ObjectConfig{
				Name: "SameName",
			}

			types := []graphql.Type{
				graphql.MustNewObject(SameName1),
				graphql.MustNewObject(SameName2),
			}

			_, err := graphql.NewSchema(&graphql.SchemaConfig{
				Types: types,
			})
			Expect(err).Should(MatchError(`Schema must contain unique named types but contains multiple types named "SameName".`))

			Expect(func() {
				graphql.MustNewSchema(&graphql.SchemaConfig{
					Types: types,
				})
			}).Should(Panic())
		})

		It("rejects a Schema which defines fields with conflicting types", func() {
			QueryType := graphql.MustNewObject(&graphql.ObjectConfig{
				Name: "Query",
				Fields: graphql.Fields{
					"a": {
						Type: &graphql.ObjectConfig{
							Name: "SameName",
						},
					},
					"b": {
						Type: &graphql.ObjectConfig{
							Name: "SameName",
						},
					},
				},
			})

			_, err := graphql.NewSchema(&graphql.SchemaConfig{
				Query: QueryType,
			})
			Expect(err).Should(MatchError(`Schema must contain unique named types but contains multiple types named "SameName".`))

			Expect(func() {
				graphql.MustNewSchema(&graphql.SchemaConfig{
					Query: QueryType,
				})
			}).Should(Panic())
		})
	})

	Describe("Standard Directives", func() {
		It("includes standard directives by default", func() {
			schema := graphql.MustNewSchema(&graphql.SchemaConfig{})
			for _, directive := range graphql.StandardDirectives() {
				Expect(schema.Directives()).Should(ContainElement(directive))
			}
		})

		Context("when ExcludeStandardDirectives is set", func() {
			It("does not include standard directives", func() {
				schema := graphql.MustNewSchema(&graphql.SchemaConfig{
					ExcludeStandardDirectives: true,
				})
				for _, directive := range graphql.StandardDirectives() {
					Expect(schema.Directives()).ShouldNot(ContainElement(directive))
				}
			})
		})
	})
})
