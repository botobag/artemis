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
	"github.com/botobag/artemis/graphql"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// graphql-js/src/utilities/__tests__/typeComparators-test.js@8c96dc8
var _ = Describe("TypeComparators", func() {
	Describe("graphql.IsTypeSubTypeOf", func() {
		testSchema := func(fields graphql.Fields) graphql.Schema {
			return graphql.MustNewSchema(&graphql.SchemaConfig{
				Query: graphql.MustNewObject(&graphql.ObjectConfig{
					Name:   "Query",
					Fields: fields,
				}),
			})
		}

		It("same reference is subtype", func() {
			schema := testSchema(graphql.Fields{
				"field": {
					Type: graphql.T(graphql.String()),
				},
			})
			Expect(graphql.IsTypeSubTypeOf(schema, graphql.String(), graphql.String())).Should(BeTrue())
		})

		It("int is not subtype of float", func() {
			schema := testSchema(graphql.Fields{
				"field": {
					Type: graphql.T(graphql.String()),
				},
			})
			Expect(graphql.IsTypeSubTypeOf(schema, graphql.Int(), graphql.Float())).Should(BeFalse())
		})

		It("non-null is subtype of nullable", func() {
			schema := testSchema(graphql.Fields{
				"field": {
					Type: graphql.T(graphql.String()),
				},
			})
			Expect(
				graphql.IsTypeSubTypeOf(schema, graphql.MustNewNonNullOfType(graphql.Int()), graphql.Int()),
			).Should(BeTrue())
		})

		It("nullable is not subtype of non-null", func() {
			schema := testSchema(graphql.Fields{
				"field": {
					Type: graphql.T(graphql.String()),
				},
			})
			Expect(
				graphql.IsTypeSubTypeOf(schema, graphql.Int(), graphql.MustNewNonNullOfType(graphql.Int())),
			).Should(BeFalse())
		})

		It("item is not subtype of list", func() {
			schema := testSchema(graphql.Fields{
				"field": {
					Type: graphql.T(graphql.String()),
				},
			})
			Expect(
				graphql.IsTypeSubTypeOf(schema, graphql.Int(), graphql.MustNewListOfType(graphql.Int())),
			).Should(BeFalse())
		})

		It("list is not subtype of item", func() {
			schema := testSchema(graphql.Fields{
				"field": {
					Type: graphql.T(graphql.String()),
				},
			})
			Expect(
				graphql.IsTypeSubTypeOf(schema, graphql.MustNewListOfType(graphql.Int()), graphql.Int()),
			).Should(BeFalse())
		})

		It("member is subtype of union", func() {
			member := &graphql.ObjectConfig{
				Name: "Object",
				Fields: graphql.Fields{
					"field": {
						Type: graphql.T(graphql.String()),
					},
				},
			}
			union := &graphql.UnionConfig{
				Name: "Union",
				PossibleTypes: []graphql.ObjectTypeDefinition{
					member,
				},
			}
			schema := testSchema(graphql.Fields{
				"field": {
					Type: union,
				},
			})
			Expect(
				graphql.IsTypeSubTypeOf(schema, graphql.MustNewObject(member), graphql.MustNewUnion(union)),
			).Should(BeTrue())
		})

		It("implementation is subtype of interface", func() {
			iface := &graphql.InterfaceConfig{
				Name: "Interface",
				Fields: graphql.Fields{
					"field": {
						Type: graphql.T(graphql.String()),
					},
				},
			}
			impl := &graphql.ObjectConfig{
				Name:       "Object",
				Interfaces: []graphql.InterfaceTypeDefinition{iface},
				Fields: graphql.Fields{
					"field": {
						Type: graphql.T(graphql.String()),
					},
				},
			}
			schema := testSchema(graphql.Fields{
				"field": {
					Type: impl,
				},
			})
			Expect(
				graphql.IsTypeSubTypeOf(schema, graphql.MustNewObject(impl), graphql.MustNewInterface(iface)),
			).Should(BeTrue())
		})
	})
})
