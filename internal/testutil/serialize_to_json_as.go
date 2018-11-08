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

package testutil

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/onsi/gomega/types"
)

type serializeToJSONAsMatcher struct {
	expected interface{}
}

// SerializeToJSONAs returns a Gomega matcher that first serializes actual value into JSON data
// format and then decodes the data into a variable that has the same type as the expected value and
// compare the decoded result against the expected value.
func SerializeToJSONAs(expected interface{}) types.GomegaMatcher {
	return serializeToJSONAsMatcher{
		expected: expected,
	}
}

// Match implements types.GomegaMatcher.
func (matcher serializeToJSONAsMatcher) Match(actual interface{}) (success bool, err error) {
	// Encode actual into JSON data format.
	encodedActual, err := json.Marshal(actual)
	if err != nil {
		return false, fmt.Errorf("SerializeToJSONAs matcher cannot encode actual into JSON: %s", err)
	}

	// Encode expected into JSON data format.
	encodedExpected, err := json.Marshal(matcher.expected)
	if err != nil {
		return false, fmt.Errorf("SerializeToJSONAs matcher cannot encode expected into JSON: %s", err)
	}

	// Allocate an object with the same type as expectedType.
	expectedType := reflect.TypeOf(matcher.expected)
	decodedExpected := reflect.New(expectedType).Interface()
	decodedActual := reflect.New(expectedType).Interface()

	// Unmarshal encodedActual into decodedActual.
	if err := json.Unmarshal(encodedActual, decodedActual); err != nil {
		return false, fmt.Errorf("SerializeToJSONAs matcher cannot re-encode actual value from JSON into type %T: %s", decodedActual, err)
	}
	// Unmarshal encodedExpected into decodedExpected.
	if err := json.Unmarshal(encodedExpected, decodedExpected); err != nil {
		return false, fmt.Errorf("SerializeToJSONAs matcher cannot re-encode expected value from JSON into type %T: %s", decodedExpected, err)
	}

	//fmt.Printf("actual: %#v\n", decodedActual)
	//fmt.Printf("expected: %#v\n", decodedExpected)

	// Compare!
	return reflect.DeepEqual(decodedActual, decodedExpected), nil
}

// FailureMessage implements types.GomegaMatcher.
func (matcher serializeToJSONAsMatcher) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%#v\nto serialize to JSON value as\n\t%#v", actual, matcher.expected)
}

// NegatedFailureMessage implements types.GomegaMatcher.
func (matcher serializeToJSONAsMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%#v\nnot to serialize to JSON value as\n\t%#v", actual, matcher.expected)
}
