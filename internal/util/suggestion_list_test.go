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

package util_test

import (
	"github.com/botobag/artemis/internal/util"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SuggestionList", func() {
	It("returns results when input is empty", func() {
		Expect(util.SuggestionList("", []string{"a"})).Should(Equal([]string{"a"}))
	})

	It("returns empty array when there are no options", func() {
		Expect(util.SuggestionList("input", []string{""})).Should(BeEmpty())
		Expect(util.SuggestionList("input", nil)).Should(BeEmpty())
	})

	It("returns options sorted based on similarity", func() {
		Expect(util.SuggestionList("abc", []string{"a", "ab", "abc"})).Should(Equal([]string{"abc", "ab"}))
	})

	It("considers case changes as a single edit", func() {
		// Though 3 characters in "ABC" are all different than "abc", it has lower distance than "a"
		// (which is 2) because case change is specially treated as 1.
		Expect(util.SuggestionList("abc", []string{"a", "ABC"})).Should(Equal([]string{"ABC"}))
	})

	It("considers a swap of two adjacent characters as distance 1", func() {
		// distance("abcd", "badc") is 2 because we only need 2 swaps to turns "badc" to "abcd".
		Expect(util.SuggestionList("abcd", []string{"badc", "ab"})).Should(Equal([]string{"badc", "ab"}))
	})
})
