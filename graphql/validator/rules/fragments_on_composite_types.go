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

// FragmentsOnCompositeTypes implements the "Fragments on Composite Types" validation rule.
//
// See https://facebook.github.io/graphql/June2018/#sec-Fragments-On-Composite-Types.
type FragmentsOnCompositeTypes struct{}

// Fragments use a type condition to determine if they apply, since fragments can only be spread
// into a composite type (object, interface, or union), the type condition must also be a composite
// type.

// CheckFragment implements validator.FragmentRule.
func (rule FragmentsOnCompositeTypes) CheckFragment(
	ctx *validator.ValidationContext,
	fragmentInfo *validator.FragmentInfo,
	fragment *ast.FragmentDefinition) validator.NextCheckAction {

	typeCondition := fragmentInfo.TypeCondition()
	if typeCondition != nil && !graphql.IsCompositeType(typeCondition) {
		ctx.ReportError(
			messages.FragmentOnNonCompositeErrorMessage(
				fragment.Name.Value(),
				ast.Print(fragment.TypeCondition),
			),
			graphql.ErrorLocationOfASTNode(fragment.TypeCondition),
		)
	}

	return validator.ContinueCheck
}

// CheckInlineFragment implements validator.InlineFragmentRule.
func (rule FragmentsOnCompositeTypes) CheckInlineFragment(
	ctx *validator.ValidationContext,
	parentType graphql.Type,
	fragment *ast.InlineFragment) validator.NextCheckAction {

	if fragment.HasTypeCondition() && parentType != nil {
		// parentType must be resolved to the type condition in this case.
		if !graphql.IsCompositeType(parentType) {
			ctx.ReportError(
				messages.InlineFragmentOnNonCompositeErrorMessage(ast.Print(fragment.TypeCondition)),
				graphql.ErrorLocationOfASTNode(fragment.TypeCondition),
			)
		}
	}

	return validator.ContinueCheck
}
