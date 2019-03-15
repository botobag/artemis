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

package util

import (
	"strings"
)

// Dedent fixes indentation by removing leading spaces and tabs from each line.
func Dedent(s string) string {
	n := len(s)

	// Remove leading newlines.
	for n > 0 {
		if s[0] != '\n' {
			break
		}
		s = s[1:]
		n--
	}

	// Remove trailing spaces and tabs.
	for n > 0 {
		r := s[n-1]
		if r != '\t' && r != ' ' {
			break
		}
		s = s[:n-1]
		n--
	}

	// Find the indent from the first line.
	indent := s
	for i := 0; i < n; i++ {
		if s[i] != '\t' && s[i] != ' ' {
			indent = s[:i]
			break
		}
	}

	if len(indent) > 0 {
		return strings.Replace(s[len(indent):], "\n"+indent, "\n", -1)
	}

	return s
}
