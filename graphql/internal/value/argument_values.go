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

// DefinitionWithArguments describes a GraphQL definition that can accept arguments.
type DefinitionWithArguments interface {
	Args() []graphql.Argument
}

// Both graphql.Field and graphql.Directive have arguments.
var (
	_ DefinitionWithArguments = (graphql.Field)(nil)
	_                         = (*graphql.Directive)(nil)
)

// ASTNodeWithArguments describes an AST node that can accept arguments.
type ASTNodeWithArguments interface {
	ast.Node

	GetArguments() ast.Arguments
}

// ArgumentValues prepares an object map of argument values given a list of argument definitions and
// list of argument AST nodes.
func ArgumentValues(
	def DefinitionWithArguments,
	node ast.NodeWithArguments,
	variableValues graphql.VariableValues) (graphql.ArgumentValues, error) {

	coercedValues := map[string]interface{}{}
	argDefs := def.Args()
	argNodes := node.GetArguments()
	if len(argDefs) == 0 && len(argNodes) == 0 {
		return graphql.NoArgumentValues(), nil
	}

	argNodeMap := make(map[string]*ast.Argument, len(argNodes))
	for _, argNode := range argNodes {
		argNodeMap[argNode.Name.Value()] = argNode
	}

	for _, argDef := range argDefs {
		argName := argDef.Name()
		argType := argDef.Type()
		argNode := argNodeMap[argName]

		var (
			hasValue         bool
			isNil            bool
			argVariable      ast.Variable
			argVariableValue interface{}
		)
		if argNode != nil {
			hasValue = true
			switch argValue := argNode.Value.(type) {
			case ast.Variable:
				argVariable = argValue
				argVariableValue, hasValue = variableValues.Lookup(argVariable.Name.Value())
				isNil = hasValue && (argVariableValue == nil)

			case ast.NullValue:
				isNil = true
			}
		}

		if !hasValue && argDef.HasDefaultValue() {
			// If no argument was provided where the definition has a default value, use the default
			// value.
			coercedValues[argName] = argDef.DefaultValue()
		} else if (!hasValue || isNil) && graphql.IsNonNullType(argType) {
			// If no argument or a null value was provided to an argument with a non-null type (required),
			// produce a field error.
			if isNil {
				return graphql.NoArgumentValues(), graphql.NewError(
					fmt.Sprintf(`Argument "%s" of non-null type "%v" must not be null.`, argName, argType),
					graphql.ErrorLocationOfASTNode(argNode))
			} else if argVariable.Name.Token != nil {
				return graphql.NoArgumentValues(), graphql.NewError(
					fmt.Sprintf(`Argument "%s" of required type "%v" was provided the variable "$%s" which was `+
						`not provided a runtime value.`, argName, argType, argVariable.Name.Value()),
					graphql.ErrorLocationOfASTNode(argNode))
			} else {
				return graphql.NoArgumentValues(), graphql.NewError(
					fmt.Sprintf(`Argument "%s" of required type "%v" was provided.`, argName, argType),
					graphql.ErrorLocationOfASTNode(node))
			}
		} else if hasValue {
			if argVariable.Name.Token != nil {
				// Note: This does no further checking that this variable is correct.  This assumes that
				// this query has been validated and the variable usage here is of the correct type.
				coercedValues[argName] = argVariableValue
			} else if isNil {
				// If the explicit value `null` was provided, an entry in the coerced values must exist as
				// the value `null`.
				coercedValues[argName] = nil
			} else {
				argValue := argNode.Value
				coercedValue, err := CoerceFromAST(argValue, argType, variableValues)
				if err != nil {
					// Note: ValuesOfCorrectType validation should catch this before execution. This is a
					// runtime check to ensure execution does not continue with an invalid argument value.
					return graphql.NoArgumentValues(), graphql.NewError(
						fmt.Sprintf(`Argument "%s" has invalid value %s.`,
							argName, graphql.Inspect(argValue.Interface())),
						graphql.ErrorLocationOfASTNode(argValue), err)
				}
				coercedValues[argName] = coercedValue
			}
		}
	}

	return graphql.NewArgumentValues(coercedValues), nil
}
