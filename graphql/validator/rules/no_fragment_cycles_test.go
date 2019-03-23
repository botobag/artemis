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

// graphql-js/src/validation/__tests__/NoFragmentCycles-test.js@8c96dc8
var _ = Describe("Validate: No circular fragment spreads", func() {
	expectErrors := func(queryStr string) GomegaAssertion {
		return expectValidationErrors(rules.NoFragmentCycles{}, queryStr)
	}

	expectValid := func(queryStr string) {
		expectErrors(queryStr).Should(Equal(graphql.NoErrors()))
	}

	It("single reference is valid", func() {
		expectValid(`
      fragment fragA on Dog { ...fragB }
      fragment fragB on Dog { name }
    `)
	})

	It("spreading twice is not circular", func() {
		expectValid(`
      fragment fragA on Dog { ...fragB, ...fragB }
      fragment fragB on Dog { name }
    `)
	})

	It("spreading twice indirectly is not circular", func() {
		expectValid(`
      fragment fragA on Dog { ...fragB, ...fragC }
      fragment fragB on Dog { ...fragC }
      fragment fragC on Dog { name }
    `)
	})

	It("double spread within abstract types", func() {
		expectValid(`
      fragment nameFragment on Pet {
        ... on Dog { name }
        ... on Cat { name }
      }

      fragment spreadsInAnon on Pet {
        ... on Dog { ...nameFragment }
        ... on Cat { ...nameFragment }
      }
    `)
	})

	It("does not false positive on unknown fragment", func() {
		expectValid(`
      fragment nameFragment on Pet {
        ...UnknownFragment
      }
    `)
	})

	It("spreading recursively within field fails", func() {
		expectErrors(`
      fragment fragA on Human { relatives { ...fragA } },
    `).Should(Equal(graphql.ErrorsOf(
			graphql.NewError(
				validator.CycleErrorMessage("fragA", nil),
				[]graphql.ErrorLocation{
					{Line: 2, Column: 45},
				},
			),
		)))
	})

	It("no spreading itself directly", func() {
		expectErrors(`
      fragment fragA on Dog { ...fragA }
    `).Should(Equal(graphql.ErrorsOf(
			graphql.NewError(
				validator.CycleErrorMessage("fragA", nil),
				[]graphql.ErrorLocation{
					{Line: 2, Column: 31},
				},
			),
		)))
	})

	It("no spreading itself directly within inline fragment", func() {
		expectErrors(`
      fragment fragA on Pet {
        ... on Dog {
          ...fragA
        }
      }
    `).Should(Equal(graphql.ErrorsOf(
			graphql.NewError(
				validator.CycleErrorMessage("fragA", nil),
				[]graphql.ErrorLocation{
					{Line: 4, Column: 11},
				},
			),
		)))
	})

	It("no spreading itself indirectly", func() {
		expectErrors(`
      fragment fragA on Dog { ...fragB }
      fragment fragB on Dog { ...fragA }
    `).Should(Equal(graphql.ErrorsOf(
			graphql.NewError(
				validator.CycleErrorMessage("fragA", []string{"fragB"}),
				[]graphql.ErrorLocation{
					{Line: 2, Column: 31},
					{Line: 3, Column: 31},
				},
			),
		)))
	})

	It("no spreading itself indirectly reports opposite order", func() {
		expectErrors(`
      fragment fragB on Dog { ...fragA }
      fragment fragA on Dog { ...fragB }
    `).Should(Equal(graphql.ErrorsOf(
			graphql.NewError(
				validator.CycleErrorMessage("fragB", []string{"fragA"}),
				[]graphql.ErrorLocation{
					{Line: 2, Column: 31},
					{Line: 3, Column: 31},
				},
			),
		)))
	})

	It("no spreading itself indirectly within inline fragment", func() {
		expectErrors(`
      fragment fragA on Pet {
        ... on Dog {
          ...fragB
        }
      }
      fragment fragB on Pet {
        ... on Dog {
          ...fragA
        }
      }
    `).Should(Equal(graphql.ErrorsOf(
			graphql.NewError(
				validator.CycleErrorMessage("fragA", []string{"fragB"}),
				[]graphql.ErrorLocation{
					{Line: 4, Column: 11},
					{Line: 9, Column: 11},
				},
			),
		)))
	})

	It("no spreading itself deeply", func() {
		expectErrors(`
      fragment fragA on Dog { ...fragB }
      fragment fragB on Dog { ...fragC }
      fragment fragC on Dog { ...fragO }
      fragment fragX on Dog { ...fragY }
      fragment fragY on Dog { ...fragZ }
      fragment fragZ on Dog { ...fragO }
      fragment fragO on Dog { ...fragP }
      fragment fragP on Dog { ...fragA, ...fragX }
    `).Should(Equal(graphql.ErrorsOf(
			graphql.NewError(
				validator.CycleErrorMessage("fragA", []string{
					"fragB",
					"fragC",
					"fragO",
					"fragP",
				}),
				[]graphql.ErrorLocation{
					{Line: 2, Column: 31},
					{Line: 3, Column: 31},
					{Line: 4, Column: 31},
					{Line: 8, Column: 31},
					{Line: 9, Column: 31},
				},
			),
			graphql.NewError(
				validator.CycleErrorMessage("fragO", []string{
					"fragP",
					"fragX",
					"fragY",
					"fragZ",
				}),
				[]graphql.ErrorLocation{
					{Line: 8, Column: 31},
					{Line: 9, Column: 41},
					{Line: 5, Column: 31},
					{Line: 6, Column: 31},
					{Line: 7, Column: 31},
				},
			),
		)))
	})

	It("no spreading itself deeply two paths", func() {
		expectErrors(`
      fragment fragA on Dog { ...fragB, ...fragC }
      fragment fragB on Dog { ...fragA }
      fragment fragC on Dog { ...fragA }
    `).Should(Or(
			Equal(graphql.ErrorsOf(
				graphql.NewError(
					validator.CycleErrorMessage("fragA", []string{"fragB"}),
					[]graphql.ErrorLocation{
						{Line: 2, Column: 31},
						{Line: 3, Column: 31},
					},
				),
				graphql.NewError(
					validator.CycleErrorMessage("fragA", []string{"fragC"}),
					[]graphql.ErrorLocation{
						{Line: 2, Column: 41},
						{Line: 4, Column: 31},
					},
				),
			)),
		))
	})

	It("no spreading itself deeply two paths -- alt traverse order", func() {
		expectErrors(`
      fragment fragA on Dog { ...fragC }
      fragment fragB on Dog { ...fragC }
      fragment fragC on Dog { ...fragA, ...fragB }
    `).Should(Equal(graphql.ErrorsOf(
			graphql.NewError(
				validator.CycleErrorMessage("fragA", []string{"fragC"}),
				[]graphql.ErrorLocation{
					{Line: 2, Column: 31},
					{Line: 4, Column: 31},
				},
			),
			graphql.NewError(
				validator.CycleErrorMessage("fragC", []string{"fragB"}),
				[]graphql.ErrorLocation{
					{Line: 4, Column: 41},
					{Line: 3, Column: 31},
				},
			),
		)))
	})

	It("no spreading itself deeply and immediately", func() {
		expectErrors(`
      fragment fragA on Dog { ...fragB }
      fragment fragB on Dog { ...fragB, ...fragC }
      fragment fragC on Dog { ...fragA, ...fragB }
    `).Should(Equal(graphql.ErrorsOf(
			graphql.NewError(
				validator.CycleErrorMessage("fragB", nil),
				[]graphql.ErrorLocation{
					{Line: 3, Column: 31},
				},
			),
			// FIXME: The following order doesn't match graphql-js counterpart.
			graphql.NewError(
				validator.CycleErrorMessage("fragB", []string{"fragC"}),
				[]graphql.ErrorLocation{
					{Line: 3, Column: 41},
					{Line: 4, Column: 41},
				},
			),
			graphql.NewError(
				validator.CycleErrorMessage("fragA", []string{"fragB", "fragC"}),
				[]graphql.ErrorLocation{
					{Line: 2, Column: 31},
					{Line: 3, Column: 41},
					{Line: 4, Column: 31},
				},
			),
		)))
	})
})
