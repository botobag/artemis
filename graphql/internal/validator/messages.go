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
	"fmt"

	"github.com/botobag/artemis/internal/util"
)

// DuplicateOperationNameMessage returns message describing error occurred in rule "Operation Name
// Uniqueness" (rules.UniqueOperationNames).
func DuplicateOperationNameMessage(operationName string) string {
	return fmt.Sprintf(`There can be only one operation named "%s".`, operationName)
}

// AnonOperationNotAloneMessage returns message describing error occurred in rule "Lone Anonymous
// Operation" (rules.LoneAnonymousOperation).
func AnonOperationNotAloneMessage() string {
	return "This anonymous operation must be the only defined operation."
}

// SingleFieldOnlyMessage returns message describing error occurred in rule "Single Field
// Subscriptions" (rules.SingleFieldSubscriptions).
func SingleFieldOnlyMessage(name string) string {
	if len(name) == 0 {
		return "Anonymous Subscription must select only one top level field."
	}
	return fmt.Sprintf(`Subscription "%s" must select only one top level field.`, name)
}

// UndefinedFieldMessage returns message describing error occurred in rule "Field Selections on
// Objects, Interfaces, and Unions Types" (rules.FieldsOnCorrectType).
func UndefinedFieldMessage(
	fieldName string,
	parentTypeName string,
	suggestedTypeNames []string,
	suggestedFieldNames []string) string {

	var message util.StringBuilder
	message.WriteString(`Cannot query field "`)
	message.WriteString(fieldName)
	message.WriteString(`" on type "`)
	message.WriteString(parentTypeName)
	message.WriteString(`".`)

	if len(suggestedTypeNames) > 0 {
		message.WriteString(` Did you mean to use an inline fragment on `)
		util.OrList(&message, suggestedTypeNames, 5, true /*quoted*/)
		message.WriteString(`?`)
	} else if len(suggestedFieldNames) > 0 {
		message.WriteString(` Did you mean `)
		util.OrList(&message, suggestedFieldNames, 5, true /*quoted*/)
		message.WriteString(`?`)
	}

	return message.String()
}

// FieldConflictReason contains diagnostic message telling why two fields are conflict.
type FieldConflictReason struct {
	ResponseKey string

	// Reason could only be:
	//
	//  - A string, or
	//  - A nested list of conflicts (i.e., []*FieldConflictReason).
	MessageOrSubFieldReasons interface{}
}

// FieldsConflictMessage returns message describing error occurred in rule "Field Selection Merging"
// (rules.OverlappingFieldsCanBeMerged)
func FieldsConflictMessage(reason *FieldConflictReason) string {
	var message util.StringBuilder

	message.WriteString(`Fields "`)
	message.WriteString(reason.ResponseKey)
	message.WriteString(`" conflict because `)
	subReasonMessage(&message, reason.MessageOrSubFieldReasons)
	message.WriteString(`. Use different aliases on the fields to fetch both if this was intentional.`)

	return message.String()
}

func subReasonMessage(builder *util.StringBuilder, subReasons interface{}) {
	switch reason := subReasons.(type) {
	case string:
		builder.WriteString(reason)

	case []*FieldConflictReason:
		for i, r := range reason {
			if i != 0 {
				builder.WriteString(" and ")
			}
			builder.WriteString(`subfields "`)
			builder.WriteString(r.ResponseKey)
			builder.WriteString(`" conflict because `)
			subReasonMessage(builder, r.MessageOrSubFieldReasons)
		}
	}
}
