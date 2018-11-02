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

package value_test

import (
	"math"

	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/graphql/internal/value"
	"github.com/botobag/artemis/internal/testutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CoerceValue", func() {
	// graphql-js/src/utilities/__tests__/coerceValue-test.js
	Describe("for String", func() {
		It("returns error for array input as string", func() {
			_, errs := value.CoerceValue([]interface{}{1, 2, 3}, graphql.String(), nil)
			Expect(errs).Should(ConsistOf(testutil.MatchGraphQLError(
				testutil.MessageEqual("Expected type String; String cannot represent [1 2 3]: invalid variable type `[]interface {}`"),
			)))
		})
	})

	Describe("for ID", func() {
		It("returns error for array input as ID", func() {
			_, errs := value.CoerceValue([]interface{}{1, 2, 3}, graphql.ID(), nil)
			Expect(errs).Should(ConsistOf(testutil.MatchGraphQLError(
				testutil.MessageEqual("Expected type ID; ID cannot represent [1 2 3]: invalid variable type `[]interface {}`"),
			)))
		})
	})

	Describe("for Int", func() {
		It("returns value for integer", func() {
			Expect(value.CoerceValue(1, graphql.Int(), nil)).Should(Equal(1))
		})

		It("returns error for numeric looking string", func() {
			_, errs := value.CoerceValue("1", graphql.Int(), nil)
			Expect(errs).Should(ConsistOf(testutil.MatchGraphQLError(
				testutil.MessageEqual("Expected type Int; Int cannot represent \"1\": invalid variable type `string`"),
			)))
		})

		It("returns value for negative int input", func() {
			Expect(value.CoerceValue(-1, graphql.Int(), nil)).Should(Equal(-1))
		})

		It("rejects value for exponent input", func() {
			_, errs := value.CoerceValue(1e3, graphql.Int(), nil)
			Expect(errs).Should(ConsistOf(testutil.MatchGraphQLError(
				testutil.MessageEqual("Expected type Int; Int cannot represent 1000: invalid variable type `float64`"),
			)))
		})

		It("returns null for null value", func() {
			Expect(value.CoerceValue(nil, graphql.Int(), nil)).Should(BeNil())
		})

		It("returns a single error for empty string as value", func() {
			_, errs := value.CoerceValue("", graphql.Int(), nil)
			Expect(errs).Should(ConsistOf(testutil.MatchGraphQLError(
				testutil.MessageEqual("Expected type Int; Int cannot represent \"\": invalid variable type `string`"),
			)))
		})

		It("returns a single error for 2^32 input as int", func() {
			_, errs := value.CoerceValue(uint64(1<<32), graphql.Int(), nil)
			Expect(errs).Should(ConsistOf(testutil.MatchGraphQLError(
				testutil.MessageEqual("Expected type Int; Int cannot represent 4294967296: value too large for 32-bit signed integer"),
			)))
		})

		It("returns a single error for float input as int", func() {
			_, errs := value.CoerceValue(1.5, graphql.Int(), nil)
			Expect(errs).Should(ConsistOf(testutil.MatchGraphQLError(
				testutil.MessageEqual("Expected type Int; Int cannot represent 1.5: invalid variable type `float64`"),
			)))
		})

		It("returns a single error for NaN input as int", func() {
			_, errs := value.CoerceValue(math.NaN(), graphql.Int(), nil)
			Expect(errs).Should(ConsistOf(testutil.MatchGraphQLError(
				testutil.MessageEqual("Expected type Int; Int cannot represent NaN: not an integer"),
			)))
		})

		It("returns a single error for Infinity input as int", func() {
			_, errs := value.CoerceValue(math.Inf(+1), graphql.Int(), nil)
			Expect(errs).Should(ConsistOf(testutil.MatchGraphQLError(
				testutil.MessageEqual("Expected type Int; Int cannot represent +Inf: not an integer"),
			)))

			_, errs = value.CoerceValue(math.Inf(-1), graphql.Int(), nil)
			Expect(errs).Should(ConsistOf(testutil.MatchGraphQLError(
				testutil.MessageEqual("Expected type Int; Int cannot represent -Inf: not an integer"),
			)))
		})

		It("returns a single error for char input", func() {
			_, errs := value.CoerceValue("a", graphql.Int(), nil)
			Expect(errs).Should(ConsistOf(testutil.MatchGraphQLError(
				testutil.MessageEqual("Expected type Int; Int cannot represent \"a\": invalid variable type `string`"),
			)))
		})

		It("returns a single error for string input", func() {
			_, errs := value.CoerceValue("meow", graphql.Int(), nil)
			Expect(errs).Should(ConsistOf(testutil.MatchGraphQLError(
				testutil.MessageEqual("Expected type Int; Int cannot represent \"meow\": invalid variable type `string`"),
			)))
		})
	})

	Describe("for Float", func() {
		It("returns value for integer", func() {
			Expect(value.CoerceValue(1, graphql.Float(), nil)).Should(Equal(1.0))
		})

		It("returns value for decimal", func() {
			Expect(value.CoerceValue(1.1, graphql.Float(), nil)).Should(Equal(1.1))
		})

		It("returns value for exponent input", func() {
			Expect(value.CoerceValue(1e3, graphql.Float(), nil)).Should(Equal(1000.0))
		})

		It("returns error for numeric looking string", func() {
			_, errs := value.CoerceValue("1", graphql.Float(), nil)
			Expect(errs).Should(ConsistOf(testutil.MatchGraphQLError(
				testutil.MessageEqual("Expected type Float; Float cannot represent \"1\": invalid variable type `string`"),
			)))
		})

		It("returns null for null value", func() {
			Expect(value.CoerceValue(nil, graphql.Float(), nil)).Should(BeNil())
		})

		It("returns a single error for empty string as value", func() {
			_, errs := value.CoerceValue("", graphql.Float(), nil)
			Expect(errs).Should(ConsistOf(testutil.MatchGraphQLError(
				testutil.MessageEqual("Expected type Float; Float cannot represent \"\": invalid variable type `string`"),
			)))
		})

		It("returns a single error for NaN input as int", func() {
			_, errs := value.CoerceValue(math.NaN(), graphql.Float(), nil)
			Expect(errs).Should(ConsistOf(testutil.MatchGraphQLError(
				testutil.MessageEqual("Expected type Float; Float cannot represent NaN: not a numeric value"),
			)))
		})

		It("returns a single error for Infinity input as int", func() {
			_, errs := value.CoerceValue(math.Inf(+1), graphql.Float(), nil)
			Expect(errs).Should(ConsistOf(testutil.MatchGraphQLError(
				testutil.MessageEqual("Expected type Float; Float cannot represent +Inf: not a numeric value"),
			)))

			_, errs = value.CoerceValue(math.Inf(-1), graphql.Float(), nil)
			Expect(errs).Should(ConsistOf(testutil.MatchGraphQLError(
				testutil.MessageEqual("Expected type Float; Float cannot represent -Inf: not a numeric value"),
			)))
		})

		It("returns a single error for char input", func() {
			_, errs := value.CoerceValue("a", graphql.Float(), nil)
			Expect(errs).Should(ConsistOf(testutil.MatchGraphQLError(
				testutil.MessageEqual("Expected type Float; Float cannot represent \"a\": invalid variable type `string`"),
			)))
		})

		It("returns a single error for string input", func() {
			_, errs := value.CoerceValue("meow", graphql.Float(), nil)
			Expect(errs).Should(ConsistOf(testutil.MatchGraphQLError(
				testutil.MessageEqual("Expected type Float; Float cannot represent \"meow\": invalid variable type `string`"),
			)))
		})
	})

	Describe("for Enum", func() {
		var TestEnum graphql.Type

		BeforeEach(func() {
			var err error
			TestEnum, err = graphql.NewEnum(&graphql.EnumConfig{
				Name: "TestEnum",
				Values: graphql.EnumValueDefinitionMap{
					"FOO": {Value: "InternalFoo"},
					"BAR": {Value: 123456789},
				},
			})
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("returns no error for a known enum name", func() {
			Expect(value.CoerceValue("FOO", TestEnum, nil)).Should(Equal("InternalFoo"))
			Expect(value.CoerceValue("BAR", TestEnum, nil)).Should(Equal(123456789))
		})

		It("results error for misspelled enum value", func() {
			_, errs := value.CoerceValue("foo", TestEnum, nil)
			Expect(errs).Should(ConsistOf(testutil.MatchGraphQLError(
				testutil.MessageEqual("Expected type TestEnum; did you mean FOO?"),
			)))
		})

		It("results error for incorrect value type", func() {
			_, errs := value.CoerceValue(123, TestEnum, nil)
			Expect(errs).Should(ConsistOf(testutil.MatchGraphQLError(
				testutil.MessageEqual("Expected type TestEnum."),
			)))

			_, errs = value.CoerceValue(map[string]interface{}{"field": "value"}, TestEnum, nil)
			Expect(errs).Should(ConsistOf(testutil.MatchGraphQLError(
				testutil.MessageEqual("Expected type TestEnum."),
			)))
		})
	})

	Describe("for InputObject", func() {
		var TestInputObject graphql.Type

		BeforeEach(func() {
			var err error
			TestInputObject, err = graphql.NewInputObject(&graphql.InputObjectConfig{
				Name: "TestInputObject",
				Fields: graphql.InputFields{
					"foo": {
						Type: graphql.NonNullOfType(graphql.Int()),
					},
					"bar": {
						Type: graphql.T(graphql.Int()),
					},
				},
			})
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("returns no error for a valid input", func() {
			Expect(value.CoerceValue(map[string]interface{}{"foo": 123}, TestInputObject, nil)).
				Should(Equal(map[string]interface{}{"foo": 123}))
		})

		It("returns an error for a non-object type", func() {
			_, errs := value.CoerceValue(123, TestInputObject, nil)
			Expect(errs).Should(ConsistOf(testutil.MatchGraphQLError(
				testutil.MessageEqual("Expected type TestInputObject to be an object."),
			)))
		})

		It("returns an error for an invalid field", func() {
			_, errs := value.CoerceValue(map[string]interface{}{"foo": "abc"}, TestInputObject, nil)
			Expect(errs).Should(ConsistOf(testutil.MatchGraphQLError(
				testutil.MessageEqual("Expected type Int at value.foo; Int cannot represent \"abc\": invalid variable type `string`"),
			)))
		})

		It("returns multiple errors for multiple invalid fields", func() {
			_, errs := value.CoerceValue(map[string]interface{}{
				"foo": "abc",
				"bar": "def",
			}, TestInputObject, nil)
			Expect(errs).Should(ConsistOf(
				testutil.MatchGraphQLError(
					testutil.MessageEqual("Expected type Int at value.foo; Int cannot represent \"abc\": invalid variable type `string`"),
				),
				testutil.MatchGraphQLError(
					testutil.MessageEqual("Expected type Int at value.bar; Int cannot represent \"def\": invalid variable type `string`"),
				)))
		})

		It("returns error for a missing required field", func() {
			_, errs := value.CoerceValue(map[string]interface{}{"bar": 123}, TestInputObject, nil)
			Expect(errs).Should(ConsistOf(testutil.MatchGraphQLError(
				testutil.MessageEqual("Field value.foo of required type Int! was not provided."),
			)))
		})

		It("returns error for an unknown field", func() {
			_, errs := value.CoerceValue(map[string]interface{}{
				"foo":          123,
				"unknownField": 123,
			}, TestInputObject, nil)

			Expect(errs).Should(ConsistOf(testutil.MatchGraphQLError(
				testutil.MessageEqual(`Field "unknownField" is not defined by type TestInputObject.`),
			)))
		})

		It("returns error for a misspelled field", func() {
			_, errs := value.CoerceValue(map[string]interface{}{
				"foo":  123,
				"bart": 123,
			}, TestInputObject, nil)

			Expect(errs).Should(ConsistOf(testutil.MatchGraphQLError(
				testutil.MessageEqual(`Field "bart" is not defined by type TestInputObject; did you mean bar?`),
			)))
		})
	})

	Describe("for List", func() {
		var TestList graphql.Type

		BeforeEach(func() {
			var err error
			TestList, err = graphql.NewListOfType(graphql.Int())
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("returns no error for a valid input", func() {
			Expect(value.CoerceValue([]interface{}{1, 2, 3}, TestList, nil)).
				Should(Equal([]interface{}{1, 2, 3}))
		})

		It("returns an error for an invalid input", func() {
			_, errs := value.CoerceValue([]interface{}{1, "b", true}, TestList, nil)

			Expect(errs).Should(ConsistOf(
				testutil.MatchGraphQLError(
					testutil.MessageEqual("Expected type Int at value[1]; Int cannot represent \"b\": invalid variable type `string`"),
				),
				testutil.MatchGraphQLError(
					testutil.MessageEqual("Expected type Int at value[2]; Int cannot represent true: invalid variable type `bool`"),
				),
			))
		})

		It("returns a list for a non-list value", func() {
			Expect(value.CoerceValue(42, TestList, nil)).Should(Equal([]interface{}{42}))
		})

		It("returns null for a null value", func() {
			Expect(value.CoerceValue(nil, TestList, nil)).Should(BeNil())
		})

		It("returns an empty list for an empty list", func() {
			Expect(value.CoerceValue([]interface{}{}, TestList, nil)).Should(BeEmpty())
		})
	})

	Describe("for nested List", func() {
		var TestNestedList graphql.Type

		BeforeEach(func() {
			var err error
			TestNestedList, err = graphql.NewListOf(graphql.ListOfType(graphql.Int()))
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("returns no error for a valid input", func() {
			testValue := []interface{}{
				[]interface{}{1},
				[]interface{}{2, 3},
			}
			Expect(value.CoerceValue(testValue, TestNestedList, nil)).Should(Equal(testValue))
		})

		It("returns a list for a non-list value", func() {
			Expect(value.CoerceValue(42, TestNestedList, nil)).Should(Equal([]interface{}{[]interface{}{42}}))
		})

		It("returns null for a null value", func() {
			Expect(value.CoerceValue(nil, TestNestedList, nil)).Should(BeNil())
		})

		It("returns nested lists for nested non-list values", func() {
			Expect(value.CoerceValue([]interface{}{1, 2, 3}, TestNestedList, nil)).Should(Equal(
				[]interface{}{
					[]interface{}{1},
					[]interface{}{2},
					[]interface{}{3},
				}))
		})

		It("returns nested null for nested null values", func() {
			Expect(value.CoerceValue([]interface{}{42, []interface{}{nil}, nil}, TestNestedList, nil)).Should(Equal(
				[]interface{}{
					[]interface{}{42},
					[]interface{}{nil},
					nil,
				}))
		})
	})

	It("rejects null values for non-null types", func() {
		testNonNull, err := graphql.NewNonNullOfType(graphql.Int())
		Expect(err).ShouldNot(HaveOccurred())

		_, errs := value.CoerceValue(nil, testNonNull, nil)
		Expect(errs).Should(ConsistOf(testutil.MatchGraphQLError(
			testutil.MessageEqual(`Expected non-nullable type Int! not to be null.`),
		)))
	})

	It("rejects non-input type", func() {
		testObject, err := graphql.NewObject(&graphql.ObjectConfig{
			Name: "TestObject",
			Fields: graphql.Fields{
				"int": {
					Type: graphql.T(graphql.Int()),
				},
			},
		})
		Expect(err).ShouldNot(HaveOccurred())

		_, errs := value.CoerceValue(map[string]interface{}{"int": 2}, testObject, nil)
		Expect(errs).Should(ConsistOf(testutil.MatchGraphQLError(
			testutil.MessageEqual(`TestObject is not a valid input type.`),
		)))
	})
})
