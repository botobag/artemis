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
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/graphql/executor"
	"github.com/botobag/artemis/graphql/parser"
	"github.com/botobag/artemis/graphql/token"
)

func queryStarWarsCharacterFriends(b *testing.B) *executor.ExecutionResult {
	// See BeforeEach in dataloader_test.go.

	// Build a simplified Star Wars schema.
	//
	// Reference: https://github.com/graphql/graphql.github.io/blob/e7b61aa/site/_core/swapiSchema.js
	//
	// schema {
	//   query: Query
	// }
	//
	// type Query {
	//   character(id: ID!): Character
	// }
	//
	// # A character from the Star Wars universe
	// type Character {
	//   # The ID of the character
	//   id: ID!
	//
	//   # The name of the character
	//   name: String!
	//
	//   # The friends of the character, or an empty list if they have none
	//   friends: [Character]
	// }
	characterType := &graphql.ObjectConfig{
		Name:        "Character",
		Description: "A character from the Star Wars universe",
	}

	characterFields := graphql.Fields{
		"id": {
			Type: graphql.NonNullOfType(graphql.ID()),
			Resolver: graphql.FieldResolverFunc(func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
				return string(source.(*Character).ID), nil
			}),
		},
		"name": {
			Type: graphql.NonNullOfType(graphql.String()),
			Resolver: graphql.FieldResolverFunc(func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
				return string(source.(*Character).Name), nil
			}),
		},
		"friends": {
			Type: graphql.ListOf(characterType),
			Resolver: graphql.FieldResolverFunc(func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
				friendIDs := source.(*Character).Friends
				friends := make([]*Character, len(friendIDs))
				for i, friendID := range friendIDs {
					friends[i] = CharacterData[friendID]
				}
				return friends, nil
			}),
		},
	}

	characterType.Fields = characterFields

	schema, err := graphql.NewSchema(&graphql.SchemaConfig{
		Query: graphql.MustNewObject(&graphql.ObjectConfig{
			Name: "Query",
			Fields: graphql.Fields{
				"character": {
					Type: characterType,
					Args: graphql.ArgumentConfigMap{
						"id": {
							Type: graphql.NonNullOfType(graphql.ID()),
						},
					},
					Resolver: graphql.FieldResolverFunc(func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
						id := info.Args().Get("id").(string)
						return CharacterData[CharacterID(id)], nil
					}),
				},
			},
		}),
	})

	document, err := parser.Parse(token.NewSource(&token.SourceConfig{
		Body: token.SourceBody([]byte(`{
				character(id: "1000") {
					id
					name
					friends {
						id
						name
						friends {
							id
							name
						}
					}
				}
			}`)),
	}))
	if err != nil {
		b.Fatal(err)
	}

	operation, errs := executor.Prepare(executor.PrepareParams{
		Schema:   schema,
		Document: document,
	})
	if errs.HaveOccurred() {
		b.Fatal(errs)
	}

	result := <-operation.Execute(context.Background(), executor.ExecuteParams{})

	return &result
}

func BenchmarkMarshalStarWarsFriendsQueryResultToJSONWithGo(b *testing.B) {
	var (
		buf     bytes.Buffer
		result  = queryStarWarsCharacterFriends(b)
		encoder = json.NewEncoder(&buf)
	)
	for i := 0; i < b.N; i++ {
		if err := encoder.Encode(result); err != nil {
			b.Fatal(err)
		}
		buf.Reset()
	}
}

func BenchmarkMarshalStarWarsFriendsQueryResultToJSONWithJsonwriter(b *testing.B) {
	var (
		buf    bytes.Buffer
		result = queryStarWarsCharacterFriends(b)
	)
	for i := 0; i < b.N; i++ {
		if err := result.MarshalJSONTo(&buf); err != nil {
			b.Fatal(err)
		}
		buf.Reset()
	}
}
