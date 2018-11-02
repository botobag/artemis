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

package graphql

import (
	"fmt"
)

// Type interfaces provided by a GraphQL type.
//
// Reference: https://facebook.github.io/graphql/June2018/#sec-Types
type Type interface {
	// String representation when printing the type
	fmt.Stringer

	// graphqlType is a special mark to indicate a Type. It makes sure that only
	// a set of object can be assigned to Type.
	graphqlType()
}

// LeafType can represent a leaf value where execution of the GraphQL hierarchical queries
// terminates. Currently only Scalar and Enum are valid types for leaf nodes in GraphQL. See [0] and
// [1].
//
// [0]: https://facebook.github.io/graphql/June2018/#sec-Scalars
// [1]: https://facebook.github.io/graphql/June2018/#sec-Enums
type LeafType interface {
	Type

	// CoerceResultValue coerces the given value to be returned as result of field with the type.
	CoerceResultValue(value interface{}) (interface{}, error)

	// graphqlLeafType puts a special mark for a GraphQL leaf type.
	graphqlLeafType()
}

// AbstractType indicates a GraphQL abstract type. Namely, interfaces and unions.
//
// Reference: https://facebook.github.io/graphql/June2018/#sec-Types
type AbstractType interface {
	Type

	// graphqlAbstractType puts a special mark for an abstract type.
	graphqlAbstractType()
}

// WrappingType is a type that wraps another type. There are two wrapping type in GraphQL: List and
// NonNull.
//
// Reference: https://facebook.github.io/graphql/draft/#sec-Wrapping-Types
type WrappingType interface {
	Type

	// UnwrappedType returns the type that is wrapped by this type.
	UnwrappedType() Type

	graphqlWrappingType()
}

// Deprecation contains information about deprecation for a field or an enum value.
//
// See https://facebook.github.io/graphql/June2018/#sec-Deprecation.
type Deprecation struct {
	// Reason provides a description of why the subject is deprecated.
	Reason string
}

// Defined returns true if the deprecation is active.
func (d *Deprecation) Defined() bool {
	return d != nil
}

//===----------------------------------------------------------------------------------------====//
// Metafields that only available in certian types
//===----------------------------------------------------------------------------------------====//

// TypeWithName is implemented by the type definition for named type.
type TypeWithName interface {
	// Name of the defining type
	Name() string
}

// TypeWithDescription is implemented by the types that provides description.
type TypeWithDescription interface {
	// Description provides documentation for the type.
	Description() string
}

//===------------------------------------------------------------------------------------------===//
// Type Predication
//===------------------------------------------------------------------------------------------===//

// NamedTypeOf returns the given type if it is a non-wrapping type. Otherwise, return the underlying
// type of a wrapping type.
//
// Reference: https://facebook.github.io/graphql/draft/#sec-Wrapping-Types
func NamedTypeOf(t Type) Type {
	for {
		switch ttype := t.(type) {
		case *List:
			if ttype == nil {
				return nil
			}
			t = ttype.ElementType()

		case *NonNull:
			if ttype == nil {
				return nil
			}
			t = ttype.InnerType()

		default:
			return t
		}
	}
}

// NullableTypeOf return the given type if it is not a non-null type. Otherwise, return the inner
// type of the non-null type.
func NullableTypeOf(t Type) Type {
	if t, ok := t.(*NonNull); ok && t != nil {
		return t.InnerType()
	}
	return t
}

// IsInputType returns true if the given type is valid for values in input arguments and variables.
//
// Reference: https://facebook.github.io/graphql/June2018/#IsInputType()
func IsInputType(t Type) bool {
	switch NamedTypeOf(t).(type) {
	case *Scalar, *Enum, *InputObject:
		return true
	default:
		return false
	}
}

// IsOutputType returns true if the given type is valid for values in field output.
//
// Reference: https://facebook.github.io/graphql/draft/#IsOutputType()
func IsOutputType(t Type) bool {
	switch NamedTypeOf(t).(type) {
	case *Scalar, *Object, *Interface, *Union, *Enum:
		return true
	default:
		return false
	}
}

// IsCompositeType true if the given type is one of object, interface or union.
func IsCompositeType(t Type) bool {
	switch t.(type) {
	case *Object, *Interface, *Union:
		return true
	default:
		return false
	}
}

// IsNullableType returns true if the type accepts null value.
func IsNullableType(t Type) bool {
	_, ok := t.(*NonNull)
	return !ok
}

// IsNamedType returns true if the type is a non-wrapping type.
//
// Reference: https://facebook.github.io/graphql/draft/#sec-Wrapping-Types
func IsNamedType(t Type) bool {
	return !IsWrappingType(t)
}

// The following predications are simple wrappers of type assertions to corresponding class. This
// makes the use of predications in "if" easily.

// IsLeafType returns true if the given type is a leaf.
func IsLeafType(t Type) bool {
	_, ok := t.(LeafType)
	return ok
}

// IsAbstractType returns true if the given type is a abstract.
func IsAbstractType(t Type) bool {
	_, ok := t.(AbstractType)
	return ok
}

// IsWrappingType returns true if the given type is a wrapping type.
func IsWrappingType(t Type) bool {
	_, ok := t.(WrappingType)
	return ok
}

// IsScalarType returns true if the given type is a Scalar type.
func IsScalarType(t Type) bool {
	_, ok := t.(*Scalar)
	return ok
}

// IsObjectType returns true if the given type is an Object type.
func IsObjectType(t Type) bool {
	_, ok := t.(*Object)
	return ok
}

// IsInterfaceType returns true if the given type is an Interface type.
func IsInterfaceType(t Type) bool {
	_, ok := t.(*Interface)
	return ok
}

// IsUnionType returns true if the given type is an Union type.
func IsUnionType(t Type) bool {
	_, ok := t.(*Union)
	return ok
}

// IsEnumType returns true if the given type is an Enum type.
func IsEnumType(t Type) bool {
	_, ok := t.(*Enum)
	return ok
}

// IsInputObjectType returns true if the given type is an Input Object type.
func IsInputObjectType(t Type) bool {
	_, ok := t.(*InputObject)
	return ok
}

// IsListType returns true if the given type is a List type.
func IsListType(t Type) bool {
	_, ok := t.(*List)
	return ok
}

// IsNonNullType returns true if the given type is a NonNull type.
func IsNonNullType(t Type) bool {
	_, ok := t.(*NonNull)
	return ok
}
