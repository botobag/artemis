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

package token

import (
	"fmt"
)

// Kind describes the different kinds of tokens that the lexer emits.
type Kind int

// Enumeration of Kind
//
// Reference: https://facebook.github.io/graphql/June2018/#sec-Appendix-Grammar-Summary.Lexical-Tokens.
const (
	// <SOF>
	KindSOF Kind = iota + 1
	// <EOF>
	KindEOF
	// !
	KindBang
	// $
	KindDollar
	// &
	KindAmp
	// (
	KindLeftParen
	// )
	KindRightParen
	// ...
	KindSpread
	// :
	KindColon
	// =
	KindEquals
	// @
	KindAt
	// [
	KindLeftBracket
	// ]
	KindRightBracket
	// {
	KindLeftBrace
	// |
	KindPipe
	// }
	KindRightBrace
	// Ref: https://facebook.github.io/graphql/June2018/#Name
	KindName
	// Ref: https://facebook.github.io/graphql/June2018/#IntValue
	KindInt
	// Ref: https://facebook.github.io/graphql/June2018/#sec-Float-Value
	KindFloat
	// Ref: https://facebook.github.io/graphql/June2018/#sec-String-Value
	KindString
	// Ref: https://facebook.github.io/graphql/June2018/#sec-String-Value
	KindBlockString
	// Ref: https://facebook.github.io/graphql/June2018/#sec-Comments
	KindComment
)

var _ fmt.Stringer = Kind(0)

func (kind Kind) String() string {
	switch kind {
	case KindSOF:
		return "<SOF>"
	case KindEOF:
		return "<EOF>"
	case KindBang:
		return "!"
	case KindDollar:
		return "$"
	case KindAmp:
		return "&"
	case KindLeftParen:
		return "("
	case KindRightParen:
		return ")"
	case KindSpread:
		return "..."
	case KindColon:
		return ":"
	case KindEquals:
		return "="
	case KindAt:
		return "@"
	case KindLeftBracket:
		return "["
	case KindRightBracket:
		return "]"
	case KindLeftBrace:
		return "{"
	case KindPipe:
		return "|"
	case KindRightBrace:
		return "}"
	case KindName:
		return "Name"
	case KindInt:
		return "Int"
	case KindFloat:
		return "Float"
	case KindString:
		return "String"
	case KindBlockString:
		return "BlockString"
	case KindComment:
		return "Comment"
	}
	panic("unsupported token kind")
}

// SourceLocation encodes a position in source file. It lives in the context of Source. Its value is
// an 1-indexed offset relative to the beginning of source measured in bytes. Given a SourceLocation
// value loc and the Source s, you can convert it into larger representation SourceLocationInfo by
// calling s.LocationInfoOf(loc).
type SourceLocation uint

// NoSourceLocation is a special SourceLocation that doesn't exists in any source. Method that deals
// with SourceLocation must take special care to handle this value.
const NoSourceLocation SourceLocation = 0

// IsValid return true if the SourceLocation is valid.
func (location SourceLocation) IsValid() bool {
	return location != NoSourceLocation
}

// Token represents a range of characters represented by a lexical token within a Source.
type Token struct {
	// The kind of Token.
	Kind Kind

	// The position at which this Token begins in the source
	Location SourceLocation

	// The length of the token in the source
	Length uint

	// For punctuation and comment tokens, this is empty. For other kinds of
	// token, this represents the interpreted value of the token.
	Value string

	// Tokens exist as nodes in a double-linked-list amongst all tokens including ignored tokens.
	// <SOF> is always the first node and <EOF> the last.
	Prev *Token
	Next *Token
}

// Description describe a token as a string for debugging.
func (token *Token) Description() string {
	if len(token.Value) > 0 {
		return fmt.Sprintf(`%s "%s"`, token.Kind.String(), token.Value)
	}
	return token.Kind.String()
}
