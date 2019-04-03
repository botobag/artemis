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

package executor_test

import (
	"context"
	"testing"

	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/graphql/executor"
	"github.com/botobag/artemis/graphql/parser"
	"github.com/botobag/artemis/graphql/token"
)

var helloWorldSchema = graphql.MustNewSchema(&graphql.SchemaConfig{
	Query: graphql.MustNewObject(&graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"hello": {
				Type: graphql.T(graphql.String()),
				Resolver: graphql.FieldResolverFunc(func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
					return "world", nil
				}),
			},
		},
	}),
})

func BenchmarkSimpleHelloWorldQueryWithoutPreparedOperation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		document, _ := parser.Parse(token.NewSource("{hello}"))
		operation, _ := executor.Prepare(helloWorldSchema, document)
		operation.Execute(context.Background())
	}
}

func BenchmarkSimpleHelloWorldQueryWithPreparedOperation(b *testing.B) {
	document, _ := parser.Parse(token.NewSource("{hello}"))
	operation, _ := executor.Prepare(helloWorldSchema, document)

	for i := 0; i < b.N; i++ {
		operation.Execute(context.Background())
	}
}
