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

// graphql-js/src/validation/__tests__/UniqueDirectivesPerLocation-test.js@8c96dc8
var _ = Describe("Validate: Directives Are Unique Per Location", func() {
	expectErrors := func(queryStr string) GomegaAssertion {
		return expectValidationErrors(rules.UniqueDirectivesPerLocation{}, queryStr)
	}

	expectValid := func(queryStr string) {
		expectErrors(queryStr).Should(Equal(graphql.NoErrors()))
	}

	duplicateDirective := func(
		directiveName string,
		l1 uint, c1 uint,
		l2 uint, c2 uint) error {

		return graphql.NewError(
			validator.DuplicateDirectiveMessage(directiveName),
			[]graphql.ErrorLocation{
				{Line: l1, Column: c1},
				{Line: l2, Column: c2},
			},
		)
	}

	It("no directives", func() {
		expectValid(`
      fragment Test on Type {
        field
      }
    `)
	})

	It("unique directives in different locations", func() {
		expectValid(`
      fragment Test on Type @directiveA {
        field @directiveB
      }
    `)
	})

	It("unique directives in same locations", func() {
		expectValid(`
      fragment Test on Type @directiveA @directiveB {
        field @directiveA @directiveB
      }
    `)
	})

	It("same directives in different locations", func() {
		expectValid(`
      fragment Test on Type @directiveA {
        field @directiveA
      }
    `)
	})

	It("same directives in similar locations", func() {
		expectValid(`
      fragment Test on Type {
        field @directive
        field @directive
      }
    `)
	})

	It("duplicate directives in one location", func() {
		expectErrors(`
      fragment Test on Type {
        field @directive @directive
      }
    `).Should(Equal(graphql.ErrorsOf(duplicateDirective("directive", 3, 15, 3, 26))))
	})

	It("many duplicate directives in one location", func() {
		expectErrors(`
      fragment Test on Type {
        field @directive @directive @directive
      }
    `).Should(Equal(graphql.ErrorsOf(
			duplicateDirective("directive", 3, 15, 3, 26),
			duplicateDirective("directive", 3, 15, 3, 37),
		)))
	})

	It("different duplicate directives in one location", func() {
		expectErrors(`
      fragment Test on Type {
        field @directiveA @directiveB @directiveA @directiveB
      }
    `).Should(Equal(graphql.ErrorsOf(
			duplicateDirective("directiveA", 3, 15, 3, 39),
			duplicateDirective("directiveB", 3, 27, 3, 51),
		)))
	})

	It("duplicate directives in many locations", func() {
		expectErrors(`
      fragment Test on Type @directive @directive {
        field @directive @directive
      }
    `).Should(Equal(graphql.ErrorsOf(
			duplicateDirective("directive", 2, 29, 2, 40),
			duplicateDirective("directive", 3, 15, 3, 26),
		)))
	})
})
