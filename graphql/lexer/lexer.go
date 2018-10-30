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

package lexer

import (
	"bytes"
	"fmt"

	"github.com/botobag/artemis/graphql"
	lexerinternal "github.com/botobag/artemis/graphql/internal/lexer"
	"github.com/botobag/artemis/graphql/token"
)

// Lexer is the return type of newLexer.
type Lexer struct {
	source *graphql.Source

	// The previously focused non-ignored token
	lastToken *token.Token

	// The currently focused non-ignored token
	token *token.Token

	// Current offest into the source body; Moved by only consume() and consumeWhitespace().
	bytePos uint

	// This caches the value of source.Body().Size().
	bodySize uint
}

// New initializes a Lexer for given Source object. A Lexer is a stateful stream generator in that
// every time it is advanced, it returns the next token in the Source. Assuming the source lexes,
// the final Token emitted by the lexer will be of kind EOF, after which the lexer will repeatedly
// return the same EOF token whenever called.
func New(source *graphql.Source) *Lexer {
	startOfFileToken := &token.Token{
		Kind: token.KindSOF,
	}
	return &Lexer{
		source:    source,
		lastToken: startOfFileToken,
		token:     startOfFileToken,
		bytePos:   0,
		bodySize:  source.Body().Size(),
	}
}

// Source returns the source being lexed.
func (lexer *Lexer) Source() *graphql.Source {
	return lexer.source
}

// Token returns current token being lexed.
func (lexer *Lexer) Token() *token.Token {
	return lexer.token
}

// Advance the token stream to the next non-ignored token.
func (lexer *Lexer) Advance() (*token.Token, error) {
	nextToken, err := lexer.Lookahead()
	if err != nil {
		return nil, err
	}
	lexer.lastToken, lexer.token = lexer.token, nextToken
	return nextToken, nil
}

// Lookahead looks ahead and returns the next non-ignored token, but does not switch current token.
func (lexer *Lexer) Lookahead() (*token.Token, error) {
	tok := lexer.token
	if tok.Kind != token.KindEOF {
		for {
			// Read next token and save to token.net if we haven't done yet.
			if tok.Next == nil {
				nextToken, err := lexer.lexToken()
				if err != nil {
					return nil, err
				}
				tok.Next = nextToken
			}
			tok = tok.Next

			if tok.Kind != token.KindComment {
				break
			}
			// Continue lexing the next token to skip comments. Update prev link in the next token is
			// correctly linked to the comment token.
			lexer.token = tok
		}
	}
	return tok, nil
}

// Location returns SourceLocation for the current position in the source.
func (lexer *Lexer) Location() token.SourceLocation {
	return lexer.LocationWithPos(lexer.bytePos)
}

// LocationWithPos returns SourceLocation for the specified position in the source.
func (lexer *Lexer) LocationWithPos(bytePos uint) token.SourceLocation {
	return lexer.source.LocationFromPos(bytePos)
}

// peek peeks the next byte at bytePos without consume it.
func (lexer *Lexer) peek() byte {
	return lexer.source.Body().At(lexer.bytePos)
}

// consume reads a byte at current bytePos and then advances the bytePos. Return the byte.
func (lexer *Lexer) consume() byte {
	b := lexer.source.Body().At(lexer.bytePos)
	if lexer.bytePos < lexer.bodySize {
		lexer.bytePos++
	}
	return b
}

// consumeWhitespace consumes bytes from body starting at current bytePos until it finds a
// non-whitespace character.
func (lexer *Lexer) consumeWhitespace() {
	body := lexer.source.Body()
	bodySize := lexer.bodySize

	// Cache bytePos locally. Will update back before return.
	bytePos := lexer.bytePos

	// Handle BOM at the beginning of source specially.
	if bytePos == 0 && (bodySize-bytePos) >= 3 {
		if body[bytePos] == '\xEF' &&
			body[bytePos+1] == '\xBB' &&
			body[bytePos+2] == '\xBF' {
			// Skip BOM.
			bytePos += 3
		}
	}

	// Whitespaces are all ASCII characters (BOM is handled specially above).
	for bytePos < bodySize {
		switch body[bytePos] {
		case '\t', ' ', ',', '\n':
			bytePos++

		case '\r':
			if (bodySize-bytePos) >= 2 && body[bytePos+1] == '\n' {
				bytePos++
			}
			bytePos++

		default:
			lexer.bytePos = bytePos
			return
		}
	}

	// If here, there're whitespace characters before EOF. Remember to update local bytePos to lexer
	// before return.
	lexer.bytePos = bytePos
}

// consumeDigits consumes bytes that represent a digit (i.e., from "0" to "9"). This is used by
// lexNumber as helper function. Return the rune that contains the first non-digits.
func (lexer *Lexer) consumeDigits() byte {
	for {
		char := lexer.peek()
		if char >= '0' && char <= '9' {
			lexer.consume()
		} else {
			return char
		}
	}
}

func (lexer *Lexer) charAtPosToStr(bytePos uint) string {
	if bytePos >= lexer.bodySize {
		return "<EOF>"
	}

	// Try to decode a rune at bytePos.
	r, _ := lexer.source.Body().RuneAt(bytePos)

	// Print as ASCII for printable range.
	if r >= 0x20 && r < 0x7F {
		return fmt.Sprintf(`"%c"`, r)
	}

	// Print the escaped form. e.g. `"\\u0007"`
	return fmt.Sprintf(`"\u%04X"`, r)
}

// newUnexpectedCharacter creates a syntax error to indicate an unexpected character at the given
// offset was encountered.
func (lexer *Lexer) newUnexpectedCharacterError(bytePos uint) error {
	var message string

	char := lexer.source.Body().At(bytePos)
	if (char < 0x0020) && (char != 0x0009) && (char != 0x000a) && (char != 0x000d) {
		message = fmt.Sprintf("Cannot contain the invalid character %s.", lexer.charAtPosToStr(bytePos))
	} else if char == '\'' {
		message = "Unexpected single quote character ('), did you mean to use a double quote (\")?"
	} else {
		message = fmt.Sprintf("Cannot parse the unexpected character %s.", lexer.charAtPosToStr(bytePos))
	}

	return graphql.NewSyntaxError(lexer.source, lexer.LocationWithPos(bytePos), message)
}

func (lexer *Lexer) makeToken(kind token.Kind, length uint) *token.Token {
	return lexer.makeTokenWithValue(kind, length, "")
}

func (lexer *Lexer) makeTokenWithValue(kind token.Kind, length uint, value string) *token.Token {
	return &token.Token{
		Kind:     kind,
		Location: lexer.LocationWithPos(lexer.bytePos - length),
		Length:   length,
		Value:    value,
		// We're returning the next token. So the current token is the previous token.
		Prev: lexer.token,
	}
}

// lexToken gets the next token from the source starting at the lexet.bytePos. This skips over
// whitespaces until it finds the next lexable token, then lexes punctuators immediately or calls
// the appropriate helper function for more complicated tokens.
func (lexer *Lexer) lexToken() (*token.Token, error) {
	// We're reading the next token. So the current token is the previous token.
	prev := lexer.token

	// Consume whitespace characters.
	lexer.consumeWhitespace()

	// Peek a byte.
	char := lexer.peek()
	if char == 0 && (lexer.bytePos >= lexer.bodySize) {
		return &token.Token{
			Kind:     token.KindEOF,
			Location: lexer.Location(),
			Prev:     prev,
		}, nil
	}

	// lexSimpleToken lexes a byte and produces a token of the given type with location information.
	lexSimpleToken := func(kind token.Kind) (*token.Token, error) {
		// Consume the byte.
		lexer.consume()

		// Make the token and return.
		return lexer.makeToken(kind, 1), nil
	}

	switch char {
	case '!':
		return lexSimpleToken(token.KindBang)
	case '#':
		return lexer.lexComment(), nil
	case '$':
		return lexSimpleToken(token.KindDollar)
	case '&':
		return lexSimpleToken(token.KindAmp)
	case '(':
		return lexSimpleToken(token.KindLeftParen)
	case ')':
		return lexSimpleToken(token.KindRightParen)
	case '.':
		// Consume the dot.
		lexer.consume()
		if lexer.peek() != '.' {
			return nil, lexer.newUnexpectedCharacterError(lexer.bytePos - 1)
		}

		// Consume the dot again.
		lexer.consume()
		if lexer.peek() != '.' {
			return nil, lexer.newUnexpectedCharacterError(lexer.bytePos - 2)
		}

		// Consume the last dot.
		lexer.consume()
		// Make the token for return.
		return lexer.makeToken(token.KindSpread, 3), nil
	case ':':
		return lexSimpleToken(token.KindColon)
	case '=':
		return lexSimpleToken(token.KindEquals)
	case '@':
		return lexSimpleToken(token.KindAt)
	case '[':
		return lexSimpleToken(token.KindLeftBracket)
	case ']':
		return lexSimpleToken(token.KindRightBracket)
	case '{':
		return lexSimpleToken(token.KindLeftBrace)
	case '|':
		return lexSimpleToken(token.KindPipe)
	case '}':
		return lexSimpleToken(token.KindRightBrace)

		// A-Z _ a-z
	case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N',
		'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z',
		'_', 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n',
		'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z':
		return lexer.lexName(), nil

	// - 0-9
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return lexer.lexNumber()

	case '"':
		// Consume the quote.
		lexer.consume()
		// Peek the next character.
		if r := lexer.peek(); r == '"' {
			// Consume the second quote.
			lexer.consume()

			// See whether we have the third one.
			if lexer.peek() == '"' {
				lexer.consume()
				return lexer.lexBlockString()
			}

			// We alread consumed 2 double quotes but failed to get the 3rd one. Return an empty string
			// value.
			return lexer.makeTokenWithValue(token.KindString, 2, ""), nil
		}
		return lexer.lexString()
	}

	return nil, lexer.newUnexpectedCharacterError(lexer.bytePos)
}

// lexComment reads a comment token from the source file.
//
//	Comment ::
//		# CommentCharlistopt
//
//	CommentChar ::
//		SourceCharacter but not LineTerminator
//
//	SourceCharacter ::
//		/[\u0009\u000A\u000D\u0020-\uFFFF]/
//
//	LineTerminator ::
//		New Line (U+000A)
//		Carriage Return (U+000D) [lookhead != New Line (U+000A)]
//		Carriage Return (U+000D) New Line (U+000A)
//
// Reference: https://facebook.github.io/graphql/June2018/#sec-Comments
func (lexer *Lexer) lexComment() *token.Token {
	// Remember where the token begins.
	startPos := lexer.bytePos

	// Consume #.
	lexer.consume()
	for {
		char := lexer.peek()
		// SourceCharacter but not LineTerminator
		if char > 0x1F || char == '\t' {
			lexer.consume()
			continue
		}
		break
	}

	return lexer.makeToken(token.KindComment, lexer.bytePos-startPos)
}

// lexNumber reads a number token from the source file, either a float [0] or an int [1] depending
// on whether a decimal point appears.
//
// [0]: https://facebook.github.io/graphql/June2018/#sec-Float-Value
// [1]: https://facebook.github.io/graphql/June2018/#sec-Int-Value
func (lexer *Lexer) lexNumber() (*token.Token, error) {
	// Remember where the token begins.
	startPos := lexer.bytePos

	// Consume one character that have been read in lexToken.
	char := lexer.consume()
	tokenKind := token.KindInt

	if char == '-' {
		char = lexer.peek()
		if char < '0' || char > '9' {
			return nil, graphql.NewSyntaxError(
				lexer.source,
				lexer.Location(),
				fmt.Sprintf("Invalid number, expected digit after '-' but got: %s.",
					lexer.charAtPosToStr(lexer.bytePos)))
		}
		lexer.consume()
	}

	if char == '0' {
		char = lexer.peek()
		if char >= '0' && char <= '9' {
			return nil, graphql.NewSyntaxError(
				lexer.source,
				lexer.Location(),
				fmt.Sprintf("Invalid number, unexpected digit after 0: %s.",
					lexer.charAtPosToStr(lexer.bytePos)))
		}
	} else {
		// char must be "1" .. "9". Consume all digits.
		char = lexer.consumeDigits()
	}

	if char == '.' {
		tokenKind = token.KindFloat

		// Consume the decimal point.
		lexer.consume()

		// Expect at least one digits.
		char = lexer.peek()
		if char >= '0' && char <= '9' {
			// Consume the first digits.
			lexer.consume()
			// Consume all subsequent digits.
			char = lexer.consumeDigits()
		} else {
			return nil, graphql.NewSyntaxError(
				lexer.source,
				lexer.Location(),
				fmt.Sprintf("Invalid number, expected digit after decimal point ('.') but got: %s.",
					lexer.charAtPosToStr(lexer.bytePos)))
		}
	}

	if char == 'E' || char == 'e' {
		// Consume "E" or "e".
		lexer.consume()
		tokenKind = token.KindFloat

		char = lexer.peek()
		if char == '+' || char == '-' {
			lexer.consume()
		}

		// Expect at least one digits.
		char = lexer.peek()
		if char >= '0' && char <= '9' {
			// Consume the first digits.
			lexer.consume()
			// Consume all subsequent digits.
			lexer.consumeDigits()
		} else {
			return nil, graphql.NewSyntaxError(
				lexer.source,
				lexer.Location(),
				fmt.Sprintf("Invalid number, expected digit but got: %s.",
					lexer.charAtPosToStr(lexer.bytePos)))
		}
	}

	return lexer.makeTokenWithValue(
		tokenKind,
		lexer.bytePos-startPos,
		string(lexer.source.Body()[startPos:lexer.bytePos])), nil
}

// lexString reads a string token from the source file.
//
//	StringValue ::
//		" StringCharacter "
//		""" BlockStringCharacter """
//
//	StringCharacter ::
//		SourceCharacter but not " or \ or LineTerminator
//		\u EscapedUnicode
//		\ EscapedCharacter
//
//	EscapedUnicode ::
//		/[0-9A-Fa-f]{4}/
//
//	EscapedCharacter :: one of
//		"	\	/	b	f	n	r	t
//
//	BlockStringCharacter
//		SourceCharacter but not """ or \"""
//		\"""
//
// Reference: https://facebook.github.io/graphql/June2018/#sec-String-Value
func (lexer *Lexer) lexString() (*token.Token, error) {
	// Note that the " was already consumed in lexToken.
	startPos := lexer.bytePos - 1

	var value bytes.Buffer
	// Loop until EOF.
	for lexer.bytePos < lexer.bodySize {
		char := lexer.peek()

		// Exit when encounter a LineTerminator.
		if char == '\n' || char == '\r' {
			break
		}

		if char == '"' {
			// Consume the closing quote (").
			lexer.consume()
			// Return a string token.
			return lexer.makeTokenWithValue(token.KindString, lexer.bytePos-startPos, value.String()), nil
		}

		// Make sure the character is a valid SourceCharacter.
		if char < 0x0020 && char != '\t' {
			return nil, graphql.NewSyntaxError(
				lexer.source,
				lexer.Location(),
				fmt.Sprintf("Invalid character within String: %s.",
					lexer.charAtPosToStr(lexer.bytePos)))
		}

		// Consume the character.
		lexer.consume()

		// Handle escape sequence.
		if char != '\\' {
			// Early exit for non-escape sequence.
			value.WriteByte(char)
			continue
		}

		// Peek next character.
		char = lexer.consume()
		switch char {
		// EscapedCharacter
		case '"':
			value.WriteRune('"')
		case '\\':
			value.WriteRune('\\')
		case '/':
			value.WriteRune('/')
		case 'b':
			value.WriteRune('\b')
		case 'f':
			value.WriteRune('\f')
		case 'n':
			value.WriteRune('\n')
		case 'r':
			value.WriteRune('\r')
		case 't':
			value.WriteRune('\t')

		// EscapedUnicode
		case 'u':
			var (
				escapeSeqPos = lexer.bytePos
				escapeSeqEnd uint
			)
			if lexer.bodySize-lexer.bytePos < 4 {
				escapeSeqEnd = lexer.bodySize
			} else {
				escapeSeqEnd = lexer.bytePos + 4
				charCode := uniCharCode(
					lexer.consume(),
					lexer.consume(),
					lexer.consume(),
					lexer.consume(),
				)
				if charCode >= 0 {
					value.WriteRune(charCode)
					break
				}
			}

			return nil, graphql.NewSyntaxError(
				lexer.source,
				lexer.LocationWithPos(escapeSeqPos-1),
				fmt.Sprintf("Invalid character escape sequence: \\u%s.",
					string(lexer.source.Body()[escapeSeqPos:escapeSeqEnd])),
			)

		default:
			return nil, graphql.NewSyntaxError(
				lexer.source,
				lexer.LocationWithPos(lexer.bytePos-1),
				fmt.Sprintf("Invalid character escape sequence: \\%c.", char))
		}
	}

	return nil, graphql.NewSyntaxError(lexer.source, lexer.Location(), "Unterminated string.")
}

// Converts four hexadecimal chars to the integer that the string represents. For example,
// uniCharCode('0','0','0','f') will return 15, and uniCharCode('0','0','f','f') returns 255.
//
// Returns a negative number on error, if a char was invalid.
//
// This is implemented by noting that char2hex() returns -1 on error, which means the result of
// ORing the char2hex() will also be negative.
func uniCharCode(a byte, b byte, c byte, d byte) rune {
	return (char2hex(a) << 12) | (char2hex(b) << 8) | (char2hex(c) << 4) | char2hex(d)
}

// Converts a hex character to its integer value.
//
// '0' becomes 0, '9' becomes 9
// 'A' becomes 10, 'F' becomes 15
// 'a' becomes 10, 'f' becomes 15
//
// Returns -1 on error.
func char2hex(a byte) rune {
	if a >= '0' && a <= '9' { // 0-9
		return rune(a - '0')
	} else if a >= 'A' && a <= 'F' { // A-F
		return rune(a - 55)
	} else if a >= 'a' && a <= 'f' {
		return rune(a - 87)
	}
	return -1
}

// lexBlockString reads a block string token from the source file.
func (lexer *Lexer) lexBlockString() (*token.Token, error) {
	// Note that the opening triple-quote (""") was already consumed in lexToken.
	startPos := lexer.bytePos - 3

	var value bytes.Buffer
	for lexer.bytePos < lexer.bodySize {
		char := lexer.peek()

		if char == '"' {
			// Consume 1st quote.
			lexer.consume()

			if char := lexer.peek(); char == '"' {
				// Consume the 2nd quote.
				lexer.consume()

				if char := lexer.peek(); char == '"' {
					// This is a closing triple-quote (""").
					lexer.consume()

					// Return a block string token.
					return lexer.makeTokenWithValue(
						token.KindBlockString,
						lexer.bytePos-startPos,
						lexerinternal.BlockStringValue(value.String())), nil
				}
				value.WriteRune('"')
			}
			value.WriteRune('"')
		} else if char == '\\' {
			// Check escape triple-quote (\"""). Consume backslash.
			lexer.consume()

			if char := lexer.peek(); char != '"' {
				// Write backslash.
				value.WriteRune('\\')
			} else {
				// Consume the 1st quote.
				lexer.consume()

				if char := lexer.peek(); char != '"' {
					// Write one backslash and one quote.
					value.WriteString("\\\"")
				} else {
					// Consume the 2nd quote.
					lexer.consume()

					if char := lexer.peek(); char != '"' {
						// Write one backslash and two quotes.
						value.WriteString("\\\"\"")
					} else {
						// Consume the 3rd quote. Got an escape triple-quote (\""").
						lexer.consume()
						value.WriteString("\"\"\"")
					}
				}
			}
		} else {
			// Make sure the character is a valid SourceCharacter.
			if char < 0x0020 && char != '\t' && char != '\r' && char != '\n' {
				return nil, graphql.NewSyntaxError(
					lexer.source,
					lexer.Location(),
					fmt.Sprintf("Invalid character within String: %s.",
						lexer.charAtPosToStr(lexer.bytePos)))
			}
			// Consume a valid character.
			lexer.consume()
			value.WriteByte(char)
		}
	}

	return nil, graphql.NewSyntaxError(lexer.source, lexer.Location(), "Unterminated string.")
}

// lexName lexes a Name token from source.
//
//	Name ::
//		/[_A-Za-z][_0-9A-Za-z]*/
//
// Reference: https://facebook.github.io/graphql/June2018/#sec-Names
func (lexer *Lexer) lexName() *token.Token {
	// Remember where the token begins.
	startPos := lexer.bytePos

	// Consume one rune which was read in lexToken before here.
	lexer.consume()

	for {
		char := lexer.peek()
		if char == '_' ||
			(char >= '0' && char <= '9') ||
			(char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') {
			lexer.consume()
			continue
		}
		break
	}

	return lexer.makeTokenWithValue(
		token.KindName,
		lexer.bytePos-startPos,
		string(lexer.source.Body()[startPos:lexer.bytePos]),
	)
}
