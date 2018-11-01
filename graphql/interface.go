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

// InterfaceConfig provides specification to define a Interface type. It is served as a convenient way to
// create a InterfaceTypeDefinition for creating an interface type.
type InterfaceConfig struct {
	ThisIsInterfaceTypeDefinition

	// Name of the defining Interface
	Name string

	// Description for the Interface type
	Description string

	// TypeResolver resolves the concrete Object type implementing the defining interface from given
	// value.
	TypeResolver TypeResolver

	// Fields in the Interface Type
	Fields Fields
}

var (
	_ TypeDefinition          = (*InterfaceConfig)(nil)
	_ InterfaceTypeDefinition = (*InterfaceConfig)(nil)
)

// TypeData implements InterfaceTypeDefinition.
func (config *InterfaceConfig) TypeData() InterfaceTypeData {
	return InterfaceTypeData{
		Name:        config.Name,
		Description: config.Description,
		Fields:      config.Fields,
	}
}

// NewTypeResolver implments InterfaceTypeDefinition.
func (config *InterfaceConfig) NewTypeResolver(iface *Interface) (TypeResolver, error) {
	return config.TypeResolver, nil
}

// interfaceTypeCreator is given to newTypeImpl for creating a Interface.
type interfaceTypeCreator struct {
	typeDef InterfaceTypeDefinition
}

// interfaceTypeCreator implements typeCreator.
var _ typeCreator = (*interfaceTypeCreator)(nil)

// TypeDefinition implements typeCreator.
func (creator *interfaceTypeCreator) TypeDefinition() TypeDefinition {
	return creator.typeDef
}

// LoadDataAndNew implements typeCreator.
func (creator *interfaceTypeCreator) LoadDataAndNew() (Type, error) {
	typeDef := creator.typeDef
	// Load data.
	data := typeDef.TypeData()

	// Must provide a name.
	if len(data.Name) == 0 {
		return nil, NewError("Must provide name for Interface.")
	}

	// Create instance.
	return &Interface{
		data: data,
	}, nil
}

// Finalize implements typeCreator.
func (creator *interfaceTypeCreator) Finalize(t Type, typeDefResolver typeDefinitionResolver) error {
	iface := t.(*Interface)

	// Initialize type resolver for the Interface type.
	typeResolver, err := creator.typeDef.NewTypeResolver(iface)
	if err != nil {
		return err
	}
	iface.typeResolver = typeResolver

	// Build field map.
	fieldMap, err := buildFieldMap(iface.data.Fields, typeDefResolver)
	if err != nil {
		return err
	}
	iface.fields = fieldMap

	return nil
}

// Interface Type Definition
//
// When a field can return one of a heterogeneous set of types, a Interface type is used to describe
// what types are possible, what fields are in common across all types, as well as a function to
// determine which type is actually used when the field is resolved.
//
// Reference: https://facebook.github.io/graphql/June2018/#sec-Interfaces
type Interface struct {
	data         InterfaceTypeData
	typeResolver TypeResolver
	fields       FieldMap
}

var (
	_ Type                = (*Interface)(nil)
	_ AbstractType        = (*Interface)(nil)
	_ TypeWithName        = (*Interface)(nil)
	_ TypeWithDescription = (*Interface)(nil)
)

// NewInterface initializes an instance of "Interface".
func NewInterface(typeDef InterfaceTypeDefinition) (*Interface, error) {
	t, err := newTypeImpl(&interfaceTypeCreator{
		typeDef: typeDef,
	})
	if err != nil {
		return nil, err
	}
	return t.(*Interface), nil
}

// MustNewInterface is a convenience function equivalent to NewInterface but panics on failure instead of
// returning an error.
func MustNewInterface(typeDef InterfaceTypeDefinition) *Interface {
	iface, err := NewInterface(typeDef)
	if err != nil {
		panic(err)
	}
	return iface
}

// graphqlType implements Type.
func (*Interface) graphqlType() {}

// graphqlAbstractType implements AbstractType.
func (*Interface) graphqlAbstractType() {}

// TypeResolver implements AbstractType.
func (iface *Interface) TypeResolver() TypeResolver {
	return iface.typeResolver
}

// Name implements TypeWithName.
func (iface *Interface) Name() string {
	return iface.data.Name
}

// Description implements TypeWithDescription.
func (iface *Interface) Description() string {
	return iface.data.Description
}

// String implements Type.
func (iface *Interface) String() string {
	return iface.Name()
}

// Fields returns set of fields that needs to be provided when implementing this interface.
func (iface *Interface) Fields() FieldMap {
	return iface.fields
}
