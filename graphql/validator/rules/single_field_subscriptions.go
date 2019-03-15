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

// SingleFieldSubscriptions implements the "Single root field" validation rule.
//
// See https://facebook.github.io/graphql/June2018/#sec-Single-root-field.
type SingleFieldSubscriptions struct{}

// CheckOperation implements validator.OperationRule.
func (rule SingleFieldSubscriptions) CheckOperation(ctx *validator.ValidationContext, operation *ast.OperationDefinition) validator.NextCheckAction {
	// A GraphQL subscription is valid only if it contains a single root field.
	if operation.OperationType() == ast.OperationTypeSubscription {
		// FIXME: The check is not comprehensive. As per spec, we need to run CollectFields algorithm to
		//        expand fragments and take variables into account to evaluate @skip and @include. The
		//        implementation here simply matchs graphql-js.
		if len(operation.SelectionSet) != 1 {
			var (
				name      string
				locations []graphql.ErrorLocation
			)

			if !operation.Name.IsNil() {
				name = operation.Name.Value()
			}

			if len(operation.SelectionSet) > 1 {
				locations = make([]graphql.ErrorLocation, len(operation.SelectionSet)-1)
				for i, selection := range operation.SelectionSet[1:] {
					locations[i] = graphql.ErrorLocationOfASTNode(selection)
				}
			}

			ctx.ReportError(messages.SingleFieldOnlyMessage(name), locations)

			return validator.SkipCheckForChildNodes
		}
	}
	return validator.ContinueCheck
}
