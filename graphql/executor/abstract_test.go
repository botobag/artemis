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

package executor_test

import (
	"context"

	"github.com/botobag/artemis/concurrent"
	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/graphql/executor"
	"github.com/botobag/artemis/graphql/parser"
	"github.com/botobag/artemis/graphql/token"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// graphql-js/src/execution/__tests__/abstract-test.js
var _ = DescribeExecute("Execute: Handles execution of abstract types", func(runner concurrent.Executor) {
	type Dog struct {
		Name  string
		Woofs bool
	}

	type Cat struct {
		Name  string
		Meows bool
	}

	type Human struct {
		Name string
	}

	It("resolveType on Interface yields useful error", func() {
		petType := graphql.InterfaceConfig{
			Name: "Pet",
		}

		dogType := graphql.ObjectConfig{
			Name:       "Dog",
			Interfaces: []graphql.InterfaceTypeDefinition{&petType},
			Fields: graphql.Fields{
				"name": {
					Type: graphql.T(graphql.String()),
				},
				"woofs": {
					Type: graphql.T(graphql.Boolean()),
				},
			},
		}

		catType := graphql.ObjectConfig{
			Name:       "Cat",
			Interfaces: []graphql.InterfaceTypeDefinition{&petType},
			Fields: graphql.Fields{
				"name": {
					Type: graphql.T(graphql.String()),
				},
				"meows": {
					Type: graphql.T(graphql.Boolean()),
				},
			},
		}

		humanType := graphql.ObjectConfig{
			Name: "Human",
			Fields: graphql.Fields{
				"name": {
					Type: graphql.T(graphql.String()),
				},
			},
		}

		petType.TypeResolver = graphql.TypeResolverFunc(func(ctx context.Context, value interface{}, info graphql.ResolveInfo) (graphql.Object, error) {
			switch value.(type) {
			case *Dog:
				return graphql.NewObject(&dogType)
			case *Cat:
				return graphql.NewObject(&catType)
			case *Human:
				return graphql.NewObject(&humanType)
			default:
				return nil, nil
			}
		})

		queryType, err := graphql.NewObject(&graphql.ObjectConfig{
			Name: "Query",
			Fields: graphql.Fields{
				"pets": {
					Type: graphql.ListOf(&petType),
					Resolver: graphql.FieldResolverFunc(func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
						return []interface{}{
							&Dog{"Odie", true},
							&Cat{"Garfield", false},
							&Human{"Jon"},
						}, nil
					}),
				},
			},
		})
		Expect(err).ShouldNot(HaveOccurred())

		schema, err := graphql.NewSchema(&graphql.SchemaConfig{
			Query: queryType,
			Types: []graphql.Type{
				graphql.MustNewObject(&dogType),
				graphql.MustNewObject(&catType),
			},
		})
		Expect(err).ShouldNot(HaveOccurred())

		document, err := parser.Parse(token.NewSource(&token.SourceConfig{
			Body: token.SourceBody([]byte(`{
      pets {
        name
        ... on Dog {
          woofs
        }
        ... on Cat {
          meows
        }
      }
    }`))}), parser.ParseOptions{})
		Expect(err).ShouldNot(HaveOccurred())

		operation, errs := executor.Prepare(executor.PrepareParams{
			Schema:   schema,
			Document: document,
		})
		Expect(errs.HaveOccurred()).ShouldNot(BeTrue())

		result := operation.Execute(context.Background(), executor.ExecuteParams{
			Runner: runner,
		})

		Eventually(result).Should(MatchResultInJSON(`{
			"data": {
				"pets": [
					{
						"name": "Odie",
						"woofs": true
					},
					{
						"name": "Garfield",
						"meows": false
					},
					null
				]
			},
			"errors": [
				{
					"message": "Runtime Object type \"Human\" is not a possible type for \"Pet\".",
					"locations": [{ "line": 2, "column": 7 }],
					"path": ["pets", 2]
				}
			]
		}`))
	})

	It("resolveType on Union yields useful error", func() {
		dogType := graphql.ObjectConfig{
			Name: "Dog",
			Fields: graphql.Fields{
				"name": {
					Type: graphql.T(graphql.String()),
				},
				"woofs": {
					Type: graphql.T(graphql.Boolean()),
				},
			},
		}

		catType := graphql.ObjectConfig{
			Name: "Cat",
			Fields: graphql.Fields{
				"name": {
					Type: graphql.T(graphql.String()),
				},
				"meows": {
					Type: graphql.T(graphql.Boolean()),
				},
			},
		}

		humanType := graphql.ObjectConfig{
			Name: "Human",
			Fields: graphql.Fields{
				"name": {
					Type: graphql.T(graphql.String()),
				},
			},
		}

		petType := graphql.UnionConfig{
			Name: "Pet",
			TypeResolver: graphql.TypeResolverFunc(func(ctx context.Context, value interface{}, info graphql.ResolveInfo) (graphql.Object, error) {
				switch value.(type) {
				case *Dog:
					return graphql.NewObject(&dogType)
				case *Cat:
					return graphql.NewObject(&catType)
				case *Human:
					return graphql.NewObject(&humanType)
				default:
					return nil, nil
				}
			}),
			PossibleTypes: []graphql.ObjectTypeDefinition{
				&dogType,
				&catType,
			},
		}

		queryType, err := graphql.NewObject(&graphql.ObjectConfig{
			Name: "Query",
			Fields: graphql.Fields{
				"pets": {
					Type: graphql.ListOf(&petType),
					Resolver: graphql.FieldResolverFunc(func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
						return []interface{}{
							&Dog{"Odie", true},
							&Cat{"Garfield", false},
							&Human{"Jon"},
						}, nil
					}),
				},
			},
		})
		Expect(err).ShouldNot(HaveOccurred())

		schema, err := graphql.NewSchema(&graphql.SchemaConfig{
			Query: queryType,
		})
		Expect(err).ShouldNot(HaveOccurred())

		document, err := parser.Parse(token.NewSource(&token.SourceConfig{
			Body: token.SourceBody([]byte(`{
      pets {
        ... on Dog {
          name
          woofs
        }
        ... on Cat {
          name
          meows
        }
      }
    }`))}), parser.ParseOptions{})
		Expect(err).ShouldNot(HaveOccurred())

		operation, errs := executor.Prepare(executor.PrepareParams{
			Schema:   schema,
			Document: document,
		})
		Expect(errs.HaveOccurred()).ShouldNot(BeTrue())

		result := operation.Execute(context.Background(), executor.ExecuteParams{
			Runner: runner,
		})

		Eventually(result).Should(MatchResultInJSON(`{
			"data": {
				"pets": [
					{
						"name": "Odie",
						"woofs": true
					},
					{
						"name": "Garfield",
						"meows": false
					},
					null
				]
			},
			"errors": [
				{
					"message": "Runtime Object type \"Human\" is not a possible type for \"Pet\".",
					"locations": [{ "line": 2, "column": 7 }],
					"path": ["pets", 2]
				}
			]
		}`))
	})

	It("returning invalid value from resolveType yields useful error", func() {
		fooInterface := &graphql.InterfaceConfig{
			Name: "FooInterface",
			Fields: graphql.Fields{
				"bar": {
					Type: graphql.T(graphql.String()),
				},
			},
			TypeResolver: graphql.TypeResolverFunc(func(ctx context.Context, value interface{}, info graphql.ResolveInfo) (graphql.Object, error) {
				return nil, nil
			}),
		}

		fooObject := &graphql.ObjectConfig{
			Name:       "FooObject",
			Interfaces: []graphql.InterfaceTypeDefinition{fooInterface},
			Fields: graphql.Fields{
				"bar": {
					Type: graphql.T(graphql.String()),
				},
			},
		}

		queryType, err := graphql.NewObject(&graphql.ObjectConfig{
			Name: "Query",
			Fields: graphql.Fields{
				"foo": {
					Type: fooInterface,
					Resolver: graphql.FieldResolverFunc(func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
						return "dummy", nil
					}),
				},
			},
		})
		Expect(err).ShouldNot(HaveOccurred())

		schema, err := graphql.NewSchema(&graphql.SchemaConfig{
			Query: queryType,
			Types: []graphql.Type{graphql.MustNewObject(fooObject)},
		})
		Expect(err).ShouldNot(HaveOccurred())

		document, err := parser.Parse(token.NewSource(&token.SourceConfig{
			Body: token.SourceBody([]byte("{ foo { bar } }"))}), parser.ParseOptions{})
		Expect(err).ShouldNot(HaveOccurred())

		operation, errs := executor.Prepare(executor.PrepareParams{
			Schema:   schema,
			Document: document,
		})
		Expect(errs.HaveOccurred()).ShouldNot(BeTrue())

		result := operation.Execute(context.Background(), executor.ExecuteParams{
			Runner: runner,
		})

		Eventually(result).Should(MatchResultInJSON(`{
			"data": {
				"foo": null
			},
			"errors": [
				{
					"message": "Abstract type FooInterface must resolve to an Object type at runtime for field Query.foo with value \"dummy\", received nil.",
					"locations": [{ "line": 1, "column": 3 }],
					"path": ["foo"]
				}
			]
		}`))
	})

	It("resolveType on Interface without type resolver yields useful error", func() {
		fooInterface := &graphql.InterfaceConfig{
			Name: "FooInterface",
			Fields: graphql.Fields{
				"bar": {
					Type: graphql.T(graphql.String()),
				},
			},
			/* TypeResolver: nil, */
		}

		queryType, err := graphql.NewObject(&graphql.ObjectConfig{
			Name: "Query",
			Fields: graphql.Fields{
				"foo": {
					Type: fooInterface,
					Resolver: graphql.FieldResolverFunc(func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
						return "dummy", nil
					}),
				},
			},
		})
		Expect(err).ShouldNot(HaveOccurred())

		schema, err := graphql.NewSchema(&graphql.SchemaConfig{
			Query: queryType,
		})
		Expect(err).ShouldNot(HaveOccurred())

		document, err := parser.Parse(token.NewSource(&token.SourceConfig{
			Body: token.SourceBody([]byte("{ foo { bar } }"))}), parser.ParseOptions{})
		Expect(err).ShouldNot(HaveOccurred())

		operation, errs := executor.Prepare(executor.PrepareParams{
			Schema:   schema,
			Document: document,
		})
		Expect(errs.HaveOccurred()).ShouldNot(BeTrue())

		result := operation.Execute(context.Background(), executor.ExecuteParams{
			Runner: runner,
		})

		Eventually(result).Should(MatchResultInJSON(`{
			"data": {
				"foo": null
			},
			"errors": [
				{
					"message": "Abstract type FooInterface must provide resolver to resolve to an Object type at runtime for field Query.foo with value \"dummy\"",
					"locations": [{ "line": 1, "column": 3 }],
					"path": ["foo"]
				}
			]
		}`))
	})

	It("resolveType on Union without type resolver yields useful error", func() {
		fooUnion := &graphql.UnionConfig{
			Name: "FooUnion",
			/* TypeResolver: nil, */
		}

		queryType, err := graphql.NewObject(&graphql.ObjectConfig{
			Name: "Query",
			Fields: graphql.Fields{
				"foo": {
					Type: fooUnion,
					Resolver: graphql.FieldResolverFunc(func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
						return "dummy", nil
					}),
				},
			},
		})
		Expect(err).ShouldNot(HaveOccurred())

		schema, err := graphql.NewSchema(&graphql.SchemaConfig{
			Query: queryType,
		})
		Expect(err).ShouldNot(HaveOccurred())

		document, err := parser.Parse(token.NewSource(&token.SourceConfig{
			Body: token.SourceBody([]byte("{ foo }"))}), parser.ParseOptions{})
		Expect(err).ShouldNot(HaveOccurred())

		operation, errs := executor.Prepare(executor.PrepareParams{
			Schema:   schema,
			Document: document,
		})
		Expect(errs.HaveOccurred()).ShouldNot(BeTrue())

		result := operation.Execute(context.Background(), executor.ExecuteParams{
			Runner: runner,
		})

		Eventually(result).Should(MatchResultInJSON(`{
			"data": {
				"foo": null
			},
			"errors": [
				{
					"message": "Abstract type FooUnion must provide resolver to resolve to an Object type at runtime for field Query.foo with value \"dummy\"",
					"locations": [{ "line": 1, "column": 3 }],
					"path": ["foo"]
				}
			]
		}`))
	})

	It("returns runtime type from resolve info when accessing fields in Interface", func() {
		fooInterface := &graphql.InterfaceConfig{
			Name: "FooInterface",
			Fields: graphql.Fields{
				"bar": {
					Type: graphql.T(graphql.String()),
				},
			},
		}

		fooObject := &graphql.ObjectConfig{
			Name:       "FooObject",
			Interfaces: []graphql.InterfaceTypeDefinition{fooInterface},
			Fields: graphql.Fields{
				"bar": {
					Type: graphql.T(graphql.String()),
					Resolver: graphql.FieldResolverFunc(func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
						Expect(info.Object().Name()).Should(Equal("FooObject"))
						return source, nil
					}),
				},
			},
		}

		fooInterface.TypeResolver = graphql.TypeResolverFunc(func(ctx context.Context, value interface{}, info graphql.ResolveInfo) (graphql.Object, error) {
			// Always resolve to FooObject.
			return graphql.NewObject(fooObject)
		})

		queryType, err := graphql.NewObject(&graphql.ObjectConfig{
			Name: "Query",
			Fields: graphql.Fields{
				"foo": {
					Type: fooInterface,
					Resolver: graphql.FieldResolverFunc(func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
						return "dummy", nil
					}),
				},
			},
		})
		Expect(err).ShouldNot(HaveOccurred())

		schema, err := graphql.NewSchema(&graphql.SchemaConfig{
			Query: queryType,
			Types: []graphql.Type{graphql.MustNewObject(fooObject)},
		})
		Expect(err).ShouldNot(HaveOccurred())

		document, err := parser.Parse(token.NewSource(&token.SourceConfig{
			Body: token.SourceBody([]byte("{ foo { bar } }"))}), parser.ParseOptions{})
		Expect(err).ShouldNot(HaveOccurred())

		operation, errs := executor.Prepare(executor.PrepareParams{
			Schema:   schema,
			Document: document,
		})
		Expect(errs.HaveOccurred()).ShouldNot(BeTrue())

		result := operation.Execute(context.Background(), executor.ExecuteParams{})
		Eventually(result).Should(MatchResultInJSON(`{
			"data": {
				"foo": {
					"bar": "dummy"
				}
			}
		}`))
	})

	It("returns runtime type from resolve info when accessing fields in Union", func() {
		fooObject := &graphql.ObjectConfig{
			Name: "FooObject",
			Fields: graphql.Fields{
				"bar": {
					Type: graphql.T(graphql.String()),
					Resolver: graphql.FieldResolverFunc(func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
						Expect(info.Object().Name()).Should(Equal("FooObject"))
						return source, nil
					}),
				},
			},
		}

		fooUnion := &graphql.UnionConfig{
			Name:          "FooUnion",
			PossibleTypes: []graphql.ObjectTypeDefinition{fooObject},
			TypeResolver: graphql.TypeResolverFunc(func(ctx context.Context, value interface{}, info graphql.ResolveInfo) (graphql.Object, error) {
				// Always resolve to FooObject.
				return graphql.NewObject(fooObject)
			}),
		}

		queryType, err := graphql.NewObject(&graphql.ObjectConfig{
			Name: "Query",
			Fields: graphql.Fields{
				"foo": {
					Type: fooUnion,
					Resolver: graphql.FieldResolverFunc(func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
						return "dummy", nil
					}),
				},
			},
		})
		Expect(err).ShouldNot(HaveOccurred())

		schema, err := graphql.NewSchema(&graphql.SchemaConfig{
			Query: queryType,
			Types: []graphql.Type{graphql.MustNewObject(fooObject)},
		})
		Expect(err).ShouldNot(HaveOccurred())

		query := `{
			foo {
				... on FooObject {
					bar
				}
			}
		}`

		document, err := parser.Parse(token.NewSource(&token.SourceConfig{
			Body: token.SourceBody([]byte(query))}), parser.ParseOptions{})
		Expect(err).ShouldNot(HaveOccurred())

		operation, errs := executor.Prepare(executor.PrepareParams{
			Schema:   schema,
			Document: document,
		})
		Expect(errs.HaveOccurred()).ShouldNot(BeTrue())

		result := operation.Execute(context.Background(), executor.ExecuteParams{})
		Eventually(result).Should(MatchResultInJSON(`{
			"data": {
				"foo": {
					"bar": "dummy"
				}
			}
		}`))
	})
})
