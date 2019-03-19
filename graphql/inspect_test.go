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

package graphql_test

import (
	"errors"
	"io"
	"math"

	"github.com/botobag/artemis/graphql"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func testFunc() {}

type objectWithCustomInspect struct {
	str string
}

func (o objectWithCustomInspect) Inspect(out io.Writer) error {
	s := o.str
	if len(s) == 0 {
		s = "<custom inspect>"
	}
	if _, err := out.Write([]byte(s)); err != nil {
		return err
	}
	return nil
}

type objectWithErrCustomInspect struct {
	err error
}

func (o objectWithErrCustomInspect) Inspect(out io.Writer) error {
	return o.err
}

// graphql-js/src/jsutils/__tests__/inspect-test.js@ffccf3f
var _ = Describe("Inspect", func() {
	It("null", func() {
		Expect(graphql.Inspect(nil)).Should(Equal("null"))
	})

	It("boolean", func() {
		Expect(graphql.Inspect(true)).Should(Equal("true"))
		Expect(graphql.Inspect(false)).Should(Equal("false"))
	})

	It("string", func() {
		Expect(graphql.Inspect("")).Should(Equal(`""`))
		Expect(graphql.Inspect("abc")).Should(Equal(`"abc"`))
		Expect(graphql.Inspect("\"")).Should(Equal(`"\""`))
	})

	It("number", func() {
		Expect(graphql.Inspect(0)).Should(Equal(`0`))
		Expect(graphql.Inspect(3.14)).Should(Equal(`3.14`))
		Expect(graphql.Inspect(math.NaN())).Should(Equal(`NaN`))
		// The following are different than graphql-js which are printed as Infinity and -Infinity,
		// repectively.
		Expect(graphql.Inspect(math.Inf(+1))).Should(Equal(`+Inf`))
		Expect(graphql.Inspect(math.Inf(-1))).Should(Equal(`-Inf`))
	})

	It("function", func() {
		Expect(graphql.Inspect(func() int { return 0 })).Should(MatchRegexp(`^\[function github.com/botobag/artemis/graphql_test\.glob.+\]$`))
		Expect(graphql.Inspect(testFunc)).Should(MatchRegexp(`^\[function github.com/botobag/artemis/graphql_test\.testFunc]$`))
	})

	It("array", func() {
		Expect(graphql.Inspect([]interface{}(nil))).Should(Equal(`[]`))
		Expect(graphql.Inspect([]interface{}{})).Should(Equal(`[]`))
		Expect(graphql.Inspect([]interface{}{nil})).Should(Equal(`[null]`))
		Expect(graphql.Inspect([]interface{}{1, math.NaN()})).Should(Equal(`[1, NaN]`))
		Expect(graphql.Inspect([]interface{}{
			[]string{"a", "b"},
			"c",
		})).Should(Equal(`[["a", "b"], "c"]`))

		Expect(graphql.Inspect([][][]interface{}{{{}}})).Should(Equal(`[[[]]]`))
		Expect(graphql.Inspect([][][]interface{}{{{"a"}}})).Should(Equal(`[[[Array]]]`))
		Expect(graphql.Inspect([][][]interface{}{{{"a"}, {"b"}}})).Should(Equal(`[[[Array], [Array]]]`))

		Expect(graphql.Inspect([]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9})).Should(Equal(
			`[0, 1, 2, 3, 4, 5, 6, 7, 8, 9]`,
		))
		Expect(graphql.Inspect([]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10})).Should(Equal(
			`[0, 1, 2, 3, 4, 5, 6, 7, 8, 9, ... 1 more item]`,
		))
		Expect(graphql.Inspect([]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11})).Should(Equal(
			`[0, 1, 2, 3, 4, 5, 6, 7, 8, 9, ... 2 more items]`,
		))
	})

	It("object", func() {
		Expect(graphql.Inspect(struct{}{})).Should(Equal(`{}`))
		Expect(graphql.Inspect((*struct{})(nil))).Should(Equal(`null`))

		Expect(graphql.Inspect(struct {
			A int
		}{
			A: 1,
		})).Should(Equal(`{ A: 1 }`))

		Expect(graphql.Inspect(struct {
			A int
			B int
		}{
			A: 1,
			B: 2,
		})).Should(Equal(`{ A: 1, B: 2 }`))

		Expect(graphql.Inspect(struct {
			Array []interface{}
		}{
			Array: []interface{}{nil, 0},
		})).Should(Equal(`{ Array: [null, 0] }`))

		Expect(graphql.Inspect(struct {
			A struct {
				B struct{}
			}
		}{})).Should(Equal(`{ A: { B: {} } }`))

		Expect(graphql.Inspect(struct {
			A struct {
				B struct {
					C int
				}
			}
		}{
			A: struct {
				B struct {
					C int
				}
			}{
				B: struct {
					C int
				}{
					C: 1,
				},
			},
		})).Should(Equal(`{ A: { B: [Object] } }`))

		// Named struct
		type Depth3 struct {
			D4 int
		}
		type Depth2 struct {
			D3 *Depth3
		}
		type Depth1 struct {
			D2 *Depth2
		}

		Expect(graphql.Inspect(&Depth1{
			D2: &Depth2{
				D3: &Depth3{
					D4: 1,
				},
			},
		})).Should(Equal(`{ D2: { D3: [Depth3] } }`))

		Expect(graphql.Inspect(struct {
			A bool
			B interface{}
		}{
			A: true,
			B: nil,
		})).Should(Equal(`{ A: true, B: null }`))

		// Unexported field won't be shown (and cannot be shown).
		Expect(graphql.Inspect(struct {
			A bool
			b interface{}
			C string
		}{
			A: true,
			C: "c",
		})).Should(Equal(`{ A: true, C: "c" }`))

		Expect(graphql.Inspect(struct {
			a bool
			b interface{}
		}{})).Should(Equal(`{}`))
	})

	It("map", func() {
		Expect(graphql.Inspect(map[string]interface{}(nil))).Should(Equal(`{}`))

		Expect(graphql.Inspect(map[string]interface{}{
			"a": 1,
		})).Should(Equal(`{ "a": 1 }`))

		Expect(graphql.Inspect(map[string]interface{}{
			"a": 1,
			"b": 2,
		})).Should(Or(
			Equal(`{ "a": 1, "b": 2 }`),
			Equal(`{ "b": 2, "a": 1 }`),
		))

		Expect(graphql.Inspect(map[string]interface{}{
			"array": []interface{}{nil, 0},
		})).Should(Equal(`{ "array": [null, 0] }`))

		Expect(graphql.Inspect(map[string]interface{}{
			"a": true,
			"b": nil,
		})).Should(Or(
			Equal(`{ "a": true, "b": null }`),
			Equal(`{ "b": null, "a": true }`),
		))

		Expect(graphql.Inspect(map[string]map[string]map[string]int{
			"a": {
				"b": {
					"c": 1,
				},
			},
		})).Should(Equal(`{ "a": { "b": [Map] } }`))

		circularMap := map[string]interface{}{}
		circularMap["a"] = circularMap
		Expect(graphql.Inspect(circularMap)).Should(Equal(`{ "a": [Circular] }`))
	})

	It("pointer", func() {
		s := "Hello, World!"
		Expect(graphql.Inspect(&s)).Should(Equal(`"Hello, World!"`))
	})

	It("custom inspect", func() {
		Expect(graphql.Inspect(objectWithCustomInspect{})).Should(Equal(`<custom inspect>`))
	})

	It("custom inspect function that uses this", func() {
		Expect(graphql.Inspect(objectWithCustomInspect{"Hello World!"})).Should(Equal(`Hello World!`))
	})

	It("custom inspect function that returns error", func() {
		Expect(func() {
			graphql.Inspect(objectWithErrCustomInspect{errors.New("error")})
		}).Should(Panic())
	})

	It("detect circular objects", func() {
		obj := &struct {
			Self     interface{}
			DeepSelf interface{}
		}{}
		obj.Self = obj
		obj.DeepSelf = &struct {
			Self interface{}
		}{
			Self: obj,
		}

		Expect(graphql.Inspect(obj)).Should(Equal(`{ Self: [Circular], DeepSelf: { Self: [Circular] } }`))

		array := make([]interface{}, 2)
		array[0] = array
		array[1] = []interface{}{array}
		Expect(graphql.Inspect(array)).Should(Equal(`[[Circular], [[Circular]]]`))

		mixed := &struct {
			Array []interface{}
		}{}
		mixed.Array = []interface{}{mixed}
		Expect(graphql.Inspect(mixed)).Should(Equal("{ Array: [[Circular]] }"))
	})

	It("Use class names for the shortform of an object", func() {
		type Foo struct {
			Foo string
		}
		Expect(graphql.Inspect([][]*Foo{{&Foo{}}})).Should(Equal("[[[Foo]]]"))
	})
})
