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

package validator_test

import (
	"github.com/botobag/artemis/graphql/internal/validator"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Validation Message", func() {

	// UndefinedFieldMessage
	//
	// graphql-js/src/validation/__tests__/FieldsOnCorrectType-test.js@8c96dc8
	Describe("Fields on correct type error message", func() {
		It("Works with no suggestions", func() {
			Expect(validator.UndefinedFieldMessage("f", "T", nil, nil)).Should(Equal(
				`Cannot query field "f" on type "T".`,
			))
		})

		It("Works with no small numbers of type suggestions", func() {
			Expect(validator.UndefinedFieldMessage("f", "T", []string{"A", "B"}, nil)).Should(Equal(
				`Cannot query field "f" on type "T". Did you mean to use an inline fragment on "A" or "B"?`,
			))
		})

		It("Works with no small numbers of field suggestions", func() {
			Expect(validator.UndefinedFieldMessage("f", "T", nil, []string{"z", "y"})).Should(Equal(
				`Cannot query field "f" on type "T". Did you mean "z" or "y"?`,
			))
		})

		It("Only shows one set of suggestions at a time, preferring types", func() {
			Expect(validator.UndefinedFieldMessage("f", "T", []string{"A", "B"}, []string{"z", "y"})).Should(Equal(
				`Cannot query field "f" on type "T". Did you mean to use an inline fragment on "A" or "B"?`,
			))
		})

		It("Limits lots of type suggestions", func() {
			Expect(
				validator.UndefinedFieldMessage("f", "T", []string{"A", "B", "C", "D", "E", "F"}, nil),
			).Should(Equal(
				`Cannot query field "f" on type "T". Did you mean to use an inline fragment on "A", "B", "C", "D", or "E"?`,
			))
		})

		It("Limits lots of field suggestions", func() {
			Expect(
				validator.UndefinedFieldMessage("f", "T", nil, []string{"z", "y", "x", "w", "v", "u"}),
			).Should(Equal(
				`Cannot query field "f" on type "T". Did you mean "z", "y", "x", "w", or "v"?`,
			))
		})
	})
})
