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

// LoneAnonymousOperation implements the "Lone Anonymous Operation" validation rule.
//
// See https://facebook.github.io/graphql/June2018/#sec-Lone-Anonymous-Operation.
type LoneAnonymousOperation struct{}

// CheckOperation implements validator.OperationRule.
func (rule LoneAnonymousOperation) CheckOperation(ctx *validator.ValidationContext, operation *ast.OperationDefinition) validator.NextCheckAction {
	// A GraphQL document is only valid if when it contains an anonymous operation (the query
	// short-hand) that it contains only that one operation definition.
	if operation.Name.IsNil() {
		for _, definition := range ctx.Document().Definitions {
			if op, ok := definition.(*ast.OperationDefinition); ok && op != operation {
				ctx.ReportError(
					messages.AnonOperationNotAloneMessage(),
					graphql.ErrorLocationOfASTNode(operation),
				)
			}
		}
	}

	return validator.ContinueCheck
}
