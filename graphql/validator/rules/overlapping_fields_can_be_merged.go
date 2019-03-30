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
	"fmt"
	"reflect"

	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/graphql/ast"
	internal "github.com/botobag/artemis/graphql/internal/validator"
	messages "github.com/botobag/artemis/graphql/internal/validator"
	"github.com/botobag/artemis/graphql/validator"
	"github.com/botobag/artemis/internal/util"
)

// OverlappingFieldsCanBeMerged implements the "Field Selection Merging" validation rule.
//
// See https://graphql.github.io/graphql-spec/June2018/#sec-Field-Selection-Merging.
type OverlappingFieldsCanBeMerged struct{}

type fieldConflict struct {
	Reason  messages.FieldConflictReason
	Fields1 []*ast.Field
	Fields2 []*ast.Field
}

// CheckSelectionSet implements validator.SelectionSetRule.
func (rule OverlappingFieldsCanBeMerged) CheckSelectionSet(
	ctx *validator.ValidationContext,
	ttype graphql.Type,
	selectionSet ast.SelectionSet) validator.NextCheckAction {

	// A selection set is only valid if all fields (including spreading any fragments) either
	// correspond to distinct response names or can be merged without ambiguity.

	conflicts := findConflictsWithinSelectionSet(ctx, ttype, selectionSet)
	for _, conflict := range conflicts {
		locations := make([]graphql.ErrorLocation, 0, len(conflict.Fields1)+len(conflict.Fields2))
		for _, field := range conflict.Fields1 {
			locations = append(locations, graphql.ErrorLocationOfASTNode(field))
		}
		for _, field := range conflict.Fields2 {
			locations = append(locations, graphql.ErrorLocationOfASTNode(field))
		}

		ctx.ReportError(
			messages.FieldsConflictMessage(&conflict.Reason),
			locations,
		)
	}

	return validator.ContinueCheck
}

/**
 * Algorithm:
 *
 * Conflicts occur when two fields exist in a query which will produce the same response name, but
 * represent differing values, thus creating a conflict.  The algorithm below finds all conflicts
 * via making a series of comparisons between fields. In order to compare as few fields as possible,
 * this makes a series of comparisons "within" sets of fields and "between" sets of fields.
 *
 * Given any selection set, a collection produces both a set of fields by also including all inline
 * fragments, as well as a list of fragments referenced by fragment spreads.
 *
 * A) Each selection set represented in the document first compares "within" its collected set of
 * fields, finding any conflicts between every pair of overlapping fields.
 * Note: This is the *only time* that a the fields "within" a set are compared to each other. After
 * this only fields "between" sets are compared.
 *
 * B) Also, if any fragment is referenced in a selection set, then a comparison is made "between"
 * the original set of fields and the referenced fragment.
 *
 * C) Also, if multiple fragments are referenced, then comparisons are made "between" each
 * referenced fragment.
 *
 * D) When comparing "between" a set of fields and a referenced fragment, first a comparison is made
 * between each field in the original set of fields and each field in the the referenced set of
 * fields.
 *
 * E) Also, if any fragment is referenced in the referenced selection set, then a comparison is made
 * "between" the original set of fields and the referenced fragment (recursively referring to step
 * D).
 *
 * F) When comparing "between" two fragments, first a comparison is made between each field in the
 * first referenced set of fields and each field in the the second referenced set of fields.
 *
 * G) Also, any fragments referenced by the first must be compared to the second, and any fragments
 * referenced by the second must be compared to the first (recursively referring to step F).
 *
 * H) When comparing two fields, if both have selection sets, then a comparison is made "between"
 * both selection sets, first comparing the set of fields in the first selection set with the set of
 * fields in the second.
 *
 * I) Also, if any fragment is referenced in either selection set, then a comparison is made
 * "between" the other set of fields and the referenced fragment.
 *
 * J) Also, if two fragments are referenced in both selection sets, then a comparison is made
 * "between" the two fragments.
 *
 */

// Find all conflicts found "within" a selection set, including those found via spreading in
// fragments. Called when visiting each SelectionSet in the GraphQL Document.
func findConflictsWithinSelectionSet(
	ctx *validator.ValidationContext,
	parentType graphql.Type,
	selectionSet ast.SelectionSet) []*fieldConflict {

	var (
		cachedFieldsAndFragmentNames = ctx.FieldsAndFragmentNamesCache
		comparedFragmentPairs        = ctx.FragmentPairSet
	)

	fieldsAndFragmentNames := internal.CollectFieldsAndFragmentNamesInSelectionSet(
		ctx.Schema(),
		cachedFieldsAndFragmentNames,
		parentType,
		selectionSet,
	)

	fieldMap, fragmentNames := fieldsAndFragmentNames.Fields, fieldsAndFragmentNames.FragmentNames

	// (A) Find find all conflicts "within" the fields of this selection set.
	// Note: this is the *only place* `collectConflictsWithin` is called.
	result := collectConflictsWithin(ctx, cachedFieldsAndFragmentNames, comparedFragmentPairs, fieldMap)

	if len(fragmentNames) > 0 {
		// (B) Then collect conflicts between these fields and those represented by each spread fragment
		// name found.
		comparedFragments := map[string]bool{}
		for i, fragmentName := range fragmentNames {
			conflicts := collectConflictsBetweenFieldsAndFragment(
				ctx,
				cachedFieldsAndFragmentNames,
				comparedFragments,
				comparedFragmentPairs,
				false, /* areMutuallyExclusive */
				fieldMap,
				fragmentName,
			)
			result = append(result, conflicts...)

			// (C) Then compare this fragment with all other fragments found in this selection set to
			// collect conflicts between fragments spread together.  This compares each item in the list
			// of fragment names to every other item in that same list (except for itself).
			for _, otherFragmentName := range fragmentNames[i+1:] {
				conflicts := collectConflictsBetweenFragments(
					ctx,
					cachedFieldsAndFragmentNames,
					comparedFragmentPairs,
					false, /* areMutuallyExclusive */
					fragmentName,
					otherFragmentName,
				)
				result = append(result, conflicts...)
			}
		}
	}

	return result
}

// Collect all Conflicts "within" one collection of fields.
func collectConflictsWithin(
	ctx *validator.ValidationContext,
	cachedFieldsAndFragmentNames internal.FieldsAndFragmentNamesCache,
	comparedFragmentPairs internal.ConflictFragmentPairSet,
	fieldMap internal.FieldNodeAndDefMap) []*fieldConflict {

	// A field map is a keyed collection, where each key represents a response name and the value at
	// that key is a list of all fields which provide that response name. For every response name, if
	// there are multiple fields, they must be compared to find a potential conflict.
	var conflicts []*fieldConflict

	for responseKey, fields := range fieldMap {
		// This compares every field in the list to every other field in this list (except to itself).
		// If the list only has one item, nothing needs to be compared.
		if len(fields) > 1 {
			for i, field := range fields {
				for _, otherField := range fields[i+1:] {
					conflict := findConflict(
						ctx,
						cachedFieldsAndFragmentNames,
						comparedFragmentPairs,
						// within one collection is never mutually exclusive
						false, /* parentFieldsAreMutuallyExclusive */
						responseKey,
						field,
						otherField,
					)
					if conflict != nil {
						conflicts = append(conflicts, conflict)
					}
				}
			}
		}
	}

	return conflicts
}

// Determines if there is a conflict between two particular fields, including comparing their
// sub-fields.
func findConflict(
	ctx *validator.ValidationContext,
	cachedFieldsAndFragmentNames internal.FieldsAndFragmentNamesCache,
	comparedFragmentPairs internal.ConflictFragmentPairSet,
	parentFieldsAreMutuallyExclusive bool,
	responseKey string,
	field1 *internal.FieldNodeAndDef,
	field2 *internal.FieldNodeAndDef) *fieldConflict {

	var (
		parentType1 = field1.ParentType
		node1       = field1.Node
		def1        = field1.Def

		parentType2 = field2.ParentType
		node2       = field2.Node
		def2        = field2.Def
	)

	// If it is known that two fields could not possibly apply at the same time, due to the parent
	// types, then it is safe to permit them to diverge in aliased field or arguments used as they
	// will not present any ambiguity by differing.
	//
	// It is known that two parent types could never overlap if they are different Object types.
	// Interface or Union types might overlap - if not in the current state of the schema, then
	// perhaps in some future version, thus may not safely diverge.
	areMutuallyExclusive :=
		parentFieldsAreMutuallyExclusive ||
			(parentType1 != parentType2 &&
				graphql.IsObjectType(parentType1) &&
				graphql.IsObjectType(parentType2))

	if !areMutuallyExclusive {
		// Two aliases must refer to the same field.
		var (
			name1 = node1.Name.Value()
			name2 = node2.Name.Value()
		)
		if name1 != name2 {
			return &fieldConflict{
				Reason: messages.FieldConflictReason{
					ResponseKey:              responseKey,
					MessageOrSubFieldReasons: fmt.Sprintf("%s and %s are different fields", name1, name2),
				},
				Fields1: []*ast.Field{node1},
				Fields2: []*ast.Field{node2},
			}
		}

		// Two field calls must have the same arguments.
		if !sameArguments(node1.Arguments, node2.Arguments) {
			return &fieldConflict{
				Reason: messages.FieldConflictReason{
					ResponseKey:              responseKey,
					MessageOrSubFieldReasons: "they have differing arguments",
				},
				Fields1: []*ast.Field{node1},
				Fields2: []*ast.Field{node2},
			}
		}
	}

	// The return type for each field.
	var (
		type1 graphql.Type
		type2 graphql.Type
	)
	if def1 != nil {
		type1 = def1.Type()
	}
	if def2 != nil {
		type2 = def2.Type()
	}

	if type1 != nil && type2 != nil && doTypesConflict(type1, type2) {
		var reason util.StringBuilder
		reason.WriteString("they return conflicting types ")
		graphql.InspectTo(&reason, type1)
		reason.WriteString(" and ")
		graphql.InspectTo(&reason, type2)

		return &fieldConflict{
			Reason: messages.FieldConflictReason{
				ResponseKey:              responseKey,
				MessageOrSubFieldReasons: reason.String(),
			},
			Fields1: []*ast.Field{node1},
			Fields2: []*ast.Field{node2},
		}
	}

	// Collect and compare sub-fields. Use the same "visited fragment names" list for both collections
	// so fields in a fragment reference are never compared to themselves.
	var (
		selectionSet1 = node1.SelectionSet
		selectionSet2 = node2.SelectionSet
	)
	if len(selectionSet1) > 0 && len(selectionSet2) > 0 {
		conflicts := findConflictsBetweenSubSelectionSets(
			ctx,
			cachedFieldsAndFragmentNames,
			comparedFragmentPairs,
			areMutuallyExclusive,
			graphql.NamedTypeOf(type1),
			selectionSet1,
			graphql.NamedTypeOf(type2),
			selectionSet2,
		)
		return subfieldConflicts(conflicts, responseKey, node1, node2)
	}

	return nil
}

// Find all conflicts found between two selection sets, including those found via spreading in
// fragments. Called when determining if conflicts exist between the sub-fields of two overlapping
// fields.
func findConflictsBetweenSubSelectionSets(
	ctx *validator.ValidationContext,
	cachedFieldsAndFragmentNames internal.FieldsAndFragmentNamesCache,
	comparedFragmentPairs internal.ConflictFragmentPairSet,
	areMutuallyExclusive bool,
	parentType1 graphql.Type,
	selectionSet1 ast.SelectionSet,
	parentType2 graphql.Type,
	selectionSet2 ast.SelectionSet) []*fieldConflict {

	fieldsAndFragmentNames1 := internal.CollectFieldsAndFragmentNamesInSelectionSet(
		ctx.Schema(),
		cachedFieldsAndFragmentNames,
		parentType1,
		selectionSet1,
	)

	fieldsAndFragmentNames2 := internal.CollectFieldsAndFragmentNamesInSelectionSet(
		ctx.Schema(),
		cachedFieldsAndFragmentNames,
		parentType2,
		selectionSet2,
	)

	var (
		fieldMap1 = fieldsAndFragmentNames1.Fields
		fieldMap2 = fieldsAndFragmentNames2.Fields

		fragmentNames1 = fieldsAndFragmentNames1.FragmentNames
		fragmentNames2 = fieldsAndFragmentNames2.FragmentNames
	)

	// (H) First, collect all conflicts between these two collections of field.
	result := collectConflictsBetween(
		ctx,
		cachedFieldsAndFragmentNames,
		comparedFragmentPairs,
		areMutuallyExclusive,
		fieldMap1,
		fieldMap2,
	)

	// (I) Then collect conflicts between the first collection of fields and those referenced by each
	// fragment name associated with the second.
	if len(fragmentNames2) > 0 {
		comparedFragments := map[string]bool{}
		for _, fragmentName := range fragmentNames2 {
			conflicts := collectConflictsBetweenFieldsAndFragment(
				ctx,
				cachedFieldsAndFragmentNames,
				comparedFragments,
				comparedFragmentPairs,
				areMutuallyExclusive,
				fieldMap1,
				fragmentName,
			)
			result = append(result, conflicts...)
		}
	}

	// (I) Then collect conflicts between the second collection of fields and those referenced by each
	// fragment name associated with the first.
	if len(fragmentNames1) > 0 {
		comparedFragments := map[string]bool{}
		for _, fragmentName := range fragmentNames1 {
			conflicts := collectConflictsBetweenFieldsAndFragment(
				ctx,
				cachedFieldsAndFragmentNames,
				comparedFragments,
				comparedFragmentPairs,
				areMutuallyExclusive,
				fieldMap2,
				fragmentName,
			)
			result = append(result, conflicts...)
		}
	}

	// (J) Also collect conflicts between any fragment names by the first and fragment names by the
	// second. This compares each item in the first set of names to each item in the second set of
	// names.
	for _, fragmentName1 := range fragmentNames1 {
		for _, fragmentName2 := range fragmentNames2 {
			conflicts := collectConflictsBetweenFragments(
				ctx,
				cachedFieldsAndFragmentNames,
				comparedFragmentPairs,
				areMutuallyExclusive,
				fragmentName1,
				fragmentName2,
			)
			result = append(result, conflicts...)
		}
	}

	return result
}

// Given a series of Conflicts which occurred between two sub-fields, generate a single Conflict.
func subfieldConflicts(
	conflicts []*fieldConflict,
	responseKey string,
	node1 *ast.Field,
	node2 *ast.Field,
) *fieldConflict {
	if len(conflicts) == 0 {
		return nil
	}

	conflict := &fieldConflict{
		Reason: messages.FieldConflictReason{
			ResponseKey: responseKey,
		},
		Fields1: []*ast.Field{node1},
		Fields2: []*ast.Field{node2},
	}

	subFieldReasons := make([]*messages.FieldConflictReason, len(conflicts))
	for i, c := range conflicts {
		subFieldReasons[i] = &c.Reason
		conflict.Fields1 = append(conflict.Fields1, c.Fields1...)
		conflict.Fields2 = append(conflict.Fields2, c.Fields2...)
	}
	conflict.Reason.MessageOrSubFieldReasons = subFieldReasons

	return conflict
}

func sameArguments(arguments1 ast.Arguments, arguments2 ast.Arguments) bool {
	if len(arguments1) != len(arguments2) {
		return false
	}

	for _, argument1 := range arguments1 {
		name1 := argument1.Name.Value()

		var argument2 *ast.Argument
		for i := range arguments2 {
			if arguments2[i].Name.Value() == name1 {
				argument2 = arguments2[i]
				break
			}
		}

		if argument2 == nil {
			return false
		}

		if !sameValue(argument1.Value, argument2.Value) {
			return false
		}
	}

	return true
}

func sameValue(value1, value2 ast.Value) bool {
	return reflect.TypeOf(value1) == reflect.TypeOf(value2) &&
		reflect.DeepEqual(value1.Interface(), value2.Interface())
}

// Two types conflict if both types could not apply to a value simultaneously.  Composite types are
// ignored as their individual field types will be compared later recursively. However List and
// Non-Null types must match.
func doTypesConflict(type1 graphql.Type, type2 graphql.Type) bool {
	switch type1 := type1.(type) {
	case graphql.List:
		if type2, ok := type2.(graphql.List); ok {
			return doTypesConflict(type1.ElementType(), type2.ElementType())
		}
		return true

	case graphql.NonNull:
		if type2, ok := type2.(graphql.NonNull); ok {
			return doTypesConflict(type1.InnerType(), type2.InnerType())
		}
		return true

	default:
		// type1 is not be wrapping type (List or NonNull) here. If type2 is a wrapping type, they're
		// conflict.
		if graphql.IsWrappingType(type2) {
			return true
		}

		if graphql.IsLeafType(type1) || graphql.IsLeafType(type2) {
			return type1 != type2
		}

		return false
	}
}

func collectConflictsBetweenFieldsAndFragment(
	ctx *validator.ValidationContext,
	cachedFieldsAndFragmentNames internal.FieldsAndFragmentNamesCache,
	comparedFragments map[string]bool,
	comparedFragmentPairs internal.ConflictFragmentPairSet,
	areMutuallyExclusive bool,
	fieldMap internal.FieldNodeAndDefMap,
	fragmentName string,
) []*fieldConflict {
	// Memoize so a fragment is not compared for conflicts more than once.
	if _, compared := comparedFragments[fragmentName]; compared {
		return nil
	}
	comparedFragments[fragmentName] = true

	fragment := ctx.Fragment(fragmentName)
	if fragment == nil {
		return nil
	}

	fieldsAndFragmentNames2 := internal.CollectFieldsAndFragmentNamesInFragmentDefinition(
		ctx.Schema(),
		cachedFieldsAndFragmentNames,
		fragment,
	)

	fieldMap2, fragmentNames2 := fieldsAndFragmentNames2.Fields, fieldsAndFragmentNames2.FragmentNames

	// Do not compare a fragment's fieldMap to itself.
	if reflect.DeepEqual(fieldMap, fieldMap2) {
		return nil
	}

	// (D) First collect any conflicts between the provided collection of fields and the collection of
	// fields represented by the given fragment.
	result := collectConflictsBetween(
		ctx,
		cachedFieldsAndFragmentNames,
		comparedFragmentPairs,
		areMutuallyExclusive,
		fieldMap,
		fieldMap2,
	)

	// (E) Then collect any conflicts between the provided collection of fields and any fragment names
	// found in the given fragment.
	for _, fragmentName2 := range fragmentNames2 {
		conflicts := collectConflictsBetweenFieldsAndFragment(
			ctx,
			cachedFieldsAndFragmentNames,
			comparedFragments,
			comparedFragmentPairs,
			areMutuallyExclusive,
			fieldMap,
			fragmentName2,
		)
		result = append(result, conflicts...)
	}

	return result
}

// Collect all Conflicts between two collections of fields. This is similar to,
// but different from the `collectConflictsWithin` function above. This check
// assumes that `collectConflictsWithin` has already been called on each
// provided collection of fields. This is true because this validator traverses
// each individual selection set.
func collectConflictsBetween(
	ctx *validator.ValidationContext,
	cachedFieldsAndFragmentNames internal.FieldsAndFragmentNamesCache,
	comparedFragmentPairs internal.ConflictFragmentPairSet,
	parentFieldsAreMutuallyExclusive bool,
	fieldMap1 internal.FieldNodeAndDefMap,
	fieldMap2 internal.FieldNodeAndDefMap) []*fieldConflict {

	var conflicts []*fieldConflict

	// A field map is a keyed collection, where each key represents a response name and the value at
	// that key is a list of all fields which provide that response name. For any response name which
	// appears in both provided field maps, each field from the first field map must be compared to
	// every field in the second field map to find potential conflicts.
	for responseKey, fields1 := range fieldMap1 {
		fields2, exists := fieldMap2[responseKey]
		if !exists {
			continue
		}

		for _, field1 := range fields1 {
			for _, field2 := range fields2 {
				conflict := findConflict(
					ctx,
					cachedFieldsAndFragmentNames,
					comparedFragmentPairs,
					parentFieldsAreMutuallyExclusive,
					responseKey,
					field1,
					field2,
				)
				if conflict != nil {
					conflicts = append(conflicts, conflict)
				}
			}
		}
	}

	return conflicts
}

// Collect all conflicts found between two fragments, including via spreading in
// any nested fragments.
func collectConflictsBetweenFragments(
	ctx *validator.ValidationContext,
	cachedFieldsAndFragmentNames internal.FieldsAndFragmentNamesCache,
	comparedFragmentPairs internal.ConflictFragmentPairSet,
	areMutuallyExclusive bool,
	fragmentName1 string,
	fragmentName2 string,
) []*fieldConflict {

	// No need to compare a fragment to itself.
	if fragmentName1 == fragmentName2 {
		return nil
	}

	// Memoize so two fragments are not compared for conflicts more than once.
	if comparedFragmentPairs.Has(fragmentName1, fragmentName2, areMutuallyExclusive) {
		return nil
	}
	comparedFragmentPairs.Add(fragmentName1, fragmentName2, areMutuallyExclusive)

	var (
		fragment1 = ctx.Fragment(fragmentName1)
		fragment2 = ctx.Fragment(fragmentName2)
	)
	if fragment1 == nil || fragment2 == nil {
		return nil
	}

	fieldsAndFragmentNames1 := internal.CollectFieldsAndFragmentNamesInFragmentDefinition(
		ctx.Schema(),
		cachedFieldsAndFragmentNames,
		fragment1,
	)

	fieldsAndFragmentNames2 := internal.CollectFieldsAndFragmentNamesInFragmentDefinition(
		ctx.Schema(),
		cachedFieldsAndFragmentNames,
		fragment2,
	)

	var (
		fieldMap1 = fieldsAndFragmentNames1.Fields
		fieldMap2 = fieldsAndFragmentNames2.Fields

		fragmentNames1 = fieldsAndFragmentNames1.FragmentNames
		fragmentNames2 = fieldsAndFragmentNames2.FragmentNames
	)

	// (F) First, collect all conflicts between these two collections of fields (not including any
	// nested fragments).
	result := collectConflictsBetween(
		ctx,
		cachedFieldsAndFragmentNames,
		comparedFragmentPairs,
		areMutuallyExclusive,
		fieldMap1,
		fieldMap2,
	)

	// (G) Then collect conflicts between the first fragment and any nested fragments spread in the
	// second fragment.
	for _, fragmentName := range fragmentNames2 {
		conflicts := collectConflictsBetweenFragments(
			ctx,
			cachedFieldsAndFragmentNames,
			comparedFragmentPairs,
			areMutuallyExclusive,
			fragmentName1,
			fragmentName,
		)
		result = append(result, conflicts...)
	}

	for _, fragmentName := range fragmentNames1 {
		conflicts := collectConflictsBetweenFragments(
			ctx,
			cachedFieldsAndFragmentNames,
			comparedFragmentPairs,
			areMutuallyExclusive,
			fragmentName,
			fragmentName2,
		)
		result = append(result, conflicts...)
	}

	return result
}
