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
	"github.com/botobag/artemis/graphql"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Type System: Schema", func() {
	// graphql-js/src/type/__tests__/schema-test.js
	var (
		InterfaceType             graphql.Interface
		DirectiveInputType        graphql.InputObject
		WrappedDirectiveInputType graphql.InputObject
		Directive                 *graphql.Directive
		Schema                    *graphql.Schema
	)

	BeforeEach(func() {
		var err error

		InterfaceType, err = graphql.NewInterface(&graphql.InterfaceConfig{
			Name: "Interface",
			Fields: graphql.Fields{
				"fieldName": {
					Type: graphql.T(graphql.String()),
				},
			},
		})
		Expect(err).ShouldNot(HaveOccurred())

		DirectiveInputType, err = graphql.NewInputObject(&graphql.InputObjectConfig{
			Name: "DirInput",
			Fields: graphql.InputFields{
				"field": {
					Type: graphql.T(graphql.String()),
				},
			},
		})
		Expect(err).ShouldNot(HaveOccurred())

		WrappedDirectiveInputType, err = graphql.NewInputObject(&graphql.InputObjectConfig{
			Name: "WrappedDirInput",
			Fields: graphql.InputFields{
				"field": {
					Type: graphql.T(graphql.String()),
				},
			},
		})
		Expect(err).ShouldNot(HaveOccurred())

		Directive, err = graphql.NewDirective(&graphql.DirectiveConfig{
			Name: "dir",
			Locations: []graphql.DirectiveLocation{
				graphql.DirectiveLocationObject,
			},
			Args: graphql.ArgumentConfigMap{
				"arg": {
					Type: graphql.T(DirectiveInputType),
				},
				"argList": {
					Type: graphql.ListOfType(WrappedDirectiveInputType),
				},
			},
		})
		Expect(err).ShouldNot(HaveOccurred())

		query, err := graphql.NewObject(&graphql.ObjectConfig{
			Name: "Query",
			Fields: graphql.Fields{
				"getObject": {
					Type: graphql.T(InterfaceType),
				},
			},
		})
		Expect(err).ShouldNot(HaveOccurred())

		Schema, err = graphql.NewSchema(&graphql.SchemaConfig{
			Query: query,
			Directives: graphql.DirectiveList{
				Directive,
			},
		})
		Expect(err).ShouldNot(HaveOccurred())
	})

	Describe("Type Map", func() {
		It("includes input types only used in directives", func() {
			Expect(Schema.TypeMap().Lookup("DirInput")).ShouldNot(BeNil())
			Expect(Schema.TypeMap().Lookup("WrappedDirInput")).ShouldNot(BeNil())
		})
	})

	It("defines a Schema", func() {
		Expect(Schema.Query()).ShouldNot(BeNil())
		Expect(Schema.Query().Name()).Should(Equal("Query"))

		Expect(Schema.Mutation()).Should(BeNil())
		Expect(Schema.Subscription()).Should(BeNil())
	})

	Describe("Directives", func() {
		It("includes standard directives by default", func() {
			for _, directive := range graphql.StandardDirectives() {
				Expect(Schema.Directives()).Should(ContainElement(directive))
			}
		})

		Context("when ExcludeStandardDirectives is set", func() {
			It("does not include standard directives", func() {
				schema, err := graphql.NewSchema(&graphql.SchemaConfig{
					ExcludeStandardDirectives: true,
				})
				Expect(err).ShouldNot(HaveOccurred())
				for _, directive := range graphql.StandardDirectives() {
					Expect(schema.Directives()).ShouldNot(ContainElement(directive))
				}
			})
		})
	})

})
