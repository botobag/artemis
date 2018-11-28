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

// nonNullTypeCreator is given to newTypeImpl for creating a NonNull.
type nonNullTypeCreator struct {
	typeDef NonNullTypeDefinition
}

// nonNullTypeCreator implements typeCreator.
var _ typeCreator = (*nonNullTypeCreator)(nil)

// TypeDefinition implements typeCreator.
func (creator *nonNullTypeCreator) TypeDefinition() TypeDefinition {
	return creator.typeDef
}

// LoadDataAndNew implements typeCreator.
func (creator *nonNullTypeCreator) LoadDataAndNew() (Type, error) {
	return &nonNull{}, nil
}

// Finalize implements typeCreator.
func (creator *nonNullTypeCreator) Finalize(t Type, typeDefResolver typeDefinitionResolver) error {
	// Resolve element type.
	elementType, err := typeDefResolver(creator.typeDef.InnerType())
	if err != nil {
		return err
	} else if elementType == nil {
		return NewError("Must provide an non-nil element type for NonNull.")
	} else if !IsNullableType(elementType) {
		return NewError(fmt.Sprintf("Expected a nullable type for NonNull but got an %s.", elementType.String()))
	}

	nonNull := t.(*nonNull)
	nonNull.elementType = elementType
	nonNull.notation = fmt.Sprintf("%s!", elementType.String())
	return nil
}

// nonNullTypeDefinitionOf wraps a TypeDefinition of the element type and implements
// NonNullTypeDefinition.
type nonNullTypeDefinitionOf struct {
	ThisIsTypeDefinition
	elementTypeDef TypeDefinition
}

var _ NonNullTypeDefinition = nonNullTypeDefinitionOf{}

// InnerType implements NonNullTypeDefinition.
func (typeDef nonNullTypeDefinitionOf) InnerType() TypeDefinition {
	return typeDef.elementTypeDef
}

// NonNullOf returns a NonNullTypeDefinition with the given TypeDefinition of element type.
func NonNullOf(elementTypeDef TypeDefinition) NonNullTypeDefinition {
	return nonNullTypeDefinitionOf{
		elementTypeDef: elementTypeDef,
	}
}

// nonNullTypeDefinitionOfType wraps a Type of the element type and implements
// NonNullTypeDefinition.
type nonNullTypeDefinitionOfType struct {
	ThisIsTypeDefinition
	elementType Type
}

var _ NonNullTypeDefinition = nonNullTypeDefinitionOfType{}

// InnerType implements NonNullTypeDefinition.
func (typeDef nonNullTypeDefinitionOfType) InnerType() TypeDefinition {
	return T(typeDef.elementType)
}

// NonNullOfType returns a NonNullTypeDefinition with the given Type of element type.
func NonNullOfType(elementType Type) NonNullTypeDefinition {
	return nonNullTypeDefinitionOfType{
		elementType: elementType,
	}
}

// nonNull is our built-in implementation for NonNull. It is configured with and built from
// NonNullTypeDefinition.
type nonNull struct {
	ThisIsNonNullType
	elementType Type
	// notation is cached value for returning from String() and is initialized in constructor.
	notation string
}

var _ NonNull = (*nonNull)(nil)

// NewNonNullOfType defines a NonNull type from a given Type of element type.
func NewNonNullOfType(elementType Type) (NonNull, error) {
	return NewNonNull(NonNullOfType(elementType))
}

// MustNewNonNullOfType is a panic-on-fail version of NewNonNullOfType.
func MustNewNonNullOfType(elementType Type) NonNull {
	return MustNewNonNull(NonNullOfType(elementType))
}

// NewNonNullOf defines a NonNull type from a given TypeDefinition of element type.
func NewNonNullOf(elementTypeDef TypeDefinition) (NonNull, error) {
	return NewNonNull(NonNullOf(elementTypeDef))
}

// MustNewNonNullOf is a panic-on-fail version of NewNonNullOf.
func MustNewNonNullOf(elementTypeDef TypeDefinition) NonNull {
	return MustNewNonNull(NonNullOf(elementTypeDef))
}

// NewNonNull defines a NonNull type from a NonNullTypeDefinition.
func NewNonNull(typeDef NonNullTypeDefinition) (NonNull, error) {
	t, err := newTypeImpl(&nonNullTypeCreator{
		typeDef: typeDef,
	})
	if err != nil {
		return nil, err
	}
	return t.(*nonNull), nil
}

// MustNewNonNull is a convenience function equivalent to NewNonNull but panics on failure instead of
// returning an error.
func MustNewNonNull(typeDef NonNullTypeDefinition) NonNull {
	n, err := NewNonNull(typeDef)
	if err != nil {
		panic(err)
	}
	return n
}

// String implemennts fmt.Stringer.
func (n *nonNull) String() string {
	return n.notation
}

// UnwrappedType implements WrappingType.
func (n *nonNull) UnwrappedType() Type {
	return n.InnerType()
}

// InnerType implements NonNull.
func (n *nonNull) InnerType() Type {
	return n.elementType
}
