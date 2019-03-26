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

// KnownArgumentNames implements the "Argument Names" validation rule.
//
// See https://facebook.github.io/graphql/June2018/#sec-Argument-Names.
type KnownArgumentNames struct {
	KnownArgumentNamesOnDirectives
}

// CheckFieldArgument implements validator.FieldArgumentRule.
func (rule KnownArgumentNames) CheckFieldArgument(
	ctx *validator.ValidationContext,
	field *validator.FieldInfo,
	argDef *graphql.Argument,
	arg *ast.Argument) validator.NextCheckAction {

	// A GraphQL field is only valid if all supplied arguments are defined by that field.

	if argDef != nil {
		// Argument is defined.
		return validator.ContinueCheck
	}

	var (
		fieldDef   = field.Def()
		parentType = field.ParentType()
	)

	// If we're unable to resolve field and parent type statically, we don't have argument
	// definitions for the field. Don't throw error for this case.
	if fieldDef == nil || parentType == nil {
		return validator.ContinueCheck
	}

	argName := arg.Name.Value()
	ctx.ReportError(
		messages.UnknownArgMessage(
			argName,
			fieldDef.Name(),
			parentType.(graphql.TypeWithName).Name(),
			util.SuggestionList(argName, field.KnownArgNames()),
		),
		graphql.ErrorLocationOfASTNode(arg),
	)

	return validator.ContinueCheck
}

// KnownArgumentNamesOnDirectives checks the "Argument Names" validation rule on directives.
//
// See https://facebook.github.io/graphql/June2018/#sec-Argument-Names.
type KnownArgumentNamesOnDirectives struct{}

// CheckDirectiveArgument implements validator.DirectiveArgumentRule.
func (rule KnownArgumentNames) CheckDirectiveArgument(
	ctx *validator.ValidationContext,
	directive *validator.DirectiveInfo,
	argDef *graphql.Argument,
	arg *ast.Argument) validator.NextCheckAction {

	if argDef != nil {
		// Quick return for known arguments.
		return validator.ContinueCheck
	}

	if directive.Def() == nil {
		// We cannot run the validation if we're unable to find directive definition in schema. Quick
		// return to Skip the check in this case.
		return validator.ContinueCheck
	}

	argName := arg.Name.Value()
	ctx.ReportError(
		messages.UnknownDirectiveArgMessage(
			argName,
			directive.Name(),
			util.SuggestionList(argName, directive.KnownArgNames()),
		),
		graphql.ErrorLocationOfASTNode(arg),
	)

	return validator.ContinueCheck
}
