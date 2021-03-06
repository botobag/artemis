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

package lexer_test

import (
	"strings"

	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/graphql/lexer"
	"github.com/botobag/artemis/graphql/token"
	"github.com/botobag/artemis/internal/testutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
)

func lexOne(str string) (*token.Token, error) {
	return lexer.New(token.NewSource(str)).Advance()
}

func lexSecond(str string) (*token.Token, error) {
	lexer := lexer.New(token.NewSource(str))
	_, err := lexer.Advance()
	Expect(err).ShouldNot(HaveOccurred())
	return lexer.Advance()
}

func expectSyntaxError(text string, message string, location graphql.ErrorLocation) {
	_, err := lexOne(text)
	Expect(err).Should(testutil.MatchGraphQLError(
		testutil.MessageContainSubstring(message),
		testutil.LocationEqual(location),
		testutil.KindIs(graphql.ErrKindSyntax),
	))
}

// A custom Gomega matcher to skip matching Prev and Next fields in the Token.
func MatchToken(token *token.Token) types.GomegaMatcher {
	return PointTo(MatchFields(IgnoreExtras, Fields{
		"Kind":     Equal(token.Kind),
		"Location": Equal(token.Location),
		"Length":   Equal(token.Length),
		"Value":    Equal(token.Value),
	}))
}

var _ = Describe("Lexer", func() {

	// graphql-js/src/language/__tests__/lexer-test.js
	It("disallows uncommon control characters", func() {
		expectSyntaxError(
			"\u0007",
			`Cannot contain the invalid character "\u0007".`,
			graphql.ErrorLocation{
				Line:   1,
				Column: 1,
			},
		)
	})

	It("accepts BOM header", func() {
		Expect(lexOne("\uFEFF foo")).Should(MatchToken(&token.Token{
			Kind:     token.KindName,
			Location: token.SourceLocation(5),
			Length:   3,
			Value:    "foo",
		}))
	})

	It("records line and column", func() {
		Expect(lexOne("\n \r\n \r  foo\n")).Should(MatchToken(&token.Token{
			Kind:     token.KindName,
			Location: token.SourceLocation(9),
			Length:   3,
			Value:    "foo",
		}))
	})

	It("skips whitespace and comments", func() {
		Expect(lexOne(`

    foo


`)).Should(MatchToken(&token.Token{
			Kind:     token.KindName,
			Location: token.SourceLocation(7),
			Length:   3,
			Value:    "foo",
		}))

		Expect(lexOne(`
    #comment
    foo#comment
`)).Should(MatchToken(&token.Token{
			Kind:     token.KindName,
			Location: token.SourceLocation(19),
			Length:   3,
			Value:    "foo",
		}))

		Expect(lexOne(",,,foo,,,")).Should(MatchToken(&token.Token{
			Kind:     token.KindName,
			Location: token.SourceLocation(4),
			Length:   3,
			Value:    "foo",
		}))
	})

	It("lexes strings", func() {
		Expect(lexOne(`"simple"`)).Should(MatchToken(&token.Token{
			Kind:     token.KindString,
			Location: token.SourceLocation(1),
			Length:   8,
			Value:    "simple",
		}))

		Expect(lexOne(`" white space "`)).Should(MatchToken(&token.Token{
			Kind:     token.KindString,
			Location: token.SourceLocation(1),
			Length:   15,
			Value:    " white space ",
		}))

		Expect(lexOne(`"quote \""`)).Should(MatchToken(&token.Token{
			Kind:     token.KindString,
			Location: token.SourceLocation(1),
			Length:   10,
			Value:    "quote \"",
		}))

		Expect(lexOne(`"escaped \n\r\b\t\f"`)).Should(MatchToken(&token.Token{
			Kind:     token.KindString,
			Location: token.SourceLocation(1),
			Length:   20,
			Value:    "escaped \n\r\b\t\f",
		}))

		Expect(lexOne(`"slashes \\ \/"`)).Should(MatchToken(&token.Token{
			Kind:     token.KindString,
			Location: token.SourceLocation(1),
			Length:   15,
			Value:    "slashes \\ /",
		}))

		Expect(lexOne(`"unicode \u1234\u5678\u90AB\uCDEF"`)).Should(MatchToken(&token.Token{
			Kind:     token.KindString,
			Location: token.SourceLocation(1),
			Length:   34,
			Value:    "unicode \u1234\u5678\u90AB\uCDEF",
		}))
	})

	It("lex reports useful string errors", func() {
		expectSyntaxError(`"`, "Unterminated string.", graphql.ErrorLocation{
			Line:   1,
			Column: 2,
		})

		expectSyntaxError(`"no end quote`, "Unterminated string.", graphql.ErrorLocation{
			Line:   1,
			Column: 14,
		})

		expectSyntaxError(
			"'single quotes'",
			`Unexpected single quote character ('), did you mean to use a double quote (")?`,
			graphql.ErrorLocation{
				Line:   1,
				Column: 1,
			},
		)

		expectSyntaxError(
			"\"contains unescaped \u0007 control char\"",
			`Invalid character within String: "\u0007".`,
			graphql.ErrorLocation{
				Line:   1,
				Column: 21,
			},
		)

		expectSyntaxError(
			"\"null-byte is not \u0000 end of file\"",
			`Invalid character within String: "\u0000".`,
			graphql.ErrorLocation{
				Line:   1,
				Column: 19,
			},
		)

		expectSyntaxError("\"multi\nLine\"", "Unterminated string.", graphql.ErrorLocation{
			Line:   1,
			Column: 7,
		})

		expectSyntaxError("\"multi\rLine\"", "Unterminated string.", graphql.ErrorLocation{
			Line:   1,
			Column: 7,
		})

		expectSyntaxError(
			`"bad \z esc"`,
			`Invalid character escape sequence: \z.`,
			graphql.ErrorLocation{
				Line:   1,
				Column: 7,
			},
		)

		expectSyntaxError(
			`"bad \x esc"`,
			`Invalid character escape sequence: \x.`,
			graphql.ErrorLocation{
				Line:   1,
				Column: 7,
			},
		)

		expectSyntaxError(
			`"bad \u1 esc"`,
			`Invalid character escape sequence: \u1 es.`,
			graphql.ErrorLocation{
				Line:   1,
				Column: 7,
			},
		)

		expectSyntaxError(
			`"bad \u0XX1 esc"`,
			`Invalid character escape sequence: \u0XX1.`,
			graphql.ErrorLocation{
				Line:   1,
				Column: 7,
			},
		)

		expectSyntaxError(
			`"bad \uXXXX esc"`,
			`Invalid character escape sequence: \uXXXX.`,
			graphql.ErrorLocation{
				Line:   1,
				Column: 7,
			},
		)

		expectSyntaxError(
			`"bad \uFXXX esc"`,
			`Invalid character escape sequence: \uFXXX.`,
			graphql.ErrorLocation{
				Line:   1,
				Column: 7,
			},
		)

		expectSyntaxError(
			`"bad \uXXXF esc"`,
			`Invalid character escape sequence: \uXXXF.`,
			graphql.ErrorLocation{
				Line:   1,
				Column: 7,
			},
		)
	})

	It("lexes block strings", func() {
		Expect(lexOne(`"""simple"""`)).Should(MatchToken(&token.Token{
			Kind:     token.KindBlockString,
			Location: token.SourceLocation(1),
			Length:   12,
			Value:    "simple",
		}))

		Expect(lexOne(`""" white space """`)).Should(MatchToken(&token.Token{
			Kind:     token.KindBlockString,
			Location: token.SourceLocation(1),
			Length:   19,
			Value:    " white space ",
		}))

		Expect(lexOne(`"""contains " quote"""`)).Should(MatchToken(&token.Token{
			Kind:     token.KindBlockString,
			Location: token.SourceLocation(1),
			Length:   22,
			Value:    `contains " quote`,
		}))

		Expect(lexOne(`"""contains \""" triplequote"""`)).Should(MatchToken(&token.Token{
			Kind:     token.KindBlockString,
			Location: token.SourceLocation(1),
			Length:   31,
			Value:    `contains """ triplequote`,
		}))

		Expect(lexOne("\"\"\"multi\nline\"\"\"")).Should(MatchToken(&token.Token{
			Kind:     token.KindBlockString,
			Location: token.SourceLocation(1),
			Length:   16,
			Value:    "multi\nline",
		}))

		Expect(lexOne("\"\"\"multi\rline\r\nnormalized\"\"\"")).Should(MatchToken(&token.Token{
			Kind:     token.KindBlockString,
			Location: token.SourceLocation(1),
			Length:   28,
			Value:    "multi\nline\nnormalized",
		}))

		Expect(lexOne(`"""unescaped \n\r\b\t\f\u1234"""`)).Should(MatchToken(&token.Token{
			Kind:     token.KindBlockString,
			Location: token.SourceLocation(1),
			Length:   32,
			Value:    `unescaped \n\r\b\t\f\u1234`,
		}))

		Expect(lexOne(`"""slashes \\ \/"""`)).Should(MatchToken(&token.Token{
			Kind:     token.KindBlockString,
			Location: token.SourceLocation(1),
			Length:   19,
			Value:    "slashes \\\\ \\/",
		}))

		Expect(lexOne(`"""

        spans
          multiple
            lines

        """`)).Should(MatchToken(&token.Token{
			Kind:     token.KindBlockString,
			Location: token.SourceLocation(1),
			Length:   68,
			Value:    "spans\n  multiple\n    lines",
		}))
	})

	It("advance line after lexing multiline block string", func() {
		Expect(lexSecond(`"""

        spans
          multiple
            lines

        ` + "\n \"\"\" second_token")).Should(MatchToken(&token.Token{
			Kind:     token.KindName,
			Location: token.SourceLocation(72),
			Length:   12,
			Value:    "second_token",
		}))

		Expect(lexSecond(strings.Join([]string{
			"\"\"\" \n",
			"spans \r\n",
			"multiple \n\r",
			"lines \n\n",
			"\"\"\"\n second_token",
		}, ""))).Should(MatchToken(&token.Token{
			Kind:     token.KindName,
			Location: token.SourceLocation(38),
			Length:   12,
			Value:    "second_token",
		}))
	})

	It("lex reports useful block string errors", func() {
		expectSyntaxError(`"""`, "Unterminated string.", graphql.ErrorLocation{
			Line:   1,
			Column: 4,
		})

		expectSyntaxError(`"""no end quote`, "Unterminated string.", graphql.ErrorLocation{
			Line:   1,
			Column: 16,
		})

		expectSyntaxError(
			"\"\"\"contains unescaped \u0007 control char\"\"\"",
			`Invalid character within String: "\u0007".`,
			graphql.ErrorLocation{
				Line:   1,
				Column: 23,
			},
		)

		expectSyntaxError(
			"\"\"\"null-byte is not \u0000 end of file\"\"\"",
			`Invalid character within String: "\u0000".`,
			graphql.ErrorLocation{
				Line:   1,
				Column: 21,
			},
		)
	})

	It("lexes numbers", func() {
		tests := []struct {
			text      string
			tokenKind token.Kind
		}{
			{"4", token.KindInt},
			{"4.123", token.KindFloat},
			{"-4", token.KindInt},
			{"9", token.KindInt},
			{"0", token.KindInt},
			{"-4.123", token.KindFloat},
			{"0.123", token.KindFloat},
			{"123e4", token.KindFloat},
			{"123E4", token.KindFloat},
			{"123e-4", token.KindFloat},
			{"123e+4", token.KindFloat},
			{"-1.123e4", token.KindFloat},
			{"-1.123E4", token.KindFloat},
			{"-1.123e-4", token.KindFloat},
			{"-1.123e+4", token.KindFloat},
			{"-1.123e4567", token.KindFloat},
		}

		for _, test := range tests {
			Expect(lexOne(test.text)).Should(MatchToken(&token.Token{
				Kind:     test.tokenKind,
				Location: token.SourceLocation(1),
				Length:   uint(len(test.text)),
				Value:    test.text,
			}))
		}
	})

	It("lex reports useful number errors", func() {
		tests := []struct {
			text    string
			message string
			line    uint
			column  uint
		}{
			{"00", `Invalid number, unexpected digit after 0: "0".`, 1, 2},
			{"+1", `Cannot parse the unexpected character "+".`, 1, 1},
			{"1.", "Invalid number, expected digit after decimal point ('.') but got: <EOF>.", 1, 3},
			{"1.e1", `Invalid number, expected digit after decimal point ('.') but got: "e".`, 1, 3},
			{".123", `Cannot parse the unexpected character ".".`, 1, 1},
			{"1.A", `Invalid number, expected digit after decimal point ('.') but got: "A".`, 1, 3},
			{"-A", `Invalid number, expected digit after '-' but got: "A".`, 1, 2},
			{"1.0e", `Invalid number, expected digit but got: <EOF>.`, 1, 5},
			{"1.0eA", `Invalid number, expected digit but got: "A".`, 1, 5},
		}
		for _, test := range tests {
			expectSyntaxError(test.text, test.message, graphql.ErrorLocation{
				Line:   test.line,
				Column: test.column,
			})
		}
	})

	It("lexes punctuation", func() {
		tests := []struct {
			text      string
			tokenKind token.Kind
		}{
			{"!", token.KindBang},
			{"$", token.KindDollar},
			{"&", token.KindAmp},
			{"(", token.KindLeftParen},
			{")", token.KindRightParen},
			{"...", token.KindSpread},
			{":", token.KindColon},
			{"=", token.KindEquals},
			{"@", token.KindAt},
			{"[", token.KindLeftBracket},
			{"]", token.KindRightBracket},
			{"{", token.KindLeftBrace},
			{"|", token.KindPipe},
			{"}", token.KindRightBrace},
		}

		for _, test := range tests {
			Expect(lexOne(test.text)).Should(MatchToken(&token.Token{
				Kind:     test.tokenKind,
				Location: token.SourceLocation(1),
				Length:   uint(len(test.text)),
				Value:    "",
			}))
		}
	})

	It("lex reports useful unknown character error", func() {
		expectSyntaxError("..", `Cannot parse the unexpected character ".".`, graphql.ErrorLocation{
			Line:   1,
			Column: 1,
		})

		expectSyntaxError("?", `Cannot parse the unexpected character "?".`, graphql.ErrorLocation{
			Line:   1,
			Column: 1,
		})

		expectSyntaxError("\u203B", `Cannot parse the unexpected character "\u203B".`, graphql.ErrorLocation{
			Line:   1,
			Column: 1,
		})

		expectSyntaxError("\u200b", `Cannot parse the unexpected character "\u200B".`, graphql.ErrorLocation{
			Line:   1,
			Column: 1,
		})
	})

	It("lex reports useful information for dashes in names", func() {
		lexer := lexer.New(token.NewSource("a-b"))

		Expect(lexer.Advance()).Should(MatchToken(&token.Token{
			Kind:     token.KindName,
			Location: token.SourceLocation(1),
			Length:   1,
			Value:    "a",
		}))

		_, err := lexer.Advance()
		e, ok := err.(*graphql.Error)
		Expect(ok).Should(BeTrue())
		Expect(e.Message).Should(Equal(`Syntax Error: Invalid number, expected digit after '-' but got: "b".`))
		Expect(e.Locations).Should(Equal([]graphql.ErrorLocation{
			{Line: 1, Column: 3},
		}))
	})

	It("produces double linked list of tokens, including comments", func() {
		lexer := lexer.New(token.NewSource(`{
      #comment
      field
    }`))

		var (
			endToken *token.Token
			err      error
		)

		startToken := lexer.Token()
		for {
			endToken, err = lexer.Advance()
			Expect(err).ShouldNot(HaveOccurred())
			if endToken.Kind == token.KindEOF {
				break
			}
			Expect(endToken.Kind).ShouldNot(Equal(token.KindComment))
		}

		Expect(startToken.Prev).Should(BeNil())
		Expect(endToken.Next).Should(BeNil())

		var tokens []*token.Token
		for token := startToken; token != nil; token = token.Next {
			if len(tokens) > 0 {
				// Tokens are double-linked, prev should point to last seen token.
				Expect(token.Prev).Should(Equal(tokens[len(tokens)-1]))
			}
			tokens = append(tokens, token)
		}

		expectedTokens := []struct {
			Kind        token.Kind
			Description string
		}{
			{token.KindSOF, "<SOF>"},
			{token.KindLeftBrace, "{"},
			{token.KindComment, "Comment"},
			{token.KindName, `Name "field"`},
			{token.KindRightBrace, "}"},
			{token.KindEOF, "<EOF>"},
		}
		Expect(len(tokens)).Should(Equal(len(expectedTokens)))
		for i, expectedToken := range expectedTokens {
			token := tokens[i]
			Expect(token.Kind).Should(Equal(expectedToken.Kind))
			Expect(token.Description()).Should(Equal(expectedToken.Description))
		}
	})

	It("accepts empty string", func() {
		Expect(lexOne(`""`)).Should(MatchToken(&token.Token{
			Kind:     token.KindString,
			Location: token.SourceLocation(1),
			Length:   2,
			Value:    "",
		}))
	})

	It("accepts incomplete triple-quotes as normal bytes in block string", func() {
		Expect(lexOne(`"""one quote: " """`)).Should(MatchToken(&token.Token{
			Kind:     token.KindBlockString,
			Location: token.SourceLocation(1),
			Length:   19,
			Value:    `one quote: " `,
		}))

		Expect(lexOne(`"""two quote: "" """`)).Should(MatchToken(&token.Token{
			Kind:     token.KindBlockString,
			Location: token.SourceLocation(1),
			Length:   20,
			Value:    `two quote: "" `,
		}))

		Expect(lexOne(`"""one quote: \" """`)).Should(MatchToken(&token.Token{
			Kind:     token.KindBlockString,
			Location: token.SourceLocation(1),
			Length:   20,
			Value:    `one quote: \" `,
		}))

		Expect(lexOne(`"""two quote: \"" """`)).Should(MatchToken(&token.Token{
			Kind:     token.KindBlockString,
			Location: token.SourceLocation(1),
			Length:   21,
			Value:    `two quote: \"" `,
		}))
	})

	It("rejects incomplete escape unicode sequence at the end", func() {
		expectSyntaxError(
			`"\u"`,
			`Invalid character escape sequence: \u`,
			graphql.ErrorLocation{
				Line:   1,
				Column: 3,
			},
		)

		expectSyntaxError(
			`"\u0"`,
			`Invalid character escape sequence: \u0`,
			graphql.ErrorLocation{
				Line:   1,
				Column: 3,
			},
		)
		expectSyntaxError(
			`"\u00"`,
			`Invalid character escape sequence: \u00`,
			graphql.ErrorLocation{
				Line:   1,
				Column: 3,
			},
		)
		expectSyntaxError(
			`"\u000"`,
			`Invalid character escape sequence: \u000`,
			graphql.ErrorLocation{
				Line:   1,
				Column: 3,
			},
		)
	})

	It("accepts whitespace characters at the end", func() {
		Expect(lexOne(`simple



`)).Should(MatchToken(&token.Token{
			Kind:     token.KindName,
			Location: token.SourceLocation(1),
			Length:   6,
			Value:    "simple",
		}))
	})

	It("lexes tokens with matching source", func() {
		source := token.NewSource(`{
      field
      123
      123.45
    }`, token.SourceName("Source Test"))
		lexer := lexer.New(source)

		tokens := []*token.Token{
			lexer.Token(),
		}
		for {
			tok, err := lexer.Advance()
			Expect(err).ShouldNot(HaveOccurred())
			tokens = append(tokens, tok)
			if tok.Kind == token.KindEOF {
				break
			}
		}

		expectedLocationInfo := [][2]token.SourceLocationInfo{
			{{Name: "Source Test", Line: 0, Column: 0}, {Name: "Source Test", Line: 0, Column: 0}},
			{{Name: "Source Test", Line: 1, Column: 1}, {Name: "Source Test", Line: 1, Column: 2}},
			{{Name: "Source Test", Line: 2, Column: 7}, {Name: "Source Test", Line: 2, Column: 12}},
			{{Name: "Source Test", Line: 3, Column: 7}, {Name: "Source Test", Line: 3, Column: 10}},
			{{Name: "Source Test", Line: 4, Column: 7}, {Name: "Source Test", Line: 4, Column: 13}},
			{{Name: "Source Test", Line: 5, Column: 5}, {Name: "Source Test", Line: 5, Column: 6}},
			{{Name: "Source Test", Line: 5, Column: 6}, {Name: "Source Test", Line: 5, Column: 6}},
		}

		Expect(len(tokens)).Should(Equal(len(expectedLocationInfo)))
		for i, tok := range tokens {
			Expect(tok.Source()).Should(Equal(source))
			Expect(tok.LocationInfo()).Should(Equal(expectedLocationInfo[i][0]))
			Expect(tok.EndLocationInfo()).Should(Equal(expectedLocationInfo[i][1]))
		}
	})
})
