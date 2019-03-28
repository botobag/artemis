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

// graphql-js/src/validation/__tests__/UniqueInputFieldNames-test.js@8c96dc8
var _ = Describe("Validate: Unique input field names", func() {
	expectErrors := func(queryStr string) GomegaAssertion {
		return expectValidationErrors(rules.UniqueInputFieldNames{}, queryStr)
	}

	expectValid := func(queryStr string) {
		expectErrors(queryStr).Should(Equal(graphql.NoErrors()))
	}

	duplicateField := func(
		fieldName string,
		l1 uint, c1 uint,
		l2 uint, c2 uint,
	) error {

		return graphql.NewError(
			validator.DuplicateInputFieldMessage(fieldName),
			[]graphql.ErrorLocation{
				{Line: l1, Column: c1},
				{Line: l2, Column: c2},
			},
		)
	}

	It("input object with fields", func() {
		expectValid(`
      {
        field(arg: { f: true })
      }
    `)
	})

	It("same input object within two args", func() {
		expectValid(`
      {
        field(arg1: { f: true }, arg2: { f: true })
      }
    `)
	})

	It("multiple input object fields", func() {
		expectValid(`
      {
        field(arg: { f1: "value", f2: "value", f3: "value" })
      }
    `)
	})

	It("allows for nested input objects with similar fields", func() {
		expectValid(`
      {
        field(arg: {
          deep: {
            deep: {
              id: 1
            }
            id: 1
          }
          id: 1
        })
      }
    `)
	})

	It("duplicate input object fields", func() {
		expectErrors(`
      {
        field(arg: { f1: "value", f1: "value" })
      }
    `).Should(Equal(graphql.ErrorsOf(duplicateField("f1", 3, 22, 3, 35))))
	})

	It("many duplicate input object fields", func() {
		expectErrors(`
      {
        field(arg: { f1: "value", f1: "value", f1: "value" })
      }
    `).Should(Equal(graphql.ErrorsOf(
			duplicateField("f1", 3, 22, 3, 35),
			duplicateField("f1", 3, 22, 3, 48),
		)))
	})

	It("nested duplicate input object fields", func() {
		expectErrors(`
      {
        field(arg: { f1: {f2: "value", f2: "value" }})
      }
    `).Should(Equal(graphql.ErrorsOf(duplicateField("f2", 3, 27, 3, 40))))
	})
})
