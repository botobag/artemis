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
	"strings"

	"github.com/botobag/artemis/internal/util"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("dedent", func() {
	// graphql-js/src/jsutils/__tests__/dedent-test.js@8c96dc8
	It("removes indentation in typical usage", func() {
		output := util.Dedent(`
      type Query {
        me: User
      }

      type User {
        id: ID
        name: String
      }
    `)

		Expect(output).Should(Equal(strings.Join([]string{
			"type Query {",
			"  me: User",
			"}",
			"",
			"type User {",
			"  id: ID",
			"  name: String",
			"}",
			"",
		}, "\n")))
	})

	It("removes only the first level of indentation", func() {
		output := util.Dedent(`
            qux
              quux
                quuux
                  quuuux
    `)

		Expect(output).Should(Equal(strings.Join([]string{
			"qux",
			"  quux",
			"    quuux",
			"      quuuux",
			"",
		}, "\n")))
	})

	It("does not escape special characters", func() {
		output := util.Dedent(`
      type Root {
        field(arg: String = "wi\th de\fault"): String
      }
    `)

		Expect(output).Should(Equal(strings.Join([]string{
			`type Root {`,
			`  field(arg: String = "wi\th de\fault"): String`,
			`}`,
			``,
		}, "\n")))
	})

	// It("also works as an ordinary function on strings", func() {
	// })

	It("also removes indentation using tabs", func() {
		output := util.Dedent(`
        		    type Query {
        		      me: User
        		    }
    `)

		Expect(output).Should(Equal(strings.Join([]string{
			"type Query {",
			"  me: User",
			"}",
			"",
		}, "\n")))
	})

	It("removes leading newlines", func() {
		output := util.Dedent(`


      type Query {
        me: User
      }`)

		Expect(output).Should(Equal(strings.Join([]string{
			"type Query {",
			"  me: User",
			"}",
		}, "\n")))
	})

	It("does not remove trailing newlines", func() {
		output := util.Dedent(`
      type Query {
        me: User
      }

    `)

		Expect(output).Should(Equal(strings.Join([]string{
			"type Query {",
			"  me: User",
			"}",
			"",
			"",
		}, "\n")))
	})

	It("removes all trailing spaces and tabs", func() {
		output := util.Dedent(`
      type Query {
        me: User
      }
          		  	 `)
		Expect(output).Should(Equal(strings.Join([]string{
			"type Query {",
			"  me: User",
			"}",
			"",
		}, "\n")))
	})

	It("works on text without leading newline", func() {
		output := util.Dedent(`      type Query {
        me: User
      }`)

		Expect(output).Should(Equal(strings.Join([]string{
			"type Query {",
			"  me: User",
			"}",
		}, "\n")))
	})

	It("works on empty string", func() {
		Expect(util.Dedent("")).Should(Equal(""))
	})

	It("works on string without any identation", func() {
		output := util.Dedent(`
type Query {
  me: User
}
`)

		Expect(output).Should(Equal(strings.Join([]string{
			"type Query {",
			"  me: User",
			"}",
			"",
		}, "\n")))
	})

	// It("supports expression interpolation", func() {
	// })
})
