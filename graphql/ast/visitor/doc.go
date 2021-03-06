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

// Package visitor implements AST traversal.
//
// Visitor is a collection of actions, each of which specifies work to be done for certain type of
// node during AST traversal. There're may ways to construct a Visitor instance:
//
//	* NewFooVisitor creates a visitor that applies actions when visiting a Foo node.
//
//	v := visitor.NewNodeVisitor(
//		visitor.NodeVisitActionFunc(func(node ast.Node, ctx interface{}) visitor.Result {
//			// Return visitor.Continue or visitor.SkipSubTree or visitor.Break. See comments for
//			// visitor.Result for their semantics.
//		}))
//
//	Walk(doc, nil, v)
//
// This package also provides a Walk function which does preorder depth-first traversal on an AST
// and visits each node by calling corresponding visit functions in Visitor.
//
// Visitor is designed to be a "persistent" instance. That is, a visitor instance is expected to be
// initialized once and run many times for the same purpose. The 2nd argument to Walk will also
// passed to the Visit functions which is useful for preserving any states during traversal.
//
// The visitor itself is generated by code and avoids using reflection approach like the one in
// graphql-go [0] to reduce the overheads. Validators relies heavily on visitor efficiency.
package visitor
