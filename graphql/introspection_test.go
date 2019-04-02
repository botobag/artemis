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

package graphql_test

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/graphql/executor"
	"github.com/botobag/artemis/graphql/parser"
	"github.com/botobag/artemis/graphql/token"
	"github.com/botobag/artemis/graphql/util/introspection"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/matchers"
	"github.com/onsi/gomega/types"
)

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
		Data struct {
			Schema *struct {
				Types []interface{} `json:"types,omitempty"`
			} `json:"__schema,omitempty"`

			Type map[string]interface{} `json:"__type,omitempty"`
		} `json:"data"`

		Errors interface{} `json:"errors,omitempty"`
	}
	if err := json.Unmarshal([]byte(resultJSON), &result); err != nil {
		return nil, err
	}

	sortFieldByNameKey := func(t map[string]interface{}, field string) {
		v := t[field]
		if v != nil {
			sort.Sort(ByNameKey{v})
		}
	}

	sortType := func(t map[string]interface{}) {
		sortFieldByNameKey(t, "inputFields")
		sortFieldByNameKey(t, "fields")
		sortFieldByNameKey(t, "enumValues")
		// Hack: the test that check includeDeprecated for fields aliases "fields" field to the
		// following names.
		sortFieldByNameKey(t, "trueFields")
		sortFieldByNameKey(t, "falseFields")
		sortFieldByNameKey(t, "omittedFields")
		// Hack: the test that check includeDeprecated for enums aliases "enumValues" field to the
		// following names.
		sortFieldByNameKey(t, "trueValues")
		sortFieldByNameKey(t, "falseValues")
		sortFieldByNameKey(t, "omittedValues")
	}

	_schema := result.Data.Schema
	if _schema != nil {
		types := _schema.Types
		if types != nil {
			// Sort types by their names.
			sort.Sort(ByNameKey{types})
			// Sort fields and enumValues in a type by their names.
			for _, t := range types {
				sortType(t.(map[string]interface{}))
			}
		}
	}

	if result.Data.Type != nil {
		sortType(result.Data.Type)
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

var _ = Describe("Introspection", func() {
	// graphql-js/src/type/__tests__/introspection-test.js@4b55f10
	It("executes an introspection query", func() {
		EmptySchema := graphql.MustNewSchema(&graphql.SchemaConfig{
			Query: graphql.MustNewObject(&graphql.ObjectConfig{
				Name: "QueryRoot",
				Fields: graphql.Fields{
					"onlyField": {
						Type: graphql.T(graphql.String()),
					},
				},
			}),
		})

		query := introspection.Query(introspection.OmitDescriptions())

		Expect(executeQuery(EmptySchema, query)).Should(MatchIntrospectionInJSON(`{
      "data": {
        "__schema": {
          "mutationType": null,
          "subscriptionType": null,
          "queryType": {
            "name": "QueryRoot"
          },
          "types": [
            {
              "kind": "OBJECT",
              "name": "QueryRoot",
              "fields": [
                {
                  "name": "onlyField",
                  "args": [],
                  "type": {
                    "kind": "SCALAR",
                    "name": "String",
                    "ofType": null
                  },
                  "isDeprecated": false,
                  "deprecationReason": null
                }
              ],
              "inputFields": null,
              "interfaces": [],
              "enumValues": null,
              "possibleTypes": null
            },
            {
              "kind": "SCALAR",
              "name": "String",
              "fields": null,
              "inputFields": null,
              "interfaces": null,
              "enumValues": null,
              "possibleTypes": null
            },
            {
              "kind": "OBJECT",
              "name": "__Schema",
              "fields": [
                {
                  "name": "types",
                  "args": [],
                  "type": {
                    "kind": "NON_NULL",
                    "name": null,
                    "ofType": {
                      "kind": "LIST",
                      "name": null,
                      "ofType": {
                        "kind": "NON_NULL",
                        "name": null,
                        "ofType": {
                          "kind": "OBJECT",
                          "name": "__Type",
                          "ofType": null
                        }
                      }
                    }
                  },
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "queryType",
                  "args": [],
                  "type": {
                    "kind": "NON_NULL",
                    "name": null,
                    "ofType": {
                      "kind": "OBJECT",
                      "name": "__Type",
                      "ofType": null
                    }
                  },
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "mutationType",
                  "args": [],
                  "type": {
                    "kind": "OBJECT",
                    "name": "__Type",
                    "ofType": null
                  },
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "subscriptionType",
                  "args": [],
                  "type": {
                    "kind": "OBJECT",
                    "name": "__Type",
                    "ofType": null
                  },
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "directives",
                  "args": [],
                  "type": {
                    "kind": "NON_NULL",
                    "name": null,
                    "ofType": {
                      "kind": "LIST",
                      "name": null,
                      "ofType": {
                        "kind": "NON_NULL",
                        "name": null,
                        "ofType": {
                          "kind": "OBJECT",
                          "name": "__Directive",
                          "ofType": null
                        }
                      }
                    }
                  },
                  "isDeprecated": false,
                  "deprecationReason": null
                }
              ],
              "inputFields": null,
              "interfaces": [],
              "enumValues": null,
              "possibleTypes": null
            },
            {
              "kind": "OBJECT",
              "name": "__Type",
              "fields": [
                {
                  "name": "kind",
                  "args": [],
                  "type": {
                    "kind": "NON_NULL",
                    "name": null,
                    "ofType": {
                      "kind": "ENUM",
                      "name": "__TypeKind",
                      "ofType": null
                    }
                  },
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "name",
                  "args": [],
                  "type": {
                    "kind": "SCALAR",
                    "name": "String",
                    "ofType": null
                  },
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "description",
                  "args": [],
                  "type": {
                    "kind": "SCALAR",
                    "name": "String",
                    "ofType": null
                  },
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "fields",
                  "args": [
                    {
                      "name": "includeDeprecated",
                      "type": {
                        "kind": "SCALAR",
                        "name": "Boolean",
                        "ofType": null
                      },
                      "defaultValue": "false"
                    }
                  ],
                  "type": {
                    "kind": "LIST",
                    "name": null,
                    "ofType": {
                      "kind": "NON_NULL",
                      "name": null,
                      "ofType": {
                        "kind": "OBJECT",
                        "name": "__Field",
                        "ofType": null
                      }
                    }
                  },
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "interfaces",
                  "args": [],
                  "type": {
                    "kind": "LIST",
                    "name": null,
                    "ofType": {
                      "kind": "NON_NULL",
                      "name": null,
                      "ofType": {
                        "kind": "OBJECT",
                        "name": "__Type",
                        "ofType": null
                      }
                    }
                  },
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "possibleTypes",
                  "args": [],
                  "type": {
                    "kind": "LIST",
                    "name": null,
                    "ofType": {
                      "kind": "NON_NULL",
                      "name": null,
                      "ofType": {
                        "kind": "OBJECT",
                        "name": "__Type",
                        "ofType": null
                      }
                    }
                  },
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "enumValues",
                  "args": [
                    {
                      "name": "includeDeprecated",
                      "type": {
                        "kind": "SCALAR",
                        "name": "Boolean",
                        "ofType": null
                      },
                      "defaultValue": "false"
                    }
                  ],
                  "type": {
                    "kind": "LIST",
                    "name": null,
                    "ofType": {
                      "kind": "NON_NULL",
                      "name": null,
                      "ofType": {
                        "kind": "OBJECT",
                        "name": "__EnumValue",
                        "ofType": null
                      }
                    }
                  },
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "inputFields",
                  "args": [],
                  "type": {
                    "kind": "LIST",
                    "name": null,
                    "ofType": {
                      "kind": "NON_NULL",
                      "name": null,
                      "ofType": {
                        "kind": "OBJECT",
                        "name": "__InputValue",
                        "ofType": null
                      }
                    }
                  },
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "ofType",
                  "args": [],
                  "type": {
                    "kind": "OBJECT",
                    "name": "__Type",
                    "ofType": null
                  },
                  "isDeprecated": false,
                  "deprecationReason": null
                }
              ],
              "inputFields": null,
              "interfaces": [],
              "enumValues": null,
              "possibleTypes": null
            },
            {
              "kind": "ENUM",
              "name": "__TypeKind",
              "fields": null,
              "inputFields": null,
              "interfaces": null,
              "enumValues": [
                {
                  "name": "SCALAR",
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "OBJECT",
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "INTERFACE",
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "UNION",
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "ENUM",
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "INPUT_OBJECT",
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "LIST",
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "NON_NULL",
                  "isDeprecated": false,
                  "deprecationReason": null
                }
              ],
              "possibleTypes": null
            },
            {
              "kind": "SCALAR",
              "name": "Boolean",
              "fields": null,
              "inputFields": null,
              "interfaces": null,
              "enumValues": null,
              "possibleTypes": null
            },
            {
              "kind": "OBJECT",
              "name": "__Field",
              "fields": [
                {
                  "name": "name",
                  "args": [],
                  "type": {
                    "kind": "NON_NULL",
                    "name": null,
                    "ofType": {
                      "kind": "SCALAR",
                      "name": "String",
                      "ofType": null
                    }
                  },
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "description",
                  "args": [],
                  "type": {
                    "kind": "SCALAR",
                    "name": "String",
                    "ofType": null
                  },
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "args",
                  "args": [],
                  "type": {
                    "kind": "NON_NULL",
                    "name": null,
                    "ofType": {
                      "kind": "LIST",
                      "name": null,
                      "ofType": {
                        "kind": "NON_NULL",
                        "name": null,
                        "ofType": {
                          "kind": "OBJECT",
                          "name": "__InputValue",
                          "ofType": null
                        }
                      }
                    }
                  },
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "type",
                  "args": [],
                  "type": {
                    "kind": "NON_NULL",
                    "name": null,
                    "ofType": {
                      "kind": "OBJECT",
                      "name": "__Type",
                      "ofType": null
                    }
                  },
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "isDeprecated",
                  "args": [],
                  "type": {
                    "kind": "NON_NULL",
                    "name": null,
                    "ofType": {
                      "kind": "SCALAR",
                      "name": "Boolean",
                      "ofType": null
                    }
                  },
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "deprecationReason",
                  "args": [],
                  "type": {
                    "kind": "SCALAR",
                    "name": "String",
                    "ofType": null
                  },
                  "isDeprecated": false,
                  "deprecationReason": null
                }
              ],
              "inputFields": null,
              "interfaces": [],
              "enumValues": null,
              "possibleTypes": null
            },
            {
              "kind": "OBJECT",
              "name": "__InputValue",
              "fields": [
                {
                  "name": "name",
                  "args": [],
                  "type": {
                    "kind": "NON_NULL",
                    "name": null,
                    "ofType": {
                      "kind": "SCALAR",
                      "name": "String",
                      "ofType": null
                    }
                  },
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "description",
                  "args": [],
                  "type": {
                    "kind": "SCALAR",
                    "name": "String",
                    "ofType": null
                  },
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "type",
                  "args": [],
                  "type": {
                    "kind": "NON_NULL",
                    "name": null,
                    "ofType": {
                      "kind": "OBJECT",
                      "name": "__Type",
                      "ofType": null
                    }
                  },
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "defaultValue",
                  "args": [],
                  "type": {
                    "kind": "SCALAR",
                    "name": "String",
                    "ofType": null
                  },
                  "isDeprecated": false,
                  "deprecationReason": null
                }
              ],
              "inputFields": null,
              "interfaces": [],
              "enumValues": null,
              "possibleTypes": null
            },
            {
              "kind": "OBJECT",
              "name": "__EnumValue",
              "fields": [
                {
                  "name": "name",
                  "args": [],
                  "type": {
                    "kind": "NON_NULL",
                    "name": null,
                    "ofType": {
                      "kind": "SCALAR",
                      "name": "String",
                      "ofType": null
                    }
                  },
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "description",
                  "args": [],
                  "type": {
                    "kind": "SCALAR",
                    "name": "String",
                    "ofType": null
                  },
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "isDeprecated",
                  "args": [],
                  "type": {
                    "kind": "NON_NULL",
                    "name": null,
                    "ofType": {
                      "kind": "SCALAR",
                      "name": "Boolean",
                      "ofType": null
                    }
                  },
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "deprecationReason",
                  "args": [],
                  "type": {
                    "kind": "SCALAR",
                    "name": "String",
                    "ofType": null
                  },
                  "isDeprecated": false,
                  "deprecationReason": null
                }
              ],
              "inputFields": null,
              "interfaces": [],
              "enumValues": null,
              "possibleTypes": null
            },
            {
              "kind": "OBJECT",
              "name": "__Directive",
              "fields": [
                {
                  "name": "name",
                  "args": [],
                  "type": {
                    "kind": "NON_NULL",
                    "name": null,
                    "ofType": {
                      "kind": "SCALAR",
                      "name": "String",
                      "ofType": null
                    }
                  },
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "description",
                  "args": [],
                  "type": {
                    "kind": "SCALAR",
                    "name": "String",
                    "ofType": null
                  },
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "locations",
                  "args": [],
                  "type": {
                    "kind": "NON_NULL",
                    "name": null,
                    "ofType": {
                      "kind": "LIST",
                      "name": null,
                      "ofType": {
                        "kind": "NON_NULL",
                        "name": null,
                        "ofType": {
                          "kind": "ENUM",
                          "name": "__DirectiveLocation",
                          "ofType": null
                        }
                      }
                    }
                  },
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "args",
                  "args": [],
                  "type": {
                    "kind": "NON_NULL",
                    "name": null,
                    "ofType": {
                      "kind": "LIST",
                      "name": null,
                      "ofType": {
                        "kind": "NON_NULL",
                        "name": null,
                        "ofType": {
                          "kind": "OBJECT",
                          "name": "__InputValue",
                          "ofType": null
                        }
                      }
                    }
                  },
                  "isDeprecated": false,
                  "deprecationReason": null
                }
              ],
              "inputFields": null,
              "interfaces": [],
              "enumValues": null,
              "possibleTypes": null
            },
            {
              "kind": "ENUM",
              "name": "__DirectiveLocation",
              "fields": null,
              "inputFields": null,
              "interfaces": null,
              "enumValues": [
                {
                  "name": "QUERY",
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "MUTATION",
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "SUBSCRIPTION",
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "FIELD",
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "FRAGMENT_DEFINITION",
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "FRAGMENT_SPREAD",
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "INLINE_FRAGMENT",
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "VARIABLE_DEFINITION",
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "SCHEMA",
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "SCALAR",
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "OBJECT",
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "FIELD_DEFINITION",
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "ARGUMENT_DEFINITION",
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "INTERFACE",
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "UNION",
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "ENUM",
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "ENUM_VALUE",
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "INPUT_OBJECT",
                  "isDeprecated": false,
                  "deprecationReason": null
                },
                {
                  "name": "INPUT_FIELD_DEFINITION",
                  "isDeprecated": false,
                  "deprecationReason": null
                }
              ],
              "possibleTypes": null
            }
          ],
          "directives": [
            {
              "name": "include",
              "locations": [
                "FIELD",
                "FRAGMENT_SPREAD",
                "INLINE_FRAGMENT"
              ],
              "args": [
                {
                  "defaultValue": null,
                  "name": "if",
                  "type": {
                    "kind": "NON_NULL",
                    "name": null,
                    "ofType": {
                      "kind": "SCALAR",
                      "name": "Boolean",
                      "ofType": null
                    }
                  }
                }
              ]
            },
            {
              "name": "skip",
              "locations": [
                "FIELD",
                "FRAGMENT_SPREAD",
                "INLINE_FRAGMENT"
              ],
              "args": [
                {
                  "defaultValue": null,
                  "name": "if",
                  "type": {
                    "kind": "NON_NULL",
                    "name": null,
                    "ofType": {
                      "kind": "SCALAR",
                      "name": "Boolean",
                      "ofType": null
                    }
                  }
                }
              ]
            },
            {
              "name": "deprecated",
              "locations": [
                "FIELD_DEFINITION",
                "ENUM_VALUE"
              ],
              "args": [
                {
                  "defaultValue": "\"No longer supported\"",
                  "name": "reason",
                  "type": {
                    "kind": "SCALAR",
                    "name": "String",
                    "ofType": null
                  }
                }
              ]
            }
          ]
        }
      }
    }`))
	})

	It("introspects on input object", func() {
		TestInputObject := &graphql.InputObjectConfig{
			Name: "TestInputObject",
			Fields: graphql.InputFields{
				"a": {
					Type:         graphql.T(graphql.String()),
					DefaultValue: "tes\t de\fault",
				},
				"b": {
					Type: graphql.ListOfType(graphql.String()),
				},
				"c": {
					Type:         graphql.T(graphql.String()),
					DefaultValue: graphql.NilInputFieldDefaultValue,
				},
			},
		}

		TestType := graphql.MustNewObject(&graphql.ObjectConfig{
			Name: "TestType",
			Fields: graphql.Fields{
				"field": {
					Type: graphql.T(graphql.String()),
					Args: graphql.ArgumentConfigMap{
						"complex": {
							Type: TestInputObject,
						},
					},
				},
			},
		})

		schema := graphql.MustNewSchema(&graphql.SchemaConfig{
			Query: TestType,
		})

		query := `
      {
        __type(name: "TestInputObject") {
          kind
          name
          inputFields {
            name
            type { ...TypeRef }
            defaultValue
          }
        }
      }

      fragment TypeRef on __Type {
        kind
        name
        ofType {
          kind
          name
          ofType {
            kind
            name
            ofType {
              kind
              name
            }
          }
        }
      }
    `

		Expect(executeQuery(schema, query)).Should(MatchIntrospectionInJSON(`{
      "data": {
        "__type": {
          "kind": "INPUT_OBJECT",
          "name": "TestInputObject",
          "inputFields": [
            {
              "name": "a",
              "type": {
                "kind": "SCALAR",
                "name": "String",
                "ofType": null
              },
              "defaultValue": "tes\t de\fault"
            },
            {
              "name": "b",
              "type": {
                "kind": "LIST",
                "name": null,
                "ofType": {
                  "kind": "SCALAR",
                  "name": "String",
                  "ofType": null
                }
              },
              "defaultValue": null
            },
            {
              "name": "c",
              "type": {
                "kind": "SCALAR",
                "name": "String",
                "ofType": null
              },
              "defaultValue": null
            }
          ]
        }
      }
    }`))
	})

	It("supports the __type root field", func() {
		TestType := graphql.MustNewObject(&graphql.ObjectConfig{
			Name: "TestType",
			Fields: graphql.Fields{
				"testField": {
					Type: graphql.T(graphql.String()),
				},
			},
		})

		schema := graphql.MustNewSchema(&graphql.SchemaConfig{
			Query: TestType,
		})

		query := `
      {
        __type(name: "TestType") {
          name
        }
      }
		`

		Expect(executeQuery(schema, query)).Should(MatchIntrospectionInJSON(`{
      "data": {
        "__type": {
          "name": "TestType"
        }
      }
    }`))
	})

	It("identifies deprecated fields", func() {
		TestType := graphql.MustNewObject(&graphql.ObjectConfig{
			Name: "TestType",
			Fields: graphql.Fields{
				"nonDeprecated": {
					Type: graphql.T(graphql.String()),
				},
				"deprecated": {
					Type: graphql.T(graphql.String()),
					Deprecation: &graphql.Deprecation{
						Reason: "Removed in 1.0",
					},
				},
			},
		})

		schema := graphql.MustNewSchema(&graphql.SchemaConfig{
			Query: TestType,
		})

		query := `
      {
        __type(name: "TestType") {
          name
          fields(includeDeprecated: true) {
            name
            isDeprecated,
            deprecationReason
          }
        }
      }
		`

		Eventually(executeQuery(schema, query)).Should(MatchIntrospectionInJSON(`{
      "data": {
        "__type": {
          "name": "TestType",
          "fields": [
            {
              "name": "nonDeprecated",
              "isDeprecated": false,
              "deprecationReason": null
            },
            {
              "name": "deprecated",
              "isDeprecated": true,
              "deprecationReason": "Removed in 1.0"
            }
          ]
        }
      }
    }`))
	})

	It("respects the includeDeprecated parameter for fields", func() {
		TestType := graphql.MustNewObject(&graphql.ObjectConfig{
			Name: "TestType",
			Fields: graphql.Fields{
				"nonDeprecated": {
					Type: graphql.T(graphql.String()),
				},
				"deprecated": {
					Type: graphql.T(graphql.String()),
					Deprecation: &graphql.Deprecation{
						Reason: "Removed in 1.0",
					},
				},
			},
		})

		schema := graphql.MustNewSchema(&graphql.SchemaConfig{
			Query: TestType,
		})

		query := `
      {
        __type(name: "TestType") {
          name
          trueFields: fields(includeDeprecated: true) {
            name
          }
          falseFields: fields(includeDeprecated: false) {
            name
          }
          omittedFields: fields {
            name
          }
        }
      }
		`

		Expect(executeQuery(schema, query)).Should(MatchIntrospectionInJSON(`{
      "data": {
        "__type": {
          "name": "TestType",
          "trueFields": [
            {
              "name": "nonDeprecated"
            },
            {
              "name": "deprecated"
            }
          ],
          "falseFields": [
            {
              "name": "nonDeprecated"
            }
          ],
          "omittedFields": [
            {
              "name": "nonDeprecated"
            }
          ]
        }
      }
    }`))
	})

	Context("Enum", func() {
		var schema graphql.Schema

		BeforeEach(func() {
			TestEnum := &graphql.EnumConfig{
				Name: "TestEnum",
				Values: graphql.EnumValueDefinitionMap{
					"NONDEPRECATED": {
						Value: 0,
					},
					"DEPRECATED": {
						Value: 1,
						Deprecation: &graphql.Deprecation{
							Reason: "Removed in 1.0",
						},
					},
					"ALSONONDEPRECATED": {
						Value: 2,
					},
				},
			}

			TestType := graphql.MustNewObject(&graphql.ObjectConfig{
				Name: "TestType",
				Fields: graphql.Fields{
					"testEnum": {
						Type: TestEnum,
					},
				},
			})

			schema = graphql.MustNewSchema(&graphql.SchemaConfig{
				Query: TestType,
			})
		})

		It("identifies deprecated enum values", func() {
			query := `
      {
        __type(name: "TestEnum") {
          name
          enumValues(includeDeprecated: true) {
            name
            isDeprecated,
            deprecationReason
          }
        }
      }
		`

			Expect(executeQuery(schema, query)).Should(MatchIntrospectionInJSON(`{
        "data": {
          "__type": {
            "name": "TestEnum",
            "enumValues": [
              {
                "name": "NONDEPRECATED",
                "isDeprecated": false,
                "deprecationReason": null
              },
              {
                "name": "DEPRECATED",
                "isDeprecated": true,
                "deprecationReason": "Removed in 1.0"
              },
              {
                "name": "ALSONONDEPRECATED",
                "isDeprecated": false,
                "deprecationReason": null
              }
            ]
          }
        }
      }`))
		})

		It("respects the includeDeprecated parameter for enum values", func() {
			query := `{
        __type(name: "TestEnum") {
          name
          trueValues: enumValues(includeDeprecated: true) {
            name
          }
          falseValues: enumValues(includeDeprecated: false) {
            name
          }
          omittedValues: enumValues {
            name
          }
        }
      }
    `

			Expect(executeQuery(schema, query)).Should(MatchIntrospectionInJSON(`{
        "data": {
          "__type": {
            "name": "TestEnum",
            "trueValues": [
              {
                "name": "NONDEPRECATED"
              },
              {
                "name": "DEPRECATED"
              },
              {
                "name": "ALSONONDEPRECATED"
              }
            ],
            "falseValues": [
              {
                "name": "NONDEPRECATED"
              },
              {
                "name": "ALSONONDEPRECATED"
              }
            ],
            "omittedValues": [
              {
                "name": "NONDEPRECATED"
              },
              {
                "name": "ALSONONDEPRECATED"
              }
            ]
          }
        }
      }`))
		})
	})

	// It("fails as expected on the __type root field without an arg", func() {
	// TODO: Validation
	// })

	It("exposes descriptions on types and fields", func() {
		QueryRoot := graphql.MustNewObject(&graphql.ObjectConfig{
			Name: "QueryRoot",
			Fields: graphql.Fields{
				"onlyField": {
					Type: graphql.T(graphql.String()),
				},
			},
		})

		schema := graphql.MustNewSchema(&graphql.SchemaConfig{
			Query: QueryRoot,
		})

		query := `
      {
        schemaType: __type(name: "__Schema") {
          name,
          description,
          fields {
            name,
            description
          }
        }
      }
		`

		Expect(executeQuery(schema, query)).Should(MatchIntrospectionInJSON(`{
      "data": {
        "typeKindType": {
          "name": "__TypeKind",
          "description": "` + "An enum describing what kind of type a given `__Type` is." + `",
          "enumValues": [
            {
              "description": "Indicates this type is a scalar.",
              "name": "SCALAR"
            },
            {
              "description": "` + "Indicates this type is an object. `fields` and `interfaces` are valid fields." + `",
              "name": "OBJECT"
            },
            {
              "description": "` + "Indicates this type is an interface. `fields` and `possibleTypes` are valid fields." + `",
              "name": "INTERFACE"
            },
            {
              "description": "` + "Indicates this type is a union. `possibleTypes` is a valid field." + `",
              "name": "UNION"
            },
            {
              "description": "` + "Indicates this type is an enum. `enumValues` is a valid field." + `",
              "name": "ENUM"
            },
            {
              "description": "` + "Indicates this type is an input object. `inputFields` is a valid field." + `",
              "name": "INPUT_OBJECT"
            },
            {
              "description": "` + "Indicates this type is a list. `ofType` is a valid field." + `",
              "name": "LIST"
            },
            {
              "description": "` + "Indicates this type is a non-null. `ofType` is a valid field." + `",
              "name": "NON_NULL"
            }
          ]
        }
      }
    }`))
	})

	It("executes an introspection query without calling global fieldResolver", func() {
		QueryRoot := graphql.MustNewObject(&graphql.ObjectConfig{
			Name: "QueryRoot",
			Fields: graphql.Fields{
				"onlyField": {
					Type: graphql.T(graphql.String()),
				},
			},
		})

		schema := graphql.MustNewSchema(&graphql.SchemaConfig{
			Query: QueryRoot,
		})

		query := introspection.Query()
		calledForFields := map[string]bool{}

		document, err := parser.Parse(token.NewSource(query))
		Expect(err).ShouldNot(HaveOccurred())

		operation, errs := executor.Prepare(schema, document, executor.DefaultFieldResolver(
			graphql.FieldResolverFunc(func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
				calledForFields[fmt.Sprintf("%s::%s", info.Object().Name(), info.Field().Name())] = true
				return nil, nil
			})))
		Expect(errs.HaveOccurred()).ShouldNot(BeTrue())

		var result executor.ExecutionResult
		Eventually(operation.Execute(context.Background())).Should(Receive(&result))

		Expect(calledForFields).Should(BeEmpty())
	})
})
