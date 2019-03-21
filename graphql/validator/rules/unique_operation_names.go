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

// UniqueOperationNames implements the "Operation Name Uniqueness" validation rule.
//
// See https://facebook.github.io/graphql/June2018/#sec-Operation-Name-Uniqueness.
type UniqueOperationNames struct{}

// CheckOperation implements validator.OperationRule.
func (rule UniqueOperationNames) CheckOperation(ctx *validator.ValidationContext, operation *ast.OperationDefinition) validator.NextCheckAction {
	operationName := operation.Name
	if !operationName.IsNil() {
		operationNameValue := operationName.Value()
		knownOperationNames := ctx.KnownOperationNames
		if prevName, exists := knownOperationNames[operationNameValue]; exists {
			ctx.ReportError(
				messages.DuplicateOperationNameMessage(operationNameValue),
				[]graphql.ErrorLocation{
					graphql.ErrorLocationOfASTNode(prevName),
					graphql.ErrorLocationOfASTNode(operationName),
				},
			)
		} else {
			knownOperationNames[operationNameValue] = operationName
		}
	}

	// It is safe to stop running this rule on the child nodes because operation nodes are only valid
	// to appear at the top-level.
	return validator.SkipCheckForChildNodes
}
