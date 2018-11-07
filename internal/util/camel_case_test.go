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

var _ = Describe("CamelCase", func() {
	It("converts strng to CamelCase", func() {
		testcases := map[string]string{
			"":           "",
			"a":          "A",
			"foo":        "Foo",
			"A":          "A",
			"FOO":        "FOO",
			"CamelCase":  "CamelCase",
			"Foo_Bar":    "FooBar",
			"foo_bar":    "FooBar",
			"foo_bar_":   "FooBar",
			"_foo_bar":   "FooBar",
			"_foo_bar_":  "FooBar",
			"___foo_bar": "FooBar",
			"foo___bar":  "FooBar",
			"foo_bar___": "FooBar",
			"foo1_bar2":  "Foo1Bar2",
		}

		for s, expected := range testcases {
			Expect(util.CamelCase(s)).Should(Equal(expected), "%s", s)
		}
	})
})
