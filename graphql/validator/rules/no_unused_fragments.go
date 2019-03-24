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

// NoUnusedFragments implements the "Fragments must be used" validation rule.
//
// See https://facebook.github.io/graphql/June2018/#sec-Fragments-Must-Be-Used.
type NoUnusedFragments struct{}

// A GraphQL document is only valid if all fragment definitions are spread within operations, or
// spread within other fragments spread within operations.

// CheckFragmentSpread implements validator.FragmentSpreadRule.
func (rule NoUnusedFragments) CheckFragmentSpread(
	ctx *validator.ValidationContext,
	parentType graphql.Type,
	fragmentInfo *validator.FragmentInfo,
	fragmentSpread *ast.FragmentSpread) validator.NextCheckAction {

	if fragmentInfo != nil {
		// Mark fragment to be used.
		fragmentInfo.RecursivelyMarkUsed(ctx)
	}
	return validator.ContinueCheck
}

// CheckFragment implements validator.FragmentRule.
func (rule NoUnusedFragments) CheckFragment(
	ctx *validator.ValidationContext,
	fragmentInfo *validator.FragmentInfo,
	fragment *ast.FragmentDefinition) validator.NextCheckAction {

	if !fragmentInfo.Used() {
		ctx.ReportError(
			messages.UnusedFragMessage(fragment.Name.Value()),
			graphql.ErrorLocationOfASTNode(fragment),
		)
	}

	// Skip scanning fragment definition body. This is safe and required. FragmentDefinition should
	// only appear as top-level definition of a GraphQL document. Skipping child nodes also prevents
	// FragmentSpread's within the FragmentDefinition from being accounted to the uses of fragments.
	return validator.SkipCheckForChildNodes
}
