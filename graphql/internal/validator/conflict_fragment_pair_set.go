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

package validator

// ConflictFragmentPairSet is used by rules.OverlappingFieldsCanBeMerged to track (and cache) the
// result of conflict checking between two fragments.
type ConflictFragmentPairSet struct {
	data map[string]map[string]bool
}

// NewConflictFragmentPairSet initializes an empty ConflictFragmentPairSet.
func NewConflictFragmentPairSet() ConflictFragmentPairSet {
	return ConflictFragmentPairSet{
		data: map[string]map[string]bool{},
	}
}

func (pairSet ConflictFragmentPairSet) add(a string, b string, areMutuallyExclusive bool) {
	m := pairSet.data[a]
	if m == nil {
		m = map[string]bool{}
		pairSet.data[a] = m
	}
	m[b] = areMutuallyExclusive
}

// Add adds a pair of FragmentSpread with result of conflicting check.
func (pairSet ConflictFragmentPairSet) Add(a string, b string, areMutuallyExclusive bool) {
	pairSet.add(a, b, areMutuallyExclusive)
	pairSet.add(b, a, areMutuallyExclusive)
}

// Has returns true if the
func (pairSet ConflictFragmentPairSet) Has(a string, b string, areMutuallyExclusive bool) bool {
	am, exists := pairSet.data[a]
	if !exists {
		return false
	}
	result, exists := am[b]
	if !exists {
		return false
	}

	// areMutuallyExclusive being false is a superset of being true, hence if we want to know if this
	// PairSet "has" these two with no exclusivity, we have to ensure it was added as such.
	if !areMutuallyExclusive {
		return !result
	}

	return true
}
