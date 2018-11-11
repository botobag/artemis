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
		ScalarType      graphql.Type
		EnumType        graphql.Type
		InterfaceType   graphql.Type
		UnionType       graphql.Type
		InputObjectType graphql.Type
		ObjectType      graphql.Type
	)

	BeforeEach(func() {
		var err error
		ScalarType, err = graphql.NewScalar(&graphql.ScalarConfig{
			Name: "Scalar",
			ResultCoercer: graphql.CoerceScalarResultFunc(
				func(value interface{}) (interface{}, error) {
					return nil, nil
				}),
		})
		Expect(err).ShouldNot(HaveOccurred())

		EnumType, err = graphql.NewEnum(&graphql.EnumConfig{
			Name: "Enum",
		})
		Expect(err).ShouldNot(HaveOccurred())

		InterfaceType, err = graphql.NewInterface(&graphql.InterfaceConfig{
			Name: "Interface",
		})
		Expect(err).ShouldNot(HaveOccurred())

		UnionType, err = graphql.NewUnion(&graphql.UnionConfig{
			Name: "UnionType",
		})
		Expect(err).ShouldNot(HaveOccurred())

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
	Describe("IsScalarType", func() {
		It("returns true for spec defined scalar", func() {
			Expect(graphql.IsScalarType(graphql.String())).Should(BeTrue())
		})

		It("returns true for custom scalar", func() {
			Expect(graphql.IsScalarType(ScalarType)).Should(BeTrue())
		})

		It("returns false for wrapped scalar", func() {
			Expect(graphql.IsScalarType(graphql.MustNewListOfType(ScalarType))).Should(BeFalse())
		})

		It("returns false for non-scalar", func() {
			Expect(graphql.IsScalarType(EnumType)).Should(BeFalse())
		})
	})

	Describe("IsObjectType", func() {
		It("returns true for object type", func() {
			Expect(graphql.IsObjectType(ObjectType)).Should(BeTrue())
		})

		It("returns false for wrapped object type", func() {
			Expect(graphql.IsObjectType(graphql.MustNewListOfType(ObjectType))).Should(BeFalse())
		})

		It("returns false for non-object type", func() {
			Expect(graphql.IsObjectType(InterfaceType)).Should(BeFalse())
		})
	})

	Describe("IsInterfaceType", func() {
		It("returns true for interface type", func() {
			Expect(graphql.IsInterfaceType(InterfaceType)).Should(BeTrue())
		})

		It("returns false for wrapped interface type", func() {
			Expect(graphql.IsInterfaceType(graphql.MustNewListOfType(InterfaceType))).Should(BeFalse())
		})

		It("returns false for non-interface type", func() {
			Expect(graphql.IsInterfaceType(ObjectType)).Should(BeFalse())
		})
	})

	Describe("IsUnionType", func() {
		It("returns true for union type", func() {
			Expect(graphql.IsUnionType(UnionType)).Should(BeTrue())
		})

		It("returns false for wrapped union type", func() {
			Expect(graphql.IsUnionType(graphql.MustNewListOfType(UnionType))).Should(BeFalse())
		})

		It("returns false for non-union type", func() {
			Expect(graphql.IsUnionType(ObjectType)).Should(BeFalse())
		})
	})

	Describe("IsEnumType", func() {
		It("returns true for enum type", func() {
			Expect(graphql.IsEnumType(EnumType)).Should(BeTrue())
		})

		It("returns false for wrapped enum type", func() {
			Expect(graphql.IsEnumType(graphql.MustNewListOfType(EnumType))).Should(BeFalse())
		})

		It("returns false for non-enum type", func() {
			Expect(graphql.IsEnumType(ScalarType)).Should(BeFalse())
		})
	})

	Describe("isInputObjectType", func() {
		It("returns true for input object type", func() {
			Expect(graphql.IsInputObjectType(InputObjectType)).Should(BeTrue())
		})

		It("returns false for wrapped input object type", func() {
			Expect(graphql.IsInputObjectType(graphql.MustNewListOfType(InputObjectType))).Should(BeFalse())
		})

		It("returns false for non-input object type", func() {
			Expect(graphql.IsInputObjectType(ObjectType)).Should(BeFalse())
		})
	})

	Describe("IsListType", func() {
		It("returns true for list wrapper type", func() {
			Expect(graphql.IsListType(graphql.MustNewListOfType(ObjectType))).Should(BeTrue())
		})

		It("returns false for an unwrapped type", func() {
			Expect(graphql.IsListType(ObjectType)).Should(BeFalse())
		})

		It("returns false for a non-list wrapped type", func() {
			Expect(graphql.IsListType(
				graphql.MustNewNonNullOfType(
					graphql.MustNewListOfType(ObjectType)),
			)).Should(BeFalse())
		})
	})

	Describe("IsNonNullType", func() {
		It("returns true for non-null wrapper type", func() {
			Expect(graphql.IsNonNullType(graphql.MustNewNonNullOfType(ObjectType))).Should(BeTrue())
		})

		It("returns false for an unwrapped type", func() {
			Expect(graphql.IsNonNullType(ObjectType)).Should(BeFalse())
		})

		It("returns false for a not non-null wrapped type", func() {
			Expect(graphql.IsNonNullType(
				graphql.MustNewListOfType(
					graphql.MustNewNonNullOfType(ObjectType)),
			)).Should(BeFalse())
		})
	})

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

	Describe("IsOutputType", func() {
		It("returns true for an output type", func() {
			Expect(graphql.IsOutputType(ObjectType)).Should(BeTrue())
		})

		It("returns true for a wrapped output type", func() {
			Expect(graphql.IsOutputType(graphql.MustNewListOfType(ObjectType))).Should(BeTrue())
			Expect(graphql.IsOutputType(graphql.MustNewNonNullOfType(ObjectType))).Should(BeTrue())
		})

		It("returns false for an input type", func() {
			Expect(graphql.IsOutputType(InputObjectType)).Should(BeFalse())
		})

		It("returns false for a wrapped input type", func() {
			Expect(graphql.IsOutputType(graphql.MustNewListOfType(InputObjectType))).Should(BeFalse())
			Expect(graphql.IsOutputType(graphql.MustNewNonNullOfType(InputObjectType))).Should(BeFalse())
		})
	})

	Describe("IsLeafType", func() {
		It("returns true for scalar and enum types", func() {
			Expect(graphql.IsLeafType(ScalarType)).Should(BeTrue())
			Expect(graphql.IsLeafType(EnumType)).Should(BeTrue())
		})

		It("returns false for wrapped leaf type", func() {
			Expect(graphql.IsLeafType(graphql.MustNewListOfType(ScalarType))).Should(BeFalse())
		})

		It("returns false for non-leaf type", func() {
			Expect(graphql.IsLeafType(ObjectType)).Should(BeFalse())
		})

		It("returns false for wrapped non-leaf type", func() {
			Expect(graphql.IsLeafType(graphql.MustNewListOfType(ObjectType))).Should(BeFalse())
		})
	})

	Describe("IsCompositeType", func() {
		It("returns true for object, interface, and union types", func() {
			Expect(graphql.IsCompositeType(ObjectType)).Should(BeTrue())
			Expect(graphql.IsCompositeType(InterfaceType)).Should(BeTrue())
			Expect(graphql.IsCompositeType(UnionType)).Should(BeTrue())
		})

		It("returns false for wrapped composite type", func() {
			Expect(graphql.IsCompositeType(graphql.MustNewListOfType(ObjectType))).Should(BeFalse())
		})

		It("returns false for non-composite type", func() {
			Expect(graphql.IsCompositeType(InputObjectType)).Should(BeFalse())
		})

		It("returns false for wrapped non-composite type", func() {
			Expect(graphql.IsCompositeType(graphql.MustNewListOfType(InputObjectType))).Should(BeFalse())
		})
	})

	Describe("IsAbstractType", func() {
		It("returns true for interface and union types", func() {
			Expect(graphql.IsAbstractType(InterfaceType)).Should(BeTrue())
			Expect(graphql.IsAbstractType(UnionType)).Should(BeTrue())
		})

		It("returns false for wrapped abstract type", func() {
			Expect(graphql.IsAbstractType(graphql.MustNewListOfType(InterfaceType))).Should(BeFalse())
			Expect(graphql.IsAbstractType(graphql.MustNewListOfType(UnionType))).Should(BeFalse())
		})

		It("returns false for non-abstract type", func() {
			Expect(graphql.IsAbstractType(ObjectType)).Should(BeFalse())
		})

		It("returns false for wrapped non-abstract type", func() {
			Expect(graphql.IsAbstractType(graphql.MustNewListOfType(ObjectType))).Should(BeFalse())
		})
	})

	Describe("IsNullableType", func() {
		It("returns true for unwrapped types", func() {
			Expect(graphql.IsNullableType(ObjectType)).Should(BeTrue())
		})

		It("returns true for list of non-null types", func() {
			Expect(graphql.IsNullableType(
				graphql.MustNewListOfType(
					graphql.MustNewNonNullOfType(ObjectType)),
			)).Should(BeTrue())
		})

		It("returns false for non-null types", func() {
			Expect(graphql.IsNullableType(graphql.MustNewNonNullOfType(ObjectType))).Should(BeFalse())
		})
	})

	Describe("NullableTypeOf", func() {
		It("returns nil for no type", func() {
			Expect(graphql.NullableTypeOf(nil)).Should(BeNil())
			Expect(graphql.NullableTypeOf((*graphql.List)(nil))).Should(BeNil())
			Expect(graphql.NullableTypeOf((*graphql.NonNull)(nil))).Should(BeNil())
		})

		It("returns self for a nullable type", func() {
			Expect(graphql.NullableTypeOf(ObjectType)).Should(Equal(ObjectType))
			objectListType := graphql.MustNewListOfType(ObjectType)
			Expect(graphql.NullableTypeOf(objectListType)).Should(Equal(objectListType))
		})

		It("unwraps non-null type", func() {
			Expect(graphql.NullableTypeOf(graphql.MustNewNonNullOfType(ObjectType))).Should(Equal(ObjectType))
		})
	})

	Describe("IsNamedType", func() {
		It("returns true for unwrapped types", func() {
			Expect(graphql.IsNamedType(ObjectType)).Should(BeTrue())
		})

		It("returns false for list and non-null types", func() {
			Expect(graphql.IsNamedType(graphql.MustNewListOfType(ObjectType))).Should(BeFalse())
			Expect(graphql.IsNamedType(graphql.MustNewNonNullOfType(ObjectType))).Should(BeFalse())
		})
	})

	Describe("NamedTypeOf", func() {
		It("returns nil for no type", func() {
			Expect(graphql.NamedTypeOf(nil)).Should(BeNil())
			Expect(graphql.NamedTypeOf((graphql.Scalar)(nil))).Should(BeNil())
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
