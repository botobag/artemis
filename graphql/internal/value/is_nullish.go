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
	"math"
	"reflect"
)

// IsNullish returns true if the given value v is an nil interface or an interface with nil
// underlying value or NaN.
func IsNullish(v interface{}) bool {
	switch value := v.(type) {
	case float32:
		return math.IsNaN(float64(value))

	case float64:
		return math.IsNaN(value)

	case nil:
		return true

	default:
		// Use reflect.Value.IsNil to check underlying value.
		v := reflect.ValueOf(value)
		switch v.Kind() {
		case reflect.Invalid:
			return true
		// See https://golang.org/pkg/reflect/#Value.IsNil.
		case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.Interface, reflect.Slice:
			return v.IsNil()
		}
	}

	return false
}
