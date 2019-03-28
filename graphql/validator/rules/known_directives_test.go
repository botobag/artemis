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

package rules_test

import (
	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/graphql/internal/validator"
	"github.com/botobag/artemis/graphql/validator/rules"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// graphql-js/src/validation/__tests__/KnownDirectives-test.js@8c96dc8
var _ = Describe("Validate: Known directives", func() {
	expectErrors := func(queryStr string) GomegaAssertion {
		return expectValidationErrors(rules.KnownDirectives{}, queryStr)
	}

	expectValid := func(queryStr string) {
		expectErrors(queryStr).Should(Equal(graphql.NoErrors()))
	}

	unknownDirective := func(
		directiveName string,
		line uint,
		column uint) error {

		return graphql.NewError(
			validator.UnknownDirectiveMessage(directiveName),
			[]graphql.ErrorLocation{
				{Line: line, Column: column},
			},
		)
	}

	It("with no directives", func() {
		expectValid(`
      query Foo {
        name
        ...Frag
      }

      fragment Frag on Dog {
        name
      }
    `)
	})

	It("with known directives", func() {
		expectValid(`
      {
        dog @include(if: true) {
          name
        }
        human @skip(if: false) {
          name
        }
      }
    `)
	})

	It("with unknown directive", func() {
		expectErrors(`
      {
        dog @unknown(directive: "value") {
          name
        }
      }
    `).Should(Equal(graphql.ErrorsOf(unknownDirective("unknown", 3, 13))))
	})

	It("with many unknown directives", func() {
		expectErrors(`
      {
        dog @unknown(directive: "value") {
          name
        }
        human @unknown(directive: "value") {
          name
          pets @unknown(directive: "value") {
            name
          }
        }
      }
    `).Should(Equal(graphql.ErrorsOf(
			unknownDirective("unknown", 3, 13),
			unknownDirective("unknown", 6, 15),
			unknownDirective("unknown", 8, 16),
		)))
	})
})
