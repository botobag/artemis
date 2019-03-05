/**
 * Copyright (c) 2019, The Artemis Authors.
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
	"fmt"

	"github.com/botobag/artemis/internal/util"
	"github.com/botobag/artemis/iterator"
)

var (
	_schema            Object
	_directive         Object
	_directiveLocation Enum
	_type              Object
	_field             Object
	_inputValue        Object
	_enumValue         Object
	_typeKind          Enum
)

//===----------------------------------------------------------------------------------------====//
// __Schema
//===----------------------------------------------------------------------------------------====//

var _schemaDefinition = &ObjectConfig{
	Name: "__Schema",
	Description: "A GraphQL Schema defines the capabilities of a GraphQL server. It exposes all" +
		"available types and directives on the server, as well as the entry points for query, " +
		"mutation, and subscription operations.",
	Fields: Fields{
		"types": {
			Description: "A list of all types supported by this server.",
			Type:        NonNullOf(ListOf(NonNullOf(_typeDefinition))),
			Resolver: FieldResolverFunc(func(ctx context.Context, source interface{}, info ResolveInfo) (interface{}, error) {
				return source.(Schema).TypeMap(), nil
			}),
		},
		"queryType": {
			Description: "The type that query operations will be rooted at.",
			Type:        NonNullOf(_typeDefinition),
			Resolver: FieldResolverFunc(func(ctx context.Context, source interface{}, info ResolveInfo) (interface{}, error) {
				return source.(Schema).Query(), nil
			}),
		},
		"mutationType": {
			Description: "If this server supports mutation, the type that mutation operations will be " +
				"rooted at.",
			Type: _typeDefinition,
			Resolver: FieldResolverFunc(func(ctx context.Context, source interface{}, info ResolveInfo) (interface{}, error) {
				return source.(Schema).Mutation(), nil
			}),
		},
		"subscriptionType": {
			Description: "If this server support subscription, the type that subscription operations " +
				"will be rooted at.",
			Type: _typeDefinition,
			Resolver: FieldResolverFunc(func(ctx context.Context, source interface{}, info ResolveInfo) (interface{}, error) {
				return source.(Schema).Subscription(), nil
			}),
		},
		"directives": {
			Description: "A list of all directives supported by this server.",
			Type:        NonNullOf(ListOf(NonNullOf(_directiveDefinition))),
			Resolver: FieldResolverFunc(func(ctx context.Context, source interface{}, info ResolveInfo) (interface{}, error) {
				return source.(Schema).Directives(), nil
			}),
		},
	},
}

//===----------------------------------------------------------------------------------------====//
// __Directive
//===----------------------------------------------------------------------------------------====//

var _directiveDefinition = &ObjectConfig{
	Name: "__Directive",
	Description: "'A Directive provides a way to describe alternate runtime execution and type " +
		"validation behavior in a GraphQL document.\n\nIn some cases, you need to provide options to " +
		"alter GraphQL's execution behavior in ways field arguments will not suffice, such as " +
		"conditionally including or skipping a field. Directives provide this by describing " +
		"additional information to the executor.",
	Fields: Fields{
		"name": {
			Type: NonNullOfType(String()),
			Resolver: FieldResolverFunc(func(ctx context.Context, source interface{}, info ResolveInfo) (interface{}, error) {
				return source.(Directive).Name(), nil
			}),
		},
		"description": {
			Type: T(String()),
			Resolver: FieldResolverFunc(func(ctx context.Context, source interface{}, info ResolveInfo) (interface{}, error) {
				return source.(Directive).Description(), nil
			}),
		},
		"locations": {
			Type: NonNullOf(ListOf(NonNullOf(_directiveLocationDefinition))),
			Resolver: FieldResolverFunc(func(ctx context.Context, source interface{}, info ResolveInfo) (interface{}, error) {
				return source.(Directive).Locations(), nil
			}),
		},
		"args": {
			Type: NonNullOf(ListOf(NonNullOf(_inputValueDefinition))),
			Resolver: FieldResolverFunc(func(ctx context.Context, source interface{}, info ResolveInfo) (interface{}, error) {
				return _argsIterable{source.(Directive).Args()}, nil
			}),
		},
	},
}

//===----------------------------------------------------------------------------------------====//
// __DirectiveLocation
//===----------------------------------------------------------------------------------------====//

var _directiveLocationDefinition = &EnumConfig{
	Name: "__DirectiveLocation",
	Description: "A Directive can be adjacent to many parts of the GraphQL language, a " +
		"__DirectiveLocation describes one such possible adjacencies.",
	Values: EnumValueDefinitionMap{
		"QUERY": {
			Value:       DirectiveLocationQuery,
			Description: "Location adjacent to a query operation.",
		},
		"MUTATION": {
			Value:       DirectiveLocationMutation,
			Description: "Location adjacent to a mutation operation.",
		},
		"SUBSCRIPTION": {
			Value:       DirectiveLocationSubscription,
			Description: "Location adjacent to a subscription operation.",
		},
		"FIELD": {
			Value:       DirectiveLocationField,
			Description: "Location adjacent to a field.",
		},
		"FRAGMENT_DEFINITION": {
			Value:       DirectiveLocationFragmentDefinition,
			Description: "Location adjacent to a fragment definition.",
		},
		"FRAGMENT_SPREAD": {
			Value:       DirectiveLocationFragmentSpread,
			Description: "Location adjacent to a fragment spread.",
		},
		"INLINE_FRAGMENT": {
			Value:       DirectiveLocationInlineFragment,
			Description: "Location adjacent to an inline fragment.",
		},
		"VARIABLE_DEFINITION": {
			Value:       DirectiveLocationVariableDefinition,
			Description: "Location adjacent to a variable definition.",
		},
		"SCHEMA": {
			Value:       DirectiveLocationSchema,
			Description: "Location adjacent to a schema definition.",
		},
		"SCALAR": {
			Value:       DirectiveLocationScalar,
			Description: "Location adjacent to a scalar definition.",
		},
		"OBJECT": {
			Value:       DirectiveLocationObject,
			Description: "Location adjacent to an object type definition.",
		},
		"FIELD_DEFINITION": {
			Value:       DirectiveLocationFieldDefinition,
			Description: "Location adjacent to a field definition.",
		},
		"ARGUMENT_DEFINITION": {
			Value:       DirectiveLocationArgumentDefinition,
			Description: "Location adjacent to an argument definition.",
		},
		"INTERFACE": {
			Value:       DirectiveLocationInterface,
			Description: "Location adjacent to an interface definition.",
		},
		"UNION": {
			Value:       DirectiveLocationUnion,
			Description: "Location adjacent to a union definition.",
		},
		"ENUM": {
			Value:       DirectiveLocationEnum,
			Description: "Location adjacent to an enum definition.",
		},
		"ENUM_VALUE": {
			Value:       DirectiveLocationEnumValue,
			Description: "Location adjacent to an enum value definition.",
		},
		"INPUT_OBJECT": {
			Value:       DirectiveLocationInputObject,
			Description: "Location adjacent to an input object type definition.",
		},
		"INPUT_FIELD_DEFINITION": {
			Value:       DirectiveLocationInputFieldDefinition,
			Description: "Location adjacent to an input object field definition.",
		},
	},
}

//===----------------------------------------------------------------------------------------====//
// __Type
//===----------------------------------------------------------------------------------------====//

var _typeDefinition = &ObjectConfig{
	Name: "__Type",
	Description: "The fundamental unit of any GraphQL Schema is the type. There are many kinds of" +
		"types in GraphQL as represented by the `__TypeKind` enum.\n\nDepending on the kind of a " +
		"type, certain fields describe information about that type. Scalar types provide no " +
		"information beyond a name and description, while Enum types provide their values. Object " +
		"and Interface types provide the fields they describe. Abstract types, Union and Interface, " +
		"provide the Object types possible at runtime. List and NonNull types compose other types.",
	// Fields are initialized in init to break cyclic dependencies between _typeDefinition and
	// others (e.g., _inputValueDefinition).
}

//===----------------------------------------------------------------------------------------====//
// __Field
//===----------------------------------------------------------------------------------------====//

var _fieldDefinition = &ObjectConfig{
	Name: "__Field",
	Description: "Object and Interface types are described by a list of Fields, each of which has " +
		"a name, potentially a list of arguments, and a return type.",
	Fields: Fields{
		"name": {
			Type: NonNullOfType(String()),
			Resolver: FieldResolverFunc(func(ctx context.Context, source interface{}, info ResolveInfo) (interface{}, error) {
				return source.(Field).Name(), nil
			}),
		},
		"description": {
			Type: T(String()),
			Resolver: FieldResolverFunc(func(ctx context.Context, source interface{}, info ResolveInfo) (interface{}, error) {
				return source.(Field).Description(), nil
			}),
		},
		"args": {
			Type: NonNullOf(ListOf(NonNullOf(_inputValueDefinition))),
			Resolver: FieldResolverFunc(func(ctx context.Context, source interface{}, info ResolveInfo) (interface{}, error) {
				return _argsIterable{source.(Field).Args()}, nil
			}),
		},
		"type": {
			Type: NonNullOf(_typeDefinition),
			Resolver: FieldResolverFunc(func(ctx context.Context, source interface{}, info ResolveInfo) (interface{}, error) {
				return source.(Field).Type(), nil
			}),
		},
		"isDeprecated": {
			Type: NonNullOfType(Boolean()),
			Resolver: FieldResolverFunc(func(ctx context.Context, source interface{}, info ResolveInfo) (interface{}, error) {
				return source.(Field).Deprecation().Defined(), nil
			}),
		},
		"deprecationReason": {
			Type: T(String()),
			Resolver: FieldResolverFunc(func(ctx context.Context, source interface{}, info ResolveInfo) (interface{}, error) {
				deprecation := source.(Field).Deprecation()
				if deprecation.Defined() {
					return deprecation.Reason, nil
				}
				return nil, nil
			}),
		},
	},
}

//===----------------------------------------------------------------------------------------====//
// __InputValue
//===----------------------------------------------------------------------------------------====//

type inputValue interface {
	Name() string
	Description() string
	Type() Type
	HasDefaultValue() bool
	DefaultValue() interface{}
}

var _ inputValue = (InputField)(nil)
var _ inputValue = (*Argument)(nil)

var _inputValueDefinition = &ObjectConfig{
	Name: "__InputValue",
	Description: "Arguments provided to Fields or Directives and the input fields of an " +
		"InputObject are represented as Input Values which describe their type and optionally a " +
		"default value.",
	Fields: Fields{
		"name": {
			Type: NonNullOfType(String()),
			Resolver: FieldResolverFunc(func(ctx context.Context, source interface{}, info ResolveInfo) (interface{}, error) {
				return source.(inputValue).Name(), nil
			}),
		},
		"description": {
			Type: T(String()),
			Resolver: FieldResolverFunc(func(ctx context.Context, source interface{}, info ResolveInfo) (interface{}, error) {
				return source.(inputValue).Description(), nil
			}),
		},
		"type": {
			Type: NonNullOf(_typeDefinition),
			Resolver: FieldResolverFunc(func(ctx context.Context, source interface{}, info ResolveInfo) (interface{}, error) {
				return source.(inputValue).Type(), nil
			}),
		},
		"defaultValue": {
			Type:        T(String()),
			Description: "A GraphQL-formatted string representing the default value for this",
			Resolver: FieldResolverFunc(func(ctx context.Context, source interface{}, info ResolveInfo) (interface{}, error) {
				value := source.(inputValue)
				if !value.HasDefaultValue() {
					return nil, nil
				}
				// FIXME: Inconsistent implementation from graphql-js.
				return value.DefaultValue(), nil
			}),
		},
	},
}

//===----------------------------------------------------------------------------------------====//
// __EnumValue
//===----------------------------------------------------------------------------------------====//

var _enumValueDefinition = &ObjectConfig{
	Name: "__EnumValue",
	Description: "One possible value for a given Enum. Enum values are unique values, not a " +
		"placeholder for a string or numeric value. However an Enum value is returned in a JSON " +
		"response as a string.",
	Fields: Fields{
		"name": {
			Type: NonNullOfType(String()),
			Resolver: FieldResolverFunc(func(ctx context.Context, source interface{}, info ResolveInfo) (interface{}, error) {
				return source.(EnumValue).Name(), nil
			}),
		},
		"description": {
			Type: T(String()),
			Resolver: FieldResolverFunc(func(ctx context.Context, source interface{}, info ResolveInfo) (interface{}, error) {
				return source.(EnumValue).Description(), nil
			}),
		},
		"isDeprecated": {
			Type: NonNullOfType(Boolean()),
			Resolver: FieldResolverFunc(func(ctx context.Context, source interface{}, info ResolveInfo) (interface{}, error) {
				return source.(EnumValue).Deprecation().Defined(), nil
			}),
		},
		"deprecationReason": {
			Type: T(String()),
			Resolver: FieldResolverFunc(func(ctx context.Context, source interface{}, info ResolveInfo) (interface{}, error) {
				deprecation := source.(EnumValue).Deprecation()
				if deprecation.Defined() {
					return deprecation.Reason, nil
				}
				return nil, nil
			}),
		},
	},
}

//===----------------------------------------------------------------------------------------====//
// __TypeKind
//===----------------------------------------------------------------------------------------====//

type typeKindEnum string

const (
	scalarTypeKind      typeKindEnum = "SCALAR"
	objectTypeKind                   = "OBJECT"
	interfaceTypeKind                = "INTERFACE"
	unionTypeKind                    = "UNION"
	enumTypeKind                     = "ENUM"
	inputObjectTypeKind              = "INPUT_OBJECT"
	listTypeKind                     = "LIST"
	nonNullTypeKind                  = "NON_NULL"
)

var _typeKindDefinition = &EnumConfig{
	Name:        "__TypeKind",
	Description: "An enum describing what kind of type a given `__Type` is.",
	Values: EnumValueDefinitionMap{
		"SCALAR": {
			Value:       scalarTypeKind,
			Description: "Indicates this type is a scalar.",
		},
		"OBJECT": {
			Value:       objectTypeKind,
			Description: "Indicates this type is an object. `fields` and `interfaces` are valid fields.",
		},
		"INTERFACE": {
			Value:       interfaceTypeKind,
			Description: "Indicates this type is an interface. `fields` and `possibleTypes` are valid fields.",
		},
		"UNION": {
			Value:       unionTypeKind,
			Description: "Indicates this type is a union. `possibleTypes` is a valid field.",
		},
		"ENUM": {
			Value:       enumTypeKind,
			Description: "Indicates this type is an enum. `enumValues` is a valid field.",
		},
		"INPUT_OBJECT": {
			Value:       inputObjectTypeKind,
			Description: "Indicates this type is an input object. `inputFields` is a valid field.",
		},
		"LIST": {
			Value:       listTypeKind,
			Description: "Indicates this type is a list. `ofType` is a valid field.",
		},
		"NON_NULL": {
			Value:       nonNullTypeKind,
			Description: "Indicates this type is a non-null. `ofType` is a valid field.",
		},
	},
}

//===----------------------------------------------------------------------------------------====//
// _fieldsIterable
//===----------------------------------------------------------------------------------------====//

type _fieldsIterable struct {
	fields            FieldMap
	includeDeprecated bool
}

func (iterable _fieldsIterable) Iterator() Iterator {
	if iterable.includeDeprecated {
		return NewMapValuesIterator(iterable.fields)
	}
	return noDeprecatedFieldsIter{util.NewImmutableMapIter(iterable.fields)}
}

type noDeprecatedFieldsIter struct {
	fieldIter *util.ImmutableMapIter
}

// Next implements Iterator.
func (iter noDeprecatedFieldsIter) Next() (interface{}, error) {
	fieldIter := iter.fieldIter
	for fieldIter.Next() {
		field := fieldIter.Value().Interface().(Field)
		if !field.Deprecation().Defined() {
			return field, nil
		}
	}
	return nil, iterator.Done
}

//===----------------------------------------------------------------------------------------====//
// _argsIterable
//===----------------------------------------------------------------------------------------====//

type _argsIterable struct {
	args []Argument
}

func (iterable _argsIterable) Iterator() Iterator {
	return &argsIter{
		args: iterable.args,
	}
}

type argsIter struct {
	args    []Argument
	nextIdx int
}

// Next implements Iterator.
func (iter *argsIter) Next() (interface{}, error) {
	var (
		args    = iter.args
		nextIdx = iter.nextIdx
	)
	if nextIdx >= len(args) {
		return nil, iterator.Done
	}
	iter.nextIdx++
	return &args[nextIdx], nil
}

//===----------------------------------------------------------------------------------------====//
// _enumValuesIterable
//===----------------------------------------------------------------------------------------====//

type _enumValuesIterable struct {
	values            EnumValueMap
	includeDeprecated bool
}

func (iterable _enumValuesIterable) Iterator() Iterator {
	if iterable.includeDeprecated {
		return NewMapValuesIterator(iterable.values)
	}
	return noDeprecatedEnumValuesIter{util.NewImmutableMapIter(iterable.values)}
}

type noDeprecatedEnumValuesIter struct {
	valueIter *util.ImmutableMapIter
}

// Next implements Iterator.
func (iter noDeprecatedEnumValuesIter) Next() (interface{}, error) {
	valueIter := iter.valueIter
	for valueIter.Next() {
		value := valueIter.Value().Interface().(EnumValue)
		if !value.Deprecation().Defined() {
			return value, nil
		}
	}
	return nil, iterator.Done
}

//===----------------------------------------------------------------------------------------====//
// _inputFieldsIterable
//===----------------------------------------------------------------------------------------====//

type _inputFieldsIterable struct {
	fields InputFieldMap
}

func (iterable _inputFieldsIterable) Iterator() Iterator {
	return NewMapValuesIterator(iterable.fields)
}

func init() {
	_typeDefinition.Fields = Fields{
		"kind": {
			Type: NonNullOf(_typeKindDefinition),
			Resolver: FieldResolverFunc(func(ctx context.Context, source interface{}, info ResolveInfo) (interface{}, error) {
				t := source.(Type)
				switch {
				case IsScalarType(t):
					return scalarTypeKind, nil
				case IsObjectType(t):
					return objectTypeKind, nil
				case IsInterfaceType(t):
					return interfaceTypeKind, nil
				case IsUnionType(t):
					return unionTypeKind, nil
				case IsEnumType(t):
					return enumTypeKind, nil
				case IsInputObjectType(t):
					return inputObjectTypeKind, nil
				case IsListType(t):
					return listTypeKind, nil
				case IsNonNullType(t):
					return nonNullTypeKind, nil
				default:
					return nil, NewError(fmt.Sprintf(`Unexpected type: "%s"`, Inspect(t)))
				}
			}),
		},
		"name": {
			Type: T(String()),
			Resolver: FieldResolverFunc(func(ctx context.Context, source interface{}, info ResolveInfo) (interface{}, error) {
				if t, ok := source.(TypeWithName); ok {
					return t.Name(), nil
				}
				return nil, nil
			}),
		},
		"description": {
			Type: T(String()),
			Resolver: FieldResolverFunc(func(ctx context.Context, source interface{}, info ResolveInfo) (interface{}, error) {
				if t, ok := source.(TypeWithDescription); ok {
					return t.Description(), nil
				}
				return nil, nil
			}),
		},
		"fields": {
			Type: ListOf(NonNullOf(_fieldDefinition)),
			Args: ArgumentConfigMap{
				"includeDeprecated": {
					Type:         T(Boolean()),
					DefaultValue: false,
				},
			},
			Resolver: FieldResolverFunc(func(ctx context.Context, source interface{}, info ResolveInfo) (interface{}, error) {
				switch t := source.(type) {
				case Object:
					return _fieldsIterable{
						fields:            t.Fields(),
						includeDeprecated: info.Args().Get("includeDeprecated").(bool),
					}, nil

				case Interface:
					return _fieldsIterable{
						fields:            t.Fields(),
						includeDeprecated: info.Args().Get("includeDeprecated").(bool),
					}, nil
				}
				return nil, nil
			}),
		},
		"interfaces": {
			Type: ListOf(NonNullOf(_typeDefinition)),
			Resolver: FieldResolverFunc(func(ctx context.Context, source interface{}, info ResolveInfo) (interface{}, error) {
				if t, ok := source.(Object); ok {
					interfaces := t.Interfaces()
					if interfaces != nil {
						return interfaces, nil
					}
					return []Interface{}, nil
				}
				return nil, nil
			}),
		},
		"possibleTypes": {
			Type: ListOf(NonNullOf(_typeDefinition)),
			Resolver: FieldResolverFunc(func(ctx context.Context, source interface{}, info ResolveInfo) (interface{}, error) {
				if t, ok := source.(AbstractType); ok {
					return info.Schema().PossibleTypes(t), nil
				}
				return nil, nil
			}),
		},
		"enumValues": {
			Type: ListOf(NonNullOf(_enumValueDefinition)),
			Args: ArgumentConfigMap{
				"includeDeprecated": {
					Type:         T(Boolean()),
					DefaultValue: false,
				},
			},
			Resolver: FieldResolverFunc(func(ctx context.Context, source interface{}, info ResolveInfo) (interface{}, error) {
				if enum, ok := source.(Enum); ok {
					return _enumValuesIterable{
						values:            enum.Values(),
						includeDeprecated: info.Args().Get("includeDeprecated").(bool),
					}, nil
				}
				return nil, nil
			}),
		},
		"inputFields": {
			Type: ListOf(NonNullOf(_inputValueDefinition)),
			Resolver: FieldResolverFunc(func(ctx context.Context, source interface{}, info ResolveInfo) (interface{}, error) {
				if t, ok := source.(InputObject); ok {
					return _inputFieldsIterable{t.Fields()}, nil
				}
				return nil, nil
			}),
		},
		"ofType": {
			Type: _typeDefinition,
			Resolver: FieldResolverFunc(func(ctx context.Context, source interface{}, info ResolveInfo) (interface{}, error) {
				if t, ok := source.(WrappingType); ok {
					return t.UnwrappedType(), nil
				}
				return nil, nil
			}),
		},
	}

	_schema = MustNewObject(_schemaDefinition)
	_directive = MustNewObject(_directiveDefinition)
	_directiveLocation = MustNewEnum(_directiveLocationDefinition)
	_type = MustNewObject(_typeDefinition)
	_field = MustNewObject(_fieldDefinition)
	_inputValue = MustNewObject(_inputValueDefinition)
	_enumValue = MustNewObject(_enumValueDefinition)
	_typeKind = MustNewEnum(_typeKindDefinition)
}

type introspectionTypes struct{}

// Schema returns _Schema type.
func (introspectionTypes) Schema() Object {
	return _schema
}

// Directive returns _Directive type.
func (introspectionTypes) Directive() Object {
	return _directive
}

// Type returns _Type type.
func (introspectionTypes) Type() Object {
	return _type
}

// Field returns _Field type.
func (introspectionTypes) Field() Object {
	return _field
}

// InputValue returns _InputValue type.
func (introspectionTypes) InputValue() Object {
	return _inputValue
}

// EnumValue returns _EnumValue type.
func (introspectionTypes) EnumValue() Object {
	return _enumValue
}

// IntrospectionTypes provides accessors to the types used specifically for introspection.
var IntrospectionTypes = introspectionTypes{}
