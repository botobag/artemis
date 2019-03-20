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

var _ = Describe("Field and Argument", func() {

	// graphql-js/src/type/__tests__/predicate-test.js@8c96dc8
	Describe("IsRequiredArgument", func() {
		It("returns true for required arguments", func() {
			requiredArg := graphql.MockArgument("someArg", "", graphql.MustNewNonNullOfType(graphql.String()), nil)
			Expect(graphql.IsRequiredArgument(&requiredArg)).Should(BeTrue())
		})

		It("returns false for optional arguments", func() {
			optArg1 := graphql.MockArgument("someArg", "", graphql.String(), nil)
			Expect(graphql.IsRequiredArgument(&optArg1)).Should(BeFalse())

			optArg2 := graphql.MockArgument("someArg", "", graphql.String(), graphql.NilArgumentDefaultValue)
			Expect(graphql.IsRequiredArgument(&optArg2)).Should(BeFalse())

			optArg3 := graphql.MockArgument(
				"someArg",
				"",
				graphql.MustNewListOf(graphql.NonNullOfType(graphql.String())),
				nil)
			Expect(graphql.IsRequiredArgument(&optArg3)).Should(BeFalse())

			optArg4 := graphql.MockArgument(
				"someArg",
				"",
				graphql.MustNewNonNullOfType(graphql.String()),
				"default",
			)
			Expect(graphql.IsRequiredArgument(&optArg4)).Should(BeFalse())
		})
	})
})
