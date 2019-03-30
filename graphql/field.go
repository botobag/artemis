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

// FieldResolver resolves field value during execution.
//
// Reference: https://graphql.github.io/graphql-spec/June2018/#ResolveFieldValue()
type FieldResolver interface {
	// Context carries deadlines and cancelation signals.
	//
	// Source is the "source" value. It contains the value that has been resolved by field's enclosing
	// object.
	//
	// Info contains a collection of information about the current execution state.
	Resolve(ctx context.Context, source interface{}, info ResolveInfo) (interface{}, error)
}

// FieldResolverFunc is an adapter to allow the use of ordinary functions as FieldResolver.
type FieldResolverFunc func(ctx context.Context, source interface{}, info ResolveInfo) (interface{}, error)

// Resolve calls f(ctx, source, info).
func (f FieldResolverFunc) Resolve(
	ctx context.Context,
	source interface{},
	info ResolveInfo) (interface{}, error) {
	return f(ctx, source, info)
}

// FieldResolverFunc implements FieldResolver.
var _ FieldResolver = FieldResolverFunc(nil)

// Fields maps field name to its definition. In general, this should be named as "FieldConfigMap".
// However, this type is used frequently so we try to make it shorter to save some typing efforts.
// Unfortunately we cannot offer FieldConfig as Field because the name is used for representing
// fields within Object and Interface types.
type Fields map[string]FieldConfig

// FieldConfig provides definition of a field when defining an object.
type FieldConfig struct {
	// Description of the defining field
	Description string

	// TypeDefinition instance of the defining field; It will be resolved during type initialization.
	Type TypeDefinition

	// Argument configuration of the field
	Args ArgumentConfigMap

	// Resolver for resolving field value during execution
	Resolver FieldResolver

	// Deprecation is non-nil when the value is tagged as deprecated.
	Deprecation *Deprecation
}

// FieldMap maps field name to the Field.
type FieldMap map[string]Field

// BuildFieldMap builds a FieldMap from given Fields.
func BuildFieldMap(fieldConfigMap Fields, typeDefResolver typeDefinitionResolver) (FieldMap, error) {
	numFields := len(fieldConfigMap)
	if numFields == 0 {
		return nil, nil
	}

	fieldMap := make(FieldMap, numFields)
	for name, fieldConfig := range fieldConfigMap {
		// Resolve the type.
		fieldType, err := typeDefResolver(fieldConfig.Type)
		if err != nil {
			return nil, err
		}

		// Build field arguments.
		args, err := buildArguments(fieldConfig.Args, typeDefResolver)
		if err != nil {
			return nil, err
		}

		field := &field{
			config: fieldConfig,
			name:   name,
			ttype:  fieldType,
			args:   args,
		}

		// Add newly created field to fieldMap.
		fieldMap[name] = field
	}

	return fieldMap, nil
}

// Field representing a field in an object or an interface. It yields a value of a specific type.
//
// Reference: https://graphql.github.io/graphql-spec/June2018/#sec-Objects
type Field interface {
	// Name of the field
	Name() string

	// Description of the field
	Description() string

	// Type of value yielded by the field
	Type() Type

	// Args specifies the definitions of arguments being taken when querying this field.
	Args() []Argument

	// Resolver determines the result value for the field from the value resolved by parent Object.
	//
	// Reference: https://graphql.github.io/graphql-spec/June2018/#ResolveFieldValue()
	Resolver() FieldResolver

	// Deprecation is non-nil when the field is tagged as deprecated.
	Deprecation() *Deprecation
}

// field is our built-in implementation for Field.
type field struct {
	config FieldConfig
	name   string
	ttype  Type
	args   []Argument
}

var _ Field = (*field)(nil)

// Name implements Field.
func (f *field) Name() string {
	return f.name
}

// Description implements Field.
func (f *field) Description() string {
	return f.config.Description
}

// Type implements Field.
func (f *field) Type() Type {
	return f.ttype
}

// Args implements Field.
func (f *field) Args() []Argument {
	return f.args
}

// Resolver implements Field.
func (f *field) Resolver() FieldResolver {
	return f.config.Resolver
}

// Deprecation implements Field.
func (f *field) Deprecation() *Deprecation {
	return f.config.Deprecation
}

// ArgumentConfigMap maps argument name to its definition.
type ArgumentConfigMap map[string]ArgumentConfig

// An intentionally internal type for marking a "null" as default value for an argument
type argumentNilValueType int

// NilArgumentDefaultValue is a value that has a special meaning when it is given to the
// DefaultValue in ArgumentDefinition. It sets the argument with default value set to "null". While
// setting DefaultValue to "nil" or not giving it a value means there's no default value. We need
// this trick because using only "nil" cannot tells whether it's an "undefined" or a "null"
// DefaultValue. The constant has an internal type, therefore there's no way to create one outside
// the package.
const NilArgumentDefaultValue argumentNilValueType = 0

// ArgumentConfig provides definition for defining an argument in a field.
type ArgumentConfig struct {
	// Description fo the argument
	Description string

	// Type of the value that can be given to the argument
	Type TypeDefinition

	// DefaultValue specified the value to be assigned to the argument when no value is provided.
	DefaultValue interface{}
}

// buildArguments builds a list of Argument from an ArgumentConfigMap.
func buildArguments(argConfigMap ArgumentConfigMap, typeDefResolver typeDefinitionResolver) ([]Argument, error) {
	numArgs := len(argConfigMap)
	if numArgs == 0 {
		return nil, nil
	}

	argIdx := 0
	args := make([]Argument, numArgs)
	for name, argConfig := range argConfigMap {
		arg := &args[argIdx]

		// Resolve type.
		argType, err := typeDefResolver(argConfig.Type)
		if err != nil {
			return nil, err
		}

		arg.name = name
		arg.description = argConfig.Description
		arg.ttype = argType
		arg.defaultValue = argConfig.DefaultValue

		argIdx++
	}

	return args, nil
}

// Argument is accepted in querying a field to further specify the return value.
//
// Reference: https://graphql.github.io/graphql-spec/June2018/#sec-Field-Arguments
type Argument struct {
	name         string
	description  string
	ttype        Type
	defaultValue interface{}
}

// Name of the argument
func (arg *Argument) Name() string {
	return arg.name
}

// Description of the argument
func (arg *Argument) Description() string {
	return arg.description
}

// Type of the value that can be given to the argument
func (arg *Argument) Type() Type {
	return arg.ttype
}

// HasDefaultValue returns true if the argument has a default value.
func (arg *Argument) HasDefaultValue() bool {
	return arg.defaultValue != nil
}

// DefaultValue specifies the value to be assigned to the argument when no value is provided.
func (arg *Argument) DefaultValue() interface{} {
	// Deal with NilArgumentDefaultValue specially.
	if _, ok := arg.defaultValue.(argumentNilValueType); ok {
		// We have default value which is "null".
		return nil
	}
	return arg.defaultValue
}

// IsRequiredArgument returns true if value must be provided to the argument for execution.
func IsRequiredArgument(arg *Argument) bool {
	return IsNonNullType(arg.Type()) && !arg.HasDefaultValue()
}

// MockArgument creates an Argument object. This is only used in the tests to create an Argument for
// comparing with one in Type instances. We never use this to create an Argument.
func MockArgument(name string, description string, t Type, defaultValue interface{}) Argument {
	return Argument{
		name:         name,
		description:  description,
		ttype:        t,
		defaultValue: defaultValue,
	}
}
