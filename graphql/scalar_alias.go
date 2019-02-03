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
	"github.com/botobag/artemis/graphql/ast"
)

// defaultScalarInputCoercer is used for scalar that doesn't provide coercer for processing input values.

// ScalarAliasConfig provides specification to define a scalar type. It is served as a convenient way to
// create a ScalarAliasTypeDefinition for creating a scalar type.
type ScalarAliasConfig struct {
	ThisIsTypeDefinition

	AliasFor Scalar

	// ResultCoercer serializes value for return in execution result. If omitted, the newly created
	// ScalarAlias will apply result coercion as its aliasing Scalar.
	ResultCoercer ScalarResultCoercer

	// InputCoercer parses input value given to the scalar field. If omitted, the newly created
	// ScalarAlias will apply input coercion as its aliasing Scalar.
	InputCoercer ScalarInputCoercer
}

var (
	_ TypeDefinition            = (*ScalarAliasConfig)(nil)
	_ ScalarAliasTypeDefinition = (*ScalarAliasConfig)(nil)
)

// TypeData implements ScalarAliasTypeDefinition.
func (config *ScalarAliasConfig) TypeData() ScalarAliasTypeData {
	return ScalarAliasTypeData{
		AliasFor: config.AliasFor,
	}
}

// NewResultCoercer implments ScalarAliasTypeDefinition.
func (config *ScalarAliasConfig) NewResultCoercer(alias ScalarAlias) (ScalarResultCoercer, error) {
	return config.ResultCoercer, nil
}

// NewInputCoercer implments ScalarAliasTypeDefinition.
func (config *ScalarAliasConfig) NewInputCoercer(alias ScalarAlias) (ScalarInputCoercer, error) {
	return config.InputCoercer, nil
}

// scalarAliasTypeCreator is given to newTypeImpl for creating a scalarAlias.
type scalarAliasTypeCreator struct {
	typeDef ScalarAliasTypeDefinition
}

// scalarAliasTypeCreator implements typeCreator.
var _ typeCreator = (*scalarAliasTypeCreator)(nil)

// TypeDefinition implements typeCreator.
func (creator *scalarAliasTypeCreator) TypeDefinition() TypeDefinition {
	return creator.typeDef
}

// LoadDataAndNew implements typeCreator.
func (creator *scalarAliasTypeCreator) LoadDataAndNew() (Type, error) {
	typeDef := creator.typeDef
	// Load data.
	data := typeDef.TypeData()

	// Must provide the type being aliased to.
	if data.AliasFor == nil {
		return nil, NewError("Must provide aliasing Scalar type for ScalarAlias.")
	}

	// Create instance.
	return &scalarAlias{
		Scalar: typeDef.TypeData().AliasFor,
	}, nil
}

// Finalize implements typeCreator.
func (creator *scalarAliasTypeCreator) Finalize(t Type, typeDefResolver typeDefinitionResolver) error {
	alias := t.(*scalarAlias)
	typeDef := creator.typeDef

	// Create result coercer.
	resultCoercer, err := typeDef.NewResultCoercer(alias)
	if err != nil {
		return err
	}
	alias.resultCoercer = resultCoercer

	// Create input coercer.
	inputCoercer, err := typeDef.NewInputCoercer(alias)
	if err != nil {
		return err
	}
	alias.inputCoercer = inputCoercer

	return nil
}

// scalarAlias is our built-in implementation for ScalarAlias. It is configured with and built from
// ScalarAliasTypeDefinition.
type scalarAlias struct {
	Scalar
	resultCoercer ScalarResultCoercer
	inputCoercer  ScalarInputCoercer
}

var _ ScalarAlias = (*scalarAlias)(nil)

// NewScalarAlias defines a scalar type from a ScalarAliasTypeDefinition.
func NewScalarAlias(typeDef ScalarAliasTypeDefinition) (ScalarAlias, error) {
	t, err := newTypeImpl(&scalarAliasTypeCreator{
		typeDef: typeDef,
	})
	if err != nil {
		return nil, err
	}
	return t.(*scalarAlias), nil
}

// MustNewScalarAlias is a convenience function equivalent to NewScalarAlias but panics on failure
// instead of returning an error.
func MustNewScalarAlias(typeDef ScalarAliasTypeDefinition) ScalarAlias {
	s, err := NewScalarAlias(typeDef)
	if err != nil {
		panic(err)
	}
	return s
}

// AliasFor implements ScalarAlias.
func (a *scalarAlias) AliasFor() Scalar {
	return a.Scalar
}

// CoerceResultValue implmenets LeafType.
func (a *scalarAlias) CoerceResultValue(value interface{}) (interface{}, error) {
	if a.resultCoercer == nil {
		return a.Scalar.CoerceResultValue(value)
	}
	return a.resultCoercer.CoerceResultValue(value)
}

// CoerceVariableValue implmenets ScalarAlias.
func (a *scalarAlias) CoerceVariableValue(value interface{}) (interface{}, error) {
	if a.inputCoercer == nil {
		return a.Scalar.CoerceVariableValue(value)
	}
	return a.inputCoercer.CoerceVariableValue(value)
}

// CoerceLiteralValue implmenets ScalarAlias.
func (a *scalarAlias) CoerceLiteralValue(value ast.Value) (interface{}, error) {
	if a.inputCoercer == nil {
		return a.Scalar.CoerceLiteralValue(value)
	}
	return a.inputCoercer.CoerceLiteralValue(value)
}
