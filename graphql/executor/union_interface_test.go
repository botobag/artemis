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
	"encoding/json"
	"sort"

	"github.com/botobag/artemis/concurrent"
	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/graphql/executor"
	"github.com/botobag/artemis/graphql/parser"
	"github.com/botobag/artemis/graphql/token"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/matchers"
	"github.com/onsi/gomega/types"
)

// TODO: The followings are mostly replicated from introspection_test.go (only sorts "possibleTypes".)
type ByNameKey struct {
	data interface{}
}

func (s ByNameKey) Len() int {
	return len(s.data.([]interface{}))
}

func (s ByNameKey) Less(i, j int) bool {
	objects := s.data.([]interface{})
	o1 := objects[i].(map[string]interface{})
	o2 := objects[j].(map[string]interface{})
	return o1["name"].(string) < o2["name"].(string)
}

func (s ByNameKey) Swap(i, j int) {
	objects := s.data.([]interface{})
	objects[i], objects[j] = objects[j], objects[i]
}

func sortInspectionResult(resultJSON []byte) ([]byte, error) {
	var result struct {
		Data   map[string]interface{} `json:"data,omitempty"`
		Errors interface{}            `json:"errors,omitempty"`
	}
	if err := json.Unmarshal([]byte(resultJSON), &result); err != nil {
		return nil, err
	}

	if result.Data == nil {
		return resultJSON, nil
	}

	sortFieldByNameKey := func(t map[string]interface{}, field string) {
		v := t[field]
		if v != nil {
			sort.Sort(ByNameKey{v})
		}
	}

	sortType := func(t map[string]interface{}) {
		sortFieldByNameKey(t, "possibleTypes")
	}

	for _, field := range []string{
		"__type",
		// For the test that aliases __type to "Named" and "Pet".
		"Named",
		"Pet",
	} {
		data := result.Data[field]
		if data != nil {
			sortType(data.(map[string]interface{}))
		}
	}

	return json.Marshal(&result)
}

type introspectionResultMatcher struct {
	matchers.MatchJSONMatcher
	actual []byte
}

func (matcher *introspectionResultMatcher) Match(actual interface{}) (success bool, err error) {
	// Expect an executor.ExecutionResult.
	result := actual.(executor.ExecutionResult)

	// Encode to JSON.
	actualJSON, err := json.Marshal(result)
	if err != nil {
		return false, err
	}

	// Normalize the result for comparison with sort.
	actualJSON, err = sortInspectionResult(actualJSON)
	if err != nil {
		return false, err
	}

	// Cache actualJSON for error reporting.
	matcher.actual = actualJSON

	return matcher.MatchJSONMatcher.Match(actualJSON)
}

func (matcher *introspectionResultMatcher) FailureMessage(actual interface{}) (message string) {
	return matcher.MatchJSONMatcher.FailureMessage(matcher.actual)
}

func (matcher *introspectionResultMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return matcher.MatchJSONMatcher.NegatedFailureMessage(matcher.actual)
}

func MatchIntrospectionInJSON(json interface{}) types.GomegaMatcher {
	// Normalize the expected json. Expect the input json is a string.
	expectedJSON, err := sortInspectionResult([]byte(json.(string)))
	Expect(err).ShouldNot(HaveOccurred())

	return &introspectionResultMatcher{
		MatchJSONMatcher: matchers.MatchJSONMatcher{
			JSONToMatch: expectedJSON,
		},
	}
}

// graphql-js/src/execution/__tests__/union-interface-test.js@35cb32b
var _ = DescribeExecute("Execute: Union and intersection types", func(runner concurrent.Executor) {
	type Dog struct {
		Name  string
		Barks bool
	}

	type Cat struct {
		Name  string
		Meows bool
	}

	type Person struct {
		Name    string
		Pets    []interface{}
		Friends []interface{}
	}

	execute := func(schema graphql.Schema, query string, rootValue interface{}, appContext interface{}) <-chan executor.ExecutionResult {
		document, err := parser.Parse(token.NewSource(&token.SourceConfig{
			Body: token.SourceBody([]byte(query)),
		}))
		Expect(err).ShouldNot(HaveOccurred())

		operation, errs := executor.Prepare(executor.PrepareParams{
			Schema:   schema,
			Document: document,
		})
		Expect(errs.HaveOccurred()).ShouldNot(BeTrue())

		return operation.Execute(context.Background(), executor.ExecuteParams{
			Runner:     runner,
			RootValue:  rootValue,
			AppContext: appContext,
		})
	}

	var (
		schema graphql.Schema
		liz    *Person
		john   *Person
	)

	BeforeEach(func() {
		NamedType := &graphql.InterfaceConfig{
			Name: "Named",
			Fields: graphql.Fields{
				"name": {
					Type: graphql.T(graphql.String()),
				},
			},
		}

		DogType := &graphql.ObjectConfig{
			Name: "Dog",
			Interfaces: []graphql.InterfaceTypeDefinition{
				NamedType,
			},
			Fields: graphql.Fields{
				"name": {
					Type: graphql.T(graphql.String()),
				},
				"barks": {
					Type: graphql.T(graphql.Boolean()),
				},
			},
		}

		CatType := &graphql.ObjectConfig{
			Name: "Cat",
			Interfaces: []graphql.InterfaceTypeDefinition{
				NamedType,
			},
			Fields: graphql.Fields{
				"name": {
					Type: graphql.T(graphql.String()),
				},
				"meows": {
					Type: graphql.T(graphql.Boolean()),
				},
			},
		}

		PetType := &graphql.UnionConfig{
			Name: "Pet",
			PossibleTypes: []graphql.ObjectTypeDefinition{
				DogType,
				CatType,
			},
			TypeResolver: graphql.TypeResolverFunc(func(ctx context.Context, value interface{}, info graphql.ResolveInfo) (graphql.Object, error) {
				switch value.(type) {
				case *Dog:
					return graphql.NewObject(DogType)
				case *Cat:
					return graphql.NewObject(CatType)
				default:
					return nil, nil
				}
			}),
		}

		PersonType := &graphql.ObjectConfig{
			Name: "Person",
			Interfaces: []graphql.InterfaceTypeDefinition{
				NamedType,
			},
			Fields: graphql.Fields{
				"name": {
					Type: graphql.T(graphql.String()),
				},
				"pets": {
					Type: graphql.ListOf(PetType),
				},
				"friends": {
					Type: graphql.ListOf(NamedType),
				},
			},
		}

		NamedType.TypeResolver = graphql.TypeResolverFunc(func(ctx context.Context, value interface{}, info graphql.ResolveInfo) (graphql.Object, error) {
			switch value.(type) {
			case *Dog:
				return graphql.NewObject(DogType)
			case *Cat:
				return graphql.NewObject(CatType)
			case *Person:
				return graphql.NewObject(PersonType)
			default:
				return nil, nil
			}
		})

		schema = graphql.MustNewSchema(&graphql.SchemaConfig{
			Query: graphql.MustNewObject(PersonType),
			Types: []graphql.Type{
				graphql.MustNewUnion(PetType),
			},
		})

		garfield := &Cat{
			Name:  "Garfield",
			Meows: false,
		}

		odie := &Dog{
			Name:  "Odie",
			Barks: true,
		}

		liz = &Person{
			Name: "Liz",
		}

		john = &Person{
			Name:    "John",
			Pets:    []interface{}{garfield, odie},
			Friends: []interface{}{liz, odie},
		}
	})

	It("can introspect on union and intersection types", func() {
		query := `
      {
        Named: __type(name: "Named") {
          kind
          name
          fields { name }
          interfaces { name }
          possibleTypes { name }
          enumValues { name }
          inputFields { name }
        }
        Pet: __type(name: "Pet") {
          kind
          name
          fields { name }
          interfaces { name }
          possibleTypes { name }
          enumValues { name }
          inputFields { name }
        }
      }
    `

		Eventually(execute(schema, query, nil, nil)).Should(Receive(MatchIntrospectionInJSON(`{
      "data": {
        "Named": {
          "kind": "INTERFACE",
          "name": "Named",
          "fields": [
            {
              "name": "name"
            }
          ],
          "interfaces": null,
          "possibleTypes": [
            {
              "name": "Person"
            },
            {
              "name": "Dog"
            },
            {
              "name": "Cat"
            }
          ],
          "enumValues": null,
          "inputFields": null
        },
        "Pet": {
          "kind": "UNION",
          "name": "Pet",
          "fields": null,
          "interfaces": null,
          "possibleTypes": [
            {
              "name": "Dog"
            },
            {
              "name": "Cat"
            }
          ],
          "enumValues": null,
          "inputFields": null
        }
      }
    }`)))
	})

	It("executes using union types", func() {
		// NOTE: This is an *invalid* query, but it should be an *executable* query.
		query := `
      {
        __typename
        name
        pets {
          __typename
          name
          barks
          meows
        }
      }
    `

		Eventually(execute(schema, query, john, nil)).Should(Receive(MatchIntrospectionInJSON(`{
      "data": {
        "__typename": "Person",
        "name": "John",
        "pets": [
          {
            "__typename": "Cat",
            "name": "Garfield",
            "meows": false
          },
          {
            "__typename": "Dog",
            "name": "Odie",
            "barks": true
          }
        ]
      }
    }`)))
	})

	It("executes union types with inline fragments", func() {
		// NOTE: This is an *invalid* query, but it should be an *executable* query.
		query := `
      {
        __typename
        name
        pets {
          __typename
          ... on Dog {
            name
            barks
          }
          ... on Cat {
            name
            meows
          }
        }
      }
    `

		Eventually(execute(schema, query, john, nil)).Should(Receive(MatchIntrospectionInJSON(`{
      "data": {
        "__typename": "Person",
        "name": "John",
        "pets": [
          {
            "__typename": "Cat",
            "name": "Garfield",
            "meows": false
          },
          {
            "__typename": "Dog",
            "name": "Odie",
            "barks": true
          }
        ]
      }
    }`)))
	})

	It("executes using interface types", func() {
		query := `
      {
        __typename
        name
        friends {
          __typename
          name
          barks
          meows
        }
      }
    `

		Eventually(execute(schema, query, john, nil)).Should(Receive(MatchIntrospectionInJSON(`{
      "data": {
        "__typename": "Person",
        "name": "John",
        "friends": [
          {
            "__typename": "Person",
            "name": "Liz"
          },
          {
            "__typename": "Dog",
            "name": "Odie",
            "barks": true
          }
        ]
      }
    }`)))
	})

	It("executes interface types with inline fragments", func() {
		query := `
      {
        __typename
        name
        friends {
          __typename
          name
          ... on Dog {
            barks
          }
          ... on Cat {
            meows
          }
        }
      }
    `

		Eventually(execute(schema, query, john, nil)).Should(Receive(MatchIntrospectionInJSON(`{
      "data": {
        "__typename": "Person",
        "name": "John",
        "friends": [
          {
            "__typename": "Person",
            "name": "Liz"
          },
          {
            "__typename": "Dog",
            "name": "Odie",
            "barks": true
          }
        ]
      }
    }`)))
	})

	It("allows fragment conditions to be abstract types", func() {
		query := `
      {
        __typename
        name
        pets { ...PetFields }
        friends { ...FriendFields }
      }

      fragment PetFields on Pet {
        __typename
        ... on Dog {
          name
          barks
        }
        ... on Cat {
          name
          meows
        }
      }

      fragment FriendFields on Named {
        __typename
        name
        ... on Dog {
          barks
        }
        ... on Cat {
          meows
        }
      }
    `

		Eventually(execute(schema, query, john, nil)).Should(Receive(MatchIntrospectionInJSON(`{
      "data": {
        "__typename": "Person",
        "name": "John",
        "pets": [
          {
            "__typename": "Cat",
            "name": "Garfield",
            "meows": false
          },
          {
            "__typename": "Dog",
            "name": "Odie",
            "barks": true
          }
        ],
        "friends": [
          {
            "__typename": "Person",
            "name": "Liz"
          },
          {
            "__typename": "Dog",
            "name": "Odie",
            "barks": true
          }
        ]
      }
    }`)))
	})

	It("gets execution info in resolver", func() {
		var (
			encounteredContext   interface{}
			encounteredSchema    graphql.Schema
			encounteredRootValue interface{}
		)

		NamedType2 := &graphql.InterfaceConfig{
			Name: "Named",
			Fields: graphql.Fields{
				"name": {
					Type: graphql.T(graphql.String()),
				},
			},
		}

		PersonType2 := &graphql.ObjectConfig{
			Name: "Person",
			Interfaces: []graphql.InterfaceTypeDefinition{
				NamedType2,
			},
			Fields: graphql.Fields{
				"name": {
					Type: graphql.T(graphql.String()),
				},
				"friends": {
					Type: graphql.ListOf(NamedType2),
				},
			},
		}

		NamedType2.TypeResolver = graphql.TypeResolverFunc(func(ctx context.Context, value interface{}, info graphql.ResolveInfo) (graphql.Object, error) {
			encounteredContext = info.AppContext()
			encounteredSchema = info.Schema()
			encounteredRootValue = info.RootValue()
			return graphql.NewObject(PersonType2)
		})

		schema2 := graphql.MustNewSchema(&graphql.SchemaConfig{
			Query: graphql.MustNewObject(PersonType2),
		})

		john2 := &Person{
			Name: "John",
			Friends: []interface{}{
				liz,
			},
		}

		context := map[string]string{
			"authToken": "123abc",
		}

		query := "{ name, friends { name } }"

		Eventually(execute(schema2, query, john2, context)).Should(Receive(MatchIntrospectionInJSON(`{
      "data": {
        "name": "John",
        "friends": [
          {
            "name": "Liz"
          }
        ]
      }
    }`)))

		Expect(encounteredContext).Should(Equal(context))
		Expect(encounteredSchema).Should(Equal(schema2))
		Expect(encounteredRootValue).Should(Equal(john2))
	})
})
