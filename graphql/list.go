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

// listTypeCreator is given to newTypeImpl for creating a List.
type listTypeCreator struct {
	typeDef ListTypeDefinition
}

// listTypeCreator implements typeCreator.
var _ typeCreator = (*listTypeCreator)(nil)

// TypeDefinition implements typeCreator.
func (creator *listTypeCreator) TypeDefinition() TypeDefinition {
	return creator.typeDef
}

// LoadDataAndNew implements typeCreator.
func (creator *listTypeCreator) LoadDataAndNew() (Type, error) {
	return &list{}, nil
}

// Finalize implements typeCreator.
func (creator *listTypeCreator) Finalize(t Type, typeDefResolver typeDefinitionResolver) error {
	// Resolve element type.
	elementType, err := typeDefResolver(creator.typeDef.ElementType())
	if err != nil {
		return err
	} else if elementType == nil {
		return NewError("Must provide an non-nil element type for List.")
	}

	list := t.(*list)
	list.elementType = elementType
	return nil
}

// listTypeDefinitionOf wraps a TypeDefinition of the element type and implements
// ListTypeDefinition.
type listTypeDefinitionOf struct {
	ThisIsTypeDefinition
	elementTypeDef TypeDefinition
}

var _ ListTypeDefinition = listTypeDefinitionOf{}

// ElementType implements ListTypeDefinition.
func (typeDef listTypeDefinitionOf) ElementType() TypeDefinition {
	return typeDef.elementTypeDef
}

// ListOf returns a ListTypeDefinition with the given TypeDefinition of element type.
func ListOf(elementTypeDef TypeDefinition) ListTypeDefinition {
	return listTypeDefinitionOf{
		elementTypeDef: elementTypeDef,
	}
}

// listTypeDefinitionOfType wraps a Type of the element type and implements
// ListTypeDefinition.
type listTypeDefinitionOfType struct {
	ThisIsTypeDefinition
	elementType Type
}

var _ ListTypeDefinition = listTypeDefinitionOfType{}

// ElementType implements ListTypeDefinition.
func (typeDef listTypeDefinitionOfType) ElementType() TypeDefinition {
	return T(typeDef.elementType)
}

// ListOfType returns a ListTypeDefinition with the given Type of element type.
func ListOfType(elementType Type) ListTypeDefinition {
	return listTypeDefinitionOfType{
		elementType: elementType,
	}
}

// list is our built-in implementation for List. It is configured with and built from
// ListTypeDefinition.
type list struct {
	ThisIsListType
	elementType Type
}

var _ List = (*list)(nil)

// NewListOfType defines a List type from a given Type of element type.
func NewListOfType(elementType Type) (List, error) {
	return NewList(ListOfType(elementType))
}

// MustNewListOfType is a panic-on-fail version of NewListOfType.
func MustNewListOfType(elementType Type) List {
	return MustNewList(ListOfType(elementType))
}

// NewListOf defines a List type from a given TypeDefinition of element type.
func NewListOf(elementTypeDef TypeDefinition) (List, error) {
	return NewList(ListOf(elementTypeDef))
}

// MustNewListOf is a panic-on-fail version of NewListOf.
func MustNewListOf(elementTypeDef TypeDefinition) List {
	return MustNewList(ListOf(elementTypeDef))
}

// NewList defines a List type from a ListTypeDefinition.
func NewList(typeDef ListTypeDefinition) (List, error) {
	t, err := newTypeImpl(&listTypeCreator{
		typeDef: typeDef,
	})
	if err != nil {
		return nil, err
	}
	return t.(List), nil
}

// MustNewList is a convenience function equivalent to NewList but panics on failure instead of
// returning an error.
func MustNewList(typeDef ListTypeDefinition) List {
	l, err := NewList(typeDef)
	if err != nil {
		panic(err)
	}
	return l
}

// UnwrappedType implements WrappingType.
func (l *list) UnwrappedType() Type {
	return l.ElementType()
}

// ElementType implements List.
func (l *list) ElementType() Type {
	return l.elementType
}
