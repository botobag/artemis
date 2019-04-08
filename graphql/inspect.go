/**
 * Copyright (c) 2019, The Artemis Authors.
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

package graphql

import (
	"fmt"
	"io"
	"reflect"
	"runtime"
	"strconv"

	"github.com/botobag/artemis/internal/unsafe"
	"github.com/botobag/artemis/internal/util"
	"github.com/botobag/artemis/jsonwriter"
)

const (
	maxArrayLength    = 10
	maxRecursiveDepth = 2
)

// ValueWithCustomInspect provides custom inspect function to serialize value in Inspect.
type ValueWithCustomInspect interface {
	Inspect(out io.Writer) error
}

// InspectTo prints Go values v to the given buf in the same format as graphql-js's inspect
// function. The implementation matches
// https://github.com/graphql/graphql-js/blob/1375776/src/jsutils/inspect.js.
//
// Note that errors returned from out.Write are ignored.
func InspectTo(out io.Writer, v interface{}) error {
	return inspectTo(out, v, nil)
}

func inspectTo(out io.Writer, v interface{}, seenValues []interface{}) error {
	if v, ok := v.(ValueWithCustomInspect); ok {
		return v.Inspect(out)
	}
	// Special types that have custom inspection
	switch v := v.(type) {
	case Type:
		inspectTypeTo(out, v)
		return nil

	case Directive:
		out.Write([]byte{'@'})
		out.Write(unsafe.Bytes(v.Name()))
		return nil

	case ValueWithCustomInspect:
		return v.Inspect(out)
	}

	value := reflect.ValueOf(v)
	switch value.Kind() {
	case reflect.String:
		// graphql-js: JSON.stringify(value)
		stream := jsonwriter.NewStream(out)
		stream.WriteString(v.(string))
		if err := stream.Flush(); err != nil {
			return err
		}

	case reflect.Func:
		f := runtime.FuncForPC(value.Pointer())
		out.Write([]byte{'[', 'f', 'u', 'n', 'c', 't', 'i', 'o', 'n', ' '})
		out.Write(unsafe.Bytes(f.Name()))
		out.Write([]byte{']'})

	case reflect.Array, reflect.Slice:
		seenValues = append(seenValues, v)

		size := value.Len()
		if size == 0 {
			out.Write([]byte{'[', ']'})
			break
		}

		if len(seenValues) > maxRecursiveDepth {
			out.Write([]byte{'[', 'A', 'r', 'r', 'a', 'y', ']'})
			break
		}

		out.Write([]byte{'['})

		// Write the first item.
		if err := inspectToWithCircularCheck(out, value.Index(0).Interface(), seenValues); err != nil {
			return err
		}

		// Write the remaining items.
		l := size
		if l > maxArrayLength {
			l = maxArrayLength
		}
		remaining := size - l
		for i := 1; i < l; i++ {
			out.Write([]byte{',', ' '})
			if err := inspectToWithCircularCheck(out, value.Index(i).Interface(), seenValues); err != nil {
				return err
			}
		}
		if remaining == 1 {
			out.Write([]byte{',', ' ', '.', '.', '.', ' ', '1', ' ', 'm', 'o', 'r', 'e', ' ', 'i', 't', 'e', 'm'})
		} else if remaining > 1 {
			out.Write([]byte{',', ' ', '.', '.', '.', ' '})
			out.Write(strconv.AppendInt(nil, int64(remaining), 10))
			out.Write([]byte{' ', 'm', 'o', 'r', 'e', ' ', 'i', 't', 'e', 'm', 's'})
		}
		out.Write([]byte{']'})

	case reflect.Map:
		seenValues = append(seenValues, v)

		size := value.Len()
		if size == 0 {
			out.Write([]byte{'{', '}'})
			break
		}

		if len(seenValues) > maxRecursiveDepth {
			out.Write([]byte{'[', 'M', 'a', 'p', ']'})
			break
		}

		out.Write([]byte{'{', ' '})

		keys := value.MapKeys()
		for i, key := range keys {
			// Write key.
			if err := inspectToWithCircularCheck(out, key.Interface(), seenValues); err != nil {
				return err
			}
			out.Write([]byte{':', ' '})
			// Write value.
			if err := inspectToWithCircularCheck(out, value.MapIndex(key).Interface(), seenValues); err != nil {
				return err
			}
			if i != len(keys)-1 {
				out.Write([]byte{',', ' '})
			}
		}

		out.Write([]byte{' ', '}'})

	case reflect.Struct:
		seenValues = append(seenValues, v)

		typ := value.Type()
		numFields := typ.NumField()
		if numFields == 0 {
			out.Write([]byte{'{', '}'})
			break
		}

		if len(seenValues) > maxRecursiveDepth {
			out.Write([]byte{'['})
			if len(typ.Name()) == 0 {
				out.Write([]byte{'O', 'b', 'j', 'e', 'c', 't'})
			} else {
				out.Write(unsafe.Bytes(typ.Name()))
			}
			out.Write([]byte{']'})
			break
		}

		out.Write([]byte{'{'})

		// Set to true if any field was printed.
		printed := false
		for i := 0; i < numFields; i++ {
			fieldValue := value.Field(i)
			if !fieldValue.CanInterface() {
				// Skip unexported fields.
				continue
			}

			if printed {
				out.Write([]byte{',', ' '})
			} else {
				// Add a space after "{".
				out.Write([]byte{' '})
				printed = true
			}

			field := typ.Field(i)
			// Write field name.
			out.Write(unsafe.Bytes(field.Name))
			out.Write([]byte{':', ' '})

			// Write value.
			if err := inspectToWithCircularCheck(out, fieldValue.Interface(), seenValues); err != nil {
				return err
			}
		}

		if printed {
			out.Write([]byte{' '})
		}
		out.Write([]byte{'}'})

	case reflect.Ptr:
		elem := value.Elem()
		if !elem.IsValid() {
			out.Write([]byte{'n', 'u', 'l', 'l'})
			return nil
		}
		return inspectToWithCircularCheck(out, elem.Interface(), seenValues)

	case reflect.Invalid:
		out.Write([]byte{'n', 'u', 'l', 'l'})

	default:
		if _, err := fmt.Fprint(out, v); err != nil {
			return err
		}
	}

	return nil
}

func inspectTypeTo(out io.Writer, t Type) {
	var wrapTypes []byte

	for {
		switch ttype := t.(type) {
		case TypeWithName:
			out.Write(unsafe.Bytes(ttype.Name()))
			// Reverse wrapTypes.
			n := len(wrapTypes)
			for i := 0; i < n/2; i++ {
				wrapTypes[i], wrapTypes[n-i-1] = wrapTypes[n-i-1], wrapTypes[i]
			}
			out.Write(wrapTypes)
			return

		case List:
			out.Write([]byte{'['})
			wrapTypes = append(wrapTypes, ']')
			t = ttype.ElementType()

		case NonNull:
			wrapTypes = append(wrapTypes, '!')
			t = ttype.InnerType()

		default:
			panic(fmt.Sprintf("unknown Type object: %T", t))
		}
	}
}

func inspectToWithCircularCheck(out io.Writer, v interface{}, previouslySeenValues []interface{}) error {
	for _, previouslySeenValue := range previouslySeenValues {
		if reflect.DeepEqual(previouslySeenValue, v) {
			out.Write([]byte{'[', 'C', 'i', 'r', 'c', 'u', 'l', 'a', 'r', ']'})
			return nil
		}
	}

	return inspectTo(out, v, previouslySeenValues)
}

// Inspect calls InspectOrErr but panics on error.
func Inspect(v interface{}) string {
	var buf util.StringBuilder
	if err := InspectTo(&buf, v); err != nil {
		panic(fmt.Sprintf("inspect %+v with error: %s", v, err))
	}
	return buf.String()
}
