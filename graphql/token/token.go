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
	"unsafe"
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

// EndLocation returns the pass-the-end location of the token in the source.
func (token *Token) EndLocation() SourceLocation {
	return token.Location.WithOffset(int(token.Length))
}

// Range returns the range of this token in the source.
func (token *Token) Range() SourceRange {
	return SourceRange{
		Begin: token.Location,
		End:   token.EndLocation(),
	}
}

// Description describe a token as a string for debugging.
func (token *Token) Description() string {
	if len(token.Value) > 0 {
		return fmt.Sprintf(`%s "%s"`, token.Kind.String(), token.Value)
	}
	return token.Kind.String()
}

// Source finds the Source where this token is lexed from.
func (token *Token) Source() *Source {
	// Follow the link to get the SOF token.
	tok := token
	for tok.Prev != nil {
		tok = tok.Prev
	}

	// Assume tok is embedded in a sofToken object. Use unsafe.Pointer to calculate the address of its
	// adjacent Source reference.
	return (*sofToken)(unsafe.Pointer(
		uintptr(unsafe.Pointer(tok)) -
			unsafe.Offsetof((*sofToken)(nil).token))).source
}

// LocationInfo returns the line and column number at which a token begins in the source.
func (token *Token) LocationInfo() SourceLocationInfo {
	return token.Source().LocationInfoOf(token.Location)
}

// EndLocationInfo returns the line and column number at which a token ends in the source.
func (token *Token) EndLocationInfo() SourceLocationInfo {
	return token.Source().LocationInfoOf(token.EndLocation())
}

//===------------------------------------------------------------------------------------------===//
// "Magic" SOF Token
//===------------------------------------------------------------------------------------------===//

// This token is a *real hack*.
//
// The token is just an ordinary Start-of-File token (i.e., token.Kind is token.KindSOF and
// token.Prev is nil). What makes it special is that it is put along side with a Source. With the
// help of unsafe.Pointer [0] we can get the Source object from that Token. Then, based on fact that
// all tokens in a Source are put in a doubly linked list, by walking along the links to this SOF
// token, we can know the Source of a Token, and we can know its location (line and column number)
// by calling Source.LocationInfoOf. Furthermore, every AST node exposes an interface TokenRange().
// By applying the abovementioned techniques on the token returning from this interface, we can
// access the location information for a AST node without storing Source reference in every ast.Node
// and token.Token!
//
// [0]: https://golang.org/pkg/unsafe/#Pointer
type sofToken struct {
	token  Token
	source *Source
}

// NewSOFToken creates a special SOF token which enables tokens to trace back to their Source.
func NewSOFToken(source *Source) *Token {
	tok := sofToken{
		// Initialize an ordinary SOF token.
		token: Token{
			Kind: KindSOF,
		},
		source: source,
	}
	return &tok.token
}

//===------------------------------------------------------------------------------------------===//
// Range
//===------------------------------------------------------------------------------------------===//

// Range represent a range of tokens covered by [First, Last].
type Range struct {
	First *Token
	Last  *Token
}

// SourceRange indicates the source covered by this range with a pair of SourceLocation.
func (r Range) SourceRange() SourceRange {
	return SourceRange{
		Begin: r.First.Location,
		End:   r.Last.EndLocation(),
	}
}
