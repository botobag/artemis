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
	"fmt"

	"github.com/botobag/artemis/graphql"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Union", func() {
	It("stringifies to type name", func() {
		unionType, err := graphql.NewUnion(&graphql.UnionConfig{
			Name: "Union",
		})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(fmt.Sprintf("%s", unionType)).Should(Equal("Union"))
		Expect(fmt.Sprintf("%v", unionType)).Should(Equal("Union"))
	})

	It("accepts empty set of possible types", func() {
		unionType, err := graphql.NewUnion(&graphql.UnionConfig{
			Name:          "UnionWithEmptySetOfPossibleTypes",
			PossibleTypes: []graphql.ObjectTypeDefinition{},
		})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(unionType.PossibleTypes()).Should(BeEmpty())

		unionType, err = graphql.NewUnion(&graphql.UnionConfig{
			Name: "UnionWithoutPossibleTypes",
		})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(unionType.PossibleTypes()).Should(BeEmpty())
	})

	It("rejects creating type without a name", func() {
		_, err := graphql.NewUnion(&graphql.UnionConfig{})
		Expect(err).Should(MatchError("Must provide name for Union."))

		Expect(func() {
			graphql.MustNewUnion(&graphql.UnionConfig{})
		}).Should(Panic())
	})
})
