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
type FieldResolver interface {
	Resolve(params *ResolveFieldParams) ResolveFieldResult
}

// FieldResolverFunc is an adapter to allow the use of ordinary functions as FieldResolver.
type FieldResolverFunc func(params *ResolveFieldParams) ResolveFieldResult

// Resolve calls f(params).
func (f FieldResolverFunc) Resolve(params *ResolveFieldParams) ResolveFieldResult {
	return f(params)
}

// FieldResolverFunc implements FieldResolver.
var _ FieldResolver = FieldResolverFunc(nil)

// ResolveFieldResult contains results returned by field resolver.
type ResolveFieldResult struct {
	// Value being resolved as the result
	Value interface{}

	// Error occurred during resolution
	Err error
}

// ArgumentValues contains argument values given to a field.
type ArgumentValues map[string]interface{}

// ResolveFieldParams specifies parameters passed to Field resolver for fetch the result value.
type ResolveFieldParams struct {
	// Source is the "source" value. It contains the value that has been resolved by field's enclosing
	// object.
	Source interface{}

	// ArgumentValues maps name of argument to its value that was given to current GraphQL request.
	ArgValues ArgumentValues

	// Info is a collection of information about the current execution state.
	Info *ResolveInfo

	// Context argument is a context value that is provided to every resolve function within an
	// execution.  It is commonly used to represent an authenticated user, or request-specific caches.
	Context context.Context
}

// ResolveInfo contains collection of information about execution state for resolvers.
type ResolveInfo struct {
	// TODO
}

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
type FieldMap map[string]*Field

// buildFieldMap takes a Fields to build a FieldMap.
func buildFieldMap(fieldConfigMap Fields, typeDefResolver typeDefinitionResolver) (FieldMap, error) {
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

		field := &Field{
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
// Reference: https://facebook.github.io/graphql/June2018/#sec-Field-Arguments
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

// Field representing a field in an object or an interface. It yields a value of a specific type.
//
// Reference: https://facebook.github.io/graphql/June2018/#sec-Objects
type Field struct {
	config FieldConfig
	name   string
	ttype  Type
	args   []Argument
}

// Name of the field
func (f *Field) Name() string {
	return f.name
}

// Description of the field
func (f *Field) Description() string {
	return f.config.Description
}

// Type of value yielded by the field
func (f *Field) Type() Type {
	return f.ttype
}

// Args specifies the definitions of arguments being taken when querying this field.
func (f *Field) Args() []Argument {
	return f.args
}

// Resolver used for resolving the field result from Object source value.
func (f *Field) Resolver() FieldResolver {
	return f.config.Resolver
}

// Deprecation is non-nil when the field is tagged as deprecated.
func (f *Field) Deprecation() *Deprecation {
	return f.config.Deprecation
}
