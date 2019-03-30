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
)

// This files contains definitions of meta-fields for accessing introspection system as per [0] and
// [1]. The fields are implicit and do not appear in any defined types. Executors take special cares
// on them (by matching the field name in the query) to execute introspection queries.
//
// [0]: https://graphql.github.io/graphql-spec/June2018/#sec-Type-Name-Introspection
// [1]: https://graphql.github.io/graphql-spec/June2018/#sec-Schema-Introspection

// List of meta-field names
const (
	SchemaMetaFieldName   = "__schema"
	TypeMetaFieldName     = "__type"
	TypenameMetaFieldName = "__typename"
)

var (
	// Type of __schema meta-field
	schemaMetaFieldType Type

	// Type of __type meta-field
	typeMetaFieldType Type
	// Arguments taken by __type
	typeMetaFieldArgs []Argument

	// Type of __typename meta-field
	typenameMetaFieldType Type
)

func init() {
	schemaMetaFieldType = MustNewNonNullOfType(IntrospectionTypes.Schema())
	typeMetaFieldArgs = []Argument{
		// FIXME: Access private data structure.
		{
			name:  "name",
			ttype: MustNewNonNullOfType(String()),
		},
	}
	typeMetaFieldType = IntrospectionTypes.Type()
	typenameMetaFieldType = MustNewNonNullOfType(String())
}

//===----------------------------------------------------------------------------------------====//
// __schema
//===----------------------------------------------------------------------------------------====//

// schemaMetaField implemens __schema meta-field to access schema introspection system [0]. The
// field looks like,
//
//	__schema: __Schema!
//
// [0]: https://graphql.github.io/graphql-spec/June2018/#sec-Schema-Introspection
type schemaMetaField struct{}

// Name implements Field.
func (schemaMetaField) Name() string {
	return SchemaMetaFieldName
}

// Description implements Field.
func (schemaMetaField) Description() string {
	return "Access the current type schema of this server."
}

// Type implements Field.
func (schemaMetaField) Type() Type {
	return schemaMetaFieldType
}

// Args implements Field.
func (schemaMetaField) Args() []Argument {
	return nil
}

type schemaMetaFieldResolver struct{}

func (schemaMetaFieldResolver) Resolve(ctx context.Context, source interface{}, info ResolveInfo) (interface{}, error) {
	return info.Schema(), nil
}

// Resolver implements Field.
func (schemaMetaField) Resolver() FieldResolver {
	return schemaMetaFieldResolver{}
}

// Deprecation is non-nil when the field is tagged as deprecated.
func (schemaMetaField) Deprecation() *Deprecation {
	return nil
}

//===----------------------------------------------------------------------------------------====//
// __type
//===----------------------------------------------------------------------------------------====//

// typeMetaField implemens __type meta-field to access type introspection system [0]. The field
// looks like,
//
//	__type(name: String!): __Type
//
// [0]: https://graphql.github.io/graphql-spec/June2018/#sec-Schema-Introspection
type typeMetaField struct{}

// Name implements Field.
func (typeMetaField) Name() string {
	return TypeMetaFieldName
}

// Description implements Field.
func (typeMetaField) Description() string {
	return "Request the type information of a single type."
}

// Type implements Field.
func (typeMetaField) Type() Type {
	return typeMetaFieldType
}

// Args implements Field.
func (typeMetaField) Args() []Argument {
	return typeMetaFieldArgs
}

type typeMetaFieldResolver struct{}

func (typeMetaFieldResolver) Resolve(ctx context.Context, source interface{}, info ResolveInfo) (interface{}, error) {
	return info.Schema().TypeMap().Lookup(info.Args().Get("name").(string)), nil
}

// Resolver implements Field.
func (typeMetaField) Resolver() FieldResolver {
	return typeMetaFieldResolver{}
}

// Deprecation is non-nil when the field is tagged as deprecated.
func (typeMetaField) Deprecation() *Deprecation {
	return nil
}

//===----------------------------------------------------------------------------------------====//
// __typename
//===----------------------------------------------------------------------------------------====//

// typenameMetaField implemens __typename meta-field [0] to access the name of the object type being
// queried.
//
// [0]: https://graphql.github.io/graphql-spec/June2018/#sec-Type-Name-Introspection
type typenameMetaField struct{}

// Name implements Field.
func (typenameMetaField) Name() string {
	return TypenameMetaFieldName
}

// Description implements Field.
func (typenameMetaField) Description() string {
	return "The name of the current Object type at runtime."
}

// Type implements Field.
func (typenameMetaField) Type() Type {
	return typenameMetaFieldType
}

// Args implements Field.
func (typenameMetaField) Args() []Argument {
	return nil
}

type typenameMetaFieldResolver struct{}

func (typenameMetaFieldResolver) Resolve(ctx context.Context, source interface{}, info ResolveInfo) (interface{}, error) {
	return info.Object().Name(), nil
}

// Resolver implements Field.
func (typenameMetaField) Resolver() FieldResolver {
	return typenameMetaFieldResolver{}
}

// Deprecation is non-nil when the field is tagged as deprecated.
func (typenameMetaField) Deprecation() *Deprecation {
	return nil
}

// SchemaMetaFieldDef returns the field that is used to introspect schema.
func SchemaMetaFieldDef() Field {
	return schemaMetaField{}
}

// TypeMetaFieldDef returns the field that is used to introspect type definition.
func TypeMetaFieldDef() Field {
	return typeMetaField{}
}

// TypenameMetaFieldDef returns the field that is used to introspect type name.
func TypenameMetaFieldDef() Field {
	return typenameMetaField{}
}
