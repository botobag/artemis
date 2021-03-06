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

	"github.com/botobag/artemis/concurrent/future"
	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/graphql/ast"
	"github.com/botobag/artemis/graphql/executor"
	"github.com/botobag/artemis/graphql/parser"
	"github.com/botobag/artemis/graphql/token"
	"github.com/botobag/artemis/internal/testutil"
	"github.com/botobag/artemis/iterator"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

type ReadyOnNextPollFuture struct {
	data   interface{}
	polled bool
}

func (f *ReadyOnNextPollFuture) Poll(waker future.Waker) (future.PollResult, error) {
	if !f.polled {
		// f has not been polled. Return future.PollResultPending to indicate the Future is not ready
		// yet and notify waker to poll again.
		f.polled = true
		if err := waker.Wake(); err != nil {
			return nil, err
		}
		return future.PollResultPending, nil
	}
	return f.data, nil
}

type TestIterable struct {
	values []interface{}
}

func (iter *TestIterable) Iterator() graphql.Iterator {
	return iter
}

func (iter *TestIterable) Next() (interface{}, error) {
	if len(iter.values) == 0 {
		return nil, iterator.Done
	}

	// Pop one value.
	value := iter.values[0]
	iter.values = iter.values[1:]
	if err, ok := value.(error); ok {
		return nil, err
	}
	return value, nil
}

type SizedTestIterable struct {
	TestIterable
}

func (iter *SizedTestIterable) Size() int {
	return len(iter.values)
}

// graphql-js/src/execution/__tests__/executor-test.js
var _ = Describe("Execute: Handles basic execution tasks", func() {
	// ast.Document is not a pointer and can never be nil.
	// It("throws if no document is provided", func() {
	// })

	It("throws if no schema is provided", func() {
		// TODO: #192
	})

	It("accepts positional arguments", func() {
		document := parser.MustParse(token.NewSource("{ a }"))

		schema := graphql.MustNewSchema(&graphql.SchemaConfig{
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

		result := execute(schema, document, executor.RootValue("rootValue"))
		Expect(result).Should(MatchResultInJSON(`{
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
				arg := info.Args().Get("size")
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

		document := parser.MustParse(token.NewSource(`
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
    `))

		var (
			dataType *graphql.ObjectConfig = &graphql.ObjectConfig{
				Name: "DataType",
			}
			deepDataType = &graphql.ObjectConfig{
				Name: "DeepDataType",
			}
		)

		dataType.Fields = graphql.Fields{
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
				Type: deepDataType,
			},
			"promise": {
				Type: dataType,
				Resolver: graphql.FieldResolverFunc(func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
					return &ReadyOnNextPollFuture{
						data: data,
					}, nil
				}),
			},
		}

		deepDataType.Fields = graphql.Fields{
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
				Type: graphql.ListOf(dataType),
			},
		}

		schema := graphql.MustNewSchema(&graphql.SchemaConfig{
			Query: graphql.MustNewObject(dataType),
		})

		result := execute(
			schema,
			document,
			executor.RootValue(data),
			executor.VariableValues(map[string]interface{}{
				"size": 100,
			}))
		Expect(result).Should(MatchResultInJSON(`{
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
		document := parser.MustParse(token.NewSource(`
				{ a, ...FragOne, ...FragTwo }

				fragment FragOne on Type {
					b
					deep { b, deeper: deep { b } }
				}

				fragment FragTwo on Type {
					c
					deep { c, deeper: deep { c } }
				}
    `))

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

		queryType := graphql.MustNewObject(typeDef)

		schema := graphql.MustNewSchema(&graphql.SchemaConfig{
			Query: queryType,
		})

		Expect(execute(schema, document)).Should(MatchResultInJSON(`{
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
		document := parser.MustParse(token.NewSource(`query ($var: String) { result: test }`))

		var resolvedInfo graphql.ResolveInfo

		queryType := graphql.MustNewObject(&graphql.ObjectConfig{
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

		schema := graphql.MustNewSchema(&graphql.SchemaConfig{
			Query: queryType,
		})

		rootValue := map[string]interface{}{
			"root": "val",
		}
		result := execute(
			schema,
			document,
			executor.RootValue(rootValue),
			executor.VariableValues(map[string]interface{}{
				"var": "abc",
			}))
		Expect(result).Should(PointTo(MatchFields(IgnoreExtras, Fields{
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
		document := parser.MustParse(token.NewSource(`query Example { a }`))

		data := map[string]string{
			"contextThing": "thing",
		}

		var resolvedRootValue interface{}

		queryType := graphql.MustNewObject(&graphql.ObjectConfig{
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

		schema := graphql.MustNewSchema(&graphql.SchemaConfig{
			Query: queryType,
		})

		result := execute(schema, document, executor.RootValue(data))
		Expect(result).Should(PointTo(MatchFields(IgnoreExtras, Fields{
			"Errors": Equal(graphql.NoErrors()),
		})))

		Expect(resolvedRootValue).Should(HaveKeyWithValue("contextThing", "thing"))
	})

	It("correctly threads arguments", func() {
		document := parser.MustParse(token.NewSource(`
				query Example {
					b(numArg: 123, stringArg: "foo")
				}
    `))

		var resolvedArgs graphql.ArgumentValues

		queryType := graphql.MustNewObject(&graphql.ObjectConfig{
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
						resolvedArgs = info.Args()
						return nil, nil
					}),
				},
			},
		})

		schema := graphql.MustNewSchema(&graphql.SchemaConfig{
			Query: queryType,
		})

		Expect(execute(schema, document)).Should(PointTo(MatchFields(IgnoreExtras, Fields{
			"Errors": Equal(graphql.NoErrors()),
		})))

		Expect(resolvedArgs.Get("numArg")).Should(Equal(123))
		Expect(resolvedArgs.Get("stringArg")).Should(Equal("foo"))
	})

	It("nulls out error subtrees", func() {
		document := parser.MustParse(token.NewSource(`
      {
        sync
        syncError
        syncRawError
        syncReturnError
        syncReturnErrorList
        async
        asyncReject
        asyncRawReject
        asyncEmptyReject
        asyncError
        asyncRawError
        asyncReturnError
        asyncReturnErrorWithExtensions
      }`))

		data := struct {
			Sync                string
			SyncError           func(ctx context.Context) (interface{}, error)
			SyncRawError        func(ctx context.Context) (interface{}, error)
			SyncReturnError     error
			SyncReturnErrorList []interface{}

			Async                          future.Future
			AsyncReject                    future.Future
			AsyncRawReject                 future.Future
			AsyncEmptyReject               future.Future
			AsyncError                     future.Future
			AsyncRawError                  future.Future
			AsyncReturnError               future.Future
			AsyncReturnErrorWithExtensions future.Future
		}{
			//                      graphql-js                            Artemis
			//                      ===================================== ======================================================
			// SyncError:           throw new Error                       return nil, graphql.NewError
			// SyncRawError:        throw                                 return nil, errors.New
			// SyncReturnError:     return new Error                      return graphql.NewError, nil
			// SyncReturnErrorList: return {<containing new Error>}       return []interface{<containing graphql.NewError>}, nil
			// AsyncReject:         return Promise.reject(new Error)      return future.Err(graphql.NewError)
			// AsyncRawReject:      return Promise.reject                 return future.Err(errors.New)
			// AsyncEmptyReject:    return Promise.reject()               return future.Err(nil)
			// AsyncError:          return Promise(() => throw new Error) return future.Err(graphql.NewError)
			// AsyncRawError:       return Promise(() => throw)           return future.Err(errors.New)
			// AsyncReturnError:    return Promise.resolve(new Error)     return future.Ready(graphql.NewError)

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

			Async:            future.Ready("async"),
			AsyncReject:      future.Err(graphql.NewError("Error getting asyncReject")),
			AsyncRawReject:   future.Err(errors.New("Error getting asyncRawReject")),
			AsyncEmptyReject: future.Err(nil),
			AsyncError:       future.Err(graphql.NewError("Error getting asyncError")),
			AsyncRawError:    future.Err(errors.New("Error getting asyncRawError")),
			AsyncReturnError: future.Ready(graphql.NewError("Error getting asyncReturnError")),
			AsyncReturnErrorWithExtensions: future.Ready(graphql.NewError("Error getting asyncReturnErrorWithExtensions", graphql.ErrorExtensions{
				"foo": "bar",
			})),
		}

		queryType := graphql.MustNewObject(&graphql.ObjectConfig{
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
				"async": {
					Type: graphql.T(graphql.String()),
				},
				"asyncReject": {
					Type: graphql.T(graphql.String()),
				},
				"asyncRawReject": {
					Type: graphql.T(graphql.String()),
				},
				"asyncEmptyReject": {
					Type: graphql.T(graphql.String()),
				},
				"asyncError": {
					Type: graphql.T(graphql.String()),
				},
				"asyncRawError": {
					Type: graphql.T(graphql.String()),
				},
				"asyncReturnError": {
					Type: graphql.ListOfType(graphql.String()),
				},
				"asyncReturnErrorWithExtensions": {
					Type: graphql.T(graphql.String()),
				},
			},
		})

		schema := graphql.MustNewSchema(&graphql.SchemaConfig{
			Query: queryType,
		})

		result := execute(schema, document, executor.RootValue(data))
		Expect(result).Should(MatchResultInJSON(`{
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
				"async": "async",
				"asyncReject": null,
				"asyncRawReject": null,
				"asyncEmptyReject": null,
				"asyncError": null,
				"asyncRawError": null,
				"asyncReturnError": null,
				"asyncReturnErrorWithExtensions": null
			},
			"errors": [
				{
					"message": "Error getting syncError",
					"locations": [
						{
							"line": 4,
							"column": 9
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
							"line": 5,
							"column": 9
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
							"line": 6,
							"column": 9
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
							"line": 7,
							"column": 9
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
							"line": 7,
							"column": 9
						}
					],
					"path": [
						"syncReturnErrorList",
						3
					]
				},
				{
					"message": "Error getting asyncReject",
					"locations": [
						{
							"line": 9,
							"column": 9
						}
					],
					"path": [
						"asyncReject"
					]
				},
				{
					"message": "Error getting asyncRawReject",
					"locations": [
						{
							"line": 10,
							"column": 9
						}
					],
					"path": [
						"asyncRawReject"
					]
				},
				{
					"message": "",
					"locations": [
						{
							"line": 11,
							"column": 9
						}
					],
					"path": [
						"asyncEmptyReject"
					]
				},
				{
					"message": "Error getting asyncError",
					"locations": [
						{
							"line": 12,
							"column": 9
						}
					],
					"path": [
						"asyncError"
					]
				},
				{
					"message": "Error getting asyncRawError",
					"locations": [
						{
							"line": 13,
							"column": 9
						}
					],
					"path": [
						"asyncRawError"
					]
				},
				{
					"message": "Error getting asyncReturnError",
					"locations": [
						{
							"line": 14,
							"column": 9
						}
					],
					"path": [
						"asyncReturnError"
					]
				},
				{
					"message": "Error getting asyncReturnErrorWithExtensions",
					"locations": [
						{
							"line": 15,
							"column": 9
						}
					],
					"path": [
						"asyncReturnErrorWithExtensions"
					],
					"extensions": {
						"foo": "bar"
					}
				}
			]
		}`))
	})

	It("nulls error subtree for promise rejection", func() {
		queryType := graphql.MustNewObject(&graphql.ObjectConfig{
			Name: "Query",
			Fields: graphql.Fields{
				"foods": {
					Type: graphql.ListOf(&graphql.ObjectConfig{
						Name: "Food",
						Fields: graphql.Fields{
							"name": {
								Type: graphql.T(graphql.String()),
							},
						},
					}),
					Resolver: graphql.FieldResolverFunc(func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
						return future.Err(graphql.NewError("Dangit")), nil
					}),
				},
			},
		})

		schema := graphql.MustNewSchema(&graphql.SchemaConfig{
			Query: queryType,
		})

		document := parser.MustParse(token.NewSource(`
      query {
        foods {
          name
        }
      }
    `))

		Expect(execute(schema, document)).Should(MatchResultInJSON(`{
				"data": { "foods": null },
				"errors": [
					{
						"locations": [{ "column": 9, "line": 3 }],
						"message": "Dangit",
						"path": ["foods"]
					}
				]
			}`))
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

		queryType := graphql.MustNewObject(&graphql.ObjectConfig{
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

		schema := graphql.MustNewSchema(&graphql.SchemaConfig{
			Query: queryType,
		})

		document := parser.MustParse(token.NewSource(`
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
    `))

		Expect(execute(schema, document)).Should(MatchResultInJSON(`{
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
		document := parser.MustParse(token.NewSource(`{ a }`))

		data := map[string]string{"a": "b"}

		queryType := graphql.MustNewObject(&graphql.ObjectConfig{
			Name: "Type",
			Fields: graphql.Fields{
				"a": {
					Type: graphql.T(graphql.String()),
				},
			},
		})

		schema := graphql.MustNewSchema(&graphql.SchemaConfig{
			Query: queryType,
		})

		result := execute(schema, document, executor.RootValue(data))
		Expect(result).Should(MatchResultInJSON(`{"data":{"a":"b"}}`))
	})

	It("uses the only operation if no operation name is provided", func() {
		document := parser.MustParse(token.NewSource(`query Example { a }`))

		data := map[string]string{"a": "b"}

		queryType := graphql.MustNewObject(&graphql.ObjectConfig{
			Name: "Type",
			Fields: graphql.Fields{
				"a": {
					Type: graphql.T(graphql.String()),
				},
			},
		})

		schema := graphql.MustNewSchema(&graphql.SchemaConfig{
			Query: queryType,
		})

		result := execute(schema, document, executor.RootValue(data))
		Expect(result).Should(MatchResultInJSON(`{"data":{"a":"b"}}`))
	})

	It("uses the named operation if operation name is provided", func() {
		document := parser.MustParse(token.NewSource(
			`query Example { first: a } query OtherExample { second: a }`))

		data := map[string]string{"a": "b"}

		queryType := graphql.MustNewObject(&graphql.ObjectConfig{
			Name: "Type",
			Fields: graphql.Fields{
				"a": {
					Type: graphql.T(graphql.String()),
				},
			},
		})

		schema := graphql.MustNewSchema(&graphql.SchemaConfig{
			Query: queryType,
		})

		result := execute(
			schema,
			document,
			executor.OperationName("OtherExample"),
			executor.RootValue(data))
		Expect(result).Should(MatchResultInJSON(`{"data":{"second":"b"}}`))
	})

	It("provides error if no operation is provided", func() {
		document := parser.MustParse(token.NewSource(`fragment Example on Type { a }`))

		queryType := graphql.MustNewObject(&graphql.ObjectConfig{
			Name: "Type",
			Fields: graphql.Fields{
				"a": {
					Type: graphql.T(graphql.String()),
				},
			},
		})

		schema := graphql.MustNewSchema(&graphql.SchemaConfig{
			Query: queryType,
		})

		_, errs := executor.Prepare(schema, document, executor.WithoutValidation())
		Expect(errs).Should(testutil.ConsistOfGraphQLErrors(testutil.MatchGraphQLError(
			testutil.MessageEqual("Must provide an operation."),
		)))

		Expect(func() {
			executor.MustPrepare(schema, document, executor.WithoutValidation())
		}).Should(Panic())
	})

	Describe("Provides error when erroneous operation name is provided to query with multiple operations", func() {
		var (
			document ast.Document
			schema   graphql.Schema
		)

		BeforeEach(func() {
			document = parser.MustParse(token.NewSource(`query Example { a } query OtherExample { a }`))

			queryType := graphql.MustNewObject(&graphql.ObjectConfig{
				Name: "Type",
				Fields: graphql.Fields{
					"a": {
						Type: graphql.T(graphql.String()),
					},
				},
			})

			schema = graphql.MustNewSchema(&graphql.SchemaConfig{
				Query: queryType,
			})
		})

		It("errors if no operation name is provided", func() {
			_, errs := executor.Prepare(schema, document, executor.WithoutValidation())
			Expect(errs).Should(testutil.ConsistOfGraphQLErrors(testutil.MatchGraphQLError(
				testutil.MessageEqual("Must provide operation name if query contains multiple operations."),
			)))

			Expect(func() {
				executor.MustPrepare(schema, document, executor.WithoutValidation())
			}).Should(Panic())
		})

		It("errors if unknown operation name is provided", func() {
			_, errs := executor.Prepare(
				schema,
				document,
				executor.WithoutValidation(),
				executor.OperationName("UnknownExample"))
			Expect(errs).Should(testutil.ConsistOfGraphQLErrors(testutil.MatchGraphQLError(
				testutil.MessageEqual(`Unknown operation named "UnknownExample".`),
			)))

			Expect(func() {
				executor.MustPrepare(
					schema,
					document,
					executor.WithoutValidation(),
					executor.OperationName("UnknownExample"))
			}).Should(Panic())
		})
	})

	Describe("Schema Operation Types", func() {
		var (
			document  ast.Document
			rootValue = map[string]string{
				"a": "b",
				"c": "d",
			}
			schema graphql.Schema
		)

		BeforeEach(func() {
			document = parser.MustParse(token.NewSource(
				`query Q { a } mutation M { c } subscription S { a }`))

			queryType := graphql.MustNewObject(&graphql.ObjectConfig{
				Name: "Q",
				Fields: graphql.Fields{
					"a": {
						Type: graphql.T(graphql.String()),
					},
				},
			})

			mutationType := graphql.MustNewObject(&graphql.ObjectConfig{
				Name: "M",
				Fields: graphql.Fields{
					"c": {
						Type: graphql.T(graphql.String()),
					},
				},
			})

			subscriptionType := graphql.MustNewObject(&graphql.ObjectConfig{
				Name: "S",
				Fields: graphql.Fields{
					"a": {
						Type: graphql.T(graphql.String()),
					},
				},
			})

			schema = graphql.MustNewSchema(&graphql.SchemaConfig{
				Query:        queryType,
				Mutation:     mutationType,
				Subscription: subscriptionType,
			})
		})

		It("uses the query schema for queries", func() {
			result := execute(
				schema,
				document,
				executor.OperationName("Q"),
				executor.RootValue(rootValue))
			Expect(result).Should(MatchResultInJSON(`{"data":{"a":"b"}}`))
		})

		It("uses the mutation schema for mutations", func() {
			result := execute(
				schema,
				document,
				executor.OperationName("M"),
				executor.RootValue(rootValue))
			Expect(result).Should(MatchResultInJSON(`{"data":{"c":"d"}}`))
		})

		It("uses the subscription schema for subscriptions", func() {
			result := execute(
				schema,
				document,
				executor.OperationName("S"),
				executor.RootValue(rootValue))
			Expect(result).Should(MatchResultInJSON(`{"data":{"a":"b"}}`))
		})
	})

	It("correct field ordering despite execution order", func() {
		queryType := graphql.MustNewObject(&graphql.ObjectConfig{
			Name: "Type",
			Fields: graphql.Fields{
				"a": {Type: graphql.T(graphql.String())},
				"b": {Type: graphql.T(graphql.String())},
				"c": {Type: graphql.T(graphql.String())},
				"d": {Type: graphql.T(graphql.String())},
				"e": {Type: graphql.T(graphql.String())},
			},
		})

		schema := graphql.MustNewSchema(&graphql.SchemaConfig{
			Query: queryType,
		})

		document := parser.MustParse(token.NewSource("{ a, b, c, d, e }"))

		result := execute(
			schema,
			document,
			executor.RootValue(map[string]interface{}{
				"a": "a",
				"b": &ReadyOnNextPollFuture{data: "b"},
				"c": "c",
				"d": &ReadyOnNextPollFuture{data: "d"},
				"e": "e",
			}))
		Expect(result).Should(MatchResultInJSON(`{
				"data": {
					"a": "a",
					"b": "b",
					"c": "c",
					"d": "d",
					"e": "e"
				}
			}`))
	})

	It("avoids recursion", func() {
		document := parser.MustParse(token.NewSource(`
      {
        a
        ...Frag
        ...Frag
      }

      fragment Frag on Type {
        a,
        ...Frag
      }
    `))

		data := map[string]string{"a": "b"}

		queryType := graphql.MustNewObject(&graphql.ObjectConfig{
			Name: "Type",
			Fields: graphql.Fields{
				"a": {
					Type: graphql.T(graphql.String()),
				},
			},
		})

		schema := graphql.MustNewSchema(&graphql.SchemaConfig{
			Query: queryType,
		})

		result := execute(
			schema,
			document,
			executor.RootValue(data))
		Expect(result).Should(MatchResultInJSON(`{"data":{"a":"b"}}`))
	})

	It("does not include illegal fields in output", func() {
		document := parser.MustParse(token.NewSource(`{ thisIsIllegalDoNotIncludeMe }`))

		queryType := graphql.MustNewObject(&graphql.ObjectConfig{
			Name: "Q",
			Fields: graphql.Fields{
				"a": {
					Type: graphql.T(graphql.String()),
				},
			},
		})

		mutationType := graphql.MustNewObject(&graphql.ObjectConfig{
			Name: "M",
			Fields: graphql.Fields{
				"c": {
					Type: graphql.T(graphql.String()),
				},
			},
		})

		schema := graphql.MustNewSchema(&graphql.SchemaConfig{
			Query:    queryType,
			Mutation: mutationType,
		})

		Expect(execute(schema, document)).Should(MatchResultInJSON(`{"data":{}}`))
	})

	It("does not include arguments that were not set", func() {
		document := parser.MustParse(token.NewSource(`{ field(a: true, c: false, e: 0) }`))

		queryType := graphql.MustNewObject(&graphql.ObjectConfig{
			Name: "Type",
			Fields: graphql.Fields{
				"field": {
					Type: graphql.T(graphql.String()),
					Resolver: graphql.FieldResolverFunc(func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
						args, err := json.Marshal(info.Args())
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

		schema := graphql.MustNewSchema(&graphql.SchemaConfig{
			Query: queryType,
		})

		Expect(execute(schema, document)).Should(MatchResultInJSON(`{
			"data": {
				"field": "{\"a\":true,\"c\":false,\"e\":0}"
			}
		}`))
	})

	// We don't have isTypeOf.
	// It("fails when an isTypeOf check is not met", func() {
	// })

	It("executes ignoring invalid non-executable definitions", func() {
		// TODO: #17
		// schema := graphql.MustNewSchema(&graphql.SchemaConfig{
		// 	Query: graphql.MustNewObject(&graphql.ObjectConfig{
		// 		Name: "Query",
		// 		Fields: graphql.Fields{
		// 			"foo": {
		// 				Type: graphql.T(graphql.String()),
		// 			},
		// 		},
		// 	}),
		// })
		//
		// document := parser.MustParse(token.NewSource(`
		//   { foo }
		//
		//   type Query { bar: String }
		// `))
		//
		// result := execute(schema, document)
		// Expect(result).Should(MatchResultInJSON(`{"data":{"foo":null}}`))
	})

	It("uses a custom field resolver", func() {
		document := parser.MustParse(token.NewSource(`{ foo }`))

		queryType := graphql.MustNewObject(&graphql.ObjectConfig{
			Name: "Query",
			Fields: graphql.Fields{
				"foo": {
					Type: graphql.T(graphql.String()),
				},
			},
		})

		schema := graphql.MustNewSchema(&graphql.SchemaConfig{
			Query: queryType,
		})

		// For the purposes of test, just return the name of the field!
		fieldResolver := graphql.FieldResolverFunc(func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
			return info.Field().Name(), nil
		})

		result := execute(schema, document, executor.DefaultFieldResolver(fieldResolver))
		Expect(result).Should(MatchResultInJSON(`{"data":{"foo":"foo"}}`))
	})

	Describe("Iterable: custom iterator to enumerate values for list field", func() {
		var (
			executeWithRootValue func(rootValue interface{}) *executor.ExecutionResult
			values               []interface{}
		)

		BeforeEach(func() {
			schema := graphql.MustNewSchema(&graphql.SchemaConfig{
				Query: graphql.MustNewObject(&graphql.ObjectConfig{
					Name: "Query",
					Fields: graphql.Fields{
						"foo": {
							Type: graphql.ListOfType(graphql.Int()),
							Resolver: graphql.FieldResolverFunc(func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
								return info.RootValue(), nil
							}),
						},
					},
				}),
			})

			document := parser.MustParse(token.NewSource(`{ foo }`))

			executeWithRootValue = func(rootValue interface{}) *executor.ExecutionResult {
				return execute(schema, document, executor.RootValue(rootValue))
			}

			values = make([]interface{}, 100)
			for i := 0; i < len(values); i++ {
				values[i] = i
			}
		})

		It("uses custom iterator to enumerate values", func() {
			valuesJSON, err := json.Marshal(values)
			Expect(err).ShouldNot(HaveOccurred())

			expectedJSON := `{"data":{"foo":` + string(valuesJSON) + `}}`

			Expect(
				executeWithRootValue(&TestIterable{values}),
			).Should(MatchResultInJSON(expectedJSON))

			// Also test iterable with size hint.
			Expect(
				executeWithRootValue(&SizedTestIterable{TestIterable{values}}),
			).Should(MatchResultInJSON(expectedJSON))
		})

		It("handles error during iteration", func() {
			// Set an error value in the middle of values.
			values[len(values)/2] = errors.New("iterator error")

			Expect(executeWithRootValue(&TestIterable{values})).Should(MatchResultInJSON(`{
				"errors": [
					{
						"message": "Error occurred while enumerates values in the list field Query.foo.",
						"locations": [
							{
								"line": 1,
								"column": 3
							}
						],
						"path": [
							"foo"
						]
					}
				],
				"data": {
					"foo": null
				}
			}`))
		})
	})
})
