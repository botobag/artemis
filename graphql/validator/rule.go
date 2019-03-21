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

package validator

import (
	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/graphql/ast"
)

// A Rule implements an ast.Visitor to validate nodes in a GraphQL document according to one of the
// sections under "Validation" in specification [0].
//
// [0]: https://facebook.github.io/graphql/June2018/#sec-Validation

// NextCheckAction is the type of return value from rule's Check function. It specifies which action
// to take when the rule is invoked next time in current validation request.
type NextCheckAction int

// Enumeration of NextCheckAction
const (
	// Continue running the rule
	ContinueCheck NextCheckAction = iota

	// Don't run the rule on any of child nodes of the current one
	SkipCheckForChildNodes

	// Stop running the rule in current validation request
	StopCheck
)

// OperationRule validates an OperationDefinition.
type OperationRule interface {
	CheckOperation(ctx *ValidationContext, operation *ast.OperationDefinition) NextCheckAction
}

// FragmentRule validates an FragmentDefinition.
type FragmentRule interface {
	CheckFragment(ctx *ValidationContext, fragment *ast.FragmentDefinition) NextCheckAction
}

// SelectionSetRule validates a SelectionSet.
type SelectionSetRule interface {
	CheckSelectionSet(
		ctx *ValidationContext,
		ttype graphql.Type,
		selectionSet ast.SelectionSet) NextCheckAction
}

// FieldRule validates a Field.
type FieldRule interface {
	CheckField(
		ctx *ValidationContext,
		parentType graphql.Type,
		fieldDef graphql.Field,
		field *ast.Field) NextCheckAction
}

// InlineFragmentRule validates a InlineFragment.
type InlineFragmentRule interface {
	CheckInlineFragment(
		ctx *ValidationContext,
		parentType graphql.Type,
		fragment *ast.InlineFragment) NextCheckAction
}

// DirectiveRule validates a Directive.
type DirectiveRule interface {
	CheckDirective(
		ctx *ValidationContext,
		directiveDef graphql.Directive,
		directive *ast.Directive,
		location graphql.DirectiveLocation) NextCheckAction
}
