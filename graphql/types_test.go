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

var _ = Describe("Type", func() {
	var (
		InputObjectType graphql.Type
		ObjectType      graphql.Type
	)

	BeforeEach(func() {
		var err error
		InputObjectType, err = graphql.NewInputObject(&graphql.InputObjectConfig{
			Name: "InputObject",
		})
		Expect(err).ShouldNot(HaveOccurred())

		ObjectType, err = graphql.NewObject(&graphql.ObjectConfig{
			Name: "Object",
		})
		Expect(err).ShouldNot(HaveOccurred())
	})

	// graphql-js/src/type/__tests__/predicate-test.js
	Describe("IsInputType", func() {
		It("returns true for an input type", func() {
			Expect(graphql.IsInputType(InputObjectType)).Should(BeTrue())
			Expect(graphql.IsInputType(graphql.Int())).Should(BeTrue())
		})

		It("returns true for a wrapped input type", func() {
			inputObjectListType, err := graphql.NewListOfType(InputObjectType)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(graphql.IsInputType(inputObjectListType)).Should(BeTrue())

			nonNullInputObjectType, err := graphql.NewNonNullOfType(InputObjectType)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(graphql.IsInputType(nonNullInputObjectType)).Should(BeTrue())
		})

		It("returns false for an output type", func() {
			Expect(graphql.IsInputType(ObjectType)).Should(BeFalse())
		})

		It("returns false for a wrapped output type", func() {
			objectListType, err := graphql.NewListOfType(ObjectType)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(graphql.IsInputType(objectListType)).Should(BeFalse())

			nonNullObjectType, err := graphql.NewNonNullOfType(ObjectType)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(graphql.IsInputType(nonNullObjectType)).Should(BeFalse())
		})
	})

	Describe("NamedTypeOf", func() {
		It("returns nil for no type", func() {
			Expect(graphql.NamedTypeOf(nil)).Should(BeNil())
			Expect(graphql.NamedTypeOf((*graphql.List)(nil))).Should(BeNil())
		})

		It("returns self for a unwrapped type", func() {
			Expect(graphql.NamedTypeOf(ObjectType)).Should(Equal(ObjectType))
		})

		It("unwraps wrapper types", func() {
			objectListType, err := graphql.NewListOfType(ObjectType)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(graphql.NamedTypeOf(objectListType)).Should(Equal(ObjectType))

			nonNullObjectType, err := graphql.NewNonNullOfType(ObjectType)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(graphql.NamedTypeOf(nonNullObjectType)).Should(Equal(ObjectType))
		})

		It("unwraps deeply wrapper types", func() {
			var (
				t   graphql.Type
				err error
			)

			t, err = graphql.NewNonNullOfType(ObjectType)
			Expect(err).ShouldNot(HaveOccurred())

			t, err = graphql.NewListOfType(t)
			Expect(err).ShouldNot(HaveOccurred())

			t, err = graphql.NewListOfType(t)
			Expect(err).ShouldNot(HaveOccurred())

			Expect(graphql.NamedTypeOf(t)).Should(Equal(ObjectType))
		})
	})

	Describe("WrappingType", func() {
		It("has an unwrapped type", func() {
			objectListType, err := graphql.NewListOfType(ObjectType)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(objectListType.UnwrappedType()).Should(Equal(ObjectType))

			nonNullObjectType, err := graphql.NewNonNullOfType(ObjectType)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(nonNullObjectType.UnwrappedType()).Should(Equal(ObjectType))
		})
	})
})
