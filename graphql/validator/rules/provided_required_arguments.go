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

// ProvidedRequiredArguments implements the "Required Arguments" validation rule.
//
// See https://graphql.github.io/graphql-spec/June2018/#sec-Required-Arguments.
type ProvidedRequiredArguments struct {
	ProvidedRequiredArgumentsOnDirectives
}

// CheckField implements validator.FieldRule.
func (rule ProvidedRequiredArguments) CheckField(
	ctx *validator.ValidationContext,
	field *validator.FieldInfo) validator.NextCheckAction {

	// A field or directive is only valid if all required (non-null without a default value) field
	// arguments have been provided.
	fieldDef := field.Def()

	if fieldDef == nil {
		// If we're unable to resolve field and parent type statically, we don't have argument
		// definitions for the field. Skip the check.
		return validator.ContinueCheck
	}

	var (
		fieldNode = field.Node()
		argNodes  = fieldNode.Arguments
		argDefs   = fieldDef.Args()
	)

check_next_arg:
	for i := range argDefs {
		argDef := &argDefs[i]
		if !graphql.IsRequiredArgument(argDef) {
			continue
		}

		// Find corresponding argNode.
		argName := argDef.Name()
		for _, argNode := range argNodes {
			if argNode.Name.Value() == argName {
				continue check_next_arg
			}
		}

		ctx.ReportError(
			messages.MissingFieldArgMessage(
				argName,
				fieldDef.Name(),
				graphql.Inspect(argDef.Type()),
			),
			graphql.ErrorLocationOfASTNode(fieldNode),
		)
	}

	return validator.ContinueCheck
}

// ProvidedRequiredArgumentsOnDirectives checks the "Required Arguments" validation rule on
// directives.
//
// See https://graphql.github.io/graphql-spec/June2018/#sec-Required-Arguments.
type ProvidedRequiredArgumentsOnDirectives struct{}

// CheckDirective implements validator.DirectiveRule.
func (rule ProvidedRequiredArguments) CheckDirective(
	ctx *validator.ValidationContext,
	directive *validator.DirectiveInfo) validator.NextCheckAction {

	directiveDef := directive.Def()

	if directiveDef == nil {
		// We cannot run the validation if we're unable to find directive definition in schema. Quick
		// return to Skip the check in this case.
		return validator.ContinueCheck
	}

	var (
		directiveNode = directive.Node()
		argNodes      = directiveNode.Arguments
		argDefs       = directiveDef.Args()
	)

check_next_arg:
	for i := range argDefs {
		argDef := &argDefs[i]
		if !graphql.IsRequiredArgument(argDef) {
			continue
		}

		// Find corresponding argNode.
		argName := argDef.Name()
		for _, argNode := range argNodes {
			if argNode.Name.Value() == argName {
				continue check_next_arg
			}
		}

		ctx.ReportError(
			messages.MissingDirectiveArgMessage(
				argName,
				directiveDef.Name(),
				graphql.Inspect(argDef.Type()),
			),
			graphql.ErrorLocationOfASTNode(directiveNode),
		)
	}

	return validator.ContinueCheck
}
