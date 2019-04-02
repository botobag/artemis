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

package token_test

import (
	"github.com/botobag/artemis/graphql/token"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Token", func() {
	Describe("Magic SOF Token", func() {
		var source *token.Source

		BeforeEach(func() {
			source = token.NewSourceFromBytes(nil, token.SourceName("Test Magic SOF Token Source"))
		})

		It("can finds its Source", func() {
			tok := token.NewSOFToken(source)
			Expect(tok.Source()).Should(Equal(source))
		})

		It("enables other tokens in the list to find the Source", func() {
			tok := token.NewSOFToken(source)
			Expect(tok.Source()).Should(Equal(source))

			tok2 := &token.Token{
				Kind: token.KindString,
				Prev: tok,
			}
			tok.Next = tok2
			Expect(tok2.Source()).Should(Equal(source))

		})
	})
})
