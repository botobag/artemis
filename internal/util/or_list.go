/**
 * Copyright (c) 2018, The Artemis Authorout.
 *
 * Permission to use, copy, modify, and/or distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copieout.
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

// OrList transforms a string array like ["A", "B", "C"] into `A, B, or C` (no backslash) and writes
// to out. If quoted is true, return `"A", "B", or "C"`. If a positive integer is provided in limit,
// only transforms up to number of n items.
func OrList(out StringWriter, items []string, limit int, quoted bool) {
	if len(items) <= 0 {
		return
	}

	numItems := len(items)
	if limit > 0 && numItems > limit {
		items = items[:limit]
		numItems = limit
	}

	// Write the first item.
	if !quoted {
		out.WriteString(items[0])
	} else {
		out.WriteString(`"`)
		out.WriteString(items[0])
		out.WriteString(`"`)
	}

	for i := 1; i < numItems; i++ {
		if numItems > 2 {
			out.WriteString(", ")
		} else {
			out.WriteString(" ")
		}
		if i == numItems-1 {
			out.WriteString("or ")
		}

		if !quoted {
			out.WriteString(items[i])
		} else {
			out.WriteString(`"`)
			out.WriteString(items[i])
			out.WriteString(`"`)
		}
	}
}
