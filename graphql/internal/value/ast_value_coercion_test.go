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
	"github.com/botobag/artemis/graphql/parser"
	"github.com/botobag/artemis/graphql/token"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func valueFromAST(t graphql.Type, valueText string) (interface{}, error) {
	return valueFromASTWithVars(nil, t, valueText)
}

type vars map[string]interface{}

func valueFromASTWithVars(variables vars, t graphql.Type, valueText string) (interface{}, error) {
	// Parse value.
	astValue, err := parser.ParseValue(token.NewSource(&token.SourceConfig{
		Body: token.SourceBody([]byte(valueText)),
	}))
	Expect(err).ShouldNot(HaveOccurred())

	return value.CoerceFromAST(astValue, t, variables)
}

type testCase struct {
	valueText     string
	hasError      bool
	expectedValue interface{}
}

func runTestCasesForType(t graphql.Type, tests []testCase) {
	for _, test := range tests {
		if test.hasError {
			_, err := valueFromAST(t, test.valueText)
			Expect(err).Should(HaveOccurred())
		} else if test.expectedValue == nil {
			// Gomega: Refusing to compare <nil> to <nil>. Be explicit and use BeNil() instead.
			Expect(valueFromAST(t, test.valueText)).Should(BeNil())
		} else {
			Expect(valueFromAST(t, test.valueText)).Should(Equal(test.expectedValue))
		}
	}
}

var _ = Describe("CoerceFromAST", func() {
	// graphql-js/src/utilities/__tests__/valueFromAST-test.js
	It("rejects empty input", func() {
		_, err := value.CoerceFromAST(nil, graphql.Boolean(), nil)
		Expect(err).Should(HaveOccurred())
	})

	It("converts according to input coercion rules", func() {
		Expect(valueFromAST(graphql.Boolean(), "true")).Should(Equal(true))
		Expect(valueFromAST(graphql.Boolean(), "false")).Should(Equal(false))
		Expect(valueFromAST(graphql.Int(), "123")).Should(Equal(123))
		Expect(valueFromAST(graphql.Float(), "123")).Should(Equal(123.0))
		Expect(valueFromAST(graphql.Float(), "123.456")).Should(Equal(123.456))
		Expect(valueFromAST(graphql.String(), `"abc123"`)).Should(Equal("abc123"))
		Expect(valueFromAST(graphql.ID(), "123456")).Should(Equal("123456"))
		Expect(valueFromAST(graphql.ID(), `"123456"`)).Should(Equal("123456"))
	})

	It("does not convert when input coercion rules reject a value", func() {
		tests := []struct {
			t         graphql.Type
			valueText string
		}{
			{graphql.Boolean(), "123"},
			{graphql.Int(), "123.456"},
			{graphql.Int(), "true"},
			{graphql.Int(), `"123"`},
			{graphql.Float(), `"123"`},
			{graphql.String(), "123"},
			{graphql.String(), "true"},
			{graphql.ID(), "123.456"},
		}

		for _, test := range tests {
			_, err := valueFromAST(test.t, test.valueText)
			Expect(err).Should(HaveOccurred(), "type = %v, value = %s", test.t, test.valueText)
		}
	})

	It("converts enum values according to input coercion rules", func() {
		testEnum, err := graphql.NewEnum(&graphql.EnumConfig{
			Name: "TestColor",
			Values: graphql.EnumValueDefinitionMap{
				"RED":   {Value: 1},
				"GREEN": {Value: 2},
				"BLUE":  {Value: 3},
				"NULL":  {Value: graphql.NilEnumInternalValue},
				// "UNDEFINED": {Value: nil},
				"NAN": {Value: math.NaN()},
			},
		})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(valueFromAST(testEnum, "RED")).Should(Equal(1))
		Expect(valueFromAST(testEnum, "BLUE")).Should(Equal(3))
		Expect(valueFromAST(testEnum, "null")).Should(BeNil())
		Expect(valueFromAST(testEnum, "NULL")).Should(BeNil())

		v, err := valueFromAST(testEnum, "NAN")
		Expect(err).ShouldNot(HaveOccurred())
		Expect(math.IsNaN(v.(float64))).Should(BeTrue())

		_, err = valueFromAST(testEnum, "3")
		Expect(err).Should(HaveOccurred())
		_, err = valueFromAST(testEnum, `"BLUE"`)
		Expect(err).Should(HaveOccurred())
		_, err = valueFromAST(testEnum, "UNDEFINED")
		Expect(err).Should(HaveOccurred())
	})

	var (
		// Boolean!
		nonNullBool graphql.Type
		// [Boolean]
		listOfBool graphql.Type
		// [Boolean!]
		listOfNonNullBool graphql.Type
		// [Boolean]!
		nonNullListOfBool graphql.Type
		// [Boolean!]!
		nonNullListOfNonNullBool graphql.Type

		testInputObj graphql.Type
	)

	BeforeEach(func() {
		var err error

		nonNullBool, err = graphql.NewNonNullOfType(graphql.Boolean())
		Expect(err).ShouldNot(HaveOccurred())

		listOfBool, err = graphql.NewListOfType(graphql.Boolean())
		Expect(err).ShouldNot(HaveOccurred())

		listOfNonNullBool, err = graphql.NewListOfType(nonNullBool)
		Expect(err).ShouldNot(HaveOccurred())

		nonNullListOfBool, err = graphql.NewNonNullOfType(listOfBool)
		Expect(err).ShouldNot(HaveOccurred())

		nonNullListOfNonNullBool, err = graphql.NewNonNullOfType(listOfNonNullBool)
		Expect(err).ShouldNot(HaveOccurred())

		testInputObj, err = graphql.NewInputObject(&graphql.InputObjectConfig{
			Name: "TestInput",
			Fields: graphql.InputFields{
				"int": {
					Type:         graphql.T(graphql.Int()),
					DefaultValue: 42,
				},
				"bool": {
					Type: graphql.T(graphql.Boolean()),
				},
				"requiredBool": {
					Type: graphql.T(nonNullBool),
				},
			},
		})
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("coerces to null unless non-null", func() {
		Expect(valueFromAST(graphql.Boolean(), "null")).Should(BeNil())

		_, err := valueFromAST(nonNullBool, "null")
		Expect(err).Should(HaveOccurred())
	})

	It("coerces lists of values", func() {
		runTestCasesForType(listOfBool, []testCase{
			{"true", false, []interface{}{true}},
			{"123", true, nil},
			{"null", false, nil},
			{"[true, false]", false, []interface{}{true, false}},
			{"[true, 123]", true, nil},
			{"[true, null]", false, []interface{}{true, nil}},
			{"{ true: true }", true, nil},
		})
	})

	It("coerces non-null lists of values", func() {
		runTestCasesForType(nonNullListOfBool, []testCase{
			{"true", false, []interface{}{true}},
			{"123", true, nil},
			{"null", true, nil},
			{"[true, false]", false, []interface{}{true, false}},
			{"[true, 123]", true, nil},
			{"[true, null]", false, []interface{}{true, nil}},
		})
	})

	It("coerces lists of non-null values", func() {
		runTestCasesForType(listOfNonNullBool, []testCase{
			{"true", false, []interface{}{true}},
			{"123", true, nil},
			{"null", false, nil},
			{"[true, false]", false, []interface{}{true, false}},
			{"[true, 123]", true, nil},
			{"[true, null]", true, nil},
		})
	})

	It("coerces non-null lists of non-null values", func() {
		runTestCasesForType(nonNullListOfNonNullBool, []testCase{
			{"true", false, []interface{}{true}},
			{"123", true, nil},
			{"null", true, nil},
			{"[true, false]", false, []interface{}{true, false}},
			{"[true, 123]", true, nil},
			{"[true, null]", true, nil},
		})
	})

	It("coerces input objects according to input coercion rules", func() {
		runTestCasesForType(testInputObj, []testCase{
			{"null", false, nil},
			{"123", true, nil},
			{"[]", true, nil},
			{
				valueText: "{ int: 123, requiredBool: false }",
				hasError:  false,
				expectedValue: map[string]interface{}{
					"int":          123,
					"requiredBool": false,
				},
			},
			{
				valueText: "{ bool: true, requiredBool: false }",
				hasError:  false,
				expectedValue: map[string]interface{}{
					"int":          42,
					"bool":         true,
					"requiredBool": false,
				},
			},
			{"{ int: true, requiredBool: true }", true, nil},
			{"{ requiredBool: null }", true, nil},
			{"{ bool: true }", true, nil},
		})
	})

	It("accepts variable values assuming already coerced", func() {
		_, err := valueFromASTWithVars(vars{}, graphql.Boolean(), "$var")
		Expect(err).Should(HaveOccurred())

		Expect(valueFromASTWithVars(vars{"var": true}, graphql.Boolean(), "$var")).Should(Equal(true))
		Expect(valueFromASTWithVars(vars{"var": nil}, graphql.Boolean(), "$var")).Should(BeNil())
	})

	It("asserts variables are provided as items in lists", func() {
		Expect(valueFromASTWithVars(vars{}, listOfBool, "[ $foo ]")).Should(ConsistOf(BeNil()))
		_, err := valueFromASTWithVars(vars{}, listOfNonNullBool, "[ $foo ]")
		Expect(err).Should(HaveOccurred())

		Expect(valueFromASTWithVars(vars{"foo": true}, listOfNonNullBool, "[ $foo ]")).Should(Equal([]interface{}{true}))
		// Note: variables are expected to have already been coerced, so we do not expect the singleton
		// wrapping behavior for variables.
		Expect(valueFromASTWithVars(vars{"foo": true}, listOfNonNullBool, "$foo")).Should(Equal(true))
		Expect(valueFromASTWithVars(vars{"foo": []bool{true}}, listOfNonNullBool, "$foo")).Should(Equal([]bool{true}))
	})

	It("omits input object fields for unprovided variables", func() {
		Expect(valueFromASTWithVars(vars{}, testInputObj, "{ int: $foo, bool: $foo, requiredBool: true }")).Should(Equal(
			map[string]interface{}{
				"int":          42,
				"requiredBool": true,
			},
		))

		_, err := valueFromASTWithVars(vars{}, testInputObj, "{ requiredBool: $foo }")
		Expect(err).Should(HaveOccurred())

		Expect(valueFromASTWithVars(vars{"foo": true}, testInputObj, "{ requiredBool: $foo }")).Should(Equal(
			map[string]interface{}{
				"int":          42,
				"requiredBool": true,
			},
		))
	})

	It("rejects missing variable", func() {
		var err error

		_, err = valueFromASTWithVars(vars{}, graphql.Boolean(), "$var")
		Expect(err).Should(HaveOccurred())
		_, err = valueFromASTWithVars(nil, graphql.Boolean(), "$var")
		Expect(err).Should(HaveOccurred())

		_, err = valueFromASTWithVars(vars{}, listOfNonNullBool, "[ $var ]")
		Expect(err).Should(HaveOccurred())
		_, err = valueFromASTWithVars(nil, listOfNonNullBool, "[ $var ]")
		Expect(err).Should(HaveOccurred())
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

		_, err = valueFromAST(testObject, "{ int: 2 }")
		Expect(err).Should(HaveOccurred())
	})
})
