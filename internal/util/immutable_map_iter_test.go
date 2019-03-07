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

package util_test

import (
	"fmt"
	"sort"
	"strings"

	"github.com/botobag/artemis/internal/util"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/types"
)

// The tests are modified from Go's source for reflect.MapIter in Gomega styles:
//
// https://go.googlesource.com/go/+/3b66c00/src/reflect/all_test.go#6593
//
// The license is reproduced below.

/**
 * Copyright (c) 2009 The Go Authors. All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions are
 * met:
 *
 *    * Redistributions of source code must retain the above copyright
 * notice, this list of conditions and the following disclaimer.
 *    * Redistributions in binary form must reproduce the above
 * copyright notice, this list of conditions and the following disclaimer
 * in the documentation and/or other materials provided with the
 * distribution.
 *    * Neither the name of Google Inc. nor the names of its
 * contributors may be used to endorse or promote products derived from
 * this software without specific prior written permission.
 *
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
 * "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
 * LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
 * A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
 * OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
 * SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
 * LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
 * DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
 * THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 * (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
 * OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 */

type iterateToStringMatcher struct {
	expected string
	actual   string
}

func (matcher *iterateToStringMatcher) Match(actual interface{}) (success bool, err error) {
	var (
		got []string
		// Assume util.ImmutableMapIter.
		it = actual.(*util.ImmutableMapIter)
	)
	for it.Next() {
		line := fmt.Sprintf("%v: %v", it.Key(), it.Value())
		got = append(got, line)
	}
	sort.Strings(got)
	matcher.actual = "[" + strings.Join(got, ", ") + "]"
	return matcher.actual == matcher.expected, nil
}

func (matcher *iterateToStringMatcher) FailureMessage(actual interface{}) (message string) {
	return format.Message(matcher.actual, "to equal")
}

func (matcher *iterateToStringMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return format.Message(matcher.actual, "not to equal")
}

func IterateToString(expected string) types.GomegaMatcher {
	return &iterateToStringMatcher{
		expected: expected,
	}
}

var _ = Describe("ImmutableMapIter", func() {
	It("iterates non-empty map", func() {
		m := map[string]int{"one": 1, "two": 2, "three": 3}
		iter := util.NewImmutableMapIter(m)
		Expect(iter).Should(IterateToString(`[one: 1, three: 3, two: 2]`))
	})

	It("iterates nil map", func() {
		var m map[string]int
		iter := util.NewImmutableMapIter(m)
		Expect(iter).Should(IterateToString(`[]`))
	})

	It("panics when it is initialized with a non-map value", func() {
		Expect(func() {
			util.NewImmutableMapIter(0)
		}).Should(Panic())
	})

	It("panics when using zero iterator", func() {
		Expect(func() {
			new(util.ImmutableMapIter).Key()
		}).Should(Panic())

		Expect(func() {
			new(util.ImmutableMapIter).Value()
		}).Should(Panic())

		Expect(func() {
			new(util.ImmutableMapIter).Next()
		}).Should(Panic())
	})

	It("panics when calling Key/Value on an iterator before Next", func() {
		var m map[string]int
		iter := util.NewImmutableMapIter(m)

		Expect(func() {
			iter.Key()
		}).Should(Panic())

		Expect(func() {
			iter.Value()
		}).Should(Panic())
	})

	It("panics when calling Next, Key, or Value on an exhausted iterator", func() {
		var m map[string]int
		iter := util.NewImmutableMapIter(m)

		Expect(iter.Next()).Should(BeFalse())

		Expect(func() {
			new(util.ImmutableMapIter).Key()
		}).Should(Panic())

		Expect(func() {
			new(util.ImmutableMapIter).Value()
		}).Should(Panic())

		Expect(func() {
			new(util.ImmutableMapIter).Next()
		}).Should(Panic())
	})

	It("reflects any insertions to the map since the iterator was created on first call to Next", func() {
		m := map[string]int{}
		iter := util.NewImmutableMapIter(m)
		m["one"] = 1
		Expect(iter).Should(IterateToString(`[one: 1]`))
	})

	It("reflects deletion of all elements before first iteration", func() {
		m := map[string]int{"one": 1, "two": 2, "three": 3}
		iter := util.NewImmutableMapIter(m)
		delete(m, "one")
		delete(m, "two")
		delete(m, "three")
		Expect(iter).Should(IterateToString(`[]`))
	})
})
