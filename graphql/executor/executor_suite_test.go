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
	"bytes"
	"context"
	"fmt"
	"runtime"
	"testing"

	"github.com/botobag/artemis/concurrent"
	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/graphql/ast"
	"github.com/botobag/artemis/graphql/executor"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

func TestGraphQLExecutor(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "GraphQL Executor Suite")
}

func MatchResultInJSON(resultJSON string) types.GomegaMatcher {
	stringify := func(result executor.ExecutionResult) []byte {
		var buf bytes.Buffer
		Expect(result.MarshalJSONTo(&buf)).Should(Succeed())
		return buf.Bytes()
	}
	return Receive(WithTransform(stringify, MatchJSON(resultJSON)))
}

// Prototype of "execute" function
type ExecuteFunc func(schema graphql.Schema, document ast.Document, opts ...interface{}) <-chan executor.ExecutionResult

// execute is a convenient function using in test that wraps executor.Prepare and
// PreparedOperation.Execute. Options passed in opts must each be either an executor.PrepareOption
// or an executor.ExecuteOption, or it panics.
func execute(schema graphql.Schema, document ast.Document, opts ...interface{}) <-chan executor.ExecutionResult {
	// Packing options.
	var (
		prepareOpts []executor.PrepareOption
		executeOpts []executor.ExecuteOption
	)

	for _, opt := range opts {
		switch opt := opt.(type) {
		case executor.PrepareOption:
			prepareOpts = append(prepareOpts, opt)

		case executor.ExecuteOption:
			executeOpts = append(executeOpts, opt)

		default:
			panic(fmt.Sprintf("%+v is not a valid options to execute (should be either "+
				"executor.PrepareOption or executor.ExecuteOption, but got %T", opt, opt))
		}
	}

	operation, errs := executor.Prepare(schema, document, prepareOpts...)
	Expect(errs.HaveOccurred()).ShouldNot(BeTrue())

	return operation.Execute(context.Background(), executeOpts...)
}

// wrapExecute wraps an "execute" function to run with additional options. A good example of usage
// is to redefine "execute" function which appends executor.Runner to the option list passed to
// execute automatically within DescribeExecute:
//
//	var _ = DescribeExecute("...", func(runner concurrent.Executor) {
//		execute := wrapExecute(executor.Runner(runner))
//	})
func wrapExecute(moreOpts ...interface{}) ExecuteFunc {
	return func(schema graphql.Schema, document ast.Document, opts ...interface{}) <-chan executor.ExecutionResult {
		return execute(schema, document, append(opts, moreOpts...)...)
	}
}

func DescribeExecute(message string, body func(runner concurrent.Executor)) bool {
	return Describe(message, func() {
		Context("without concurrent runner", func() {
			body(nil)
		})

		Context("with concurrent runner", func() {
			var runner concurrent.Executor

			BeforeEach(func() {
				var err error
				runner, err = concurrent.NewWorkerPoolExecutor(concurrent.WorkerPoolExecutorConfig{
					MaxPoolSize: uint32(runtime.GOMAXPROCS(-1)),
				})
				Expect(err).ShouldNot(HaveOccurred())
			})

			AfterEach(func() {
				terminated, err := runner.Shutdown()
				Expect(err).ShouldNot(HaveOccurred())
				Eventually(terminated).Should(Receive(BeTrue()))
			})

			body(runner)
		})
	})
}
