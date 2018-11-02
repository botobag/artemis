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
	"math"
	"sort"
	"strings"
)

type suggestionListSorter struct {
	options   []string
	distances []int
}

var _ sort.Interface = (*suggestionListSorter)(nil)

// Len implements sort.Interface.
func (s *suggestionListSorter) Len() int {
	return len(s.options)
}

// Swap implements sort.Interface.
func (s *suggestionListSorter) Swap(i, j int) {
	s.options[i], s.options[j] = s.options[j], s.options[i]
	s.distances[i], s.distances[j] = s.distances[j], s.distances[i]
}

// Less implements sort.Interface.
func (s *suggestionListSorter) Less(i, j int) bool {
	return s.distances[i] < s.distances[j]
}

// SuggestionList given an invalid input string and a list of valid options, returns a filtered
// list of valid options sorted based on their similarity with the input.
func SuggestionList(input string, options []string) []string {
	numOptions := len(options)
	if numOptions > 0 {
		var (
			filterOptions   []string
			optionDistances []int
		)
		inputThreshold := float64(len(input)) / 2.0
		for _, option := range options {
			distance := lexicalDistance(input, option)
			threshold := math.Max(math.Max(inputThreshold, float64(len(option))/2.0), 1)
			if float64(distance) <= threshold {
				filterOptions = append(filterOptions, option)
				optionDistances = append(optionDistances, distance)
			}
		}

		// Sort option by their distance.
		sort.Sort(&suggestionListSorter{filterOptions, optionDistances})
		return filterOptions
	}

	return nil
}

// Computes the lexical distance between strings A and B.
//
// The "distance" between two strings is given by counting the minimum number of edits needed to
// transform string A into string B. An edit can be an insertion, deletion, or substitution of a
// single character, or a swap of two adjacent characters.
//
// Includes a custom alteration from Damerau-Levenshtein to treat case changes as a single edit
// which helps identify mis-cased values with an edit distance of 1.
//
// This distance can be useful for detecting typos in input or sorting.
func lexicalDistance(aStr string, bStr string) int {
	if aStr == bStr {
		return 0
	}

	a := strings.ToLower(aStr)
	b := strings.ToLower(bStr)
	aLength := len(a)
	bLength := len(b)
	d := make([][]int, aLength+1)

	// Any case change counts as a single edit
	if a == b {
		return 1
	}

	for i := 0; i <= aLength; i++ {
		d[i] = make([]int, bLength+1)
		d[i][0] = i
	}

	for j := 1; j <= bLength; j++ {
		d[0][j] = j
	}

	for i := 1; i <= aLength; i++ {
		for j := 1; j <= bLength; j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}

			x := d[i-1][j] + 1
			y := d[i][j-1] + 1
			z := d[i-1][j-1] + cost

			// Select min(x, y, z).
			min := x
			if y < min {
				min = y
			}
			if z < min {
				min = z
			}

			// Account adjacent swap.
			if i > 1 && j > 1 {
				if a[i-1] == b[j-2] && a[i-2] == b[j-1] {
					w := d[i-2][j-2] + cost
					if w < min {
						min = w
					}
				}
			}

			d[i][j] = min
		}
	}

	return d[aLength][bLength]
}
