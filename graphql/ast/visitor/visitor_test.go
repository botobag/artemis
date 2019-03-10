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

package visitor_test

import (
	"fmt"
	"io/ioutil"

	"github.com/botobag/artemis/graphql/ast"
	"github.com/botobag/artemis/graphql/ast/visitor"
	"github.com/botobag/artemis/graphql/parser"
	"github.com/botobag/artemis/graphql/token"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func parse(s string) ast.Document {
	doc, err := parser.Parse(token.NewSource(&token.SourceConfig{
		Body: token.SourceBody([]byte(s)),
	}))
	Expect(err).ShouldNot(HaveOccurred(), "%s", s)
	return doc
}

func nodeKind(node ast.Node) string {
	typeName := fmt.Sprintf("%T", node)
	// Strip "ast." or "*ast.".
	if typeName[0] == '*' {
		typeName = typeName[1:]
	}
	return typeName[4:]
}

func nodeValue(node ast.Node) interface{} {
	switch node := node.(type) {
	case ast.Name:
		return node.Value()
	case ast.IntValue, ast.FloatValue, ast.StringValue, ast.BooleanValue, ast.NullValue, ast.EnumValue:
		return node.(ast.Value).Interface()
	default:
		return nil
	}
}

var _ = Describe("Visitor", func() {
	It("validates ancestors argument", func() {
		var visitedNodes []ast.Node

		doc := parse("{ a }")

		v := visitor.NewBuilder().
			VisitNodeWith(&visitor.NodeVisitorFuncs{
				Enter: func(node ast.Node, info *visitor.Info) visitor.Result {
					Expect(info.Ancestors().AsArray()).Should(Equal(visitedNodes))
					visitedNodes = append(visitedNodes, node)
					return visitor.Continue
				},
				Leave: func(node ast.Node, info *visitor.Info) visitor.Result {
					visitedNodes = visitedNodes[:len(visitedNodes)-1]
					return visitor.Continue
				},
			}).Build()

		v.VisitDocument(doc, nil)
	})

	It("allows skipping a sub-tree", func() {
		var visited [][3]interface{}

		doc := parse("{ a, b { x }, c }")

		v := visitor.NewBuilder().
			VisitNodeWith(&visitor.NodeVisitorFuncs{
				Enter: func(node ast.Node, info *visitor.Info) visitor.Result {
					visited = append(visited, [3]interface{}{
						"enter", nodeKind(node), nodeValue(node),
					})

					if node, ok := node.(*ast.Field); ok && node.Name.Value() == "b" {
						return visitor.SkipSubTree
					}

					return visitor.Continue
				},
				Leave: func(node ast.Node, info *visitor.Info) visitor.Result {
					visited = append(visited, [3]interface{}{
						"leave", nodeKind(node), nodeValue(node),
					})
					return visitor.Continue
				},
			}).Build()

		v.VisitDocument(doc, nil)

		Expect(visited).Should(Equal([][3]interface{}{
			{"enter", "Document", nil},
			{"enter", "Definitions", nil},
			{"enter", "OperationDefinition", nil},
			{"enter", "SelectionSet", nil},
			{"enter", "Field", nil},
			{"enter", "Name", "a"},
			{"leave", "Name", "a"},
			{"leave", "Field", nil},
			{"enter", "Field", nil},
			{"enter", "Field", nil},
			{"enter", "Name", "c"},
			{"leave", "Name", "c"},
			{"leave", "Field", nil},
			{"leave", "SelectionSet", nil},
			{"leave", "OperationDefinition", nil},
			{"leave", "Definitions", nil},
			{"leave", "Document", nil},
		}))
	})

	It("allows early exit while visiting", func() {
		var visited [][3]interface{}

		doc := parse("{ a, b { x }, c }")

		v := visitor.NewBuilder().
			VisitNodeWith(&visitor.NodeVisitorFuncs{
				Enter: func(node ast.Node, info *visitor.Info) visitor.Result {
					visited = append(visited, [3]interface{}{
						"enter", nodeKind(node), nodeValue(node),
					})

					if node, ok := node.(ast.Name); ok && node.Value() == "x" {
						return visitor.Break
					}

					return visitor.Continue
				},
				Leave: func(node ast.Node, info *visitor.Info) visitor.Result {
					visited = append(visited, [3]interface{}{
						"leave", nodeKind(node), nodeValue(node),
					})
					return visitor.Continue
				},
			}).Build()

		v.VisitDocument(doc, nil)

		Expect(visited).Should(Equal([][3]interface{}{
			{"enter", "Document", nil},
			{"enter", "Definitions", nil},
			{"enter", "OperationDefinition", nil},
			{"enter", "SelectionSet", nil},
			{"enter", "Field", nil},
			{"enter", "Name", "a"},
			{"leave", "Name", "a"},
			{"leave", "Field", nil},
			{"enter", "Field", nil},
			{"enter", "Name", "b"},
			{"leave", "Name", "b"},
			{"enter", "SelectionSet", nil},
			{"enter", "Field", nil},
			{"enter", "Name", "x"},
		}))
	})

	It("allows early exit while leaving", func() {
		var visited [][3]interface{}

		doc := parse("{ a, b { x }, c }")

		v := visitor.NewBuilder().
			VisitNodeWith(&visitor.NodeVisitorFuncs{
				Enter: func(node ast.Node, info *visitor.Info) visitor.Result {
					visited = append(visited, [3]interface{}{
						"enter", nodeKind(node), nodeValue(node),
					})
					return visitor.Continue
				},
				Leave: func(node ast.Node, info *visitor.Info) visitor.Result {
					visited = append(visited, [3]interface{}{
						"leave", nodeKind(node), nodeValue(node),
					})

					if node, ok := node.(ast.Name); ok && node.Value() == "x" {
						return visitor.Break
					}

					return visitor.Continue
				},
			}).Build()

		v.VisitDocument(doc, nil)

		Expect(visited).Should(Equal([][3]interface{}{
			{"enter", "Document", nil},
			{"enter", "Definitions", nil},
			{"enter", "OperationDefinition", nil},
			{"enter", "SelectionSet", nil},
			{"enter", "Field", nil},
			{"enter", "Name", "a"},
			{"leave", "Name", "a"},
			{"leave", "Field", nil},
			{"enter", "Field", nil},
			{"enter", "Name", "b"},
			{"leave", "Name", "b"},
			{"enter", "SelectionSet", nil},
			{"enter", "Field", nil},
			{"enter", "Name", "x"},
			{"leave", "Name", "x"},
		}))
	})

	It("allows a named functions visitor API", func() {
		var visited [][3]interface{}

		doc := parse("{ a, b { x }, c }")

		v := visitor.NewBuilder().
			VisitNameWith(visitor.NameVisitorFunc(
				func(node ast.Name, info *visitor.Info) visitor.Result {
					visited = append(visited, [3]interface{}{
						"enter", nodeKind(node), nodeValue(node),
					})
					return visitor.Continue
				})).
			VisitSelectionSetWith(&visitor.SelectionSetVisitorFuncs{
				Enter: func(node ast.SelectionSet, info *visitor.Info) visitor.Result {
					visited = append(visited, [3]interface{}{
						"enter", nodeKind(node), nodeValue(node),
					})
					return visitor.Continue
				},
				Leave: func(node ast.SelectionSet, info *visitor.Info) visitor.Result {
					visited = append(visited, [3]interface{}{
						"leave", nodeKind(node), nodeValue(node),
					})
					return visitor.Continue
				},
			}).Build()

		v.VisitDocument(doc, nil)

		Expect(visited).Should(Equal([][3]interface{}{
			{"enter", "SelectionSet", nil},
			{"enter", "Name", "a"},
			{"enter", "Name", "b"},
			{"enter", "SelectionSet", nil},
			{"enter", "Name", "x"},
			{"leave", "SelectionSet", nil},
			{"enter", "Name", "c"},
			{"leave", "SelectionSet", nil},
		}))
	})

	It("Experimental: visits variables defined in fragments", func() {
		var visited [][3]interface{}

		doc, err := parser.Parse(token.NewSource(&token.SourceConfig{
			Body: token.SourceBody([]byte("fragment a($v: Boolean = false) on t { f }")),
		}), parser.EnableFragmentVariables())
		Expect(err).ShouldNot(HaveOccurred())

		v := visitor.NewBuilder().
			VisitNodeWith(&visitor.NodeVisitorFuncs{
				Enter: func(node ast.Node, info *visitor.Info) visitor.Result {
					visited = append(visited, [3]interface{}{
						"enter", nodeKind(node), nodeValue(node),
					})
					return visitor.Continue
				},
				Leave: func(node ast.Node, info *visitor.Info) visitor.Result {
					visited = append(visited, [3]interface{}{
						"leave", nodeKind(node), nodeValue(node),
					})
					return visitor.Continue
				},
			}).Build()

		v.VisitDocument(doc, nil)

		Expect(visited).Should(Equal([][3]interface{}{
			{"enter", "Document", nil},
			{"enter", "Definitions", nil},
			{"enter", "FragmentDefinition", nil},
			{"enter", "Name", "a"},
			{"leave", "Name", "a"},
			{"enter", "VariableDefinitions", nil},
			{"enter", "VariableDefinition", nil},
			{"enter", "Variable", nil},
			{"enter", "Name", "v"},
			{"leave", "Name", "v"},
			{"leave", "Variable", nil},
			{"enter", "NamedType", nil},
			{"enter", "Name", "Boolean"},
			{"leave", "Name", "Boolean"},
			{"leave", "NamedType", nil},
			{"enter", "BooleanValue", false},
			{"leave", "BooleanValue", false},
			{"leave", "VariableDefinition", nil},
			{"leave", "VariableDefinitions", nil},
			{"enter", "NamedType", nil},
			{"enter", "Name", "t"},
			{"leave", "Name", "t"},
			{"leave", "NamedType", nil},
			{"enter", "SelectionSet", nil},
			{"enter", "Field", nil},
			{"enter", "Name", "f"},
			{"leave", "Name", "f"},
			{"leave", "Field", nil},
			{"leave", "SelectionSet", nil},
			{"leave", "FragmentDefinition", nil},
			{"leave", "Definitions", nil},
			{"leave", "Document", nil},
		}))
	})

	It("visits kitchen sink", func() {
		kitchenSink, err := ioutil.ReadFile("../../parser/kitchen-sink.graphql")
		Expect(err).ShouldNot(HaveOccurred())

		doc, err := parser.Parse(token.NewSource(&token.SourceConfig{
			Body: token.SourceBody(kitchenSink),
		}))
		Expect(err).ShouldNot(HaveOccurred())

		var visited [][3]interface{}

		v := visitor.NewBuilder().
			VisitNodeWith(&visitor.NodeVisitorFuncs{
				Enter: func(node ast.Node, info *visitor.Info) visitor.Result {
					var parentKind interface{}
					if info.Parent() != nil {
						parentKind = nodeKind(info.Parent())
					}
					visited = append(visited, [3]interface{}{
						"enter", nodeKind(node), parentKind,
					})
					return visitor.Continue
				},
				Leave: func(node ast.Node, info *visitor.Info) visitor.Result {
					var parentKind interface{}
					if info.Parent() != nil {
						parentKind = nodeKind(info.Parent())
					}
					visited = append(visited, [3]interface{}{
						"leave", nodeKind(node), parentKind,
					})
					return visitor.Continue
				},
			}).Build()

		v.VisitDocument(doc, nil)

		Expect(visited).Should(Equal([][3]interface{}{
			{"enter", "Document", nil},
			{"enter", "Definitions", "Document"},
			{"enter", "OperationDefinition", "Definitions"},
			{"enter", "Name", "OperationDefinition"},
			{"leave", "Name", "OperationDefinition"},
			{"enter", "VariableDefinitions", "OperationDefinition"},
			{"enter", "VariableDefinition", "VariableDefinitions"},
			{"enter", "Variable", "VariableDefinition"},
			{"enter", "Name", "Variable"},
			{"leave", "Name", "Variable"},
			{"leave", "Variable", "VariableDefinition"},
			{"enter", "NamedType", "VariableDefinition"},
			{"enter", "Name", "NamedType"},
			{"leave", "Name", "NamedType"},
			{"leave", "NamedType", "VariableDefinition"},
			{"leave", "VariableDefinition", "VariableDefinitions"},
			{"enter", "VariableDefinition", "VariableDefinitions"},
			{"enter", "Variable", "VariableDefinition"},
			{"enter", "Name", "Variable"},
			{"leave", "Name", "Variable"},
			{"leave", "Variable", "VariableDefinition"},
			{"enter", "NamedType", "VariableDefinition"},
			{"enter", "Name", "NamedType"},
			{"leave", "Name", "NamedType"},
			{"leave", "NamedType", "VariableDefinition"},
			{"enter", "EnumValue", "VariableDefinition"},
			{"leave", "EnumValue", "VariableDefinition"},
			{"leave", "VariableDefinition", "VariableDefinitions"},
			{"leave", "VariableDefinitions", "OperationDefinition"},
			{"enter", "Directives", "OperationDefinition"},
			{"enter", "Directive", "Directives"},
			{"enter", "Name", "Directive"},
			{"leave", "Name", "Directive"},
			{"leave", "Directive", "Directives"},
			{"leave", "Directives", "OperationDefinition"},
			{"enter", "SelectionSet", "OperationDefinition"},
			{"enter", "Field", "SelectionSet"},
			{"enter", "Name", "Field"},
			{"leave", "Name", "Field"},
			{"enter", "Name", "Field"},
			{"leave", "Name", "Field"},
			{"enter", "Arguments", "Field"},
			{"enter", "Argument", "Arguments"},
			{"enter", "Name", "Argument"},
			{"leave", "Name", "Argument"},
			{"enter", "ListValue", "Argument"},
			{"enter", "IntValue", "ListValue"},
			{"leave", "IntValue", "ListValue"},
			{"enter", "IntValue", "ListValue"},
			{"leave", "IntValue", "ListValue"},
			{"leave", "ListValue", "Argument"},
			{"leave", "Argument", "Arguments"},
			{"leave", "Arguments", "Field"},
			{"enter", "SelectionSet", "Field"},
			{"enter", "Field", "SelectionSet"},
			{"enter", "Name", "Field"},
			{"leave", "Name", "Field"},
			{"leave", "Field", "SelectionSet"},
			{"enter", "InlineFragment", "SelectionSet"},
			{"enter", "NamedType", "InlineFragment"},
			{"enter", "Name", "NamedType"},
			{"leave", "Name", "NamedType"},
			{"leave", "NamedType", "InlineFragment"},
			{"enter", "Directives", "InlineFragment"},
			{"enter", "Directive", "Directives"},
			{"enter", "Name", "Directive"},
			{"leave", "Name", "Directive"},
			{"leave", "Directive", "Directives"},
			{"leave", "Directives", "InlineFragment"},
			{"enter", "SelectionSet", "InlineFragment"},
			{"enter", "Field", "SelectionSet"},
			{"enter", "Name", "Field"},
			{"leave", "Name", "Field"},
			{"enter", "SelectionSet", "Field"},
			{"enter", "Field", "SelectionSet"},
			{"enter", "Name", "Field"},
			{"leave", "Name", "Field"},
			{"leave", "Field", "SelectionSet"},
			{"enter", "Field", "SelectionSet"},
			{"enter", "Name", "Field"},
			{"leave", "Name", "Field"},
			{"enter", "Name", "Field"},
			{"leave", "Name", "Field"},
			{"enter", "Arguments", "Field"},
			{"enter", "Argument", "Arguments"},
			{"enter", "Name", "Argument"},
			{"leave", "Name", "Argument"},
			{"enter", "IntValue", "Argument"},
			{"leave", "IntValue", "Argument"},
			{"leave", "Argument", "Arguments"},
			{"enter", "Argument", "Arguments"},
			{"enter", "Name", "Argument"},
			{"leave", "Name", "Argument"},
			{"enter", "Variable", "Argument"},
			{"enter", "Name", "Variable"},
			{"leave", "Name", "Variable"},
			{"leave", "Variable", "Argument"},
			{"leave", "Argument", "Arguments"},
			{"leave", "Arguments", "Field"},
			{"enter", "Directives", "Field"},
			{"enter", "Directive", "Directives"},
			{"enter", "Name", "Directive"},
			{"leave", "Name", "Directive"},
			{"enter", "Arguments", "Directive"},
			{"enter", "Argument", "Arguments"},
			{"enter", "Name", "Argument"},
			{"leave", "Name", "Argument"},
			{"enter", "Variable", "Argument"},
			{"enter", "Name", "Variable"},
			{"leave", "Name", "Variable"},
			{"leave", "Variable", "Argument"},
			{"leave", "Argument", "Arguments"},
			{"leave", "Arguments", "Directive"},
			{"leave", "Directive", "Directives"},
			{"leave", "Directives", "Field"},
			{"enter", "SelectionSet", "Field"},
			{"enter", "Field", "SelectionSet"},
			{"enter", "Name", "Field"},
			{"leave", "Name", "Field"},
			{"leave", "Field", "SelectionSet"},
			{"enter", "FragmentSpread", "SelectionSet"},
			{"enter", "Name", "FragmentSpread"},
			{"leave", "Name", "FragmentSpread"},
			{"enter", "Directives", "FragmentSpread"},
			{"enter", "Directive", "Directives"},
			{"enter", "Name", "Directive"},
			{"leave", "Name", "Directive"},
			{"leave", "Directive", "Directives"},
			{"leave", "Directives", "FragmentSpread"},
			{"leave", "FragmentSpread", "SelectionSet"},
			{"leave", "SelectionSet", "Field"},
			{"leave", "Field", "SelectionSet"},
			{"leave", "SelectionSet", "Field"},
			{"leave", "Field", "SelectionSet"},
			{"leave", "SelectionSet", "InlineFragment"},
			{"leave", "InlineFragment", "SelectionSet"},
			{"enter", "InlineFragment", "SelectionSet"},
			{"enter", "Directives", "InlineFragment"},
			{"enter", "Directive", "Directives"},
			{"enter", "Name", "Directive"},
			{"leave", "Name", "Directive"},
			{"enter", "Arguments", "Directive"},
			{"enter", "Argument", "Arguments"},
			{"enter", "Name", "Argument"},
			{"leave", "Name", "Argument"},
			{"enter", "Variable", "Argument"},
			{"enter", "Name", "Variable"},
			{"leave", "Name", "Variable"},
			{"leave", "Variable", "Argument"},
			{"leave", "Argument", "Arguments"},
			{"leave", "Arguments", "Directive"},
			{"leave", "Directive", "Directives"},
			{"leave", "Directives", "InlineFragment"},
			{"enter", "SelectionSet", "InlineFragment"},
			{"enter", "Field", "SelectionSet"},
			{"enter", "Name", "Field"},
			{"leave", "Name", "Field"},
			{"leave", "Field", "SelectionSet"},
			{"leave", "SelectionSet", "InlineFragment"},
			{"leave", "InlineFragment", "SelectionSet"},
			{"enter", "InlineFragment", "SelectionSet"},
			{"enter", "SelectionSet", "InlineFragment"},
			{"enter", "Field", "SelectionSet"},
			{"enter", "Name", "Field"},
			{"leave", "Name", "Field"},
			{"leave", "Field", "SelectionSet"},
			{"leave", "SelectionSet", "InlineFragment"},
			{"leave", "InlineFragment", "SelectionSet"},
			{"leave", "SelectionSet", "Field"},
			{"leave", "Field", "SelectionSet"},
			{"leave", "SelectionSet", "OperationDefinition"},
			{"leave", "OperationDefinition", "Definitions"},
			{"enter", "OperationDefinition", "Definitions"},
			{"enter", "Name", "OperationDefinition"},
			{"leave", "Name", "OperationDefinition"},
			{"enter", "Directives", "OperationDefinition"},
			{"enter", "Directive", "Directives"},
			{"enter", "Name", "Directive"},
			{"leave", "Name", "Directive"},
			{"leave", "Directive", "Directives"},
			{"leave", "Directives", "OperationDefinition"},
			{"enter", "SelectionSet", "OperationDefinition"},
			{"enter", "Field", "SelectionSet"},
			{"enter", "Name", "Field"},
			{"leave", "Name", "Field"},
			{"enter", "Arguments", "Field"},
			{"enter", "Argument", "Arguments"},
			{"enter", "Name", "Argument"},
			{"leave", "Name", "Argument"},
			{"enter", "IntValue", "Argument"},
			{"leave", "IntValue", "Argument"},
			{"leave", "Argument", "Arguments"},
			{"leave", "Arguments", "Field"},
			{"enter", "Directives", "Field"},
			{"enter", "Directive", "Directives"},
			{"enter", "Name", "Directive"},
			{"leave", "Name", "Directive"},
			{"leave", "Directive", "Directives"},
			{"leave", "Directives", "Field"},
			{"enter", "SelectionSet", "Field"},
			{"enter", "Field", "SelectionSet"},
			{"enter", "Name", "Field"},
			{"leave", "Name", "Field"},
			{"enter", "SelectionSet", "Field"},
			{"enter", "Field", "SelectionSet"},
			{"enter", "Name", "Field"},
			{"leave", "Name", "Field"},
			{"enter", "Directives", "Field"},
			{"enter", "Directive", "Directives"},
			{"enter", "Name", "Directive"},
			{"leave", "Name", "Directive"},
			{"leave", "Directive", "Directives"},
			{"leave", "Directives", "Field"},
			{"leave", "Field", "SelectionSet"},
			{"leave", "SelectionSet", "Field"},
			{"leave", "Field", "SelectionSet"},
			{"leave", "SelectionSet", "Field"},
			{"leave", "Field", "SelectionSet"},
			{"leave", "SelectionSet", "OperationDefinition"},
			{"leave", "OperationDefinition", "Definitions"},
			{"enter", "OperationDefinition", "Definitions"},
			{"enter", "Name", "OperationDefinition"},
			{"leave", "Name", "OperationDefinition"},
			{"enter", "VariableDefinitions", "OperationDefinition"},
			{"enter", "VariableDefinition", "VariableDefinitions"},
			{"enter", "Variable", "VariableDefinition"},
			{"enter", "Name", "Variable"},
			{"leave", "Name", "Variable"},
			{"leave", "Variable", "VariableDefinition"},
			{"enter", "NamedType", "VariableDefinition"},
			{"enter", "Name", "NamedType"},
			{"leave", "Name", "NamedType"},
			{"leave", "NamedType", "VariableDefinition"},
			{"leave", "VariableDefinition", "VariableDefinitions"},
			{"leave", "VariableDefinitions", "OperationDefinition"},
			{"enter", "Directives", "OperationDefinition"},
			{"enter", "Directive", "Directives"},
			{"enter", "Name", "Directive"},
			{"leave", "Name", "Directive"},
			{"leave", "Directive", "Directives"},
			{"leave", "Directives", "OperationDefinition"},
			{"enter", "SelectionSet", "OperationDefinition"},
			{"enter", "Field", "SelectionSet"},
			{"enter", "Name", "Field"},
			{"leave", "Name", "Field"},
			{"enter", "Arguments", "Field"},
			{"enter", "Argument", "Arguments"},
			{"enter", "Name", "Argument"},
			{"leave", "Name", "Argument"},
			{"enter", "Variable", "Argument"},
			{"enter", "Name", "Variable"},
			{"leave", "Name", "Variable"},
			{"leave", "Variable", "Argument"},
			{"leave", "Argument", "Arguments"},
			{"leave", "Arguments", "Field"},
			{"enter", "SelectionSet", "Field"},
			{"enter", "Field", "SelectionSet"},
			{"enter", "Name", "Field"},
			{"leave", "Name", "Field"},
			{"enter", "SelectionSet", "Field"},
			{"enter", "Field", "SelectionSet"},
			{"enter", "Name", "Field"},
			{"leave", "Name", "Field"},
			{"enter", "SelectionSet", "Field"},
			{"enter", "Field", "SelectionSet"},
			{"enter", "Name", "Field"},
			{"leave", "Name", "Field"},
			{"leave", "Field", "SelectionSet"},
			{"leave", "SelectionSet", "Field"},
			{"leave", "Field", "SelectionSet"},
			{"enter", "Field", "SelectionSet"},
			{"enter", "Name", "Field"},
			{"leave", "Name", "Field"},
			{"enter", "SelectionSet", "Field"},
			{"enter", "Field", "SelectionSet"},
			{"enter", "Name", "Field"},
			{"leave", "Name", "Field"},
			{"leave", "Field", "SelectionSet"},
			{"leave", "SelectionSet", "Field"},
			{"leave", "Field", "SelectionSet"},
			{"leave", "SelectionSet", "Field"},
			{"leave", "Field", "SelectionSet"},
			{"leave", "SelectionSet", "Field"},
			{"leave", "Field", "SelectionSet"},
			{"leave", "SelectionSet", "OperationDefinition"},
			{"leave", "OperationDefinition", "Definitions"},
			{"enter", "FragmentDefinition", "Definitions"},
			{"enter", "Name", "FragmentDefinition"},
			{"leave", "Name", "FragmentDefinition"},
			{"enter", "NamedType", "FragmentDefinition"},
			{"enter", "Name", "NamedType"},
			{"leave", "Name", "NamedType"},
			{"leave", "NamedType", "FragmentDefinition"},
			{"enter", "Directives", "FragmentDefinition"},
			{"enter", "Directive", "Directives"},
			{"enter", "Name", "Directive"},
			{"leave", "Name", "Directive"},
			{"leave", "Directive", "Directives"},
			{"leave", "Directives", "FragmentDefinition"},
			{"enter", "SelectionSet", "FragmentDefinition"},
			{"enter", "Field", "SelectionSet"},
			{"enter", "Name", "Field"},
			{"leave", "Name", "Field"},
			{"enter", "Arguments", "Field"},
			{"enter", "Argument", "Arguments"},
			{"enter", "Name", "Argument"},
			{"leave", "Name", "Argument"},
			{"enter", "Variable", "Argument"},
			{"enter", "Name", "Variable"},
			{"leave", "Name", "Variable"},
			{"leave", "Variable", "Argument"},
			{"leave", "Argument", "Arguments"},
			{"enter", "Argument", "Arguments"},
			{"enter", "Name", "Argument"},
			{"leave", "Name", "Argument"},
			{"enter", "Variable", "Argument"},
			{"enter", "Name", "Variable"},
			{"leave", "Name", "Variable"},
			{"leave", "Variable", "Argument"},
			{"leave", "Argument", "Arguments"},
			{"enter", "Argument", "Arguments"},
			{"enter", "Name", "Argument"},
			{"leave", "Name", "Argument"},
			{"enter", "ObjectValue", "Argument"},
			{"enter", "ObjectField", "ObjectValue"},
			{"enter", "Name", "ObjectField"},
			{"leave", "Name", "ObjectField"},
			{"enter", "StringValue", "ObjectField"},
			{"leave", "StringValue", "ObjectField"},
			{"leave", "ObjectField", "ObjectValue"},
			{"enter", "ObjectField", "ObjectValue"},
			{"enter", "Name", "ObjectField"},
			{"leave", "Name", "ObjectField"},
			{"enter", "StringValue", "ObjectField"},
			{"leave", "StringValue", "ObjectField"},
			{"leave", "ObjectField", "ObjectValue"},
			{"leave", "ObjectValue", "Argument"},
			{"leave", "Argument", "Arguments"},
			{"leave", "Arguments", "Field"},
			{"leave", "Field", "SelectionSet"},
			{"leave", "SelectionSet", "FragmentDefinition"},
			{"leave", "FragmentDefinition", "Definitions"},
			{"enter", "OperationDefinition", "Definitions"},
			{"enter", "SelectionSet", "OperationDefinition"},
			{"enter", "Field", "SelectionSet"},
			{"enter", "Name", "Field"},
			{"leave", "Name", "Field"},
			{"enter", "Arguments", "Field"},
			{"enter", "Argument", "Arguments"},
			{"enter", "Name", "Argument"},
			{"leave", "Name", "Argument"},
			{"enter", "BooleanValue", "Argument"},
			{"leave", "BooleanValue", "Argument"},
			{"leave", "Argument", "Arguments"},
			{"enter", "Argument", "Arguments"},
			{"enter", "Name", "Argument"},
			{"leave", "Name", "Argument"},
			{"enter", "BooleanValue", "Argument"},
			{"leave", "BooleanValue", "Argument"},
			{"leave", "Argument", "Arguments"},
			{"enter", "Argument", "Arguments"},
			{"enter", "Name", "Argument"},
			{"leave", "Name", "Argument"},
			{"enter", "NullValue", "Argument"},
			{"leave", "NullValue", "Argument"},
			{"leave", "Argument", "Arguments"},
			{"leave", "Arguments", "Field"},
			{"leave", "Field", "SelectionSet"},
			{"enter", "Field", "SelectionSet"},
			{"enter", "Name", "Field"},
			{"leave", "Name", "Field"},
			{"leave", "Field", "SelectionSet"},
			{"leave", "SelectionSet", "OperationDefinition"},
			{"leave", "OperationDefinition", "Definitions"},
			{"enter", "OperationDefinition", "Definitions"},
			{"enter", "SelectionSet", "OperationDefinition"},
			{"enter", "Field", "SelectionSet"},
			{"enter", "Name", "Field"},
			{"leave", "Name", "Field"},
			{"leave", "Field", "SelectionSet"},
			{"leave", "SelectionSet", "OperationDefinition"},
			{"leave", "OperationDefinition", "Definitions"},
			{"leave", "Definitions", "Document"},
			{"leave", "Document", nil},
		}))
	})
})
