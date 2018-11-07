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

var _ = Describe("SnakeCase", func() {
	It("converts strng to snake_case", func() {
		testcases := map[string]string{
			"":           "",
			"a":          "a",
			"foo":        "foo",
			"A":          "a",
			"FOO":        "foo",
			"SnakeCase":  "snake_case",
			"FooBar":     "foo_bar",
			"Foo_Bar":    "foo_bar",
			"foo_bar":    "foo_bar",
			"foo_bar_":   "foo_bar_",
			"_foo_bar":   "_foo_bar",
			"_foo_bar_":  "_foo_bar_",
			"___foo_bar": "___foo_bar",
			"foo___bar":  "foo___bar",
			"foo_bar___": "foo_bar___",
			"foo1_bar2":  "foo1_bar2",
			"fooD":       "foo_d",
			"foOD":       "fo_od",
		}

		for s, expected := range testcases {
			Expect(util.SnakeCase(s)).Should(Equal(expected), "%s", s)
		}
	})
})
