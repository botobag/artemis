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
	TypeWithName
	TypeWithDescription

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
	TypeWithName
	TypeWithDescription

	// TypeResolver returns resolver that could determine the concrete Object type for the abstract
	// type from resolved value.
	//
	// Reference: https://facebook.github.io/graphql/June2018/#ResolveAbstractType()
	TypeResolver() TypeResolver

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
// Metafields that are only available in certain types
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

//===----------------------------------------------------------------------------------------====//
// Scalar
//===----------------------------------------------------------------------------------------====//

// Scalar Type Definition
//
// The leaf values of any request and input values to arguments are Scalars (or Enums) and are
// defined with a name and a series of functions used to parse input from ast or variables and to
// ensure validity.
//
// Reference: https://facebook.github.io/graphql/June2018/#sec-Scalars
type Scalar interface {
	LeafType

	// CoerceVariableValue coerces values in input variables into eligible Go values for the scalar.
	CoerceVariableValue(value interface{}) (interface{}, error)

	// CoerceArgumentValue coerces values in field or directive argument into eligible Go values for
	// the scalar.
	CoerceArgumentValue(value ast.Value) (interface{}, error)

	// graphqlScalarType puts a special mark for scalar type.
	graphqlScalarType()
}

// ThisIsScalarType is required to be embedded in struct that intends to be a Scalar.
type ThisIsScalarType struct{}

// graphqlType implements Type.
func (*ThisIsScalarType) graphqlType() {}

// graphqlLeafType implements LeafType.
func (*ThisIsScalarType) graphqlLeafType() {}

// graphqlScalarType implements Scalar.
func (*ThisIsScalarType) graphqlScalarType() {}

// ScalarResultCoercer coerces result value into a value represented in the Scalar type. Please read
// "Result Coercion" in [0] to provide appropriate implementation.
//
// [0]: https://facebook.github.io/graphql/June2018/#sec-Scalars
type ScalarResultCoercer interface {
	// CoerceResultValue coerces the given value for the field to return. It is called in
	// CompleteValue() [0] as per spec.
	//
	// [0]: https://facebook.github.io/graphql/June2018/#CompleteValue()
	CoerceResultValue(value interface{}) (interface{}, error)
}

// CoerceScalarResultFunc is an adapter to allow the use of ordinary functions as
// ScalarResultCoercer.
type CoerceScalarResultFunc func(value interface{}) (interface{}, error)

// CoerceResultValue calls f(value).
func (f CoerceScalarResultFunc) CoerceResultValue(value interface{}) (interface{}, error) {
	return f(value)
}

// CoerceScalarResultFunc implements ScalarResultCoercer.
var _ ScalarResultCoercer = (CoerceScalarResultFunc)(nil)

// ScalarInputCoercer coerces input values in the GraphQL requests into a value represented the
// Scalar type. Please read "Input Coercion" in [0] to provide appropriate implementation.
//
// [0]: https://facebook.github.io/graphql/June2018/#sec-Scalars
type ScalarInputCoercer interface {
	// CoerceVariableValue coerces a scalar value in input query variables [0].
	//
	// [0]: https://facebook.github.io/graphql/June2018/#CoerceVariableValues()
	CoerceVariableValue(value interface{}) (interface{}, error)

	// CoerceArgumentValue coerces a scalar value in input field arguments [0].
	//
	// [0]: https://facebook.github.io/graphql/June2018/#CoerceArgumentValues()
	CoerceArgumentValue(value ast.Value) (interface{}, error)
}

// ScalarInputCoercerFuncs is an adapter to create a ScalarInputCoercer from function values.
type ScalarInputCoercerFuncs struct {
	CoerceVariableValueFunc func(value interface{}) (interface{}, error)
	CoerceArgumentValueFunc func(value ast.Value) (interface{}, error)
}

// CoerceVariableValue calls f.CoerceVariableValueFunc(value).
func (f ScalarInputCoercerFuncs) CoerceVariableValue(value interface{}) (interface{}, error) {
	return f.CoerceVariableValueFunc(value)
}

// CoerceArgumentValue calls f.CoerceArgumentValueFunc(value).
func (f ScalarInputCoercerFuncs) CoerceArgumentValue(value ast.Value) (interface{}, error) {
	return f.CoerceArgumentValueFunc(value)
}

// ScalarInputCoercerFuncs implements ScalarInputCoercer.
var _ ScalarInputCoercer = ScalarInputCoercerFuncs{}

//===----------------------------------------------------------------------------------------====//
// Object
//===----------------------------------------------------------------------------------------====//

// Object Type Definition
//
// GraphQL queries are hierarchical and composed, describing a tree of information. While Scalar
// types describe the leaf values of these hierarchical queries, Objects describe the intermediate
// levels.
//
// Reference: https://facebook.github.io/graphql/June2018/#sec-Objects
type Object interface {
	Type
	TypeWithName
	TypeWithDescription

	// Fields in the object
	Fields() FieldMap

	// Interfaces includes interfaces that implemented by the Object type.
	Interfaces() []Interface

	// graphqlObjectType puts a special mark for an Object type.
	graphqlObjectType()
}

// ThisIsObjectType is required to be embedded in struct that intends to be an Object.
type ThisIsObjectType struct{}

// graphqlType implements Type.
func (*ThisIsObjectType) graphqlType() {}

// graphqlObjectType implements Object.
func (*ThisIsObjectType) graphqlObjectType() {}

//===----------------------------------------------------------------------------------------====//
// Interface
//===----------------------------------------------------------------------------------------====//

// Interface Type Definition
//
// When a field can return one of a heterogeneous set of types, a Interface type is used to describe
// what types are possible, what fields are in common across all types, as well as a function to
// determine which type is actually used when the field is resolved.
//
// Reference: https://facebook.github.io/graphql/June2018/#sec-Interfaces
type Interface interface {
	AbstractType

	// Fields returns set of fields that needs to be provided when implementing this interface.
	Fields() FieldMap

	// graphqlInterfaceType puts a special mark for an Interface type.
	graphqlInterfaceType()
}

// ThisIsInterfaceType is required to be embedded in struct that intends to be an Interface.
type ThisIsInterfaceType struct{}

// graphqlType implements Type.
func (*ThisIsInterfaceType) graphqlType() {}

// graphqlAbstractType implements AbstractType.
func (*ThisIsInterfaceType) graphqlAbstractType() {}

// graphqlInterfaceType implements Interface.
func (*ThisIsInterfaceType) graphqlInterfaceType() {}

//===----------------------------------------------------------------------------------------====//
// Union
//===----------------------------------------------------------------------------------------====//

// Union Type Definition
//
// When a field can return one of a heterogeneous set of types, a Union type is used to describe
// what types are possible as well as providing a function to determine which type is actually used
// when the field is resolved.
//
// Reference: https://facebook.github.io/graphql/June2018/#sec-Unions
type Union interface {
	AbstractType

	// Types returns member of the union type.
	PossibleTypes() []Object

	// graphqlUnionType puts a special mark for an Union type.
	graphqlUnionType()
}

// ThisIsUnionType is required to be embedded in struct that intends to be an Union.
type ThisIsUnionType struct{}

// graphqlType implements Type.
func (*ThisIsUnionType) graphqlType() {}

// graphqlAbstractType implements AbstractType.
func (*ThisIsUnionType) graphqlAbstractType() {}

// graphqlUnionType implements Union.
func (*ThisIsUnionType) graphqlUnionType() {}

//===----------------------------------------------------------------------------------------====//
// Enum
//===----------------------------------------------------------------------------------------====//

// EnumValueMap maps enum value names to their corresponding value definitions in an enum type.
type EnumValueMap map[string]EnumValue

// Lookup finds the enum value with given name or return nil if there's no such one.
func (m EnumValueMap) Lookup(name string) EnumValue {
	return m[name]
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
type Enum interface {
	LeafType

	// Values return all enum values defined in this Enum type.
	Values() EnumValueMap

	// graphqlEnumType puts a special mark for enum type.
	graphqlEnumType()
}

// ThisIsEnumType is required to be embedded in struct that intends to be a Enum.
type ThisIsEnumType struct{}

// graphqlType implements Type.
func (*ThisIsEnumType) graphqlType() {}

// graphqlLeafType implements LeafType.
func (*ThisIsEnumType) graphqlLeafType() {}

// graphqlEnumType implements Enum.
func (*ThisIsEnumType) graphqlEnumType() {}

// EnumValue provides definition for a value in enum.
//
// Reference: https://facebook.github.io/graphql/June2018/#EnumValue
type EnumValue interface {
	// Name of enum value.
	Name() string

	// Description of the enum value
	Description() string

	// Value returns the internal value to be used when the enum value is read from input.
	Value() interface{}

	// Deprecation is non-nil when the value is tagged as deprecated.
	Deprecation() *Deprecation
}

//===------------------------------------------------------------------------------------------===//
// InputObject
//===------------------------------------------------------------------------------------------===//

// InputFieldMap maps field name to the field definition in an Input Object type.
type InputFieldMap map[string]InputField

// InputObject Type Definition
//
// An input object defines a structured collection of fields which may be supplied to a field
// argument. It is essentially an Object type but with some contraints on the fields so it can be
// used as an input argument. More specifically, fields in an Input Object type cannot define
// arguments or contain references to interfaces and unions.
//
// Reference: https://facebook.github.io/graphql/June2018/#sec-Input-Objects
type InputObject interface {
	Type

	Fields() InputFieldMap

	// graphqlInputObjectType puts a special mark for an Input Object type.
	graphqlInputObjectType()
}

// ThisIsInputObjectType is required to be embedded in struct that intends to be a InputObject.
type ThisIsInputObjectType struct{}

// graphqlType implements Type.
func (*ThisIsInputObjectType) graphqlType() {}

// graphqlInputObjectType implements InputObject.
func (*ThisIsInputObjectType) graphqlInputObjectType() {}

// InputField defines a field in an InputObject. It is much simpler than Field because it doesn't
// resolve value nor can have arguments.
type InputField interface {
	// Name of the field
	Name() string

	// Description of the field
	Description() string

	// Type of value yielded by the field
	Type() Type

	// HasDefaultValue returns true if the input field has a default value. Calling DefaultValue when
	// this returns false results an undefined behavior.
	HasDefaultValue() bool

	// DefaultValue specified the value to be assigned to the field when no input is provided.
	DefaultValue() interface{}
}

//===------------------------------------------------------------------------------------------===//
// List
//===------------------------------------------------------------------------------------------===//

// List Type Modifier
//
// A list is a wrapping type which points to another type. Lists are often created within the
// context of defining the fields of an object type.
//
// Reference: https://facebook.github.io/graphql/June2018/#sec-Type-System.List
type List interface {
	WrappingType

	// ElementType indicates the the type of the elements in the list.
	ElementType() Type

	// graphqlListType puts a special mark for a List type.
	graphqlListType()
}

// ThisIsListType is required to be embedded in struct that intends to be a List.
type ThisIsListType struct{}

// graphqlType implements Type.
func (*ThisIsListType) graphqlType() {}

// graphqlWrappingType implements WrappingType.
func (*ThisIsListType) graphqlWrappingType() {}

// graphqlListType implements List.
func (*ThisIsListType) graphqlListType() {}

//===------------------------------------------------------------------------------------------===//
// NonNull
//===------------------------------------------------------------------------------------------===//

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
type NonNull interface {
	WrappingType

	// InnerType indicates the type of the element wrapped in this non-null type.
	InnerType() Type

	// graphqlNonNullType puts a special mark for an NonNull type.
	graphqlNonNullType()
}

// ThisIsNonNullType is required to be embedded in struct that intends to be a NonNull.
type ThisIsNonNullType struct{}

// graphqlType implements Type.
func (*ThisIsNonNullType) graphqlType() {}

// graphqlWrappingType implements WrappingType.
func (*ThisIsNonNullType) graphqlWrappingType() {}

// graphqlNonNullType implements NonNull.
func (*ThisIsNonNullType) graphqlNonNullType() {}

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
		case List:
			if ttype == nil {
				return nil
			}
			t = ttype.ElementType()

		case NonNull:
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
	if t, ok := t.(NonNull); ok && t != nil {
		return t.InnerType()
	}
	return t
}

// IsInputType returns true if the given type is valid for values in input arguments and variables.
//
// Reference: https://facebook.github.io/graphql/June2018/#IsInputType()
func IsInputType(t Type) bool {
	switch NamedTypeOf(t).(type) {
	case Scalar, Enum, InputObject:
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
	case Scalar, Object, Interface, Union, Enum:
		return true
	default:
		return false
	}
}

// IsCompositeType true if the given type is one of object, interface or union.
func IsCompositeType(t Type) bool {
	switch t.(type) {
	case Object, Interface, Union:
		return true
	default:
		return false
	}
}

// IsNullableType returns true if the type accepts null value.
func IsNullableType(t Type) bool {
	_, ok := t.(NonNull)
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
	_, ok := t.(Scalar)
	return ok
}

// IsObjectType returns true if the given type is an Object type.
func IsObjectType(t Type) bool {
	_, ok := t.(Object)
	return ok
}

// IsInterfaceType returns true if the given type is an Interface type.
func IsInterfaceType(t Type) bool {
	_, ok := t.(Interface)
	return ok
}

// IsUnionType returns true if the given type is an Union type.
func IsUnionType(t Type) bool {
	_, ok := t.(Union)
	return ok
}

// IsEnumType returns true if the given type is an Enum type.
func IsEnumType(t Type) bool {
	_, ok := t.(Enum)
	return ok
}

// IsInputObjectType returns true if the given type is an Input Object type.
func IsInputObjectType(t Type) bool {
	_, ok := t.(InputObject)
	return ok
}

// IsListType returns true if the given type is a List type.
func IsListType(t Type) bool {
	_, ok := t.(List)
	return ok
}

// IsNonNullType returns true if the given type is a NonNull type.
func IsNonNullType(t Type) bool {
	_, ok := t.(NonNull)
	return ok
}
