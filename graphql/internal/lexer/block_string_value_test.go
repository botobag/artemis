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

package lexer_test

import (
	"strings"

	"github.com/botobag/artemis/graphql/internal/lexer"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// graphql-js/src/language/__tests__/blockStringValue-test.js
var _ = Describe("blockStringValue", func() {
	It("removes uniform indentation from a string", func() {
		rawValue := strings.Join([]string{
			``,
			`    Hello,`,
			`      World!`,
			``,
			`    Yours,`,
			`      GraphQL.`,
		}, "\n")

		Expect(lexer.BlockStringValue(rawValue)).Should(Equal(strings.Join([]string{
			"Hello,",
			"  World!",
			"",
			"Yours,",
			"  GraphQL.",
		}, "\n")))
	})

	It("removes empty leading and trailing lines", func() {
		rawValue := strings.Join([]string{
			``,
			``,
			`    Hello,`,
			`      World!`,
			``,
			`    Yours,`,
			`      GraphQL.`,
			``,
			``,
		}, "\n")

		Expect(lexer.BlockStringValue(rawValue)).Should(Equal(strings.Join([]string{
			"Hello,",
			"  World!",
			"", "Yours,",
			"  GraphQL.",
		}, "\n")))
	})

	It("removes blank leading and trailing lines", func() {
		rawValue := strings.Join([]string{
			`  `,
			`        `,
			`    Hello,`,
			`      World!`,
			``,
			`    Yours,`,
			`      GraphQL.`,
			`        `,
			`  `,
		}, "\n")

		Expect(lexer.BlockStringValue(rawValue)).Should(Equal(strings.Join([]string{
			"Hello,",
			"  World!",
			"",
			"Yours,",
			"  GraphQL.",
		}, "\n")))
	})

	It("retains indentation from first line", func() {
		rawValue := strings.Join([]string{
			`    Hello,`,
			`      World!`,
			``,
			`    Yours,`,
			`      GraphQL.`,
		}, "\n")

		Expect(lexer.BlockStringValue(rawValue)).Should(Equal(strings.Join([]string{
			"    Hello,",
			"  World!",
			"",
			"Yours,",
			"  GraphQL.",
		}, "\n")))
	})

	It("does not alter trailing spaces", func() {
		rawValue := strings.Join([]string{
			`               `,
			`    Hello,     `,
			`      World!   `,
			`               `,
			`    Yours,     `,
			`      GraphQL. `,
			`               `,
		}, "\n")

		Expect(lexer.BlockStringValue(rawValue)).Should(Equal(strings.Join([]string{
			"Hello,     ",
			"  World!   ",
			"           ",
			"Yours,     ",
			"  GraphQL. ",
		}, "\n")))
	})
})
