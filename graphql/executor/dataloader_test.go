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
	"runtime"
	"sync"

	"github.com/botobag/artemis/concurrent"
	"github.com/botobag/artemis/concurrent/future"
	"github.com/botobag/artemis/dataloader"
	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/graphql/executor"
	"github.com/botobag/artemis/graphql/parser"
	"github.com/botobag/artemis/graphql/token"
	"github.com/botobag/artemis/iterator"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type CharacterID string

type Character struct {
	ID      CharacterID
	Name    string
	Friends []CharacterID
}

var CharacterData = map[CharacterID]*Character{
	"1000": {
		ID:      "1000",
		Name:    "Luke Skywalker",
		Friends: []CharacterID{"1002", "1003", "2000", "2001"},
	},
	"1001": {
		ID:      "1001",
		Name:    "Darth Vader",
		Friends: []CharacterID{"1004"},
	},
	"1002": {
		ID:      "1002",
		Name:    "Han Solo",
		Friends: []CharacterID{"1000", "1003", "2001"},
	},
	"1003": {
		ID:      "1003",
		Name:    "Leia Organa",
		Friends: []CharacterID{"1000", "1002", "2000", "2001"},
	},
	"1004": {
		ID:      "1004",
		Name:    "Wilhuff Tarkin",
		Friends: []CharacterID{"1001"},
	},
	"2000": {
		ID:      "2000",
		Name:    "C-3PO",
		Friends: []CharacterID{"1000", "1002", "1003", "2001"},
	},
	"2001": {
		ID:      "2001",
		Name:    "R2-D2",
		Friends: []CharacterID{"1000", "1002", "1003"},
	},
}

// CharacterBatchLoader performs batch load of character data.
type CharacterBatchLoader struct {
	mutex     sync.Mutex
	data      map[CharacterID]*Character
	loadCalls [][]dataloader.Key
}

func NewCharacterBatchLoader() *CharacterBatchLoader {
	return &CharacterBatchLoader{
		data: CharacterData,
	}
}

// Load implements dataloader.BatchLoader.
func (loader *CharacterBatchLoader) Load(ctx context.Context, tasks *dataloader.TaskList) {
	// Collect keys for check.
	var keys []dataloader.Key

	taskIter := tasks.Iterator()
	for {
		task, done := taskIter.Next()
		if done {
			break
		}
		key := task.Key()
		Expect(task.Complete(loader.data[key.(CharacterID)])).Should(Succeed())
		keys = append(keys, key)
	}

	// Append keys to loadCalls.
	mutex := &loader.mutex
	mutex.Lock()
	loader.loadCalls = append(loader.loadCalls, keys)
	mutex.Unlock()
}

func (loader *CharacterBatchLoader) LoadCalls() [][]dataloader.Key {
	return loader.loadCalls
}

// StarWarsDataLoaderManager implements graphql.DataLoaderManager to load data for Star Wars schema.
type StarWarsDataLoaderManager struct {
	graphql.DataLoaderManagerBase

	// Batch loader that fetches character data; This is used to obtain the batch load that have been
	// executed.
	characterBatchLoader *CharacterBatchLoader
	// DataLoader for batch load data for characters
	characterLoader *dataloader.DataLoader
}

func NewStarWarsDataLoaderManager() *StarWarsDataLoaderManager {
	characterBatchLoader := NewCharacterBatchLoader()
	characterLoader, err := dataloader.New(dataloader.Config{
		BatchLoader: characterBatchLoader,
	})
	Expect(err).ShouldNot(HaveOccurred())
	return &StarWarsDataLoaderManager{
		characterBatchLoader: characterBatchLoader,
		characterLoader:      characterLoader,
	}
}

func (manager *StarWarsDataLoaderManager) LoadCharacterByID(id CharacterID) (future.Future, error) {
	return manager.LoadWith(manager.characterLoader, id)
}

// characterIDArray is a return value for KeysFromArray which implements KeysWithSize.
type characterIDArray struct {
	ids []CharacterID
}

type characterIDArrayIterator struct {
	ids  []CharacterID
	i    int
	size int
}

// Iterator implements Keys.
func (a characterIDArray) Iterator() dataloader.KeyIterator {
	return &characterIDArrayIterator{
		ids:  a.ids,
		i:    0,
		size: len(a.ids),
	}
}

// Size implements KeysWithSize.
func (a characterIDArray) Size() int {
	return len(a.ids)
}

// Next implements KeyIterator.
func (iter *characterIDArrayIterator) Next() (dataloader.Key, error) {
	i := iter.i
	if i != iter.size {
		iter.i++
		return iter.ids[i], nil
	}
	return nil, iterator.Done
}

func (manager *StarWarsDataLoaderManager) LoadManyCharactersByID(ids []CharacterID) (future.Future, error) {
	return manager.LoadManyWith(manager.characterLoader, characterIDArray{ids})
}

func (manager *StarWarsDataLoaderManager) CharacterLoadCalls() [][]dataloader.Key {
	return manager.characterBatchLoader.LoadCalls()
}

var _ = Describe("Execute: fetch data with DataLoader", func() {
	var (
		schema            graphql.Schema
		runner            concurrent.Executor
		dataloaderManager *StarWarsDataLoaderManager
	)

	BeforeEach(func() {
		var err error

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
					return info.DataLoaderManager().(*StarWarsDataLoaderManager).LoadManyCharactersByID(friendIDs)
				}),
			},
		}

		characterType.Fields = characterFields

		schema, err = graphql.NewSchema(&graphql.SchemaConfig{
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
							return info.DataLoaderManager().(*StarWarsDataLoaderManager).LoadCharacterByID(CharacterID(id))
						}),
					},
				},
			}),
		})
		Expect(err).ShouldNot(HaveOccurred())

		runner, err = concurrent.NewWorkerPoolExecutor(concurrent.WorkerPoolExecutorConfig{
			MaxPoolSize: uint32(runtime.GOMAXPROCS(-1)),
		})
		Expect(err).ShouldNot(HaveOccurred())

		dataloaderManager = NewStarWarsDataLoaderManager()
	})

	AfterEach(func() {
		terminated, err := runner.Shutdown()
		Expect(err).ShouldNot(HaveOccurred())
		Eventually(terminated).Should(Receive(BeTrue()))
	})

	It("executes a simple query with single load", func() {
		document, err := parser.Parse(token.NewSource(&token.SourceConfig{
			Body: token.SourceBody([]byte(`{
				character(id: "1000") {
					id
					name
				}
			}`)),
		}))
		Expect(err).ShouldNot(HaveOccurred())

		operation, errs := executor.Prepare(executor.PrepareParams{
			Schema:   schema,
			Document: document,
		})
		Expect(errs.HaveOccurred()).ShouldNot(BeTrue())

		result := operation.Execute(context.Background(), executor.ExecuteParams{
			Runner:            runner,
			DataLoaderManager: dataloaderManager,
		})

		Eventually(result).Should(MatchResultInJSON(`{
      "data": {
        "character": {
          "id": "1000",
          "name": "Luke Skywalker"
        }
      }
    }`))

		Expect(dataloaderManager.CharacterLoadCalls()).Should(Equal([][]dataloader.Key{
			{CharacterID("1000")},
		}))
	})

	It("executes a query with multiple and a nested loads", func() {
		document, err := parser.Parse(token.NewSource(&token.SourceConfig{
			Body: token.SourceBody([]byte(`{
				character(id: "1000") {
					id
					name
					friends {
						id
						name
					}
				}
			}`)),
		}))
		Expect(err).ShouldNot(HaveOccurred())

		operation, errs := executor.Prepare(executor.PrepareParams{
			Schema:   schema,
			Document: document,
		})
		Expect(errs.HaveOccurred()).ShouldNot(BeTrue())

		result := operation.Execute(context.Background(), executor.ExecuteParams{
			Runner:            runner,
			DataLoaderManager: dataloaderManager,
		})

		Eventually(result).Should(MatchResultInJSON(`{
      "data": {
        "character": {
          "id": "1000",
          "name": "Luke Skywalker",
          "friends": [
            {
              "id": "1002",
              "name": "Han Solo"
            },
            {
              "id": "1003",
              "name": "Leia Organa"
            },
            {
              "id": "2000",
              "name": "C-3PO"
            },
            {
              "id": "2001",
              "name": "R2-D2"
            }
          ]
        }
      }
    }`))

		Expect(dataloaderManager.CharacterLoadCalls()).Should(Equal([][]dataloader.Key{
			{CharacterID("1000")},
			{
				CharacterID("1002"),
				CharacterID("1003"),
				CharacterID("2000"),
				CharacterID("2001"),
			},
		}))
	})

	It("executes a query with loads nested in multiple levels", func() {
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
		Expect(err).ShouldNot(HaveOccurred())

		operation, errs := executor.Prepare(executor.PrepareParams{
			Schema:   schema,
			Document: document,
		})
		Expect(errs.HaveOccurred()).ShouldNot(BeTrue())

		result := operation.Execute(context.Background(), executor.ExecuteParams{
			Runner:            runner,
			DataLoaderManager: dataloaderManager,
		})

		Eventually(result).Should(MatchResultInJSON(`{
      "data": {
        "character": {
          "id": "1000",
          "name": "Luke Skywalker",
          "friends": [
            {
              "id": "1002",
              "name": "Han Solo",
              "friends": [
                {
                  "id": "1000",
                  "name": "Luke Skywalker"
                },
                {
                  "id": "1003",
                  "name": "Leia Organa"
                },
                {
                  "id": "2001",
                  "name": "R2-D2"
                }
              ]
            },
            {
              "id": "1003",
              "name": "Leia Organa",
              "friends": [
                {
                  "id": "1000",
                  "name": "Luke Skywalker"
                },
                {
                  "id": "1002",
                  "name": "Han Solo"
                },
                {
                  "id": "2000",
                  "name": "C-3PO"
                },
                {
                  "id": "2001",
                  "name": "R2-D2"
                }
              ]
            },
            {
              "id": "2000",
              "name": "C-3PO",
              "friends": [
                {
                  "id": "1000",
                  "name": "Luke Skywalker"
                },
                {
                  "id": "1002",
                  "name": "Han Solo"
                },
                {
                  "id": "1003",
                  "name": "Leia Organa"
                },
                {
                  "id": "2001",
                  "name": "R2-D2"
                }
              ]
            },
            {
              "id": "2001",
              "name": "R2-D2",
              "friends": [
                {
                  "id": "1000",
                  "name": "Luke Skywalker"
                },
                {
                  "id": "1002",
                  "name": "Han Solo"
                },
                {
                  "id": "1003",
                  "name": "Leia Organa"
                }
              ]
            }
          ]
        }
      }
    }`))

		Expect(dataloaderManager.CharacterLoadCalls()).Should(Equal([][]dataloader.Key{
			{CharacterID("1000")},
			{
				CharacterID("1002"),
				CharacterID("1003"),
				CharacterID("2000"),
				CharacterID("2001"),
			},
		}))
	})
})
