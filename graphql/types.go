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

// IsInputType returns true if the given type is valid an input field argument.
//
// Reference: https://facebook.github.io/graphql/June2018/#IsInputType()
func IsInputType(t Type) bool {
	switch t.(type) {
	case *Scalar, *Enum:
		return true
	}
	return false
}

// IsNullableType returns true if the type accepts null value.
func IsNullableType(t Type) bool {
	_, ok := t.(*NonNull)
	return !ok
}
