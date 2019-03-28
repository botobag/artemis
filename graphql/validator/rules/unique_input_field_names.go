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

// UniqueInputFieldNames implements the "Input Object Field Uniqueness" validation rule.
//
// See https://graphql.github.io/graphql-spec/June2018/#sec-Input-Object-Field-Uniqueness.
type UniqueInputFieldNames struct{}

// CheckValue implements validator.ValueRule.
func (rule UniqueInputFieldNames) CheckValue(
	ctx *validator.ValidationContext,
	valueType graphql.Type,
	value ast.Value) validator.NextCheckAction {

	// A GraphQL input object value is only valid if all supplied fields are uniquely named.

	// The rule only applies on object value.
	objectValue, ok := value.(ast.ObjectValue)
	if !ok {
		return validator.ContinueCheck
	}

	fieldNodes := objectValue.Fields()
	if len(fieldNodes) > 0 {
		knownNames := make(map[string]ast.Name, len(fieldNodes))
		for _, fieldNode := range fieldNodes {
			var (
				fieldName      = fieldNode.Name
				fieldNameValue = fieldName.Value()
			)
			prevName, exists := knownNames[fieldNameValue]
			if !exists {
				knownNames[fieldNameValue] = fieldName
			} else {
				ctx.ReportError(
					messages.DuplicateInputFieldMessage(fieldNameValue),
					[]graphql.ErrorLocation{
						graphql.ErrorLocationOfASTNode(prevName),
						graphql.ErrorLocationOfASTNode(fieldName),
					},
				)
			}
		}
	}

	return validator.ContinueCheck
}
