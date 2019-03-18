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

	"github.com/botobag/artemis/internal/util"
	"github.com/botobag/artemis/jsonwriter"
)

// ValueWithCustomInspect provides custom inspect function to serialize value in Inspect.
type ValueWithCustomInspect interface {
	Inspect(out io.Writer) error
}

// InspectTo prints Go values v to the given buf in the same format as graphql-js's inspect
// function. The implementation matches
// https://github.com/graphql/graphql-js/blob/4cdc8e2/src/jsutils/inspect.js.
//
// Note that errors returned from out.Write are ignored.
func InspectTo(out io.Writer, v interface{}) error {
	if v, ok := v.(ValueWithCustomInspect); ok {
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
		out.Write([]byte(f.Name()))
		out.Write([]byte{']'})

	case reflect.Array, reflect.Slice:
		out.Write([]byte{'['})
		size := value.Len()
		if size > 0 {
			if err := InspectTo(out, value.Index(0).Interface()); err != nil {
				return err
			}
			for i := 1; i < size; i++ {
				out.Write([]byte{',', ' '})
				if err := InspectTo(out, value.Index(i).Interface()); err != nil {
					return err
				}
			}
		}
		out.Write([]byte{']'})

	case reflect.Map:
		size := value.Len()
		if size == 0 {
			out.Write([]byte{'{', '}'})
		} else {
			out.Write([]byte{'{', ' '})

			keys := value.MapKeys()
			for i, key := range keys {
				// Write key.
				if err := InspectTo(out, key.Interface()); err != nil {
					return err
				}
				out.Write([]byte{':', ' '})
				// Write value.
				if err := InspectTo(out, value.MapIndex(key).Interface()); err != nil {
					return err
				}
				if i != len(keys)-1 {
					out.Write([]byte{',', ' '})
				}
			}

			out.Write([]byte{' ', '}'})
		}

	case reflect.Struct:
		typ := value.Type()
		numFields := typ.NumField()
		if numFields == 0 {
			out.Write([]byte{'{', '}'})
		} else {
			out.Write([]byte{'{', ' '})

			for i := 0; i < numFields; i++ {
				field := typ.Field(i)
				// Write field name.
				out.Write([]byte(field.Name))
				out.Write([]byte{':', ' '})

				// Write value.
				if err := InspectTo(out, value.Field(i).Interface()); err != nil {
					return err
				}

				if i != numFields-1 {
					out.Write([]byte{',', ' '})
				}
			}

			out.Write([]byte{' ', '}'})
		}

	case reflect.Ptr:
		elem := value.Elem()
		if !elem.IsValid() {
			out.Write([]byte{'n', 'u', 'l', 'l'})
			return nil
		}
		return InspectTo(out, elem.Interface())

	case reflect.Invalid:
		out.Write([]byte{'n', 'u', 'l', 'l'})

	default:
		if _, err := fmt.Fprint(out, v); err != nil {
			return err
		}
	}

	return nil
}

// Inspect calls InspectOrErr but panics on error.
func Inspect(v interface{}) string {
	var buf util.StringBuilder
	if err := InspectTo(&buf, v); err != nil {
		panic(fmt.Sprintf("inspect %+v with error: %s", v, err))
	}
	return buf.String()
}
