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
	"unsafe"

	"github.com/botobag/artemis/graphql"

	"github.com/json-iterator/go"
)

// resultMarshaller implements jsoniter.ValEncoder to encode ExecutionResult to JSON.
type resultMarshaller struct{}

var _ jsoniter.ValEncoder = resultMarshaller{}

// IsEmpty implements jsoniter.ValEncoder.
func (resultMarshaller) IsEmpty(ptr unsafe.Pointer) bool {
	result := (*ExecutionResult)(ptr)
	return result == nil || (result.Data == nil && !result.Errors.HaveOccurred())
}

// Encode implements jsoniter.ValEncoder.
func (resultMarshaller) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	result := (*ExecutionResult)(ptr)
	stream.WriteObjectStart()

	// Specification [0] suggests placing the "errors" first in response to make it clear.
	//
	// [0]: See the note for https://facebook.github.io/graphql/June2018/#sec-Response-Format.
	if result.Errors.HaveOccurred() {
		stream.WriteObjectField("errors")
		stream.WriteVal(result.Errors)
		if result.Data != nil {
			stream.WriteMore()
		}
	}

	if result.Data != nil {
		stream.WriteObjectField("data")
		stream.WriteVal(result.Data)
	}

	stream.WriteObjectEnd()
}

// MarshalJSON implements json.Marshaler interface for ExecutionResult.
func (result ExecutionResult) MarshalJSON() ([]byte, error) {
	return jsoniter.Marshal(&result)
}

// resultNodeMarshaller implements jsoniter.ValEncoder to encode a ResultNode to JSON.
type resultNodeMarshaller struct{}

// IsEmpty implements jsoniter.ValEncoder.
func (resultNodeMarshaller) IsEmpty(ptr unsafe.Pointer) bool {
	result := (*ResultNode)(ptr)
	return result == nil
}

// Encode implements jsoniter.ValEncoder.
func (resultNodeMarshaller) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	var (
		// objectEndTask calls stream.WriteObjectEnd().
		objectEndTask interface{} = &struct{ int }{1}
		// arrayEndTask calls stream.WriteArrayEnd().
		arrayEndTask interface{} = &struct{ int }{2}
		// moreTask calls stream.WriteMore().
		moreTask interface{} = &struct{ int }{3}
		stack                = []interface{}{(*ResultNode)(ptr)}
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
				values := result.ListValue()
				if len(values) == 0 {
					stream.WriteEmptyArray()
				} else {
					stream.WriteArrayStart()
					stack = append(stack, arrayEndTask)
					for i := len(values) - 1; i >= 0; i-- {
						stack = append(stack, &values[i], moreTask)
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
						stream.Error = graphql.NewError("malformed object result value: mismatch length of " +
							"field values with the execution nodes")
						return
					}

					for i := len(nodes) - 1; i >= 0; i-- {
						stack = append(stack, &values[i], nodes[i], moreTask)
					}
					// Pop the moreTask at the top. Don't write "," before first field.
					stack = stack[:len(stack)-1]
				}

			case ResultKindLeaf:
				stream.WriteVal(result.Value)
			}
		}
	}
}

// MarshalJSON implements json.Marshaler interface for ResultNode.
func (result *ResultNode) MarshalJSON() ([]byte, error) {
	return jsoniter.Marshal(result)
}

func init() {
	jsoniter.RegisterTypeEncoder("executor.ExecutionResult", resultMarshaller{})
	jsoniter.RegisterTypeEncoder("executor.ResultNode", resultNodeMarshaller{})
}
