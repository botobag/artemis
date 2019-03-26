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

package rules

import (
	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/graphql/ast"
	messages "github.com/botobag/artemis/graphql/internal/validator"
	"github.com/botobag/artemis/graphql/validator"
	"github.com/botobag/artemis/internal/util"
)

// ValuesOfCorrectType implements the "Value Type Correctness" validation rule.
//
// See https://graphql.github.io/graphql-spec/June2018/#sec-Values-of-Correct-Type.
type ValuesOfCorrectType struct{}

// CheckValue implements validator.ValueRule.
func (rule ValuesOfCorrectType) CheckValue(
	ctx *validator.ValidationContext,
	valueType graphql.Type,
	value ast.Value) validator.NextCheckAction {

	switch value := value.(type) {
	case ast.NullValue:
		if graphql.IsNonNullType(valueType) {
			ctx.ReportError(
				messages.BadValueMessage(
					graphql.Inspect(valueType),
					ast.Print(value),
					nil,
				),
				graphql.ErrorLocationOfASTNode(value),
			)
		}

	case ast.ListValue:
		if _, ok := graphql.NullableTypeOf(valueType).(graphql.List); !ok {
			rule.isValidScalar(ctx, valueType, value)
			// Don't traverse further.
			return validator.SkipCheckForChildNodes
		}

	case ast.ObjectValue:
		objectType, ok := graphql.NamedTypeOf(valueType).(graphql.InputObject)
		if !ok {
			rule.isValidScalar(ctx, valueType, value)
			// Don't traverse further.
			return validator.SkipCheckForChildNodes
		}

		// Ensure every required field exists.
		var (
			fieldDefs  = objectType.Fields()
			fieldNodes = value.Fields()
		)
		for fieldName, fieldDef := range fieldDefs {
			if !graphql.IsRequiredInputField(fieldDef) {
				continue
			}

			// Find corresponding field node.
			var fieldNode *ast.ObjectField
			for _, node := range fieldNodes {
				if node.Name.Value() == fieldName {
					fieldNode = node
					break
				}
			}

			if fieldNode == nil {
				ctx.ReportError(
					messages.RequiredFieldMessage(
						objectType.Name(),
						fieldDef.Name(),
						graphql.Inspect(fieldDef.Type()),
					),
					graphql.ErrorLocationOfASTNode(value),
				)
			}
		}

		// Ensure that objectType has fields specified in fieldNodes.
		var fieldNames []string
		for _, fieldNode := range fieldNodes {
			fieldName := fieldNode.Name.Value()
			fieldDef, exists := fieldDefs[fieldName]
			if !exists || !graphql.IsInputType(fieldDef.Type()) {
				if fieldNames == nil {
					fieldNames = make([]string, 0, len(fieldDefs))
					for name := range fieldDefs {
						fieldNames = append(fieldNames, name)
					}
				}

				ctx.ReportError(
					messages.UnknownFieldMessage(
						objectType.Name(),
						fieldName,
						util.SuggestionList(fieldName, fieldNames),
					),
					graphql.ErrorLocationOfASTNode(fieldNode),
				)
			}
		}

	case ast.EnumValue:
		enumType, ok := graphql.NamedTypeOf(valueType).(graphql.Enum)
		if !ok {
			rule.isValidScalar(ctx, valueType, value)
		} else if _, exists := enumType.Values()[value.Value()]; !exists {
			valueName := ast.Print(value)
			ctx.ReportError(
				messages.BadValueMessage(
					graphql.Inspect(enumType),
					valueName,
					rule.enumTypeSuggestion(valueName, enumType),
				),
				graphql.ErrorLocationOfASTNode(value),
			)
		}

	case ast.IntValue, ast.FloatValue, ast.StringValue, ast.BooleanValue:
		rule.isValidScalar(ctx, valueType, value)
	}

	return validator.ContinueCheck
}

// Any value literal may be a valid representation of a Scalar, depending on that scalar type.
func (rule ValuesOfCorrectType) isValidScalar(
	ctx *validator.ValidationContext,
	valueType graphql.Type,
	value ast.Value) {

	// Report any error at the full type expected by the location.
	if !graphql.IsInputType(valueType) {
		return
	}

	var namedType = graphql.NamedTypeOf(valueType)
	if !graphql.IsScalarType(namedType) {
		valueName := ast.Print(value)
		ctx.ReportError(
			messages.BadValueMessage(
				graphql.Inspect(valueType),
				valueName,
				rule.enumTypeSuggestion(valueName, namedType),
			),
			graphql.ErrorLocationOfASTNode(value),
		)
		return
	}

	// Scalars determine if a literal value is valid via CoerceLiteralValue which may throw or return
	// an error value to indicate failure.
	_, err := namedType.(graphql.Scalar).CoerceLiteralValue(value)
	if e, ok := err.(*graphql.Error); ok && e.Kind == graphql.ErrKindCoercion {
		ctx.ReportError(
			messages.BadValueMessage(graphql.Inspect(valueType), ast.Print(value), nil),
			graphql.ErrorLocationOfASTNode(value),
		)
	} else if err != nil {
		ctx.ReportError(
			messages.BadScalarValueMessage(
				graphql.Inspect(valueType),
				ast.Print(value),
				err.Error(),
			),
			graphql.ErrorLocationOfASTNode(value),
			err,
		)
	}
}

func (rule ValuesOfCorrectType) enumTypeSuggestion(valueName string, valueType graphql.Type) []string {
	if enumType, ok := valueType.(graphql.Enum); ok {
		var (
			enumValues = enumType.Values()
			names      = make([]string, 0, len(enumValues))
		)
		for name := range enumValues {
			names = append(names, name)
		}

		return util.SuggestionList(valueName, names)
	}
	return nil
}
