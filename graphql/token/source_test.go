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

package token_test

import (
	"regexp"

	"github.com/botobag/artemis/graphql/token"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type SourceTest struct {
	name         string
	source       []byte
	size         uint
	lineOffset   uint
	columnOffset uint
}

var lineRegexp = regexp.MustCompile("\r\n|[\n\r]")

// Reference implementation from graphql-go [0].
//
// [0]: https://github.com/graphql-go/graphql/blob/a7e15c0/language/location/location.go#L14
func getLocationReference(source []byte, position uint) (uint, uint) {
	var (
		line   uint = 1
		column      = position + 1
	)

	matches := lineRegexp.FindAllIndex(source, -1)
	for _, match := range matches {
		matchIndex := uint(match[0])
		if matchIndex < position {
			line++
			column = position + 1 - (matchIndex + uint(match[1]-match[0]))
			continue
		}
		break
	}

	return line, column
}

func verifyLocationInfo(test *SourceTest) {
	sourceBody := test.source
	Expect(uint(len(sourceBody))).Should(Equal(test.size), test.name)

	source := token.NewSource(&token.SourceConfig{
		Body:         token.SourceBody(sourceBody),
		Name:         test.name,
		LineOffset:   test.lineOffset,
		ColumnOffset: test.columnOffset,
	})
	Expect(source).ShouldNot(BeNil(), test.name)

	for pos := uint(0); pos < test.size; pos++ {
		location := source.LocationFromPos(pos)
		locationInfo := source.LocationInfoOf(location)

		line, column := getLocationReference(sourceBody, pos)
		Expect(locationInfo).Should(Equal(token.SourceLocationInfo{
			Name:   test.name,
			Line:   line + test.lineOffset,
			Column: column + test.columnOffset,
		}), "pos = %d", pos)
	}
}

var _ = Describe("Source", func() {
	It("accepts nil Body", func() {
		Expect(token.NewSource(&token.SourceConfig{})).ShouldNot(BeNil())
	})

	It("converts offset into SourceLocation", func() {
		body := token.SourceBody([]byte("hello"))
		source := token.NewSource(&token.SourceConfig{
			Body: body,
		})
		Expect(source).ShouldNot(BeNil())

		// Valid offsets are converted to an unique offset.
		locations := map[token.SourceLocation]bool{}
		for pos := range body {
			location := source.LocationFromPos(uint(pos))
			Expect(locations).ShouldNot(ContainElement(location))
			locations[location] = true
			// Can convert back to position with PosFromLocation.
			Expect(source.PosFromLocation(location)).Should(Equal(uint(pos)))
		}

		// Offset just passes the end of source is accepted.
		endLocation := source.LocationFromPos(uint(len(body)))
		Expect(locations).ShouldNot(ContainElement(endLocation))
		Expect(source.PosFromLocation(endLocation)).Should(Equal(uint(len(body))))

		// Invalid offset causes panics.
		for pos := len(body) + 1; pos < len(body)+10; pos++ {
			Expect(func() {
				_ = source.LocationFromPos(uint(pos))
			}).Should(Panic())
		}

		// Invalid location causes panics.
		Expect(func() {
			_ = source.PosFromLocation(token.NoSourceLocation)
		}).Should(Panic())
	})

	Describe("converts SourceLocation into LocationInfo", func() {
		It("accepts empty source", func() {
			verifyLocationInfo(&SourceTest{
				name:   "empty-source",
				source: []byte{},
				size:   0,
			})
		})

		It("accepts one line text", func() {
			verifyLocationInfo(&SourceTest{
				name:   "one-line-source",
				source: []byte("0123456789"),
				size:   10,
			})
		})

		It("accepts many empty lines", func() {
			verifyLocationInfo(&SourceTest{
				name:   "many-empty-lines-source",
				source: []byte("\n\n\n\n\n\n\n\n\n\n"),
				size:   10,
			})
		})

		It("accepts carriage return as newline", func() {
			verifyLocationInfo(&SourceTest{
				name:   "carriage-return-as-newlines-source",
				source: []byte("0\r1\r2\r3\r4\r5\r6\r7\r8\r9\r"),
				size:   20,
			})
		})

		It("accepts line feed with carriage return as two newlines", func() {
			verifyLocationInfo(&SourceTest{
				name:   "line-feed-with-carriage-return-is-two-newlines-source",
				source: []byte("0\n\r1\n\r2\n\r3\n\r4\n\r5\n\r6\n\r7\n\r8\n\r9\n\r"),
				size:   30,
			})
		})

		It("accepts carriage return with line feed as two newlines", func() {
			verifyLocationInfo(&SourceTest{
				name:   "carriage-return-with-line-feed-is-one-newlines-source",
				source: []byte("0\r\n1\r\n2\r\n3\r\n4\r\n5\r\n6\r\n7\r\n8\r\n9\r\n"),
				size:   30,
			})
		})

		It("accepts line offset and column offset two newlines", func() {
			verifyLocationInfo(&SourceTest{
				name:         "with-column-offset-source",
				source:       []byte("abcde"),
				size:         5,
				lineOffset:   40,
				columnOffset: 10,
			})
		})

		It("accepts invalid SourceLoction", func() {
			source := token.NewSource(&token.SourceConfig{
				Name: "test",
				Body: token.SourceBody([]byte("test source")),
			})
			Expect(source.LocationInfoOf(token.NoSourceLocation)).Should(Equal(token.SourceLocationInfo{
				Name: "test",
			}))
		})
	})
})
