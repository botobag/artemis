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

package graphql_test

import (
	"github.com/botobag/artemis/graphql"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Interface", func() {
	It("rejects creating type without name", func() {
		_, err := graphql.NewInterface(&graphql.InterfaceConfig{
			Name: "",
		})
		Expect(err).Should(MatchError("Must provide name for Interface."))

		Expect(func() {
			graphql.MustNewInterface(&graphql.InterfaceConfig{})
		}).Should(Panic())
	})

	It("accepts creating type without fields", func() {
		Expect(graphql.MustNewInterface(&graphql.InterfaceConfig{
			Name: "InterfaceWithoutFields1",
		}).Fields()).Should(BeEmpty())

		Expect(graphql.MustNewInterface(&graphql.InterfaceConfig{
			Name: "InterfaceWithoutFields2",
		}).Fields()).Should(BeEmpty())
	})
})
