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

// PossibleFragmentSpreads implements the "Fragment spread is possible" validation rule.
//
// See https://graphql.github.io/graphql-spec/June2018/#sec-Fragment-spread-is-possible.
type PossibleFragmentSpreads struct{}

// A fragment spread is only valid if the type condition could ever possibly be true: if there is a
// non-empty intersection of the possible parent types, and possible types which pass the type
// condition.

// CheckInlineFragment implements validator.InlineFragmentRule.
func (rule PossibleFragmentSpreads) CheckInlineFragment(
	ctx *validator.ValidationContext,
	parentType graphql.Type,
	typeCondition graphql.Type,
	fragment *ast.InlineFragment) validator.NextCheckAction {
	if graphql.IsCompositeType(parentType) &&
		// IsCompositeType returns false for nil Type.
		graphql.IsCompositeType(typeCondition) &&
		!rule.doTypesOverlap(ctx.Schema(), typeCondition, parentType) {
		ctx.ReportError(
			messages.TypeIncompatibleAnonSpreadMessage(
				graphql.Inspect(parentType),
				graphql.Inspect(typeCondition),
			),
			graphql.ErrorLocationOfASTNode(fragment),
		)
	}
	return validator.ContinueCheck
}

// CheckFragmentSpread implements validator.FragmentSpreadRule.
func (rule PossibleFragmentSpreads) CheckFragmentSpread(
	ctx *validator.ValidationContext,
	parentType graphql.Type,
	fragmentInfo *validator.FragmentInfo,
	fragmentSpread *ast.FragmentSpread) validator.NextCheckAction {

	fragType := fragmentInfo.TypeCondition()
	if parentType != nil &&
		graphql.IsCompositeType(fragType) &&
		!rule.doTypesOverlap(ctx.Schema(), fragType, parentType) {
		ctx.ReportError(
			messages.TypeIncompatibleSpreadMessage(
				fragmentSpread.Name.Value(),
				graphql.Inspect(parentType),
				graphql.Inspect(fragType),
			),
			graphql.ErrorLocationOfASTNode(fragmentSpread),
		)
	}
	return validator.ContinueCheck
}

// Provided two composite types, determine if they "overlap". Two composite types overlap when the
// Sets of possible concrete types for each intersect.
//
// This is used to determine if a fragment of a given type could possibly be visited in a context of
// another type.
//
// This function is commutative.
//
// Both typeA and typeB must be composite types (Object, Interface or Union).
func (rule PossibleFragmentSpreads) doTypesOverlap(
	schema graphql.Schema,
	typeA graphql.Type,
	typeB graphql.Type) bool {
	// Equivalent types overlap
	if typeA == typeB {
		return true
	}

	if typeA, ok := typeA.(graphql.AbstractType); ok {
		possibleTypesA := schema.PossibleTypes(typeA)
		if typeB, ok := typeB.(graphql.AbstractType); ok {
			// If both types are abstract, then determine if there is any intersection
			// between possible concrete types of each.
			return possibleTypesA.DoesIntersect(schema.PossibleTypes(typeB))
		}

		// Determine if the latter type is a possible concrete type of the former.
		return schema.PossibleTypes(typeA).Contains(typeB.(graphql.Object))
	}

	if typeB, ok := typeB.(graphql.AbstractType); ok {
		// Determine if the former type is a possible concrete type of the latter.
		return schema.PossibleTypes(typeB).Contains(typeA.(graphql.Object))
	}

	// Otherwise the types do not overlap.
	return false
}
