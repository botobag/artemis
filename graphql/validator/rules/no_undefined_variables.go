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

// NoUndefinedVariables implements the "All Variable Uses Defined" validation rule.
//
// See https://graphql.github.io/graphql-spec/June2018/#sec-All-Variable-Uses-Defined.
type NoUndefinedVariables struct{}

// CheckVariableUsage implements validator.VariableUsageRule.
func (rule NoUndefinedVariables) CheckVariableUsage(
	ctx *validator.ValidationContext,
	ttype graphql.Type,
	variable ast.Variable,
	hasLocationDefaultValue bool,
	info *validator.VariableInfo) validator.NextCheckAction {

	// A GraphQL operation is only valid if all variables encountered, both directly and via fragment
	// spreads, are defined by that operation.

	if info == nil {
		var (
			operationName string
			operation     = ctx.CurrentOperation()
		)
		if !operation.Name.IsNil() {
			operationName = operation.Name.Value()
		}

		ctx.ReportError(
			messages.UndefinedVarMessage(variable.Name.Value(), operationName),
			[]graphql.ErrorLocation{
				graphql.ErrorLocationOfASTNode(variable),
				graphql.ErrorLocationOfASTNode(operation),
			},
		)
	}

	return validator.ContinueCheck
}
