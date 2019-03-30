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

package util

import (
	"strings"
)

/*
func isSnakeCaseLower(b byte) bool {
	return b >= 'a' && b <= 'z'
}

func toSnakeCaseLower(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b - 'A' + 'a'
	}
	return b
}
*/

func toCamelCaseUpper(b byte) byte {
	if b >= 'a' && b <= 'z' {
		return b - 'a' + 'A'
	}
	return b
}

// CamelCase converts a string of the form "/[_A-Za-z][_0-9A-Za-z]*/" [0] into camel case. For
// example, it returns "CamelCase" for "camel_case".
//
// [0]: https://graphql.github.io/graphql-spec/June2018/#Name
func CamelCase(s string) string {
	sLen := len(s)
	if sLen == 0 {
		return s
	} else if sLen == 1 {
		return strings.ToUpper(s)
	}

	var buf StringBuilder
	buf.Grow(sLen)

	// Handle the first character.
	i := 0
	for i < sLen {
		if s[i] == '_' {
			i++
			continue
		}
		buf.WriteByte(toCamelCaseUpper(s[i]))
		i++
		break
	}

	for ; i < sLen; i++ {
		if s[i] != '_' {
			buf.WriteByte(s[i])
		} else {
			for i < sLen {
				if s[i] == '_' {
					i++
					continue
				}
				buf.WriteByte(toCamelCaseUpper(s[i]))
				break
			}
		}
	}

	return buf.String()
}
