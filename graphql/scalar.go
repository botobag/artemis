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

	"github.com/botobag/artemis/graphql/ast"
)

// defaultScalarInputCoercer is used for scalar that doesn't provide coercer for processing input values.
type defaultScalarInputCoercer struct {
	scalar *scalar
}

// CoerceVariableValue implements ScalarInputCoercer.
func (coercer *defaultScalarInputCoercer) CoerceVariableValue(value interface{}) (interface{}, error) {
	return value, nil
}

// CoerceArgumentValue implements ScalarInputCoercer.
func (coercer *defaultScalarInputCoercer) CoerceArgumentValue(value ast.Value) (interface{}, error) {
	return nil, NewError(fmt.Sprintf("coercer for the input type %s was not provided", coercer.scalar.Name()))
}

// ScalarConfig provides specification to define a scalar type. It is served as a convenient way to
// create a ScalarTypeDefinition for creating a scalar type.
type ScalarConfig struct {
	ThisIsScalarTypeDefinition

	// Name of the scalar type
	Name string

	// Description of the scalar type
	Description string

	// ResultCoercer serializes value for return in execution result
	ResultCoercer ScalarResultCoercer

	// InputCoercer parses input value given to the scalar field (optional)
	InputCoercer ScalarInputCoercer
}

var (
	_ TypeDefinition       = (*ScalarConfig)(nil)
	_ ScalarTypeDefinition = (*ScalarConfig)(nil)
)

// TypeData implements ScalarTypeDefinition.
func (config *ScalarConfig) TypeData() ScalarTypeData {
	return ScalarTypeData{
		Name:        config.Name,
		Description: config.Description,
	}
}

// NewResultCoercer implments ScalarTypeDefinition.
func (config *ScalarConfig) NewResultCoercer(scalar Scalar) (ScalarResultCoercer, error) {
	return config.ResultCoercer, nil
}

// NewInputCoercer implments ScalarTypeDefinition.
func (config *ScalarConfig) NewInputCoercer(scalar Scalar) (ScalarInputCoercer, error) {
	return config.InputCoercer, nil
}

// scalarTypeCreator is given to newTypeImpl for creating a scalar.
type scalarTypeCreator struct {
	typeDef ScalarTypeDefinition
}

// scalarTypeCreator implements typeCreator.
var _ typeCreator = (*scalarTypeCreator)(nil)

// TypeDefinition implements typeCreator.
func (creator *scalarTypeCreator) TypeDefinition() TypeDefinition {
	return creator.typeDef
}

// LoadDataAndNew implements typeCreator.
func (creator *scalarTypeCreator) LoadDataAndNew() (Type, error) {
	typeDef := creator.typeDef
	// Load data.
	data := typeDef.TypeData()

	// Must provide a name.
	if len(data.Name) == 0 {
		return nil, NewError("Must provide name for Scalar.")
	}

	// Create instance.
	return &scalar{
		data: data,
	}, nil
}

// Finalize implements typeCreator.
func (creator *scalarTypeCreator) Finalize(t Type, typeDefResolver typeDefinitionResolver) error {
	scalar := t.(*scalar)
	typeDef := creator.typeDef

	// Create result coercer.
	resultCoercer, err := typeDef.NewResultCoercer(scalar)
	if err != nil {
		return err
	} else if resultCoercer == nil {
		return NewError(fmt.Sprintf(
			`%v must provide ResultCoercer. If this custom Scalar is also used as an input type, `+
				`ensure InputCoercer is also provided.`, scalar.data.Name))
	}
	scalar.resultCoercer = resultCoercer

	// Create input coercer.
	inputCoercer, err := typeDef.NewInputCoercer(scalar)
	if err != nil {
		return err
	}

	if inputCoercer != nil {
		scalar.inputCoercer = inputCoercer
	} else {
		scalar.inputCoercer = &defaultScalarInputCoercer{scalar}
	}

	return nil
}

// scalar is our built-in implementation for Scalar. It is configured with and built from
// ScalarTypeDefinition.
type scalar struct {
	ThisIsScalarType

	data          ScalarTypeData
	resultCoercer ScalarResultCoercer
	inputCoercer  ScalarInputCoercer
}

var _ Scalar = (*scalar)(nil)

// NewScalar defines a scalar type from a ScalarTypeDefinition.
func NewScalar(typeDef ScalarTypeDefinition) (Scalar, error) {
	t, err := newTypeImpl(&scalarTypeCreator{
		typeDef: typeDef,
	})
	if err != nil {
		return nil, err
	}
	return t.(*scalar), nil
}

// MustNewScalar is a convenience function equivalent to NewScalar but panics on failure instead of
// returning an error.
func MustNewScalar(typeDef ScalarTypeDefinition) Scalar {
	s, err := NewScalar(typeDef)
	if err != nil {
		panic(err)
	}
	return s
}

// String implements fmt.Stringer.
func (s *scalar) String() string {
	return s.Name()
}

// Name implements TypeWithName.
func (s *scalar) Name() string {
	return s.data.Name
}

// Description implements TypeWithDescription.
func (s *scalar) Description() string {
	return s.data.Description
}

// CoerceResultValue implmenets LeafType.
func (s *scalar) CoerceResultValue(value interface{}) (interface{}, error) {
	return s.resultCoercer.CoerceResultValue(value)
}

// CoerceVariableValue implmenets Scalar.
func (s *scalar) CoerceVariableValue(value interface{}) (interface{}, error) {
	return s.inputCoercer.CoerceVariableValue(value)
}

// CoerceArgumentValue implmenets Scalar.
func (s *scalar) CoerceArgumentValue(value ast.Value) (interface{}, error) {
	return s.inputCoercer.CoerceArgumentValue(value)
}
