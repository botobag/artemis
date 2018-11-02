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

var _ = Describe("OrList", func() {
	It("accepts an empty list", func() {
		Expect(util.OrList(nil, 5, false)).Should(BeEmpty())
		Expect(util.OrList([]string{}, 5, false)).Should(BeEmpty())
	})

	It("returns single item", func() {
		Expect(util.OrList([]string{"A"}, 5, false)).Should(Equal("A"))
		Expect(util.OrList([]string{"A"}, 5, true)).Should(Equal(`"A"`))
	})

	It("returns two item list", func() {
		Expect(util.OrList([]string{"A", "B"}, 5, false)).Should(Equal("A or B"))
		Expect(util.OrList([]string{"A", "B"}, 5, true)).Should(Equal(`"A" or "B"`))
	})

	It("returns comma separated many item list", func() {
		Expect(util.OrList([]string{"A", "B", "C"}, 5, false)).Should(Equal("A, B, or C"))
		Expect(util.OrList([]string{"A", "B", "C"}, 5, true)).Should(Equal(`"A", "B", or "C"`))
	})

	It("limits to five items", func() {
		Expect(util.OrList([]string{"A", "B", "C", "D", "E", "F"}, 5, false)).Should(Equal("A, B, C, D, or E"))
		Expect(util.OrList([]string{"A", "B", "C", "D", "E", "F"}, 5, true)).Should(Equal(`"A", "B", "C", "D", or "E"`))
	})
})
