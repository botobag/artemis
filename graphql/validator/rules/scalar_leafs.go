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
	messages "github.com/botobag/artemis/graphql/internal/validator"
	"github.com/botobag/artemis/graphql/validator"
)

// ScalarLeafs implements the "Leaf Field Selections" validation rule.
//
// See https://graphql.github.io/graphql-spec/June2018/#sec-Leaf-Field-Selections.
type ScalarLeafs struct{}

// CheckField implements validator.FieldRule.
func (rule ScalarLeafs) CheckField(
	ctx *validator.ValidationContext,
	field *validator.FieldInfo) validator.NextCheckAction {

	// A GraphQL document is valid only if all leaf fields (fields without sub selections) are of
	// scalar or enum types.
	var (
		fieldDef  = field.Def()
		fieldNode = field.Node()
	)

	if fieldDef != nil {
		fieldType := fieldDef.Type()
		selectionSet := fieldNode.SelectionSet
		if graphql.IsLeafType(graphql.NamedTypeOf(fieldType)) {
			if len(selectionSet) > 0 {
				ctx.ReportError(
					messages.NoSubselectionAllowedMessage(
						field.Name(),
						graphql.Inspect(fieldType),
					),
					graphql.ErrorLocationOfASTNode(selectionSet),
				)
			}
		} else if len(selectionSet) == 0 {
			ctx.ReportError(
				messages.RequiredSubselectionMessage(
					field.Name(),
					graphql.Inspect(fieldType),
				),
				graphql.ErrorLocationOfASTNode(fieldNode),
			)
		}
	}

	return validator.ContinueCheck
}
