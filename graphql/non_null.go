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
	return &NonNull{}, nil
}

// Finalize implements typeCreator.
func (creator *nonNullTypeCreator) Finalize(t Type, typeDefResolver typeDefinitionResolver) error {
	// Resolve element type.
	elementType, err := typeDefResolver(creator.typeDef.ElementType())
	if err != nil {
		return err
	} else if elementType == nil {
		return NewError("Must provide an non-nil element type for NonNull.")
	} else if !IsNullableType(elementType) {
		return NewError(fmt.Sprintf("Expected a nullable type for NonNull but got an %s.", elementType.String()))
	}

	nonNull := t.(*NonNull)
	nonNull.elementType = elementType
	nonNull.notation = fmt.Sprintf("%s!", elementType.String())
	return nil
}

// nonNullTypeDefinitionOf wraps a TypeDefinition of the element type and implements
// NonNullTypeDefinition.
type nonNullTypeDefinitionOf struct {
	ThisIsNonNullTypeDefinition
	elementTypeDef TypeDefinition
}

var _ NonNullTypeDefinition = nonNullTypeDefinitionOf{}

// ElementType implements NonNullTypeDefinition.
func (typeDef nonNullTypeDefinitionOf) ElementType() TypeDefinition {
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
	ThisIsNonNullTypeDefinition
	elementType Type
}

var _ NonNullTypeDefinition = nonNullTypeDefinitionOfType{}

// ElementType implements NonNullTypeDefinition.
func (typeDef nonNullTypeDefinitionOfType) ElementType() TypeDefinition {
	return T(typeDef.elementType)
}

// NonNullOfType returns a NonNullTypeDefinition with the given Type of element type.
func NonNullOfType(elementType Type) NonNullTypeDefinition {
	return nonNullTypeDefinitionOfType{
		elementType: elementType,
	}
}

// NonNull Type Modifier
//
// A non-null is a wrapping type which points to another type. Non-null types enforce that their
// values are never null and can ensure an error is raised if this ever occurs during a request. It
// is useful for fields which you can make a strong guarantee on non-nullability, for example
// usually the id field of a database row will never be null.
//
// Note: the enforcement of non-nullability occurs within the executor.
//
// Reference: https://facebook.github.io/graphql/June2018/#sec-Type-System.Non-Null
type NonNull struct {
	elementType Type
	// notation is cached value for returning from String() and is initialized in constructor.
	notation string
}

var (
	_ Type = (*NonNull)(nil)
)

// NewNonNullOfType defines a NonNull type from a given Type of element type.
func NewNonNullOfType(elementType Type) (*NonNull, error) {
	return NewNonNull(NonNullOfType(elementType))
}

// MustNewNonNullOfType is a panic-on-fail version of NewNonNullOfType.
func MustNewNonNullOfType(elementType Type) *NonNull {
	return MustNewNonNull(NonNullOfType(elementType))
}

// NewNonNullOf defines a NonNull type from a given TypeDefinition of element type.
func NewNonNullOf(elementTypeDef TypeDefinition) (*NonNull, error) {
	return NewNonNull(NonNullOf(elementTypeDef))
}

// MustNewNonNullOf is a panic-on-fail version of NewNonNullOf.
func MustNewNonNullOf(elementTypeDef TypeDefinition) *NonNull {
	return MustNewNonNull(NonNullOf(elementTypeDef))
}

// NewNonNull defines a NonNull type from a NonNullTypeDefinition.
func NewNonNull(typeDef NonNullTypeDefinition) (*NonNull, error) {
	t, err := newTypeImpl(&nonNullTypeCreator{
		typeDef: typeDef,
	})
	if err != nil {
		return nil, err
	}
	return t.(*NonNull), nil
}

// MustNewNonNull is a convenience function equivalent to NewNonNull but panics on failure instead of
// returning an error.
func MustNewNonNull(typeDef NonNullTypeDefinition) *NonNull {
	n, err := NewNonNull(typeDef)
	if err != nil {
		panic(err)
	}
	return n
}

// graphqlType implements Type.
func (*NonNull) graphqlType() {}

// Values implemennts Type.
func (n *NonNull) String() string {
	return n.notation
}

// ElementType indicates the the type of the element wrapped in this non-null type.
func (n *NonNull) ElementType() Type {
	return n.elementType
}
