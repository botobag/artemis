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

package executor

import (
	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/jsonwriter"
)

// resultMarshaler implements jsonwriter.ValueMarshaler to encode ExecutionResult to JSON.
type resultMarshaler struct {
	result *ExecutionResult
}

// NewExecutionResultMarshaler creates marshaler to write JSON encoding for given ExecutionResult
// with jsonwriter.
func NewExecutionResultMarshaler(result *ExecutionResult) jsonwriter.ValueMarshaler {
	return resultMarshaler{result}
}

// Encode implements jsonwriter.ValueMarshaler.
func (marshaler resultMarshaler) MarshalJSONTo(stream *jsonwriter.Stream) error {
	result := marshaler.result
	stream.WriteObjectStart()

	// Specification [0] suggests placing the "errors" first in response to make it clear.
	//
	// [0]: See the note for https://graphql.github.io/graphql-spec/June2018/#sec-Response-Format.
	if result.Errors.HaveOccurred() {
		stream.WriteObjectField("errors")
		stream.WriteValue(graphql.NewErrorsMarshaler(result.Errors))
		if result.Data != nil {
			stream.WriteMore()
		}
	}

	if result.Data != nil {
		stream.WriteObjectField("data")
		stream.WriteValue(NewResultNodeMarshaler(result.Data))
	}

	stream.WriteObjectEnd()

	return nil
}

// resultNodeMarshaler implements jsonwriter.ValueMarshaler to encode a ResultNode to JSON.
type resultNodeMarshaler struct {
	node *ResultNode
}

// NewResultNodeMarshaler creates marshaler to write JSON encoding for given ResultNode with
// jsonwriter.
func NewResultNodeMarshaler(result *ResultNode) jsonwriter.ValueMarshaler {
	return resultNodeMarshaler{result}
}

// MarshalJSONTo implements jsonwriter.ValueMarshaler.
func (marshaler resultNodeMarshaler) MarshalJSONTo(stream *jsonwriter.Stream) error {
	var (
		// objectEndTask calls stream.WriteObjectEnd().
		objectEndTask interface{} = &struct{ int }{1}
		// arrayEndTask calls stream.WriteArrayEnd().
		arrayEndTask interface{} = &struct{ int }{2}
		// moreTask calls stream.WriteMore().
		moreTask interface{} = &struct{ int }{3}
		stack                = []interface{}{marshaler.node}
	)

	for len(stack) > 0 {
		var task interface{}
		task, stack = stack[len(stack)-1], stack[:len(stack)-1]

		if task == objectEndTask {
			stream.WriteObjectEnd()
		} else if task == arrayEndTask {
			stream.WriteArrayEnd()
		} else if task == moreTask {
			stream.WriteMore()
		} else if node, ok := task.(*ExecutionNode); ok {
			stream.WriteObjectField(node.ResponseKey())
		} else {
			result := task.(*ResultNode)
			switch result.Kind {
			case ResultKindNil:
				stream.WriteNil()

			case ResultKindList:
				nodeList := result.ListValue()
				if nodeList.Empty() {
					stream.WriteEmptyArray()
				} else {
					stream.WriteArrayStart()
					stack = append(stack, arrayEndTask)

					// Traverse nodes in nodeList in reverse order.
					var (
						firstChunk = nodeList.Chunks()
						// Start from the last chunk.
						chunk = firstChunk.Prev()
					)
					for {
						nodes := chunk.Nodes()
						for i := len(nodes) - 1; i >= 0; i-- {
							stack = append(stack, &nodes[i], moreTask)
						}

						// Loop until the first chunk.
						if chunk == firstChunk {
							break
						}

						// Move to the previous chunk in the list.
						chunk = chunk.prev
					}

					// Pop the moreTask at the top. Don't write "," before first element.
					stack = stack[:len(stack)-1]
				}

			case ResultKindObject:
				object := result.ObjectValue()
				if len(object.FieldValues) == 0 {
					// It's not possible in GraphQL though ...
					stream.WriteEmptyObject()
				} else {
					stream.WriteObjectStart()
					stack = append(stack, objectEndTask)

					nodes := object.ExecutionNodes
					values := object.FieldValues
					if len(nodes) != len(values) {
						return graphql.NewError("malformed object result value: mismatch length of " +
							"field values with the execution nodes")
					}

					for i := len(nodes) - 1; i >= 0; i-- {
						stack = append(stack, &values[i], nodes[i], moreTask)
					}
					// Pop the moreTask at the top. Don't write "," before first field.
					stack = stack[:len(stack)-1]
				}

			case ResultKindLeaf:
				stream.WriteInterface(result.Value)
			}
		}
	}

	return nil
}

// MarshalJSON implements json.Marshaler interface for ResultNode.
func (result *ResultNode) MarshalJSON() ([]byte, error) {
	return jsonwriter.Marshal(resultNodeMarshaler{result})
}
