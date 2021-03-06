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
	"reflect"

	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/graphql/ast"
)

// rules contains a collection of actions to be performed on nodes for validation.
type rules struct {
	size                   int
	operationRules         operationRules
	variableRules          variableRules
	fragmentRules          fragmentRules
	selectionSetRules      selectionSetRules
	fieldRules             fieldRules
	fieldArgumentRules     fieldArgumentRules
	inlineFragmentRules    inlineFragmentRules
	fragmentSpreadRules    fragmentSpreadRules
	valueRules             valueRules
	variableUsageRules     variableUsageRules
	directivesRules        directivesRules
	directiveRules         directiveRules
	directiveArgumentRules directiveArgumentRules
}

func buildRules(rs ...interface{}) *rules {
	rules := &rules{
		size: len(rs),
	}
	for i, rule := range rs {
		isRule := false
		if r, ok := rule.(OperationRule); ok {
			operationRules := &rules.operationRules
			operationRules.indices = append(operationRules.indices, i)
			operationRules.rules = append(operationRules.rules, r)
			isRule = true
		}

		if r, ok := rule.(VariableRule); ok {
			variableRules := &rules.variableRules
			variableRules.indices = append(variableRules.indices, i)
			variableRules.rules = append(variableRules.rules, r)
			isRule = true
		}

		if r, ok := rule.(FragmentRule); ok {
			fragmentRules := &rules.fragmentRules
			fragmentRules.indices = append(fragmentRules.indices, i)
			fragmentRules.rules = append(fragmentRules.rules, r)
			isRule = true
		}

		if r, ok := rule.(SelectionSetRule); ok {
			selectionSetRules := &rules.selectionSetRules
			selectionSetRules.indices = append(selectionSetRules.indices, i)
			selectionSetRules.rules = append(selectionSetRules.rules, r)
			isRule = true
		}

		if r, ok := rule.(FieldRule); ok {
			fieldRules := &rules.fieldRules
			fieldRules.indices = append(fieldRules.indices, i)
			fieldRules.rules = append(fieldRules.rules, r)
			isRule = true
		}

		if r, ok := rule.(FieldArgumentRule); ok {
			fieldArgumentRules := &rules.fieldArgumentRules
			fieldArgumentRules.indices = append(fieldArgumentRules.indices, i)
			fieldArgumentRules.rules = append(fieldArgumentRules.rules, r)
			isRule = true
		}

		if r, ok := rule.(InlineFragmentRule); ok {
			inlineFragmentRules := &rules.inlineFragmentRules
			inlineFragmentRules.indices = append(inlineFragmentRules.indices, i)
			inlineFragmentRules.rules = append(inlineFragmentRules.rules, r)
			isRule = true
		}

		if r, ok := rule.(FragmentSpreadRule); ok {
			fragmentSpreadRules := &rules.fragmentSpreadRules
			fragmentSpreadRules.indices = append(fragmentSpreadRules.indices, i)
			fragmentSpreadRules.rules = append(fragmentSpreadRules.rules, r)
			isRule = true
		}

		if r, ok := rule.(ValueRule); ok {
			valueRules := &rules.valueRules
			valueRules.indices = append(valueRules.indices, i)
			valueRules.rules = append(valueRules.rules, r)
			isRule = true
		}

		if r, ok := rule.(VariableUsageRule); ok {
			variableUsageRules := &rules.variableUsageRules
			variableUsageRules.indices = append(variableUsageRules.indices, i)
			variableUsageRules.rules = append(variableUsageRules.rules, r)
			isRule = true
		}

		if r, ok := rule.(DirectivesRule); ok {
			directivesRule := &rules.directivesRules
			directivesRule.indices = append(directivesRule.indices, i)
			directivesRule.rules = append(directivesRule.rules, r)
			isRule = true
		}

		if r, ok := rule.(DirectiveRule); ok {
			directiveRules := &rules.directiveRules
			directiveRules.indices = append(directiveRules.indices, i)
			directiveRules.rules = append(directiveRules.rules, r)
			isRule = true
		}

		if r, ok := rule.(DirectiveArgumentRule); ok {
			directiveArgumentRules := &rules.directiveArgumentRules
			directiveArgumentRules.indices = append(directiveArgumentRules.indices, i)
			directiveArgumentRules.rules = append(directiveArgumentRules.rules, r)
			isRule = true
		}

		if !isRule {
			panic(fmt.Sprintf(`"%T" is not a validation rule`, rule))
		}
	}
	return rules
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

type operationRules struct {
	indices []int
	rules   []OperationRule
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

type variableRules struct {
	indices []int
	rules   []VariableRule
}

func (r *variableRules) Run(ctx *ValidationContext, info *VariableInfo) {
	indices := r.indices
	for i, rule := range r.rules {
		index := indices[i]
		// See whether we can run the rule.
		if !shouldSkipRule(ctx, index) {
			// Run the rule and set skipping state.
			setSkipping(ctx, index, info.Node(), rule.CheckVariable(ctx, info))
		}
	}
}

type fragmentRules struct {
	indices []int
	rules   []FragmentRule
}

func (r *fragmentRules) Run(ctx *ValidationContext, fragmentInfo *FragmentInfo, fragment *ast.FragmentDefinition) {
	indices := r.indices
	for i, rule := range r.rules {
		index := indices[i]
		// See whether we can run the rule.
		if !shouldSkipRule(ctx, index) {
			// Run the rule and set skipping state.
			setSkipping(ctx, index, fragment, rule.CheckFragment(ctx, fragmentInfo, fragment))
		}
	}
}

type selectionSetRules struct {
	indices []int
	rules   []SelectionSetRule
}

func (r *selectionSetRules) Run(ctx *ValidationContext, ttype graphql.Type, selectionSet ast.SelectionSet) {
	indices := r.indices
	for i, rule := range r.rules {
		index := indices[i]
		// See whether we can run the rule.
		if !shouldSkipRule(ctx, index) {
			// Run the rule and set skipping state.
			setSkipping(ctx, index, selectionSet, rule.CheckSelectionSet(ctx, ttype, selectionSet))
		}
	}
}

type fieldRules struct {
	indices []int
	rules   []FieldRule
}

func (r *fieldRules) Run(ctx *ValidationContext, field *FieldInfo) {
	indices := r.indices
	for i, rule := range r.rules {
		index := indices[i]
		// See whether we can run the rule.
		if !shouldSkipRule(ctx, index) {
			// Run the rule and set skipping state.
			setSkipping(ctx, index, field.Node(), rule.CheckField(ctx, field))
		}
	}
}

type fieldArgumentRules struct {
	indices []int
	rules   []FieldArgumentRule
}

func (r *fieldArgumentRules) Run(
	ctx *ValidationContext,
	field *FieldInfo,
	argDef *graphql.Argument,
	arg *ast.Argument) {

	indices := r.indices
	for i, rule := range r.rules {
		index := indices[i]
		// See whether we can run the rule.
		if !shouldSkipRule(ctx, index) {
			// Run the rule and set skipping state.
			setSkipping(ctx, index, arg, rule.CheckFieldArgument(ctx, field, argDef, arg))
		}
	}
}

type inlineFragmentRules struct {
	indices []int
	rules   []InlineFragmentRule
}

func (r *inlineFragmentRules) Run(ctx *ValidationContext, parentType graphql.Type, typeCondition graphql.Type, fragment *ast.InlineFragment) {
	indices := r.indices
	for i, rule := range r.rules {
		index := indices[i]
		// See whether we can run the rule.
		if !shouldSkipRule(ctx, index) {
			// Run the rule and set skipping state.
			setSkipping(ctx, index, fragment, rule.CheckInlineFragment(ctx, parentType, typeCondition, fragment))
		}
	}
}

type fragmentSpreadRules struct {
	indices []int
	rules   []FragmentSpreadRule
}

func (r *fragmentSpreadRules) Run(ctx *ValidationContext, parentType graphql.Type, fragmentInfo *FragmentInfo, fragmentSpread *ast.FragmentSpread) {
	indices := r.indices
	for i, rule := range r.rules {
		index := indices[i]
		// See whether we can run the rule.
		if !shouldSkipRule(ctx, index) {
			// Run the rule and set skipping state.
			setSkipping(ctx, index, fragmentSpread, rule.CheckFragmentSpread(ctx, parentType, fragmentInfo, fragmentSpread))
		}
	}
}

type valueRules struct {
	indices []int
	rules   []ValueRule
}

func (r *valueRules) Run(ctx *ValidationContext, valueType graphql.Type, value ast.Value) {
	indices := r.indices
	for i, rule := range r.rules {
		index := indices[i]
		// See whether we can run the rule.
		if !shouldSkipRule(ctx, index) {
			// Run the rule and set skipping state.
			setSkipping(ctx, index, value, rule.CheckValue(ctx, valueType, value))
		}
	}
}

type variableUsageRules struct {
	indices []int
	rules   []VariableUsageRule
}

func (r *variableUsageRules) Run(
	ctx *ValidationContext,
	ttype graphql.Type,
	variable ast.Variable,
	hasLocationDefaultValue bool,
	info *VariableInfo) {

	indices := r.indices
	for i, rule := range r.rules {
		index := indices[i]
		// See whether we can run the rule.
		if !shouldSkipRule(ctx, index) {
			// Run the rule and set skipping state.
			setSkipping(ctx, index, variable,
				rule.CheckVariableUsage(ctx, ttype, variable, hasLocationDefaultValue, info))
		}
	}
}

type directivesRules struct {
	indices []int
	rules   []DirectivesRule
}

func (r *directivesRules) Run(ctx *ValidationContext, directives ast.Directives, location graphql.DirectiveLocation) {
	indices := r.indices
	for i, rule := range r.rules {
		index := indices[i]
		// See whether we can run the rule.
		if !shouldSkipRule(ctx, index) {
			// Run the rule and set skipping state.
			setSkipping(ctx, index, directives, rule.CheckDirectives(ctx, directives, location))
		}
	}
}

type directiveRules struct {
	indices []int
	rules   []DirectiveRule
}

func (r *directiveRules) Run(ctx *ValidationContext, directive *DirectiveInfo) {
	indices := r.indices
	for i, rule := range r.rules {
		index := indices[i]
		// See whether we can run the rule.
		if !shouldSkipRule(ctx, index) {
			// Run the rule and set skipping state.
			setSkipping(ctx, index, directive.Node(), rule.CheckDirective(ctx, directive))
		}
	}
}

type directiveArgumentRules struct {
	indices []int
	rules   []DirectiveArgumentRule
}

func (r *directiveArgumentRules) Run(
	ctx *ValidationContext,
	directive *DirectiveInfo,
	argDef *graphql.Argument,
	arg *ast.Argument) {

	indices := r.indices
	for i, rule := range r.rules {
		index := indices[i]
		// See whether we can run the rule.
		if !shouldSkipRule(ctx, index) {
			// Run the rule and set skipping state.
			setSkipping(ctx, index, arg, rule.CheckDirectiveArgument(ctx, directive, argDef, arg))
		}
	}
}

func walk(ctx *ValidationContext) {
	for _, definitions := range ctx.Document().Definitions {
		if operation, ok := definitions.(*ast.OperationDefinition); ok {
			walkOperationDefinition(ctx, operation)
		}
	}

	// Fragment validation must come after all Operations have been validated so all uses of fragments
	// from operations can be seen by NoUnusedFragments.
	for _, definitions := range ctx.Document().Definitions {
		if fragment, ok := definitions.(*ast.FragmentDefinition); ok {
			walkFragmentDefinition(ctx, fragment)
		}
	}
}

func leaveNode(ctx *ValidationContext, node ast.Node) {
	skippingRules := ctx.skippingRules
	for i, skipping := range skippingRules {
		if skippingNode, ok := skipping.(ast.Node); ok && reflect.DeepEqual(skippingNode, node) {
			// Re-enable the rule.
			skippingRules[i] = nil
		}
	}
}

func walkOperationDefinition(ctx *ValidationContext, operation *ast.OperationDefinition) {
	ctx.currentOperation = operation

	// Build ctx.variableInfos.
	varDefs := operation.VariableDefinitions
	if len(varDefs) > 0 {
		var (
			typeResolver  = ctx.TypeResolver()
			variableInfos = make(map[string]*VariableInfo, len(varDefs))
		)
		for _, varDef := range varDefs {
			variableInfos[varDef.Variable.Name.Value()] = &VariableInfo{
				node:    varDef,
				typeDef: typeResolver.ResolveType(varDef.Type),
			}
		}
		ctx.variableInfos = variableInfos
	}

	// Allocate ctx.validatedFragments.
	ctx.validatedFragments = map[string]bool{}

	// Run operation rules.
	ctx.rules.operationRules.Run(ctx, operation)

	// Determine the Object type of the operation and directive location.
	var (
		object   graphql.Object
		location graphql.DirectiveLocation
	)
	switch operation.OperationType() {
	case ast.OperationTypeQuery:
		object = ctx.Schema().Query()
		location = graphql.DirectiveLocationQuery

	case ast.OperationTypeMutation:
		object = ctx.Schema().Mutation()
		location = graphql.DirectiveLocationMutation

	case ast.OperationTypeSubscription:
		object = ctx.Schema().Subscription()
		location = graphql.DirectiveLocationSubscription
	}

	// Visit directives.
	walkDirectives(ctx, operation.Directives, location)

	// Visit selection set.
	walkSelectionSet(ctx, object, operation.SelectionSet)

	// Visit variables. This must be the last one so uses of variables within the operation mark in
	// corresponding VariableInfo (by NoUnusedVariables.CheckVariableUsage) before NoUnusedVariables
	// to check unused variables (via NoUnusedVariables.CheckVariable).
	walkVariables(ctx, varDefs)

	// Call leave before return.
	leaveNode(ctx, operation)

	ctx.variableInfos = nil
	ctx.validatedFragments = nil
	ctx.currentOperation = nil
}

func walkVariables(ctx *ValidationContext, variables ast.VariableDefinitions) {
	infos := ctx.variableInfos

	for _, variable := range variables {
		// Look up info for the variable by name.
		info, exists := infos[variable.Variable.Name.Value()]

		// Check info.node.
		if !exists || info.node != variable {
			info = &VariableInfo{
				node:    variable,
				typeDef: ctx.TypeResolver().ResolveType(variable.Type),
			}
		}

		if variable.DefaultValue != nil {
			walkValue(ctx, info.TypeDef(), variable.DefaultValue, false /* hasLocationDefaultValue */)
		}

		// Visit directives on variable definition.
		walkDirectives(ctx, variable.Directives, graphql.DirectiveLocationVariableDefinition)

		// Run variable rules. These must be executed at the end so all variable usages are collected
		// before NoUnusedVariables checks for unused variables.
		ctx.rules.variableRules.Run(ctx, info)

		// Call leave before return.
		leaveNode(ctx, variable)
	}
}

func walkFragmentDefinition(ctx *ValidationContext, fragment *ast.FragmentDefinition) {
	fragmentInfo := ctx.FragmentInfo(fragment.Name.Value())

	// Run fragment rules.
	ctx.rules.fragmentRules.Run(ctx, fragmentInfo, fragment)

	walkDirectives(ctx, fragment.Directives, graphql.DirectiveLocationFragmentDefinition)

	walkSelectionSet(ctx, fragmentInfo.TypeCondition(), fragment.SelectionSet)

	// Call leave before return.
	leaveNode(ctx, fragment)
}

func walkSelectionSet(ctx *ValidationContext, ttype graphql.Type, selectionSet ast.SelectionSet) {
	ttype = graphql.NamedTypeOf(ttype)

	// Run selection set rules.
	ctx.rules.selectionSetRules.Run(ctx, ttype, selectionSet)

	for _, selection := range selectionSet {
		walkSelection(ctx, ttype, selection)
	}

	// Call leave before return.
	leaveNode(ctx, selectionSet)
}

func walkSelection(ctx *ValidationContext, parentType graphql.Type, selection ast.Selection) {
	switch selection := selection.(type) {
	case *ast.Field:
		walkField(ctx, parentType, selection)

	case *ast.InlineFragment:
		walkInlineFragment(ctx, parentType, selection)

	case *ast.FragmentSpread:
		walkFragmentSpread(ctx, parentType, selection)
	}
}

func walkFieldArguments(ctx *ValidationContext, field *FieldInfo) {
	var (
		arguments = field.Node().Arguments
		fieldDef  = field.Def()
	)
	if fieldDef == nil {
		for _, arg := range arguments {
			ctx.rules.fieldArgumentRules.Run(ctx, field, nil, arg)

			// Visit argument value.
			walkValue(ctx, nil, arg.Value, false /* hasLocationDefaultValue */)
		}
	} else {
		argDefs := fieldDef.Args()

		for _, arg := range arguments {
			var (
				argDef             *graphql.Argument
				argType            graphql.Type
				argHasDefaultValue bool
			)

			// Search definition for arg node from argDefs by name.
			argName := arg.Name.Value()
			for i := range argDefs {
				if argDefs[i].Name() == argName {
					argDef = &argDefs[i]
					argType = argDef.Type()
					argHasDefaultValue = argDef.HasDefaultValue()
					break
				}
			}

			ctx.rules.fieldArgumentRules.Run(ctx, field, argDef, arg)

			// Visit argument value.
			walkValue(ctx, argType, arg.Value, argHasDefaultValue)
		}
	}
}

func walkField(ctx *ValidationContext, parentType graphql.Type, field *ast.Field) {
	info := &FieldInfo{
		parentType: parentType,
		def:        ctx.TypeResolver().ResolveField(parentType, field),
		node:       field,
	}

	// Run field rules.
	ctx.rules.fieldRules.Run(ctx, info)

	// Visit arguments.
	walkFieldArguments(ctx, info)

	// Visit directives.
	walkDirectives(ctx, field.Directives, graphql.DirectiveLocationField)

	// Visit selection set of field.
	walkSelectionSet(ctx, info.Type(), field.SelectionSet)

	// Call leave before return.
	leaveNode(ctx, field)
}

func walkInlineFragment(ctx *ValidationContext, parentType graphql.Type, fragment *ast.InlineFragment) {
	var (
		typeCondition  graphql.Type
		nextParentType = parentType
	)
	if fragment.HasTypeCondition() {
		typeCondition = ctx.TypeResolver().ResolveType(fragment.TypeCondition)
		nextParentType = typeCondition
	}

	// Run inline fragment rules.
	ctx.rules.inlineFragmentRules.Run(ctx, parentType, typeCondition, fragment)

	// Visit directives.
	walkDirectives(ctx, fragment.Directives, graphql.DirectiveLocationInlineFragment)

	// Visit selection set.
	walkSelectionSet(ctx, nextParentType, fragment.SelectionSet)

	// Call leave before return.
	leaveNode(ctx, fragment)
}

func walkFragmentSpread(ctx *ValidationContext, parentType graphql.Type, fragmentSpread *ast.FragmentSpread) {
	fragmentInfo := ctx.FragmentInfo(fragmentSpread.Name.Value())

	// Run fragment spread rules.
	ctx.rules.fragmentSpreadRules.Run(ctx, parentType, fragmentInfo, fragmentSpread)

	// Visit directives.
	walkDirectives(ctx, fragmentSpread.Directives, graphql.DirectiveLocationFragmentSpread)

	if ctx.currentOperation != nil && fragmentInfo != nil {
		var (
			validatedFragments = ctx.validatedFragments
			fragmentName       = fragmentInfo.Name()
		)

		// Make sure the fragment hasn't been validated in current operation to avoid infinite
		// recursion.
		if !validatedFragments[fragmentName] {
			validatedFragments[fragmentName] = true

			// Use rulesForFragmentSpreads for visiting the selection set.
			savedRules := ctx.rules
			ctx.rules = ctx.rulesForFragmentSpreads

			walkSelectionSet(ctx, fragmentInfo.TypeCondition(), fragmentInfo.Definition().SelectionSet)

			// Backtrack the rule set.
			ctx.rules = savedRules
		}
	}

	// Call leave before return.
	leaveNode(ctx, fragmentSpread)
}

func walkValue(ctx *ValidationContext, valueType graphql.Type, value ast.Value, hasLocationDefaultValue bool) {
	// Run value rules.
	ctx.rules.valueRules.Run(ctx, valueType, value)

	switch value := value.(type) {
	case ast.Variable:
		// Run variable usage rules.
		if ctx.currentOperation != nil {
			ctx.rules.variableUsageRules.Run(
				ctx, valueType, value, hasLocationDefaultValue, ctx.VariableInfo(value.Name.Value()))
		}

	case ast.ListValue:
		listType, ok := graphql.NullableTypeOf(valueType).(graphql.List)
		if !ok {
			for _, v := range value.Values() {
				walkValue(ctx, nil, v, false /* hasLocationDefaultValue */)
			}
		} else {
			elementType := listType.ElementType()
			if !graphql.IsInputType(elementType) {
				elementType = nil
			}
			for _, v := range value.Values() {
				walkValue(ctx, elementType, v, false /* hasLocationDefaultValue */)
			}
		}

	case ast.ObjectValue:
		objectType, ok := graphql.NamedTypeOf(valueType).(graphql.InputObject)
		if !ok {
			for _, field := range value.Fields() {
				walkValue(ctx, nil, field.Value, false /* hasLocationDefaultValue */)
			}
		} else {
			fieldDefs := objectType.Fields()
			for _, field := range value.Fields() {
				var (
					fieldDef             = fieldDefs[field.Name.Value()]
					fieldType            graphql.Type
					fieldHasDefaultValue bool
				)
				if fieldDef != nil && graphql.IsInputType(fieldDef.Type()) {
					fieldType = fieldDef.Type()
					fieldHasDefaultValue = fieldDef.HasDefaultValue()
				}
				walkValue(ctx, fieldType, field.Value, fieldHasDefaultValue)
			}
		}
	}

	// Call leave before return.
	leaveNode(ctx, value)
}

func walkDirectives(ctx *ValidationContext, directives ast.Directives, location graphql.DirectiveLocation) {
	if len(directives) == 0 {
		return
	}

	// Run directives rules.
	ctx.rules.directivesRules.Run(ctx, directives, location)

	var (
		info = &DirectiveInfo{
			location: location,
		}
		directiveDefs = ctx.schema.Directives()
	)
	for _, directive := range directives {
		info.node = directive
		info.def = directiveDefs.Lookup(directive.Name.Value())

		// Run directive rules.
		ctx.rules.directiveRules.Run(ctx, info)

		// Visit arguments.
		walkDirectiveArguments(ctx, info)

		// Call leave.
		leaveNode(ctx, directive)
	}

	// Call leave before return.
	leaveNode(ctx, directives)
}

func walkDirectiveArguments(ctx *ValidationContext, directive *DirectiveInfo) {
	var (
		arguments    = directive.Node().Arguments
		directiveDef = directive.Def()
	)

	if directiveDef == nil {
		for _, arg := range arguments {
			ctx.rules.directiveArgumentRules.Run(ctx, directive, nil, arg)

			// Visit argument value.
			walkValue(ctx, nil, arg.Value, false /* hasLocationDefaultValue */)
		}
	} else {
		argDefs := directiveDef.Args()

		for _, arg := range arguments {
			var (
				argDef             *graphql.Argument
				argType            graphql.Type
				argHasDefaultValue bool
			)

			// Search definition for arg node from argDefs by name.
			argName := arg.Name.Value()
			for i := range argDefs {
				if argDefs[i].Name() == argName {
					argDef = &argDefs[i]
					argType = argDef.Type()
					argHasDefaultValue = argDef.HasDefaultValue()
					break
				}
			}

			ctx.rules.directiveArgumentRules.Run(ctx, directive, argDef, arg)

			// Visit argument value.
			walkValue(ctx, argType, arg.Value, argHasDefaultValue)
		}
	}
}
