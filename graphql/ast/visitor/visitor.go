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

//go:generate go run gen.go

package visitor

import (
	"github.com/botobag/artemis/graphql/ast"
)

// Result contains the return value for visitor function. The behavior of the visitor can be altered
// based on the value, including skipping over a sub-tree of AST (by returning SkipSubTree) or to
// stop the whole traversal (by returning Break).
type Result interface {
	// result puts a special mark for a Result type. The unexpored field forbids external package to
	// use our defined result constants.
	result()
}

// resultConstant is a constant to tell visitor of the next action to take
type resultConstant int

// result implements Result.
func (resultConstant) result() {}

// Enumeration of resultConstant.
const (
	// No action, continue the traversal.
	Continue resultConstant = iota

	// Skip over the sub-tree of AST
	SkipSubTree

	// Stop the traversal on return
	Break
)

// ASTNodes represents a list of AST node. It is designed for storing ancestor nodes when running
// through the visitors. Both appending and backward iteration are very efficient.
//
// Appending a new node to the list creates a new list without modifing the original one. Iterating
// nodes in the list backward is supported natively by following Prev link. However, the most
// efficient way to do a forward iteration is to walk through the entire list once to convert the
// list into an array and then scan the array. Fortunately, there's no use case for duing forward
// iteration on ancestor nodes.
type ASTNodes struct {
	node ast.Node
	prev *ASTNodes
}

// Append appends a node at the end of the current list. A new list is created and returned. The
// current list remains unchanged.
func (nodes *ASTNodes) Append(node ast.Node) *ASTNodes {
	return &ASTNodes{
		node: node,
		prev: nodes,
	}
}

// Back returns the node at the end of the list.
func (nodes *ASTNodes) Back() ast.Node {
	if nodes != nil {
		return nodes.node
	}
	return nil
}

// Prev returns the node list by excluding the last node in the list.
func (nodes *ASTNodes) Prev() *ASTNodes {
	return nodes.prev
}

// AsArray converts the node list into an array.
func (nodes *ASTNodes) AsArray() []ast.Node {
	var result []ast.Node
	for nodes != nil {
		result = append([]ast.Node{nodes.node}, result...)
		nodes = nodes.prev
	}
	return result
}

// Info is the 2nd argument to the visitor function which provides status about the visiting.
type Info struct {
	// All nodes visited before reaching parent of this node
	ancestors *ASTNodes

	// The user provided context
	context interface{}
}

// Ancestors returns info.ancestors.
func (info *Info) Ancestors() *ASTNodes {
	return info.ancestors
}

// Parent returns the parent of the visiting node.
func (info *Info) Parent() ast.Node {
	return info.ancestors.Back()
}

// Context returns info.context.
func (info *Info) Context() interface{} {
	return info.context
}

// withParent constructs a new Info with the given "parent" pushed at the end of info.ancestors. It
// is useful in preparing Info object for visiting children nodes.
func (info *Info) withParent(parent ast.Node) *Info {
	return &Info{
		ancestors: info.ancestors.Append(parent),
		context:   info.context,
	}
}
