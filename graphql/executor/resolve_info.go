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
	"fmt"

	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/graphql/ast"
)

// ResolveInfo implements graphql.ResolveInfo to provide execution states for field and type
// resolvers.
type ResolveInfo struct {
	ExecutionContext *ExecutionContext
	ExecutionNode    *ExecutionNode
	ResultNode       *ResultNode
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
func (info *ResolveInfo) Schema() graphql.Schema {
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

// DataLoaderManager implements graphql.ResolveInfo.
func (info *ResolveInfo) DataLoaderManager() graphql.DataLoaderManager {
	return info.ExecutionContext.DataLoaderManager()
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

func parentFieldType(ctx *ExecutionContext, node *ExecutionNode) graphql.Object {
	parent := node.Parent.Field
	if parent != nil {
		switch parentType := graphql.NamedTypeOf(parent.Type()).(type) {
		case graphql.Object:
			return parentType

		case graphql.AbstractType:
			// Search node.Parent.Children to find the runtime object type of the given node.
			for runtimeType, nodes := range node.Parent.Children {
				for _, n := range nodes {
					if n == node {
						return runtimeType
					}
				}
			}
			panic(fmt.Sprintf(`unable to determine runtime type for field "%s" within abstract type `+
				`"%s"`, node.Field.Name(), parentType.Name()))

		default:
			panic(fmt.Sprintf("parent is unexpectedly a non-object type: %T", parentType))
		}
	}

	var (
		operation = ctx.Operation()
		schema    = operation.Schema()
	)
	switch operation.Type() {
	case ast.OperationTypeQuery:
		return schema.Query()
	case ast.OperationTypeMutation:
		return schema.Mutation()
	case ast.OperationTypeSubscription:
		return schema.Subscription()
	}

	panic("unknown object type")
}

// Object implements graphql.ResolveInfo.
func (info *ResolveInfo) Object() graphql.Object {
	return parentFieldType(info.ExecutionContext, info.ExecutionNode)
}

// FieldDefinitions implements graphql.ResolveInfo.
func (info *ResolveInfo) FieldDefinitions() []*ast.Field {
	return info.ExecutionNode.Definitions
}

// Field implements graphql.ResolveInfo.
func (info *ResolveInfo) Field() graphql.Field {
	return info.ExecutionNode.Field
}

// Path implements graphql.ResolveInfo.
func (info *ResolveInfo) Path() graphql.ResponsePath {
	return info.ResultNode.Path()
}

// Args implements graphql.ResolveInfo.
func (info *ResolveInfo) Args() graphql.ArgumentValues {
	return info.ExecutionNode.Args
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
func (info fieldSelectionInfo) Field() graphql.Field {
	return info.node.Field
}

// Args implements graphql.FieldSelectionInfo.
func (info fieldSelectionInfo) Args() graphql.ArgumentValues {
	return info.node.Args
}
