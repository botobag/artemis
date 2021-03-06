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
	"unsafe"

	"github.com/botobag/artemis/graphql"
)

// ResultKind specifies which kind of value is held in the ResultNode. More specifically, it
// describes the type of ResultNode.Value.
type ResultKind uint16

// Enumeration of ResultKind
const (
	// ResultNode was resolved to a nil value (either because the field resolve to a nil value
	// or because an error occurred.) The Value contains an nil interface.
	ResultKindNil ResultKind = iota

	// ResultNode was resolved to a List value. The value contains will be an []ResultNode.
	ResultKindList

	// ResultNode was resolved to an Object value. The value contains contains an ObjectResultValue.
	ResultKindObject

	// ResultNode was resolved to a Scalar or Enum value. The value contains contains value that has
	// went through type's result coercion.
	ResultKindLeaf
)

// ResultFlag includes some useful properties of this ResultNode.
type ResultFlag uint16

// Enumeration of ResultFlag
const (
	// ResultNode must represents a value other than nil.
	ResultFlagRejectNull ResultFlag = 1 << iota
)

// A ResultNode holds a field value. Result data from an execution of a GraphQL Operation [0] is
// made up of ResultNode's formed in a tree structure. ResultNode can be serialized to the response
// format
//
// [0]: https://graphql.github.io/graphql-spec/June2018/#sec-Executing-Operations
// [1]: https://graphql.github.io/graphql-spec/June2018/#sec-Data
type ResultNode struct {
	// Pointer to the upper level of ResultNode in the result tree
	Parent *ResultNode

	// Kind describes kind of Value
	Kind ResultKind

	// Flags describes properties of Value. It is a bit set containing ResultFlag's.
	Flags uint16

	// The result value; This could be in a various format based on Kind.
	Value interface{}
}

const sizeOfResultNode = unsafe.Sizeof(ResultNode{})

// ObjectResultValue stores result from executing an Object field.
type ObjectResultValue struct {
	// The ExecutionNode's of Select Set that resolved FieldValues
	ExecutionNodes []*ExecutionNode

	// An array of ResultNode's each of which stores the result from executing the corresponding
	// ExecutionNode in ExecutionNode.
	FieldValues []ResultNode
}

// IsNil returns true if the node holds nil value (either because the field resolve to a nil value
// or because an error occurred.)
func (node *ResultNode) IsNil() bool {
	return node.Kind == ResultKindNil
}

// IsList returns true if the node holds result for a List field.
func (node *ResultNode) IsList() bool {
	return node.Kind == ResultKindList
}

// IsObject returns true if the node holds result for an Object field.
func (node *ResultNode) IsObject() bool {
	return node.Kind == ResultKindObject
}

// IsLeaf returns true if the node holds result for a Scalar or a Enum field.
func (node *ResultNode) IsLeaf() bool {
	return node.Kind == ResultKindLeaf
}

// SetToRejectNull marks the result to reject a nil value.
func (node *ResultNode) SetToRejectNull() {
	node.Flags = node.Flags | uint16(ResultFlagRejectNull)
	return
}

// ShouldRejectNull describes the result should not be nil.
func (node *ResultNode) ShouldRejectNull() bool {
	return (node.Flags & uint16(ResultFlagRejectNull)) != 0
}

// ListValue returns a value that is held by this node for a List field. It would panic if this is
// not a resolved List result (i.e., IsList returns false).
func (node *ResultNode) ListValue() ResultNodeList {
	return node.Value.(ResultNodeList)
}

// ObjectValue returns a value that is held by this node for a Object field. It would panic if this is
// not a resolved Object result (i.e., IsObject returns false).
func (node *ResultNode) ObjectValue() *ObjectResultValue {
	return node.Value.(*ObjectResultValue)
}

// Path in the response to this node.
func (node *ResultNode) Path() graphql.ResponsePath {
	var (
		path      graphql.ResponsePath
		pathKeys  []interface{}
		childNode = node
	)

	if node == nil {
		return path
	}

	for node := node.Parent; node != nil; node = node.Parent {
		if node.IsList() {
			pathKeys = append(pathKeys, node.ListValue().IndexOf(childNode))
		} else if node.IsObject() {
			fieldNodes := node.ObjectValue().FieldValues

			// Find index.
			childNodeAddr := uintptr(unsafe.Pointer(childNode))
			firstFieldNodeAddr := uintptr(unsafe.Pointer(&fieldNodes[0]))
			fieldIndex := int((childNodeAddr - firstFieldNodeAddr) / sizeOfResultNode)

			pathKeys = append(pathKeys, node.ObjectValue().ExecutionNodes[fieldIndex].ResponseKey())
		} else {
			// ??
			continue
		}

		childNode = node
	}

	// Pour keys in pathKeys to path in reverse order.
	for i := len(pathKeys) - 1; i >= 0; i-- {
		pathKey := pathKeys[i]
		switch pathKey := pathKey.(type) {
		case int:
			// List index
			path.AppendIndex(pathKey)
		case string:
			// Object field
			path.AppendFieldName(pathKey)
		}
	}

	return path
}
