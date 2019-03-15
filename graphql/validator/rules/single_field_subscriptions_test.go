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

// graphql-js/src/validation/__tests__/SingleFieldSubscriptions-test.js@8c96dc8
var _ = Describe("Validate: Anonymous operation must be alone", func() {
	expectErrors := func(queryStr string) GomegaAssertion {
		return expectValidationErrors(rules.SingleFieldSubscriptions{}, queryStr)
	}

	expectValid := func(queryStr string) {
		expectErrors(queryStr).Should(Equal(graphql.NoErrors()))
	}

	singleFieldOnlyMessage := func(name string, locations ...graphql.ErrorLocation) error {
		return graphql.NewError(validator.SingleFieldOnlyMessage(name), locations)
	}

	It("valid subscription", func() {
		expectValid(`
      subscription ImportantEmails {
        importantEmails
      }
    `)
	})

	It("fails with more than one root field", func() {
		expectErrors(`
      subscription ImportantEmails {
        importantEmails
        notImportantEmails
      }
    `).Should(Equal(graphql.ErrorsOf(
			singleFieldOnlyMessage("ImportantEmails", graphql.ErrorLocation{
				Line:   4,
				Column: 9,
			})),
		))
	})

	It("fails with more than one root field including introspection", func() {
		expectErrors(`
      subscription ImportantEmails {
        importantEmails
        __typename
      }
    `).Should(Equal(graphql.ErrorsOf(
			singleFieldOnlyMessage("ImportantEmails", graphql.ErrorLocation{
				Line:   4,
				Column: 9,
			})),
		))
	})

	It("fails with many more than one root field", func() {
		expectErrors(`
      subscription ImportantEmails {
        importantEmails
        notImportantEmails
        spamEmails
      }
    `).Should(Equal(graphql.ErrorsOf(
			singleFieldOnlyMessage(
				"ImportantEmails",
				graphql.ErrorLocation{
					Line:   4,
					Column: 9,
				},
				graphql.ErrorLocation{
					Line:   5,
					Column: 9,
				})),
		))
	})

	It("fails with more than one root field in anonymous subscriptions", func() {
		expectErrors(`
      subscription {
        importantEmails
        notImportantEmails
      }
    `).Should(Equal(graphql.ErrorsOf(
			singleFieldOnlyMessage("", graphql.ErrorLocation{
				Line:   4,
				Column: 9,
			})),
		))
	})

})
