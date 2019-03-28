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

// KnownDirectives implements the "Directives Are Defined" validation rule.
//
// See https://graphql.github.io/graphql-spec/June2018/#sec-Directives-Are-Defined.
type KnownDirectives struct{}

// CheckDirective implements validator.DirectiveRule.
func (rule KnownDirectives) CheckDirective(
	ctx *validator.ValidationContext,
	directive *validator.DirectiveInfo) validator.NextCheckAction {

	// A GraphQL document is only valid if all `@directives` are known by the schema and legally
	// positioned.

	if directive.Def() == nil {
		ctx.ReportError(
			messages.UnknownDirectiveMessage(directive.Name()),
			graphql.ErrorLocationOfASTNode(directive.Node()),
		)
	}

	return validator.ContinueCheck
}
