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

package executor_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"

	"github.com/botobag/artemis/concurrent"
	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/graphql/ast"
	"github.com/botobag/artemis/graphql/executor"
	"github.com/botobag/artemis/graphql/parser"
	"github.com/botobag/artemis/graphql/token"
	"github.com/botobag/artemis/internal/testutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
)

func MatchResultInJSON(resultJSON string) types.GomegaMatcher {
	stringify := func(result executor.ExecutionResult) []byte {
		json, err := json.Marshal(&result)
		Expect(err).ShouldNot(HaveOccurred())
		return json
	}
	return Receive(WithTransform(stringify, MatchJSON(resultJSON)))
}

func WithWorkerPool(config *concurrent.WorkerPoolExecutorConfig) func() {
	return func() {
		// graphql-js/src/execution/__tests__/executor-test.js

		var runner concurrent.Executor

		BeforeEach(func() {
			if config != nil {
				var err error
				runner, err = concurrent.NewWorkerPoolExecutor(*config)
				Expect(err).ShouldNot(HaveOccurred())
			}
		})

		AfterEach(func() {
			if runner != nil {
				terminated, err := runner.Shutdown()
				Expect(err).ShouldNot(HaveOccurred())
				Eventually(terminated).Should(Receive(BeTrue()))
			}
		})

		// ast.Document is not a pointer and can never be nil.
		// It("throws if no document is provided", func() {
		// })

		It("throws if no schema is provided", func() {
			// TODO: #71
		})

		It("accepts positional arguments", func() {
			document, err := parser.Parse(token.NewSource(&token.SourceConfig{
				Body: token.SourceBody([]byte("{ a }")),
			}), parser.ParseOptions{})
			Expect(err).ShouldNot(HaveOccurred())

			schema, err := graphql.NewSchema(&graphql.SchemaConfig{
				Query: graphql.MustNewObject(&graphql.ObjectConfig{
					Name: "Type",
					Fields: graphql.Fields{
						"a": {
							Type: graphql.T(graphql.String()),
							Resolver: graphql.FieldResolverFunc(func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
								return info.RootValue(), nil
							}),
						},
					},
				}),
			})
			Expect(err).ShouldNot(HaveOccurred())

			operation, errs := executor.Prepare(executor.PrepareParams{
				Schema:   schema,
				Document: document,
			})
			Expect(errs.HaveOccurred()).ShouldNot(BeTrue())

			result := operation.Execute(context.Background(), executor.ExecuteParams{
				Runner:    runner,
				RootValue: "rootValue",
			})

			Eventually(result).Should(MatchResultInJSON(`{
      "data": { "a": "rootValue" }
    }`))
		})

		It("executes arbitrary code", func() {
			var (
				data     interface{}
				deepData interface{}
			)

			data = &struct {
				A       func(ctx context.Context) (interface{}, error)
				B       func(ctx context.Context) (interface{}, error)
				C       func(ctx context.Context) (interface{}, error)
				D       func(ctx context.Context) (interface{}, error)
				E       func(ctx context.Context) (interface{}, error)
				F       func(ctx context.Context) (interface{}, error)
				Pic     func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error)
				Deep    func(ctx context.Context) (interface{}, error)
				Promise func(ctx context.Context) (interface{}, error)
			}{
				A: func(ctx context.Context) (interface{}, error) {
					return "Apple", nil
				},
				B: func(ctx context.Context) (interface{}, error) {
					return "Banana", nil
				},
				C: func(ctx context.Context) (interface{}, error) {
					return "Cookie", nil
				},
				D: func(ctx context.Context) (interface{}, error) {
					return "Donut", nil
				},
				E: func(ctx context.Context) (interface{}, error) {
					return "Egg", nil
				},
				F: func(ctx context.Context) (interface{}, error) {
					return "Fish", nil
				},
				Pic: func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
					arg := info.ArgumentValues().Get("size")
					if arg == nil {
						arg = 50
					}
					return fmt.Sprintf("Pic of size: %v", arg), nil
				},
				Deep: func(ctx context.Context) (interface{}, error) {
					return deepData, nil
				},
				Promise: func(ctx context.Context) (interface{}, error) {
					return data, nil
				},
			}

			deepData = &struct {
				A      func(ctx context.Context) (interface{}, error)
				B      func(ctx context.Context) (interface{}, error)
				C      func(ctx context.Context) (interface{}, error)
				Deeper func(ctx context.Context) (interface{}, error)
			}{
				A: func(ctx context.Context) (interface{}, error) {
					return "Already Been Done", nil
				},
				B: func(ctx context.Context) (interface{}, error) {
					return "Boring", nil
				},
				C: func(ctx context.Context) (interface{}, error) {
					return []interface{}{
						"Contrived",
						nil,
						"Confusing",
					}, nil
				},
				Deeper: func(ctx context.Context) (interface{}, error) {
					return []interface{}{
						data,
						nil,
						data,
					}, nil
				},
			}

			document, err := parser.Parse(token.NewSource(&token.SourceConfig{
				Body: token.SourceBody([]byte(`
      query ($size: Int) {
        a,
        b,
        x: c
        ...c
        f
        ...on DataType {
          pic(size: $size)
          promise {
            a
          }
        }
        deep {
          a
          b
          c
          deeper {
            a
            b
          }
        }
      }

      fragment c on DataType {
        d
        e
      }
    `))}), parser.ParseOptions{})
			Expect(err).ShouldNot(HaveOccurred())

			var (
				dataTypeDef *graphql.ObjectConfig = &graphql.ObjectConfig{
					Name: "DataType",
				}
				deepDataTypeDef = &graphql.ObjectConfig{
					Name: "DeepDataType",
				}
			)

			dataTypeDef.Fields = graphql.Fields{
				"a": {
					Type: graphql.T(graphql.String()),
				},
				"b": {
					Type: graphql.T(graphql.String()),
				},
				"c": {
					Type: graphql.T(graphql.String()),
				},
				"d": {
					Type: graphql.T(graphql.String()),
				},
				"e": {
					Type: graphql.T(graphql.String()),
				},
				"f": {
					Type: graphql.T(graphql.String()),
				},
				"pic": {
					Type: graphql.T(graphql.String()),
					Args: graphql.ArgumentConfigMap{
						"size": {
							Type: graphql.T(graphql.Int()),
						},
					},
				},
				"deep": {
					Type: deepDataTypeDef,
				},
				// FIXME: #70
				"promise": {
					Type: dataTypeDef,
				},
			}

			deepDataTypeDef.Fields = graphql.Fields{
				"a": {
					Type: graphql.T(graphql.String()),
				},
				"b": {
					Type: graphql.T(graphql.String()),
				},
				"c": {
					Type: graphql.ListOfType(graphql.String()),
				},
				"deeper": {
					Type: graphql.ListOf(dataTypeDef),
				},
			}

			dataType, err := graphql.NewObject(dataTypeDef)
			Expect(err).ShouldNot(HaveOccurred())

			schema, err := graphql.NewSchema(&graphql.SchemaConfig{
				Query: dataType,
			})
			Expect(err).ShouldNot(HaveOccurred())

			operation, errs := executor.Prepare(executor.PrepareParams{
				Schema:   schema,
				Document: document,
			})
			Expect(errs.HaveOccurred()).ShouldNot(BeTrue())

			result := operation.Execute(context.Background(), executor.ExecuteParams{
				Runner:    runner,
				RootValue: data,
				VariableValues: map[string]interface{}{
					"size": 100,
				},
			})

			Eventually(result).Should(MatchResultInJSON(`{
			"data": {
				"a": "Apple",
				"b": "Banana",
				"x": "Cookie",
				"d": "Donut",
				"e": "Egg",
				"f": "Fish",
				"pic": "Pic of size: 100",
				"promise": {
					"a": "Apple"
				},
				"deep": {
					"a": "Already Been Done",
					"b": "Boring",
					"c": [
						"Contrived",
						null,
						"Confusing"
					],
					"deeper": [
						{
							"a": "Apple",
							"b": "Banana"
						},
						null,
						{
							"a": "Apple",
							"b": "Banana"
						}
					]
				}
			}
		}`))
		})

		It("merges parallel fragments", func() {
			document, err := parser.Parse(token.NewSource(&token.SourceConfig{
				Body: token.SourceBody([]byte(`
				{ a, ...FragOne, ...FragTwo }

				fragment FragOne on Type {
					b
					deep { b, deeper: deep { b } }
				}

				fragment FragTwo on Type {
					c
					deep { c, deeper: deep { c } }
				}
    `))}), parser.ParseOptions{})
			Expect(err).ShouldNot(HaveOccurred())

			typeDef := &graphql.ObjectConfig{
				Name: "Type",
			}

			typeDef.Fields = graphql.Fields{
				"a": {
					Type: graphql.T(graphql.String()),
					Resolver: graphql.FieldResolverFunc(func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
						return "Apple", nil
					}),
				},
				"b": {
					Type: graphql.T(graphql.String()),
					Resolver: graphql.FieldResolverFunc(func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
						return "Banana", nil
					}),
				},
				"c": {
					Type: graphql.T(graphql.String()),
					Resolver: graphql.FieldResolverFunc(func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
						return "Cherry", nil
					}),
				},
				"deep": {
					Type: typeDef,
					Resolver: graphql.FieldResolverFunc(func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
						return map[string]string{}, nil
					}),
				},
			}

			queryType, err := graphql.NewObject(typeDef)
			Expect(err).ShouldNot(HaveOccurred())

			schema, err := graphql.NewSchema(&graphql.SchemaConfig{
				Query: queryType,
			})
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
				"a": "Apple",
				"b": "Banana",
				"deep": {
					"b": "Banana",
					"deeper": {
						"b": "Banana",
						"c": "Cherry"
					},
					"c": "Cherry"
				},
				"c": "Cherry"
			}
		}`))
		})

		It("provides info about current execution state", func() {
			document, err := parser.Parse(token.NewSource(&token.SourceConfig{
				Body: token.SourceBody([]byte(`query ($var: String) { result: test }`))}), parser.ParseOptions{})
			Expect(err).ShouldNot(HaveOccurred())

			var resolvedInfo graphql.ResolveInfo

			queryType, err := graphql.NewObject(&graphql.ObjectConfig{
				Name: "Test",
				Fields: graphql.Fields{
					"test": {
						Type: graphql.T(graphql.String()),
						Resolver: graphql.FieldResolverFunc(func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
							resolvedInfo = info
							return nil, nil
						}),
					},
				},
			})
			Expect(err).ShouldNot(HaveOccurred())

			schema, err := graphql.NewSchema(&graphql.SchemaConfig{
				Query: queryType,
			})
			Expect(err).ShouldNot(HaveOccurred())

			operation, errs := executor.Prepare(executor.PrepareParams{
				Schema:   schema,
				Document: document,
			})
			Expect(errs.HaveOccurred()).ShouldNot(BeTrue())

			rootValue := map[string]interface{}{
				"root": "val",
			}
			result := operation.Execute(context.Background(), executor.ExecuteParams{
				Runner:    runner,
				RootValue: rootValue,
				VariableValues: map[string]interface{}{
					"var": "abc",
				},
			})
			Eventually(result).Should(Receive(MatchFields(IgnoreExtras, Fields{
				"Errors": Equal(graphql.NoErrors()),
			})))

			Expect(resolvedInfo.Field().Name()).Should(Equal("test"))
			Expect(len(resolvedInfo.FieldDefinitions())).Should(Equal(1))
			Expect(resolvedInfo.FieldDefinitions()[0]).Should(Equal(
				document.Definitions[0].(*ast.OperationDefinition).SelectionSet[0]))
			Expect(resolvedInfo.FieldDefinitions()[0]).Should(Equal(
				document.Definitions[0].(*ast.OperationDefinition).SelectionSet[0]))
			Expect(resolvedInfo.Field().Type()).Should(Equal(graphql.String()))
			Expect(resolvedInfo.Object()).Should(Equal(schema.Query()))
			Expect(resolvedInfo.Path().String()).Should(Equal(`result`))
			Expect(resolvedInfo.Schema()).Should(Equal(schema))
			Expect(resolvedInfo.RootValue()).Should(Equal(rootValue))
			Expect(resolvedInfo.Operation()).Should(Equal(document.Definitions[0]))
			Expect(resolvedInfo.VariableValues()).Should(
				Equal(graphql.NewVariableValues(map[string]interface{}{"var": "abc"})))
		})

		It("threads root value context correctly", func() {
			document, err := parser.Parse(token.NewSource(&token.SourceConfig{
				Body: token.SourceBody([]byte(`query Example { a }`))}), parser.ParseOptions{})
			Expect(err).ShouldNot(HaveOccurred())

			data := map[string]string{
				"contextThing": "thing",
			}

			var resolvedRootValue interface{}

			queryType, err := graphql.NewObject(&graphql.ObjectConfig{
				Name: "Type",
				Fields: graphql.Fields{
					"a": {
						Type: graphql.T(graphql.String()),
						Resolver: graphql.FieldResolverFunc(func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
							resolvedRootValue = info.RootValue()
							return nil, nil
						}),
					},
				},
			})
			Expect(err).ShouldNot(HaveOccurred())

			schema, err := graphql.NewSchema(&graphql.SchemaConfig{
				Query: queryType,
			})
			Expect(err).ShouldNot(HaveOccurred())

			operation, errs := executor.Prepare(executor.PrepareParams{
				Schema:   schema,
				Document: document,
			})
			Expect(errs.HaveOccurred()).ShouldNot(BeTrue())

			result := operation.Execute(context.Background(), executor.ExecuteParams{
				Runner:    runner,
				RootValue: data,
			})
			Eventually(result).Should(Receive(MatchFields(IgnoreExtras, Fields{
				"Errors": Equal(graphql.NoErrors()),
			})))

			Expect(resolvedRootValue).Should(HaveKeyWithValue("contextThing", "thing"))
		})

		It("correctly threads arguments", func() {
			document, err := parser.Parse(token.NewSource(&token.SourceConfig{
				Body: token.SourceBody([]byte(`
				query Example {
					b(numArg: 123, stringArg: "foo")
				}
    `))}), parser.ParseOptions{})
			Expect(err).ShouldNot(HaveOccurred())

			var resolvedArgs graphql.ArgumentValues

			queryType, err := graphql.NewObject(&graphql.ObjectConfig{
				Name: "Type",
				Fields: graphql.Fields{
					"b": {
						Type: graphql.T(graphql.String()),
						Args: graphql.ArgumentConfigMap{
							"numArg": {
								Type: graphql.T(graphql.Int()),
							},
							"stringArg": {
								Type: graphql.T(graphql.String()),
							},
						},
						Resolver: graphql.FieldResolverFunc(func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
							resolvedArgs = info.ArgumentValues()
							return nil, nil
						}),
					},
				},
			})
			Expect(err).ShouldNot(HaveOccurred())

			schema, err := graphql.NewSchema(&graphql.SchemaConfig{
				Query: queryType,
			})
			Expect(err).ShouldNot(HaveOccurred())

			operation, errs := executor.Prepare(executor.PrepareParams{
				Schema:   schema,
				Document: document,
			})
			Expect(errs.HaveOccurred()).ShouldNot(BeTrue())

			result := operation.Execute(context.Background(), executor.ExecuteParams{
				Runner: runner,
			})
			Eventually(result).Should(Receive(MatchFields(IgnoreExtras, Fields{
				"Errors": Equal(graphql.NoErrors()),
			})))

			Expect(resolvedArgs.Get("numArg")).Should(Equal(123))
			Expect(resolvedArgs.Get("stringArg")).Should(Equal("foo"))
		})

		It("nulls out error subtrees", func() {
			// TODO: #70
			document, err := parser.Parse(token.NewSource(&token.SourceConfig{
				Body: token.SourceBody([]byte(`{
      sync
      syncError
      syncRawError
      syncReturnError
      syncReturnErrorList
      syncReturnErrorWithExtensions
      # async
      # asyncReject
      # asyncRawReject
      # asyncEmptyReject
      # asyncError
      # asyncRawError
      # asyncReturnError
      # asyncReturnErrorWithExtensions
  }`))}), parser.ParseOptions{})
			Expect(err).ShouldNot(HaveOccurred())

			data := struct {
				Sync                          string
				SyncError                     func(ctx context.Context) (interface{}, error)
				SyncRawError                  func(ctx context.Context) (interface{}, error)
				SyncReturnError               error
				SyncReturnErrorList           []interface{}
				SyncReturnErrorWithExtensions error
			}{
				Sync: "sync",
				SyncError: func(ctx context.Context) (interface{}, error) {
					return nil, graphql.NewError("Error getting syncError")
				},
				SyncRawError: func(ctx context.Context) (interface{}, error) {
					return nil, errors.New("Error getting syncRawError")
				},
				SyncReturnError: graphql.NewError("Error getting syncReturnError"),
				SyncReturnErrorList: []interface{}{
					"sync0",
					graphql.NewError("Error getting syncReturnErrorList1"),
					"sync2",
					graphql.NewError("Error getting syncReturnErrorList3"),
				},
				SyncReturnErrorWithExtensions: graphql.NewError("Error getting syncReturnErrorWithExtensions", graphql.ErrorExtensions{"foo": "bar"}),
			}

			queryType, err := graphql.NewObject(&graphql.ObjectConfig{
				Name: "Type",
				Fields: graphql.Fields{
					"sync": {
						Type: graphql.T(graphql.String()),
					},
					"syncError": {
						Type: graphql.T(graphql.String()),
					},
					"syncRawError": {
						Type: graphql.T(graphql.String()),
					},
					"syncReturnError": {
						Type: graphql.T(graphql.String()),
					},
					"syncReturnErrorList": {
						Type: graphql.ListOfType(graphql.String()),
					},
					"syncReturnErrorWithExtensions": {
						Type: graphql.T(graphql.String()),
					},
				},
			})
			Expect(err).ShouldNot(HaveOccurred())

			schema, err := graphql.NewSchema(&graphql.SchemaConfig{
				Query: queryType,
			})
			Expect(err).ShouldNot(HaveOccurred())

			operation, errs := executor.Prepare(executor.PrepareParams{
				Schema:   schema,
				Document: document,
			})
			Expect(errs.HaveOccurred()).ShouldNot(BeTrue())

			result := operation.Execute(context.Background(), executor.ExecuteParams{
				Runner:    runner,
				RootValue: data,
			})
			Eventually(result).Should(MatchResultInJSON(`{
			"data": {
				"sync": "sync",
				"syncError": null,
				"syncRawError": null,
				"syncReturnError": null,
				"syncReturnErrorList": [
					"sync0",
					null,
					"sync2",
					null
				],
				"syncReturnErrorWithExtensions": null
			},
			"errors": [
				{
					"message": "Error getting syncError",
					"locations": [
						{
							"line": 3,
							"column": 7
						}
					],
					"path": [
						"syncError"
					]
				},
				{
					"message": "Error getting syncRawError",
					"locations": [
						{
							"line": 4,
							"column": 7
						}
					],
					"path": [
						"syncRawError"
					]
				},
				{
					"message": "Error getting syncReturnError",
					"locations": [
						{
							"line": 5,
							"column": 7
						}
					],
					"path": [
						"syncReturnError"
					]
				},
				{
					"message": "Error getting syncReturnErrorList1",
					"locations": [
						{
							"line": 6,
							"column": 7
						}
					],
					"path": [
						"syncReturnErrorList",
						1
					]
				},
				{
					"message": "Error getting syncReturnErrorList3",
					"locations": [
						{
							"line": 6,
							"column": 7
						}
					],
					"path": [
						"syncReturnErrorList",
						3
					]
				},
				{
					"message": "Error getting syncReturnErrorWithExtensions",
					"locations": [
						{
							"line": 7,
							"column": 7
						}
					],
					"path": [
						"syncReturnErrorWithExtensions"
					],
					"extensions": {
						"foo": "bar"
					}
				}
			]
		}`))
		})

		It("nulls error subtree for promise rejection", func() {
			// TODO: #70
		})

		It("outputs full response path included for non-nullable fields", func() {
			emptyObject := &struct{}{}
			a := &graphql.ObjectConfig{
				Name: "A",
			}

			a.Fields = graphql.Fields{
				"nullableA": {
					Type: a,
					Resolver: graphql.FieldResolverFunc(func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
						return emptyObject, nil
					}),
				},
				"nonNullA": {
					Type: graphql.NonNullOf(a),
					Resolver: graphql.FieldResolverFunc(func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
						return emptyObject, nil
					}),
				},
				"throws": {
					Type: graphql.NonNullOfType(graphql.String()),
					Resolver: graphql.FieldResolverFunc(func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
						return nil, graphql.NewError("Catch me if you can")
					}),
				},
			}

			queryType, err := graphql.NewObject(&graphql.ObjectConfig{
				Name: "query",
				Fields: graphql.Fields{
					"nullableA": {
						Type: a,
						Resolver: graphql.FieldResolverFunc(func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
							return emptyObject, nil
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
				Body: token.SourceBody([]byte(`
      query {
        nullableA {
          aliasedA: nullableA {
            nonNullA {
              anotherA: nonNullA {
                throws
              }
            }
          }
        }
      }
    `))}), parser.ParseOptions{})
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
				"nullableA": {
					"aliasedA": null
				}
			},
			"errors": [
				{
					"message": "Catch me if you can",
					"locations": [
						{
							"line": 7,
							"column": 17
						}
					],
					"path": [
						"nullableA",
						"aliasedA",
						"nonNullA",
						"anotherA",
						"throws"
					]
				}
			]
		}`))
		})

		It("uses the inline operation if no operation name is provided", func() {
			document, err := parser.Parse(token.NewSource(&token.SourceConfig{
				Body: token.SourceBody([]byte(`{ a }`))}), parser.ParseOptions{})
			Expect(err).ShouldNot(HaveOccurred())

			data := map[string]string{"a": "b"}

			queryType, err := graphql.NewObject(&graphql.ObjectConfig{
				Name: "Type",
				Fields: graphql.Fields{
					"a": {
						Type: graphql.T(graphql.String()),
					},
				},
			})
			Expect(err).ShouldNot(HaveOccurred())

			schema, err := graphql.NewSchema(&graphql.SchemaConfig{
				Query: queryType,
			})
			Expect(err).ShouldNot(HaveOccurred())

			operation, errs := executor.Prepare(executor.PrepareParams{
				Schema:   schema,
				Document: document,
			})
			Expect(errs.HaveOccurred()).ShouldNot(BeTrue())

			result := operation.Execute(context.Background(), executor.ExecuteParams{
				Runner:    runner,
				RootValue: data,
			})
			Eventually(result).Should(MatchResultInJSON(`{"data":{"a":"b"}}`))
		})

		It("uses the only operation if no operation name is provided", func() {
			document, err := parser.Parse(token.NewSource(&token.SourceConfig{
				Body: token.SourceBody([]byte(`query Example { a }`))}), parser.ParseOptions{})
			Expect(err).ShouldNot(HaveOccurred())

			data := map[string]string{"a": "b"}

			queryType, err := graphql.NewObject(&graphql.ObjectConfig{
				Name: "Type",
				Fields: graphql.Fields{
					"a": {
						Type: graphql.T(graphql.String()),
					},
				},
			})
			Expect(err).ShouldNot(HaveOccurred())

			schema, err := graphql.NewSchema(&graphql.SchemaConfig{
				Query: queryType,
			})
			Expect(err).ShouldNot(HaveOccurred())

			operation, errs := executor.Prepare(executor.PrepareParams{
				Schema:   schema,
				Document: document,
			})
			Expect(errs.HaveOccurred()).ShouldNot(BeTrue())

			result := operation.Execute(context.Background(), executor.ExecuteParams{
				Runner:    runner,
				RootValue: data,
			})
			Eventually(result).Should(MatchResultInJSON(`{"data":{"a":"b"}}`))
		})

		It("uses the named operation if operation name is provided", func() {
			document, err := parser.Parse(token.NewSource(
				&token.SourceConfig{
					Body: token.SourceBody([]byte(`query Example { first: a } query OtherExample { second: a }`))}),
				parser.ParseOptions{})
			Expect(err).ShouldNot(HaveOccurred())

			data := map[string]string{"a": "b"}

			queryType, err := graphql.NewObject(&graphql.ObjectConfig{
				Name: "Type",
				Fields: graphql.Fields{
					"a": {
						Type: graphql.T(graphql.String()),
					},
				},
			})
			Expect(err).ShouldNot(HaveOccurred())

			schema, err := graphql.NewSchema(&graphql.SchemaConfig{
				Query: queryType,
			})
			Expect(err).ShouldNot(HaveOccurred())

			operation, errs := executor.Prepare(executor.PrepareParams{
				Schema:        schema,
				Document:      document,
				OperationName: "OtherExample",
			})
			Expect(errs.HaveOccurred()).ShouldNot(BeTrue())

			result := operation.Execute(context.Background(), executor.ExecuteParams{
				Runner:    runner,
				RootValue: data,
			})
			Eventually(result).Should(MatchResultInJSON(`{"data":{"second":"b"}}`))
		})

		It("provides error if no operation is provided", func() {
			document, err := parser.Parse(token.NewSource(
				&token.SourceConfig{
					Body: token.SourceBody([]byte(`fragment Example on Type { a }`))}),
				parser.ParseOptions{})
			Expect(err).ShouldNot(HaveOccurred())

			queryType, err := graphql.NewObject(&graphql.ObjectConfig{
				Name: "Type",
				Fields: graphql.Fields{
					"a": {
						Type: graphql.T(graphql.String()),
					},
				},
			})
			Expect(err).ShouldNot(HaveOccurred())

			schema, err := graphql.NewSchema(&graphql.SchemaConfig{
				Query: queryType,
			})
			Expect(err).ShouldNot(HaveOccurred())

			_, errs := executor.Prepare(executor.PrepareParams{
				Schema:   schema,
				Document: document,
			})
			Expect(errs).Should(testutil.ConsistOfGraphQLErrors(testutil.MatchGraphQLError(
				testutil.MessageEqual("Must provide an operation."),
			)))
		})

		Describe("Provides error when erroneous operation name is provided to query with multiple operations", func() {
			var (
				document ast.Document
				schema   *graphql.Schema
			)

			BeforeEach(func() {
				var err error
				document, err = parser.Parse(token.NewSource(
					&token.SourceConfig{
						Body: token.SourceBody([]byte(`query Example { a } query OtherExample { a }`))}),
					parser.ParseOptions{})
				Expect(err).ShouldNot(HaveOccurred())

				queryType, err := graphql.NewObject(&graphql.ObjectConfig{
					Name: "Type",
					Fields: graphql.Fields{
						"a": {
							Type: graphql.T(graphql.String()),
						},
					},
				})
				Expect(err).ShouldNot(HaveOccurred())

				schema, err = graphql.NewSchema(&graphql.SchemaConfig{
					Query: queryType,
				})
				Expect(err).ShouldNot(HaveOccurred())
			})

			It("errors if no operation name is provided", func() {
				_, errs := executor.Prepare(executor.PrepareParams{
					Schema:   schema,
					Document: document,
				})
				Expect(errs).Should(testutil.ConsistOfGraphQLErrors(testutil.MatchGraphQLError(
					testutil.MessageEqual("Must provide operation name if query contains multiple operations."),
				)))
			})

			It("errors if unknown operation name is provided", func() {
				_, errs := executor.Prepare(executor.PrepareParams{
					Schema:        schema,
					Document:      document,
					OperationName: "UnknownExample",
				})
				Expect(errs).Should(testutil.ConsistOfGraphQLErrors(testutil.MatchGraphQLError(
					testutil.MessageEqual(`Unknown operation named "UnknownExample".`),
				)))
			})
		})

		Describe("Schema Operation Types", func() {
			var (
				document  ast.Document
				rootValue = map[string]string{
					"a": "b",
					"c": "d",
				}
				schema *graphql.Schema
			)

			BeforeEach(func() {
				var err error

				document, err = parser.Parse(token.NewSource(
					&token.SourceConfig{
						Body: token.SourceBody([]byte(`query Q { a } mutation M { c } subscription S { a }`))}),
					parser.ParseOptions{})
				Expect(err).ShouldNot(HaveOccurred())

				queryType, err := graphql.NewObject(&graphql.ObjectConfig{
					Name: "Q",
					Fields: graphql.Fields{
						"a": {
							Type: graphql.T(graphql.String()),
						},
					},
				})
				Expect(err).ShouldNot(HaveOccurred())

				mutationType, err := graphql.NewObject(&graphql.ObjectConfig{
					Name: "M",
					Fields: graphql.Fields{
						"c": {
							Type: graphql.T(graphql.String()),
						},
					},
				})
				Expect(err).ShouldNot(HaveOccurred())

				subscriptionType, err := graphql.NewObject(&graphql.ObjectConfig{
					Name: "S",
					Fields: graphql.Fields{
						"a": {
							Type: graphql.T(graphql.String()),
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

			It("uses the query schema for queries", func() {
				operation, errs := executor.Prepare(executor.PrepareParams{
					Schema:        schema,
					Document:      document,
					OperationName: "Q",
				})
				Expect(errs.HaveOccurred()).ShouldNot(BeTrue())

				result := operation.Execute(context.Background(), executor.ExecuteParams{
					Runner:    runner,
					RootValue: rootValue,
				})
				Eventually(result).Should(MatchResultInJSON(`{"data":{"a":"b"}}`))
			})

			It("uses the mutation schema for mutations", func() {
				operation, errs := executor.Prepare(executor.PrepareParams{
					Schema:        schema,
					Document:      document,
					OperationName: "M",
				})
				Expect(errs.HaveOccurred()).ShouldNot(BeTrue())

				result := operation.Execute(context.Background(), executor.ExecuteParams{
					Runner:    runner,
					RootValue: rootValue,
				})
				Eventually(result).Should(MatchResultInJSON(`{"data":{"c":"d"}}`))
			})

			It("uses the subscription schema for subscriptions", func() {
				operation, errs := executor.Prepare(executor.PrepareParams{
					Schema:        schema,
					Document:      document,
					OperationName: "S",
				})
				Expect(errs.HaveOccurred()).ShouldNot(BeTrue())

				result := operation.Execute(context.Background(), executor.ExecuteParams{
					Runner:    runner,
					RootValue: rootValue,
				})
				Eventually(result).Should(MatchResultInJSON(`{"data":{"a":"b"}}`))
			})
		})

		It("correct field ordering despite execution order", func() {
			// #70
		})

		It("avoids recursion", func() {
			document, err := parser.Parse(token.NewSource(&token.SourceConfig{
				Body: token.SourceBody([]byte(`
      {
        a
        ...Frag
        ...Frag
      }

      fragment Frag on Type {
        a,
        ...Frag
      }
    `))}), parser.ParseOptions{})
			Expect(err).ShouldNot(HaveOccurred())

			data := map[string]string{"a": "b"}

			queryType, err := graphql.NewObject(&graphql.ObjectConfig{
				Name: "Type",
				Fields: graphql.Fields{
					"a": {
						Type: graphql.T(graphql.String()),
					},
				},
			})
			Expect(err).ShouldNot(HaveOccurred())

			schema, err := graphql.NewSchema(&graphql.SchemaConfig{
				Query: queryType,
			})
			Expect(err).ShouldNot(HaveOccurred())

			operation, errs := executor.Prepare(executor.PrepareParams{
				Schema:   schema,
				Document: document,
			})
			Expect(errs.HaveOccurred()).ShouldNot(BeTrue())

			result := operation.Execute(context.Background(), executor.ExecuteParams{
				Runner:    runner,
				RootValue: data,
			})
			Eventually(result).Should(MatchResultInJSON(`{"data":{"a":"b"}}`))
		})

		It("does not include illegal fields in output", func() {
			document, err := parser.Parse(token.NewSource(&token.SourceConfig{
				Body: token.SourceBody([]byte(`{ thisIsIllegalDoNotIncludeMe }`))}),
				parser.ParseOptions{})
			Expect(err).ShouldNot(HaveOccurred())

			queryType, err := graphql.NewObject(&graphql.ObjectConfig{
				Name: "Q",
				Fields: graphql.Fields{
					"a": {
						Type: graphql.T(graphql.String()),
					},
				},
			})
			Expect(err).ShouldNot(HaveOccurred())

			mutationType, err := graphql.NewObject(&graphql.ObjectConfig{
				Name: "M",
				Fields: graphql.Fields{
					"c": {
						Type: graphql.T(graphql.String()),
					},
				},
			})
			Expect(err).ShouldNot(HaveOccurred())

			schema, err := graphql.NewSchema(&graphql.SchemaConfig{
				Query:    queryType,
				Mutation: mutationType,
			})
			Expect(err).ShouldNot(HaveOccurred())

			operation, errs := executor.Prepare(executor.PrepareParams{
				Schema:   schema,
				Document: document,
			})
			Expect(errs.HaveOccurred()).ShouldNot(BeTrue())

			result := operation.Execute(context.Background(), executor.ExecuteParams{
				Runner: runner,
			})
			Eventually(result).Should(MatchResultInJSON(`{"data":{}}`))
		})

		It("does not include arguments that were not set", func() {
			document, err := parser.Parse(token.NewSource(&token.SourceConfig{
				Body: token.SourceBody([]byte(`{ field(a: true, c: false, e: 0) }`))}), parser.ParseOptions{})
			Expect(err).ShouldNot(HaveOccurred())

			queryType, err := graphql.NewObject(&graphql.ObjectConfig{
				Name: "Type",
				Fields: graphql.Fields{
					"field": {
						Type: graphql.T(graphql.String()),
						Resolver: graphql.FieldResolverFunc(func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
							args, err := json.Marshal(info.ArgumentValues())
							return string(args), err
						}),
						Args: graphql.ArgumentConfigMap{
							"a": {
								Type: graphql.T(graphql.Boolean()),
							},
							"b": {
								Type: graphql.T(graphql.Boolean()),
							},
							"c": {
								Type: graphql.T(graphql.Boolean()),
							},
							"d": {
								Type: graphql.T(graphql.Int()),
							},
							"e": {
								Type: graphql.T(graphql.Int()),
							},
						},
					},
				},
			})
			Expect(err).ShouldNot(HaveOccurred())

			schema, err := graphql.NewSchema(&graphql.SchemaConfig{
				Query: queryType,
			})
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
				"field": "{\"a\":true,\"c\":false,\"e\":0}"
			}
		}`))
		})

		// We don't have isTypeOf.
		// It("fails when an isTypeOf check is not met", func() {
		// })

		It("executes ignoring invalid non-executable definitions", func() {
			// TODO: #71
		})

		It("uses a custom field resolver", func() {
			document, err := parser.Parse(token.NewSource(&token.SourceConfig{
				Body: token.SourceBody([]byte(`{ foo }`))}), parser.ParseOptions{})
			Expect(err).ShouldNot(HaveOccurred())

			queryType, err := graphql.NewObject(&graphql.ObjectConfig{
				Name: "Query",
				Fields: graphql.Fields{
					"foo": {
						Type: graphql.T(graphql.String()),
					},
				},
			})
			Expect(err).ShouldNot(HaveOccurred())

			schema, err := graphql.NewSchema(&graphql.SchemaConfig{
				Query: queryType,
			})
			Expect(err).ShouldNot(HaveOccurred())

			// For the purposes of test, just return the name of the field!
			fieldResolver := graphql.FieldResolverFunc(func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
				return info.Field().Name(), nil
			})

			operation, errs := executor.Prepare(executor.PrepareParams{
				Schema:               schema,
				Document:             document,
				DefaultFieldResolver: fieldResolver,
			})
			Expect(errs.HaveOccurred()).ShouldNot(BeTrue())

			result := operation.Execute(context.Background(), executor.ExecuteParams{
				Runner: runner,
			})
			Eventually(result).Should(MatchResultInJSON(`{"data":{"foo":"foo"}}`))
		})
	}
} // WithRunner

var _ = Describe("Execute: Handles basic execution tasks", func() {
	Context("without concurrent runner", WithWorkerPool(nil))
	Context("with concurrent runner", WithWorkerPool(&concurrent.WorkerPoolExecutorConfig{
		MaxPoolSize: uint32(runtime.GOMAXPROCS(-1)),
	}))
})
