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

	"github.com/botobag/artemis/graphql/ast"
)

// rules contains a collection of actions to be performed on nodes for validation.
type rules struct {
	numRules       int
	operationRules operationRules
}

func buildRules(rs ...interface{}) *rules {
	rules := &rules{
		numRules: len(rs),
	}
	for i, rule := range rs {
		isRule := false
		if r, ok := rule.(OperationRule); ok {
			operationRules := &rules.operationRules
			operationRules.indices = append(operationRules.indices, i)
			operationRules.rules = append(operationRules.rules, r)
			isRule = true
		}

		if !isRule {
			panic(fmt.Sprintf(`"%T" is not a validation rule`, rule))
		}
	}
	return rules
}

type operationRules struct {
	indices []int
	rules   []OperationRule
}

func shouldSkipRule(ctx *ValidationContext, ruleIndex int) bool {
	return ctx.skippingRules[ruleIndex] != nil
}

func setSkipping(ctx *ValidationContext, ruleIndex int, node ast.Node, nextCheckAction NextCheckAction) {
	switch nextCheckAction {
	case ContinueCheck:
		/* Nothing to do */

	case SkipCheckForChildNodes:
		ctx.skippingRules[ruleIndex] = node

	case StopCheck:
		ctx.skippingRules[ruleIndex] = StopCheck
	}
}

func (r *operationRules) Run(ctx *ValidationContext, operation *ast.OperationDefinition) {
	indices := r.indices
	for i, rule := range r.rules {
		index := indices[i]
		// See whether we can run the rule.
		if !shouldSkipRule(ctx, index) {
			// Run the rule and set skipping state.
			setSkipping(ctx, index, operation, rule.CheckOperation(ctx, operation))
		}
	}
}

func walk(ctx *ValidationContext) {
	for _, definitions := range ctx.Document().Definitions {
		switch def := definitions.(type) {
		case *ast.OperationDefinition:
			walkOperationDefinition(ctx, def)

		case *ast.FragmentDefinition:
			walkFragmentDefinition(ctx, def)
		}
	}
}

func leaveNode(ctx *ValidationContext, node ast.Node) {
	skippingRules := ctx.skippingRules
	for i, skipping := range skippingRules {
		if skippingNode, ok := skipping.(ast.Node); ok && skippingNode == node {
			// Re-enable the rule.
			skippingRules[i] = nil
		}
	}
}

func walkOperationDefinition(ctx *ValidationContext, operation *ast.OperationDefinition) {
	ctx.currentOperation = operation

	// Run operation rules.
	ctx.rules.operationRules.Run(ctx, operation)

	// Call leave before return.
	leaveNode(ctx, operation)

	ctx.currentOperation = nil
}

func walkFragmentDefinition(ctx *ValidationContext, fragment *ast.FragmentDefinition) {
	// TODO
}
