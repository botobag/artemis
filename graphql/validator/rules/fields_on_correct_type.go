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
	"sort"

	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/graphql/ast"
	messages "github.com/botobag/artemis/graphql/internal/validator"
	"github.com/botobag/artemis/graphql/validator"
	"github.com/botobag/artemis/internal/util"
)

// FieldsOnCorrectType implements the "Lone Anonymous Operation" validation rule.
//
// See https://facebook.github.io/graphql/June2018/#sec-Field-Selections-on-Objects-Interfaces-and-Unions-Types.
type FieldsOnCorrectType struct{}

// CheckField implements validator.FieldRule.
func (rule FieldsOnCorrectType) CheckField(
	ctx *validator.ValidationContext,
	parentType graphql.Type,
	fieldDef graphql.Field,
	field *ast.Field) validator.NextCheckAction {

	// A GraphQL document is only valid if all fields selected are defined by the parent type, or are
	// an allowed meta field such as __typename.

	if parentType == nil {
		// If we're unable to resolve parent type statically, we cannot correctly reason the field type.
		// Skip the check.
		return validator.ContinueCheck
	}

	if fieldDef != nil {
		// Field looks good as its definition is found in schema.
		return validator.ContinueCheck
	}

	// If here, field definition was not found in parentType which results a validation error.

	// This field doesn't exist, lets look for suggestions.
	var (
		schema              = ctx.Schema()
		fieldName           = field.Name.Value()
		suggestedTypeNames  []string
		suggestedFieldNames []string
	)

	// First determine if there are any suggested types to condition on.
	suggestedTypeNames = getSuggestedTypeNames(schema, parentType, fieldName)

	// If there are no suggested types, then perhaps this was a typo?
	if len(suggestedTypeNames) == 0 {
		suggestedFieldNames = getSuggestedFieldNames(schema, parentType, fieldName)
	}

	// Report an error, including helpful suggestions.
	ctx.ReportError(
		messages.UndefinedFieldMessage(
			fieldName,
			graphql.Inspect(parentType),
			suggestedTypeNames,
			suggestedFieldNames,
		),
		graphql.ErrorLocationOfASTNode(field),
	)

	return validator.ContinueCheck
}

// getSuggestedTypeNames goes through all of the implementations of type, as well as the interfaces
// that they implement. If any of those types include the provided field, suggest them, sorted by
// how often the type is referenced, starting with Interfaces.
func getSuggestedTypeNames(schema graphql.Schema, parentType graphql.Type, fieldName string) []string {
	ttype, ok := parentType.(graphql.AbstractType)
	if !ok {
		// parentType must be an Object type, which does not have possible fields.
		return nil
	}

	var (
		possibleTypes           = schema.PossibleTypes(ttype)
		suggestedObjectTypes    []string
		suggestedInterfaceTypes []string
		interfaceUsageCount     = map[string]int{}
	)

	if possibleTypes.Empty() {
		return nil
	}

	possibleTypeIter := possibleTypes.Iterator()
	for {
		possibleType, err := possibleTypeIter.Next()
		if err != nil {
			break
		}

		suggestedObjectType := possibleType.(graphql.Object)

		if _, containsField := suggestedObjectType.Fields()[fieldName]; containsField {
			// This object type defines this field.
			suggestedObjectTypes = append(suggestedObjectTypes, suggestedObjectType.Name())

			for _, iface := range suggestedObjectType.Interfaces() {
				possibleInterface := iface.Name()
				if _, containsField := iface.Fields()[fieldName]; containsField {
					// This interface type defines this field.
					usageCount := interfaceUsageCount[possibleInterface]
					if usageCount == 0 {
						suggestedInterfaceTypes = append(suggestedInterfaceTypes, possibleInterface)
					}
					interfaceUsageCount[possibleInterface] = usageCount + 1
				}
			}
		}
	}

	sort.SliceStable(suggestedInterfaceTypes, func(i, j int) bool {
		return interfaceUsageCount[suggestedInterfaceTypes[i]] > interfaceUsageCount[suggestedInterfaceTypes[j]]
	})

	return append(suggestedInterfaceTypes, suggestedObjectTypes...)
}

// For the field name provided, determine if there are any similar field names that may be the
// result of a typo.
func getSuggestedFieldNames(schema graphql.Schema, parentType graphql.Type, fieldName string) []string {
	var fields graphql.FieldMap

	switch ttype := parentType.(type) {
	case graphql.Object:
		fields = ttype.Fields()

	case graphql.Interface:
		fields = ttype.Fields()

	default:
		// Otherwise, must be a Union type, which does not define fields.
		return nil
	}

	possibleFieldNames := make([]string, 0, len(fields))
	for name := range fields {
		possibleFieldNames = append(possibleFieldNames, name)
	}

	return util.SuggestionList(fieldName, possibleFieldNames)
}
