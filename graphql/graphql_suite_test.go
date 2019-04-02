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

package graphql_test

import (
	"context"
	"testing"

	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/graphql/executor"
	"github.com/botobag/artemis/graphql/parser"
	"github.com/botobag/artemis/graphql/token"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestGraphQLCore(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "GraphQL Core Suite")
}

func executeQueryWithParams(schema graphql.Schema, query string, params map[string]interface{}) executor.ExecutionResult {
	document, err := parser.Parse(token.NewSource(&token.SourceConfig{
		Body: token.SourceBody([]byte(query)),
	}))
	Expect(err).ShouldNot(HaveOccurred())

	operation, errs := executor.Prepare(schema, document)
	Expect(errs.HaveOccurred()).ShouldNot(BeTrue())

	var result executor.ExecutionResult
	Eventually(
		operation.Execute(context.Background(), executor.VariableValues(params)),
	).Should(Receive(&result))
	return result
}

func executeQuery(schema graphql.Schema, query string) executor.ExecutionResult {
	return executeQueryWithParams(schema, query, nil)
}
