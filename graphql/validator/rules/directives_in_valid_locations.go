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

// DirectivesInValidLocations implements the "Directives Are In Valid Locations" validation rule.
//
// See https://graphql.github.io/graphql-spec/June2018/#sec-Directives-Are-In-Valid-Locations.
type DirectivesInValidLocations struct{}

// CheckDirective implements validator.DirectiveRule.
func (rule DirectivesInValidLocations) CheckDirective(
	ctx *validator.ValidationContext,
	directive *validator.DirectiveInfo) validator.NextCheckAction {

	// A GraphQL document is only valid if all `@directives` are legally positioned.

	var (
		directiveDef = directive.Def()
		directiveLoc = directive.Location()
	)

	if directiveDef == nil {
		// Skip the check if we don't have directive definition because we don't known which locations
		// are valid for the directive.
		return validator.ContinueCheck
	}

	for _, candidateLoc := range directiveDef.Locations() {
		if directiveLoc == candidateLoc {
			return validator.ContinueCheck
		}
	}

	ctx.ReportError(
		messages.MisplacedDirectiveMessage(directive.Name(), directiveLoc),
		graphql.ErrorLocationOfASTNode(directive.Node()),
	)

	return validator.ContinueCheck
}
