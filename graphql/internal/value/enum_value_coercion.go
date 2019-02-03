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
	"errors"
	"reflect"

	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/graphql/ast"
)

// These errors are returned when coercion failed in coerceEnumVariableValue and
// coreceEnumLiteralValue. These are ordinary error instead of CoercionError to let the caller
// present default message to the user instead of these internal details.
var (
	errNilEnumValue      = errors.New("enum value is not provided")
	errInvalidEnumValue  = errors.New("invalid enum value")
	errEnumValueNotFound = errors.New("not a value for the type")
)

// coerceEnumVariableValue coerces a value read from input query variable that specifies a name of
// enum value and return the internal value that represents the enum. Return nil if there's no such
// enum value for given name were found.
func coerceEnumVariableValue(enum graphql.Enum, value interface{}) (interface{}, error) {
	var enumValue graphql.EnumValue
	switch name := value.(type) {
	case string:
		enumValue = enum.Values().Lookup(name)

	case *string:
		if name != nil {
			enumValue = enum.Values().Lookup(*name)
		} else {
			return nil, errNilEnumValue
		}

	default:
		// Check whether the given value is string-like or pointer to string-like via reflection.
		nameValue := reflect.ValueOf(value)
		if nameValue.Kind() == reflect.Ptr {
			if nameValue.IsNil() {
				return nil, errNilEnumValue
			}
			nameValue = nameValue.Elem()
		}

		if nameValue.Kind() != reflect.String {
			return nil, errInvalidEnumValue
		}

		enumValue = enum.Values().Lookup(nameValue.String())
	}

	if enumValue != nil {
		return enumValue.Value(), nil
	}

	return nil, errEnumValueNotFound
}

// coreceEnumLiteralValue is similar to coerceEnumVariableValue but coerces a value from an AST
// value (could come from input field argument) that specifies a name of enum value.
func coreceEnumLiteralValue(enum graphql.Enum, value ast.Value) (interface{}, error) {
	if value, ok := value.(ast.EnumValue); ok {
		if enumValue := enum.Values().Lookup(value.Value()); enumValue != nil {
			return enumValue.Value(), nil
		}
		return nil, errEnumValueNotFound
	}
	return nil, errInvalidEnumValue
}
