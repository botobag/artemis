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

// graphql-js/src/validation/__tests__/UniqueArgumentNames-test.js@8c96dc8
var _ = Describe("Validate: Unique argument names", func() {
	expectErrors := func(queryStr string) GomegaAssertion {
		return expectValidationErrors(rules.UniqueArgumentNames{}, queryStr)
	}

	expectValid := func(queryStr string) {
		expectErrors(queryStr).Should(Equal(graphql.NoErrors()))
	}

	duplicateArg := func(argName string, l1 uint, c1 uint, l2 uint, c2 uint) error {
		return graphql.NewError(
			validator.DuplicateArgMessage(argName),
			[]graphql.ErrorLocation{
				{Line: l1, Column: c1},
				{Line: l2, Column: c2},
			},
		)
	}

	It("no arguments on field", func() {
		expectValid(`
      {
        field
      }
    `)
	})

	It("no arguments on directive", func() {
		expectValid(`
      {
        field @directive
      }
    `)
	})

	It("argument on field", func() {
		expectValid(`
      {
        field(arg: "value")
      }
    `)
	})

	It("argument on directive", func() {
		expectValid(`
      {
        field @directive(arg: "value")
      }
    `)
	})

	It("same argument on two fields", func() {
		expectValid(`
      {
        one: field(arg: "value")
        two: field(arg: "value")
      }
    `)
	})

	It("same argument on field and directive", func() {
		expectValid(`
      {
        field(arg: "value") @directive(arg: "value")
      }
    `)
	})

	It("same argument on two directives", func() {
		expectValid(`
      {
        field @directive1(arg: "value") @directive2(arg: "value")
      }
    `)
	})

	It("multiple field arguments", func() {
		expectValid(`
      {
        field(arg1: "value", arg2: "value", arg3: "value")
      }
    `)
	})

	It("multiple directive arguments", func() {
		expectValid(`
      {
        field @directive(arg1: "value", arg2: "value", arg3: "value")
      }
    `)
	})

	It("duplicate field arguments", func() {
		expectErrors(`
      {
        field(arg1: "value", arg1: "value")
      }
    `).Should(Equal(graphql.ErrorsOf(
			duplicateArg("arg1", 3, 15, 3, 30),
		)))
	})

	It("many duplicate field arguments", func() {
		expectErrors(`
      {
        field(arg1: "value", arg1: "value", arg1: "value")
      }
    `).Should(Equal(graphql.ErrorsOf(
			duplicateArg("arg1", 3, 15, 3, 30),
			duplicateArg("arg1", 3, 15, 3, 45),
		)))
	})

	It("duplicate directive arguments", func() {
		expectErrors(`
      {
        field @directive(arg1: "value", arg1: "value")
      }
    `).Should(Equal(graphql.ErrorsOf(
			duplicateArg("arg1", 3, 26, 3, 41),
		)))
	})

	It("many duplicate directive arguments", func() {
		expectErrors(`
      {
        field @directive(arg1: "value", arg1: "value", arg1: "value")
      }
    `).Should(Equal(graphql.ErrorsOf(
			duplicateArg("arg1", 3, 26, 3, 41),
			duplicateArg("arg1", 3, 26, 3, 56),
		)))
	})
})
