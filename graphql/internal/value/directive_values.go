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

package value

import (
	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/graphql/ast"
)

// DirectiveValues prepares an object map of argument values given a directive definition and a AST
// node which may contain directives. Optionally also accepts a map of variable values. If the
// directive does not exist on the node, returns nil.
func DirectiveValues(
	directiveDef *graphql.Directive,
	nodeDirectives ast.Directives,
	variableValues graphql.VariableValues) (graphql.ArgumentValues, error) {

	// Find the directive specified by the node that matches the name of directiveDef.
	var directiveNode *ast.Directive

	for _, directive := range nodeDirectives {
		if directive.Name.Value() == directiveDef.Name() {
			directiveNode = directive
			break
		}
	}

	// Quick return if there's no such directive.
	if directiveNode == nil {
		return graphql.NoArgumentValues(), nil
	}

	return ArgumentValues(directiveDef, directiveNode, variableValues)
}
