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

// graphql-js/src/validation/__tests__/PossibleFragmentSpreads-test.js@8c96dc8
var _ = Describe("Validate: Possible fragment spreads", func() {
	expectErrors := func(queryStr string) GomegaAssertion {
		return expectValidationErrors(rules.PossibleFragmentSpreads{}, queryStr)
	}

	expectValid := func(queryStr string) {
		expectErrors(queryStr).Should(Equal(graphql.NoErrors()))
	}

	errorSpread := func(
		fragName string,
		parentType string,
		fragType string,
		line uint,
		column uint) error {

		return graphql.NewError(
			validator.TypeIncompatibleSpreadMessage(fragName, parentType, fragType),
			[]graphql.ErrorLocation{
				{Line: line, Column: column},
			},
		)
	}

	errorAnon := func(
		parentType string,
		fragType string,
		line uint,
		column uint) error {

		return graphql.NewError(
			validator.TypeIncompatibleAnonSpreadMessage(parentType, fragType),
			[]graphql.ErrorLocation{
				{Line: line, Column: column},
			},
		)
	}

	It("of the same object", func() {
		expectValid(`
      fragment objectWithinObject on Dog { ...dogFragment }
      fragment dogFragment on Dog { barkVolume }
    `)
	})

	It("of the same object with inline fragment", func() {
		expectValid(`
      fragment objectWithinObjectAnon on Dog { ... on Dog { barkVolume } }
    `)
	})

	It("object into an implemented interface", func() {
		expectValid(`
      fragment objectWithinInterface on Pet { ...dogFragment }
      fragment dogFragment on Dog { barkVolume }
    `)
	})

	It("object into containing union", func() {
		expectValid(`
      fragment objectWithinUnion on CatOrDog { ...dogFragment }
      fragment dogFragment on Dog { barkVolume }
    `)
	})

	It("union into contained object", func() {
		expectValid(`
      fragment unionWithinObject on Dog { ...catOrDogFragment }
      fragment catOrDogFragment on CatOrDog { __typename }
    `)
	})

	It("union into overlapping interface", func() {
		expectValid(`
      fragment unionWithinInterface on Pet { ...catOrDogFragment }
      fragment catOrDogFragment on CatOrDog { __typename }
    `)
	})

	It("union into overlapping union", func() {
		expectValid(`
      fragment unionWithinUnion on DogOrHuman { ...catOrDogFragment }
      fragment catOrDogFragment on CatOrDog { __typename }
    `)
	})

	It("interface into implemented object", func() {
		expectValid(`
      fragment interfaceWithinObject on Dog { ...petFragment }
      fragment petFragment on Pet { name }
    `)
	})

	It("interface into overlapping interface", func() {
		expectValid(`
      fragment interfaceWithinInterface on Pet { ...beingFragment }
      fragment beingFragment on Being { name }
    `)
	})

	It("interface into overlapping interface in inline fragment", func() {
		expectValid(`
      fragment interfaceWithinInterface on Pet { ... on Being { name } }
    `)
	})

	It("interface into overlapping union", func() {
		expectValid(`
      fragment interfaceWithinUnion on CatOrDog { ...petFragment }
      fragment petFragment on Pet { name }
    `)
	})

	It("ignores incorrect type (caught by FragmentsOnCompositeTypes)", func() {
		expectValid(`
      fragment petFragment on Pet { ...badInADifferentWay }
      fragment badInADifferentWay on String { name }
    `)
	})

	It("different object into object", func() {
		expectErrors(`
      fragment invalidObjectWithinObject on Cat { ...dogFragment }
      fragment dogFragment on Dog { barkVolume }
    `).Should(Equal(graphql.ErrorsOf(errorSpread("dogFragment", "Cat", "Dog", 2, 51))))
	})

	It("different object into object in inline fragment", func() {
		expectErrors(`
      fragment invalidObjectWithinObjectAnon on Cat {
        ... on Dog { barkVolume }
      }
    `).Should(Equal(graphql.ErrorsOf(errorAnon("Cat", "Dog", 3, 9))))
	})

	It("object into not implementing interface", func() {
		expectErrors(`
      fragment invalidObjectWithinInterface on Pet { ...humanFragment }
      fragment humanFragment on Human { pets { name } }
    `).Should(Equal(graphql.ErrorsOf(errorSpread("humanFragment", "Pet", "Human", 2, 54))))
	})

	It("object into not containing union", func() {
		expectErrors(`
      fragment invalidObjectWithinUnion on CatOrDog { ...humanFragment }
      fragment humanFragment on Human { pets { name } }
    `).Should(Equal(graphql.ErrorsOf(errorSpread("humanFragment", "CatOrDog", "Human", 2, 55))))
	})

	It("union into not contained object", func() {
		expectErrors(`
      fragment invalidUnionWithinObject on Human { ...catOrDogFragment }
      fragment catOrDogFragment on CatOrDog { __typename }
    `).Should(Equal(graphql.ErrorsOf(errorSpread("catOrDogFragment", "Human", "CatOrDog", 2, 52))))
	})

	It("union into non overlapping interface", func() {
		expectErrors(`
      fragment invalidUnionWithinInterface on Pet { ...humanOrAlienFragment }
      fragment humanOrAlienFragment on HumanOrAlien { __typename }
    `).Should(Equal(graphql.ErrorsOf(
			errorSpread("humanOrAlienFragment", "Pet", "HumanOrAlien", 2, 53),
		)))
	})

	It("union into non overlapping union", func() {
		expectErrors(`
      fragment invalidUnionWithinUnion on CatOrDog { ...humanOrAlienFragment }
      fragment humanOrAlienFragment on HumanOrAlien { __typename }
    `).Should(Equal(graphql.ErrorsOf(
			errorSpread("humanOrAlienFragment", "CatOrDog", "HumanOrAlien", 2, 54),
		)))
	})

	It("interface into non implementing object", func() {
		expectErrors(`
      fragment invalidInterfaceWithinObject on Cat { ...intelligentFragment }
      fragment intelligentFragment on Intelligent { iq }
    `).Should(Equal(graphql.ErrorsOf(
			errorSpread("intelligentFragment", "Cat", "Intelligent", 2, 54),
		)))
	})

	It("interface into non overlapping interface", func() {
		expectErrors(`
      fragment invalidInterfaceWithinInterface on Pet {
        ...intelligentFragment
      }
      fragment intelligentFragment on Intelligent { iq }
    `).Should(Equal(graphql.ErrorsOf(
			errorSpread("intelligentFragment", "Pet", "Intelligent", 3, 9),
		)))
	})

	It("interface into non overlapping interface in inline fragment", func() {
		expectErrors(`
      fragment invalidInterfaceWithinInterfaceAnon on Pet {
        ...on Intelligent { iq }
      }
    `).Should(Equal(graphql.ErrorsOf(errorAnon("Pet", "Intelligent", 3, 9))))
	})

	It("interface into non overlapping union", func() {
		expectErrors(`
      fragment invalidInterfaceWithinUnion on HumanOrAlien { ...petFragment }
      fragment petFragment on Pet { name }
    `).Should(Equal(graphql.ErrorsOf(errorSpread("petFragment", "HumanOrAlien", "Pet", 2, 62))))
	})
})
