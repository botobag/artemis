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

package ast

import (
	"math"
	"strconv"

	"github.com/botobag/artemis/graphql/token"
)

// Node represents a node in an AST tree from parsing GraphQL language.
type Node interface {
	// TokenRange indicates the region of the Node in the source.
	TokenRange() token.Range
}

// Name represents a name.
//
// Reference: https://facebook.github.io/graphql/June2018/#sec-Names
type Name struct {
	// Token is the lexical token that contains the name (usually scanned by lexer) and also
	// indicates the location in the source; Its kind must be an token.KindName.
	Token *token.Token
}

var _ Node = Name{}

// Value returns the name in string.
func (node Name) Value() string {
	return node.Token.Value
}

// TokenRange implements Node.
func (node Name) TokenRange() token.Range {
	return token.Range{
		First: node.Token,
		Last:  node.Token,
	}
}

//===----------------------------------------------------------------------------------------====//
// 2.2 Document
//===----------------------------------------------------------------------------------------====//
// A GraphQL Document describes a complete file or request string operated on by a GraphQL service
// or client. A document contains multiple definitions, either executable or representative of a
// GraphQL type system.
//
// Reference: https://facebook.github.io/graphql/June2018/#sec-Language.Document

// Document represents a GraphQL Document.
//
// Reference: https://facebook.github.io/graphql/June2018/#Document
type Document struct {
	// Definitions defined in the document.
	Definitions []Definition
}

var _ Node = Document{}

// TokenRange implements Node.
func (node Document) TokenRange() token.Range {
	if len(node.Definitions) == 0 {
		return token.Range{
			First: nil,
			Last:  nil,
		}
	}
	// Note that the first token of a valid Document is always SOF and the last token is EOF for now.
	// The location of SOF is NoSourceLocation which should use with caution (e.g., it may cause crash
	// in source.PosFromLocation). (And the location of EOF is the document size.)
	return token.Range{
		First: node.Definitions[0].TokenRange().First.Prev,
		Last:  node.Definitions[len(node.Definitions)-1].TokenRange().Last.Next,
	}
}

// Definition represents a GraphQL Definition.
//
// Reference: https://facebook.github.io/graphql/June2018/#Definition
type Definition interface {
	Node

	// Directives applied to the definition to provide alternate runtime and validation behaviors.
	// (Prepend "Get" to avoid name collision with the fields in derived class.)
	GetDirectives() Directives

	// definitionNode is a special mark to indicate a Definition node. It makes sure that only
	// definition node can be assigned to Definition.
	definitionNode()
}

// DefinitionBase is a common base that is embedded in Definition implementation.
type DefinitionBase struct {
	// Directives that are applied to the definition
	Directives Directives
}

// GetDirectives provides implementation for Definition.GetDefinition.
func (base DefinitionBase) GetDirectives() Directives {
	return base.Directives
}

// definitionNode marks the embedding node as a Definition.
func (DefinitionBase) definitionNode() {}

// ExecutableDefinition represents an executable definition.
//
// Reference: https://facebook.github.io/graphql/June2018/#ExecutableDefinition
type ExecutableDefinition interface {
	Definition

	// GetSelectionSet specifies the sets of fields to fetch. (Prepend "Get" to avoid name collision
	// with the fields in derived class.)
	GetSelectionSet() SelectionSet
}

var (
	_ ExecutableDefinition = (*OperationDefinition)(nil)
	_ ExecutableDefinition = (*FragmentDefinition)(nil)
)

//===----------------------------------------------------------------------------------------====//
// 2.3 Operations
//===----------------------------------------------------------------------------------------====//
// There are three types of operations that GraphQL models:
//
//	* query – a read‐only fetch.
//	* mutation – a write followed by a fetch.
// 	* subscription – a long‐lived request that fetches data in response to source events.
//
// Reference: https://facebook.github.io/graphql/June2018/#sec-Language.Operations

// OperationType specifies the type of operation model.
//
// Reference: https://facebook.github.io/graphql/June2018/#OperationType
type OperationType string

// Enumeration of OperationType
const (
	OperationTypeQuery        OperationType = "query"
	OperationTypeMutation                   = "mutation"
	OperationTypeSubscription               = "subscription"
)

// OperationDefinition represents a GraphQL operation.
//
// Reference: https://facebook.github.io/graphql/June2018/#OperationDefinition
type OperationDefinition struct {
	DefinitionBase

	// Type is a Name token that contains operation type.
	Type *token.Token

	// Name of the operation
	Name Name

	// VariableDefinitions contains variables given to the operation
	VariableDefinitions []*VariableDefinition

	// SelectionSet specifies the sets of fields to fetch.
	SelectionSet SelectionSet
}

var _ Node = (*OperationDefinition)(nil)

// TokenRange implements Node.
func (definition *OperationDefinition) TokenRange() token.Range {
	if definition.IsQueryShorthand() {
		return definition.SelectionSet.TokenRange()
	}

	return token.Range{
		First: definition.Type,
		Last:  definition.SelectionSet.LastToken(),
	}
}

// GetSelectionSet implements ExecutableDefinition.
func (definition *OperationDefinition) GetSelectionSet() SelectionSet {
	return definition.SelectionSet
}

// IsQueryShorthand returns true if this is a short form of query operation such as "{ field }"
// (this is a valid GraphQL Document). Query shorthand doesn't specify operation type. It is
// implicit a query.
func (definition *OperationDefinition) IsQueryShorthand() bool {
	return definition.Type == nil
}

// OperationType returns the type of operation.
func (definition *OperationDefinition) OperationType() OperationType {
	if definition.IsQueryShorthand() {
		return OperationTypeQuery
	}
	return OperationType(definition.Type.Value)
}

//===----------------------------------------------------------------------------------------====//
// 2.4 Selection Sets
//===----------------------------------------------------------------------------------------====//
// An operation selects the set of information it needs, and will receive exactly that information
// and nothing more, avoiding over‐fetching and under‐fetching data.
//
// Reference: https://facebook.github.io/graphql/June2018/#sec-Selection-Sets

// SelectionSet specifies the information to be fetched.
//
// Reference: https://facebook.github.io/graphql/June2018/#SelectionSet
type SelectionSet []Selection

var _ Node = SelectionSet{}

// FirstToken returns the first token in the sequence of selection set.
func (set SelectionSet) FirstToken() *token.Token {
	if len(set) == 0 {
		return nil
	}
	// Find left brace "{" token in prior to the first Selection.
	return set[0].TokenRange().First.Prev
}

// LastToken returns the last token in the sequence of selection set.
func (set SelectionSet) LastToken() *token.Token {
	if len(set) == 0 {
		return nil
	}
	// Find right brace "}" token after the last Selection.
	return set[len(set)-1].TokenRange().Last.Next
}

// TokenRange implements Node.
func (set SelectionSet) TokenRange() token.Range {
	return token.Range{
		First: set.FirstToken(),
		Last:  set.LastToken(),
	}
}

// Selection represents a field or a set of fields.
//
//	Selection ::
//		Field
//		FragmentSpread
//		InlineFragment
//
// Reference: https://facebook.github.io/graphql/June2018/#Selection
type Selection interface {
	Node

	// sleectionNode is a special mark to indicate a Selection node. It makes sure that only selection
	// node can be assigned to Selection.
	sleectionNode()
}

var (
	_ Selection = (*Field)(nil)
	_ Selection = (*FragmentSpread)(nil)
	_ Selection = (*InlineFragment)(nil)
)

//===----------------------------------------------------------------------------------------====//
// 2.5 Field
//===----------------------------------------------------------------------------------------====//
// A selection set is primarily composed of fields. A field describes one discrete piece of
// information available to request within a selection set.
//
// Reference: https://facebook.github.io/graphql/June2018/#sec-Language.Fields

// Field describes a field selection.
//
// Reference: https://facebook.github.io/graphql/June2018/#Field
type Field struct {
	// Alias specifies a different name of the key to be used in response object for returning the
	// field value.
	//
	// Reference: https://facebook.github.io/graphql/June2018/#sec-Field-Alias
	Alias Name

	// Name of the field
	Name Name

	// Arguments taken by the field
	Arguments Arguments

	// Directives applied to the field
	Directives Directives

	// Set of information to be fetched that is nested in the field.
	SelectionSet SelectionSet
}

// TokenRange implements Node.
func (node *Field) TokenRange() token.Range {
	var r token.Range

	if node.Alias.Token != nil {
		r.First = node.Alias.Token
	} else {
		r.First = node.Name.Token
	}

	if len(node.SelectionSet) > 0 {
		r.Last = node.SelectionSet.LastToken()
	} else if len(node.Directives) > 0 {
		r.Last = node.Directives.LastToken()
	} else if len(node.Arguments) > 0 {
		r.Last = node.Arguments.LastToken()
	} else {
		r.Last = node.Name.Token
	}

	return r
}

// sleectionNode implements Selection.
func (*Field) sleectionNode() {}

//===----------------------------------------------------------------------------------------====//
// 2.6 Argument
//===----------------------------------------------------------------------------------------====//
// Fields are conceptually functions which return values, and occasionally accept arguments which
// alter their behavior.
//
// Reference: https://facebook.github.io/graphql/June2018/#sec-Language.Arguments

// Arguments specifies a list of Arguments
type Arguments []*Argument

// FirstToken returns the first token in the sequence of argument.
func (nodes Arguments) FirstToken() *token.Token {
	if len(nodes) == 0 {
		return nil
	}
	// Find left paren "(" token.
	return nodes[0].Name.Token.Prev
}

// LastToken returns the last token in the sequence of argument.
func (nodes Arguments) LastToken() *token.Token {
	if len(nodes) == 0 {
		return nil
	}
	// Find right paren ")" token which is next to the token of the last value.
	return nodes[len(nodes)-1].Value.TokenRange().Last.Next
}

// An Argument is an argument taken by a field.
//
// Reference: https://facebook.github.io/graphql/June2018/#Argument
type Argument struct {
	// Name of the argument
	Name Name

	// Value given to the argument
	Value Value
}

var _ Node = (*Argument)(nil)

// TokenRange implements Node.
func (node *Argument) TokenRange() token.Range {
	return token.Range{
		First: node.Name.Token,
		Last:  node.Value.TokenRange().Last,
	}
}

//===----------------------------------------------------------------------------------------====//
// 2.8 Fragments
//===----------------------------------------------------------------------------------------====//
// Fragments allow for the reuse of common repeated selections of fields, reducing duplicated text
// in the document.
//
// Reference: https://facebook.github.io/graphql/June2018/#sec-Language.Fragments

// FragmentDefinition represents a reusable selections of fields.
//
// Reference: https://facebook.github.io/graphql/June2018/#FragmentDefinition
type FragmentDefinition struct {
	DefinitionBase

	// Name of the fragment
	Name Name

	// VariableDefinitions contains variables given to the fragment; This is an experimental feature
	// and may be subject to change. See RFC in https://github.com/facebook/graphql/issues/204.
	VariableDefinitions []*VariableDefinition

	// TypeCondition specifies the type this fragment applies to.
	TypeCondition NamedType

	// SelectionSet describes set of fields to be requested by the fragment
	SelectionSet SelectionSet
}

// TokenRange implements Node.
func (definition *FragmentDefinition) TokenRange() token.Range {
	return token.Range{
		First: definition.Name.Token.Prev, // "fragment" keyword
		Last:  definition.SelectionSet.LastToken(),
	}
}

// GetSelectionSet implements ExecutableDefinition.
func (definition *FragmentDefinition) GetSelectionSet() SelectionSet {
	return definition.SelectionSet
}

// FragmentSpread uses the spread operator (...) on a fragment to adds a set of fields defined by
// the fragment to selection set.
//
// Reference: https://facebook.github.io/graphql/June2018/#FragmentSpread
type FragmentSpread struct {
	// Name of the fragment to be consumed by the selection set
	Name Name

	// Directives applied to the fragment
	Directives Directives
}

// TokenRange implements Node.
func (node *FragmentSpread) TokenRange() token.Range {
	var lastToken *token.Token
	if len(node.Directives) > 0 {
		lastToken = node.Directives.LastToken()
	} else {
		lastToken = node.Name.Token
	}

	return token.Range{
		First: node.Name.Token.Prev, // "..." token
		Last:  lastToken,
	}
}

// sleectionNode implements Selection.
func (*FragmentSpread) sleectionNode() {}

// InlineFragment defines a fragment inline within a selection set.
//
// ReF: https://facebook.github.io/graphql/June2018/#sec-Inline-Fragments
type InlineFragment struct {
	// TypeCondition specifies the type this inline fragment applies to.
	TypeCondition NamedType

	// Directives applied to the inline fragment
	Directives Directives

	// SelectionSet describes the set of fields to be added into current selection set
	SelectionSet SelectionSet
}

// TokenRange implements Node.
func (node *InlineFragment) TokenRange() token.Range {
	var firstToken *token.Token
	if node.HasTypeCondition() {
		firstToken = node.TypeCondition.Name.Token
	} else if len(node.Directives) > 0 {
		firstToken = node.Directives.FirstToken()
	} else {
		firstToken = node.SelectionSet.FirstToken()
	}
	return token.Range{
		First: firstToken,
		Last:  node.SelectionSet.LastToken(),
	}
}

// HasTypeCondition returns true if the inline fragment specifies a type condition.
func (node *InlineFragment) HasTypeCondition() bool {
	// Check the existence of name token.
	return node.TypeCondition.Name.Token == nil
}

// sleectionNode implements Selection.
func (*InlineFragment) sleectionNode() {}

//===----------------------------------------------------------------------------------------====//
// 2.9 Input Values
//===----------------------------------------------------------------------------------------====//
// Field and directive arguments accept input values of various literal primitives; input values can
// be scalars, enumeration values, lists, or input objects.
//
// Reference: https://facebook.github.io/graphql/June2018/#sec-Input-Values

// Value represents a node containing a value.
//
// Reference: https://facebook.github.io/graphql/June2018/#Value
type Value interface {
	Node

	// Interface returns the value as an interface{}.
	Interface() interface{}

	// valueNode is a special mark to indicate a Type node. It makes sure that only value node can be
	// assigned to Value.
	valueNode()
}

// The following implement Value interface.
var (
	_ Value = Variable{}
	_ Value = IntValue{}
	_ Value = FloatValue{}
	_ Value = StringValue{}
	_ Value = BooleanValue{}
	_ Value = NullValue{}
	_ Value = EnumValue{}
	_ Value = ListValue{}
	_ Value = ObjectValue{}
)

// IntValue represents a value node containing an integer.
//
// Reference: https://facebook.github.io/graphql/June2018/#IntValue
type IntValue struct {
	// Token is the lexical token that contains the value (usually scanned by lexer) and also
	// indicates the location in the source; Its kind must be an token.KindInt.
	Token *token.Token
}

// TokenRange implements Node.
func (value IntValue) TokenRange() token.Range {
	return token.Range{
		First: value.Token,
		Last:  value.Token,
	}
}

// Interface implements Value.
func (value IntValue) Interface() interface{} {
	v, err := value.Int32Value()
	if err == nil {
		return v
	}
	return int32(0)
}

// valueNode implements Value.
func (IntValue) valueNode() {}

// String return the literal in string that specifies the integer value.
func (value IntValue) String() string {
	return value.Token.Value
}

// Uint32Value parses literal into an uint32.
func (value IntValue) Uint32Value() (uint32, error) {
	v, err := strconv.ParseUint(value.String(), 10, 32)
	return uint32(v), err
}

// Int32Value parses literal into an int32.
func (value IntValue) Int32Value() (int32, error) {
	v, err := strconv.ParseInt(value.String(), 10, 32)
	return int32(v), err
}

// Uint64Value parses literal into an uint64.
func (value IntValue) Uint64Value() (uint64, error) {
	return strconv.ParseUint(value.String(), 10, 64)
}

// Int64Value parses literal into an int64.
func (value IntValue) Int64Value() (int64, error) {
	return strconv.ParseInt(value.String(), 10, 64)
}

// FloatValue represents a value node containing a float.
//
// Reference: https://facebook.github.io/graphql/June2018/#FloatValue
type FloatValue struct {
	// Token is the lexical token that contains the value (usually scanned by lexer) and also
	// indicates the location in the source; Its kind must be an token.KindFloat.
	Token *token.Token
}

// TokenRange implements Node.
func (value FloatValue) TokenRange() token.Range {
	return token.Range{
		First: value.Token,
		Last:  value.Token,
	}
}

// Interface implements Value.
func (value FloatValue) Interface() interface{} {
	v, err := value.FloatValue()
	if err != nil {
		return math.NaN()
	}
	return v
}

// valueNode implements Value.
func (FloatValue) valueNode() {}

// Value return the literal in string that specifies the flaot value.
func (value FloatValue) String() string {
	return value.Token.Value
}

// FloatValue parses literal into a float64.
func (value FloatValue) FloatValue() (float64, error) {
	return strconv.ParseFloat(value.String(), 64)
}

// StringValue represents a value node containing a string.
//
// Reference: https://facebook.github.io/graphql/June2018/#StringValue
type StringValue struct {
	// Token is the lexical token that contains the value (usually scanned by lexer) and also
	// indicates the location in the source; Its kind must be an token.KindString or
	// token.KindBlockString.
	Token *token.Token
}

// TokenRange implements Node.
func (value StringValue) TokenRange() token.Range {
	return token.Range{
		First: value.Token,
		Last:  value.Token,
	}
}

// Interface implements Value.
func (value StringValue) Interface() interface{} {
	return value.Value()
}

// valueNode implements Value.
func (StringValue) valueNode() {}

// Value returns the string value.
func (value StringValue) Value() string {
	return value.Token.Value
}

// BooleanValue represents a value node containing a boolean.
//
// Reference: https://facebook.github.io/graphql/June2018/#BooleanValue
type BooleanValue struct {
	// Token is the lexical token that contains the value (usually scanned by lexer) and also
	// indicates the location in the source; It should be a token.KindName containing either "true" or
	// "false" (in strings) value.
	Token *token.Token
}

// TokenRange implements Node.
func (value BooleanValue) TokenRange() token.Range {
	return token.Range{
		First: value.Token,
		Last:  value.Token,
	}
}

// Interface implements Value.
func (value BooleanValue) Interface() interface{} {
	return value.Value()
}

// Value returns true if the token contains "true".
func (value BooleanValue) Value() bool {
	return value.Token.Value[0] == 't'
}

// valueNode implements Value.
func (BooleanValue) valueNode() {}

// NullValue represents the keyword "null".
//
// Reference: https://facebook.github.io/graphql/June2018/#NullValue
type NullValue struct {
	// Token is the lexical token that contains the value (usually scanned by lexer) and also
	// indicates the location in the source; It should be an token.KindName containing a "null".
	Token *token.Token
}

// TokenRange implements Node.
func (value NullValue) TokenRange() token.Range {
	return token.Range{
		First: value.Token,
		Last:  value.Token,
	}
}

// Interface implements Value.
func (value NullValue) Interface() interface{} {
	return nil
}

// valueNode implements Value.
func (NullValue) valueNode() {}

// EnumValue represents a value node containing a boolean.
//
// Reference: https://facebook.github.io/graphql/June2018/#EnumValue
type EnumValue struct {
	// Token is the lexical token that contains the value (usually scanned by lexer) and also
	// indicates the location in the source; Its kind must be an token.KindName.
	Token *token.Token
}

// TokenRange implements Node.
func (value EnumValue) TokenRange() token.Range {
	return token.Range{
		First: value.Token,
		Last:  value.Token,
	}
}

// Interface implements Value.
func (value EnumValue) Interface() interface{} {
	return value.Value()
}

// valueNode implements Value.
func (EnumValue) valueNode() {}

// Value returns the enum value.
func (value EnumValue) Value() string {
	return value.Token.Value
}

// ListValue represents a value node containing list of values.
//
// Reference: https://facebook.github.io/graphql/June2018/#ListValue
type ListValue struct {
	// This field contains either []Value or a *token.Token.
	//
	// If the ListValue specifies an empty list, this is a *token.Token (which should be a left
	// bracket) that starts the ListValue. This is used to know the source location of for a ListValue
	// without any values.
	//
	// Otherwise it is a []Value.
	ValuesOrStartToken interface{}
}

// FirstToken returns the first token (should be a left bracket) that starts the ListValue.
func (value ListValue) FirstToken() *token.Token {
	if value.IsEmpty() {
		// ValuesOrStartToken contains the desired token.
		return value.ValuesOrStartToken.(*token.Token)
	}
	return value.Values()[0].TokenRange().First.Prev
}

// LastToken returns the last token (should be a right bracket) that ends the ListValue.
func (value ListValue) LastToken() *token.Token {
	if value.IsEmpty() {
		return value.ValuesOrStartToken.(*token.Token).Next
	}
	values := value.Values()
	return values[len(values)-1].TokenRange().Last
}

// TokenRange implements Node.
func (value ListValue) TokenRange() token.Range {
	return token.Range{
		First: value.FirstToken(),
		Last:  value.LastToken(),
	}
}

// Interface implements Value.
func (value ListValue) Interface() interface{} {
	// Return an array containing the values returning from calling Interface() on each item.
	values := value.Values()
	result := make([]interface{}, len(values))
	for i := range values {
		result[i] = values[i].Interface()
	}
	return result
}

// IsEmpty returns true if this list contains no any value (i.e., an empty list.)
func (value ListValue) IsEmpty() bool {
	_, ok := value.ValuesOrStartToken.([]Value)
	return !ok
}

// Values returns values in the list. Return nil if this is an empty list.
func (value ListValue) Values() []Value {
	if values, ok := value.ValuesOrStartToken.([]Value); ok {
		return values
	}
	return nil
}

// valueNode implements Value.
func (ListValue) valueNode() {}

// ObjectValue represents a value node containing list of values
//
// Reference: https://facebook.github.io/graphql/June2018/#ObjectValue
type ObjectValue struct {
	// This field contains either []*ObjectField or a *token.Token.
	//
	// If the ObjectValue specifies an empty list, this is a *token.Token (which should be a left
	// brace) that starts the ObjectValue. This is used to know the source location of for a
	// ObjectValue without any fields.
	//
	// Otherwise it is a []*ObjectField.
	FieldsOrStartToken interface{}
}

// FirstToken returns the first token (should be a left brace) that starts the ObjectValue.
func (value ObjectValue) FirstToken() *token.Token {
	if value.HasFields() {
		return value.Fields()[0].Name.Token.Prev
	}
	// FieldsOrStartToken contains the desired token.
	return value.FieldsOrStartToken.(*token.Token)
}

// LastToken returns the last token (should be a right brace) that ends the ObjectValue.
func (value ObjectValue) LastToken() *token.Token {
	if value.HasFields() {
		fields := value.Fields()
		return fields[len(fields)-1].Value.TokenRange().Last.Next
	}
	return value.FieldsOrStartToken.(*token.Token).Next
}

// TokenRange implements Node.
func (value ObjectValue) TokenRange() token.Range {
	return token.Range{
		First: value.FirstToken(),
		Last:  value.LastToken(),
	}
}

// Interface implements Value.
func (value ObjectValue) Interface() interface{} {
	// Return a map that maps field name to its assigned value.
	fields := value.Fields()
	values := make(map[string]interface{}, len(fields))
	for i := range fields {
		field := fields[i]
		values[field.Name.Value()] = field.Value.Interface()
	}
	return values
}

// HasFields returns true if this object contains no any fields (i.e., an empty object.)
func (value ObjectValue) HasFields() bool {
	_, ok := value.FieldsOrStartToken.([]*ObjectField)
	return ok
}

// Fields returns field values in the object. Return nil if this is an empty list.
func (value ObjectValue) Fields() []*ObjectField {
	if fields, ok := value.FieldsOrStartToken.([]*ObjectField); ok {
		return fields
	}
	return nil
}

// valueNode implements Value.
func (ObjectValue) valueNode() {}

// ObjectField represent a node that assigns a value to an object field.
//
// https://facebook.github.io/graphql/June2018/#ObjectField
type ObjectField struct {
	// Name of the field being assigned
	Name Name

	// Value that is assigned to the field
	Value Value
}

//===----------------------------------------------------------------------------------------====//
// 2.10 Variables
//===----------------------------------------------------------------------------------------====//
// A GraphQL query can be parameterized with variables, maximizing query reuse, and avoiding costly
// string building in clients at runtime.
//
// Reference: https://facebook.github.io/graphql/June2018/#sec-Language.Variables

// Variable refers to a variable with a name.
//
// Reference: https://facebook.github.io/graphql/June2018/#Variable
type Variable struct {
	// Name of the reference
	Name Name
}

// FirstToken returns the first token at which the variable node starts.
func (value Variable) FirstToken() *token.Token {
	// Variable starts with $ token in prior to its name.
	return value.Name.Token.Prev
}

// TokenRange implements Node.
func (value Variable) TokenRange() token.Range {
	return token.Range{
		First: value.FirstToken(), // $
		Last:  value.Name.Token,
	}
}

// Interface implements Value.
func (value Variable) Interface() interface{} {
	// Return the name of variable.
	return value.Name.Value()
}

// valueNode implements Value.
func (Variable) valueNode() {}

// VariableDefinition defines a variable.
//
// Reference: https://facebook.github.io/graphql/June2018/#VariableDefinition
type VariableDefinition struct {
	// Variable that is defined by this node
	Variable Variable

	// Type of the variable value
	Type Type

	// DefaultValue describes the value to be used when no input value is supplied to the variable.
	DefaultValue Value

	// Directives applied to to the variable
	Directives Directives
}

// TokenRange implements Node.
func (value *VariableDefinition) TokenRange() token.Range {
	// VariableDefinition starts with the Variable node and ends with type or default value or the
	// last directive.
	var lastToken *token.Token
	if len(value.Directives) > 0 {
		lastToken = value.Directives.LastToken()
	} else if value.DefaultValue != nil {
		lastToken = value.DefaultValue.TokenRange().Last
	} else {
		lastToken = value.Type.TokenRange().Last
	}

	return token.Range{
		First: value.Variable.FirstToken(),
		Last:  lastToken,
	}
}

//===----------------------------------------------------------------------------------------====//
// 2.11 Type Reference
//===----------------------------------------------------------------------------------------====//
// GraphQL describes the types of data expected by query variables. Input types may be lists of
// another input type, or a non‐null variant of any other input type.
//
// Reference: https://facebook.github.io/graphql/June2018/#sec-Type-References

// Type describes a type of data.
//
//	Type
//		NamedType
//		ListType
//		NonNullType
//
// Reference: https://facebook.github.io/graphql/June2018/#Type
type Type interface {
	Node

	// typeNode is a special mark to indicate a Type node. It makes sure that only type node can be
	// assigned to Type.
	typeNode()
}

var (
	_ Type = NamedType{}
	_ Type = ListType{}
	_ Type = NonNullType{}
)

// NullableType is a Type that can be wrapped in NonNullType. More specifically, NamedType and
// ListType.
type NullableType interface {
	Type
	nullableTypeNode()
}

var (
	_ NullableType = NamedType{}
	_ NullableType = ListType{}
)

// NamedType refers to a named type.
type NamedType struct {
	// Name of the type referred by this node
	Name Name
}

// TokenRange implements Node.
func (t NamedType) TokenRange() token.Range {
	return t.Name.TokenRange()
}

// typeNode implements Type.
func (NamedType) typeNode() {}

// nullableTypeNode implements NullableType.
func (NamedType) nullableTypeNode() {}

// ListType referes to a list type of an item type.
type ListType struct {
	// ItemType specifies the type of item in the list.
	ItemType Type
}

// TokenRange implements Node.
func (t ListType) TokenRange() token.Range {
	var r token.Range

	// Find the innermost NameType. Push the intermediate Type to stack.
	stack := []Type{t}

	ttype := t.ItemType
	for r.First == nil {
		switch x := ttype.(type) {
		case NamedType:
			// Set r.First to exit the loop.
			r.First = x.Name.Token
			r.Last = x.Name.Token

		case ListType:
			stack = append(stack, ttype)
			// Unwrap the item type.
			ttype = x.ItemType

		case NonNullType:
			stack = append(stack, ttype)
			// Unwrap the nullable type.
			ttype = x.Type
		}
	}

	// Now, unwind stack to derive the first and last token of the ListType.
	for len(stack) > 0 {
		// Pop one from stack.
		ttype, stack = stack[len(stack)-1], stack[:len(stack)-1]
		switch ttype.(type) {
		case ListType:
			r.First = r.First.Prev // left bracket
			r.Last = r.Last.Next   // right bracket

		case NonNullType:
			r.Last = r.Last.Next // bang!
		}
	}

	return r
}

// typeNode implements Type
func (ListType) typeNode() {}

// nullableTypeNode implements NullableType.
func (ListType) nullableTypeNode() {}

// NonNullType refers to a type that doesn't accept null value.
type NonNullType struct {
	// Type wrapped in this non-null type; Can only be an NamedType or an ListType.
	Type NullableType
}

// TokenRange implements Node.
func (t NonNullType) TokenRange() token.Range {
	r := t.Type.TokenRange()
	// NonNullType ends with bang (!) token which should be next to the last token of the inner type.
	r.Last = r.Last.Next
	return r
}

// typeNode implements Type.
func (NonNullType) typeNode() {}

//===----------------------------------------------------------------------------------------====//
// 2.12 Directives
//===----------------------------------------------------------------------------------------====//
// Directives provide a way to describe alternate runtime execution and type validation behavior in
// a GraphQL document.
//
// Reference: https://facebook.github.io/graphql/June2018/#sec-Language.Directives

// Directives specifies a list of directives
type Directives []*Directive

// FirstToken returns the first token in the sequence of argument.
func (nodes Directives) FirstToken() *token.Token {
	if len(nodes) == 0 {
		return nil
	}
	return nodes[0].FirstToken()
}

// LastToken returns the last token in the sequence of argument.
func (nodes Directives) LastToken() *token.Token {
	if len(nodes) == 0 {
		return nil
	}
	return nodes[len(nodes)-1].LastToken()
}

// Directive applies a GraphQL directive.
type Directive struct {
	// Name of the directive
	Name Name

	// Arguments taken by the directive
	Arguments Arguments
}

var _ Node = (*Directive)(nil)

// FirstToken returns the first token where the Directive begins.
func (node *Directive) FirstToken() *token.Token {
	// Directive begins with the @ token in prior to the name.
	return node.Name.Token.Prev
}

// LastToken returns the last token where the Directive ends.
func (node *Directive) LastToken() *token.Token {
	if len(node.Arguments) == 0 {
		return node.Name.Token
	}
	return node.Arguments.LastToken()
}

// TokenRange implements Node.
func (node *Directive) TokenRange() token.Range {
	return token.Range{
		First: node.FirstToken(),
		Last:  node.LastToken(),
	}
}
