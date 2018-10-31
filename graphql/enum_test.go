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

var _ = Describe("Enum", func() {

	// graphql-js/src/type/__tests__/definition-test.js
	It("defines an enum type with deprecated value", func() {
		enumTypeWithDeprecatedValue, err := graphql.NewEnum(&graphql.EnumConfig{
			Name: "EnumWithDeprecatedValue",
			Values: graphql.EnumValueDefinitionMap{
				"foo": graphql.EnumValueDefinition{
					Deprecation: &graphql.Deprecation{
						Reason: "Just because",
					},
				},
			},
		})

		Expect(err).ShouldNot(HaveOccurred())
		Expect(enumTypeWithDeprecatedValue).ShouldNot(BeNil())

		enumValues := enumTypeWithDeprecatedValue.Values()
		Expect(len(enumValues)).Should(Equal(1))

		enumValue := enumValues[0]
		Expect(enumValue.Name()).Should(Equal("foo"))
		Expect(enumValue.Description()).Should(BeEmpty())
		Expect(enumValue.IsDeprecated()).Should(BeTrue())
		Expect(enumValue.Deprecation()).ShouldNot(BeNil())
		Expect(enumValue.Deprecation().Reason).Should(Equal("Just because"))
		Expect(enumValue.Value()).Should(Equal("foo"))
	})

	It("defines an enum type with a value of `null`", func() {
		enumTypeWithNullishValue, err := graphql.NewEnum(&graphql.EnumConfig{
			Name: "EnumTypeWithNullishValue",
			Values: graphql.EnumValueDefinitionMap{
				"NULL": graphql.EnumValueDefinition{
					Value: graphql.NilEnumInternalValue,
				},
			},
		})

		Expect(err).ShouldNot(HaveOccurred())
		Expect(enumTypeWithNullishValue).ShouldNot(BeNil())

		enumValues := enumTypeWithNullishValue.Values()
		Expect(len(enumValues)).Should(Equal(1))

		enumValue := enumValues[0]
		Expect(enumValue.Name()).Should(Equal("NULL"))
		Expect(enumValue.Description()).Should(BeEmpty())
		Expect(enumValue.IsDeprecated()).Should(BeFalse())
		Expect(enumValue.Deprecation()).Should(BeNil())
		Expect(enumValue.Value()).Should(BeNil())
	})

	It("stringifies to type name", func() {
		enumType, err := graphql.NewEnum(&graphql.EnumConfig{
			Name: "Enum",
		})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(fmt.Sprintf("%s", enumType)).Should(Equal("Enum"))
		Expect(fmt.Sprintf("%v", enumType)).Should(Equal("Enum"))
	})

	It("rejects creating type without name", func() {
		_, err := graphql.NewEnum(&graphql.EnumConfig{
			Name: "",
		})
		Expect(err).Should(MatchError("Must provide name for Enum."))

		Expect(func() {
			graphql.MustNewEnum(&graphql.EnumConfig{})
		}).Should(Panic())
	})
})
