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
	"errors"
	"fmt"
	"reflect"
)

// EnumResultCoercerFactory creates EnumResultCoercer for an initialized Enum.
type EnumResultCoercerFactory interface {
	// Create is called at the end of NewEnum when Enum is "almost" initialized to obtains an
	// EnumResultCoercer for serializing result value.
	Create(enum Enum) (EnumResultCoercer, error)
}

// CreateEnumResultCoercerFunc is an adapter to allow the use of ordinary functions as
// EnumResultCoercerFactory.
type CreateEnumResultCoercerFunc func(enum Enum) (EnumResultCoercer, error)

// Create calls f(enum).
func (f CreateEnumResultCoercerFunc) Create(enum Enum) (EnumResultCoercer, error) {
	return f(enum)
}

// DefaultEnumResultCoercerLookupStrategy specifies how to search the enum value.
type DefaultEnumResultCoercerLookupStrategy uint

// Enumeration of DefaultEnumResultCoercerLookupStrategy
const (
	// Search with the enum value whose name matches the given value when performing coercion. This is
	// considered faster than by-value and consume less memory usage. This is also the default
	// strategy.
	DefaultEnumResultCoercerLookupByName = iota

	// Search with the enum value whose internal value matches the given value when performing
	// coercion.
	DefaultEnumResultCoercerLookupByValue

	// This is the same as DefaultEnumResultCoercerLookupByValue. Except when the given value is a
	// pointer, it will look up enum value whose internal value matches the value dereferenced from
	// the pointer. This implements graphql-go's strategy.
	DefaultEnumResultCoercerLookupByValueDeref

	// TODO: DefaultEnumResultCoercerLookupByNameThenValue
	// TODO: DefaultEnumResultCoercerLookupByValueThenName
)

// defaultEnumResultCoercerLookupByValueFactory creates coercer for either
// DefaultEnumResultCoercerLookupByValue or DefaultEnumResultCoercerLookupByValueDeref.
type defaultEnumResultCoercerLookupByValueFactory struct {
	// True when creating coercer for DefaultEnumResultCoercerLookupByValueDeref
	deref bool
}

// Create implements EnumResultCoercerFactory.
func (factory defaultEnumResultCoercerLookupByValueFactory) Create(enum Enum) (EnumResultCoercer, error) {
	// Build valueMap to enable fast lookup.
	values := enum.Values()
	valueMap := make(map[interface{}]EnumValue, len(values))
	for _, value := range values {
		valueMap[value.Value()] = value
	}

	return defaultEnumResultCoercerLookupByValue{
		enum:     enum,
		deref:    factory.deref,
		valueMap: valueMap,
	}, nil
}

// defaultEnumResultCoercerLookupByValue implements an EnumResultCoercer which finds enum value
// whose internal value matches the given result value.
type defaultEnumResultCoercerLookupByValue struct {
	enum Enum

	// When the given value is a pointer and this is set to true, use the value dereferenced
	// from the pointer for searching valueMap.
	deref bool

	// valueMap maps enum value's internal value to the enum value.
	valueMap map[interface{}]EnumValue
}

var errNoSuchEnumForValue = errors.New("no enum value matches the value")

// Coerce implements EnumResultCoercer.
func (coercer defaultEnumResultCoercerLookupByValue) Coerce(value interface{}) (EnumValue, error) {
	if coercer.deref {
		v := reflect.ValueOf(value)
		if v.Kind() == reflect.Ptr {
			if !v.IsNil() {
				value = v.Elem().Interface()
			}
		}
	}

	enumValue, exists := coercer.valueMap[value]
	if !exists {
		return nil, NewDefaultResultCoercionError(coercer.enum.Name(), value, errNoSuchEnumForValue)
	}
	return enumValue, nil
}

// defaultEnumResultCoercerLookupByName implements an EnumResultCoercer which expects a string-like
// result value and will return the enum value whose name matches the value.
type defaultEnumResultCoercerLookupByName struct {
	// The subject enum
	enum Enum
}

func newDefaultenumResultCoercerLookupByName(enum Enum) (EnumResultCoercer, error) {
	return defaultEnumResultCoercerLookupByName{enum}, nil
}

var errNoSuchEnumForName = errors.New("no enum value matches the name")

// Coerce implements EnumResultCoercer.
func (coercer defaultEnumResultCoercerLookupByName) Coerce(value interface{}) (EnumValue, error) {
	enum := coercer.enum
	// Quick path for a string.
	name, ok := value.(string)
	if !ok {
		// Maybe value is some type that aliases a string.
		v := reflect.ValueOf(value)
		if v.Kind() != reflect.String {
			// We have no idea.
			return nil, NewDefaultResultCoercionError(coercer.enum.Name(), value,
				fmt.Errorf("unexpected result type `%T`", value))
		}
		// Retrieve the string value.
		name = v.String()
	}

	// Find the value.
	if value := enum.Values().Lookup(name); value != nil {
		return value, nil
	}

	// Return nil result with an error.
	return nil, NewDefaultResultCoercionError(coercer.enum.Name(), value, errNoSuchEnumForName)
}

// DefaultEnumResultCoercerFactory exposes factory to create a defaultEnumResultCoercer.
func DefaultEnumResultCoercerFactory(lookupStrategy DefaultEnumResultCoercerLookupStrategy) EnumResultCoercerFactory {
	switch lookupStrategy {
	case DefaultEnumResultCoercerLookupByName:
		return CreateEnumResultCoercerFunc(newDefaultenumResultCoercerLookupByName)

	case DefaultEnumResultCoercerLookupByValue:
		return defaultEnumResultCoercerLookupByValueFactory{
			deref: false,
		}

	case DefaultEnumResultCoercerLookupByValueDeref:
		return defaultEnumResultCoercerLookupByValueFactory{
			deref: true,
		}
	}

	panic("unknown lookup strategy for default enum value coercer")
}

// EnumConfig provides specification to define an Enum type. It is served as a convenient way to
// create an EnumTypeDefinition for creating an enum type.
type EnumConfig struct {
	ThisIsTypeDefinition

	// Name of the enum type
	Name string

	// Description for the enum type
	Description string

	// Values to be defined in the enum
	Values EnumValueDefinitionMap

	// ResultCoercerFactory creates an EnumResultCoercer to coerce an internal value into enum value
	// into. If not provided, DefaultEnumResultCoercer will be used.
	ResultCoercerFactory EnumResultCoercerFactory
}

var (
	_ TypeDefinition     = (*EnumConfig)(nil)
	_ EnumTypeDefinition = (*EnumConfig)(nil)
)

// TypeData implements EnumTypeDefinition.
func (config *EnumConfig) TypeData() EnumTypeData {
	return EnumTypeData{
		Name:        config.Name,
		Description: config.Description,
		Values:      config.Values,
	}
}

// NewResultCoercer implments EnumTypeDefinition.
func (config *EnumConfig) NewResultCoercer(enum Enum) (EnumResultCoercer, error) {
	factory := config.ResultCoercerFactory
	if factory == nil {
		factory = DefaultEnumResultCoercerFactory(DefaultEnumResultCoercerLookupByName)
	}
	return factory.Create(enum)
}

// enumTypeCreator is given to newTypeImpl for creating a Enum.
type enumTypeCreator struct {
	typeDef EnumTypeDefinition
}

// enumTypeCreator implements typeCreator.
var _ typeCreator = (*enumTypeCreator)(nil)

// TypeDefinition implements typeCreator.
func (creator *enumTypeCreator) TypeDefinition() TypeDefinition {
	return creator.typeDef
}

// LoadDataAndNew implements typeCreator.
func (creator *enumTypeCreator) LoadDataAndNew() (Type, error) {
	typeDef := creator.typeDef
	// Load data.
	data := typeDef.TypeData()

	// Must provide a name.
	if len(data.Name) == 0 {
		return nil, NewError("Must provide name for Enum.")
	}

	// Create instance and return. Value map are created in Finalize.
	return &enum{
		data: data,
	}, nil
}

// Finalize implements typeCreator.
func (creator *enumTypeCreator) Finalize(t Type, typeDefResolver typeDefinitionResolver) error {
	enum := t.(*enum)
	typeDef := creator.typeDef

	// Define values and build value map.
	valueDefMap := enum.data.Values

	values := make(EnumValueMap, len(valueDefMap))
	i := 0
	for name, valueDef := range valueDefMap {
		value := &enumValue{
			name: name,
			def:  valueDef,
		}
		if value.def.Value == nil {
			// Use name for internal value of the enum value.
			value.def.Value = name
		} else if _, ok := value.def.Value.(enumNilValueType); ok {
			// When NilEnumInternalValue is specified, initialize internal value to nil.
			value.def.Value = nil
		}
		values[name] = value
		i++
	}

	enum.values = values

	// Request a result coercer.
	resultCoercer, err := typeDef.NewResultCoercer(enum)
	if err != nil {
		return NewError("Error occurred when preparing object responsible for coercing result value", err)
	}

	if resultCoercer != nil {
		enum.resultCoercer = resultCoercer
	} else {
		// Use the default one which is to return enum value with name that matches given value.
		enum.resultCoercer = defaultEnumResultCoercerLookupByName{enum}
	}

	return nil
}

// enumValue provides definition for a value in enum.
type enumValue struct {
	// Name of the num value
	name string

	// Definition of the value
	def EnumValueDefinition
}

// Name implements EnumValue.
func (value *enumValue) Name() string {
	return value.name
}

// Description implements EnumValue.
func (value *enumValue) Description() string {
	return value.def.Description
}

// Value implements EnumValue.
func (value *enumValue) Value() interface{} {
	return value.def.Value
}

// Deprecation implements EnumValue.
func (value *enumValue) Deprecation() *Deprecation {
	return value.def.Deprecation
}

// enum is our built-in implementation for Enum. It is configured with and built from
// EnumTypeDefinition.
type enum struct {
	ThisIsEnumType

	data EnumTypeData

	// resultCoercer coerces result value into an enum value.
	resultCoercer EnumResultCoercer

	// values maps each enum value in the enum from name to its definition.
	values EnumValueMap
}

var _ Enum = (*enum)(nil)

// NewEnum defines a Enum type from a EnumTypeDefinition.
func NewEnum(typeDef EnumTypeDefinition) (Enum, error) {
	t, err := newTypeImpl(&enumTypeCreator{
		typeDef: typeDef,
	})
	if err != nil {
		return nil, err
	}
	return t.(Enum), nil
}

// MustNewEnum is a convenience function equivalent to NewEnum but panics on failure instead of
// returning an error.
func MustNewEnum(typeDef EnumTypeDefinition) Enum {
	e, err := NewEnum(typeDef)
	if err != nil {
		panic(err)
	}
	return e
}

// String implemennts fmt.Stringer.
func (e *enum) String() string {
	return e.Name()
}

// Name implemennts TypeWithName.
func (e *enum) Name() string {
	return e.data.Name
}

// Description implemennts TypeWithDescription.
func (e *enum) Description() string {
	return e.data.Description
}

// CoerceResultValue implements LeafType.
func (e *enum) CoerceResultValue(value interface{}) (interface{}, error) {
	enumValue, err := e.resultCoercer.Coerce(value)
	if err != nil {
		return nil, err
	}
	return enumValue.Name(), nil
}

// Values implements Enum.
func (e *enum) Values() EnumValueMap {
	return e.values
}
