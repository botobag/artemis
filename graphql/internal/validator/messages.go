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

	"github.com/botobag/artemis/graphql"
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
// (rules.OverlappingFieldsCanBeMerged).
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

// NoSubselectionAllowedMessage returns message describing error occurred in rule "Leaf Field
// Selections" (rules.ScalarLeafs)
func NoSubselectionAllowedMessage(fieldName string, typeName string) string {
	return fmt.Sprintf(`Field "%s" must not have a selection since type "%s" has no subfields.`,
		fieldName, typeName)
}

// RequiredSubselectionMessage returns message describing error occurred in rule "Leaf Field
// Selections" (rules.ScalarLeafs)
func RequiredSubselectionMessage(fieldName string, typeName string) string {
	return fmt.Sprintf(`Field "%s" of type "%s" must have a selection of subfields. Did you mean "%s { ... }"?`,
		fieldName, typeName, fieldName)
}

// UnknownArgMessage returns message describing error occurred in rule "Argument Names"
// (rules.KnownArgumentNames)
func UnknownArgMessage(argName string, fieldName string, typeName string, suggestedArgs []string) string {
	var message util.StringBuilder
	message.WriteString(`Unknown argument "`)
	message.WriteString(argName)
	message.WriteString(`" on field "`)
	message.WriteString(fieldName)
	message.WriteString(`" of type "`)
	message.WriteString(typeName)
	message.WriteString(`".`)

	if len(suggestedArgs) > 0 {
		message.WriteString(` Did you mean `)
		util.OrList(&message, suggestedArgs, 5, true /*quoted*/)
		message.WriteString(`?`)
	}

	return message.String()
}

// UnknownDirectiveArgMessage returns message describing error occurred in rule "Argument Names"
// (rules.KnownArgumentNamesOnDirectives)
func UnknownDirectiveArgMessage(argName string, directiveName string, suggestedArgs []string) string {
	var message util.StringBuilder
	message.WriteString(`Unknown argument "`)
	message.WriteString(argName)
	message.WriteString(`" on directive @"`)
	message.WriteString(directiveName)
	message.WriteString(`".`)

	if len(suggestedArgs) > 0 {
		message.WriteString(` Did you mean `)
		util.OrList(&message, suggestedArgs, 5, true /*quoted*/)
		message.WriteString(`?`)
	}

	return message.String()
}

// DuplicateArgMessage returns message describing error occurred in rule "Argument Uniqueness"
// (rules.UniqueArgumentNames)
func DuplicateArgMessage(argName string) string {
	return fmt.Sprintf(`There can be only one argument named "%s".`, argName)
}

// MissingFieldArgMessage returns message describing error occurred in rule "Required Arguments"
// (rules.ProvidedRequiredArguments)
func MissingFieldArgMessage(argName string, fieldName string, typeName string) string {
	return fmt.Sprintf(`Field "%s" argument "%s" of type "%s" is required, but it was not provided.`,
		fieldName, argName, typeName)
}

// MissingDirectiveArgMessage returns message describing error occurred in rule "Required Arguments"
// (rules.ProvidedRequiredArgumentsOnDirectives)
func MissingDirectiveArgMessage(argName string, directiveName string, typeName string) string {
	return fmt.Sprintf(`Directive "@%s" argument "%s" of type "%s" is required, but it was not provided.`,
		directiveName, argName, typeName)
}

// DuplicateFragmentNameMessage returns message describing error occurred in rule "Fragment Name
// Uniqueness" (rules.UniqueFragmentNames).
func DuplicateFragmentNameMessage(fragmentName string) string {
	return fmt.Sprintf(`There can be only one fragment named "%s".`, fragmentName)
}

// UnknownTypeMessage returns message describing error occurred in rule
// "Fragment Spread Type Existence" (rules.KnownTypeNames).
func UnknownTypeMessage(typeName string, suggestedTypes []string) string {
	var message util.StringBuilder
	message.WriteString(`Unknown type "`)
	message.WriteString(typeName)
	message.WriteString(`".`)

	if len(suggestedTypes) > 0 {
		message.WriteString(` Did you mean `)
		util.OrList(&message, suggestedTypes, 5, true /*quoted*/)
		message.WriteString(`?`)
	}

	return message.String()
}

// FragmentOnNonCompositeErrorMessage returns message describing error occurred in rule "Fragments
// on Composite Types" (rules.FragmentsOnCompositeTypes).
func FragmentOnNonCompositeErrorMessage(fragmentName string, typeCondition string) string {
	return fmt.Sprintf(`Fragment "%s" cannot condition on non composite type "%s".`,
		fragmentName, typeCondition)
}

// InlineFragmentOnNonCompositeErrorMessage returns message describing error occurred in rule
// "Fragments on Composite Types" (rules.FragmentsOnCompositeTypes).
func InlineFragmentOnNonCompositeErrorMessage(typeCondition string) string {
	return fmt.Sprintf(`Fragment cannot condition on non composite type "%s".`, typeCondition)
}

// UnusedFragMessage returns message describing error occurred in rule "Fragments must be used"
// (rules.NoUnusedFragments).
func UnusedFragMessage(fragName string) string {
	return fmt.Sprintf(`Fragment "%s" is never used.`, fragName)
}

// UnknownFragmentMessage returns message describing error occurred in rule "Fragment spread target
// defined" (rules.KnownFragmentNames).
func UnknownFragmentMessage(fragName string) string {
	return fmt.Sprintf(`Unknown fragment "%s".`, fragName)
}

// CycleErrorMessage returns message describing error occurred in rule "Fragments must not form
// cycles" (rules.NoFragmentCycles).
func CycleErrorMessage(fragName string, spreadNames []string) string {
	var message util.StringBuilder

	message.WriteString(`Cannot spread fragment "`)
	message.WriteString(fragName)
	message.WriteString(`" within itself`)

	if len(spreadNames) > 0 {
		message.WriteString(` via `)
		message.WriteString(spreadNames[0])
		for _, name := range spreadNames[1:] {
			message.WriteString(", ")
			message.WriteString(name)
		}
	}
	message.WriteString(`.`)

	return message.String()
}

// TypeIncompatibleAnonSpreadMessage returns message describing error occurred in rule "Fragment
// spread is possible" (rules.PossibleFragmentSpreads).
func TypeIncompatibleAnonSpreadMessage(parentType string, fragType string) string {
	return fmt.Sprintf(`Fragment cannot be spread here as objects of type "%s" can never be of type "%s"`,
		parentType, fragType)
}

// TypeIncompatibleSpreadMessage returns message describing error occurred in rule "Fragment spread
// is possible" (rules.PossibleFragmentSpreads).
func TypeIncompatibleSpreadMessage(fragName string, parentType string, fragType string) string {
	return fmt.Sprintf(`Fragment "%s" cannot be spread here as objects of type "%s" can never be of type "%s".`,
		fragName, parentType, fragType)
}

// BadValueMessage returns message describing error occurred in rule "Value Type Correctness"
// (rules.ValuesOfCorrectType).
func BadValueMessage(typeName string, valueName string, suggestedNames []string) string {
	var message util.StringBuilder

	message.WriteString("Expected type ")
	message.WriteString(typeName)
	message.WriteString(", found ")
	message.WriteString(valueName)

	if len(suggestedNames) == 0 {
		message.WriteRune('.')
	} else {
		message.WriteString("; Did you mean the enum value ")
		util.OrList(&message, suggestedNames, 5, false /*quoted*/)
		message.WriteRune('?')
	}

	return message.String()
}

// BadScalarValueMessage returns message describing error occurred in rule "Value Type Correctness"
// (rules.ValuesOfCorrectType).
func BadScalarValueMessage(typeName string, valueName string, errMessage string) string {
	return fmt.Sprintf("Expected type %s, found %s; %s", typeName, valueName, errMessage)
}

// RequiredFieldMessage returns message describing error occurred in rule "Value Type Correctness"
// (rules.ValuesOfCorrectType).
func RequiredFieldMessage(typeName string, fieldName string, fieldTypeName string) string {
	return fmt.Sprintf("Field %s.%s of required type %s was not provided.",
		typeName, fieldName, fieldTypeName)
}

// UnknownFieldMessage returns message describing error occurred in rule "Value Type Correctness"
// (rules.ValuesOfCorrectType).
func UnknownFieldMessage(typeName string, fieldName string, suggestedFields []string) string {
	var message util.StringBuilder

	message.WriteString(`Field "`)
	message.WriteString(fieldName)
	message.WriteString(`" is not defined by type `)
	message.WriteString(typeName)

	if len(suggestedFields) == 0 {
		message.WriteRune('.')
	} else {
		message.WriteString("; Did you mean ")
		util.OrList(&message, suggestedFields, 5, false /*quoted*/)
		message.WriteRune('?')
	}

	return message.String()
}

// DuplicateInputFieldMessage returns message describing error occurred in rule "Input Object Field
// Uniqueness" (rules.UniqueInputFieldNames).
func DuplicateInputFieldMessage(fieldName string) string {
	return fmt.Sprintf(`There can be only one input field named "%s".`, fieldName)
}

// UnknownDirectiveMessage returns message describing error occurred in rule "Directives Are
// Defined" (rules.KnownDirectives).
func UnknownDirectiveMessage(directiveName string) string {
	return fmt.Sprintf(`Unknown directive "%s".`, directiveName)
}

// MisplacedDirectiveMessage returns message describing error occurred in rule "Directives Are In
// Valid Locations" (rules.DirectivesInValidLocations).
func MisplacedDirectiveMessage(directiveName string, location graphql.DirectiveLocation) string {
	return fmt.Sprintf(`Directive "%s" may not be used on %s.`, directiveName, location)
}

// DuplicateDirectiveMessage returns message describing error occurred in rule "Directives Are
// Unique Per Location" (rules.KnownDirectives).
func DuplicateDirectiveMessage(directiveName string) string {
	return fmt.Sprintf(`The directive "%s" can only be used once at this location.`, directiveName)
}

// DuplicateVariableMessage returns message describing error occurred in rule "Variable Uniqueness"
// (rules.UniqueVariableNames).
func DuplicateVariableMessage(variableName string) string {
	return fmt.Sprintf(`There can be only one variable named "%s".`, variableName)
}

// NonInputTypeOnVarMessage returns message describing error occurred in rule "Variables Are Input
// Types" (rules.VariablesAreInputTypes).
func NonInputTypeOnVarMessage(variableName string, typeName string) string {
	return fmt.Sprintf(`Variable "$%s" cannot be non-input type "%s".`, variableName, typeName)
}

// UndefinedVarMessage returns message describing error occurred in rule "All Variable Uses Defined"
// (rules.NoUndefinedVariables).
func UndefinedVarMessage(variableName string, operationName string) string {
	if len(operationName) == 0 {
		return fmt.Sprintf(`Variable "$%s" is not defined.`, variableName)
	}
	return fmt.Sprintf(`Variable "$%s" is not defined by operation "%s".`, variableName, operationName)
}

// UnusedVariableMessage returns message describing error occurred in rule "No Unused Variables"
// (rules.NoUnusedVariables).
func UnusedVariableMessage(variableName string, operationName string) string {
	if len(operationName) == 0 {
		return fmt.Sprintf(`Variable "$%s" is never used.`, variableName)
	}
	return fmt.Sprintf(`Variable "$%s" is never used in operation "%s".`, variableName, operationName)
}

// BadVarPosMessage returns message describing error occurred in rule "All Variable Usages Are
// Allowed" (rules.VariablesInAllowedPosition).
func BadVarPosMessage(variableName string, variableType string, expectedType string) string {
	return fmt.Sprintf(`Variable "$%s" of type "%s" used in position expecting type "%s".`,
		variableName, variableType, expectedType)
}
