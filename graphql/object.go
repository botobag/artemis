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

// ObjectConfig provides specification to define a Object type. It is served as a convenient way to
// create a ObjectTypeDefinition for creating an object type.
type ObjectConfig struct {
	ThisIsObjectTypeDefinition

	// Name of the defining Object
	Name string

	// Description for the Object type
	Description string

	// Interfaces that implemented by the defining Object
	Interfaces []InterfaceTypeDefinition

	// Fields in the object
	Fields Fields
}

var (
	_ TypeDefinition       = (*ObjectConfig)(nil)
	_ ObjectTypeDefinition = (*ObjectConfig)(nil)
)

// TypeData implements ObjectTypeDefinition.
func (config *ObjectConfig) TypeData() ObjectTypeData {
	return ObjectTypeData{
		Name:        config.Name,
		Description: config.Description,
		Interfaces:  config.Interfaces,
		Fields:      config.Fields,
	}
}

// objectTypeCreator is given to newTypeImpl for creating a Object.
type objectTypeCreator struct {
	typeDef ObjectTypeDefinition
}

// objectTypeCreator implements typeCreator.
var _ typeCreator = (*objectTypeCreator)(nil)

// TypeDefinition implements typeCreator.
func (creator *objectTypeCreator) TypeDefinition() TypeDefinition {
	return creator.typeDef
}

// LoadDataAndNew implements typeCreator.
func (creator *objectTypeCreator) LoadDataAndNew() (Type, error) {
	typeDef := creator.typeDef
	// Load data.
	data := typeDef.TypeData()

	// Must provide a name.
	if len(data.Name) == 0 {
		return nil, NewError("Must provide name for Object.")
	}

	// Create instance.
	return &Object{
		data: data,
	}, nil
}

// Finalize implements typeCreator.
func (*objectTypeCreator) Finalize(t Type, typeDefResolver typeDefinitionResolver) error {
	object := t.(*Object)

	// Build field map.
	fieldMap, err := buildFieldMap(object.data.Fields, typeDefResolver)
	if err != nil {
		return err
	}
	object.fields = fieldMap

	// Resolve interface type.
	numInterfaces := len(object.data.Interfaces)
	if numInterfaces > 0 {
		interfaces := make([]*Interface, numInterfaces)
		for i, ifaceTypeDef := range object.data.Interfaces {
			iface, err := typeDefResolver(ifaceTypeDef)
			if err != nil {
				return err
			}
			interfaces[i] = iface.(*Interface)
		}
		object.interfaces = interfaces
	}

	return nil
}

// Object is an implementation of Object.
type Object struct {
	data       ObjectTypeData
	fields     FieldMap
	interfaces []*Interface
}

var (
	_ Type                = (*Object)(nil)
	_ TypeWithName        = (*Object)(nil)
	_ TypeWithDescription = (*Object)(nil)
)

// NewObject defines a Object type from a ObjectTypeDefinition.
func NewObject(typeDef ObjectTypeDefinition) (*Object, error) {
	t, err := newTypeImpl(&objectTypeCreator{
		typeDef: typeDef,
	})
	if err != nil {
		return nil, err
	}
	return t.(*Object), nil
}

// MustNewObject is a convenience function equivalent to NewObject but panics on failure instead of
// returning an error.
func MustNewObject(typeDef ObjectTypeDefinition) *Object {
	o, err := NewObject(typeDef)
	if err != nil {
		panic(err)
	}
	return o
}

// graphqlType implements Type.
func (*Object) graphqlType() {}

// Name implemennts TypeWithName.
func (o *Object) Name() string {
	return o.data.Name
}

// Description implemennts TypeWithDescription.
func (o *Object) Description() string {
	return o.data.Description
}

// Values implemennts Type.
func (o *Object) String() string {
	return o.Name()
}

// Fields defined in the object
func (o *Object) Fields() FieldMap {
	return o.fields
}

// Interfaces includes interfaces that implemented by the object.
func (o *Object) Interfaces() []*Interface {
	return o.interfaces
}
