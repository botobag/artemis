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
	doc, err := parser.Parse(token.NewSource(s))
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
		var visitedNodes [][2]interface{}

		doc := parse("{ a }")

		visitor.Walk(doc, nil, visitor.NewNodeVisitor(
			visitor.NodeVisitActionFunc(func(node ast.Node, ctx interface{}) visitor.Result {
				visitedNodes = append(visitedNodes, [2]interface{}{
					nodeKind(node),
					nodeValue(node),
				})
				return visitor.Continue
			})))

		Expect(visitedNodes).Should(Equal([][2]interface{}{
			{"Document", nil},
			{"Definitions", nil},
			{"OperationDefinition", nil},
			{"SelectionSet", nil},
			{"Field", nil},
			{"Name", "a"},
		}))
	})

	It("allows skipping a sub-tree", func() {
		var visited [][2]interface{}

		doc := parse("{ a, b { x }, c }")

		visitor.Walk(doc, nil, visitor.NewNodeVisitor(
			visitor.NodeVisitActionFunc(func(node ast.Node, ctx interface{}) visitor.Result {
				visited = append(visited, [2]interface{}{
					nodeKind(node),
					nodeValue(node),
				})

				if node, ok := node.(*ast.Field); ok && node.Name.Value() == "b" {
					return visitor.SkipSubTree
				}

				return visitor.Continue
			})))

		Expect(visited).Should(Equal([][2]interface{}{
			{"Document", nil},
			{"Definitions", nil},
			{"OperationDefinition", nil},
			{"SelectionSet", nil},
			{"Field", nil},
			{"Name", "a"},
			{"Field", nil},
			{"Field", nil},
			{"Name", "c"},
		}))
	})

	It("allows early exit while visiting", func() {
		var visited [][2]interface{}

		doc := parse("{ a, b { x }, c }")

		visitor.Walk(doc, nil, visitor.NewNodeVisitor(
			visitor.NodeVisitActionFunc(func(node ast.Node, ctx interface{}) visitor.Result {
				visited = append(visited, [2]interface{}{
					nodeKind(node), nodeValue(node),
				})

				if node, ok := node.(ast.Name); ok && node.Value() == "x" {
					return visitor.Break
				}

				return visitor.Continue
			})))

		Expect(visited).Should(Equal([][2]interface{}{
			{"Document", nil},
			{"Definitions", nil},
			{"OperationDefinition", nil},
			{"SelectionSet", nil},
			{"Field", nil},
			{"Name", "a"},
			{"Field", nil},
			{"Name", "b"},
			{"SelectionSet", nil},
			{"Field", nil},
			{"Name", "x"},
		}))
	})

	It("Experimental: visits variables defined in fragments", func() {
		var visited [][2]interface{}

		doc, err := parser.Parse(
			token.NewSource("fragment a($v: Boolean = false) on t { f }"),
			parser.EnableFragmentVariables())
		Expect(err).ShouldNot(HaveOccurred())

		visitor.Walk(doc, nil, visitor.NewNodeVisitor(
			visitor.NodeVisitActionFunc(func(node ast.Node, ctx interface{}) visitor.Result {
				visited = append(visited, [2]interface{}{
					nodeKind(node),
					nodeValue(node),
				})
				return visitor.Continue
			})))

		Expect(visited).Should(Equal([][2]interface{}{
			{"Document", nil},
			{"Definitions", nil},
			{"FragmentDefinition", nil},
			{"Name", "a"},
			{"VariableDefinitions", nil},
			{"VariableDefinition", nil},
			{"Variable", nil},
			{"Name", "v"},
			{"NamedType", nil},
			{"Name", "Boolean"},
			{"BooleanValue", false},
			{"NamedType", nil},
			{"Name", "t"},
			{"SelectionSet", nil},
			{"Field", nil},
			{"Name", "f"},
		}))
	})

	It("visits kitchen sink", func() {
		kitchenSink, err := ioutil.ReadFile("../../parser/kitchen-sink.graphql")
		Expect(err).ShouldNot(HaveOccurred())

		doc, err := parser.Parse(token.NewSourceFromBytes(kitchenSink))
		Expect(err).ShouldNot(HaveOccurred())

		var visited [][2]interface{}

		visitor.Walk(doc, nil, visitor.NewNodeVisitor(
			visitor.NodeVisitActionFunc(func(node ast.Node, ctx interface{}) visitor.Result {
				visited = append(visited, [2]interface{}{
					nodeKind(node),
					nodeValue(node),
				})
				return visitor.Continue
			})))

		Expect(visited).Should(Equal([][2]interface{}{
			{"Document", nil},
			{"Definitions", nil},
			{"OperationDefinition", nil},
			{"Name", "queryName"},
			{"VariableDefinitions", nil},
			{"VariableDefinition", nil},
			{"Variable", nil},
			{"Name", "foo"},
			{"NamedType", nil},
			{"Name", "ComplexType"},
			{"VariableDefinition", nil},
			{"Variable", nil},
			{"Name", "site"},
			{"NamedType", nil},
			{"Name", "Site"},
			{"EnumValue", "MOBILE"},
			{"Directives", nil},
			{"Directive", nil},
			{"Name", "onQuery"},
			{"SelectionSet", nil},
			{"Field", nil},
			{"Name", "whoever123is"},
			{"Name", "node"},
			{"Arguments", nil},
			{"Argument", nil},
			{"Name", "id"},
			{"ListValue", nil},
			{"IntValue", int32(123)},
			{"IntValue", int32(456)},
			{"SelectionSet", nil},
			{"Field", nil},
			{"Name", "id"},
			{"InlineFragment", nil},
			{"NamedType", nil},
			{"Name", "User"},
			{"Directives", nil},
			{"Directive", nil},
			{"Name", "onInlineFragment"},
			{"SelectionSet", nil},
			{"Field", nil},
			{"Name", "field2"},
			{"SelectionSet", nil},
			{"Field", nil},
			{"Name", "id"},
			{"Field", nil},
			{"Name", "alias"},
			{"Name", "field1"},
			{"Arguments", nil},
			{"Argument", nil},
			{"Name", "first"},
			{"IntValue", int32(10)},
			{"Argument", nil},
			{"Name", "after"},
			{"Variable", nil},
			{"Name", "foo"},
			{"Directives", nil},
			{"Directive", nil},
			{"Name", "include"},
			{"Arguments", nil},
			{"Argument", nil},
			{"Name", "if"},
			{"Variable", nil},
			{"Name", "foo"},
			{"SelectionSet", nil},
			{"Field", nil},
			{"Name", "id"},
			{"FragmentSpread", nil},
			{"Name", "frag"},
			{"Directives", nil},
			{"Directive", nil},
			{"Name", "onFragmentSpread"},
			{"InlineFragment", nil},
			{"Directives", nil},
			{"Directive", nil},
			{"Name", "skip"},
			{"Arguments", nil},
			{"Argument", nil},
			{"Name", "unless"},
			{"Variable", nil},
			{"Name", "foo"},
			{"SelectionSet", nil},
			{"Field", nil},
			{"Name", "id"},
			{"InlineFragment", nil},
			{"SelectionSet", nil},
			{"Field", nil},
			{"Name", "id"},
			{"OperationDefinition", nil},
			{"Name", "likeStory"},
			{"Directives", nil},
			{"Directive", nil},
			{"Name", "onMutation"},
			{"SelectionSet", nil},
			{"Field", nil},
			{"Name", "like"},
			{"Arguments", nil},
			{"Argument", nil},
			{"Name", "story"},
			{"IntValue", int32(123)},
			{"Directives", nil},
			{"Directive", nil},
			{"Name", "onField"},
			{"SelectionSet", nil},
			{"Field", nil},
			{"Name", "story"},
			{"SelectionSet", nil},
			{"Field", nil},
			{"Name", "id"},
			{"Directives", nil},
			{"Directive", nil},
			{"Name", "onField"},
			{"OperationDefinition", nil},
			{"Name", "StoryLikeSubscription"},
			{"VariableDefinitions", nil},
			{"VariableDefinition", nil},
			{"Variable", nil},
			{"Name", "input"},
			{"NamedType", nil},
			{"Name", "StoryLikeSubscribeInput"},
			{"Directives", nil},
			{"Directive", nil},
			{"Name", "onSubscription"},
			{"SelectionSet", nil},
			{"Field", nil},
			{"Name", "storyLikeSubscribe"},
			{"Arguments", nil},
			{"Argument", nil},
			{"Name", "input"},
			{"Variable", nil},
			{"Name", "input"},
			{"SelectionSet", nil},
			{"Field", nil},
			{"Name", "story"},
			{"SelectionSet", nil},
			{"Field", nil},
			{"Name", "likers"},
			{"SelectionSet", nil},
			{"Field", nil},
			{"Name", "count"},
			{"Field", nil},
			{"Name", "likeSentence"},
			{"SelectionSet", nil},
			{"Field", nil},
			{"Name", "text"},
			{"FragmentDefinition", nil},
			{"Name", "frag"},
			{"NamedType", nil},
			{"Name", "Friend"},
			{"Directives", nil},
			{"Directive", nil},
			{"Name", "onFragmentDefinition"},
			{"SelectionSet", nil},
			{"Field", nil},
			{"Name", "foo"},
			{"Arguments", nil},
			{"Argument", nil},
			{"Name", "size"},
			{"Variable", nil},
			{"Name", "size"},
			{"Argument", nil},
			{"Name", "bar"},
			{"Variable", nil},
			{"Name", "b"},
			{"Argument", nil},
			{"Name", "obj"},
			{"ObjectValue", nil},
			{"ObjectField", nil},
			{"Name", "key"},
			{"StringValue", "value"},
			{"ObjectField", nil},
			{"Name", "block"},
			{"StringValue", `block string uses """`},
			{"OperationDefinition", nil},
			{"SelectionSet", nil},
			{"Field", nil},
			{"Name", "unnamed"},
			{"Arguments", nil},
			{"Argument", nil},
			{"Name", "truthy"},
			{"BooleanValue", true},
			{"Argument", nil},
			{"Name", "falsey"},
			{"BooleanValue", false},
			{"Argument", nil},
			{"Name", "nullish"},
			{"NullValue", nil},
			{"Field", nil},
			{"Name", "query"},
			{"OperationDefinition", nil},
			{"SelectionSet", nil},
			{"Field", nil},
			{"Name", "__typename"},
		}))
	})
})
