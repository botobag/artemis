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

// CheckField implements validator.FieldRule.
func (rule KnownArgumentNames) CheckField(
	ctx *validator.ValidationContext,
	parentType graphql.Type,
	fieldDef graphql.Field,
	field *ast.Field) validator.NextCheckAction {

	// A GraphQL field is only valid if all supplied arguments are defined by that field.

	if fieldDef == nil || parentType == nil {
		// If we're unable to resolve field and parent type statically, we don't have argument
		// definitions for the field. Skip the check.
		return validator.ContinueCheck
	}

	var (
		argsNode       = field.Arguments
		argsDef        = fieldDef.Args()
		knownArgsNames []string
	)

	for _, argNode := range argsNode {
		var argDef *graphql.Argument

		// Search definition for argNode from argsDef by name.
		argName := argNode.Name.Value()
		for i := range argsDef {
			if argsDef[i].Name() == argName {
				argDef = &argsDef[i]
				break
			}
		}

		if argDef == nil {
			if knownArgsNames == nil {
				knownArgsNames = make([]string, len(argsDef))
				for i := range argsDef {
					knownArgsNames[i] = argsDef[i].Name()
				}
			}

			ctx.ReportError(
				messages.UnknownArgMessage(
					argName,
					fieldDef.Name(),
					parentType.(graphql.TypeWithName).Name(),
					util.SuggestionList(argName, knownArgsNames),
				),
				graphql.ErrorLocationOfASTNode(argNode),
			)
		} // if argDef == nil
	}

	return validator.ContinueCheck
}

// KnownArgumentNamesOnDirectives checks the "Argument Names" validation rule on directives.
//
// See https://facebook.github.io/graphql/June2018/#sec-Argument-Names.
type KnownArgumentNamesOnDirectives struct{}

// CheckDirective implements validator.DirectiveRule.
func (rule KnownArgumentNames) CheckDirective(
	ctx *validator.ValidationContext,
	directiveDef graphql.Directive,
	directive *ast.Directive,
	location graphql.DirectiveLocation) validator.NextCheckAction {

	if directiveDef == nil {
		// We cannot run the validation if we're unable to find directive definition in schema. Quick
		// return to Skip the check in this case.
		return validator.ContinueCheck
	}

	var (
		argsDef   = directiveDef.Args()
		knownArgs []string
	)
	for _, arg := range directive.Arguments {
		// Find corresponding definition for arg.
		var argDef *graphql.Argument
		argName := arg.Name.Value()
		for i := range argsDef {
			if argsDef[i].Name() == argName {
				argDef = &argsDef[i]
				break
			}
		}

		if argDef == nil {
			if knownArgs == nil {
				knownArgs = make([]string, len(argsDef))
				for i := range argsDef {
					knownArgs[i] = argsDef[i].Name()
				}
			}

			ctx.ReportError(
				messages.UnknownDirectiveArgMessage(
					argName,
					directive.Name.Value(),
					util.SuggestionList(argName, knownArgs),
				),
				graphql.ErrorLocationOfASTNode(arg),
			)
		} // if argDef == nil
	}

	return validator.ContinueCheck
}
