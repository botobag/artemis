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

package ast_test

import (
	"io/ioutil"

	"github.com/botobag/artemis/graphql/ast"
	"github.com/botobag/artemis/graphql/parser"
	"github.com/botobag/artemis/graphql/token"
	"github.com/botobag/artemis/internal/util"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func parse(s string, options ...parser.ParseOption) ast.Node {
	ast, err := parser.Parse(token.NewSource(&token.SourceConfig{
		Body: token.SourceBody(s),
	}), options...)
	Expect(err).ShouldNot(HaveOccurred())
	return ast
}

func kitchenSinkAST() ast.Node {
	kitchenSink, err := ioutil.ReadFile("../parser/kitchen-sink.graphql")
	Expect(err).ShouldNot(HaveOccurred())
	return parse(string(kitchenSink))
}

var _ = Describe("Printer: Query document", func() {
	// graphql-js/src/language/__tests__/printer-test.js@8c96dc8
	It("does not alter ast", func() {
		kitchenSink := kitchenSinkAST()
		_ = ast.Print(kitchenSink)
		Expect(kitchenSink).Should(Equal(kitchenSinkAST()))
	})

	It("prints minimal ast", func() {
		astNode := &ast.Field{
			Name: ast.Name{
				Token: &token.Token{
					Kind:  token.KindName,
					Value: "foo",
				},
			},
		}
		Expect(ast.Print(astNode)).Should(Equal("foo"))
	})

	// It("produces helpful error messages", func() {
	// })

	It("correctly prints non-query operations without name", func() {
		queryASTShorthanded := parse("query { id, name }")
		Expect(ast.Print(queryASTShorthanded)).Should(Equal(util.Dedent(`
			{
			  id
			  name
			}
		`)))

		mutationAST := parse("mutation { id, name }")
		Expect(ast.Print(mutationAST)).Should(Equal(util.Dedent(`
			mutation {
			  id
			  name
			}
		`)))

		queryASTWithArtifacts := parse("query ($foo: TestType) @testDirective { id, name }")
		Expect(ast.Print(queryASTWithArtifacts)).Should(Equal(util.Dedent(`
			query ($foo: TestType) @testDirective {
			  id
			  name
			}
		`)))

		mutationASTWithArtifacts := parse("mutation ($foo: TestType) @testDirective { id, name }")
		Expect(ast.Print(mutationASTWithArtifacts)).Should(Equal(util.Dedent(`
			mutation ($foo: TestType) @testDirective {
			  id
			  name
			}
		`)))
	})

	It("prints query with variable directives", func() {
		queryASTWithVariableDirective := parse(
			"query ($foo: TestType = {a: 123} @testDirective(if: true) @test) { id }",
		)
		Expect(ast.Print(queryASTWithVariableDirective)).Should(Equal(util.Dedent(`
			query ($foo: TestType = {a: 123} @testDirective(if: true) @test) {
			  id
			}
		`)))
	})

	It("Experimental: prints fragment with variable directives", func() {
		queryASTWithVariableDirective := parse(
			"fragment Foo($foo: TestType @test) on TestType @testDirective { id }",
			parser.EnableFragmentVariables(),
		)
		Expect(ast.Print(queryASTWithVariableDirective)).Should(Equal(util.Dedent(`
			fragment Foo($foo: TestType @test) on TestType @testDirective {
			  id
			}
		`)))
	})

	It("prints kitchen sink", func() {
		printed := ast.Print(kitchenSinkAST())

		Expect(printed).Should(Equal(util.Dedent(`
			query queryName($foo: ComplexType, $site: Site = MOBILE) @onQuery {
			  whoever123is: node(id: [123, 456]) {
			    id
			    ... on User @onInlineFragment {
			      field2 {
			        id
			        alias: field1(first: 10, after: $foo) @include(if: $foo) {
			          id
			          ...frag @onFragmentSpread
			        }
			      }
			    }
			    ... @skip(unless: $foo) {
			      id
			    }
			    ... {
			      id
			    }
			  }
			}

			mutation likeStory @onMutation {
			  like(story: 123) @onField {
			    story {
			      id @onField
			    }
			  }
			}

			subscription StoryLikeSubscription($input: StoryLikeSubscribeInput) @onSubscription {
			  storyLikeSubscribe(input: $input) {
			    story {
			      likers {
			        count
			      }
			      likeSentence {
			        text
			      }
			    }
			  }
			}

			fragment frag on Friend @onFragmentDefinition {
			  foo(size: $size, bar: $b, obj: {key: "value", block: """
			    block string uses \"""
			  """})
			}

			{
			  unnamed(truthy: true, falsey: false, nullish: null)
			  query
			}

			{
			  __typename
			}
		`)))
	})
})
