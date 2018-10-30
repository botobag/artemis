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
	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/graphql/ast"
	"github.com/botobag/artemis/graphql/token"
)

// ParseOptions contains configuration options to control parser behavior.
type ParseOptions struct {
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

// Parse parses the given GraphQL source into a Document.
func Parse(source *graphql.Source, options ParseOptions) (ast.Document, error) {
	parser, err := newParser(source, options)
	if err != nil {
		return ast.Document{}, err
	}
	return parser.parseDocument()
}

// ParseValue parses the AST for string containing a GraphQL value (e.g., `[42]`).
func ParseValue(source *graphql.Source) (ast.Value, error) {
	parser, err := newParser(source, ParseOptions{})
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

// ParseType parses the AST for string containing a GraphQL Type (e.g., `[Int!]`).
func ParseType(source *graphql.Source) (ast.Type, error) {
	parser, err := newParser(source, ParseOptions{})
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
