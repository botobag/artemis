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

// NoUnusedVariables implements the "All Variables Used" validation rule.
//
// See https://graphql.github.io/graphql-spec/June2018/#sec-All-Variables-Used.
type NoUnusedVariables struct{}

// A GraphQL operation is only valid if all variables defined by an operation are used, either
// directly or within a spread fragment.

// CheckVariableUsage implements validator.VariableUsageRule.
func (rule NoUnusedVariables) CheckVariableUsage(
	ctx *validator.ValidationContext,
	ttype graphql.Type,
	variable ast.Variable,
	hasLocationDefaultValue bool,
	info *validator.VariableInfo) validator.NextCheckAction {

	if info != nil {
		info.MarkUsed()
	}

	return validator.ContinueCheck
}

// CheckVariable implements validator.VariableRule.
func (rule NoUnusedVariables) CheckVariable(
	ctx *validator.ValidationContext,
	info *validator.VariableInfo) validator.NextCheckAction {

	if !info.Used() {
		var (
			operationName string
			operation     = ctx.CurrentOperation()
		)
		if !operation.Name.IsNil() {
			operationName = operation.Name.Value()
		}
		ctx.ReportError(
			messages.UnusedVariableMessage(info.Name(), operationName),
			graphql.ErrorLocationOfASTNode(info.Node()),
		)
	}

	return validator.ContinueCheck
}
