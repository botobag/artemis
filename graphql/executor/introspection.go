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

package executor

import (
	"context"

	"github.com/botobag/artemis/graphql"
)

// This files contains definitions of meta-fields for accessing introspection system as per [0] and
// [1]. The fields are implicit and do not appear in any defined types. Executors take special cares
// on them (by matching the field name in the query) to execute introspection queries.
//
// [0]: https://facebook.github.io/graphql/June2018/#sec-Type-Name-Introspection
// [1]: https://facebook.github.io/graphql/June2018/#sec-Schema-Introspection

var (
	schemaMetaFieldName = "__schema"
	schemaMetaFieldType = graphql.MustNewNonNullOfType(graphql.IntrospectionTypes.Schema())
	typeMetaFieldName   = "__type"
	typeMetaFieldArgs   = []graphql.Argument{
		// FIXME: Should not use graphql.MockArgument.
		graphql.MockArgument("name", "", graphql.MustNewNonNullOfType(graphql.String()), nil),
	}
	typeMetaFieldType     = graphql.IntrospectionTypes.Type()
	typenameMetaFieldName = "__typename"
	typenameMetaFieldType = graphql.MustNewNonNullOfType(graphql.String())
)

//===----------------------------------------------------------------------------------------====//
// __schema
//===----------------------------------------------------------------------------------------====//

// schemaMetaField implemens __schema meta-field to access schema introspection system [0]. The
// field looks like,
//
//	__schema: __Schema!
//
// [0]: https://facebook.github.io/graphql/June2018/#sec-Schema-Introspection
type schemaMetaField struct{}

// Name implements graphql.Field.
func (schemaMetaField) Name() string {
	return schemaMetaFieldName
}

// Description implements graphql.Field.
func (schemaMetaField) Description() string {
	return "Access the current type schema of this server."
}

// Type implements graphql.Field.
func (schemaMetaField) Type() graphql.Type {
	return schemaMetaFieldType
}

// Args implements graphql.Field.
func (schemaMetaField) Args() []graphql.Argument {
	return nil
}

type schemaMetaFieldResolver struct{}

func (schemaMetaFieldResolver) Resolve(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
	return info.Schema(), nil
}

// Resolver implements graphql.Field.
func (schemaMetaField) Resolver() graphql.FieldResolver {
	return schemaMetaFieldResolver{}
}

// Deprecation is non-nil when the field is tagged as deprecated.
func (schemaMetaField) Deprecation() *graphql.Deprecation {
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
// [0]: https://facebook.github.io/graphql/June2018/#sec-Schema-Introspection
type typeMetaField struct{}

// Name implements graphql.Field.
func (typeMetaField) Name() string {
	return typeMetaFieldName
}

// Description implements graphql.Field.
func (typeMetaField) Description() string {
	return "Request the type information of a single type."
}

// Type implements graphql.Field.
func (typeMetaField) Type() graphql.Type {
	return typeMetaFieldType
}

// Args implements graphql.Field.
func (typeMetaField) Args() []graphql.Argument {
	return typeMetaFieldArgs
}

type typeMetaFieldResolver struct{}

func (typeMetaFieldResolver) Resolve(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
	return info.Schema().TypeMap().Lookup(info.Args().Get("name").(string)), nil
}

// Resolver implements graphql.Field.
func (typeMetaField) Resolver() graphql.FieldResolver {
	return typeMetaFieldResolver{}
}

// Deprecation is non-nil when the field is tagged as deprecated.
func (typeMetaField) Deprecation() *graphql.Deprecation {
	return nil
}

//===----------------------------------------------------------------------------------------====//
// __typename
//===----------------------------------------------------------------------------------------====//

// typenameMetaField implemens __typename meta-field [0] to access the name of the object type being
// queried.
//
// [0]: https://facebook.github.io/graphql/June2018/#sec-Type-Name-Introspection
type typenameMetaField struct{}

// Name implements graphql.Field.
func (typenameMetaField) Name() string {
	return typenameMetaFieldName
}

// Description implements graphql.Field.
func (typenameMetaField) Description() string {
	return "The name of the current Object type at runtime."
}

// Type implements graphql.Field.
func (typenameMetaField) Type() graphql.Type {
	return typenameMetaFieldType
}

// Args implements graphql.Field.
func (typenameMetaField) Args() []graphql.Argument {
	return nil
}

type typenameMetaFieldResolver struct{}

func (typenameMetaFieldResolver) Resolve(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
	return info.Object().Name(), nil
}

// Resolver implements graphql.Field.
func (typenameMetaField) Resolver() graphql.FieldResolver {
	return typenameMetaFieldResolver{}
}

// Deprecation is non-nil when the field is tagged as deprecated.
func (typenameMetaField) Deprecation() *graphql.Deprecation {
	return nil
}
