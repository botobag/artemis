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

// UnionConfig provides specification to define a Union type. It is served as a convenient way to
// create a UnionTypeDefinition for creating a union type.
type UnionConfig struct {
	ThisIsTypeDefinition

	// Name of the defining Union
	Name string

	// Description for the Union type
	Description string

	// PossibleTypes describes which Object types can be represented by the defining union.
	PossibleTypes []ObjectTypeDefinition

	// TypeResolver resolves the concrete Object type implementing the defining interface from given
	// value.
	TypeResolver TypeResolver
}

var (
	_ TypeDefinition      = (*UnionConfig)(nil)
	_ UnionTypeDefinition = (*UnionConfig)(nil)
)

// TypeData implements UnionTypeDefinition.
func (config *UnionConfig) TypeData() UnionTypeData {
	return UnionTypeData{
		Name:          config.Name,
		Description:   config.Description,
		PossibleTypes: config.PossibleTypes,
	}
}

// NewTypeResolver implments UnionTypeDefinition.
func (config *UnionConfig) NewTypeResolver(union Union) (TypeResolver, error) {
	return config.TypeResolver, nil
}

// unionTypeCreator is given to newTypeImpl for creating a Union.
type unionTypeCreator struct {
	typeDef UnionTypeDefinition
}

// unionTypeCreator implements typeCreator.
var _ typeCreator = (*unionTypeCreator)(nil)

// TypeDefinition implements typeCreator.
func (creator *unionTypeCreator) TypeDefinition() TypeDefinition {
	return creator.typeDef
}

// LoadDataAndNew implements typeCreator.
func (creator *unionTypeCreator) LoadDataAndNew() (Type, error) {
	typeDef := creator.typeDef
	// Load data.
	data := typeDef.TypeData()

	if len(data.Name) == 0 {
		return nil, NewError("Must provide name for Union.")
	}

	return &union{
		data: data,
	}, nil
}

// Finalize implements typeCreator.
func (creator *unionTypeCreator) Finalize(t Type, typeDefResolver typeDefinitionResolver) error {
	union := t.(*union)

	// Initialize type resolver for the Interface type.
	typeResolver, err := creator.typeDef.NewTypeResolver(union)
	if err != nil {
		return err
	}
	union.typeResolver = typeResolver

	// Resolve possible object types.
	possibleTypes := NewPossibleTypeSet()
	for _, possibleTypeDef := range union.data.PossibleTypes {
		possibleType, err := typeDefResolver(possibleTypeDef)
		if err != nil {
			return err
		}
		possibleTypes.Add(possibleType.(Object))
	}
	union.possibleTypes = possibleTypes

	return nil
}

// union is our built-in implementation for Union. It is configured with and built from
// UnionTypeDefinition.
type union struct {
	ThisIsUnionType
	data          UnionTypeData
	possibleTypes PossibleTypeSet
	typeResolver  TypeResolver
}

var _ Union = (*union)(nil)

// NewUnion initializes an instance of "union".
func NewUnion(typeDef UnionTypeDefinition) (Union, error) {
	t, err := newTypeImpl(&unionTypeCreator{
		typeDef: typeDef,
	})
	if err != nil {
		return nil, err
	}
	return t.(Union), nil
}

// MustNewUnion is a convenience function equivalent to NewUnion but panics on failure instead of
// returning an error.
func MustNewUnion(typeDef UnionTypeDefinition) Union {
	u, err := NewUnion(typeDef)
	if err != nil {
		panic(err)
	}
	return u
}

// String implements fmt.Stringer.
func (u *union) String() string {
	return u.Name()
}

// TypeResolver implements AbstractType.
func (u *union) TypeResolver() TypeResolver {
	return u.typeResolver
}

// Name implements TypeWithName.
func (u *union) Name() string {
	return u.data.Name
}

// Description implements TypeWithDescription.
func (u *union) Description() string {
	return u.data.Description
}

// PossibleTypes implements Union.
func (u *union) PossibleTypes() PossibleTypeSet {
	return u.possibleTypes
}
