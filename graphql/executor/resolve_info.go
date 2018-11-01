/**
 * Copyright (c) 2018, The Artemis Authors.
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

package executor

import (
	"context"

	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/graphql/ast"
)

// ResolveInfo implements graphql.ResolveInfo to provide execution states for field and type
// resolvers.
type ResolveInfo struct {
	ExecutionContext *ExecutionContext
	ExecutionNode    *ExecutionNode
	ResultNode       *ResultNode
	ParentType       *graphql.Object

	// This is embedded in the struct to make pass the context to completeValue and variants
	// (specifically for calling type resolvers in completeAbstractValue) without adding a parameter.
	ctx context.Context
}

// fieldSelectionInfo is an adapter which implements graphql.FieldSelection for ExecutionNode.
type fieldSelectionInfo struct {
	node *ExecutionNode
}

var (
	_ graphql.ResolveInfo        = (*ResolveInfo)(nil)
	_ graphql.FieldSelectionInfo = fieldSelectionInfo{}
)

// Schema implements graphql.ResolveInfo.
func (info *ResolveInfo) Schema() *graphql.Schema {
	return info.ExecutionContext.Operation().Schema()
}

// Document implements graphql.ResolveInfo.
func (info *ResolveInfo) Document() ast.Document {
	return info.ExecutionContext.Operation().Document()
}

// Operation implements graphql.ResolveInfo.
func (info *ResolveInfo) Operation() *ast.OperationDefinition {
	return info.ExecutionContext.Operation().Definition()
}

// RootValue implements graphql.ResolveInfo.
func (info *ResolveInfo) RootValue() interface{} {
	return info.ExecutionContext.RootValue()
}

// AppContext implements graphql.ResolveInfo.
func (info *ResolveInfo) AppContext() interface{} {
	return info.ExecutionContext.AppContext()
}

// VariableValues implements graphql.ResolveInfo.
func (info *ResolveInfo) VariableValues() graphql.VariableValues {
	return info.ExecutionContext.VariableValues()
}

// ParentFieldSelection implements graphql.ResolveInfo.
func (info *ResolveInfo) ParentFieldSelection() graphql.FieldSelectionInfo {
	return fieldSelectionInfo{info.ExecutionNode.Parent}
}

// Object implements graphql.ResolveInfo.
func (info *ResolveInfo) Object() *graphql.Object {
	return info.ParentType
}

// FieldDefinitions implements graphql.ResolveInfo.
func (info *ResolveInfo) FieldDefinitions() []*ast.Field {
	return info.ExecutionNode.Definitions
}

// Field implements graphql.ResolveInfo.
func (info *ResolveInfo) Field() *graphql.Field {
	return info.ExecutionNode.Field
}

// Path implements graphql.ResolveInfo.
func (info *ResolveInfo) Path() graphql.ResponsePath {
	return info.ResultNode.Path()
}

// ArgumentValues implements graphql.ResolveInfo.
func (info *ResolveInfo) ArgumentValues() graphql.ArgumentValues {
	return info.ExecutionNode.ArgumentValues
}

//===------------------------------------------------------------------------------------------===//
// fieldSelectionInfo
//===------------------------------------------------------------------------------------------===//

// ParentFieldSelection implements graphql.FieldSelectionInfo.
func (info fieldSelectionInfo) Parent() graphql.FieldSelectionInfo {
	return fieldSelectionInfo{info.node.Parent}
}

// FieldDefinitions implements graphql.FieldSelectionInfo.
func (info fieldSelectionInfo) FieldDefinitions() []*ast.Field {
	return info.node.Definitions
}

// Field implements graphql.FieldSelectionInfo.
func (info fieldSelectionInfo) Field() *graphql.Field {
	return info.node.Field
}

// ArgumentValues implements graphql.FieldSelectionInfo.
func (info fieldSelectionInfo) ArgumentValues() graphql.ArgumentValues {
	return info.node.ArgumentValues
}
