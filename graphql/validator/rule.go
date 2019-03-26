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
)

// A Rule implements an ast.Visitor to validate nodes in a GraphQL document according to one of the
// sections under "Validation" in specification [0].
//
// [0]: https://facebook.github.io/graphql/June2018/#sec-Validation

// NextCheckAction is the type of return value from rule's Check function. It specifies which action
// to take when the rule is invoked next time in current validation request.
type NextCheckAction int

// Enumeration of NextCheckAction
const (
	// Continue running the rule
	ContinueCheck NextCheckAction = iota

	// Don't run the rule on any of child nodes of the current one
	SkipCheckForChildNodes

	// Stop running the rule in current validation request
	StopCheck
)

// OperationRule validates an OperationDefinition.
type OperationRule interface {
	CheckOperation(ctx *ValidationContext, operation *ast.OperationDefinition) NextCheckAction
}

// FragmentRule validates an FragmentDefinition.
type FragmentRule interface {
	CheckFragment(
		ctx *ValidationContext,
		fragmentInfo *FragmentInfo,
		fragment *ast.FragmentDefinition) NextCheckAction
}

// SelectionSetRule validates a SelectionSet.
type SelectionSetRule interface {
	CheckSelectionSet(
		ctx *ValidationContext,
		ttype graphql.Type,
		selectionSet ast.SelectionSet) NextCheckAction
}

// FieldInfo provides information of the field to be checked for FieldRule and FieldArgumentRule.
type FieldInfo struct {
	parentType    graphql.Type
	def           graphql.Field
	node          *ast.Field
	knownArgNames []string
}

// ParentType returns type of parent that includes the field; Must be a composite type (Object,
// Union or Interface.)
func (info *FieldInfo) ParentType() graphql.Type {
	return info.parentType
}

// Def returns field definition corresponding to the node in schema (could be nil; For example, in
// the case of unknown fields.)
func (info *FieldInfo) Def() graphql.Field {
	return info.def
}

// Type returns definition of the field type in schema. Could be nil if the field definition is not
// available.
func (info *FieldInfo) Type() graphql.Type {
	if info.def != nil {
		return info.def.Type()
	}
	return nil
}

// Node returns AST node that specifies the field
func (info *FieldInfo) Node() *ast.Field {
	return info.node
}

// Name returns field name.
func (info *FieldInfo) Name() string {
	return info.node.Name.Value()
}

// KnownArgNames returns list of argument names in the field. This is used by KnownArgumentNames
// rule to make suggestion when an unknown argument is given. It is lazily computed on first call to
// KnownArgName.
func (info *FieldInfo) KnownArgNames() []string {
	knownArgNames := info.knownArgNames
	if knownArgNames != nil {
		return knownArgNames
	}

	def := info.def
	if def != nil {
		argDefs := def.Args()
		knownArgNames = make([]string, len(argDefs))
		for i := range argDefs {
			knownArgNames[i] = argDefs[i].Name()
		}
		// Cache in info.knownArgNames for later accesses.
		info.knownArgNames = knownArgNames
	}

	return knownArgNames
}

// FieldRule validates a Field.
type FieldRule interface {
	CheckField(ctx *ValidationContext, field *FieldInfo) NextCheckAction
}

// FieldArgumentRule validates a Argument in a Field.
type FieldArgumentRule interface {
	CheckFieldArgument(
		ctx *ValidationContext,
		field *FieldInfo,
		argDef *graphql.Argument,
		arg *ast.Argument) NextCheckAction
}

// InlineFragmentRule validates a InlineFragment.
type InlineFragmentRule interface {
	CheckInlineFragment(
		ctx *ValidationContext,
		parentType graphql.Type,
		typeCondition graphql.Type,
		fragment *ast.InlineFragment) NextCheckAction
}

// FragmentSpreadRule validates a FragmentSpread.
type FragmentSpreadRule interface {
	CheckFragmentSpread(
		ctx *ValidationContext,
		parentType graphql.Type,
		fragmentInfo *FragmentInfo,
		fragmentSpread *ast.FragmentSpread) NextCheckAction
}

// DirectiveInfo provides information of the field to be checked for DirectiveRule and DirectiveArgumentRule.
type DirectiveInfo struct {
	def           graphql.Directive
	node          *ast.Directive
	location      graphql.DirectiveLocation
	knownArgNames []string
}

// Def returns directive definition corresponding to the node in schema (could be nil; For example,
// in the case of unknown directives.)
func (info *DirectiveInfo) Def() graphql.Directive {
	return info.def
}

// Node returns AST node that specifies the directive
func (info *DirectiveInfo) Node() *ast.Directive {
	return info.node
}

// Name returns directive name.
func (info *DirectiveInfo) Name() string {
	return info.node.Name.Value()
}

// Location indicates the place where the directive node appears in the document.
func (info *DirectiveInfo) Location() graphql.DirectiveLocation {
	return info.location
}

// KnownArgNames returns list of argument names to the directive. This is used by KnownArgumentNames
// rule to make suggestion when an unknown argument is given. It is lazily computed on first call to
// KnownArgName.
func (info *DirectiveInfo) KnownArgNames() []string {
	knownArgNames := info.knownArgNames
	if knownArgNames != nil {
		return knownArgNames
	}

	def := info.def
	if def != nil {
		argDefs := def.Args()
		knownArgNames = make([]string, len(argDefs))
		for i := range argDefs {
			knownArgNames[i] = argDefs[i].Name()
		}
		// Cache in info.knownArgNames for later accesses.
		info.knownArgNames = knownArgNames
	}

	return knownArgNames
}

// DirectiveRule validates a Directive.
type DirectiveRule interface {
	CheckDirective(ctx *ValidationContext, directive *DirectiveInfo) NextCheckAction
}

// DirectiveArgumentRule validates a Argument in a Directive.
type DirectiveArgumentRule interface {
	CheckDirectiveArgument(
		ctx *ValidationContext,
		directive *DirectiveInfo,
		argDef *graphql.Argument,
		arg *ast.Argument) NextCheckAction
}
