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

// Validate implements the "Validation" section of the spec.
//
// Validation runs synchronously, returning a graphql.Errors containing encountered errors, or
// graphql.NoErrors if no errors were encountered and the document is valid.
//
// It uses the rules defined by the GraphQL specification to validate the given document.
func Validate(schema graphql.Schema, document ast.Document) graphql.Errors {
	ctx := newValidationContext(schema, document, StandardRules())
	walk(ctx)
	return ctx.errs
}

// ValidateWithRules runs a list of specific validation rules on the given document. Every rule in
// rs must implement at least one of the following interfaces:
//
//  OperationRule
//  VariableRule
//  FragmentRule
//  SelectionSetRule
//  FieldRule
//  FieldArgumentRule
//  InlineFragmentRule
//  FragmentSpreadRule
//  ValueRule
//  VariableUsageRule
//  DirectivesRule
//  DirectiveRule
//  DirectiveArgumentRule
func ValidateWithRules(schema graphql.Schema, document ast.Document, rs ...interface{}) graphql.Errors {
	if len(rs) == 0 {
		// No validation are provided to run which disable validation effectively.
		return graphql.NoErrors()
	}

	ctx := newValidationContext(schema, document, buildRules(rs...))
	walk(ctx)
	return ctx.errs
}
