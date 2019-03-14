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

// A ValidationContext stores various states for running walk function and validation rules.
type ValidationContext struct {
	schema   graphql.Schema
	document ast.Document
	rules    *rules

	// Error list
	errs graphql.Errors

	//===----------------------------------------------------------------------------------------====//
	// States for "rules".
	//===----------------------------------------------------------------------------------------====//

	// "Skipping" state for the rule at index i; Possible values are:
	//
	// - nil: run the rule
	// - Break: stop applying the rule on any nodes
	// - an ast.Node: don't apply the rule on the child nodes of the given node
	skippingRules []interface{}

	//===----------------------------------------------------------------------------------------====//
	// States for walk functions
	//===----------------------------------------------------------------------------------------====//

	// Operation in the document that is being validated
	currentOperation *ast.OperationDefinition

	//===----------------------------------------------------------------------------------------====//
	// States for rules package
	//===----------------------------------------------------------------------------------------====//

	// UniqueOperationNames
	KnownOperationNames map[string]ast.Name
}

// newValidationContext initializes a validation context for validating given document.
func newValidationContext(schema graphql.Schema, document ast.Document, rules *rules) *ValidationContext {
	return &ValidationContext{
		schema:   schema,
		document: document,
		rules:    rules,

		skippingRules: make([]interface{}, rules.numRules),

		KnownOperationNames: map[string]ast.Name{},
	}
}

// Schema returns schema of the document being validated.
func (ctx *ValidationContext) Schema() graphql.Schema {
	return ctx.schema
}

// Document returns the document being validated.
func (ctx *ValidationContext) Document() ast.Document {
	return ctx.document
}

// CurrentOperation returns the operation in the document being validated.
func (ctx *ValidationContext) CurrentOperation() *ast.OperationDefinition {
	return ctx.currentOperation
}

// ReportError constructs a graphql.Error from message and args and appends to current validation
// context for reporting.
func (ctx *ValidationContext) ReportError(message string, args ...interface{}) {
	ctx.errs.Emplace(message, args...)
}
