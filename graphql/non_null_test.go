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

var _ = Describe("NonNull", func() {
	// graphql-js/src/type/__tests__/definition-test.js
	It("prohibits nesting NonNull inside NonNull", func() {
		_, err := graphql.NewNonNullOfType(graphql.MustNewNonNullOfType(graphql.Int()))
		Expect(err).Should(MatchError("Expected a nullable type for NonNull but got an Int!."))
	})

	It("rejects creating type without specifying element type", func() {
		_, err := graphql.NewNonNullOfType(nil)
		Expect(err).Should(MatchError("Must provide an non-nil element type for NonNull."))

		Expect(func() {
			graphql.MustNewNonNullOfType(nil)
		}).Should(Panic())
	})

	Context("specifies element type with TypeDefinition", func() {
		It("rejects nil TypeDefinition", func() {
			nilTypeDef := graphql.NonNullOf(nil)
			_, err := graphql.NewNonNullOf(nilTypeDef)
			Expect(err).Should(MatchError("Must provide an non-nil element type for NonNull."))

			Expect(func() {
				graphql.MustNewNonNullOf(nilTypeDef)
			}).Should(Panic())
		})

		It("accepts nullable TypeDefinition", func() {
			nonNullType := graphql.MustNewNonNullOf(graphql.T(graphql.Int()))
			Expect(nonNullType.InnerType()).Should(Equal(graphql.Int()))

			Expect(graphql.MustNewNonNullOf(graphql.T(graphql.Int()))).Should(Equal(nonNullType))
		})

		It("reject non-nullable TypeDefinition", func() {
			intTypeDef := graphql.T(graphql.Int())
			nonNullIntTypeDef := graphql.NonNullOf(intTypeDef)
			// Int!! is invalid.
			_, err := graphql.NewNonNullOf(nonNullIntTypeDef)
			Expect(err).Should(MatchError("Expected a nullable type for NonNull but got an Int!."))

			Expect(func() {
				graphql.MustNewNonNullOf(nonNullIntTypeDef)
			}).Should(Panic())
		})
	})
})
