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
	"fmt"

	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/graphql/ast"
)

// CoerceVariableValues prepares an object map of variableValues of the correct type based on the
// provided variable definitions and arbitrary input. If the input cannot be parsed to match the
// variable definitions, a graphql.Error will be raised.
func CoerceVariableValues(
	schema graphql.Schema,
	variableDefinitions []*ast.VariableDefinition,
	inputValues map[string]interface{}) (graphql.VariableValues, graphql.Errors) {
	var errs graphql.Errors

	coercedValues := map[string]interface{}{}
	for _, varDefNode := range variableDefinitions {
		varName := varDefNode.Variable.Name.Value()
		varType := schema.TypeFromAST(varDefNode.Type)
		// Note that IsInputType also returns false for nil type.
		if !graphql.IsInputType(varType) {
			// Must use input types for variables. This should be caught during validation, however is
			// checked again here for safety.
			errs.Emplace(fmt.Sprintf(`Variable "$%s" expected value of type "%s" which cannot be used `+
				`as an input type.`, varName, graphql.Inspect(varType)),
				graphql.ErrorLocationOfASTNode(varDefNode))
		} else {
			value, hasValue := inputValues[varName]
			if !hasValue && varDefNode.DefaultValue != nil {
				if varDefNode.DefaultValue != nil {
					// If no value was provided to a variable with a default value, use the default value.
					coerced, err := CoerceFromAST(varDefNode.DefaultValue, varType, graphql.NoVariableValues())
					if err == nil {
						// Only store the result when FromAST succeeds.
						coercedValues[varName] = coerced
					}
					// Ignore the error.
				}
			} else if (!hasValue || value == nil) && graphql.IsNonNullType(varType) {
				var message string
				if hasValue {
					message = fmt.Sprintf(`Variable "$%s" of non-null type "%s" must not be null.`,
						varName, graphql.Inspect(varType))
				} else {
					message = fmt.Sprintf(`Variable "$%s" of required type "%s" was not provided.`,
						varName, graphql.Inspect(varType))
				}
				errs.Emplace(message, graphql.ErrorLocationOfASTNode(varDefNode))
			} else { // hasValue && varType is nullable
				if value == nil {
					// If the explicit value `null` was provided, an entry in the coerced
					// values must exist as the value `null`.
					coercedValues[varName] = nil
				} else {
					// Otherwise, a non-null value was provided, coerce it to the expected type or report an
					// error if coercion fails.
					coerced, coercionErrs := CoerceValue(value, varType, varDefNode)
					if !coercionErrs.HaveOccurred() {
						coercedValues[varName] = coerced
					} else {
						for _, err := range coercionErrs.Errors {
							var message string
							if err.Kind == graphql.ErrKindCoercion {
								// Include err.message.
								message = fmt.Sprintf(`Variable "$%s" got invalid value %s; %s`,
									varName, graphql.Inspect(value), err.Message)
							} else {
								message = fmt.Sprintf(`Variable "$%s" got invalid value %s.`,
									varName, graphql.Inspect(value))
							}
							// Push the error.
							errs.Emplace(message, err)
						}
					}
				}
			}
		}
	}

	if errs.HaveOccurred() {
		return graphql.NoVariableValues(), errs
	}

	return graphql.NewVariableValues(coercedValues), graphql.NoErrors()
}
