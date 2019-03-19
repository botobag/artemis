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

// BuildInputFieldMap builds an InputFieldMap from given InputFields.
func BuildInputFieldMap(inputFieldDefMap InputFields, typeDefResolver typeDefinitionResolver) (InputFieldMap, error) {
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

		inputFieldMap[name] = &inputField{
			name:         name,
			description:  inputFieldDef.Description,
			ttype:        inputFieldType,
			defaultValue: inputFieldDef.DefaultValue,
		}
	}

	return inputFieldMap, nil
}

// inputField is our built-in implementation for InputField.
type inputField struct {
	name         string
	description  string
	ttype        Type
	defaultValue interface{}
}

var _ InputField = (*inputField)(nil)

// Name implements InputField.
func (f *inputField) Name() string {
	return f.name
}

// Description implements InputField.
func (f *inputField) Description() string {
	return f.description
}

// Type implements InputField.
func (f *inputField) Type() Type {
	return f.ttype
}

// HasDefaultValue implements InputField.
func (f *inputField) HasDefaultValue() bool {
	return f.defaultValue != nil
}

// DefaultValue implements InputField.
func (f *inputField) DefaultValue() interface{} {
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
	ThisIsTypeDefinition

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
	return &inputObject{
		data: data,
	}, nil
}

// Finalize implements typeCreator.
func (*inputObjectTypeCreator) Finalize(t Type, typeDefResolver typeDefinitionResolver) error {
	object := t.(*inputObject)

	// Build field map.
	fieldMap, err := BuildInputFieldMap(object.data.Fields, typeDefResolver)
	if err != nil {
		return err
	}
	object.fields = fieldMap

	return nil
}

// inputObject is our built-in implementation for InputObject. It is configured with and built from
// InputObjectTypeDefinition.
type inputObject struct {
	ThisIsInputObjectType
	data   InputObjectTypeData
	fields InputFieldMap
}

var _ InputObject = (*inputObject)(nil)

// NewInputObject defines a InputObject type from a InputObjectTypeDefinition.
func NewInputObject(typeDef InputObjectTypeDefinition) (InputObject, error) {
	t, err := newTypeImpl(&inputObjectTypeCreator{
		typeDef: typeDef,
	})
	if err != nil {
		return nil, err
	}
	return t.(InputObject), nil
}

// MustNewInputObject is a convenience function equivalent to NewInputObject but panics on failure
// instead of returning an error.
func MustNewInputObject(typeDef InputObjectTypeDefinition) InputObject {
	o, err := NewInputObject(typeDef)
	if err != nil {
		panic(err)
	}
	return o
}

// Name implemennts TypeWithName.
func (o *inputObject) Name() string {
	return o.data.Name
}

// Description implemennts TypeWithDescription.
func (o *inputObject) Description() string {
	return o.data.Description
}

// Fields implements InputObject.
func (o *inputObject) Fields() InputFieldMap {
	return o.fields
}
