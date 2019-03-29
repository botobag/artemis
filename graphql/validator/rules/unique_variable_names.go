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

// UniqueVariableNames implements the "Variable Uniqueness" validation rule.
//
// See https://graphql.github.io/graphql-spec/June2018/#sec-Variable-Uniqueness.
type UniqueVariableNames struct{}

// CheckOperation implements validator.DirectiveRule.
func (rule UniqueVariableNames) CheckOperation(
	ctx *validator.ValidationContext,
	operation *ast.OperationDefinition) validator.NextCheckAction {

	// A GraphQL operation is only valid if all its variables are uniquely named.

	varDefs := operation.VariableDefinitions
	if len(varDefs) > 0 {
		knownVariableNames := make(map[string]ast.Name, len(varDefs))
		for _, varDef := range varDefs {
			var (
				varName      = varDef.Variable.Name
				varNameValue = varName.Value()
			)
			prevVar, exists := knownVariableNames[varNameValue]
			if !exists {
				knownVariableNames[varNameValue] = varName
			} else {
				ctx.ReportError(
					messages.DuplicateVariableMessage(varNameValue),
					[]graphql.ErrorLocation{
						graphql.ErrorLocationOfASTNode(prevVar),
						graphql.ErrorLocationOfASTNode(varName),
					},
				)
			}
		}
	}

	return validator.ContinueCheck
}
