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

// InputFields maps field name to its definition for defining an InputField. It should be
// "InputFieldDefinitionMap" but is shorten to save some typing efforts.
type InputFields map[string]InputFieldDefinition

// An intentionally internal type for marking a "null" as default value for an argument
type inputFieldNilValueType int

// NilInputFieldDefaultValue is a value that has a special meaning when it is given to the
// DefaultValue in InputFieldDefinition. It sets the argument with default value set to "null". This
// is not the same with setting DefaultValue to "nil" or not giving it a value which means there's
// no default value. We need this trick to distiguish from whether the input field has a default
// value "nil" or it doesn't have one at all. The constant has an internal type, therefore there's
// no way to create one outside the package.
const NilInputFieldDefaultValue inputFieldNilValueType = 0

// InputFieldDefinition contains definition for defining a field in an Input Object type.
type InputFieldDefinition struct {
	// Description of the field
	Description string

	// Type of value given to this field
	Type TypeDefinition

	// DefaultValue specified the value to be assigned to the field when no input is provided.
	DefaultValue interface{}
}

// buildInputFieldMap takes an InputFields to build an InputFieldMap.
func buildInputFieldMap(inputFieldDefMap InputFields, typeDefResolver typeDefinitionResolver) (InputFieldMap, error) {
	numFields := len(inputFieldDefMap)
	if numFields == 0 {
		return nil, nil
	}

	inputFieldMap := make(InputFieldMap, numFields)
	for name, inputFieldDef := range inputFieldDefMap {
		inputFieldType, err := typeDefResolver.Resolve(inputFieldDef.Type)
		if err != nil {
			return nil, err
		}

		inputFieldMap[name] = InputField{
			name:         name,
			description:  inputFieldDef.Description,
			ttype:        inputFieldType,
			defaultValue: inputFieldDef.DefaultValue,
		}
	}

	return inputFieldMap, nil
}

// InputFieldMap maps field name to the field definition in an Input Object type.
type InputFieldMap map[string]InputField

// InputField defines a field in an InputObject. It is much simpler than Field because it doesn't
// get value from resolver nor can it have arguments.
type InputField struct {
	name         string
	description  string
	ttype        Type
	defaultValue interface{}
}

// Name of the field
func (f *InputField) Name() string {
	return f.name
}

// Description of the field
func (f *InputField) Description() string {
	return f.description
}

// Type of value yielded by the field
func (f *InputField) Type() Type {
	return f.ttype
}

// HasDefaultValue returns true if the argument has a default value.
func (f *InputField) HasDefaultValue() bool {
	return f.defaultValue != nil
}

// DefaultValue specifies the value to be assigned to the argument when no value is provided.
func (f *InputField) DefaultValue() interface{} {
	// Deal with NilInputFieldDefaultValue specially.
	if _, ok := f.defaultValue.(inputFieldNilValueType); ok {
		// We have default value which is "null".
		return nil
	}
	return f.defaultValue
}

// InputObjectConfig provides specification to define a InputObject type. It is served as a
// convenient way to create a InputObjectTypeDefinition for creating an input object type.
type InputObjectConfig struct {
	ThisIsInputObjectTypeDefinition

	// Name of the defining InputObject
	Name string

	// Description for the InputObject type
	Description string

	// Fields to be defined in the InputObject Type
	Fields InputFields
}

var (
	_ TypeDefinition            = (*InputObjectConfig)(nil)
	_ InputObjectTypeDefinition = (*InputObjectConfig)(nil)
)

// TypeData implements InputObjectTypeDefinition.
func (config *InputObjectConfig) TypeData() InputObjectTypeData {
	return InputObjectTypeData{
		Name:        config.Name,
		Description: config.Description,
		Fields:      config.Fields,
	}
}

// inputObjectTypeCreator is given to newTypeImpl for creating a Object.
type inputObjectTypeCreator struct {
	typeDef InputObjectTypeDefinition
}

// inputObjectTypeCreator implements typeCreator.
var _ typeCreator = (*inputObjectTypeCreator)(nil)

// TypeDefinition implements typeCreator.
func (creator *inputObjectTypeCreator) TypeDefinition() TypeDefinition {
	return creator.typeDef
}

// LoadDataAndNew implements typeCreator.
func (creator *inputObjectTypeCreator) LoadDataAndNew() (Type, error) {
	typeDef := creator.typeDef
	// Load data.
	data := typeDef.TypeData()

	// Must provide a name.
	if len(data.Name) == 0 {
		return nil, NewError("Must provide name for InputObject.")
	}

	// Create instance.
	return &InputObject{
		data: data,
	}, nil
}

// Finalize implements typeCreator.
func (*inputObjectTypeCreator) Finalize(t Type, typeDefResolver typeDefinitionResolver) error {
	object := t.(*InputObject)

	// Build field map.
	fieldMap, err := buildInputFieldMap(object.data.Fields, typeDefResolver)
	if err != nil {
		return err
	}
	object.fields = fieldMap

	return nil
}

// InputObject Type Definition
//
// An input object defines a structured collection of fields which may be supplied to a field
// argument. It is essentially an Object type but with some contraints on the fields so it can be
// used as an input argument. More specifically, fields in an Input Object type cannot define
// arguments or contain references to interfaces and unions.
//
// Ref: https://facebook.github.io/graphql/June2018/#sec-Input-Objects
type InputObject struct {
	data   InputObjectTypeData
	fields InputFieldMap
}

var (
	_ Type                = (*InputObject)(nil)
	_ TypeWithName        = (*InputObject)(nil)
	_ TypeWithDescription = (*InputObject)(nil)
)

// NewInputObject defines a InputObject type from a InputObjectTypeDefinition.
func NewInputObject(typeDef InputObjectTypeDefinition) (*InputObject, error) {
	t, err := newTypeImpl(&inputObjectTypeCreator{
		typeDef: typeDef,
	})
	if err != nil {
		return nil, err
	}
	return t.(*InputObject), nil
}

// MustNewInputObject is a convenience function equivalent to NewInputObject but panics on failure
// instead of returning an error.
func MustNewInputObject(typeDef InputObjectTypeDefinition) *InputObject {
	o, err := NewInputObject(typeDef)
	if err != nil {
		panic(err)
	}
	return o
}

// graphqlType implements Type.
func (*InputObject) graphqlType() {}

// Name implemennts TypeWithName.
func (o *InputObject) Name() string {
	return o.data.Name
}

// Description implemennts TypeWithDescription.
func (o *InputObject) Description() string {
	return o.data.Description
}

// String implemennts Type.
func (o *InputObject) String() string {
	return o.Name()
}

// Fields defined in the object
func (o *InputObject) Fields() InputFieldMap {
	return o.fields
}
