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
	"errors"

	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/graphql/ast"
	"github.com/botobag/artemis/internal/testutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// The following test is translated from:
//
//  https://github.com/sangria-graphql/sangria/blob/0bf8053/src/test/scala/sangria/execution/ScalarAliasSpec.scala
//
// The license is reproduced as followed:
//
// Sangria License
// ===============
//
// Copyright 2018, The Sangria Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

type UserID struct {
	ID string
}

type UserIDCoercer struct{}

func (UserIDCoercer) CoerceResultValue(value interface{}) (interface{}, error) {
	userID, ok := value.(UserID)
	if !ok {
		return nil, errors.New("unexpected value type presented to UserId; Expected an UserID value")
	}
	return graphql.String().CoerceResultValue(userID.ID)
}

func (UserIDCoercer) coerceInputValue(value interface{}) (interface{}, error) {
	id, ok := value.(string)
	if !ok {
		return nil, errors.New("unexpected input value type to UserId; Expected a string value")
	}
	return UserID{
		ID: id,
	}, nil
}

func (coercer UserIDCoercer) CoerceVariableValue(value interface{}) (interface{}, error) {
	value, err := graphql.String().CoerceVariableValue(value)
	if err != nil {
		return nil, err
	}
	return coercer.coerceInputValue(value)
}

func (coercer UserIDCoercer) CoerceArgumentValue(value ast.Value) (interface{}, error) {
	v, err := graphql.String().CoerceArgumentValue(value)
	if err != nil {
		return nil, err
	}
	return coercer.coerceInputValue(v)
}

type PositiveIntCoercer struct{}

func (PositiveIntCoercer) coerceInputValue(value interface{}) (interface{}, error) {
	i, ok := value.(int)
	if !ok {
		return nil, errors.New("unexpected input value type to PositiveInt variable; Expected an int value")
	}

	if i <= 0 {
		return nil, graphql.NewCoercionError("Int cannot represent %d: predicate failed: (%d > 0)", i, i)
	}

	return i, nil
}

func (coercer PositiveIntCoercer) CoerceVariableValue(value interface{}) (interface{}, error) {
	value, err := graphql.Int().CoerceVariableValue(value)
	if err != nil {
		return nil, err
	}
	return coercer.coerceInputValue(value)
}

func (coercer PositiveIntCoercer) CoerceArgumentValue(value ast.Value) (interface{}, error) {
	v, err := graphql.Int().CoerceArgumentValue(value)
	if err != nil {
		return nil, err
	}
	return coercer.coerceInputValue(v)
}

var _ = Describe("ScalarAlias", func() {
	It("rejects creating type without aliasing scalar", func() {
		_, err := graphql.NewScalarAlias(&graphql.ScalarAliasConfig{
			AliasFor: nil,
		})
		Expect(err).Should(MatchError("Must provide aliasing Scalar type for ScalarAlias."))

		Expect(func() {
			graphql.MustNewScalarAlias(&graphql.ScalarAliasConfig{})
		}).Should(Panic())
	})

	It("accepts creating type without specifying coercers", func() {
		_, err := graphql.NewScalarAlias(&graphql.ScalarAliasConfig{
			AliasFor:      graphql.Int(),
			ResultCoercer: nil,
			InputCoercer:  nil,
		})
		Expect(err).ShouldNot(HaveOccurred())
	})

	Describe("Type System: ScalarAlias Values", func() {
		type User struct {
			ID   UserID
			ID2  *UserID
			Name string
			Num  int
		}

		var Schema *graphql.Schema

		BeforeEach(func() {
			userIDType := &graphql.ScalarAliasConfig{
				AliasFor:      graphql.String(),
				ResultCoercer: UserIDCoercer{},
				InputCoercer:  UserIDCoercer{},
			}

			positiveIntType := &graphql.ScalarAliasConfig{
				AliasFor:     graphql.Int(),
				InputCoercer: PositiveIntCoercer{},
			}

			userType := &graphql.ObjectConfig{
				Name: "User",
				Fields: graphql.Fields{
					"id": {
						Type: graphql.NonNullOf(userIDType),
						Resolver: graphql.FieldResolverFunc(func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
							return source.(*User).ID, nil
						}),
					},
					"id2": {
						Type: userIDType,
						Resolver: graphql.FieldResolverFunc(func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
							user := source.(*User)
							if user.ID2 != nil {
								return *user.ID2, nil
							}
							return nil, nil
						}),
					},
					"name": {
						Type: graphql.T(graphql.String()),
						Resolver: graphql.FieldResolverFunc(func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
							return source.(*User).Name, nil
						}),
					},
					"num": {
						Type: positiveIntType,
						Resolver: graphql.FieldResolverFunc(func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
							return source.(*User).Num, nil
						}),
					},
				},
			}

			complexInputType := &graphql.InputObjectConfig{
				Name: "Complex",
				Fields: graphql.InputFields{
					"userId": {
						Type: userIDType,
						DefaultValue: UserID{
							ID: "5678",
						},
					},
					"userNum": {
						Type: positiveIntType,
					},
				},
			}

			queryType, err := graphql.NewObject(&graphql.ObjectConfig{
				Name: "Query",
				Fields: graphql.Fields{
					"user": {
						Type: userType,
						Args: graphql.ArgumentConfigMap{
							"id": {
								Type: userIDType,
							},
							"n": {
								Type: positiveIntType,
							},
							"c": {
								Type: complexInputType,
							},
						},
						Resolver: graphql.FieldResolverFunc(func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
							userID, ok := info.ArgumentValues().Get("id").(UserID)
							Expect(ok).Should(BeTrue())

							num, ok := info.ArgumentValues().Get("n").(int)
							Expect(ok).Should(BeTrue())

							complexValue, ok := info.ArgumentValues().Get("c").(map[string]interface{})
							Expect(ok).Should(BeTrue())

							var userID2 *UserID
							if complexValue["userId"] != nil {
								id2, ok := complexValue["userId"].(UserID)
								Expect(ok).Should(BeTrue())
								userID2 = &id2
							}

							return &User{
								ID:   userID,
								ID2:  userID2,
								Name: "generated",
								Num:  num,
							}, nil
						}),
					},
				},
			})

			Schema, err = graphql.NewSchema(&graphql.SchemaConfig{
				Query: queryType,
			})
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("represents value class as scalar type", func() {
			query := `{
  user(id: "1234", n: 42, c: {userNum: 500}) {
    id
    id2
    name
    num
  }
}`

			Expect(executeQuery(Schema, query)).Should(testutil.SerializeToJSONAs(map[string]interface{}{
				"data": map[string]interface{}{
					"user": map[string]interface{}{
						"id":   "1234",
						"id2":  "5678",
						"name": "generated",
						"num":  42,
					},
				},
			}))
		})

		It("coerces input types correctly", func() {
			query := `{
  user(id: "1234", n: -123, c: {userNum: 5}) {
    id
    name
  }
}`
			result := executeQuery(Schema, query)
			Expect(result.Errors.Errors).Should(ConsistOf(testutil.MatchGraphQLError(
				testutil.MessageEqual(`Argument "n" has invalid value "-123".`),
				testutil.LocationEqual(graphql.ErrorLocation{
					Line:   2,
					Column: 23,
				}),
			)))

			query = `{
  user(id: "1234", n: 123, c: {userId: 1, userNum: 5}) {
    id
    name
  }
}`
			result = executeQuery(Schema, query)
			Expect(result.Errors.Errors).Should(ConsistOf(testutil.MatchGraphQLError(
				testutil.MessageContainSubstring(`Argument "c" has invalid value`),
				testutil.LocationEqual(graphql.ErrorLocation{
					Line:   2,
					Column: 31,
				}),
			)))

			query = `{
  user(id: "1234", n: 123, c: {userNum: -5}) {
    id
    name
  }
}`
			result = executeQuery(Schema, query)
			Expect(result.Errors.Errors).Should(ConsistOf(testutil.MatchGraphQLError(
				testutil.MessageEqual(`Argument "c" has invalid value "map[userNum:-5]".`),
				testutil.LocationEqual(graphql.ErrorLocation{
					Line:   2,
					Column: 31,
				}),
			)))
		})
	})
})
