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

package lexer

import (
	"regexp"
	"strings"
)

var splitLinesRegex = regexp.MustCompile("\r\n|[\n\r]")

// BlockStringValue Produces the value of a block string from its parsed raw value, similar to
// CoffeeScript's block string, Python's docstring trim or Ruby's strip_heredoc.
//
// This implements the GraphQL spec's BlockStringValue() static algorithm [0].
//
// Implementation was copied from graphql-go [1] which came from graphql-js [2].
//
// [0]: https://graphql.github.io/graphql-spec/June2018/#BlockStringValue()
// [1]: https://github.com/graphql-go/graphql/blob/a7e15c0/language/lexer/lexer.go#L403
// [2]: https://github.com/graphql/graphql-js/blob/7fff8b7/src/language/blockStringValue.js
func BlockStringValue(in string) string {
	// Expand a block string's raw value into independent lines.
	lines := splitLinesRegex.Split(in, -1)

	// Remove common indentation from all lines but first.
	commonIndent := -1
	for i := 1; i < len(lines); i++ {
		line := lines[i]
		indent := leadingWhitespaceLen(line)
		if indent < len(line) && (commonIndent == -1 || indent < commonIndent) {
			commonIndent = indent
			if commonIndent == 0 {
				break
			}
		}
	}

	if commonIndent > 0 {
		for i := 1; i < len(lines); i++ {
			line := lines[i]
			if commonIndent > len(line) {
				lines[i] = ""
			} else {
				lines[i] = line[commonIndent:]
			}
		}
	}

	// Remove leading blank lines.
	for len(lines) > 0 && isBlank(lines[0]) {
		lines = lines[1:]
	}

	// Remove trailing blank lines.
	for len(lines) > 0 && isBlank(lines[len(lines)-1]) {
		lines = lines[:len(lines)-1]
	}

	// Return a string of the lines joined with U+000A.
	return strings.Join(lines, "\n")
}

// leadingWhitespaceLen returns count of whitespace characters on given line.
func leadingWhitespaceLen(in string) (n int) {
	for _, ch := range in {
		if ch == ' ' || ch == '\t' {
			n++
		} else {
			break
		}
	}
	return
}

// isBlank returns true when given line has no content.
func isBlank(in string) bool {
	return leadingWhitespaceLen(in) == len(in)
}
