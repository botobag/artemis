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
	internal "github.com/botobag/artemis/graphql/internal/validator"
	astutil "github.com/botobag/artemis/graphql/util/ast"
)

// FragmentInfo stores information about a fragment definition during validation. It is specifically
// used as value type of the fragment map in ValidationContext and exposed to FragmentRule's and
// FragmentSpreadRule's.
type FragmentInfo struct {
	def *ast.FragmentDefinition

	typeCondition graphql.Type

	// Set to true if fragments that have been referenced (used) in any operation definitions (used by
	// NoUnusedFragments)
	used bool
}

// Definition returns info.def.
func (info *FragmentInfo) Definition() *ast.FragmentDefinition {
	return info.def
}

// TypeCondition returns info.typeCondition.
func (info *FragmentInfo) TypeCondition() graphql.Type {
	return info.typeCondition
}

// RecursivelyMarkUsed marks the fragment to be used and works with ctx to also mark the fragments
// that are directly and indirectly referenced by this fragment to be used.
func (info *FragmentInfo) RecursivelyMarkUsed(ctx *ValidationContext) {
	if info.used {
		// Return quiclkly if it is already in use.
		return
	}

	stack := []*FragmentInfo{info}
	for len(stack) > 0 {
		var fragment *FragmentInfo
		fragment, stack = stack[len(stack)-1], stack[:len(stack)-1]

		// Mark used bit.
		fragment.used = true

		// Scan selection set and check any fragment spreads.
		for _, selection := range fragment.def.SelectionSet {
			if fragmentSpread, ok := selection.(*ast.FragmentSpread); ok {
				// Look up ctx to find fragment expanded by the fragment spread.
				f := ctx.FragmentInfo(fragmentSpread.Name.Value())
				if f != nil && !f.used {
					// Put f to the stack.
					stack = append(stack, f)
				}
			}
		}
	}
}

// Used returns true if this fragment has been marked as used (via RecursivelyMarkUsed) or is
// depended by any other fragments that has been marked as used.
func (info *FragmentInfo) Used() bool {
	return info.used
}

// A ValidationContext stores various states for running walk function and validation rules.
type ValidationContext struct {
	schema   graphql.Schema
	document ast.Document
	rules    *rules

	// Mapping FragmentInfo's from their names; This is lazily computed on the first call to
	// FragmentInfo.
	fragmentInfos map[string]*FragmentInfo

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

	// OverlappingFieldsCanBeMerged

	// A memoization for when two fragments are compared "between" each other for conflicts. Two
	// fragments may be compared many times, so memoizing this can dramatically improve the
	// performance of this validator.
	FragmentPairSet internal.ConflictFragmentPairSet

	// A cache for the "field map" and list of fragment names found in any given selection set.
	// Selection sets may be asked for this information multiple times, so this improves the
	// performance of this validator.
	FieldsAndFragmentNamesCache internal.FieldsAndFragmentNamesCache

	// UniqueFragmentNames
	KnownFragmentNames map[string]ast.Name

	// KnownTypeNames

	// existingTypeNames caches all type names occurred in the schema; This is lazily initialized at
	// the first time ExistingTypeNames is called. It is used by KnownTypeNames rule to make a
	// suggestion list.
	existingTypeNames []string
}

// newValidationContext initializes a validation context for validating given document.
func newValidationContext(schema graphql.Schema, document ast.Document, rules *rules) *ValidationContext {
	return &ValidationContext{
		schema:   schema,
		document: document,
		rules:    rules,

		skippingRules: make([]interface{}, rules.numRules),

		KnownOperationNames: map[string]ast.Name{},

		FragmentPairSet:             internal.NewConflictFragmentPairSet(),
		FieldsAndFragmentNamesCache: internal.NewFieldsAndFragmentNamesCache(),

		KnownFragmentNames: map[string]ast.Name{},
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

// TypeResolver creates ast.TypeResolver to resolve type for AST nodes during validation.
func (ctx *ValidationContext) TypeResolver() astutil.TypeResolver {
	return astutil.TypeResolver{
		Schema: ctx.schema,
	}
}

// FragmentInfo looks up the FragmentInfo for given fragment name in current document.
func (ctx *ValidationContext) FragmentInfo(name string) *FragmentInfo {
	fragmentInfoMap := ctx.fragmentInfos
	if fragmentInfoMap == nil {
		// Build map.
		fragmentInfoMap = map[string]*FragmentInfo{}
		resolver := ctx.TypeResolver()

		for _, definition := range ctx.document.Definitions {
			if definition, ok := definition.(*ast.FragmentDefinition); ok {
				fragmentInfoMap[definition.Name.Value()] = &FragmentInfo{
					def:           definition,
					typeCondition: resolver.ResolveType(definition.TypeCondition),
				}
			}
		}

		// Cache in ctx.
		ctx.fragmentInfos = fragmentInfoMap
	}
	return fragmentInfoMap[name]
}

// Fragment looks up fragment definition for given name in current document.
func (ctx *ValidationContext) Fragment(name string) *ast.FragmentDefinition {
	info := ctx.FragmentInfo(name)
	if info != nil {
		return info.Definition()
	}
	return nil
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

// ExistingTypeNames returns list of types declared in the schema.
func (ctx *ValidationContext) ExistingTypeNames() []string {
	existingTypeNames := ctx.existingTypeNames
	if existingTypeNames == nil {
		var (
			existingTypesMap        = ctx.Schema().TypeMap()
			existingTypesMapKeyIter = existingTypesMap.KeyIterator()
		)
		existingTypeNames = make([]string, 0, existingTypesMap.Size())
		for {
			name, err := existingTypesMapKeyIter.Next()
			if err != nil {
				break
			}
			existingTypeNames = append(existingTypeNames, name.(string))
		}

		// Cache the result in ctx.
		ctx.existingTypeNames = existingTypeNames
	}
	return existingTypeNames
}
