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

package graphql

import (
	"fmt"

	"github.com/botobag/artemis/graphql/token"
)

//===----------------------------------------------------------------------------------------====//
// Syntax Error
//===----------------------------------------------------------------------------------------====//

type syntaxError struct {
	source      *token.Source
	location    token.SourceLocation
	description string
}

var (
	_ error              = (*syntaxError)(nil)
	_ ErrorWithLocations = (*syntaxError)(nil)
)

// Error implements Go's error interface.
func (e *syntaxError) Error() string {
	return fmt.Sprintf("Syntax Error: %s", e.description)
}

// Locations implements ErrorWithLocations.
func (e *syntaxError) Locations() []ErrorLocation {
	locInfo := e.source.LocationInfoOf(e.location)
	return []ErrorLocation{
		{
			Line:   locInfo.Line,
			Column: locInfo.Column,
		},
	}
}

// NewSyntaxError produces an error representing a syntax error, containing useful descriptive
// information about the syntax error's position in the source.
func NewSyntaxError(source *token.Source, location token.SourceLocation, description string) error {
	e := &syntaxError{
		source:      source,
		location:    location,
		description: description,
	}
	return NewError(e.Error(), e, ErrKindSyntax)
}

//===----------------------------------------------------------------------------------------====//
// Coercion Error
//===----------------------------------------------------------------------------------------====//

// NewCoercionError raises an error to indicate an error due to coercion failure (for either input
// coercion or result coercion defined in [0]). The error is tagged with ErrKindCoercion and several
// functions will take special care of it which described as follows:
//
// 1. For coercion errors returning from Enum and Scalar's CoerceResultValue:
//  - When execution engine sees these error (in `CompleteValue` [1], and look our implementation of
//    `executor.Common.completeLeafValue`, more specifically), it will bypass the errors directly to
//    the caller, resulting a field/query error containing the message as the one carried by the
//    errors. Otherwise, execution engine will wrap the error with NewDefaultResultCoercionError.
//
// 2. For coercion errors returning from Enum and Scalar's CoerceVariableValue:
//  - When CoerceValue sees these errors, it will present a query error with the message specified
//    in the error to the user.
//
// 3. For coercion errors returning from Enum and Scalar's CoerceLiteralValue:
//  - Currently it makes no difference than other errors.
//
// [0]: https://facebook.github.io/graphql/June2018/#sec-Scalars
// [1]: https://facebook.github.io/graphql/June2018/#CompleteValue()
func NewCoercionError(format string, a ...interface{}) error {
	return NewError(fmt.Sprintf(format, a...), ErrKindCoercion)
}

// NewDefaultResultCoercionError creates a CoercionError for result coercion with a default message.
func NewDefaultResultCoercionError(typeName string, value interface{}, err error) error {
	message := fmt.Sprintf(`Expected a value of type "%s" but received: %v`, typeName, value)
	return NewError(message, err, ErrKindCoercion)
}
