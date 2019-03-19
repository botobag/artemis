/**
 * Copyright (c) 2019, The Artemis Authors.
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

package rules_test

import (
	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/graphql/internal/validator"
	"github.com/botobag/artemis/graphql/validator/rules"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// graphql-js/src/validation/__tests__/ScalarLeafs-test.js@8c96dc8
var _ = Describe("Validate: Scalar leafs", func() {
	expectErrors := func(queryStr string) GomegaAssertion {
		return expectValidationErrors(rules.ScalarLeafs{}, queryStr)
	}

	expectValid := func(queryStr string) {
		expectErrors(queryStr).Should(Equal(graphql.NoErrors()))
	}

	noScalarSubselection := func(
		fieldName string,
		typeName string,
		line uint,
		column uint) error {

		return graphql.NewError(
			validator.NoSubselectionAllowedMessage(fieldName, typeName),
			[]graphql.ErrorLocation{
				{Line: line, Column: column},
			},
		)
	}

	missingObjSubselection := func(
		fieldName string,
		typeName string,
		line uint,
		column uint) error {

		return graphql.NewError(
			validator.RequiredSubselectionMessage(fieldName, typeName),
			[]graphql.ErrorLocation{
				{Line: line, Column: column},
			},
		)
	}

	It("valid scalar selection", func() {
		expectValid(`
      fragment scalarSelection on Dog {
        barks
      }
    `)
	})

	It("object type missing selection", func() {
		expectErrors(`
      query directQueryOnObjectWithoutSubFields {
        human
      }
    `).Should(Equal(graphql.ErrorsOf(
			missingObjSubselection("human", "Human", 3, 9),
		)))
	})

	It("interface type missing selection", func() {
		expectErrors(`
      {
        human { pets }
      }
    `).Should(Equal(graphql.ErrorsOf(
			missingObjSubselection("pets", "[Pet]", 3, 17),
		)))
	})

	It("valid scalar selection with args", func() {
		expectValid(`
      fragment scalarSelectionWithArgs on Dog {
        doesKnowCommand(dogCommand: SIT)
      }
    `)
	})

	It("scalar selection not allowed on Boolean", func() {
		expectErrors(`
      fragment scalarSelectionsNotAllowedOnBoolean on Dog {
        barks { sinceWhen }
      }
    `).Should(Equal(graphql.ErrorsOf(
			noScalarSubselection("barks", "Boolean", 3, 15),
		)))
	})

	It("scalar selection not allowed on Enum", func() {
		expectErrors(`
      fragment scalarSelectionsNotAllowedOnEnum on Cat {
        furColor { inHexdec }
      }
    `).Should(Equal(graphql.ErrorsOf(
			noScalarSubselection("furColor", "FurColor", 3, 18),
		)))
	})

	It("scalar selection not allowed with args", func() {
		expectErrors(`
      fragment scalarSelectionsNotAllowedWithArgs on Dog {
        doesKnowCommand(dogCommand: SIT) { sinceWhen }
      }
    `).Should(Equal(graphql.ErrorsOf(
			noScalarSubselection("doesKnowCommand", "Boolean", 3, 42),
		)))
	})

	It("Scalar selection not allowed with directives", func() {
		expectErrors(`
      fragment scalarSelectionsNotAllowedWithDirectives on Dog {
        name @include(if: true) { isAlsoHumanName }
      }
    `).Should(Equal(graphql.ErrorsOf(
			noScalarSubselection("name", "String", 3, 33),
		)))
	})

	It("Scalar selection not allowed with directives and args", func() {
		expectErrors(`
      fragment scalarSelectionsNotAllowedWithDirectivesAndArgs on Dog {
        doesKnowCommand(dogCommand: SIT) @include(if: true) { sinceWhen }
      }
    `).Should(Equal(graphql.ErrorsOf(
			noScalarSubselection("doesKnowCommand", "Boolean", 3, 61),
		)))
	})
})
