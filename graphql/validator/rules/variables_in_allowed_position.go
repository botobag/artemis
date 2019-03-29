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
)

// VariablesInAllowedPosition implements the "All Variable Usages Are Allowed" validation rule.
//
// See https://graphql.github.io/graphql-spec/June2018/#sec-All-Variable-Usages-are-Allowed.
type VariablesInAllowedPosition struct{}

// A GraphQL operation is only valid if all variables defined by an operation are used, either
// directly or within a spread fragment.

// CheckVariableUsage implements validator.VariableUsageRule.
func (rule VariablesInAllowedPosition) CheckVariableUsage(
	ctx *validator.ValidationContext,
	ttype graphql.Type,
	variable ast.Variable,
	hasLocationDefaultValue bool,
	info *validator.VariableInfo) validator.NextCheckAction {

	if info != nil && ttype != nil {
		var (
			varType = info.TypeDef()
			varDef  = info.Node()
		)

		if varType != nil &&
			!rule.allowedVariableUsage(
				ctx.Schema(),
				varType,
				varDef.DefaultValue,
				ttype,
				hasLocationDefaultValue) {

			ctx.ReportError(
				messages.BadVarPosMessage(
					info.Name(),
					graphql.Inspect(varType),
					graphql.Inspect(ttype),
				),
				[]graphql.ErrorLocation{
					graphql.ErrorLocationOfASTNode(varDef),
					graphql.ErrorLocationOfASTNode(variable),
				},
			)
		}
	}

	return validator.ContinueCheck
}

// Returns true if the variable is allowed in the location it was found,
// which includes considering if default values exist for either the variable
// or the location at which it is located.
func (rule VariablesInAllowedPosition) allowedVariableUsage(
	schema graphql.Schema,
	varType graphql.Type,
	varDefaultValue ast.Value,
	locationType graphql.Type,
	hasLocationDefaultValue bool) bool {

	if locationType, ok := locationType.(graphql.NonNull); ok {
		if !graphql.IsNonNullType(varType) {
			var (
				_, varDefaultValueIsNull       = varDefaultValue.(ast.NullValue)
				hasNonNullVariableDefaultValue = varDefaultValue != nil && !varDefaultValueIsNull
			)

			if !hasNonNullVariableDefaultValue && !hasLocationDefaultValue {
				return false
			}
			return graphql.IsTypeSubTypeOf(schema, varType, locationType.InnerType())
		}
	}

	return graphql.IsTypeSubTypeOf(schema, varType, locationType)
}
