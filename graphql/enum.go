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

	"github.com/botobag/artemis/graphql/ast"
)

// EnumResultCoercerFactory creates EnumResultCoercer for an initialized Enum.
type EnumResultCoercerFactory interface {
	// Create is called at the end of NewEnum when Enum is "almost" initialized to obtains an
	// EnumResultCoercer for serializing result value.
	Create(enum *Enum) (EnumResultCoercer, error)
}

// CreateEnumResultCoercerFunc is an adapter to allow the use of ordinary functions as
// EnumResultCoercerFactory.
type CreateEnumResultCoercerFunc func(enum *Enum) (EnumResultCoercer, error)

// Create calls f(enum).
func (f CreateEnumResultCoercerFunc) Create(enum *Enum) (EnumResultCoercer, error) {
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
func (factory defaultEnumResultCoercerLookupByValueFactory) Create(enum *Enum) (EnumResultCoercer, error) {
	// Build valueMap to enable fast lookup.
	values := enum.Values()
	valueMap := make(map[interface{}]*EnumValue, len(values))
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
	enum *Enum

	// When the given value is a pointer and this is set to true, use the value dereferenced
	// from the pointer for searching valueMap.
	deref bool

	// valueMap maps enum value's internal value to the enum value.
	valueMap map[interface{}]*EnumValue
}

var errNoSuchEnumForValue = errors.New("no enum value matches the value")

// Coerce implements EnumResultCoercer.
func (coercer defaultEnumResultCoercerLookupByValue) Coerce(value interface{}) (*EnumValue, error) {
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
	enum *Enum
}

func newDefaultenumResultCoercerLookupByName(enum *Enum) (EnumResultCoercer, error) {
	return defaultEnumResultCoercerLookupByName{enum}, nil
}

var errNoSuchEnumForName = errors.New("no enum value matches the name")

// Coerce implements EnumResultCoercer.
func (coercer defaultEnumResultCoercerLookupByName) Coerce(value interface{}) (*EnumValue, error) {
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
	if value := enum.Value(name); value != nil {
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
	ThisIsEnumTypeDefinition

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
func (config *EnumConfig) NewResultCoercer(enum *Enum) (EnumResultCoercer, error) {
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

	// Create instance and return. Values and nameMap are created in Finalize.
	return &Enum{
		data: data,
	}, nil
}

// Finalize implements typeCreator.
func (creator *enumTypeCreator) Finalize(t Type, typeDefResolver typeDefinitionResolver) error {
	enum := t.(*Enum)
	typeDef := creator.typeDef

	// Define values and build nameMap.
	valueDefMap := enum.data.Values

	values := make([]*EnumValue, len(valueDefMap))
	nameMap := make(map[string]*EnumValue, len(valueDefMap))
	i := 0
	for name, valueDef := range valueDefMap {
		value := &EnumValue{
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
		values[i] = value
		nameMap[name] = value
		i++
	}

	enum.values = values
	enum.nameMap = nameMap

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

// EnumValue provides definition for a value in enum.
//
// Reference: https://facebook.github.io/graphql/June2018/#EnumValue
type EnumValue struct {
	// Name of the num value
	name string

	// Definition of the value
	def EnumValueDefinition
}

// Name of enum value.
func (value *EnumValue) Name() string {
	return value.name
}

// Description of the enum value
func (value *EnumValue) Description() string {
	return value.def.Description
}

// Value returns the internal value to be used when the enum value is read from input.
func (value *EnumValue) Value() interface{} {
	return value.def.Value
}

// IsDeprecated return true if this value is deprecated.
func (value *EnumValue) IsDeprecated() bool {
	return value.def.Deprecation.Defined()
}

// Deprecation is non-nil when the value is tagged as deprecated.
func (value *EnumValue) Deprecation() *Deprecation {
	return value.def.Deprecation
}

// Enum Type Definition
//
// Some leaf values of requests and input values are Enums. GraphQL serializes Enum values as
// strings, however internally Enums can be represented by any kind of type, often integers.
//
// Note: If a value is not provided in a definition, the name of the enum value will be used as its
//			 internal value.
//
// Reference: https://facebook.github.io/graphql/June2018/#sec-Enums
type Enum struct {
	data EnumTypeData

	// resultCoercer coerces result value into an enum value.
	resultCoercer EnumResultCoercer

	// values defined in the enum type
	values []*EnumValue

	// nameMap maps enum name to its EnumValue.
	nameMap map[string]*EnumValue
}

var (
	_ Type                = (*Enum)(nil)
	_ LeafType            = (*Enum)(nil)
	_ TypeWithName        = (*Enum)(nil)
	_ TypeWithDescription = (*Enum)(nil)
)

// NewEnum defines a Enum type from a EnumTypeDefinition.
func NewEnum(typeDef EnumTypeDefinition) (*Enum, error) {
	t, err := newTypeImpl(&enumTypeCreator{
		typeDef: typeDef,
	})
	if err != nil {
		return nil, err
	}
	return t.(*Enum), nil
}

// MustNewEnum is a convenience function equivalent to NewEnum but panics on failure instead of
// returning an error.
func MustNewEnum(typeDef EnumTypeDefinition) *Enum {
	e, err := NewEnum(typeDef)
	if err != nil {
		panic(err)
	}
	return e
}

// graphqlType implements Type.
func (*Enum) graphqlType() {}

// graphqlLeafType implements LeafType.
func (*Enum) graphqlLeafType() {}

// Name implemennts TypeWithName.
func (e *Enum) Name() string {
	return e.data.Name
}

// Description implemennts TypeWithDescription.
func (e *Enum) Description() string {
	return e.data.Description
}

// Values implemennts Type.
func (e *Enum) String() string {
	return e.Name()
}

// Values return all enum values defined in this Enum type.
func (e *Enum) Values() []*EnumValue {
	return e.values
}

// Value finds the enum value with given name or return nil if there's no such one.
func (e *Enum) Value(name string) *EnumValue {
	value, exists := e.nameMap[name]
	if exists {
		return value
	}
	return nil
}

// CoerceResultValue implements LeafType.
func (e *Enum) CoerceResultValue(value interface{}) (interface{}, error) {
	enumValue, err := e.resultCoercer.Coerce(value)
	if err != nil {
		return nil, err
	}
	return enumValue.Name(), nil
}

// These errors are returned when coercion failed in CoerceVariableValue and CoerceArgumentValue.
// These are ordinary error instead of CoercionError to let the caller present default message to
// the user instead of these internal details.
var (
	errNilEnumValue      = errors.New("enum value is not provided")
	errInvalidEnumValue  = errors.New("invalid enum value")
	errEnumValueNotFound = errors.New("not a value for the type")
)

// CoerceVariableValue coerces a value read from input query variable that specifies a name of enum
// value and return the internal value that represents the enum. Return nil if there's no such enum
// value for given name were found.
func (e *Enum) CoerceVariableValue(value interface{}) (interface{}, error) {
	var enumValue *EnumValue
	switch name := value.(type) {
	case string:
		enumValue = e.Value(name)

	case *string:
		if name != nil {
			enumValue = e.Value(*name)
		} else {
			return nil, errNilEnumValue
		}

	default:
		// Check whether the given value is string-like or pointer to string-like via reflection.
		nameValue := reflect.ValueOf(value)
		if nameValue.Kind() == reflect.Ptr {
			if nameValue.IsNil() {
				return nil, errNilEnumValue
			}
			nameValue = nameValue.Elem()
		}

		if nameValue.Kind() != reflect.String {
			return nil, errInvalidEnumValue
		}

		enumValue = e.Value(nameValue.String())
	}

	if enumValue != nil {
		return enumValue.Value(), nil
	}

	return nil, errEnumValueNotFound
}

// CoerceArgumentValue is similar to CoerceVariableValue but coerces a value from input field
// argument that specifies a name of enum value.
func (e *Enum) CoerceArgumentValue(value ast.Value) (interface{}, error) {
	if value, ok := value.(ast.EnumValue); ok {
		if enumValue := e.Value(value.Value()); enumValue != nil {
			return enumValue.Value(), nil
		}
		return nil, errEnumValueNotFound
	}
	return nil, errInvalidEnumValue
}
