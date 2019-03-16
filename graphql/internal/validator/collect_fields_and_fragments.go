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
	astutil "github.com/botobag/artemis/graphql/util/ast"
)

// FieldNodeAndDef contains information for a field node.
type FieldNodeAndDef struct {
	// The field node
	Node *ast.Field

	// The field definition of the field node in schema
	Def graphql.Field

	// Type that contains Def; Must be a composite type (i.e., Object, Interface or Union.)
	ParentType graphql.Type
}

// FieldNodeAndDefMap maps response key to their corresponding list of field nodes and definitions.
type FieldNodeAndDefMap map[string][]*FieldNodeAndDef

// SelectionSetFieldsAndFragmentNames contains a "field map" and list of fragment names found in a
// selection set.
type SelectionSetFieldsAndFragmentNames struct {
	// Fields in the selection set corresponding to a response key
	Fields FieldNodeAndDefMap

	// FragmentNames referenced by the selection set
	FragmentNames []string
}

var emptySelectionSetFieldsAndFragmentNames = &SelectionSetFieldsAndFragmentNames{}

// FieldsAndFragmentNamesCache caches the "field map" and list of fragment names found in any given
// selection set. Selection sets may be asked for this information multiple times, so this improves
// the performance of this validator.
type FieldsAndFragmentNamesCache struct {
	// The key is *ast.selectionSet which is the address of the first ast.Selection in a
	// ast.SelectionSet. ast.SelectionSet is a slice which cannot be used as map keys.
	entries map[*ast.Selection]*SelectionSetFieldsAndFragmentNames
}

// NewFieldsAndFragmentNamesCache initializes an empty FieldsAndFragmentNamesCache.
func NewFieldsAndFragmentNamesCache() FieldsAndFragmentNamesCache {
	return FieldsAndFragmentNamesCache{
		entries: map[*ast.Selection]*SelectionSetFieldsAndFragmentNames{},
	}
}

// CollectFieldsAndFragmentNamesInSelectionSet returns the collection of fields (a mapping of response
// name to field nodes and definitions) as well as a list of fragment names referenced via fragment
// spreads for given selection set.
func CollectFieldsAndFragmentNamesInSelectionSet(
	schema graphql.Schema,
	cache FieldsAndFragmentNamesCache,
	parentType graphql.Type,
	selectionSet ast.SelectionSet) *SelectionSetFieldsAndFragmentNames {

	if len(selectionSet) == 0 {
		return emptySelectionSetFieldsAndFragmentNames
	}

	// Lookup cache.
	key := &selectionSet[0]
	result, cached := cache.entries[key]
	if cached {
		return result
	}

	entry := &SelectionSetFieldsAndFragmentNames{
		Fields: map[string][]*FieldNodeAndDef{},
	}
	// Update cache.
	cache.entries[key] = entry

	fragments := map[string]bool{}

	type task struct {
		SelectionSet ast.SelectionSet
		ParentType   graphql.Type
	}
	queue := []task{
		{
			SelectionSet: selectionSet,
			ParentType:   parentType,
		},
	}

	typeResolver := astutil.TypeResolver{
		Schema: schema,
	}

	for len(queue) > 0 {
		selectionSetTask := queue[len(queue)-1]
		selectionSet, parentType, queue = selectionSetTask.SelectionSet, selectionSetTask.ParentType, queue[:len(queue)-1]

		for _, selection := range selectionSet {
			switch selection := selection.(type) {
			case *ast.Field:
				f := &FieldNodeAndDef{
					Node:       selection,
					ParentType: parentType,
				}

				fieldName := selection.Name.Value()
				switch t := parentType.(type) {
				case graphql.Object:
					f.Def = t.Fields()[fieldName]
				case graphql.Interface:
					f.Def = t.Fields()[fieldName]
				}

				responseName := selection.ResponseKey()
				entry.Fields[responseName] = append(entry.Fields[responseName], f)

			case *ast.InlineFragment:
				if selection.HasTypeCondition() {
					parentType = typeResolver.ResolveType(selection.TypeCondition)
				}

				queue = append(queue, task{
					SelectionSet: selection.SelectionSet,
					ParentType:   parentType,
				})

			case *ast.FragmentSpread:
				fragmentName := selection.Name.Value()
				if _, exists := fragments[fragmentName]; !exists {
					fragments[fragmentName] = true
					entry.FragmentNames = append(entry.FragmentNames, fragmentName)
				}
			}
		}
	}

	return entry
}

// CollectFieldsAndFragmentNamesInFragmentDefinition return the represented collection of fields as
// well as a list of nested fragment names referenced via fragment spreads.
func CollectFieldsAndFragmentNamesInFragmentDefinition(
	schema graphql.Schema,
	cache FieldsAndFragmentNamesCache,
	fragment *ast.FragmentDefinition) *SelectionSetFieldsAndFragmentNames {

	var (
		selectionSet = fragment.SelectionSet
		key          = &selectionSet[0]
	)

	// Short-circuit building a type from the node if possible.
	result, cached := cache.entries[key]
	if cached {
		return result
	}

	// Calling CollectFieldsAndFragmentNamesInSelectionSet below will update the cache.

	fragmentType := (astutil.TypeResolver{
		Schema: schema,
	}).ResolveType(fragment.TypeCondition)

	return CollectFieldsAndFragmentNamesInSelectionSet(schema, cache, fragmentType, selectionSet)
}
