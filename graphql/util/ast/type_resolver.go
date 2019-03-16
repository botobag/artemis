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

package ast

import (
	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/graphql/ast"
)

// TypeResolver is an utility class which tries to resolve type for an AST nodes in a given schema.
type TypeResolver struct {
	Schema graphql.Schema
}

// ResolveType determines Type for an ast.Type.
func (resolver TypeResolver) ResolveType(ttype ast.Type) graphql.Type {
	// A "true" is added to wrapTypes when an ast.ListType is encountered and "false" for ast.NonNullType.
	var (
		wrapTypes []bool
		t         graphql.Type
	)

	// Find the innermost ast.NamedType. Memoize what type we've went through.
named_type_loop:
	for {
		switch astType := ttype.(type) {
		case ast.ListType:
			wrapTypes = append(wrapTypes, true)
			ttype = astType.ItemType

		case ast.NamedType:
			t = resolver.Schema.TypeMap().Lookup(astType.Name.Value())
			break named_type_loop

		case ast.NonNullType:
			wrapTypes = append(wrapTypes, false)
			ttype = astType.Type
		}
	}

	if t != nil {
		// Go through wrapTypes backward to build wrapping type.
		var err error
		for i := len(wrapTypes); i > 0 && err == nil; i-- {
			if wrapTypes[i-1] {
				t, err = graphql.NewListOfType(t)
			} else {
				t, err = graphql.NewNonNullOfType(t)
			}
		}
	}

	return t
}

// ResolveField determines Field for an ast.Field.
func (resolver TypeResolver) ResolveField(parentType graphql.Type, field *ast.Field) graphql.Field {
	// We may not be able to retrieve the parent type statically.
	if parentType == nil {
		return nil
	}

	// Not exactly the same as findFieldDef in executor. In this statically evaluated environment we
	// do not always have an Object type, and need to handle Interface and Union types.
	name := field.Name.Value()

	if parentType == resolver.Schema.Query() {
		if name == graphql.SchemaMetaFieldName {
			return graphql.SchemaMetaFieldDef()
		} else if name == graphql.TypeMetaFieldName {
			return graphql.TypeMetaFieldDef()
		}
	}

	if name == graphql.TypenameMetaFieldName && graphql.IsCompositeType(parentType) {
		return graphql.TypenameMetaFieldDef()
	}

	switch parentType := parentType.(type) {
	case graphql.Object:
		return parentType.Fields()[name]

	case graphql.Interface:
		return parentType.Fields()[name]
	}

	return nil
}
