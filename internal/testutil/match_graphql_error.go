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

package testutil

import (
	"github.com/botobag/artemis/graphql"

	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
)

// ErrorFieldsMatcher sets up fields to match.
type ErrorFieldsMatcher func(gstruct.Fields)

// MessageEqual matches message in a graphql.Error to be the same as the specified string.
func MessageEqual(s string) ErrorFieldsMatcher {
	return func(fields gstruct.Fields) {
		fields["Message"] = gomega.Equal(s)
	}
}

// MessageContainSubstring matches message in a graphql.Error to contain the specified string.
func MessageContainSubstring(s string) ErrorFieldsMatcher {
	return func(fields gstruct.Fields) {
		fields["Message"] = gomega.ContainSubstring(s)
	}
}

// LocationEqual matches the locations in the error to contain the only specified location.
func LocationEqual(location graphql.ErrorLocation) ErrorFieldsMatcher {
	return func(fields gstruct.Fields) {
		fields["Locations"] = gomega.Equal([]graphql.ErrorLocation{location})
	}
}

// LocationsConsistOf matches locations in the error to include all given locations.
func LocationsConsistOf(locations []graphql.ErrorLocation) ErrorFieldsMatcher {
	return func(fields gstruct.Fields) {
		fields["Locations"] = gomega.ConsistOf(locations)
	}
}

// KindIs matches the kind in the error to be the same as the given one.
func KindIs(errKind graphql.ErrKind) ErrorFieldsMatcher {
	return func(fields gstruct.Fields) {
		fields["Kind"] = gomega.Equal(errKind)
	}
}

// MatchGraphQLError matches a graphql.Error with given fields.
//
// The following example matches a graphql.Error including "Unterminated string" in the message and
// the error kind should match graphql.ErrKindSyntax.
//
//		Expect(err).Should(MatchGraphQLError(
//			MessageContainSubstring("Unterminated string"),
//			KindIs(graphql.ErrKindSyntax),
//		))
func MatchGraphQLError(matchers ...ErrorFieldsMatcher) types.GomegaMatcher {
	fields := gstruct.Fields{}
	for _, matcher := range matchers {
		matcher(fields)
	}
	return gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, fields))
}

// ConsistOfGraphQLErrors is used to match a graphql.Errors like an array of graphql.Error's with
// Gomega's ConsistOf.
//
//		Expect(errs).Should(ConsistOfGraphQLErrors(
//			MatchGraphQLError(
//				MessageContainSubstring("First error"),
//				KindIs(graphql.ErrKindSyntax),
//			),
//			MatchGraphQLError(
//				MessageContainSubstring("Second error"),
//			),
//		))
func ConsistOfGraphQLErrors(matchers ...interface{}) types.GomegaMatcher {
	return gstruct.MatchAllFields(gstruct.Fields{
		"Errors": gomega.ConsistOf(matchers...),
	})
}
