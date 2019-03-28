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
var _ = Describe("Validate: Directives in valid locations", func() {
	expectErrors := func(queryStr string) GomegaAssertion {
		return expectValidationErrors(rules.DirectivesInValidLocations{}, queryStr)
	}

	expectValid := func(queryStr string) {
		expectErrors(queryStr).Should(Equal(graphql.NoErrors()))
	}

	misplacedDirective := func(
		directiveName string,
		placement graphql.DirectiveLocation,
		line uint,
		column uint) error {

		return graphql.NewError(
			validator.MisplacedDirectiveMessage(directiveName, placement),
			[]graphql.ErrorLocation{
				{Line: line, Column: column},
			},
		)
	}

	It("with well placed directives", func() {
		expectValid(`
      query Foo($var: Boolean) @onQuery {
        name @include(if: $var)
        ...Frag @include(if: true)
        skippedField @skip(if: true)
        ...SkippedFrag @skip(if: true)
      }

      mutation Bar @onMutation {
        someField
      }
    `)
	})

	It("with well placed variable definition directive", func() {
		expectValid(`
      query Foo($var: Boolean @onVariableDefinition) {
        name
      }
    `)
	})

	It("with misplaced directives", func() {
		expectErrors(`
      query Foo($var: Boolean) @include(if: true) {
        name @onQuery @include(if: $var)
        ...Frag @onQuery
      }

      mutation Bar @onQuery {
        someField
      }
    `).Should(Equal(graphql.ErrorsOf(
			misplacedDirective("include", "QUERY", 2, 32),
			misplacedDirective("onQuery", "FIELD", 3, 14),
			misplacedDirective("onQuery", "FRAGMENT_SPREAD", 4, 17),
			misplacedDirective("onQuery", "MUTATION", 7, 20),
		)))
	})

	It("with misplaced variable definition directive", func() {
		expectErrors(`
      query Foo($var: Boolean @onField) {
        name
      }
    `).Should(Equal(graphql.ErrorsOf(
			misplacedDirective("onField", "VARIABLE_DEFINITION", 2, 31),
		)))
	})
})
