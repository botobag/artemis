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

// VariablesAreInputTypes implements the "Variables Are Input Types" validation rule.
//
// See https://graphql.github.io/graphql-spec/June2018/#sec-Variables-Are-Input-Types.
type VariablesAreInputTypes struct{}

// CheckVariable implements validator.VariableRule.
func (rule VariablesAreInputTypes) CheckVariable(
	ctx *validator.ValidationContext,
	variable *ast.VariableDefinition,
	ttype graphql.Type) validator.NextCheckAction {

	// A GraphQL operation is only valid if all the variables it defines are of input types (scalar,
	// enum, or input object).

	if ttype != nil && !graphql.IsInputType(ttype) {
		var (
			varName = variable.Variable.Name.Value()
			varType = variable.Type
		)
		ctx.ReportError(
			messages.NonInputTypeOnVarMessage(varName, ast.Print(varType)),
			graphql.ErrorLocationOfASTNode(varType),
		)
	}

	return validator.ContinueCheck
}
