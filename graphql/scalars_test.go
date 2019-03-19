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
	"math"

	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/internal/testutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

func MatchCoercionError(message string) types.GomegaMatcher {
	return testutil.MatchGraphQLError(
		testutil.MessageEqual(message),
		testutil.KindIs(graphql.ErrKindCoercion),
	)
}

var _ = Describe("Scalars", func() {
	// graphql-js/src/type/__tests__/serialization-test.js
	Describe("Type System: Scalar coercion", func() {
		It("serializes output as Int", func() {
			Expect(graphql.Int().CoerceResultValue(1)).Should(Equal(1))
			Expect(graphql.Int().CoerceResultValue(123)).Should(Equal(123))
			Expect(graphql.Int().CoerceResultValue(0)).Should(Equal(0))
			Expect(graphql.Int().CoerceResultValue(-1)).Should(Equal(-1))
			Expect(graphql.Int().CoerceResultValue(1e5)).Should(Equal(100000))
			Expect(graphql.Int().CoerceResultValue(false)).Should(Equal(0))
			Expect(graphql.Int().CoerceResultValue(true)).Should(Equal(1))

			var err error
			// The GraphQL specification does not allow serializing non-integer values as Int to avoid
			// accidental data loss.
			_, err = graphql.Int().CoerceResultValue(0.1)
			Expect(err).Should(MatchCoercionError("Int cannot represent 0.1: not an integer"))

			_, err = graphql.Int().CoerceResultValue(1.1)
			Expect(err).Should(MatchCoercionError("Int cannot represent 1.1: not an integer"))

			_, err = graphql.Int().CoerceResultValue(-1.1)
			Expect(err).Should(MatchCoercionError("Int cannot represent -1.1: not an integer"))

			_, err = graphql.Int().CoerceResultValue("-1.1")
			Expect(err).Should(MatchCoercionError("Int cannot represent \"-1.1\": not an integer"))

			// Maybe a safe JavaScript int, but bigger than 2^32, so not
			// representable as a GraphQL Int
			_, err = graphql.Int().CoerceResultValue(9876504321)
			Expect(err).Should(MatchCoercionError("Int cannot represent 9876504321: value too large for 32-bit signed integer"))

			_, err = graphql.Int().CoerceResultValue(-9876504321)
			Expect(err).Should(MatchCoercionError("Int cannot represent -9876504321: value too small for 32-bit signed integer"))

			// Too big to represent as an Int in GraphQL
			_, err = graphql.Int().CoerceResultValue(1e100)
			Expect(err).Should(MatchCoercionError("Int cannot represent 1e+100: not an integer"))

			_, err = graphql.Int().CoerceResultValue(-1e100)
			Expect(err).Should(MatchCoercionError("Int cannot represent -1e+100: not an integer"))

			_, err = graphql.Int().CoerceResultValue("one")
			Expect(err).Should(MatchCoercionError("Int cannot represent \"one\": not an integer"))

			// Doesn't represent number
			_, err = graphql.Int().CoerceResultValue("")
			Expect(err).Should(MatchCoercionError("Int cannot represent \"\": not an integer"))

			_, err = graphql.Int().CoerceResultValue(math.NaN())
			Expect(err).Should(MatchCoercionError("Int cannot represent NaN: not an integer"))

			_, err = graphql.Int().CoerceResultValue(math.Inf(1))
			Expect(err).Should(MatchCoercionError("Int cannot represent +Inf: not an integer"))

			_, err = graphql.Int().CoerceResultValue(math.Inf(-1))
			Expect(err).Should(MatchCoercionError("Int cannot represent -Inf: not an integer"))

			_, err = graphql.Int().CoerceResultValue([]int{5})
			Expect(err).Should(MatchCoercionError("Int cannot represent [5]: unexpected result type `[]int`"))
		})

		It("serializes output as Float", func() {
			Expect(graphql.Float().CoerceResultValue(1)).Should(Equal(1.0))
			Expect(graphql.Float().CoerceResultValue(0)).Should(Equal(0.0))
			Expect(graphql.Float().CoerceResultValue("123.5")).Should(Equal(123.5))
			Expect(graphql.Float().CoerceResultValue(-1)).Should(Equal(-1.0))
			Expect(graphql.Float().CoerceResultValue(0.1)).Should(Equal(0.1))
			Expect(graphql.Float().CoerceResultValue(1.1)).Should(Equal(1.1))
			Expect(graphql.Float().CoerceResultValue(-1.1)).Should(Equal(-1.1))
			Expect(graphql.Float().CoerceResultValue("-1.1")).Should(Equal(-1.1))
			Expect(graphql.Float().CoerceResultValue(false)).Should(Equal(0.0))
			Expect(graphql.Float().CoerceResultValue(true)).Should(Equal(1.0))

			var err error

			_, err = graphql.Float().CoerceResultValue(math.NaN())
			Expect(err).Should(MatchCoercionError("Float cannot represent NaN: not a numeric value"))
			_, err = graphql.Float().CoerceResultValue(math.Inf(1))
			Expect(err).Should(MatchCoercionError("Float cannot represent +Inf: not a numeric value"))
			_, err = graphql.Float().CoerceResultValue(math.Inf(-1))
			Expect(err).Should(MatchCoercionError("Float cannot represent -Inf: not a numeric value"))

			_, err = graphql.Float().CoerceResultValue("NaN")
			Expect(err).Should(MatchCoercionError("Float cannot represent NaN: not a numeric value"))
			_, err = graphql.Float().CoerceResultValue("Inf")
			Expect(err).Should(MatchCoercionError("Float cannot represent +Inf: not a numeric value"))
			_, err = graphql.Float().CoerceResultValue("+Inf")
			Expect(err).Should(MatchCoercionError("Float cannot represent +Inf: not a numeric value"))
			_, err = graphql.Float().CoerceResultValue("-Inf")
			Expect(err).Should(MatchCoercionError("Float cannot represent -Inf: not a numeric value"))

			_, err = graphql.Float().CoerceResultValue("one")
			Expect(err).Should(MatchCoercionError("Float cannot represent \"one\": not a numeric value"))
			_, err = graphql.Float().CoerceResultValue("")
			Expect(err).Should(MatchCoercionError("Float cannot represent \"\": not a numeric value"))
			_, err = graphql.Float().CoerceResultValue([]int{5})
			Expect(err).Should(MatchCoercionError("Float cannot represent [5]: unexpected result type `[]int`"))
		})

		It("serializes output as String", func() {
			Expect(graphql.String().CoerceResultValue("string")).Should(Equal("string"))
			Expect(graphql.String().CoerceResultValue(1)).Should(Equal("1"))
			Expect(graphql.String().CoerceResultValue(uint(100))).Should(Equal("100"))
			Expect(graphql.String().CoerceResultValue(-1.1)).Should(Equal("-1.1"))
			Expect(graphql.String().CoerceResultValue(true)).Should(Equal("true"))
			Expect(graphql.String().CoerceResultValue(false)).Should(Equal("false"))

			var err error
			_, err = graphql.String().CoerceResultValue(math.NaN())
			Expect(err).Should(MatchCoercionError("String cannot represent NaN: not a value"))

			_, err = graphql.String().CoerceResultValue([]int{5})
			Expect(err).Should(MatchCoercionError("String cannot represent [5]: unexpected result type `[]int`"))
		})

		It("serializes output as Boolean", func() {
			Expect(graphql.Boolean().CoerceResultValue(100)).Should(Equal(true))
			Expect(graphql.Boolean().CoerceResultValue(1)).Should(Equal(true))
			Expect(graphql.Boolean().CoerceResultValue(0)).Should(Equal(false))
			Expect(graphql.Boolean().CoerceResultValue(-100)).Should(Equal(true))
			Expect(graphql.Boolean().CoerceResultValue(uint(100))).Should(Equal(true))
			Expect(graphql.Boolean().CoerceResultValue(uint(1))).Should(Equal(true))
			Expect(graphql.Boolean().CoerceResultValue(uint(0))).Should(Equal(false))
			Expect(graphql.Boolean().CoerceResultValue(true)).Should(Equal(true))
			Expect(graphql.Boolean().CoerceResultValue(false)).Should(Equal(false))

			var err error
			_, err = graphql.Boolean().CoerceResultValue(math.NaN())
			Expect(err).Should(MatchCoercionError("Boolean cannot represent NaN: not a boolean value"))

			_, err = graphql.Boolean().CoerceResultValue("")
			Expect(err).Should(MatchCoercionError("Boolean cannot represent \"\": unexpected result type `string`"))
			_, err = graphql.Boolean().CoerceResultValue("true")
			Expect(err).Should(MatchCoercionError("Boolean cannot represent \"true\": unexpected result type `string`"))
			_, err = graphql.Boolean().CoerceResultValue([]bool{false})
			Expect(err).Should(MatchCoercionError("Boolean cannot represent [false]: unexpected result type `[]bool`"))
			_, err = graphql.Boolean().CoerceResultValue(struct{}{})
			Expect(err).Should(MatchCoercionError("Boolean cannot represent {}: unexpected result type `struct {}`"))
		})

		It("serializes output as ID", func() {
			Expect(graphql.ID().CoerceResultValue("string")).Should(Equal("string"))
			Expect(graphql.ID().CoerceResultValue("false")).Should(Equal("false"))
			Expect(graphql.ID().CoerceResultValue("")).Should(Equal(""))
			Expect(graphql.ID().CoerceResultValue(123)).Should(Equal("123"))
			Expect(graphql.ID().CoerceResultValue(0)).Should(Equal("0"))
			Expect(graphql.ID().CoerceResultValue(-1)).Should(Equal("-1"))

			var err error
			_, err = graphql.ID().CoerceResultValue(true)
			Expect(err).Should(MatchCoercionError("ID cannot represent true: unexpected result type `bool`"))
			_, err = graphql.ID().CoerceResultValue(3.14)
			Expect(err).Should(MatchCoercionError("ID cannot represent 3.14: unexpected result type `float64`"))
			_, err = graphql.ID().CoerceResultValue(struct{}{})
			Expect(err).Should(MatchCoercionError("ID cannot represent {}: unexpected result type `struct {}`"))
			_, err = graphql.ID().CoerceResultValue([]string{"abc"})
			Expect(err).Should(MatchCoercionError("ID cannot represent [\"abc\"]: unexpected result type `[]string`"))
		})
	})

	It("stringifies built-in scalar types", func() {
		tests := []struct {
			t        graphql.Type
			expected string
		}{
			{graphql.Int(), "Int"},
			{graphql.Float(), "Float"},
			{graphql.String(), "String"},
			{graphql.Boolean(), "Boolean"},
			{graphql.ID(), "ID"},
		}

		for _, test := range tests {
			Expect(graphql.Inspect(test.t)).Should(Equal(test.expected))
		}
	})
})
