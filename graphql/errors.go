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
	source      *Source
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
func NewSyntaxError(source *Source, location token.SourceLocation, description string) error {
	e := &syntaxError{
		source:      source,
		location:    location,
		description: description,
	}
	return NewError(e.Error(), e)
}
