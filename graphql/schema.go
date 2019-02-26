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
	"reflect"

	"github.com/botobag/artemis/graphql/ast"
)

// Contains interfaces and definitions for a GraphQL schema.

// TypeMap keeps track of all named types referenced within the schema.
type TypeMap struct {
	types map[string]Type
}

// Add a type into the map. This is only used by NewSchema to initialize type map incrementally.
func (typeMap TypeMap) add(t Type) error {
	// stack contains types to be added to the map.
	stack := []Type{t}

	for len(stack) > 0 {
		// Pop a type from stack.
		t, stack = stack[len(stack)-1], stack[:len(stack)-1]

		// Skip nil type quickly. Before validation, we may have nil Type or nil type instance wrapped
		// in a Type.
		if t == nil || reflect.ValueOf(t).IsNil() {
			continue
		}

		// For ScalarAlias, we want to add its aliasing Scalar and skip the ScalarAlias.
		if alias, ok := t.(ScalarAlias); ok {
			t = alias.AliasFor()
		}

		// Map type name to corresponding Type.
		if namedType, ok := t.(TypeWithName); ok {
			name := namedType.Name()
			prev, exists := typeMap.types[name]
			if !exists {
				// Add the type into typeMap.
				typeMap.types[name] = t
			} else {
				if prev != t {
					return NewError(fmt.Sprintf(
						"Schema must contain unique named types but contains multiple types named %s.", name))
				}
				// Skip t which has been processed.
				continue
			}
		}

		// Add types referenced by t to stack.
		switch t := t.(type) {
		case Scalar:
			// Nothing to to.

		case Object:
			// Add interfaces.
			for _, iface := range t.Interfaces() {
				stack = append(stack, iface)
			}

			// Add field type and arg type.
			for _, field := range t.Fields() {
				stack = append(stack, field.Type())
				args := field.Args()
				for i := range args {
					stack = append(stack, args[i].Type())
				}
			}

		case Interface:
			// Add field type and arg type.
			for _, field := range t.Fields() {
				stack = append(stack, field.Type())
				args := field.Args()
				for i := range args {
					stack = append(stack, args[i].Type())
				}
			}

		case Union:
			for possibleType := range t.PossibleTypes().types {
				stack = append(stack, possibleType)
			}

		case Enum:
			// Nothing to to.

		case InputObject:
			// Add field type.
			for _, field := range t.Fields() {
				stack = append(stack, field.Type())
			}

		case List:
			stack = append(stack, t.ElementType())
		case NonNull:
			stack = append(stack, t.InnerType())

		case nil:
			// Skip nil type silently.
		default:
			return NewError(fmt.Sprintf("Cannot add %s to schema: unsupported type %T", t, t))
		}
	}

	return nil
}

// Lookup finds a type with given name.
func (typeMap TypeMap) Lookup(name string) Type {
	return typeMap.types[name]
}

// DirectiveList is a list of Directive.
type DirectiveList []*Directive

// Lookup finds a directive with given name in the list.
func (directiveList DirectiveList) Lookup(name string) *Directive {
	for _, directive := range directiveList {
		if directive.Name() == name {
			return directive
		}
	}
	return nil
}

// SchemaConfig contains configuration to define a GraphQL schema.
type SchemaConfig struct {
	// Query, Mutation and Subscription returns GraphQL Root Operation defined by the schema.
	Query        Object
	Mutation     Object
	Subscription Object

	// List of types that are declared in the schema.
	Types []Type

	// List of directives to be added to the schema.
	Directives DirectiveList

	// If true, the standard directives such as @skip will not be incldued in the defining schema. The
	// directives provided in Directives will be the exact list of directives represented and allowed.
	ExcludeStandardDirectives bool

	// TODO: AST node
}

// Schema Definition
//
// A GraphQL service’s collective type system capabilities are referred to as that service’s
// “schema”. A schema is defined in terms of the types and directives it supports as well as the
// root operation types for each kind of operation: query, mutation, and subscription; this
// determines the place in the type system where those operations begin.
//
// Definitions including types and directives in schema are assumed to be immutable after creation.
// This allows us to cache the results for some operations such as PossibleTypes.
//
// Reference: https://facebook.github.io/graphql/June2018/#sec-Schema
type Schema interface {
	// TypeMap keeps track of all named types referenced within the schema.
	TypeMap() TypeMap

	// Directives keeps track of all valid directives within the schema.
	Directives() DirectiveList

	// The following provides root operation types defined in a GraphQL schema.
	//
	// Reference: https://facebook.github.io/graphql/June2018/#sec-Root-Operation-Types
	Query() Object
	Mutation() Object
	Subscription() Object

	// PossibleTypes returns set of possible concrete types for the given abstract type in the schema.
	// For Interface, this contains the list of Object types that implement it. For Union, this
	// contains the list of its member types.
	PossibleTypes(t AbstractType) PossibleTypeSet

	// TypeFromAST returns a graphql.Type that applies to the ast.Type in the given schema For
	// example, if provided the parsed AST node for `[User]`, a graphql.List instance will be
	// returned, containing the type called "User" found in the schema. If a type called "User" is not
	// found in the schema, then nil will be returned.
	TypeFromAST(t ast.Type) Type
}

// schema provides an implementation to Schema which creates schema from a SchemaConfig.
type schema struct {
	// query, mutation and subscription are root operation objects.
	query        Object
	mutation     Object
	subscription Object

	// typeMap contains all named type defined in the schema.
	typeMap TypeMap

	// directives contains all directives defined in the schema.
	directives DirectiveList

	// possibleTypeSets keeps track of set of possible types for all abstract types in the schema.
	possibleTypeSets map[AbstractType]PossibleTypeSet
}

// NewSchema initializes a Schema from the given config.
func NewSchema(config *SchemaConfig) (Schema, error) {
	schema := &schema{
		query:            config.Query,
		mutation:         config.Mutation,
		subscription:     config.Subscription,
		possibleTypeSets: map[AbstractType]PossibleTypeSet{},
	}

	// Add standard directives.
	numDirectives := len(config.Directives)
	if config.ExcludeStandardDirectives {
		schema.directives = make(DirectiveList, numDirectives)
		// Make a copy.
		copy(schema.directives, config.Directives)
	} else {
		standardDirectives := StandardDirectives()
		schema.directives = make(DirectiveList, numDirectives, numDirectives+len(standardDirectives))
		// Make a copy.
		copy(schema.directives, config.Directives)
		// Append standard directives.
		schema.directives = append(schema.directives, standardDirectives...)
	}

	// Build type map now to detect any errors within this schema.
	typeMap := TypeMap{
		types: map[string]Type{},
	}

	// Add root operation types.
	if err := typeMap.add(config.Query); err != nil {
		return nil, err
	}
	if err := typeMap.add(config.Mutation); err != nil {
		return nil, err
	}
	if err := typeMap.add(config.Subscription); err != nil {
		return nil, err
	}

	// TODO: Add __Schema type in introspection.

	// Add built-in types.
	if err := typeMap.add(Int()); err != nil {
		return nil, err
	}
	if err := typeMap.add(Float()); err != nil {
		return nil, err
	}
	if err := typeMap.add(String()); err != nil {
		return nil, err
	}
	if err := typeMap.add(Boolean()); err != nil {
		return nil, err
	}
	if err := typeMap.add(ID()); err != nil {
		return nil, err
	}

	// Visit all enumerated types in config.
	for _, t := range config.Types {
		if err := typeMap.add(t); err != nil {
			return nil, err
		}
	}

	// Visit types referenced by directives.
	for _, directive := range schema.directives {
		args := directive.Args()
		for i := range args {
			if err := typeMap.add(args[i].Type()); err != nil {
				return nil, err
			}
		}
	}

	// Storing the resulting map for reference by the schema.
	schema.typeMap = typeMap

	// Build possibleTypeSets to keep track of all possible types for Interface and Union types.
	possibleTypeSets := schema.possibleTypeSets
	for _, t := range typeMap.types {
		switch t := t.(type) {
		case Object:
			// Create a reverse link from Object's implementing Interface to the Object.
			for _, iface := range t.Interfaces() {
				set, exists := possibleTypeSets[iface]
				if !exists {
					// Create a new PossibleTypeSet for iface.
					set = NewPossibleTypeSet()
					possibleTypeSets[iface] = set
				}
				// Add Object type t to the PossibleTypeSet for iface.
				set.Add(t)
			}

		case Union:
			possibleTypeSets[t] = t.PossibleTypes()
		}
	}

	return schema, nil
}

// TypeMap implements Schema.
func (schema *schema) TypeMap() TypeMap {
	return schema.typeMap
}

// Directives implements Schema.
func (schema *schema) Directives() DirectiveList {
	return schema.directives
}

// Query implements Schema.
func (schema *schema) Query() Object {
	return schema.query
}

// Mutation implements Schema.
func (schema *schema) Mutation() Object {
	return schema.mutation
}

// Subscription implements Schema.
func (schema *schema) Subscription() Object {
	return schema.subscription
}

// PossibleTypes implements Schema.
func (schema *schema) PossibleTypes(t AbstractType) PossibleTypeSet {
	return schema.possibleTypeSets[t]
}

// TypeFromAST implements Schema.
func (schema *schema) TypeFromAST(t ast.Type) Type {
	// Find the innermost ast.NamedType. Memoize what type we've went through.
	var (
		typeName string
		typePath []ast.Type
	)

	for len(typeName) == 0 {
		switch ttype := t.(type) {
		case ast.NamedType:
			typeName = ttype.Name.Value()

		case ast.ListType:
			// Append current type to typePath.
			typePath = append(typePath, t)
			// Continue on inner type.
			t = ttype.ItemType

		case ast.NonNullType:
			typePath = append(typePath, t)
			t = ttype.Type

		default:
			panic("unexpected AST type kind")
		}
	}

	// Find the graphql.Type for the name.
	result := schema.TypeMap().Lookup(typeName)
	if result == nil {
		return nil
	}

	// Go through typePath backward to build wrapping type.
	for len(typePath) > 0 {
		t, typePath = typePath[len(typePath)-1], typePath[:len(typePath)-1]
		if _, ok := t.(ast.ListType); ok {
			result = MustNewListOfType(result)
		} else {
			// Must be a NonNullType.
			result = MustNewNonNullOfType(result)
		}
	}

	return result
}
