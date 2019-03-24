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
	"github.com/botobag/artemis/internal/util"
)

// KnownTypeNames implements the "Fragment Spread Type Existence" validation rule.
//
// See https://facebook.github.io/graphql/June2018/#sec-Fragment-Spread-Type-Existence.
type KnownTypeNames struct{}

// A GraphQL document is only valid if referenced types (specifically variable definitions and
// fragment conditions) are defined by the type schema.

// CheckOperation implements validator.OperationRule.
func (rule KnownTypeNames) CheckOperation(ctx *validator.ValidationContext, operation *ast.OperationDefinition) validator.NextCheckAction {
	for _, varDef := range operation.VariableDefinitions {
		rule.checkType(ctx, varDef.Type)
	}
	return validator.ContinueCheck
}

// CheckFragment implements validator.FragmentRule.
func (rule KnownTypeNames) CheckFragment(
	ctx *validator.ValidationContext,
	fragmentInfo *validator.FragmentInfo,
	fragment *ast.FragmentDefinition) validator.NextCheckAction {

	rule.checkType(ctx, fragment.TypeCondition)
	return validator.ContinueCheck
}

// CheckInlineFragment implements validator.InlineFragmentRule.
func (rule KnownTypeNames) CheckInlineFragment(
	ctx *validator.ValidationContext,
	parentType graphql.Type,
	typeCondition graphql.Type,
	fragment *ast.InlineFragment) validator.NextCheckAction {

	if fragment.HasTypeCondition() {
		rule.checkType(ctx, fragment.TypeCondition)
	}
	return validator.ContinueCheck
}

func (rule KnownTypeNames) checkType(ctx *validator.ValidationContext, typeNode ast.Type) {
search_named_type:
	for {
		switch node := typeNode.(type) {
		case ast.NamedType:
			break search_named_type
		case ast.ListType:
			typeNode = node.ItemType
		case ast.NonNullType:
			typeNode = node.Type
		}
	}

	typeName := typeNode.(ast.NamedType).Name.Value()
	if ctx.Schema().TypeMap().Lookup(typeName) == nil {
		ctx.ReportError(
			messages.UnknownTypeMessage(
				typeName,
				util.SuggestionList(typeName, ctx.ExistingTypeNames()),
			),
			graphql.ErrorLocationOfASTNode(typeNode),
		)
	}
}
