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

package graphql_test

import (
	"github.com/botobag/artemis/graphql"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Directive", func() {
	It("accepts a directive with locations", func() {
		directive := graphql.MustNewDirective(&graphql.DirectiveConfig{
			Name: "DirectiveWithLocation",
			Locations: []graphql.DirectiveLocation{
				graphql.DirectiveLocationField,
				graphql.DirectiveLocationFragmentSpread,
				graphql.DirectiveLocationInlineFragment,
			},
		})

		Expect(directive.Name()).Should(Equal("DirectiveWithLocation"))
		Expect(directive.Description()).Should(Equal(""))
		Expect(directive.Locations()).Should(Equal([]graphql.DirectiveLocation{
			graphql.DirectiveLocationField,
			graphql.DirectiveLocationFragmentSpread,
			graphql.DirectiveLocationInlineFragment,
		}))
		Expect(directive.Args()).Should(BeEmpty())
	})

	It("accepts a directive with arguments", func() {
		directive := graphql.MustNewDirective(&graphql.DirectiveConfig{
			Name:        "DirectiveWithArguments",
			Description: "Test directive with arguments",
			Args: graphql.ArgumentConfigMap{
				"test": graphql.ArgumentConfig{
					Type:         graphql.T(graphql.Boolean()),
					Description:  "this is a test argument",
					DefaultValue: true,
				},
			},
		})

		Expect(directive.Name()).Should(Equal("DirectiveWithArguments"))
		Expect(directive.Description()).Should(Equal("Test directive with arguments"))
		Expect(directive.Locations()).Should(BeEmpty())
		Expect(directive.Args()).Should(Equal([]graphql.Argument{
			graphql.MockArgument(
				"test",
				"this is a test argument",
				graphql.Boolean(),
				true,
			),
		}))
	})

	It("accepts a directive without locations and arguments", func() {
		directive := graphql.MustNewDirective(&graphql.DirectiveConfig{
			Name: "SimpleDirective",
		})
		Expect(directive.Name()).Should(Equal("SimpleDirective"))
		Expect(directive.Description()).Should(Equal(""))
		Expect(directive.Locations()).Should(BeEmpty())
		Expect(directive.Args()).Should(BeEmpty())
	})

	It("rejects creating a directive without name", func() {
		_, err := graphql.NewDirective(&graphql.DirectiveConfig{
			Name: "",
		})
		Expect(err).Should(MatchError("Must provide name for Directive."))

		Expect(func() {
			graphql.MustNewDirective(&graphql.DirectiveConfig{})
		}).Should(Panic())
	})
})
