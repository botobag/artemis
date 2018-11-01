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

package value

import (
	"errors"
	"fmt"

	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/graphql/ast"
)

var errUndefinedValue = errors.New("undefined value")
var errAssignNullToNonNull = errors.New("null cannot be assigned to non-null type")

// isMissingVariable returns true if the provided valueNode is a variable which is not defined
// in the set of variables.
func isMissingVariable(value ast.Value, variables graphql.VariableValues) bool {
	if variableValue, isVariableValue := value.(ast.Variable); isVariableValue {
		_, exists := variables.Lookup(variableValue.Name.Value())
		if !exists {
			return true
		}
	}
	return false
}

// CoerceFromAST produces a Go value given a GraphQL Value AST.
//
// A GraphQL type must be provided, which will be used to interpret different
func CoerceFromAST(value ast.Value, t graphql.Type, variables graphql.VariableValues) (interface{}, error) {
	if value == nil {
		// When there is no node, then there is also no value. Return an error to indicate an undefined
		// value.
		return nil, errUndefinedValue
	}

	_, isNullValue := value.(ast.NullValue)
	if t, isNonNullType := t.(*graphql.NonNull); isNonNullType {
		if isNullValue {
			return nil, errAssignNullToNonNull
		}
		return CoerceFromAST(value, t.InnerType(), variables)
	}

	if isNullValue {
		// This is explicitly returning the value null.
		return nil, nil
	}

	// Apply variable value.
	if variableValue, isVariableValue := value.(ast.Variable); isVariableValue {
		varName := variableValue.Name.Value()
		varValue, exists := variables.Lookup(varName)
		if !exists {
			return nil, fmt.Errorf(`value of variable "$%s" is undefined`, varName)
		}

		// Check for non-null before return.
		if varValue == nil && graphql.IsNonNullType(t) {
			return nil, fmt.Errorf(`variable "$%s" does not accept null value`, varName)
		}

		// Note: This does no further checking that this variable is correct. This assumes that this
		// query has been validated and the variable usage here is of the correct type.
		return varValue, nil
	}

	switch ttype := t.(type) {
	case *graphql.List:
		elementType := ttype.ElementType()
		isNonNullElementType := graphql.IsNonNullType(elementType)

		if listValue, isListValue := value.(ast.ListValue); isListValue {
			astValues := listValue.Values()
			coercedValues := make([]interface{}, len(astValues))
			for i, astValue := range astValues {
				if isMissingVariable(astValue, variables) {
					// If an array contains a missing variable, it is either coerced to null or if the item
					// type is non-null, it considered invalid.
					if isNonNullElementType {
						return nil, errors.New("list does not accept null element value")
					}
					coercedValues[i] = nil
				} else { // !isMissingVariable
					elementValue, err := CoerceFromAST(astValue, elementType, variables)
					if err != nil {
						return nil, err
					}
					coercedValues[i] = elementValue
				}
			} // for each AST value node
			return coercedValues, nil
		} // value is an ast.ListValue

		// This is a single value. graphql-js coerce the value with the element type and return an array
		// containing that value.
		coercedValue, err := CoerceFromAST(value, elementType, variables)
		if err != nil {
			return nil, err
		}
		return []interface{}{coercedValue}, nil

	case *graphql.InputObject:
		objectValue, isObjectValue := value.(ast.ObjectValue)
		if !isObjectValue {
			return nil, fmt.Errorf("expected an object value, but got: %T", value)
		}

		astFields := objectValue.Fields()
		// astFieldMap maps field name to ast.ObjectField.
		astFieldMap := make(map[string]*ast.ObjectField, len(astFields))
		for _, astField := range astFields {
			astFieldMap[astField.Name.Value()] = astField
		}

		coercedValues := make(map[string]interface{}, len(astFields))
		for _, field := range ttype.Fields() {
			// Find corresponding ast.Field.
			astField, exists := astFieldMap[field.Name()]
			if !exists || isMissingVariable(astField.Value, variables) {
				if field.HasDefaultValue() {
					coercedValues[field.Name()] = field.DefaultValue()
				} else if graphql.IsNonNullType(field.Type()) {
					return nil, fmt.Errorf(`field "%s" must be assigned with a non-null value`, field.Name())
				}
				continue
			}

			fieldValue, err := CoerceFromAST(astField.Value, field.Type(), variables)
			if err != nil {
				return nil, err
			}
			coercedValues[field.Name()] = fieldValue
		}
		return coercedValues, nil

	case *graphql.Scalar:
		return ttype.CoerceArgumentValue(value)

	case *graphql.Enum:
		return ttype.CoerceArgumentValue(value)
	}

	return nil, fmt.Errorf(`"%v" is not a valid input type`, t)
}
