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

// NoFragmentCycles implements the "Fragments must not form cycles" validation rule.
//
// See https://facebook.github.io/graphql/June2018/#sec-Fragment-spreads-must-not-form-cycles.
type NoFragmentCycles struct{}

// CheckFragment implements validator.FragmentRule.
func (rule NoFragmentCycles) CheckFragment(
	ctx *validator.ValidationContext,
	fragmentInfo *validator.FragmentInfo,
	fragment *ast.FragmentDefinition) validator.NextCheckAction {

	if fragmentInfo.CycleChecked {
		return validator.SkipCheckForChildNodes
	}

	// The following detects whether fragment and the fragments that directly or indirectly referenced
	// by itself (via FragmentSpread) forms a cycle. This does a straight-forward DFS to find cycles
	// and uses fragmentInfo.CycleChecked to mark fragment that has been visited and checked. It does
	// not terminate when a cycle was found but continues to explore the graph to find all possible
	// cycles.
	type stackData struct {
		depth    int
		fragment *validator.FragmentInfo
		spread   *ast.FragmentSpread
	}

	var (
		stack []*stackData

		// Array of FragmentSpreads nodes that has been traversed used to produce meaningful errors
		spreadPath []*ast.FragmentSpread

		// Position in the spread path
		spreadPathIndexByName = map[string]int{
			fragment.Name.Value(): 0,
		}

		selectionSets []ast.SelectionSet
	)

	for {
		// Mark visited bit.
		fragmentInfo.CycleChecked = true

		selectionSets = append(selectionSets, fragmentInfo.Definition().SelectionSet)

		for len(selectionSets) > 0 {
			selectionSet := selectionSets[len(selectionSets)-1]
			selectionSets = selectionSets[:len(selectionSets)-1]

			// Scan selection set to find referenced fragments by inspecting fragment spreads.
			for i := len(selectionSet); i > 0; i-- {
				switch selection := selectionSet[i-1].(type) {
				case *ast.Field:
					selectionSets = append(selectionSets, selection.SelectionSet)

				case *ast.InlineFragment:
					selectionSets = append(selectionSets, selection.SelectionSet)

				case *ast.FragmentSpread:
					fragmentSpread := selection
					// Look up ctx to find fragment expanded by the fragment spread.
					f := ctx.FragmentInfo(fragmentSpread.Name.Value())

					if f == nil {
						// Skip spread that references to an unknown fragment.
						continue
					}

					// Detect cycle.
					spreadPathIdx, exists := spreadPathIndexByName[f.Name()]
					if !exists {
						if !f.CycleChecked {
							// Fragment has not been visited. Push to the stack to check it in the next iteration.
							stack = append(stack, &stackData{
								depth:    len(spreadPath),
								fragment: f,
								spread:   fragmentSpread,
							})
						}
						break
					}

					// Got a cycle.
					var (
						cyclePath     = spreadPath[spreadPathIdx:]
						fragmentNames []string
						locations     = make([]graphql.ErrorLocation, 0, len(cyclePath)+1)
					)
					if len(cyclePath) > 0 {
						fragmentNames = make([]string, len(cyclePath))
						for i, path := range cyclePath {
							fragmentNames[i] = path.Name.Value()
						}
					}
					for _, path := range cyclePath {
						locations = append(locations, graphql.ErrorLocationOfASTNode(path))
					}
					locations = append(locations, graphql.ErrorLocationOfASTNode(fragmentSpread))
					ctx.ReportError(
						messages.CycleErrorMessage(f.Name(), fragmentNames),
						locations,
					)
				}
			}
		}

		// Get next fragment to process from stack.
		if len(stack) == 0 {
			// No more to process.
			break
		}

		var data *stackData
		data, stack = stack[len(stack)-1], stack[:len(stack)-1]
		fragmentInfo = data.fragment

		// Handle spreadPath and spreadPathIndexByName. First remove all spreads below depth (DFS
		// traversal inherently guarantees len(spreadPath) is greater than or equal to data.depth).
		for _, spread := range spreadPath[data.depth:] {
			delete(spreadPathIndexByName, spread.Name.Value())
		}
		spreadPath = spreadPath[:data.depth]

		spreadPath = append(spreadPath, data.spread)
		spreadPathIndexByName[fragmentInfo.Name()] = len(spreadPath)
	}

	return validator.SkipCheckForChildNodes
}
