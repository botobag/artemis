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

// graphql-js/src/validation/__tests__/UniqueVariableNames-test.js@8c96dc8
var _ = Describe("Validate: Unique operation names", func() {
	expectErrors := func(queryStr string) GomegaAssertion {
		return expectValidationErrors(rules.UniqueVariableNames{}, queryStr)
	}

	expectValid := func(queryStr string) {
		expectErrors(queryStr).Should(Equal(graphql.NoErrors()))
	}

	duplicateVariable := func(name string, l1 uint, c1 uint, l2 uint, c2 uint) error {
		return graphql.NewError(validator.DuplicateVariableMessage(name), []graphql.ErrorLocation{
			{Line: l1, Column: c1},
			{Line: l2, Column: c2},
		})
	}

	It("unique variable names", func() {
		expectValid(`
      query A($x: Int, $y: String) { __typename }
      query B($x: String, $y: Int) { __typename }
    `)
	})

	It("duplicate variable names", func() {
		expectErrors(`
      query A($x: Int, $x: Int, $x: String) { __typename }
      query B($x: String, $x: Int) { __typename }
      query C($x: Int, $x: Int) { __typename }
    `).Should(Equal(graphql.ErrorsOf(
			duplicateVariable("x", 2, 16, 2, 25),
			duplicateVariable("x", 2, 16, 2, 34),
			duplicateVariable("x", 3, 16, 3, 28),
			duplicateVariable("x", 4, 16, 4, 25),
		)))
	})
})
