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
	"context"
)

// TypeDefinition defines interfaces that are provided by all TypeDefinition variants. See the
// package documentation for more information.
type TypeDefinition interface {
	// ThisIsGraphQLTypeDefinition puts a special mark for a TypeDefinition objects.
	ThisIsGraphQLTypeDefinition()
}

// ThisIsTypeDefinition is a marker struct intended to be embedded in every TypeDefinition
// implementation
type ThisIsTypeDefinition struct{}

// ThisIsGraphQLTypeDefinition implements ThisIsGraphQLTypeDefinition.
func (ThisIsTypeDefinition) ThisIsGraphQLTypeDefinition() {}

//===-----------------------------------------------------------------------------------------====//
// NewType
//===-----------------------------------------------------------------------------------------====//

// NewType creates a Type instance given a TypeDefinition. When type is known, calling a more
// specific version is prefer. For example, if you know you're creating a Scalar, call NewScalar
// with ScalarTypeDefinition.
func NewType(typeDef TypeDefinition) (Type, error) {
	switch typeDef := typeDef.(type) {
	case typeWrapperTypeDefinition:
		return typeDef.Type(), nil

	case interfaceTypeWrapperTypeDefinition:
		return typeDef.Type(), nil

	default:
		return newTypeImpl(newCreatorFor(typeDef))
	}
}

//===-----------------------------------------------------------------------------------------====//
// T Function
//===-----------------------------------------------------------------------------------------====//

// typeWrapperTypeDefinition is a wrapper for Type which implements TypeDefinition interfaces. This
// makes Type to be able to act as a TypeDefinition.
type typeWrapperTypeDefinition struct {
	ThisIsTypeDefinition
	t Type
}

var _ TypeDefinition = typeWrapperTypeDefinition{}

// Type returns the wrapper Type instance.
func (typeDef typeWrapperTypeDefinition) Type() Type {
	return typeDef.t
}

// T converts a Type into TypeDefinition. When a TypeDefinition depends on some types (e.g., Object
// depends on the types of its fields), it specifies corresponding TypeDefinition instances to
// reference dependent types. To accept Type, we create an internal pseudo-TypeDefinition that
// represent a Type instance and expose a function "T" to make create such TypeDefinition. The
// pseudo TypeDefinition is handled specially in NewType implementation.
func T(t Type) TypeDefinition {
	return typeWrapperTypeDefinition{t: t}
}

//===-----------------------------------------------------------------------------------------====//
// I Function
//===-----------------------------------------------------------------------------------------====//

// interfaceTypeWrapperTypeDefinition is a wrapper for InterfaceType which implements
// InterfaceTypeDefinition interfaces. This allows InterfaceType being able to use in specifying
// implementing interfaces when defining Object (i.e., ObjectTypeData.Interfaces).
type interfaceTypeWrapperTypeDefinition struct {
	ThisIsTypeDefinition
	i Interface
}

// TypeData implements InterfaceTypeDefinition.
func (interfaceTypeWrapperTypeDefinition) TypeData() InterfaceTypeData {
	panic("unreachable")
}

// NewTypeResolver implements InterfaceTypeDefinition.
func (interfaceTypeWrapperTypeDefinition) NewTypeResolver(iface Interface) (TypeResolver, error) {
	panic("unreachable")
}

var _ InterfaceTypeDefinition = interfaceTypeWrapperTypeDefinition{}

// Type returns the wrapper Type instance.
func (typeDef interfaceTypeWrapperTypeDefinition) Type() Type {
	return typeDef.i
}

// I is similar to T but convert an InterfaceType into InterfaceTypeDefinition.
func I(i Interface) InterfaceTypeDefinition {
	return interfaceTypeWrapperTypeDefinition{i: i}
}

//===-----------------------------------------------------------------------------------------====//
// Scalar Type Definition
//===-----------------------------------------------------------------------------------------====//

// ScalarTypeData contains type data for Scalar.
type ScalarTypeData struct {
	// The name of the Scalar type
	Name string

	// Description of the Scalar type
	Description string
}

// ScalarTypeDefinition provides data accessors that are required for defining a Scalar.
type ScalarTypeDefinition interface {
	TypeDefinition

	// TypeData reads data from the definition for the defining scalar.
	TypeData() ScalarTypeData

	// NewResultCoercer creates a ScalarResultCoercer instance for the defining Scalar type object
	// during its initialization.
	NewResultCoercer(scalar Scalar) (ScalarResultCoercer, error)

	// NewInputCoercer creates an ScalarInputCoercer instance for the defining Scalar type object
	// during its initialization.
	NewInputCoercer(scalar Scalar) (ScalarInputCoercer, error)
}

//===-----------------------------------------------------------------------------------------====//
// ScalarAlias Type Definition
//===-----------------------------------------------------------------------------------------====//

// ScalarAliasTypeData contains type data for ScalarAlias.
type ScalarAliasTypeData struct {
	// The Scalar type being aliased to. Note that we take Scalar instead of ScalarTypeDefinition here
	// for convenience.
	AliasFor Scalar
}

// ScalarAliasTypeDefinition provides data accessors that are required for defining a ScalarAlias.
type ScalarAliasTypeDefinition interface {
	TypeDefinition

	// TypeData reads data from the definition for the defining scalar type alias.
	TypeData() ScalarAliasTypeData

	// NewResultCoercer creates a ScalarAliasResultCoercer instance for the defining ScalarAlias type
	// object during its initialization.
	NewResultCoercer(scalar ScalarAlias) (ScalarResultCoercer, error)

	// NewInputCoercer creates an ScalarAliasInputCoercer instance for the defining ScalarAlias type
	// object during its initialization.
	NewInputCoercer(scalar ScalarAlias) (ScalarInputCoercer, error)
}

//===-----------------------------------------------------------------------------------------====//
// Enum Type Definition
//===-----------------------------------------------------------------------------------------====//

// EnumValueDefinitionMap maps enum name to its value definition.
type EnumValueDefinitionMap map[string]EnumValueDefinition

// An intentionally internal type for marking a enum value with nil value.
type enumNilValueType int

// NilEnumInternalValue is a value that has a special meaning when it is given to the Value field
// in EnumValueDefinition. By default, when the nil is given in Value field (this can also be
// happened when user doesn't specify value for the field), the internal value for created enum
// value will be initialized to its enum name. When this special value is used, the internal value
// will set to nil.
const NilEnumInternalValue enumNilValueType = 0

// EnumValueDefinition provides definition to an enum value.
type EnumValueDefinition struct {
	// Description of the enum value
	Description string

	// Value contains an internal value to represent the enum value. If omitted, the value will be set
	// to the name of enum value.
	Value interface{}

	// Deprecation is non-nil when the value is tagged as deprecated.
	Deprecation *Deprecation
}

// EnumTypeData contains type data for Enum.
type EnumTypeData struct {
	// The name of the Enum type
	Name string

	// Description of the Enum type
	Description string

	// Values to be defined in the Enum type
	Values EnumValueDefinitionMap
}

// EnumResultCoercer implements serialization for an Enum type. See comments for Coerce function.
type EnumResultCoercer interface {
	// Given a result value of execution, it finds corresponding enum value from the given enum that
	// represents it.
	Coerce(value interface{}) (EnumValue, error)
}

// CoerceEnumResultFunc is an adapter to allow the use of ordinary functions as EnumResultCoercer.
type CoerceEnumResultFunc func(value interface{}) (EnumValue, error)

// Coerce calls f(enum, value).
func (f CoerceEnumResultFunc) Coerce(value interface{}) (EnumValue, error) {
	return f(value)
}

// EnumTypeDefinition provides data accessors that are required for defining a Enum.
type EnumTypeDefinition interface {
	TypeDefinition

	// TypeData reads data from the definition for the defining enum.
	TypeData() EnumTypeData

	// NewResultCoercer creates a EnumResultCoercer instance for the defining Enum type object during
	// its initialization.
	NewResultCoercer(enum Enum) (EnumResultCoercer, error)
}

//===-----------------------------------------------------------------------------------------====//
// Object Type Definition
//===-----------------------------------------------------------------------------------------====//

// ObjectTypeData contains type data for Object.
type ObjectTypeData struct {
	// The name of the Object type
	Name string

	// Description of the Object type
	Description string

	// Interfaces that implemented by the defining Object
	Interfaces []InterfaceTypeDefinition

	// Fields in the Object Type
	Fields Fields
}

// ObjectTypeDefinition provides data accessors that are required for defining a Object.
type ObjectTypeDefinition interface {
	TypeDefinition

	// TypeData reads data from the definition for the defining enum.
	TypeData() ObjectTypeData
}

//===-----------------------------------------------------------------------------------------====//
// Interface Type Definition
//===-----------------------------------------------------------------------------------------====//

// InterfaceTypeData contains type data for Interface.
type InterfaceTypeData struct {
	// The name of the Interface type
	Name string

	// Description of the Interface type
	Description string

	// Fields in the Interface Type
	Fields Fields
}

// TypeResolver resolves concrete type of an Interface from given value.
type TypeResolver interface {
	// Context carries deadlines and cancelation signals.
	//
	// Value is the value returning from the field resolver of the field with abstract type that is
	// being resolved. Usually you determine the concrete Object type based on the value.
	//
	// Info contains a collection of information about the current execution state.
	//
	// Reference: https://graphql.github.io/graphql-spec/June2018/#ResolveAbstractType()
	Resolve(ctx context.Context, value interface{}, info ResolveInfo) (Object, error)
}

// TypeResolverFunc is an adapter to allow the use of ordinary functions as TypeResolver.
type TypeResolverFunc func(ctx context.Context, value interface{}, info ResolveInfo) (Object, error)

// Resolve calls f(ctx, value, info).
func (f TypeResolverFunc) Resolve(ctx context.Context, value interface{}, info ResolveInfo) (Object, error) {
	return f(ctx, value, info)
}

// TypeResolverFunc implements TypeResolver.
var _ TypeResolver = TypeResolverFunc(nil)

// InterfaceTypeDefinition provides data accessors that are required for defining a Interface.
type InterfaceTypeDefinition interface {
	TypeDefinition

	// TypeData reads data from the definition for the defining enum.
	TypeData() InterfaceTypeData

	// NewTypeResolver creates a TypeResolver instance for the defining Interface during its
	// initialization.
	NewTypeResolver(iface Interface) (TypeResolver, error)
}

//===-----------------------------------------------------------------------------------------====//
// Union Type Definition
//===-----------------------------------------------------------------------------------------====//

// UnionTypeData contains type data for Union.
type UnionTypeData struct {
	// The name of the Union type
	Name string

	// Description of the Union type
	Description string

	// PossibleTypes describes which Object types can be represented by the defining union.
	PossibleTypes []ObjectTypeDefinition
}

// UnionTypeDefinition provides data accessors that are required for defining a Union.
type UnionTypeDefinition interface {
	TypeDefinition

	// TypeData reads data from the definition for the defining enum.
	TypeData() UnionTypeData

	// NewTypeResolver creates a TypeResolver instance for the defining Union during its
	// initialization.
	NewTypeResolver(union Union) (TypeResolver, error)
}

//===-----------------------------------------------------------------------------------------====//
// InputObject Type Definition
//===-----------------------------------------------------------------------------------------====//

// InputObjectTypeData contains type data for InputObject.
type InputObjectTypeData struct {
	// The name of the Input Object type
	Name string

	// Description of the Input Object type
	Description string

	// Fields in the InputObject Type
	Fields InputFields
}

// InputObjectTypeDefinition provides data accessors that are required for defining a InputObject.
type InputObjectTypeDefinition interface {
	TypeDefinition

	// TypeData reads data from the definition for the defining enum.
	TypeData() InputObjectTypeData
}

//===-----------------------------------------------------------------------------------------====//
// List Type Definition
//===-----------------------------------------------------------------------------------------====//

// ListTypeDefinition provides data accessors that are required for defining a List.
type ListTypeDefinition interface {
	TypeDefinition

	// ElementType specifies the type being wrapped in the List type.
	ElementType() TypeDefinition
}

//===-----------------------------------------------------------------------------------------====//
// NonNull Type Definition
//===-----------------------------------------------------------------------------------------====//

// NonNullTypeDefinition provides data accessors that are required for defining a NonNull.
type NonNullTypeDefinition interface {
	TypeDefinition

	// InnerType specifies the type being wrapped in the NonNull type.
	InnerType() TypeDefinition
}
