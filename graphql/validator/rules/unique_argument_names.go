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

// UniqueArgumentNames implements the "Argument Uniqueness" validation rule.
//
// See https://facebook.github.io/graphql/June2018/#sec-Argument-Uniqueness.
type UniqueArgumentNames struct{}

// CheckField implements validator.FieldRule.
func (rule UniqueArgumentNames) CheckField(
	ctx *validator.ValidationContext,
	parentType graphql.Type,
	fieldDef graphql.Field,
	field *ast.Field) validator.NextCheckAction {

	// A GraphQL field or directive is only valid if all supplied arguments are uniquely named.
	rule.checkUniqueArgNames(ctx, field.Arguments)
	return validator.ContinueCheck
}

// CheckDirective implements validator.DirectiveRule.
func (rule UniqueArgumentNames) CheckDirective(
	ctx *validator.ValidationContext,
	directiveDef graphql.Directive,
	directive *ast.Directive,
	location graphql.DirectiveLocation) validator.NextCheckAction {
	rule.checkUniqueArgNames(ctx, directive.Arguments)
	return validator.ContinueCheck
}

func (rule UniqueArgumentNames) checkUniqueArgNames(ctx *validator.ValidationContext, args ast.Arguments) {
	if len(args) == 0 {
		return
	}

	knownArgNames := make(map[string]ast.Name, len(args))
	for _, arg := range args {
		var (
			argName      = arg.Name
			argNameValue = argName.Value()
		)
		if prevArgName, exists := knownArgNames[argNameValue]; exists {
			ctx.ReportError(
				messages.DuplicateArgMessage(argNameValue),
				[]graphql.ErrorLocation{
					graphql.ErrorLocationOfASTNode(prevArgName),
					graphql.ErrorLocationOfASTNode(argName),
				},
			)
			continue
		}
		knownArgNames[argNameValue] = argName
	}
}
