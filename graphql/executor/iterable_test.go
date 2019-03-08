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

package executor_test

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/botobag/artemis/graphql/executor"
	"github.com/botobag/artemis/iterator"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/types"
)

type iterateAsStringsMatcher struct {
	expected []string
	actual   []string
}

func (matcher *iterateAsStringsMatcher) Match(actual interface{}) (success bool, err error) {
	var (
		got []string
		// Assume an executor.Iterator.
		it = actual.(executor.Iterator)
	)

	for {
		value, err := it.Next()
		if err == iterator.Done {
			break
		} else if err != nil {
			return false, err
		} else {
			got = append(got, fmt.Sprintf("%v", value))
		}
	}
	sort.Strings(got)
	matcher.actual = got
	return reflect.DeepEqual(matcher.actual, matcher.expected), nil
}

func (matcher *iterateAsStringsMatcher) FailureMessage(actual interface{}) (message string) {
	return format.Message(matcher.actual, "to equal")
}

func (matcher *iterateAsStringsMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return format.Message(matcher.actual, "not to equal")
}

func IterateAsStrings(expected []string) types.GomegaMatcher {
	clone := make([]string, len(expected))
	copy(clone, expected)
	sort.Strings(clone)
	return &iterateAsStringsMatcher{
		expected: clone,
	}
}

var _ = Describe("Iterable", func() {
	var testMap1 = map[string]int{
		"a": 1,
		"b": 2,
		"c": 3,
	}

	Describe("MapKeysIterable", func() {
		It("iterates keys in a map", func() {
			iterable := executor.NewMapKeysIterable(testMap1)
			Expect(iterable.Size()).Should(Equal(3))
			Expect(iterable.Iterator()).Should(IterateAsStrings([]string{"a", "b", "c"}))
		})
	})

	Describe("MapValuesIterable", func() {
		It("iterates values in a map", func() {
			iterable := executor.NewMapValuesIterable(testMap1)
			Expect(iterable.Size()).Should(Equal(3))
			Expect(iterable.Iterator()).Should(IterateAsStrings([]string{"1", "2", "3"}))
		})
	})
})
