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

package rules_test

import (
	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/graphql/internal/validator"
	"github.com/botobag/artemis/graphql/validator/rules"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// graphql-js/src/validation/__tests__/OverlappingFieldsCanBeMerged-test.js@8c96dc8
var _ = Describe("Validate: Overlapping fields can be merged", func() {
	expectErrors := func(queryStr string) GomegaAssertion {
		return expectValidationErrors(rules.OverlappingFieldsCanBeMerged{}, queryStr)
	}

	expectValid := func(queryStr string) {
		expectErrors(queryStr).Should(Equal(graphql.NoErrors()))
	}

	expectErrorsWithSchema := func(schema graphql.Schema, queryStr string) GomegaAssertion {
		return expectValidationErrorsWithSchema(
			schema,
			rules.OverlappingFieldsCanBeMerged{},
			queryStr,
		)
	}

	expectValidWithSchema := func(schema graphql.Schema, queryStr string) {
		expectErrorsWithSchema(schema, queryStr).Should(Equal(graphql.NoErrors()))
	}

	fieldsConflictMessage := func(responseKey string, reason interface{}) string {
		return validator.FieldsConflictMessage(&validator.FieldConflictReason{
			ResponseKey:              responseKey,
			MessageOrSubFieldReasons: reason,
		})
	}

	It("unique fields", func() {
		expectValid(`
      fragment uniqueFields on Dog {
        name
        nickname
      }
    `)
	})

	It("identical fields", func() {
		expectValid(`
      fragment mergeIdenticalFields on Dog {
        name
        name
      }
    `)
	})

	It("identical fields with identical args", func() {
		expectValid(`
      fragment mergeIdenticalFieldsWithIdenticalArgs on Dog {
        doesKnowCommand(dogCommand: SIT)
        doesKnowCommand(dogCommand: SIT)
      }
    `)
	})

	It("identical fields with identical directives", func() {
		expectValid(`
      fragment mergeSameFieldsWithSameDirectives on Dog {
        name @include(if: true)
        name @include(if: true)
      }
    `)
	})

	It("different args with different aliases", func() {
		expectValid(`
      fragment differentArgsWithDifferentAliases on Dog {
        knowsSit: doesKnowCommand(dogCommand: SIT)
        knowsDown: doesKnowCommand(dogCommand: DOWN)
      }
    `)
	})

	It("different directives with different aliases", func() {
		expectValid(`
      fragment differentDirectivesWithDifferentAliases on Dog {
        nameIfTrue: name @include(if: true)
        nameIfFalse: name @include(if: false)
      }
    `)
	})

	It("different skip/include directives accepted", func() {
		// Note: Differing skip/include directives don"t create an ambiguous return value and are
		// acceptable in conditions where differing runtime values may have the same desired effect of
		// including or skipping a field.
		expectValid(`
      fragment differentDirectivesWithDifferentAliases on Dog {
        name @include(if: true)
        name @include(if: false)
      }
    `)
	})

	It("Same aliases with different field targets", func() {
		expectErrors(`
      fragment sameAliasesWithDifferentFieldTargets on Dog {
        fido: name
        fido: nickname
      }
    `).Should(Equal(graphql.ErrorsOf(
			graphql.NewError(
				fieldsConflictMessage("fido", "name and nickname are different fields"),
				[]graphql.ErrorLocation{
					{Line: 3, Column: 9},
					{Line: 4, Column: 9},
				},
			),
		)))
	})

	It("Same aliases allowed on non-overlapping fields", func() {
		// This is valid since no object can be both a "Dog" and a "Cat", thus
		// these fields can never overlap.
		expectValid(`
      fragment sameAliasesWithDifferentFieldTargets on Pet {
        ... on Dog {
          name
        }
        ... on Cat {
          name: nickname
        }
      }
    `)
	})

	It("Alias masking direct field access", func() {
		expectErrors(`
      fragment aliasMaskingDirectFieldAccess on Dog {
        name: nickname
        name
      }
    `).Should(Equal(graphql.ErrorsOf(
			graphql.NewError(
				fieldsConflictMessage("name", "nickname and name are different fields"),
				[]graphql.ErrorLocation{
					{Line: 3, Column: 9},
					{Line: 4, Column: 9},
				},
			),
		)))
	})

	It("different args, second adds an argument", func() {
		expectErrors(`
      fragment conflictingArgs on Dog {
        doesKnowCommand
        doesKnowCommand(dogCommand: HEEL)
      }
    `).Should(Equal(graphql.ErrorsOf(
			graphql.NewError(
				fieldsConflictMessage("doesKnowCommand", "they have differing arguments"),
				[]graphql.ErrorLocation{
					{Line: 3, Column: 9},
					{Line: 4, Column: 9},
				},
			),
		)))
	})

	It("different args, second missing an argument", func() {
		expectErrors(`
      fragment conflictingArgs on Dog {
        doesKnowCommand(dogCommand: SIT)
        doesKnowCommand
      }
    `).Should(Equal(graphql.ErrorsOf(
			graphql.NewError(
				fieldsConflictMessage("doesKnowCommand", "they have differing arguments"),
				[]graphql.ErrorLocation{
					{Line: 3, Column: 9},
					{Line: 4, Column: 9},
				},
			),
		)))
	})

	It("conflicting args", func() {
		expectErrors(`
      fragment conflictingArgs on Dog {
        doesKnowCommand(dogCommand: SIT)
        doesKnowCommand(dogCommand: HEEL)
      }
    `).Should(Equal(graphql.ErrorsOf(
			graphql.NewError(
				fieldsConflictMessage("doesKnowCommand", "they have differing arguments"),
				[]graphql.ErrorLocation{
					{Line: 3, Column: 9},
					{Line: 4, Column: 9},
				},
			),
		)))
	})

	It("allows different args where no conflict is possible", func() {
		// This is valid since no object can be both a "Dog" and a "Cat", thus
		// these fields can never overlap.
		expectValid(`
      fragment conflictingArgs on Pet {
        ... on Dog {
          name(surname: true)
        }
        ... on Cat {
          name
        }
      }
    `)
	})

	It("encounters conflict in fragments", func() {
		expectErrors(`
      {
        ...A
        ...B
      }
      fragment A on Type {
        x: a
      }
      fragment B on Type {
        x: b
      }
    `).Should(Equal(graphql.ErrorsOf(
			graphql.NewError(
				fieldsConflictMessage("x", "a and b are different fields"),
				[]graphql.ErrorLocation{
					{Line: 7, Column: 9},
					{Line: 10, Column: 9},
				},
			),
		)))
	})

	It("reports each conflict once", func() {
		expectErrors(`
      {
        f1 {
          ...A
          ...B
        }
        f2 {
          ...B
          ...A
        }
        f3 {
          ...A
          ...B
          x: c
        }
      }
      fragment A on Type {
        x: a
      }
      fragment B on Type {
        x: b
      }
    `).Should(Equal(graphql.ErrorsOf(
			graphql.NewError(
				fieldsConflictMessage("x", "a and b are different fields"),
				[]graphql.ErrorLocation{
					{Line: 18, Column: 9},
					{Line: 21, Column: 9},
				},
			),
			graphql.NewError(
				fieldsConflictMessage("x", "c and a are different fields"),
				[]graphql.ErrorLocation{
					{Line: 14, Column: 11},
					{Line: 18, Column: 9},
				},
			),
			graphql.NewError(
				fieldsConflictMessage("x", "c and b are different fields"),
				[]graphql.ErrorLocation{
					{Line: 14, Column: 11},
					{Line: 21, Column: 9},
				},
			),
		)))
	})

	It("deep conflict", func() {
		expectErrors(`
      {
        field {
          x: a
        },
        field {
          x: b
        }
      }
    `).Should(Equal(graphql.ErrorsOf(
			graphql.NewError(
				fieldsConflictMessage("field", []*validator.FieldConflictReason{
					{
						ResponseKey:              "x",
						MessageOrSubFieldReasons: "a and b are different fields",
					},
				}),
				[]graphql.ErrorLocation{
					{Line: 3, Column: 9},
					{Line: 4, Column: 11},
					{Line: 6, Column: 9},
					{Line: 7, Column: 11},
				},
			),
		)))
	})

	It("deep conflict with multiple issues", func() {
		expectErrors(`
      {
        field {
          x: a
          y: c
        },
        field {
          x: b
          y: d
        }
      }
    `).Should(Or(
			Equal(graphql.ErrorsOf(
				graphql.NewError(
					fieldsConflictMessage("field", []*validator.FieldConflictReason{
						{
							ResponseKey:              "x",
							MessageOrSubFieldReasons: "a and b are different fields",
						},
						{
							ResponseKey:              "y",
							MessageOrSubFieldReasons: "c and d are different fields",
						},
					}),
					[]graphql.ErrorLocation{
						{Line: 3, Column: 9},
						{Line: 4, Column: 11},
						{Line: 5, Column: 11},
						{Line: 7, Column: 9},
						{Line: 8, Column: 11},
						{Line: 9, Column: 11},
					},
				),
			)),
			Equal(graphql.ErrorsOf(
				graphql.NewError(
					fieldsConflictMessage("field", []*validator.FieldConflictReason{
						{
							ResponseKey:              "y",
							MessageOrSubFieldReasons: "c and d are different fields",
						},
						{
							ResponseKey:              "x",
							MessageOrSubFieldReasons: "a and b are different fields",
						},
					}),
					[]graphql.ErrorLocation{
						{Line: 3, Column: 9},
						{Line: 5, Column: 11},
						{Line: 4, Column: 11},
						{Line: 7, Column: 9},
						{Line: 9, Column: 11},
						{Line: 8, Column: 11},
					},
				),
			)),
		))
	})

	It("very deep conflict", func() {
		expectErrors(`
      {
        field {
          deepField {
            x: a
          }
        },
        field {
          deepField {
            x: b
          }
        }
      }
    `).Should(Equal(graphql.ErrorsOf(
			graphql.NewError(
				fieldsConflictMessage("field", []*validator.FieldConflictReason{
					{
						ResponseKey: "deepField",
						MessageOrSubFieldReasons: []*validator.FieldConflictReason{
							{
								ResponseKey:              "x",
								MessageOrSubFieldReasons: "a and b are different fields",
							},
						},
					},
				}),
				[]graphql.ErrorLocation{
					{Line: 3, Column: 9},
					{Line: 4, Column: 11},
					{Line: 5, Column: 13},
					{Line: 8, Column: 9},
					{Line: 9, Column: 11},
					{Line: 10, Column: 13},
				},
			),
		)))
	})

	It("reports deep conflict to nearest common ancestor", func() {
		expectErrors(`
      {
        field {
          deepField {
            x: a
          }
          deepField {
            x: b
          }
        },
        field {
          deepField {
            y
          }
        }
      }
    `).Should(Equal(graphql.ErrorsOf(
			graphql.NewError(
				fieldsConflictMessage("deepField", []*validator.FieldConflictReason{
					{
						ResponseKey:              "x",
						MessageOrSubFieldReasons: "a and b are different fields",
					},
				}),
				[]graphql.ErrorLocation{
					{Line: 4, Column: 11},
					{Line: 5, Column: 13},
					{Line: 7, Column: 11},
					{Line: 8, Column: 13},
				},
			),
		)))
	})

	It("reports deep conflict to nearest common ancestor in fragments", func() {
		expectErrors(`
      {
        field {
          ...F
        }
        field {
          ...F
        }
      }
      fragment F on T {
        deepField {
          deeperField {
            x: a
          }
          deeperField {
            x: b
          }
        },
        deepField {
          deeperField {
            y
          }
        }
      }
    `).Should(Equal(graphql.ErrorsOf(
			graphql.NewError(
				fieldsConflictMessage("deeperField", []*validator.FieldConflictReason{
					{
						ResponseKey:              "x",
						MessageOrSubFieldReasons: "a and b are different fields",
					},
				}),
				[]graphql.ErrorLocation{
					{Line: 12, Column: 11},
					{Line: 13, Column: 13},
					{Line: 15, Column: 11},
					{Line: 16, Column: 13},
				},
			),
		)))
	})

	It("reports deep conflict in nested fragments", func() {
		expectErrors(`
      {
        field {
          ...F
        }
        field {
          ...I
        }
      }
      fragment F on T {
        x: a
        ...G
      }
      fragment G on T {
        y: c
      }
      fragment I on T {
        y: d
        ...J
      }
      fragment J on T {
        x: b
      }
    `).Should(Equal(graphql.ErrorsOf(
			graphql.NewError(
				fieldsConflictMessage("field", []*validator.FieldConflictReason{
					{
						ResponseKey:              "x",
						MessageOrSubFieldReasons: "a and b are different fields",
					},
					{
						ResponseKey:              "y",
						MessageOrSubFieldReasons: "c and d are different fields",
					},
				}),
				[]graphql.ErrorLocation{
					{Line: 3, Column: 9},
					{Line: 11, Column: 9},
					{Line: 15, Column: 9},
					{Line: 6, Column: 9},
					{Line: 22, Column: 9},
					{Line: 18, Column: 9},
				},
			),
		)))
	})

	It("ignores unknown fragments", func() {
		expectValid(`
      {
        field
        ...Unknown
        ...Known
      }

      fragment Known on T {
        field
        ...OtherUnknown
      }
    `)
	})

	Describe("return types must be unambiguous", func() {
		var schema graphql.Schema

		BeforeEach(func() {
			SomeBox := &graphql.InterfaceConfig{
				Name: "SomeBox",
			}
			SomeBox.Fields = graphql.Fields{
				"deepBox": {
					Type: SomeBox,
				},
				"unrelatedField": {
					Type: graphql.T(graphql.String()),
				},
			}

			IntBox := &graphql.ObjectConfig{
				Name: "IntBox",
				Interfaces: []graphql.InterfaceTypeDefinition{
					SomeBox,
				},
			}

			StringBox := &graphql.ObjectConfig{
				Name: "StringBox",
				Interfaces: []graphql.InterfaceTypeDefinition{
					SomeBox,
				},
			}

			IntBox.Fields = graphql.Fields{
				"scalar": {
					Type: graphql.T(graphql.Int()),
				},
				"deepBox": {
					Type: IntBox,
				},
				"unrelatedField": {
					Type: graphql.T(graphql.String()),
				},
				"listStringBox": {
					Type: graphql.ListOf(StringBox),
				},
				"stringBox": {
					Type: StringBox,
				},
				"intBox": {
					Type: IntBox,
				},
			}

			StringBox.Fields = graphql.Fields{
				"scalar": {
					Type: graphql.T(graphql.String()),
				},
				"deepBox": {
					Type: StringBox,
				},
				"unrelatedField": {
					Type: graphql.T(graphql.String()),
				},
				"listStringBox": {
					Type: graphql.ListOf(StringBox),
				},
				"stringBox": {
					Type: StringBox,
				},
				"intBox": {
					Type: IntBox,
				},
			}

			NonNullStringBox1 := &graphql.InterfaceConfig{
				Name: "NonNullStringBox1",
				Fields: graphql.Fields{
					"scalar": {
						Type: graphql.NonNullOfType(graphql.String()),
					},
				},
			}

			NonNullStringBox1Impl := &graphql.ObjectConfig{
				Name: "NonNullStringBox1Impl",
				Fields: graphql.Fields{
					"scalar": {
						Type: graphql.NonNullOfType(graphql.String()),
					},
					"unrelatedField": {
						Type: graphql.T(graphql.String()),
					},
					"deepBox": {
						Type: SomeBox,
					},
				},
				Interfaces: []graphql.InterfaceTypeDefinition{
					SomeBox,
					NonNullStringBox1,
				},
			}

			NonNullStringBox2 := &graphql.InterfaceConfig{
				Name: "NonNullStringBox2",
				Fields: graphql.Fields{
					"scalar": {
						Type: graphql.NonNullOfType(graphql.String()),
					},
				},
			}

			NonNullStringBox2Impl := &graphql.ObjectConfig{
				Name: "NonNullStringBox2Impl",
				Fields: graphql.Fields{
					"scalar": {
						Type: graphql.NonNullOfType(graphql.String()),
					},
					"unrelatedField": {
						Type: graphql.T(graphql.String()),
					},
					"deepBox": {
						Type: SomeBox,
					},
				},
				Interfaces: []graphql.InterfaceTypeDefinition{
					SomeBox,
					NonNullStringBox2,
				},
			}

			Node := &graphql.ObjectConfig{
				Name: "Node",
				Fields: graphql.Fields{
					"id": {
						Type: graphql.T(graphql.ID()),
					},
					"name": {
						Type: graphql.T(graphql.String()),
					},
				},
			}

			Edge := &graphql.ObjectConfig{
				Name: "Edge",
				Fields: graphql.Fields{
					"node": {
						Type: Node,
					},
				},
			}

			Connection := &graphql.ObjectConfig{
				Name: "Connection",
				Fields: graphql.Fields{
					"edges": {
						Type: graphql.ListOf(Edge),
					},
				},
			}

			schema = graphql.MustNewSchema(&graphql.SchemaConfig{
				Query: graphql.MustNewObject(&graphql.ObjectConfig{
					Name: "Query",
					Fields: graphql.Fields{
						"someBox": {
							Type: SomeBox,
						},
						"connection": {
							Type: Connection,
						},
					},
				}),
				Types: []graphql.Type{
					graphql.MustNewObject(IntBox),
					graphql.MustNewObject(StringBox),
					graphql.MustNewObject(NonNullStringBox1Impl),
					graphql.MustNewObject(NonNullStringBox2Impl),
				},
			})
		})

		It("conflicting return types which potentially overlap", func() {
			// This is invalid since an object could potentially be both the Object type IntBox and the
			// interface type NonNullStringBox1. While that condition does not exist in the current
			// schema, the schema could expand in the future to allow this. Thus It is invalid.
			expectErrorsWithSchema(
				schema,
				`
          {
            someBox {
              ...on IntBox {
                scalar
              }
              ...on NonNullStringBox1 {
                scalar
              }
            }
          }
        `,
			).Should(Or(
				Equal(
					graphql.ErrorsOf(
						graphql.NewError(
							fieldsConflictMessage("scalar", "they return conflicting types Int and String!"),
							[]graphql.ErrorLocation{
								{Line: 5, Column: 17},
								{Line: 8, Column: 17},
							},
						),
					),
				),
				Equal(
					graphql.ErrorsOf(
						graphql.NewError(
							fieldsConflictMessage("scalar", "they return conflicting types String! and Int"),
							[]graphql.ErrorLocation{
								{Line: 8, Column: 17},
								{Line: 5, Column: 17},
							},
						),
					),
				)))
		})

		It("compatible return shapes on different return types", func() {
			// In this case `deepBox` returns `SomeBox` in the first usage, and `StringBox` in the second
			// usage. These return types are not the same! however this is valid because the return
			// *shapes* are compatible.
			expectValidWithSchema(
				schema,
				`
          {
            someBox {
              ... on SomeBox {
                deepBox {
                  unrelatedField
                }
              }
              ... on StringBox {
                deepBox {
                  unrelatedField
                }
              }
            }
          }
        `,
			)
		})

		It("disallows differing return types despite no overlap", func() {
			expectErrorsWithSchema(
				schema,
				`
          {
            someBox {
              ... on IntBox {
                scalar
              }
              ... on StringBox {
                scalar
              }
            }
          }
        `,
			).Should(Or(
				Equal(graphql.ErrorsOf(
					graphql.NewError(
						fieldsConflictMessage("scalar", "they return conflicting types Int and String"),
						[]graphql.ErrorLocation{
							{Line: 5, Column: 17},
							{Line: 8, Column: 17},
						},
					),
				)),
				Equal(graphql.ErrorsOf(
					graphql.NewError(
						fieldsConflictMessage("scalar", "they return conflicting types String and Int"),
						[]graphql.ErrorLocation{
							{Line: 8, Column: 17},
							{Line: 5, Column: 17},
						},
					),
				))))
		})

		It("reports correctly when a non-exclusive follows an exclusive", func() {
			expectErrorsWithSchema(
				schema,
				`
          {
            someBox {
              ... on IntBox {
                deepBox {
                  ...X
                }
              }
            }
            someBox {
              ... on StringBox {
                deepBox {
                  ...Y
                }
              }
            }
            memoed: someBox {
              ... on IntBox {
                deepBox {
                  ...X
                }
              }
            }
            memoed: someBox {
              ... on StringBox {
                deepBox {
                  ...Y
                }
              }
            }
            other: someBox {
              ...X
            }
            other: someBox {
              ...Y
            }
          }
          fragment X on SomeBox {
            scalar
          }
          fragment Y on SomeBox {
            scalar: unrelatedField
          }
        `,
			).Should(Equal(graphql.ErrorsOf(
				graphql.NewError(
					fieldsConflictMessage("other", []*validator.FieldConflictReason{
						{
							ResponseKey:              "scalar",
							MessageOrSubFieldReasons: "scalar and unrelatedField are different fields",
						},
					}),
					[]graphql.ErrorLocation{
						{Line: 31, Column: 13},
						{Line: 39, Column: 13},
						{Line: 34, Column: 13},
						{Line: 42, Column: 13},
					},
				),
			)))
		})

		It("disallows differing return type nullability despite no overlap", func() {
			expectErrorsWithSchema(
				schema,
				`
          {
            someBox {
              ... on NonNullStringBox1 {
                scalar
              }
              ... on StringBox {
                scalar
              }
            }
          }
        `,
			).Should(Or(
				Equal(graphql.ErrorsOf(
					graphql.NewError(
						fieldsConflictMessage("scalar", "they return conflicting types String! and String"),
						[]graphql.ErrorLocation{
							{Line: 5, Column: 17},
							{Line: 8, Column: 17},
						},
					),
				)),
				Equal(graphql.ErrorsOf(
					graphql.NewError(
						fieldsConflictMessage("scalar", "they return conflicting types String and String!"),
						[]graphql.ErrorLocation{
							{Line: 8, Column: 17},
							{Line: 5, Column: 17},
						},
					),
				)),
			))
		})

		It("disallows differing return type list despite no overlap", func() {
			expectErrorsWithSchema(
				schema,
				`
          {
            someBox {
              ... on IntBox {
                box: listStringBox {
                  scalar
                }
              }
              ... on StringBox {
                box: stringBox {
                  scalar
                }
              }
            }
          }
        `,
			).Should(Or(
				Equal(graphql.ErrorsOf(
					graphql.NewError(
						fieldsConflictMessage("box", "they return conflicting types [StringBox] and StringBox"),
						[]graphql.ErrorLocation{
							{Line: 5, Column: 17},
							{Line: 10, Column: 17},
						},
					),
				)),
				Equal(graphql.ErrorsOf(
					graphql.NewError(
						fieldsConflictMessage("box", "they return conflicting types StringBox and [StringBox]"),
						[]graphql.ErrorLocation{
							{Line: 10, Column: 17},
							{Line: 5, Column: 17},
						},
					),
				)),
			))

			expectErrorsWithSchema(
				schema,
				`
          {
            someBox {
              ... on IntBox {
                box: stringBox {
                  scalar
                }
              }
              ... on StringBox {
                box: listStringBox {
                  scalar
                }
              }
            }
          }
        `,
			).Should(Or(
				Equal(graphql.ErrorsOf(
					graphql.NewError(
						fieldsConflictMessage("box", "they return conflicting types StringBox and [StringBox]"),
						[]graphql.ErrorLocation{
							{Line: 5, Column: 17},
							{Line: 10, Column: 17},
						},
					),
				)),
				Equal(graphql.ErrorsOf(
					graphql.NewError(
						fieldsConflictMessage("box", "they return conflicting types [StringBox] and StringBox"),
						[]graphql.ErrorLocation{
							{Line: 10, Column: 17},
							{Line: 5, Column: 17},
						},
					),
				)),
			))
		})

		It("disallows differing subfields", func() {
			expectErrorsWithSchema(
				schema,
				`
          {
            someBox {
              ... on IntBox {
                box: stringBox {
                  val: scalar
                  val: unrelatedField
                }
              }
              ... on StringBox {
                box: stringBox {
                  val: scalar
                }
              }
            }
          }
        `,
			).Should(Equal(graphql.ErrorsOf(
				graphql.NewError(
					fieldsConflictMessage("val", "scalar and unrelatedField are different fields"),
					[]graphql.ErrorLocation{
						{Line: 6, Column: 19},
						{Line: 7, Column: 19},
					},
				),
			)))
		})

		It("disallows differing deep return types despite no overlap", func() {
			expectErrorsWithSchema(
				schema,
				`
          {
            someBox {
              ... on IntBox {
                box: stringBox {
                  scalar
                }
              }
              ... on StringBox {
                box: intBox {
                  scalar
                }
              }
            }
          }
        `,
			).Should(Or(
				Equal(graphql.ErrorsOf(
					graphql.NewError(
						fieldsConflictMessage("box", []*validator.FieldConflictReason{
							{
								ResponseKey:              "scalar",
								MessageOrSubFieldReasons: "they return conflicting types String and Int",
							},
						}),
						[]graphql.ErrorLocation{
							{Line: 5, Column: 17},
							{Line: 6, Column: 19},
							{Line: 10, Column: 17},
							{Line: 11, Column: 19},
						},
					),
				)),
				Equal(graphql.ErrorsOf(
					graphql.NewError(
						fieldsConflictMessage("box", []*validator.FieldConflictReason{
							{
								ResponseKey:              "scalar",
								MessageOrSubFieldReasons: "they return conflicting types Int and String",
							},
						}),
						[]graphql.ErrorLocation{
							{Line: 10, Column: 17},
							{Line: 11, Column: 19},
							{Line: 5, Column: 17},
							{Line: 6, Column: 19},
						},
					),
				)),
			))
		})

		It("allows non-conflicting overlapping types", func() {
			expectValidWithSchema(
				schema,
				`
          {
            someBox {
              ... on IntBox {
                scalar: unrelatedField
              }
              ... on StringBox {
                scalar
              }
            }
          }
        `,
			)
		})

		It("same wrapped scalar return types", func() {
			expectValidWithSchema(
				schema,
				`
          {
            someBox {
              ...on NonNullStringBox1 {
                scalar
              }
              ...on NonNullStringBox2 {
                scalar
              }
            }
          }
        `,
			)
		})

		It("allows inline typeless fragments", func() {
			expectValidWithSchema(
				schema,
				`
          {
            a
            ... {
              a
            }
          }
        `,
			)
		})

		It("compares deep types including list", func() {
			expectErrorsWithSchema(
				schema,
				`
          {
            connection {
              ...edgeID
              edges {
                node {
                  id: name
                }
              }
            }
          }

          fragment edgeID on Connection {
            edges {
              node {
                id
              }
            }
          }
        `,
			).Should(Equal(graphql.ErrorsOf(
				graphql.NewError(
					fieldsConflictMessage("edges", []*validator.FieldConflictReason{
						{
							ResponseKey: "node",
							MessageOrSubFieldReasons: []*validator.FieldConflictReason{
								{
									ResponseKey:              "id",
									MessageOrSubFieldReasons: "name and id are different fields",
								},
							},
						},
					}),
					[]graphql.ErrorLocation{
						{Line: 5, Column: 15},
						{Line: 6, Column: 17},
						{Line: 7, Column: 19},
						{Line: 14, Column: 13},
						{Line: 15, Column: 15},
						{Line: 16, Column: 17},
					},
				),
			)))
		})

		It("ignores unknown types", func() {
			expectValidWithSchema(
				schema,
				`
          {
            someBox {
              ...on UnknownType {
                scalar
              }
              ...on NonNullStringBox2 {
                scalar
              }
            }
          }
        `,
			)
		})

		It("error message contains hint for alias conflict", func() {
			// The error template should end with a hint for the user to try using
			// different aliases.
			Expect(fieldsConflictMessage("x", "a and b are different fields")).Should(Equal(
				`Fields "x" conflict because a and b are different fields. Use different aliases on the fields to fetch both if this was intentional.`,
			))
		})

		It("works for field names that are JS keywords", func() {
			schemaWithKeywords := graphql.MustNewSchema(&graphql.SchemaConfig{
				Query: graphql.MustNewObject(&graphql.ObjectConfig{
					Name: "Query",
					Fields: graphql.Fields{
						"foo": {
							Type: &graphql.ObjectConfig{
								Name: "Foo",
								Fields: graphql.Fields{
									"constructor": {
										Type: graphql.T(graphql.String()),
									},
								},
							},
						},
					},
				}),
			})

			expectValidWithSchema(
				schemaWithKeywords,
				`
          {
            foo {
              constructor
            }
          }
        `,
			)
		})
	})

	It("does not infinite loop on recursive fragment", func() {
		expectValid(`
      fragment fragA on Human { name, relatives { name, ...fragA } }
    `)
	})

	It("does not infinite loop on immediately recursive fragment", func() {
		expectValid(`
      fragment fragA on Human { name, ...fragA }
    `)
	})

	It("does not infinite loop on transitively recursive fragment", func() {
		expectValid(`
      fragment fragA on Human { name, ...fragB }
      fragment fragB on Human { name, ...fragC }
      fragment fragC on Human { name, ...fragA }
    `)
	})

	It("finds invalid case even with immediately recursive fragment", func() {
		expectErrors(`
      fragment sameAliasesWithDifferentFieldTargets on Dog {
        ...sameAliasesWithDifferentFieldTargets
        fido: name
        fido: nickname
      }
    `).Should(Equal(graphql.ErrorsOf(
			graphql.NewError(
				fieldsConflictMessage("fido", "name and nickname are different fields"),
				[]graphql.ErrorLocation{
					{Line: 4, Column: 9},
					{Line: 5, Column: 9},
				},
			),
		)))
	})
})
