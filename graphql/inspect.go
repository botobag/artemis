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

// InspectToBuf prints Go values v to the given buf in the same format as graphql-js's inspect
// function.  The implementation matches
// https://github.com/graphql/graphql-js/blob/4cdc8e2/src/jsutils/inspect.js.
func InspectToBuf(v interface{}, out io.Writer) error {
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
		if _, err := out.Write([]byte{'[', 'f', 'u', 'n', 'c', 't', 'i', 'o', 'n', ' '}); err != nil {
			return err
		}
		if _, err := out.Write([]byte(f.Name())); err != nil {
			return err
		}
		if _, err := out.Write([]byte{']'}); err != nil {
			return err
		}

	case reflect.Array, reflect.Slice:
		if _, err := out.Write([]byte{'['}); err != nil {
			return err
		}
		size := value.Len()
		if size > 0 {
			if err := InspectToBuf(value.Index(0).Interface(), out); err != nil {
				return err
			}
			for i := 1; i < size; i++ {
				if _, err := out.Write([]byte{',', ' '}); err != nil {
					return err
				}
				if err := InspectToBuf(value.Index(i).Interface(), out); err != nil {
					return err
				}
			}
		}
		if _, err := out.Write([]byte{']'}); err != nil {
			return err
		}

	case reflect.Map:
		size := value.Len()
		if size == 0 {
			if _, err := out.Write([]byte{'{', '}'}); err != nil {
				return err
			}
		} else {
			if _, err := out.Write([]byte{'{', ' '}); err != nil {
				return err
			}

			keys := value.MapKeys()
			for i, key := range keys {
				// Write key.
				if err := InspectToBuf(key.Interface(), out); err != nil {
					return err
				}
				if _, err := out.Write([]byte{':', ' '}); err != nil {
					return err
				}
				// Write value.
				if err := InspectToBuf(value.MapIndex(key).Interface(), out); err != nil {
					return err
				}
				if i != len(keys)-1 {
					if _, err := out.Write([]byte{',', ' '}); err != nil {
						return err
					}
				}
			}

			if _, err := out.Write([]byte{' ', '}'}); err != nil {
				return err
			}
		}

	case reflect.Struct:
		typ := value.Type()
		numFields := typ.NumField()
		if numFields == 0 {
			if _, err := out.Write([]byte{'{', '}'}); err != nil {
				return err
			}
		} else {
			if _, err := out.Write([]byte{'{', ' '}); err != nil {
				return err
			}

			for i := 0; i < numFields; i++ {
				field := typ.Field(i)
				// Write field name.
				if _, err := out.Write([]byte(field.Name)); err != nil {
					return err
				}

				if _, err := out.Write([]byte{':', ' '}); err != nil {
					return err
				}

				// Write value.
				if err := InspectToBuf(value.Field(i).Interface(), out); err != nil {
					return err
				}

				if i != numFields-1 {
					if _, err := out.Write([]byte{',', ' '}); err != nil {
						return err
					}
				}
			}

			if _, err := out.Write([]byte{' ', '}'}); err != nil {
				return err
			}
		}

	case reflect.Ptr:
		elem := value.Elem()
		if !elem.IsValid() {
			_, err := out.Write([]byte{'n', 'u', 'l', 'l'})
			return err
		}
		return InspectToBuf(elem.Interface(), out)

	case reflect.Invalid:
		if _, err := out.Write([]byte{'n', 'u', 'l', 'l'}); err != nil {
			return err
		}

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
	if err := InspectToBuf(v, &buf); err != nil {
		panic(fmt.Sprintf("inspect %+v with error: %s", v, err))
	}
	return buf.String()
}
