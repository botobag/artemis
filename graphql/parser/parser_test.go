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

package parser_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math"
	"text/template"

	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/graphql/ast"
	"github.com/botobag/artemis/graphql/parser"
	"github.com/botobag/artemis/graphql/token"
	"github.com/botobag/artemis/internal/testutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
)

func parse(s string) (ast.Document, error) {
	return parser.Parse(token.NewSource(&token.SourceConfig{
		Body: token.SourceBody([]byte(s)),
	}), parser.ParseOptions{})
}

func parseValue(s string) (ast.Value, error) {
	return parser.ParseValue(token.NewSource(&token.SourceConfig{
		Body: token.SourceBody([]byte(s)),
	}))
}

func parseType(s string) (ast.Type, error) {
	return parser.ParseType(token.NewSource(&token.SourceConfig{
		Body: token.SourceBody([]byte(s)),
	}))
}

func expectSyntaxError(text string, message string, location graphql.ErrorLocation) {
	_, err := parse(text)
	Expect(err).Should(testutil.MatchGraphQLError(
		testutil.MessageContainSubstring(message),
		testutil.LocationEqual(location),
		testutil.KindIs(graphql.ErrKindSyntax),
	))
}

// Fields in token.Token to match.
type TokenFields struct {
	Kind     token.Kind
	Length   uint
	Value    string
	Location uint
}

// MatchToken is a matcher for token.Token.
func MatchToken(fields TokenFields) types.GomegaMatcher {
	return PointTo(MatchFields(IgnoreExtras, Fields{
		"Kind":     Equal(fields.Kind),
		"Location": Equal(token.SourceLocation(fields.Location)),
		"Length":   Equal(fields.Length),
		"Value":    Equal(fields.Value),
	}))
}

// Fields in token.TokenRange to match.
type TokenRangeFields struct {
	First TokenFields
	Last  TokenFields
}

// MatchTokenRange is a matcher for token.TokenRange.
func MatchTokenRange(fields TokenRangeFields) types.GomegaMatcher {
	return MatchAllFields(Fields{
		"First": MatchToken(fields.First),
		"Last":  MatchToken(fields.Last),
	})
}

// Fields in ast.Field to match
type FieldNodeFields struct {
	Alias        types.GomegaMatcher
	Name         TokenFields
	Arguments    types.GomegaMatcher
	Directives   types.GomegaMatcher
	SelectionSet types.GomegaMatcher
}

// MatchFieldNode is a matcher for ast.Field.
func MatchFieldNode(fields FieldNodeFields) types.GomegaMatcher {
	return PointTo(MatchAllFields(Fields{
		"Alias":        fields.Alias,
		"Name":         MatchNameNode(fields.Name),
		"Arguments":    fields.Arguments,
		"Directives":   fields.Directives,
		"SelectionSet": fields.SelectionSet,
	}))
}

// MatchNameNode is a matcher for ast.Name.
func MatchNameNode(fields TokenFields) types.GomegaMatcher {
	return MatchAllFields(Fields{
		"Token": MatchToken(fields),
	})
}

var _ = Describe("Parser", func() {
	// graphql-js/src/language/__tests__/parser-test.js
	It("asserts that a source to parse was provided", func() {
		_, err := parser.Parse(nil, parser.ParseOptions{})
		Expect(err).Should(MatchError("Must provide Source. Received: nil"))

		_, err = parser.ParseValue(nil)
		Expect(err).Should(MatchError("Must provide Source. Received: nil"))

		_, err = parser.ParseType(nil)
		Expect(err).Should(MatchError("Must provide Source. Received: nil"))
	})

	It("parse provides useful errors", func() {
		_, err := parse("{")
		Expect(err).Should(PointTo(MatchFields(IgnoreExtras, Fields{
			"Message": Equal("Syntax Error: Expected Name, found <EOF>"),
			"Locations": Equal([]graphql.ErrorLocation{
				{Line: 1, Column: 2},
			}),
			"Kind": Equal(graphql.ErrKindSyntax),
		})))

		expectSyntaxError(
			`
      { ...MissingOn }
      fragment MissingOn Type`,
			`Expected "on", found Name "Type"`,
			graphql.ErrorLocation{
				Line:   3,
				Column: 26,
			},
		)

		expectSyntaxError("{ field: {} }", "Expected Name, found {", graphql.ErrorLocation{
			Line:   1,
			Column: 10,
		})

		expectSyntaxError(
			"notanoperation Foo { field }",
			`Unexpected Name "notanoperation"`,
			graphql.ErrorLocation{
				Line:   1,
				Column: 1,
			},
		)

		expectSyntaxError("...", "Unexpected ...", graphql.ErrorLocation{
			Line:   1,
			Column: 1,
		})
	})

	It("parses variable inline values", func() {
		_, err := parse("{ field(complex: { a: { b: [ $var ] } }) }")
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("parses constant default values", func() {
		expectSyntaxError(
			"query Foo($x: Complex = { a: { b: [ $var ] } }) { field }",
			"Unexpected $",
			graphql.ErrorLocation{
				Line:   1,
				Column: 37,
			})
	})

	It("parses variable definition directives", func() {
		_, err := parse("query Foo($x: Boolean = false @bar) { field }")
		Expect(err).ShouldNot(HaveOccurred())
	})

	It(`does not accept fragments named "on"`, func() {
		expectSyntaxError(
			"fragment on on on { on }",
			`Expected a fragment name before "on"`,
			graphql.ErrorLocation{
				Line:   1,
				Column: 10,
			})
	})

	It(`does not accept fragments spread of "on"`, func() {
		expectSyntaxError("{ ...on }", "Expected Name, found }", graphql.ErrorLocation{
			Line:   1,
			Column: 9,
		})
	})

	It("parses multi-byte characters", func() {
		// Note: \u0A0A could be naively interpreted as two line-feed chars.
		document, err := parse(`
      # This comment has a \u0A0A multi-byte character.
      { field(arg: "Has a \u0A0A multi-byte character.") }
    `)

		Expect(err).ShouldNot(HaveOccurred())
		Expect(
			document.
				Definitions[0].(ast.ExecutableDefinition).
				GetSelectionSet()[0].(*ast.Field).
				Arguments[0].
				Value.
				Interface(),
		).Should(Equal("Has a \u0A0A multi-byte character."))
	})

	It("parses kitchen sink", func() {
		kitchenSink, err := ioutil.ReadFile("./kitchen-sink.graphql")
		Expect(err).ShouldNot(HaveOccurred())

		_, err = parser.Parse(token.NewSource(&token.SourceConfig{
			Body: token.SourceBody(kitchenSink),
		}), parser.ParseOptions{})
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("allows non-keywords anywhere a Name is allowed", func() {
		nonKeywords := []string{
			"on",
			"fragment",
			"query",
			"mutation",
			"subscription",
			"true",
			"false",
		}

		document, err := template.New("keywork-document").Parse(`
        query {{.Keyword}} {
          ... {{.FragmentName}}
          ... on {{.Keyword}} { field }
        }
        fragment {{.FragmentName}} on Type {
          {{.Keyword}}({{.Keyword}}: ${{.Keyword}})
            @{{.Keyword}}({{.Keyword}}: {{.Keyword}})
        }
      `)
		Expect(err).ShouldNot(HaveOccurred())

		for _, keyword := range nonKeywords {
			fragmentName := keyword
			if fragmentName == "on" {
				// You can't define or reference a fragment named `on`.
				fragmentName = "a"
			}

			var buf bytes.Buffer
			Expect(document.Execute(&buf, struct {
				Keyword      string
				FragmentName string
			}{
				Keyword:      keyword,
				FragmentName: fragmentName,
			})).Should(Succeed())

			_, err = parser.Parse(token.NewSource(&token.SourceConfig{
				Body: token.SourceBody(buf.Bytes()),
			}), parser.ParseOptions{})
			Expect(err).ShouldNot(HaveOccurred())
		}
	})

	It("parses anonymous mutation operations", func() {
		_, err := parse(`
      mutation {
        mutationField
      }
    `)
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("parses anonymous subscription operations", func() {
		_, err := parse(`
      subscription {
        subscriptionField
      }
    `)
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("parses anonymous subscription operations", func() {
		_, err := parse(`
      mutation Foo {
        mutationField
      }
    `)
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("parses named subscription operations", func() {
		_, err := parse(`
      subscription Foo {
        subscriptionField
      }
    `)
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("creates ast", func() {
		result, err := parse(`
      {
        node(id: 4) {
          id,
          name
        }
      }
    `)
		Expect(err).ShouldNot(HaveOccurred())

		Expect(result).Should(MatchAllFields(Fields{
			"Definitions": ConsistOf(PointTo(MatchAllFields(Fields{
				"DefinitionBase": MatchAllFields(Fields{
					"Directives": BeEmpty(),
				}),
				"Type":                BeNil(),
				"Name":                Equal(ast.Name{}),
				"VariableDefinitions": BeEmpty(),
				"SelectionSet": ConsistOf(
					MatchFieldNode(FieldNodeFields{
						Alias: Equal(ast.Name{}),
						Name: TokenFields{
							Kind:     token.KindName,
							Location: 18,
							Length:   4,
							Value:    "node",
						},
						Arguments: ConsistOf(PointTo(MatchAllFields(Fields{
							"Name": MatchNameNode(TokenFields{
								Kind:     token.KindName,
								Location: 23,
								Length:   2,
								Value:    "id",
							}),
							"Value": MatchAllFields(Fields{
								"Token": MatchToken(TokenFields{
									Kind:     token.KindInt,
									Location: 27,
									Length:   1,
									Value:    "4",
								}),
							}),
						}))),
						Directives: BeEmpty(),
						SelectionSet: ConsistOf(
							MatchFieldNode(FieldNodeFields{
								Alias: Equal(ast.Name{}),
								Name: TokenFields{
									Kind:     token.KindName,
									Location: 42,
									Length:   2,
									Value:    "id",
								},
								Arguments:    BeEmpty(),
								Directives:   BeEmpty(),
								SelectionSet: BeEmpty(),
							}),
							MatchFieldNode(FieldNodeFields{
								Alias: Equal(ast.Name{}),
								Name: TokenFields{
									Kind:     token.KindName,
									Location: 56,
									Length:   4,
									Value:    "name",
								},
								Arguments:    BeEmpty(),
								Directives:   BeEmpty(),
								SelectionSet: BeEmpty(),
							}),
						),
					}),
				),
			}))),
		}))
	})

	It("creates ast from nameless query without variables", func() {
		result, err := parse(`
      query {
        node {
          id
        }
      }
    `)
		Expect(err).ShouldNot(HaveOccurred())

		Expect(result).Should(MatchAllFields(Fields{
			"Definitions": ConsistOf(PointTo(MatchAllFields(Fields{
				"DefinitionBase": MatchAllFields(Fields{
					"Directives": BeEmpty(),
				}),
				"Type": MatchToken(TokenFields{
					Kind:     token.KindName,
					Location: 8,
					Length:   5,
					Value:    "query",
				}),
				"Name":                Equal(ast.Name{}),
				"VariableDefinitions": BeEmpty(),
				"SelectionSet": ConsistOf(
					MatchFieldNode(FieldNodeFields{
						Alias: Equal(ast.Name{}),
						Name: TokenFields{
							Kind:     token.KindName,
							Location: 24,
							Length:   4,
							Value:    "node",
						},
						Arguments:  BeEmpty(),
						Directives: BeEmpty(),
						SelectionSet: ConsistOf(
							MatchFieldNode(FieldNodeFields{
								Alias: Equal(ast.Name{}),
								Name: TokenFields{
									Kind:     token.KindName,
									Location: 41,
									Length:   2,
									Value:    "id",
								},
								Arguments:    BeEmpty(),
								Directives:   BeEmpty(),
								SelectionSet: BeEmpty(),
							}),
						),
					}),
				),
			}))),
		}))
	})

	It("Experimental: allows parsing fragment defined variables", func() {
		document := "fragment a($v: Boolean = false) on t { f(v: $v) }"
		expectSyntaxError(document, `Expected "on", found (`, graphql.ErrorLocation{
			Line:   1,
			Column: 11,
		})

		_, err := parser.Parse(token.NewSource(&token.SourceConfig{
			Body: token.SourceBody([]byte(document)),
		}), parser.ParseOptions{
			ExperimentalFragmentVariables: true,
		})
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("contains location information", func() {
		source := token.NewSource(&token.SourceConfig{
			Body: token.SourceBody([]byte("{ id }")),
		})
		result, err := parser.Parse(source, parser.ParseOptions{})
		Expect(err).ShouldNot(HaveOccurred())

		sourceRange := result.TokenRange().SourceRange()
		// Note that "result" is an ast.Document whose first token is set to SOF and has
		// NoSourceLocation.
		Expect(sourceRange.Begin).Should(Equal(token.NoSourceLocation))
		Expect(sourceRange.End).Should(Equal(source.LocationFromPos(6)))
	})

	It("contains references to start and end tokens", func() {
		result, err := parse(`{ id }`)
		Expect(err).ShouldNot(HaveOccurred())

		tokenRange := result.TokenRange()
		Expect(tokenRange.First.Kind).Should(Equal(token.KindSOF))
		Expect(tokenRange.Last.Kind).Should(Equal(token.KindEOF))
	})

	Describe("ParseValue", func() {
		It("parses null value", func() {
			result, err := parseValue("null")
			Expect(err).ShouldNot(HaveOccurred())

			value, ok := result.(ast.NullValue)
			Expect(ok).Should(BeTrue())

			Expect(value.Token).Should(MatchToken(TokenFields{
				Kind:     token.KindName,
				Location: 1,
				Length:   4,
				Value:    "null",
			}))
		})

		It("parses list values", func() {
			result, err := parseValue(`[123 "abc"]`)
			Expect(err).ShouldNot(HaveOccurred())

			value, ok := result.(ast.ListValue)
			Expect(ok).Should(BeTrue())

			Expect(value.IsEmpty()).Should(BeFalse())
			Expect(value.Values()).Should(ConsistOf(
				MatchAllFields(Fields{
					"Token": MatchToken(TokenFields{
						Kind:     token.KindInt,
						Location: 2,
						Length:   3,
						Value:    "123",
					}),
				}),

				MatchAllFields(Fields{
					"Token": MatchToken(TokenFields{
						Kind:     token.KindString,
						Location: 6,
						Length:   5,
						Value:    "abc",
					}),
				}),
			))
		})

		It("parses block strings", func() {
			result, err := parseValue(`["""long""" "short"]`)
			Expect(err).ShouldNot(HaveOccurred())

			value, ok := result.(ast.ListValue)
			Expect(ok).Should(BeTrue())

			Expect(value.IsEmpty()).Should(BeFalse())
			Expect(value.Values()).Should(ConsistOf(
				MatchAllFields(Fields{
					"Token": MatchToken(TokenFields{
						Kind:     token.KindBlockString,
						Location: 2,
						Length:   10,
						Value:    "long",
					}),
				}),

				MatchAllFields(Fields{
					"Token": MatchToken(TokenFields{
						Kind:     token.KindString,
						Location: 13,
						Length:   7,
						Value:    "short",
					}),
				}),
			))
		})

		It("parse nested list value", func() {
			result, err := parseValue(`[[[[123]]]]`)
			Expect(err).ShouldNot(HaveOccurred())

			list1, ok := result.(ast.ListValue)
			Expect(ok).Should(BeTrue())
			Expect(list1.IsEmpty()).Should(BeFalse())
			Expect(len(list1.Values())).Should(Equal(1))

			list2, ok := list1.Values()[0].(ast.ListValue)
			Expect(ok).Should(BeTrue())
			Expect(list2.IsEmpty()).Should(BeFalse())
			Expect(len(list2.Values())).Should(Equal(1))

			list3, ok := list2.Values()[0].(ast.ListValue)
			Expect(ok).Should(BeTrue())
			Expect(list3.IsEmpty()).Should(BeFalse())
			Expect(len(list3.Values())).Should(Equal(1))

			list4, ok := list3.Values()[0].(ast.ListValue)
			Expect(ok).Should(BeTrue())
			Expect(list4.IsEmpty()).Should(BeFalse())
			Expect(len(list4.Values())).Should(Equal(1))

			Expect(list4.IsEmpty()).Should(BeFalse())
			Expect(list4.Values()).Should(ConsistOf(
				MatchAllFields(Fields{
					"Token": MatchToken(TokenFields{
						Kind:     token.KindInt,
						Location: 5,
						Length:   3,
						Value:    "123",
					}),
				}),
			))
		})

		It("parses an empty list", func() {
			result, err := parseValue(`    []`)
			Expect(err).ShouldNot(HaveOccurred())

			value, ok := result.(ast.ListValue)
			Expect(ok).Should(BeTrue())
			Expect(value.IsEmpty()).Should(BeTrue())
			Expect(value.Values()).Should(BeEmpty())
			Expect(value.TokenRange()).Should(MatchTokenRange(TokenRangeFields{
				First: TokenFields{
					Kind:     token.KindLeftBracket,
					Location: 5,
					Length:   1,
				},
				Last: TokenFields{
					Kind:     token.KindRightBracket,
					Location: 6,
					Length:   1,
				},
			}))
		})

		It("parses an empty object", func() {
			result, err := parseValue(`  {    }  `)
			Expect(err).ShouldNot(HaveOccurred())

			value, ok := result.(ast.ObjectValue)
			Expect(ok).Should(BeTrue())
			Expect(value.HasFields()).Should(BeFalse())
			Expect(value.Fields()).Should(BeEmpty())
			Expect(value.TokenRange()).Should(MatchTokenRange(TokenRangeFields{
				First: TokenFields{
					Kind:     token.KindLeftBrace,
					Location: 3,
					Length:   1,
				},
				Last: TokenFields{
					Kind:     token.KindRightBrace,
					Location: 8,
					Length:   1,
				},
			}))
		})

		It("parses boolean values", func() {
			tests := []string{"true", "false"}
			for _, test := range tests {
				result, err := parseValue(test)
				Expect(err).ShouldNot(HaveOccurred())

				value, ok := result.(ast.BooleanValue)
				Expect(ok).Should(BeTrue())

				Expect(value.Token).Should(MatchToken(TokenFields{
					Kind:     token.KindName,
					Location: 1,
					Length:   uint(len(test)),
					Value:    test,
				}))

				Expect(value.TokenRange()).Should(MatchTokenRange(TokenRangeFields{
					First: TokenFields{
						Kind:     token.KindName,
						Location: 1,
						Length:   uint(len(test)),
						Value:    test,
					},
					Last: TokenFields{
						Kind:     token.KindName,
						Location: 1,
						Length:   uint(len(test)),
						Value:    test,
					},
				}))

				if test == "true" {
					Expect(value.Value()).Should(BeTrue())
					Expect(value.Interface()).Should(BeTrue())
				} else {
					Expect(value.Value()).Should(BeFalse())
					Expect(value.Interface()).Should(BeFalse())
				}
			}
		})

		It("parses int values", func() {
			largeNumberTests := map[string]int32{
				"-8190283917982478127489274192749874": 0,
				"7219896182364762369416748936479639":  0,
			}

			negativeNumberTests := map[string]int32{
				"-1003748": -1003748,
				"-1003":    -1003,
				"-1":       -1,
			}

			allTests := map[string]int32{
				"0":      0,
				"1":      1,
				"123":    123,
				"123333": 123333,
			}
			for k, v := range largeNumberTests {
				allTests[k] = v
			}
			for k, v := range negativeNumberTests {
				allTests[k] = v
			}

			for test, expectedValue := range allTests {
				result, err := parseValue(test)
				Expect(err).ShouldNot(HaveOccurred())

				value, ok := result.(ast.IntValue)
				Expect(ok).Should(BeTrue())

				Expect(value.Token).Should(MatchToken(TokenFields{
					Kind:     token.KindInt,
					Location: 1,
					Length:   uint(len(test)),
					Value:    test,
				}))

				Expect(value.TokenRange()).Should(MatchTokenRange(TokenRangeFields{
					First: TokenFields{
						Kind:     token.KindInt,
						Location: 1,
						Length:   uint(len(test)),
						Value:    test,
					},
					Last: TokenFields{
						Kind:     token.KindInt,
						Location: 1,
						Length:   uint(len(test)),
						Value:    test,
					},
				}))

				Expect(value.String()).Should(Equal(test))
				Expect(value.Interface()).Should(Equal(expectedValue))

				if _, isLargeNumber := largeNumberTests[test]; isLargeNumber {
					_, err = value.Uint32Value()
					Expect(err).Should(HaveOccurred())
					_, err = value.Int32Value()
					Expect(err).Should(HaveOccurred())
					_, err = value.Uint64Value()
					Expect(err).Should(HaveOccurred())
					_, err = value.Int64Value()
					Expect(err).Should(HaveOccurred())
				} else if _, isNegativeNumber := negativeNumberTests[test]; isNegativeNumber {
					_, err = value.Uint32Value()
					Expect(err).Should(HaveOccurred())
					_, err = value.Uint64Value()
					Expect(err).Should(HaveOccurred())

					Expect(value.Int32Value()).Should(Equal(expectedValue))
					Expect(value.Int64Value()).Should(Equal(int64(expectedValue)))
				} else {
					Expect(value.Int32Value()).Should(Equal(expectedValue))
					Expect(value.Uint32Value()).Should(Equal(uint32(expectedValue)))
					Expect(value.Int64Value()).Should(Equal(int64(expectedValue)))
					Expect(value.Uint64Value()).Should(Equal(uint64(expectedValue)))
				}
			}
		})

		It("parses float values", func() {
			tests := []struct {
				s             string
				expectedValue float64
			}{
				{"1.23", 1.23},
				{"-1.23", -1.23},
				{"1e10", 1e10},
				{"0.0", 0.0},
				{"123.456e789", math.NaN()},
			}

			for _, test := range tests {
				result, err := parseValue(test.s)
				Expect(err).ShouldNot(HaveOccurred())

				value, ok := result.(ast.FloatValue)
				Expect(ok).Should(BeTrue())

				Expect(value.Token).Should(MatchToken(TokenFields{
					Kind:     token.KindFloat,
					Location: 1,
					Length:   uint(len(test.s)),
					Value:    test.s,
				}))

				Expect(value.TokenRange()).Should(MatchTokenRange(TokenRangeFields{
					First: TokenFields{
						Kind:     token.KindFloat,
						Location: 1,
						Length:   uint(len(test.s)),
						Value:    test.s,
					},
					Last: TokenFields{
						Kind:     token.KindFloat,
						Location: 1,
						Length:   uint(len(test.s)),
						Value:    test.s,
					},
				}))

				Expect(value.String()).Should(Equal(test.s))
				if math.IsNaN(test.expectedValue) {
					_, err := value.FloatValue()
					Expect(err).Should(HaveOccurred())
					Expect(math.IsNaN(value.Interface().(float64))).Should(BeTrue())
				} else {
					Expect(value.FloatValue()).Should(Equal(test.expectedValue))
					Expect(value.Interface()).Should(Equal(test.expectedValue))
				}
			}
		})

		It("rejects multiple values", func() {
			_, err := parseValue(`1 2`)
			Expect(err).Should(testutil.MatchGraphQLError(
				testutil.MessageContainSubstring(`Expected <EOF>, found Int "2"`),
				testutil.LocationEqual(graphql.ErrorLocation{
					Line:   1,
					Column: 3,
				}),
				testutil.KindIs(graphql.ErrKindSyntax),
			))
		})

		It("reject invalid values", func() {
			_, err := parseValue("@deprecated")
			Expect(err).Should(testutil.MatchGraphQLError(
				testutil.MessageContainSubstring("Unexpected @"),
				testutil.LocationEqual(graphql.ErrorLocation{
					Line:   1,
					Column: 1,
				}),
				testutil.KindIs(graphql.ErrKindSyntax),
			))
		})
	})

	Describe("ParseType", func() {
		It("parses well known types", func() {
			result, err := parseType("String")
			Expect(err).ShouldNot(HaveOccurred())

			t, ok := result.(ast.NamedType)
			Expect(ok).Should(BeTrue())

			Expect(t.Name).Should(MatchNameNode(TokenFields{
				Kind:     token.KindName,
				Location: 1,
				Length:   6,
				Value:    "String",
			}))
		})

		It("parses custom types", func() {
			result, err := parseType("MyType")
			Expect(err).ShouldNot(HaveOccurred())

			t, ok := result.(ast.NamedType)
			Expect(ok).Should(BeTrue())

			Expect(t.Name).Should(MatchNameNode(TokenFields{
				Kind:     token.KindName,
				Location: 1,
				Length:   6,
				Value:    "MyType",
			}))
		})

		It("parses list types", func() {
			result, err := parseType("[MyType]")
			Expect(err).ShouldNot(HaveOccurred())

			t, ok := result.(ast.ListType)
			Expect(ok).Should(BeTrue())
			Expect(t.TokenRange()).Should(MatchTokenRange(TokenRangeFields{
				First: TokenFields{
					Kind:     token.KindLeftBracket,
					Location: 1,
					Length:   1,
				},
				Last: TokenFields{
					Kind:     token.KindRightBracket,
					Location: 8,
					Length:   1,
				},
			}))

			itemType, ok := t.ItemType.(ast.NamedType)
			Expect(ok).Should(BeTrue())
			Expect(itemType.TokenRange()).Should(MatchTokenRange(TokenRangeFields{
				First: TokenFields{
					Kind:     token.KindName,
					Location: 2,
					Length:   6,
					Value:    "MyType",
				},
				Last: TokenFields{
					Kind:     token.KindName,
					Location: 2,
					Length:   6,
					Value:    "MyType",
				},
			}))

			Expect(itemType.Name).Should(MatchNameNode(TokenFields{
				Kind:     token.KindName,
				Location: 2,
				Length:   6,
				Value:    "MyType",
			}))
		})

		It("parses non-null types", func() {
			result, err := parseType("MyType!")
			Expect(err).ShouldNot(HaveOccurred())

			t, ok := result.(ast.NonNullType)
			Expect(ok).Should(BeTrue())
			Expect(t.TokenRange()).Should(MatchTokenRange(TokenRangeFields{
				First: TokenFields{
					Kind:     token.KindName,
					Location: 1,
					Length:   6,
					Value:    "MyType",
				},
				Last: TokenFields{
					Kind:     token.KindBang,
					Location: 7,
					Length:   1,
				},
			}))

			itemType, ok := t.Type.(ast.NamedType)
			Expect(ok).Should(BeTrue())
			Expect(itemType.TokenRange()).Should(MatchTokenRange(TokenRangeFields{
				First: TokenFields{
					Kind:     token.KindName,
					Location: 1,
					Length:   6,
					Value:    "MyType",
				},
				Last: TokenFields{
					Kind:     token.KindName,
					Location: 1,
					Length:   6,
					Value:    "MyType",
				},
			}))

			Expect(itemType.Name).Should(MatchNameNode(TokenFields{
				Kind:     token.KindName,
				Location: 1,
				Length:   6,
				Value:    "MyType",
			}))
		})

		It("parses nested types", func() {
			result, err := parseType("[MyType!]")
			Expect(err).ShouldNot(HaveOccurred())

			t, ok := result.(ast.ListType)
			Expect(ok).Should(BeTrue())
			Expect(t.TokenRange()).Should(MatchTokenRange(TokenRangeFields{
				First: TokenFields{
					Kind:     token.KindLeftBracket,
					Location: 1,
					Length:   1,
				},
				Last: TokenFields{
					Kind:     token.KindRightBracket,
					Location: 9,
					Length:   1,
				},
			}))

			itemType, ok := t.ItemType.(ast.NonNullType)
			Expect(ok).Should(BeTrue())
			Expect(itemType.TokenRange()).Should(MatchTokenRange(TokenRangeFields{
				First: TokenFields{
					Kind:     token.KindName,
					Location: 2,
					Length:   6,
					Value:    "MyType",
				},
				Last: TokenFields{
					Kind:     token.KindBang,
					Location: 8,
					Length:   1,
				},
			}))

			innermostType, ok := itemType.Type.(ast.NamedType)
			Expect(ok).Should(BeTrue())
			Expect(innermostType.TokenRange()).Should(MatchTokenRange(TokenRangeFields{
				First: TokenFields{
					Kind:     token.KindName,
					Location: 2,
					Length:   6,
					Value:    "MyType",
				},
				Last: TokenFields{
					Kind:     token.KindName,
					Location: 2,
					Length:   6,
					Value:    "MyType",
				},
			}))

			Expect(innermostType.Name).Should(MatchNameNode(TokenFields{
				Kind:     token.KindName,
				Location: 2,
				Length:   6,
				Value:    "MyType",
			}))
		})

		It("rejects incompleted list types", func() {
			_, err := parseType("[[[MyType]]")
			Expect(err).Should(testutil.MatchGraphQLError(
				testutil.MessageContainSubstring("Expected ], found <EOF>"),
				testutil.LocationEqual(graphql.ErrorLocation{
					Line:   1,
					Column: 12,
				}),
				testutil.KindIs(graphql.ErrKindSyntax),
			))
		})

		It("rejects list type without item type", func() {
			_, err := parseType("[]")
			Expect(err).Should(testutil.MatchGraphQLError(
				testutil.MessageContainSubstring("Expected Name, found ]"),
				testutil.LocationEqual(graphql.ErrorLocation{
					Line:   1,
					Column: 2,
				}),
				testutil.KindIs(graphql.ErrKindSyntax),
			))
		})

		It("rejects non-null type without item type", func() {
			_, err := parseType("!")
			Expect(err).Should(testutil.MatchGraphQLError(
				testutil.MessageContainSubstring("Expected Name, found !"),
				testutil.LocationEqual(graphql.ErrorLocation{
					Line:   1,
					Column: 1,
				}),
				testutil.KindIs(graphql.ErrKindSyntax),
			))
		})

		It("rejects non-null type with non-null item type", func() {
			_, err := parseType("MyType!!")
			Expect(err).Should(testutil.MatchGraphQLError(
				testutil.MessageContainSubstring("Expected <EOF>, found !"),
				testutil.LocationEqual(graphql.ErrorLocation{
					Line:   1,
					Column: 8,
				}),
				testutil.KindIs(graphql.ErrKindSyntax),
			))

			_, err = parseType("[[MyType!]!!]")
			Expect(err).Should(testutil.MatchGraphQLError(
				testutil.MessageContainSubstring("Expected ], found !"),
				testutil.LocationEqual(graphql.ErrorLocation{
					Line:   1,
					Column: 12,
				}),
				testutil.KindIs(graphql.ErrKindSyntax),
			))
		})
	})

	Measure("parses query with 10k field selection", func(b Benchmarker) {
		var query bytes.Buffer
		query.WriteString("{")
		for i := 0; i < 10000; i++ {
			query.WriteString(fmt.Sprintf(" field%d", i))
		}
		query.WriteString("}")

		source := token.NewSource(&token.SourceConfig{
			Body: token.SourceBody(query.Bytes()),
		})

		b.Time("parse time", func() {
			_, err := parser.Parse(source, parser.ParseOptions{})
			Expect(err).ShouldNot(HaveOccurred())
		})
	}, 10)
})
