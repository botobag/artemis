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

// UniqueFragmentNames implements the "Unique Fragment Names" validation rule.
//
// See https://graphql.github.io/graphql-spec/June2018/#sec-Fragment-Name-Uniqueness.
type UniqueFragmentNames struct{}

// CheckFragment implements validator.FragmentRule.
func (rule UniqueFragmentNames) CheckFragment(
	ctx *validator.ValidationContext,
	fragmentInfo *validator.FragmentInfo,
	fragment *ast.FragmentDefinition) validator.NextCheckAction {

	// A GraphQL document is only valid if all defined fragments have unique names.
	var (
		knownFragmentNames = ctx.KnownFragmentNames
		fragmentName       = fragment.Name
		fragmentNameValue  = fragmentName.Value()
	)

	if prevName, exists := knownFragmentNames[fragmentNameValue]; exists {
		ctx.ReportError(
			messages.DuplicateFragmentNameMessage(fragmentNameValue),
			[]graphql.ErrorLocation{
				graphql.ErrorLocationOfASTNode(prevName),
				graphql.ErrorLocationOfASTNode(fragmentName),
			},
		)
	} else {
		knownFragmentNames[fragmentNameValue] = fragmentName
	}

	// It is safe to stop running this rule on the child nodes because fragment nodes are only valid
	// to appear at the top-level.
	return validator.SkipCheckForChildNodes
}
