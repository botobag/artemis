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

package typeutil

import (
	"fmt"
	"math"
)

// CoercionMode specified the type of coercion currently running.
type CoercionMode uint

// Enumeration of CoercionMode; There are 2 kinds of coercions (Result Coercion and Input Coercion)
// occurred in GraphQL and are described in [0] and for each builtin scalar type.
//
// [0]: https://facebook.github.io/graphql/June2018/#sec-Scalars
const (
	// The coercion is used to prepare values for result.
	ResultCoercionMode CoercionMode = iota
	// The coercion is used to parse value read from query variables.
	InputCoercionMode
)

// CoercionHelperBase has two purposes:
//
//	1. It implement method dispatching to deliver value based on its type into (most) appropriated
//		 coercion handler in a CoercionHelper implementation.
//	2. It provides default implementation for coercion handlers.
//
// Implementing a CoercionHelper usually embeds CoercionHelperBase to get the default
// implementation:
//
//
//	type MyCoercionHelper struct {
//		CoercionHelperBase
//	}
//
//	// CoerceBool overrides CoercionHelperBase.
//	func (helper *MyCoercionHelper) CoerceBool(value bool, ctx *CoercionContext) (interface{}, error) {
//		...
//	}
type CoercionHelperBase struct {
	impl CoercionHelper
}

// CoercionContext contains context which is passed to coercion handlers.
type CoercionContext struct {
	Mode CoercionMode
}

// CoercionHelper defines an utility class that helps implement coercion for scalars. When
// implementing SerializeResult and ParseVariableValue for a Scalar, we usually need to use type
// switch and deal with many primitive types ({u}int{8,16,32,64}, etc.). Many of them share the same
// code. However, it's not possible to share code in Go's type switch because every `case` in type
// switch can deal with exactly one type. CoercionHelper helps us coalesce the logics by providing a
// "hierarchical type handlers".
//
// CoercionHelper also ensures special Float value, NaN, +Inf and -Inf gets special treat (they
// are not "real" values and therefore cannot serialization in many cases).
//
// To use CoercionHelper, define a struct with CoercionHelperBase embedded.  Then override the
// handler to implement your coercion. Finally, calling Coerce in your ScalarResultSerializer and
// InputParser to execute the coercion.
type CoercionHelper interface {
	RaiseError(value interface{}, ctx *CoercionContext, format string, a ...interface{}) error

	RaiseInvalidTypeError(value interface{}, ctx *CoercionContext) error
	RaiseNonValue(value interface{}, ctx *CoercionContext) error

	CoerceBool(value bool, ctx *CoercionContext) (interface{}, error)

	CoerceSignedInteger(value int64, ctx *CoercionContext) (interface{}, error)
	CoerceInt(value int, ctx *CoercionContext) (interface{}, error)
	CoerceInt8(value int8, ctx *CoercionContext) (interface{}, error)
	CoerceInt16(value int16, ctx *CoercionContext) (interface{}, error)
	CoerceInt32(value int32, ctx *CoercionContext) (interface{}, error)
	CoerceInt64(value int64, ctx *CoercionContext) (interface{}, error)

	CoerceUnsignedInteger(value uint64, ctx *CoercionContext) (interface{}, error)
	CoerceUint(value uint, ctx *CoercionContext) (interface{}, error)
	CoerceUint8(value uint8, ctx *CoercionContext) (interface{}, error)
	CoerceUint16(value uint16, ctx *CoercionContext) (interface{}, error)
	CoerceUint32(value uint32, ctx *CoercionContext) (interface{}, error)
	CoerceUint64(value uint64, ctx *CoercionContext) (interface{}, error)

	CoerceInf(value interface{}, ctx *CoercionContext) (interface{}, error)
	CoerceNaN(value interface{}, ctx *CoercionContext) (interface{}, error)
	CoerceFloat(value float64, ctx *CoercionContext) (interface{}, error)
	CoerceFloat32(value float32, ctx *CoercionContext) (interface{}, error)
	CoerceFloat64(value float64, ctx *CoercionContext) (interface{}, error)

	CoerceString(value string, ctx *CoercionContext) (interface{}, error)

	CoerceNil(value interface{}, ctx *CoercionContext) (interface{}, error)

	CoerceBoolPtr(value *bool, ctx *CoercionContext) (interface{}, error)

	CoerceIntPtr(value *int, ctx *CoercionContext) (interface{}, error)
	CoerceInt8Ptr(value *int8, ctx *CoercionContext) (interface{}, error)
	CoerceInt16Ptr(value *int16, ctx *CoercionContext) (interface{}, error)
	CoerceInt32Ptr(value *int32, ctx *CoercionContext) (interface{}, error)
	CoerceInt64Ptr(value *int64, ctx *CoercionContext) (interface{}, error)

	CoerceUintPtr(value *uint, ctx *CoercionContext) (interface{}, error)
	CoerceUint8Ptr(value *uint8, ctx *CoercionContext) (interface{}, error)
	CoerceUint16Ptr(value *uint16, ctx *CoercionContext) (interface{}, error)
	CoerceUint32Ptr(value *uint32, ctx *CoercionContext) (interface{}, error)
	CoerceUint64Ptr(value *uint64, ctx *CoercionContext) (interface{}, error)

	CoerceFloat32Ptr(value *float32, ctx *CoercionContext) (interface{}, error)
	CoerceFloat64Ptr(value *float64, ctx *CoercionContext) (interface{}, error)

	CoerceStringPtr(value *string, ctx *CoercionContext) (interface{}, error)
}

// SetImpl tells CoercionHelperBase the CoercionHelper implementation for method dispatching.
func (helper *CoercionHelperBase) SetImpl(impl CoercionHelper) {
	helper.impl = impl
}

// Coerce executes the coercion for given value.
func (helper *CoercionHelperBase) Coerce(value interface{}, ctx CoercionContext) (interface{}, error) {
	impl := helper.impl
	if impl == nil {
		panic("need to call Init to initialize CoercionHelperBase before running")
	}

	switch value := value.(type) {
	// Boolean
	case bool:
		return impl.CoerceBool(value, &ctx)

	// Integers
	case int:
		return impl.CoerceInt(value, &ctx)
	case int8:
		return impl.CoerceInt8(value, &ctx)
	case int16:
		return impl.CoerceInt16(value, &ctx)
	case int32:
		return impl.CoerceInt32(value, &ctx)
	case int64:
		return impl.CoerceInt64(value, &ctx)
	case uint:
		return impl.CoerceUint(value, &ctx)
	case uint8:
		return impl.CoerceUint8(value, &ctx)
	case uint16:
		return impl.CoerceUint16(value, &ctx)
	case uint32:
		return impl.CoerceUint32(value, &ctx)
	case uint64:
		return impl.CoerceUint64(value, &ctx)

	// Float
	case float32:
		// There's no math.IsNaN and math.IsInf for float32...
		if value != value {
			return impl.CoerceNaN(value, &ctx)
		} else if value > float32(math.MaxFloat32) {
			return impl.CoerceInf(value, &ctx)
		} else if value < float32(-math.MaxFloat32) {
			return impl.CoerceInf(value, &ctx)
		}
		return impl.CoerceFloat32(value, &ctx)

	case float64:
		if math.IsNaN(value) {
			return impl.CoerceNaN(value, &ctx)
		} else if math.IsInf(value, 1) {
			return impl.CoerceInf(value, &ctx)
		} else if math.IsInf(value, -1) {
			return impl.CoerceInf(value, &ctx)
		}
		return impl.CoerceFloat64(value, &ctx)

	// String
	case string:
		return impl.CoerceString(value, &ctx)

	// Pointer to Boolean
	case *bool:
		return impl.CoerceBoolPtr(value, &ctx)

	// Pointer to Integers
	case *int:
		return impl.CoerceIntPtr(value, &ctx)
	case *int8:
		return impl.CoerceInt8Ptr(value, &ctx)
	case *int16:
		return impl.CoerceInt16Ptr(value, &ctx)
	case *int32:
		return impl.CoerceInt32Ptr(value, &ctx)
	case *int64:
		return impl.CoerceInt64Ptr(value, &ctx)
	case *uint:
		return impl.CoerceUintPtr(value, &ctx)
	case *uint8:
		return impl.CoerceUint8Ptr(value, &ctx)
	case *uint16:
		return impl.CoerceUint16Ptr(value, &ctx)
	case *uint32:
		return impl.CoerceUint32Ptr(value, &ctx)
	case *uint64:
		return impl.CoerceUint64Ptr(value, &ctx)

	// Pointer to Float
	case *float32:
		return impl.CoerceFloat32Ptr(value, &ctx)
	case *float64:
		return impl.CoerceFloat64Ptr(value, &ctx)

	// Pointer to String
	case *string:
		return impl.CoerceStringPtr(value, &ctx)

	// nil value
	case nil:
		return impl.CoerceNil(value, &ctx)
	}

	return nil, impl.RaiseInvalidTypeError(value, &ctx)
}

// RaiseError implements CoercionHelper.
func (helper *CoercionHelperBase) RaiseError(value interface{}, ctx *CoercionContext, format string, a ...interface{}) error {
	return fmt.Errorf("failed to coerce %+v: %s", value, fmt.Sprintf(format, a...))
}

// RaiseInvalidTypeError implements CoercionHelper.
func (helper *CoercionHelperBase) RaiseInvalidTypeError(value interface{}, ctx *CoercionContext) error {
	switch ctx.Mode {
	case ResultCoercionMode:
		return helper.impl.RaiseError(value, ctx, "unexpected result type `%T`", value)

	case InputCoercionMode:
		return helper.impl.RaiseError(value, ctx, "invalid variable type `%T`", value)
	}

	panic("unknown mode")
}

// RaiseNonValue implements CoercionHelper.
func (helper *CoercionHelperBase) RaiseNonValue(value interface{}, ctx *CoercionContext) error {
	return helper.impl.RaiseError(value, ctx, "not a value")
}

// CoerceBool implements CoercionHelper.
func (helper *CoercionHelperBase) CoerceBool(value bool, ctx *CoercionContext) (interface{}, error) {
	return nil, helper.impl.RaiseInvalidTypeError(value, ctx)
}

// CoerceSignedInteger implements CoercionHelper.
func (helper *CoercionHelperBase) CoerceSignedInteger(value int64, ctx *CoercionContext) (interface{}, error) {
	return nil, helper.impl.RaiseInvalidTypeError(value, ctx)
}

// CoerceInt implements CoercionHelper.
func (helper *CoercionHelperBase) CoerceInt(value int, ctx *CoercionContext) (interface{}, error) {
	return helper.impl.CoerceSignedInteger(int64(value), ctx)
}

// CoerceInt8 implements CoercionHelper.
func (helper *CoercionHelperBase) CoerceInt8(value int8, ctx *CoercionContext) (interface{}, error) {
	return helper.impl.CoerceSignedInteger(int64(value), ctx)
}

// CoerceInt16 implements CoercionHelper.
func (helper *CoercionHelperBase) CoerceInt16(value int16, ctx *CoercionContext) (interface{}, error) {
	return helper.impl.CoerceSignedInteger(int64(value), ctx)
}

// CoerceInt32 implements CoercionHelper.
func (helper *CoercionHelperBase) CoerceInt32(value int32, ctx *CoercionContext) (interface{}, error) {
	return helper.impl.CoerceSignedInteger(int64(value), ctx)
}

// CoerceInt64 implements CoercionHelper.
func (helper *CoercionHelperBase) CoerceInt64(value int64, ctx *CoercionContext) (interface{}, error) {
	return helper.impl.CoerceSignedInteger(value, ctx)
}

// CoerceUnsignedInteger implements CoercionHelper.
func (helper *CoercionHelperBase) CoerceUnsignedInteger(value uint64, ctx *CoercionContext) (interface{}, error) {
	return nil, helper.impl.RaiseInvalidTypeError(value, ctx)
}

// CoerceUint implements CoercionHelper.
func (helper *CoercionHelperBase) CoerceUint(value uint, ctx *CoercionContext) (interface{}, error) {
	return helper.impl.CoerceUnsignedInteger(uint64(value), ctx)
}

// CoerceUint8 implements CoercionHelper.
func (helper *CoercionHelperBase) CoerceUint8(value uint8, ctx *CoercionContext) (interface{}, error) {
	return helper.impl.CoerceUnsignedInteger(uint64(value), ctx)
}

// CoerceUint16 implements CoercionHelper.
func (helper *CoercionHelperBase) CoerceUint16(value uint16, ctx *CoercionContext) (interface{}, error) {
	return helper.impl.CoerceUnsignedInteger(uint64(value), ctx)
}

// CoerceUint32 implements CoercionHelper.
func (helper *CoercionHelperBase) CoerceUint32(value uint32, ctx *CoercionContext) (interface{}, error) {
	return helper.impl.CoerceUnsignedInteger(uint64(value), ctx)
}

// CoerceUint64 implements CoercionHelper.
func (helper *CoercionHelperBase) CoerceUint64(value uint64, ctx *CoercionContext) (interface{}, error) {
	return helper.impl.CoerceUnsignedInteger(value, ctx)
}

// CoerceInf implements CoercionHelper.
func (helper *CoercionHelperBase) CoerceInf(value interface{}, ctx *CoercionContext) (interface{}, error) {
	return nil, helper.impl.RaiseNonValue(value, ctx)
}

// CoerceNaN implements CoercionHelper.
func (helper *CoercionHelperBase) CoerceNaN(value interface{}, ctx *CoercionContext) (interface{}, error) {
	return nil, helper.impl.RaiseNonValue(value, ctx)
}

// CoerceFloat implements CoercionHelper.
func (helper *CoercionHelperBase) CoerceFloat(value float64, ctx *CoercionContext) (interface{}, error) {
	return nil, helper.impl.RaiseInvalidTypeError(value, ctx)
}

// CoerceFloat32 implements CoercionHelper.
func (helper *CoercionHelperBase) CoerceFloat32(value float32, ctx *CoercionContext) (interface{}, error) {
	return helper.impl.CoerceFloat(float64(value), ctx)
}

// CoerceFloat64 implements CoercionHelper.
func (helper *CoercionHelperBase) CoerceFloat64(value float64, ctx *CoercionContext) (interface{}, error) {
	return helper.impl.CoerceFloat(value, ctx)
}

// CoerceString implements CoercionHelper.
func (helper *CoercionHelperBase) CoerceString(value string, ctx *CoercionContext) (interface{}, error) {
	return nil, helper.impl.RaiseInvalidTypeError(value, ctx)
}

// CoerceNil implements CoercionHelper.
func (helper *CoercionHelperBase) CoerceNil(value interface{}, ctx *CoercionContext) (interface{}, error) {
	// Accept nil value in coercion.
	return nil, nil
}

// CoerceBoolPtr implements CoercionHelper.
func (helper *CoercionHelperBase) CoerceBoolPtr(value *bool, ctx *CoercionContext) (interface{}, error) {
	if value == nil {
		return helper.impl.CoerceNil(value, ctx)
	}
	return helper.impl.CoerceBool(*value, ctx)
}

// CoerceIntPtr implements CoercionHelper.
func (helper *CoercionHelperBase) CoerceIntPtr(value *int, ctx *CoercionContext) (interface{}, error) {
	if value == nil {
		return helper.impl.CoerceNil(value, ctx)
	}
	return helper.impl.CoerceInt(*value, ctx)
}

// CoerceInt8Ptr implements CoercionHelper.
func (helper *CoercionHelperBase) CoerceInt8Ptr(value *int8, ctx *CoercionContext) (interface{}, error) {
	if value == nil {
		return helper.impl.CoerceNil(value, ctx)
	}
	return helper.impl.CoerceInt8(*value, ctx)
}

// CoerceInt16Ptr implements CoercionHelper.
func (helper *CoercionHelperBase) CoerceInt16Ptr(value *int16, ctx *CoercionContext) (interface{}, error) {
	if value == nil {
		return helper.impl.CoerceNil(value, ctx)
	}
	return helper.impl.CoerceInt16(*value, ctx)
}

// CoerceInt32Ptr implements CoercionHelper.
func (helper *CoercionHelperBase) CoerceInt32Ptr(value *int32, ctx *CoercionContext) (interface{}, error) {
	if value == nil {
		return helper.impl.CoerceNil(value, ctx)
	}
	return helper.impl.CoerceInt32(*value, ctx)
}

// CoerceInt64Ptr implements CoercionHelper.
func (helper *CoercionHelperBase) CoerceInt64Ptr(value *int64, ctx *CoercionContext) (interface{}, error) {
	if value == nil {
		return helper.impl.CoerceNil(value, ctx)
	}
	return helper.impl.CoerceInt64(*value, ctx)
}

// CoerceUintPtr implements CoercionHelper.
func (helper *CoercionHelperBase) CoerceUintPtr(value *uint, ctx *CoercionContext) (interface{}, error) {
	if value == nil {
		return helper.impl.CoerceNil(value, ctx)
	}
	return helper.impl.CoerceUint(*value, ctx)
}

// CoerceUint8Ptr implements CoercionHelper.
func (helper *CoercionHelperBase) CoerceUint8Ptr(value *uint8, ctx *CoercionContext) (interface{}, error) {
	if value == nil {
		return helper.impl.CoerceNil(value, ctx)
	}
	return helper.impl.CoerceUint8(*value, ctx)
}

// CoerceUint16Ptr implements CoercionHelper.
func (helper *CoercionHelperBase) CoerceUint16Ptr(value *uint16, ctx *CoercionContext) (interface{}, error) {
	if value == nil {
		return helper.impl.CoerceNil(value, ctx)
	}
	return helper.impl.CoerceUint16(*value, ctx)
}

// CoerceUint32Ptr implements CoercionHelper.
func (helper *CoercionHelperBase) CoerceUint32Ptr(value *uint32, ctx *CoercionContext) (interface{}, error) {
	if value == nil {
		return helper.impl.CoerceNil(value, ctx)
	}
	return helper.impl.CoerceUint32(*value, ctx)
}

// CoerceUint64Ptr implements CoercionHelper.
func (helper *CoercionHelperBase) CoerceUint64Ptr(value *uint64, ctx *CoercionContext) (interface{}, error) {
	if value == nil {
		return helper.impl.CoerceNil(value, ctx)
	}
	return helper.impl.CoerceUint64(*value, ctx)
}

// CoerceFloat32Ptr implements CoercionHelper.
func (helper *CoercionHelperBase) CoerceFloat32Ptr(value *float32, ctx *CoercionContext) (interface{}, error) {
	if value == nil {
		return helper.impl.CoerceNil(value, ctx)
	}
	return helper.impl.CoerceFloat32(*value, ctx)
}

// CoerceFloat64Ptr implements CoercionHelper.
func (helper *CoercionHelperBase) CoerceFloat64Ptr(value *float64, ctx *CoercionContext) (interface{}, error) {
	if value == nil {
		return helper.impl.CoerceNil(value, ctx)
	}
	return helper.impl.CoerceFloat64(*value, ctx)
}

// CoerceStringPtr implements CoercionHelper.
func (helper *CoercionHelperBase) CoerceStringPtr(value *string, ctx *CoercionContext) (interface{}, error) {
	if value == nil {
		return helper.impl.CoerceNil(value, ctx)
	}
	return helper.impl.CoerceString(*value, ctx)
}
