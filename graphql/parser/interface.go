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

package parser

import (
	"github.com/botobag/artemis/graphql/ast"
	"github.com/botobag/artemis/graphql/token"
)

// parseOptions configures parser behavior.
type parseOptions struct {
	// EXPERIMENTAL:
	//
	// If enabled, the parser will understand and parse variable definitions contained in a fragment
	// definition. They'll be represented in the `variableDefinitions` field of the
	// FragmentDefinition.
	//
	// The syntax is identical to normal, query-defined variables. For example:
	//
	//   fragment A($var: Boolean = false) on T  {
	//     ...
	//   }
	//
	// Note: this feature is experimental and may change or be removed in the future.
	//
	// See https://github.com/facebook/graphql/issues/204.
	ExperimentalFragmentVariables bool
}

// ParseOption // CORSOption represents a functional option for configuring the parser.
type ParseOption func(*parseOptions)

//
// Functional options for configuring Parser
//

// EnableFragmentVariables enables the (experimental) feature which accepts variables in fragment
// definition.
func EnableFragmentVariables() ParseOption {
	return func(options *parseOptions) {
		options.ExperimentalFragmentVariables = true
	}
}

// Parse parses the given GraphQL source into a Document.
func Parse(source *token.Source, options ...ParseOption) (ast.Document, error) {
	var opts parseOptions
	for _, applyOption := range options {
		applyOption(&opts)
	}

	parser, err := newParser(source, &opts)
	if err != nil {
		return ast.Document{}, err
	}
	return parser.parseDocument()
}

// MustParse parses the given GraphQL source into a Document and panics on errors.
func MustParse(source *token.Source, options ...ParseOption) ast.Document {
	doc, err := Parse(source, options...)
	if err != nil {
		panic(err)
	}
	return doc
}

// ParseValue parses an AST value from a string (e.g., `[42]`).
func ParseValue(source *token.Source) (ast.Value, error) {
	parser, err := newParser(source, &parseOptions{})
	if err != nil {
		return nil, err
	}

	if _, err := parser.expect(token.KindSOF); err != nil {
		return nil, err
	}

	value, err := parser.parseValue(false /*isConst */)
	if err != nil {
		return nil, err
	}

	if _, err := parser.expect(token.KindEOF); err != nil {
		return nil, err
	}

	return value, nil
}

// MustParseValue parses an AST value from a string and panics on errors.
func MustParseValue(source *token.Source) ast.Value {
	value, err := ParseValue(source)
	if err != nil {
		panic(err)
	}
	return value
}

// ParseType parses the AST for string containing a GraphQL Type (e.g., `[Int!]`).
func ParseType(source *token.Source) (ast.Type, error) {
	parser, err := newParser(source, &parseOptions{})
	if err != nil {
		return nil, err
	}

	if _, err := parser.expect(token.KindSOF); err != nil {
		return nil, err
	}

	t, err := parser.parseType()
	if err != nil {
		return nil, err
	}

	if _, err := parser.expect(token.KindEOF); err != nil {
		return nil, err
	}

	return t, nil
}

// MustParseType parses an AST type from a string and panics on errors.
func MustParseType(source *token.Source) ast.Type {
	t, err := ParseType(source)
	if err != nil {
		panic(err)
	}
	return t
}

// ParseType parses the AST for string containing a GraphQL Type (e.g., `[Int!]`).
