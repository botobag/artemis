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
	"context"
	"fmt"

	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/graphql/executor"
	"github.com/botobag/artemis/graphql/parser"
	"github.com/botobag/artemis/graphql/token"
	"github.com/botobag/artemis/internal/testutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func executeQueryWithParams(schema *graphql.Schema, query string, params map[string]interface{}) executor.ExecutionResult {
	document, err := parser.Parse(token.NewSource(&token.SourceConfig{
		Body: token.SourceBody([]byte(query)),
	}), parser.ParseOptions{})
	Expect(err).ShouldNot(HaveOccurred())

	operation, errs := executor.Prepare(executor.PrepareParams{
		Schema:   schema,
		Document: document,
	})
	Expect(errs.HaveOccurred()).ShouldNot(BeTrue())

	return operation.Execute(context.Background(), executor.ExecuteParams{
		VariableValues: params,
	})
}

func executeQuery(schema *graphql.Schema, query string) executor.ExecutionResult {
	return executeQueryWithParams(schema, query, nil)
}

var _ = Describe("Enum", func() {

	// graphql-js/src/type/__tests__/definition-test.js
	It("defines an enum type with deprecated value", func() {
		enumTypeWithDeprecatedValue, err := graphql.NewEnum(&graphql.EnumConfig{
			Name: "EnumWithDeprecatedValue",
			Values: graphql.EnumValueDefinitionMap{
				"foo": graphql.EnumValueDefinition{
					Deprecation: &graphql.Deprecation{
						Reason: "Just because",
					},
				},
			},
		})

		Expect(err).ShouldNot(HaveOccurred())
		Expect(enumTypeWithDeprecatedValue).ShouldNot(BeNil())

		enumValues := enumTypeWithDeprecatedValue.Values()
		Expect(len(enumValues)).Should(Equal(1))

		enumValue := enumValues[0]
		Expect(enumValue.Name()).Should(Equal("foo"))
		Expect(enumValue.Description()).Should(BeEmpty())
		Expect(enumValue.IsDeprecated()).Should(BeTrue())
		Expect(enumValue.Deprecation()).ShouldNot(BeNil())
		Expect(enumValue.Deprecation().Reason).Should(Equal("Just because"))
		Expect(enumValue.Value()).Should(Equal("foo"))
	})

	It("defines an enum type with a value of `null`", func() {
		enumTypeWithNullishValue, err := graphql.NewEnum(&graphql.EnumConfig{
			Name: "EnumTypeWithNullishValue",
			Values: graphql.EnumValueDefinitionMap{
				"NULL": graphql.EnumValueDefinition{
					Value: graphql.NilEnumInternalValue,
				},
			},
		})

		Expect(err).ShouldNot(HaveOccurred())
		Expect(enumTypeWithNullishValue).ShouldNot(BeNil())

		enumValues := enumTypeWithNullishValue.Values()
		Expect(len(enumValues)).Should(Equal(1))

		enumValue := enumValues[0]
		Expect(enumValue.Name()).Should(Equal("NULL"))
		Expect(enumValue.Description()).Should(BeEmpty())
		Expect(enumValue.IsDeprecated()).Should(BeFalse())
		Expect(enumValue.Deprecation()).Should(BeNil())
		Expect(enumValue.Value()).Should(BeNil())
	})

	// graphql-js/src/type/__tests__/enumType-test.js
	Describe("Type System: Enum Values", func() {
		var (
			schema           *graphql.Schema
			colorType        *graphql.Enum
			queryType        *graphql.Object
			mutationType     *graphql.Object
			subscriptionType *graphql.Object

			complexEnum *graphql.Enum
		)

		complex1 := &struct {
			someRandomFunction func()
		}{someRandomFunction: func() {}}

		complex2 := &struct {
			someRandomValue int
		}{someRandomValue: 123}

		BeforeEach(func() {
			var err error

			colorType, err = graphql.NewEnum(&graphql.EnumConfig{
				Name: "Color",
				Values: graphql.EnumValueDefinitionMap{
					"RED": graphql.EnumValueDefinition{
						Value: 0,
					},
					"GREEN": graphql.EnumValueDefinition{
						Value: 1,
					},
					"BLUE": graphql.EnumValueDefinition{
						Value: 2,
					},
				},
				ResultCoercerFactory: graphql.DefaultEnumResultCoercerFactory(graphql.DefaultEnumResultCoercerLookupByValue),
			})
			Expect(err).ShouldNot(HaveOccurred())

			complexEnum, err = graphql.NewEnum(&graphql.EnumConfig{
				Name: "Complex",
				Values: graphql.EnumValueDefinitionMap{
					"ONE": {
						Value: complex1,
					},
					"TWO": {
						Value: complex2,
					},
				},
				ResultCoercerFactory: graphql.DefaultEnumResultCoercerFactory(graphql.DefaultEnumResultCoercerLookupByValue),
			})
			Expect(err).ShouldNot(HaveOccurred())

			queryType, err = graphql.NewObject(&graphql.ObjectConfig{
				Name: "Query",
				Fields: graphql.Fields{
					"colorEnum": {
						Type: graphql.T(colorType),
						Args: graphql.ArgumentConfigMap{
							"fromEnum": {
								Type: graphql.T(colorType),
							},
							"fromInt": {
								Type: graphql.T(graphql.Int()),
							},
							"fromString": {
								Type: graphql.T(graphql.String()),
							},
						},
						Resolver: graphql.FieldResolverFunc(func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
							args := info.ArgumentValues()
							if fromInt, ok := args.Lookup("fromInt"); ok {
								return fromInt, nil
							}
							if fromString, ok := args.Lookup("fromString"); ok {
								return fromString, nil
							}
							if fromEnum, ok := args.Lookup("fromEnum"); ok {
								return fromEnum, nil
							}
							return nil, nil
						}),
					},
					"colorInt": {
						Type: graphql.T(graphql.Int()),
						Args: graphql.ArgumentConfigMap{
							"fromEnum": {
								Type: graphql.T(colorType),
							},
							"fromInt": {
								Type: graphql.T(graphql.Int()),
							},
						},
						Resolver: graphql.FieldResolverFunc(func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
							args := info.ArgumentValues()
							if fromInt, ok := args.Lookup("fromInt"); ok {
								return fromInt, nil
							}
							if fromEnum, ok := args.Lookup("fromEnum"); ok {
								return fromEnum, nil
							}
							return nil, nil
						}),
					},
					"complexEnum": {
						Type: graphql.T(complexEnum),
						Args: graphql.ArgumentConfigMap{
							"fromEnum": {
								Type: graphql.T(complexEnum),
								// Note: defaultValue is provided an *internal* representation for Enums, rather
								// than the string name.
								DefaultValue: complex1,
							},
							"provideGoodValue": {
								Type: graphql.T(graphql.Boolean()),
							},
							"provideBadValue": {
								Type: graphql.T(graphql.Boolean()),
							},
						},
						Resolver: graphql.FieldResolverFunc(func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
							args := info.ArgumentValues()
							if provideGoodValue, ok := args.Lookup("provideGoodValue"); ok && provideGoodValue.(bool) {
								// Note: this is one of the references of the internal values which complexEnum
								// allows.
								return complex2, nil
							}

							if provideBadValue, ok := args.Lookup("provideBadValue"); ok && provideBadValue.(bool) {
								// Note: similar shape, but not the same *reference* as Complex2 above. Enum internal
								// values require deep equality.
								return &struct {
									someRandomValue int
								}{someRandomValue: 123}, nil
							}

							if fromEnum, ok := args.Lookup("fromEnum"); ok {
								return fromEnum, nil
							}

							return nil, nil
						}),
					},
				},
			})
			Expect(err).ShouldNot(HaveOccurred())

			mutationType, err = graphql.NewObject(&graphql.ObjectConfig{
				Name: "Mutation",
				Fields: graphql.Fields{
					"favoriteEnum": {
						Type: graphql.T(colorType),
						Args: graphql.ArgumentConfigMap{
							"color": {
								Type: graphql.T(colorType),
							},
						},
						Resolver: graphql.FieldResolverFunc(func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
							args := info.ArgumentValues()
							if color, ok := args.Lookup("color"); ok {
								return color, nil
							}
							return nil, nil
						}),
					},
				},
			})
			Expect(err).ShouldNot(HaveOccurred())

			subscriptionType, err = graphql.NewObject(&graphql.ObjectConfig{
				Name: "Subscription",
				Fields: graphql.Fields{
					"subscribeToEnum": {
						Type: graphql.T(colorType),
						Args: graphql.ArgumentConfigMap{
							"color": {
								Type: graphql.T(colorType),
							},
						},
						Resolver: graphql.FieldResolverFunc(func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
							args := info.ArgumentValues()
							if color, ok := args.Lookup("color"); ok {
								return color, nil
							}
							return nil, nil
						}),
					},
				},
			})
			Expect(err).ShouldNot(HaveOccurred())

			schema, err = graphql.NewSchema(&graphql.SchemaConfig{
				Query:        queryType,
				Mutation:     mutationType,
				Subscription: subscriptionType,
			})
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("accepts enum literals as input", func() {
			query := "{ colorInt(fromEnum: GREEN) }"
			Expect(executeQuery(schema, query)).Should(testutil.SerializeToJSONAs(map[string]interface{}{
				"data": map[string]interface{}{
					"colorInt": 1,
				},
			}))
		})

		It("enum may be output type", func() {
			query := "{ colorEnum(fromInt: 1) }"
			Expect(executeQuery(schema, query)).Should(testutil.SerializeToJSONAs(map[string]interface{}{
				"data": map[string]interface{}{
					"colorEnum": "GREEN",
				},
			}))
		})

		It("enum may be both input and output type", func() {
			query := "{ colorEnum(fromEnum: GREEN) }"
			Expect(executeQuery(schema, query)).Should(testutil.SerializeToJSONAs(map[string]interface{}{
				"data": map[string]interface{}{
					"colorEnum": "GREEN",
				},
			}))
		})

		It("does not accept string literals", func() {
			query := `{ colorEnum(fromEnum: "GREEN") }`
			Expect(executeQuery(schema, query)).Should(testutil.SerializeToJSONAs(map[string]interface{}{
				"errors": []interface{}{
					map[string]interface{}{
						"message": `Argument "fromEnum" has invalid value "GREEN".`,
						"locations": []interface{}{
							map[string]interface{}{
								"line":   1,
								"column": 23,
							},
						},
					},
				},
			}))
		})

		It("does not accept values not in the enum", func() {
			query := `{ colorEnum(fromEnum: GREENISH) }`
			Expect(executeQuery(schema, query)).Should(testutil.SerializeToJSONAs(map[string]interface{}{
				"errors": []interface{}{
					map[string]interface{}{
						"message": `Argument "fromEnum" has invalid value "GREENISH".`,
						"locations": []interface{}{
							map[string]interface{}{
								"line":   1,
								"column": 23,
							},
						},
					},
				},
			}))
		})

		It("does not accept values with incorrect casing", func() {
			query := `{ colorEnum(fromEnum: green) }`
			Expect(executeQuery(schema, query)).Should(testutil.SerializeToJSONAs(map[string]interface{}{
				"errors": []interface{}{
					map[string]interface{}{
						"message": `Argument "fromEnum" has invalid value "green".`,
						"locations": []interface{}{
							map[string]interface{}{
								"line":   1,
								"column": 23,
							},
						},
					},
				},
			}))
		})

		It("does not accept incorrect internal value", func() {
			query := `{ colorEnum(fromString: "GREEN") }`
			Expect(executeQuery(schema, query)).Should(testutil.SerializeToJSONAs(map[string]interface{}{
				"data": map[string]interface{}{
					"colorEnum": nil,
				},
				"errors": []interface{}{
					map[string]interface{}{
						"message": `Expected a value of type "Color" but received: GREEN`,
						"locations": []interface{}{
							map[string]interface{}{
								"line":   1,
								"column": 3,
							},
						},
						"path": []interface{}{"colorEnum"},
					},
				},
			}))
		})

		It("does not accept internal value in place of enum literal", func() {
			query := `{ colorEnum(fromEnum: 1) }`
			Expect(executeQuery(schema, query)).Should(testutil.SerializeToJSONAs(map[string]interface{}{
				"errors": []interface{}{
					map[string]interface{}{
						"message": `Argument "fromEnum" has invalid value "1".`,
						"locations": []interface{}{
							map[string]interface{}{
								"line":   1,
								"column": 23,
							},
						},
					},
				},
			}))
		})

		It("does not accept internal value in place of int", func() {
			query := `{ colorEnum(fromInt: GREEN) }`
			Expect(executeQuery(schema, query)).Should(testutil.SerializeToJSONAs(map[string]interface{}{
				"errors": []interface{}{
					map[string]interface{}{
						"message": `Argument "fromInt" has invalid value "GREEN".`,
						"locations": []interface{}{
							map[string]interface{}{
								"line":   1,
								"column": 22,
							},
						},
					},
				},
			}))
		})

		It("accepts JSON string as enum variable", func() {
			query := "query ($color: Color!) { colorEnum(fromEnum: $color) }"
			params := map[string]interface{}{
				"color": "BLUE",
			}
			Expect(executeQueryWithParams(schema, query, params)).Should(testutil.SerializeToJSONAs(map[string]interface{}{
				"data": map[string]interface{}{
					"colorEnum": "BLUE",
				},
			}))
		})

		It("accepts enum literals as input arguments to mutations", func() {
			query := `mutation ($color: Color!) { favoriteEnum(color: $color) }`
			params := map[string]interface{}{
				"color": "GREEN",
			}
			Expect(executeQueryWithParams(schema, query, params)).Should(testutil.SerializeToJSONAs(map[string]interface{}{
				"data": map[string]interface{}{
					"favoriteEnum": "GREEN",
				},
			}))
		})

		It("accepts enum literals as input arguments to subscriptions", func() {
			query := `subscription ($color: Color!) { subscribeToEnum(color: $color) }`
			params := map[string]interface{}{
				"color": "GREEN",
			}
			Expect(executeQueryWithParams(schema, query, params)).Should(testutil.SerializeToJSONAs(map[string]interface{}{
				"data": map[string]interface{}{
					"subscribeToEnum": "GREEN",
				},
			}))
		})

		It("does not accept internal value as enum variable", func() {
			query := `query ($color: Color!) { colorEnum(fromEnum: $color) }`
			params := map[string]interface{}{
				"color": 2,
			}
			Expect(executeQueryWithParams(schema, query, params)).Should(testutil.SerializeToJSONAs(map[string]interface{}{
				"errors": []interface{}{
					map[string]interface{}{
						"message": "Variable \"$color\" got invalid value 2.",
						"locations": []interface{}{
							map[string]interface{}{
								"line":   1,
								"column": 8,
							},
						},
					},
				},
			}))
		})

		It("does not accept string variables as enum input", func() {
			query := `query ($color: String!) { colorEnum(fromEnum: $color) }`
			params := map[string]interface{}{
				"color": "BLUE",
			}
			Expect(executeQueryWithParams(schema, query, params)).Should(testutil.SerializeToJSONAs(map[string]interface{}{
				// FIXME: This should be a validation error.
				// "errors": []interface{}{
				// 	"message": `Variable "$color" of type "String!" used in position expecting type "Color".`,
				// 	"locations": []interface{}{
				// 		map[string]interface{}{
				// 			"line":   1,
				// 			"column": 8,
				// 		},
				// 		map[string]interface{}{
				// 			"line":   1,
				// 			"column": 47,
				// 		},
				// 	},
				// },
				"data": map[string]interface{}{
					"colorEnum": nil,
				},
				"errors": []interface{}{
					map[string]interface{}{
						"message": `Expected a value of type "Color" but received: BLUE`,
						"locations": []interface{}{
							map[string]interface{}{
								"line":   1,
								"column": 27,
							},
						},
						"path": []interface{}{"colorEnum"},
					},
				},
			}))
		})

		It("does not accept internal value variable as enum input", func() {
			query := `query ($color: Int!) { colorEnum(fromEnum: $color) }`
			params := map[string]interface{}{
				"color": 2,
			}
			Expect(executeQueryWithParams(schema, query, params)).Should(testutil.SerializeToJSONAs(map[string]interface{}{
				// FIXME: This should be a validation error.
				// "errors": []interface{}{
				// 	map[string]interface{}{
				// 		"message": `Variable "$color" of type "Int!" used in position expecting type "Color".`,
				// 		"locations": []interface{}{
				// 			map[string]interface{}{
				// 				"line":   1,
				// 				"column": 8,
				// 			},
				// 			map[string]interface{}{
				// 				"line":   1,
				// 				"column": 47,
				// 			},
				// 		},
				// 	},
				// },
				"data": map[string]interface{}{
					"colorEnum": "BLUE",
				},
			}))
		})

		It("enum value may have an internal value of 0", func() {
			query := `{
        colorEnum(fromEnum: RED)
        colorInt(fromEnum: RED)
      }`

			Expect(executeQuery(schema, query)).Should(testutil.SerializeToJSONAs(map[string]interface{}{
				"data": map[string]interface{}{
					"colorEnum": "RED",
					"colorInt":  0,
				},
			}))
		})

		It("enum inputs may be nullable", func() {
			query := `{
        colorEnum
        colorInt
      }`

			Expect(executeQuery(schema, query)).Should(testutil.SerializeToJSONAs(map[string]interface{}{
				"data": map[string]interface{}{
					"colorEnum": nil,
					"colorInt":  nil,
				},
			}))
		})

		It("presents a Values() API for complex enums", func() {
			values := complexEnum.Values()
			Expect(len(values)).Should(Equal(2))

			// Note that the order of enum values in the list is unspecified.
			for _, value := range values {
				Expect(value.Name()).Should(Or(Equal("ONE"), Equal("TWO")))

				if value.Name() == "ONE" {
					Expect(value.Value()).Should(Equal(complex1))
				} else {
					Expect(value.Value()).Should(Equal(complex2))
				}
			}
		})

		It("presents a Value() API for complex enums", func() {
			oneValue := complexEnum.Value("ONE")
			Expect(oneValue).ShouldNot(BeNil())
			Expect(oneValue.Name()).Should(Equal("ONE"))
			Expect(oneValue.Value()).Should(Equal(complex1))
		})

		It("may be internally represented with complex values", func() {
			query := `
			{
        first: complexEnum
        second: complexEnum(fromEnum: TWO)
        good: complexEnum(provideGoodValue: true)
        bad: complexEnum(provideBadValue: true)
			}`
			Expect(executeQuery(schema, query)).Should(testutil.SerializeToJSONAs(map[string]interface{}{
				"data": map[string]interface{}{
					"first":  "ONE",
					"second": "TWO",
					"good":   "TWO",
					"bad":    nil,
				},
				"errors": []interface{}{
					map[string]interface{}{
						"message": `Expected a value of type "Complex" but received: &{123}`,
						"locations": []interface{}{
							map[string]interface{}{
								"line":   6,
								"column": 9,
							},
						},
						"path": []interface{}{"bad"},
					},
				},
			}))
		})

		It("can be introspected without error", func() {
			// TODO: #72
		})

		Context("where enum value may be pointer", func() {
			var (
				colorType *graphql.Enum
				queryType *graphql.Object
				schema    *graphql.Schema
			)

			BeforeEach(func() {
				var err error

				// Same as the colorType created in the upper context but with
				// DefaultEnumResultCoercerLookupByValueDeref strategy.
				colorType, err = graphql.NewEnum(&graphql.EnumConfig{
					Name: "Color",
					Values: graphql.EnumValueDefinitionMap{
						"RED": graphql.EnumValueDefinition{
							Value: 0,
						},
						"GREEN": graphql.EnumValueDefinition{
							Value: 1,
						},
						"BLUE": graphql.EnumValueDefinition{
							Value: 2,
						},
					},
					ResultCoercerFactory: graphql.DefaultEnumResultCoercerFactory(graphql.DefaultEnumResultCoercerLookupByValueDeref),
				})
				Expect(err).ShouldNot(HaveOccurred())

				queryType, err = graphql.NewObject(&graphql.ObjectConfig{
					Name: "Query",
					Fields: graphql.Fields{
						"query": {
							Type: &graphql.ObjectConfig{
								Name: "query",
								Fields: graphql.Fields{
									"color": {
										Type: graphql.T(colorType),
									},
									"foo": {
										Description: "foo field",
										Type:        graphql.T(graphql.Int()),
									},
								},
							},
							Args: graphql.ArgumentConfigMap{
								"provideNilValue": {
									Type:         graphql.T(graphql.Boolean()),
									DefaultValue: false,
								},
							},
							Resolver: graphql.FieldResolverFunc(func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
								args := info.ArgumentValues()
								var result struct {
									Color *int
									Foo   *int
								}

								if provideNilValue, ok := args.Lookup("provideNilValue"); !ok || !provideNilValue.(bool) {
									one := 1
									result.Color = &one
									result.Foo = &one
								}

								return result, nil
							}),
						},
					},
				})
				Expect(err).ShouldNot(HaveOccurred())

				schema, err = graphql.NewSchema(&graphql.SchemaConfig{
					Query: queryType,
				})
				Expect(err).ShouldNot(HaveOccurred())
			})

			It("resolves enum value with dereferenced value from result pointer", func() {
				query := "{ query { color foo } }"
				Expect(executeQuery(schema, query)).Should(testutil.SerializeToJSONAs(map[string]interface{}{
					"data": map[string]interface{}{
						"query": map[string]interface{}{
							"color": "GREEN",
							"foo":   1,
						},
					},
				}))
			})

			It("accepts nil value in result pointer", func() {
				query := "{ query(provideNilValue: true) { color foo } }"
				Expect(executeQuery(schema, query)).Should(testutil.SerializeToJSONAs(map[string]interface{}{
					"data": map[string]interface{}{
						"query": map[string]interface{}{
							"color": nil,
							"foo":   nil,
						},
					},
				}))
			})
		})
	})

	It("stringifies to type name", func() {
		enumType, err := graphql.NewEnum(&graphql.EnumConfig{
			Name: "Enum",
		})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(fmt.Sprintf("%s", enumType)).Should(Equal("Enum"))
		Expect(fmt.Sprintf("%v", enumType)).Should(Equal("Enum"))
	})

	It("rejects creating type without name", func() {
		_, err := graphql.NewEnum(&graphql.EnumConfig{
			Name: "",
		})
		Expect(err).Should(MatchError("Must provide name for Enum."))

		Expect(func() {
			graphql.MustNewEnum(&graphql.EnumConfig{})
		}).Should(Panic())
	})
})
