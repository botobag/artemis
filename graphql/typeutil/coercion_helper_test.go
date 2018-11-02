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

package typeutil_test

import (
	"math"

	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/graphql/typeutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// tracedCoercionHelper implments graphql.CoercionHelper which traces the method dispatching in
// CoercionHelperBase.
type tracedCoercionHelper struct {
	// Avoid using embedded struct to force tracedCoercionHelper to provide implementation for all
	// interfaces required by CoercionHelper and not inherits missing ones from tracedCoercionHelper.
	base typeutil.CoercionHelperBase

	traces []string
}

func (helper *tracedCoercionHelper) appendTrace(trace string) {
	helper.traces = append(helper.traces, trace)
}

// Type implments graphql.CoercionHelper.
func (helper *tracedCoercionHelper) Type() graphql.Type {
	// It's ok to return nil as we never use it.
	return nil
}

// RaiseError implments graphql.CoercionHelper.
func (helper *tracedCoercionHelper) RaiseError(value interface{}, ctx *typeutil.CoercionContext, format string, a ...interface{}) error {
	helper.appendTrace("RaiseError")
	return nil
}

// RaiseInvalidTypeError implments graphql.CoercionHelper.
func (helper *tracedCoercionHelper) RaiseInvalidTypeError(value interface{}, ctx *typeutil.CoercionContext) error {
	helper.appendTrace("RaiseInvalidTypeError")
	return helper.base.RaiseInvalidTypeError(value, ctx)
}

// RaiseNonValue implments graphql.CoercionHelper.
func (helper *tracedCoercionHelper) RaiseNonValue(value interface{}, ctx *typeutil.CoercionContext) error {
	helper.appendTrace("RaiseNonValue")
	return helper.base.RaiseNonValue(value, ctx)
}

// CoerceBool implments graphql.CoercionHelper.
func (helper *tracedCoercionHelper) CoerceBool(value bool, ctx *typeutil.CoercionContext) (interface{}, error) {
	helper.appendTrace("CoerceBool")
	return helper.base.CoerceBool(value, ctx)
}

// CoerceSignedInteger implments graphql.CoercionHelper.
func (helper *tracedCoercionHelper) CoerceSignedInteger(value int64, ctx *typeutil.CoercionContext) (interface{}, error) {
	helper.appendTrace("CoerceSignedInteger")
	return helper.base.CoerceSignedInteger(value, ctx)
}

// CoerceInt implments graphql.CoercionHelper.
func (helper *tracedCoercionHelper) CoerceInt(value int, ctx *typeutil.CoercionContext) (interface{}, error) {
	helper.appendTrace("CoerceInt")
	return helper.base.CoerceInt(value, ctx)
}

// CoerceInt8 implments graphql.CoercionHelper.
func (helper *tracedCoercionHelper) CoerceInt8(value int8, ctx *typeutil.CoercionContext) (interface{}, error) {
	helper.appendTrace("CoerceInt8")
	return helper.base.CoerceInt8(value, ctx)
}

// CoerceInt16 implments graphql.CoercionHelper.
func (helper *tracedCoercionHelper) CoerceInt16(value int16, ctx *typeutil.CoercionContext) (interface{}, error) {
	helper.appendTrace("CoerceInt16")
	return helper.base.CoerceInt16(value, ctx)
}

// CoerceInt32 implments graphql.CoercionHelper.
func (helper *tracedCoercionHelper) CoerceInt32(value int32, ctx *typeutil.CoercionContext) (interface{}, error) {
	helper.appendTrace("CoerceInt32")
	return helper.base.CoerceInt32(value, ctx)
}

// CoerceInt64 implments graphql.CoercionHelper.
func (helper *tracedCoercionHelper) CoerceInt64(value int64, ctx *typeutil.CoercionContext) (interface{}, error) {
	helper.appendTrace("CoerceInt64")
	return helper.base.CoerceInt64(value, ctx)
}

// CoerceUnsignedInteger implments graphql.CoercionHelper.
func (helper *tracedCoercionHelper) CoerceUnsignedInteger(value uint64, ctx *typeutil.CoercionContext) (interface{}, error) {
	helper.appendTrace("CoerceUnsignedInteger")
	return helper.base.CoerceUnsignedInteger(value, ctx)
}

// CoerceUint implments graphql.CoercionHelper.
func (helper *tracedCoercionHelper) CoerceUint(value uint, ctx *typeutil.CoercionContext) (interface{}, error) {
	helper.appendTrace("CoerceUint")
	return helper.base.CoerceUint(value, ctx)
}

// CoerceUint8 implments graphql.CoercionHelper.
func (helper *tracedCoercionHelper) CoerceUint8(value uint8, ctx *typeutil.CoercionContext) (interface{}, error) {
	helper.appendTrace("CoerceUint8")
	return helper.base.CoerceUint8(value, ctx)
}

// CoerceUint16 implments graphql.CoercionHelper.
func (helper *tracedCoercionHelper) CoerceUint16(value uint16, ctx *typeutil.CoercionContext) (interface{}, error) {
	helper.appendTrace("CoerceUint16")
	return helper.base.CoerceUint16(value, ctx)
}

// CoerceUint32 implments graphql.CoercionHelper.
func (helper *tracedCoercionHelper) CoerceUint32(value uint32, ctx *typeutil.CoercionContext) (interface{}, error) {
	helper.appendTrace("CoerceUint32")
	return helper.base.CoerceUint32(value, ctx)
}

// CoerceUint64 implments graphql.CoercionHelper.
func (helper *tracedCoercionHelper) CoerceUint64(value uint64, ctx *typeutil.CoercionContext) (interface{}, error) {
	helper.appendTrace("CoerceUint64")
	return helper.base.CoerceUint64(value, ctx)
}

// CoerceInf implments graphql.CoercionHelper.
func (helper *tracedCoercionHelper) CoerceInf(value interface{}, ctx *typeutil.CoercionContext) (interface{}, error) {
	helper.appendTrace("CoerceInf")
	return helper.base.CoerceInf(value, ctx)
}

// CoerceNaN implments graphql.CoercionHelper.
func (helper *tracedCoercionHelper) CoerceNaN(value interface{}, ctx *typeutil.CoercionContext) (interface{}, error) {
	helper.appendTrace("CoerceNaN")
	return helper.base.CoerceNaN(value, ctx)
}

// CoerceFloat implments graphql.CoercionHelper.
func (helper *tracedCoercionHelper) CoerceFloat(value float64, ctx *typeutil.CoercionContext) (interface{}, error) {
	helper.appendTrace("CoerceFloat")
	return helper.base.CoerceFloat(value, ctx)
}

// CoerceFloat32 implments graphql.CoercionHelper.
func (helper *tracedCoercionHelper) CoerceFloat32(value float32, ctx *typeutil.CoercionContext) (interface{}, error) {
	helper.appendTrace("CoerceFloat32")
	return helper.base.CoerceFloat32(value, ctx)
}

// CoerceFloat64 implments graphql.CoercionHelper.
func (helper *tracedCoercionHelper) CoerceFloat64(value float64, ctx *typeutil.CoercionContext) (interface{}, error) {
	helper.appendTrace("CoerceFloat64")
	return helper.base.CoerceFloat64(value, ctx)
}

// CoerceString implments graphql.CoercionHelper.
func (helper *tracedCoercionHelper) CoerceString(value string, ctx *typeutil.CoercionContext) (interface{}, error) {
	helper.appendTrace("CoerceString")
	return helper.base.CoerceString(value, ctx)
}

// CoerceNil implments graphql.CoercionHelper.
func (helper *tracedCoercionHelper) CoerceNil(value interface{}, ctx *typeutil.CoercionContext) (interface{}, error) {
	helper.appendTrace("CoerceNil")
	return helper.base.CoerceNil(value, ctx)
}

// CoerceBoolPtr implments graphql.CoercionHelper.
func (helper *tracedCoercionHelper) CoerceBoolPtr(value *bool, ctx *typeutil.CoercionContext) (interface{}, error) {
	helper.appendTrace("CoerceBoolPtr")
	return helper.base.CoerceBoolPtr(value, ctx)
}

// CoerceIntPtr implments graphql.CoercionHelper.
func (helper *tracedCoercionHelper) CoerceIntPtr(value *int, ctx *typeutil.CoercionContext) (interface{}, error) {
	helper.appendTrace("CoerceIntPtr")
	return helper.base.CoerceIntPtr(value, ctx)
}

// CoerceInt8Ptr implments graphql.CoercionHelper.
func (helper *tracedCoercionHelper) CoerceInt8Ptr(value *int8, ctx *typeutil.CoercionContext) (interface{}, error) {
	helper.appendTrace("CoerceInt8Ptr")
	return helper.base.CoerceInt8Ptr(value, ctx)
}

// CoerceInt16Ptr implments graphql.CoercionHelper.
func (helper *tracedCoercionHelper) CoerceInt16Ptr(value *int16, ctx *typeutil.CoercionContext) (interface{}, error) {
	helper.appendTrace("CoerceInt16Ptr")
	return helper.base.CoerceInt16Ptr(value, ctx)
}

// CoerceInt32Ptr implments graphql.CoercionHelper.
func (helper *tracedCoercionHelper) CoerceInt32Ptr(value *int32, ctx *typeutil.CoercionContext) (interface{}, error) {
	helper.appendTrace("CoerceInt32Ptr")
	return helper.base.CoerceInt32Ptr(value, ctx)
}

// CoerceInt64Ptr implments graphql.CoercionHelper.
func (helper *tracedCoercionHelper) CoerceInt64Ptr(value *int64, ctx *typeutil.CoercionContext) (interface{}, error) {
	helper.appendTrace("CoerceInt64Ptr")
	return helper.base.CoerceInt64Ptr(value, ctx)
}

// CoerceUintPtr implments graphql.CoercionHelper.
func (helper *tracedCoercionHelper) CoerceUintPtr(value *uint, ctx *typeutil.CoercionContext) (interface{}, error) {
	helper.appendTrace("CoerceUintPtr")
	return helper.base.CoerceUintPtr(value, ctx)
}

// CoerceUint8Ptr implments graphql.CoercionHelper.
func (helper *tracedCoercionHelper) CoerceUint8Ptr(value *uint8, ctx *typeutil.CoercionContext) (interface{}, error) {
	helper.appendTrace("CoerceUint8Ptr")
	return helper.base.CoerceUint8Ptr(value, ctx)
}

// CoerceUint16Ptr implments graphql.CoercionHelper.
func (helper *tracedCoercionHelper) CoerceUint16Ptr(value *uint16, ctx *typeutil.CoercionContext) (interface{}, error) {
	helper.appendTrace("CoerceUint16Ptr")
	return helper.base.CoerceUint16Ptr(value, ctx)
}

// CoerceUint32Ptr implments graphql.CoercionHelper.
func (helper *tracedCoercionHelper) CoerceUint32Ptr(value *uint32, ctx *typeutil.CoercionContext) (interface{}, error) {
	helper.appendTrace("CoerceUint32Ptr")
	return helper.base.CoerceUint32Ptr(value, ctx)
}

// CoerceUint64Ptr implments graphql.CoercionHelper.
func (helper *tracedCoercionHelper) CoerceUint64Ptr(value *uint64, ctx *typeutil.CoercionContext) (interface{}, error) {
	helper.appendTrace("CoerceUint64Ptr")
	return helper.base.CoerceUint64Ptr(value, ctx)
}

// CoerceFloat32Ptr implments graphql.CoercionHelper.
func (helper *tracedCoercionHelper) CoerceFloat32Ptr(value *float32, ctx *typeutil.CoercionContext) (interface{}, error) {
	helper.appendTrace("CoerceFloat32Ptr")
	return helper.base.CoerceFloat32Ptr(value, ctx)
}

// CoerceFloat64Ptr implments graphql.CoercionHelper.
func (helper *tracedCoercionHelper) CoerceFloat64Ptr(value *float64, ctx *typeutil.CoercionContext) (interface{}, error) {
	helper.appendTrace("CoerceFloat64Ptr")
	return helper.base.CoerceFloat64Ptr(value, ctx)
}

// CoerceStringPtr implments graphql.CoercionHelper.
func (helper *tracedCoercionHelper) CoerceStringPtr(value *string, ctx *typeutil.CoercionContext) (interface{}, error) {
	helper.appendTrace("CoerceStringPtr")
	return helper.base.CoerceStringPtr(value, ctx)
}

// Run executes for given value and check the result with expected traces.
func (helper *tracedCoercionHelper) Run(value interface{}) []string {
	// Reset traces.
	helper.traces = []string{}

	// Run.
	_, err := helper.base.Coerce(value, typeutil.CoercionContext{
		Mode: typeutil.ResultCoercionMode,
	})
	Expect(err).ShouldNot(HaveOccurred())

	return helper.traces
}

func newTracedCoercionHelper() *tracedCoercionHelper {
	helper := &tracedCoercionHelper{}
	helper.base.SetImpl(helper)
	return helper
}

var _ = Describe("CoercionHelper", func() {

	Describe("CoercionHelperBase", func() {
		var (
			helper *tracedCoercionHelper

			boolValue bool = true

			intValue   int   = -1
			int8Value  int8  = -12
			int16Value int16 = -123
			int32Value int32 = -1234
			int64Value int64 = -12345

			uintValue   uint   = 1
			uint8Value  uint8  = 2
			uint16Value uint16 = 3
			uint32Value uint32 = 4
			uint64Value uint64 = 5

			float32Value float32 = float32(1.1)
			// See https://docs.oracle.com/javase/8/docs/api/java/lang/Float.html#NaN.
			float32NaN float32 = math.Float32frombits(0x7fc00000)
			// https://docs.oracle.com/javase/8/docs/api/java/lang/Float.html#POSITIVE_INFINITY
			float32PositiveInf float32 = math.Float32frombits(0x7f800000)
			// https://docs.oracle.com/javase/8/docs/api/java/lang/Float.html#NEGATIVE_INFINITY
			float32NegativeInf float32 = math.Float32frombits(0xff800000)

			float64Value               = -1.1
			float64NaN         float64 = math.NaN()
			float64PositiveInf float64 = math.Inf(+1)
			float64NegativeInf float64 = math.Inf(-1)

			emptyString = ""
			stringValue = "hello"
		)

		BeforeEach(func() {
			helper = newTracedCoercionHelper()
		})

		runAndCheck := func(value interface{}, traces ...string) {
			Expect(helper.Run(value)).Should(Equal(traces))
		}

		It("dispatches value based on its type for CoercionHelper", func() {
			runAndCheck(true, "CoerceBool", "RaiseInvalidTypeError", "RaiseError")
			runAndCheck(false, "CoerceBool", "RaiseInvalidTypeError", "RaiseError")
			runAndCheck(0, "CoerceInt", "CoerceSignedInteger", "RaiseInvalidTypeError", "RaiseError")
			runAndCheck(-1, "CoerceInt", "CoerceSignedInteger", "RaiseInvalidTypeError", "RaiseError")
			runAndCheck(int8Value,
				"CoerceInt8", "CoerceSignedInteger", "RaiseInvalidTypeError", "RaiseError")
			runAndCheck(int16Value,
				"CoerceInt16", "CoerceSignedInteger", "RaiseInvalidTypeError", "RaiseError")
			runAndCheck(int32Value,
				"CoerceInt32", "CoerceSignedInteger", "RaiseInvalidTypeError", "RaiseError")
			runAndCheck(int64Value,
				"CoerceInt64", "CoerceSignedInteger", "RaiseInvalidTypeError", "RaiseError")
			runAndCheck(uint(0),
				"CoerceUint", "CoerceUnsignedInteger", "RaiseInvalidTypeError", "RaiseError")
			runAndCheck(uintValue,
				"CoerceUint", "CoerceUnsignedInteger", "RaiseInvalidTypeError", "RaiseError")
			runAndCheck(uint8Value,
				"CoerceUint8", "CoerceUnsignedInteger", "RaiseInvalidTypeError", "RaiseError")
			runAndCheck(uint16Value,
				"CoerceUint16", "CoerceUnsignedInteger", "RaiseInvalidTypeError", "RaiseError")
			runAndCheck(uint32Value,
				"CoerceUint32", "CoerceUnsignedInteger", "RaiseInvalidTypeError", "RaiseError")
			runAndCheck(uint64Value,
				"CoerceUint64", "CoerceUnsignedInteger", "RaiseInvalidTypeError", "RaiseError")

			runAndCheck(float32NaN, "CoerceNaN", "RaiseNonValue", "RaiseError")
			runAndCheck(float32PositiveInf, "CoerceInf", "RaiseNonValue", "RaiseError")
			runAndCheck(float32NegativeInf, "CoerceInf", "RaiseNonValue", "RaiseError")

			runAndCheck(float64NaN, "CoerceNaN", "RaiseNonValue", "RaiseError")
			runAndCheck(float64PositiveInf, "CoerceInf", "RaiseNonValue", "RaiseError")
			runAndCheck(float64NegativeInf, "CoerceInf", "RaiseNonValue", "RaiseError")

			runAndCheck(float32Value,
				"CoerceFloat32", "CoerceFloat", "RaiseInvalidTypeError", "RaiseError")
			runAndCheck(0.0,
				"CoerceFloat64", "CoerceFloat", "RaiseInvalidTypeError", "RaiseError")
			runAndCheck(float64Value,
				"CoerceFloat64", "CoerceFloat", "RaiseInvalidTypeError", "RaiseError")

			runAndCheck(stringValue, "CoerceString", "RaiseInvalidTypeError", "RaiseError")
			runAndCheck(emptyString, "CoerceString", "RaiseInvalidTypeError", "RaiseError")

			// Test nil pointer for each type.
			runAndCheck(nil, "CoerceNil")
			runAndCheck((*bool)(nil), "CoerceBoolPtr", "CoerceNil")
			runAndCheck((*int)(nil), "CoerceIntPtr", "CoerceNil")
			runAndCheck((*int8)(nil), "CoerceInt8Ptr", "CoerceNil")
			runAndCheck((*int16)(nil), "CoerceInt16Ptr", "CoerceNil")
			runAndCheck((*int32)(nil), "CoerceInt32Ptr", "CoerceNil")
			runAndCheck((*int64)(nil), "CoerceInt64Ptr", "CoerceNil")
			runAndCheck((*uint)(nil), "CoerceUintPtr", "CoerceNil")
			runAndCheck((*uint8)(nil), "CoerceUint8Ptr", "CoerceNil")
			runAndCheck((*uint16)(nil), "CoerceUint16Ptr", "CoerceNil")
			runAndCheck((*uint32)(nil), "CoerceUint32Ptr", "CoerceNil")
			runAndCheck((*uint64)(nil), "CoerceUint64Ptr", "CoerceNil")
			runAndCheck((*float32)(nil), "CoerceFloat32Ptr", "CoerceNil")
			runAndCheck((*float64)(nil), "CoerceFloat64Ptr", "CoerceNil")
			runAndCheck((*string)(nil), "CoerceStringPtr", "CoerceNil")

			// Test non-nil pointers.
			runAndCheck(&boolValue, "CoerceBoolPtr", "CoerceBool", "RaiseInvalidTypeError", "RaiseError")

			runAndCheck(&intValue,
				"CoerceIntPtr", "CoerceInt", "CoerceSignedInteger", "RaiseInvalidTypeError", "RaiseError")
			runAndCheck(&int8Value,
				"CoerceInt8Ptr", "CoerceInt8", "CoerceSignedInteger", "RaiseInvalidTypeError", "RaiseError")
			runAndCheck(&int16Value,
				"CoerceInt16Ptr", "CoerceInt16", "CoerceSignedInteger", "RaiseInvalidTypeError", "RaiseError")
			runAndCheck(&int32Value,
				"CoerceInt32Ptr", "CoerceInt32", "CoerceSignedInteger", "RaiseInvalidTypeError", "RaiseError")
			runAndCheck(&int64Value,
				"CoerceInt64Ptr", "CoerceInt64", "CoerceSignedInteger", "RaiseInvalidTypeError", "RaiseError")

			runAndCheck(&uintValue,
				"CoerceUintPtr", "CoerceUint", "CoerceUnsignedInteger", "RaiseInvalidTypeError", "RaiseError")
			runAndCheck(&uint8Value,
				"CoerceUint8Ptr", "CoerceUint8", "CoerceUnsignedInteger", "RaiseInvalidTypeError", "RaiseError")
			runAndCheck(&uint16Value,
				"CoerceUint16Ptr", "CoerceUint16", "CoerceUnsignedInteger", "RaiseInvalidTypeError", "RaiseError")
			runAndCheck(&uint32Value,
				"CoerceUint32Ptr", "CoerceUint32", "CoerceUnsignedInteger", "RaiseInvalidTypeError", "RaiseError")
			runAndCheck(&uint64Value,
				"CoerceUint64Ptr", "CoerceUint64", "CoerceUnsignedInteger", "RaiseInvalidTypeError", "RaiseError")

			runAndCheck(&float32Value,
				"CoerceFloat32Ptr", "CoerceFloat32", "CoerceFloat", "RaiseInvalidTypeError", "RaiseError")
			runAndCheck(&float32NaN,
				"CoerceFloat32Ptr", "CoerceFloat32", "CoerceFloat", "RaiseInvalidTypeError", "RaiseError")
			runAndCheck(&float32PositiveInf,
				"CoerceFloat32Ptr", "CoerceFloat32", "CoerceFloat", "RaiseInvalidTypeError", "RaiseError")
			runAndCheck(&float32NegativeInf,
				"CoerceFloat32Ptr", "CoerceFloat32", "CoerceFloat", "RaiseInvalidTypeError", "RaiseError")

			runAndCheck(&float64Value,
				"CoerceFloat64Ptr", "CoerceFloat64", "CoerceFloat", "RaiseInvalidTypeError", "RaiseError")
			runAndCheck(&float64NaN,
				"CoerceFloat64Ptr", "CoerceFloat64", "CoerceFloat", "RaiseInvalidTypeError", "RaiseError")
			runAndCheck(&float64PositiveInf,
				"CoerceFloat64Ptr", "CoerceFloat64", "CoerceFloat", "RaiseInvalidTypeError", "RaiseError")
			runAndCheck(&float64NegativeInf,
				"CoerceFloat64Ptr", "CoerceFloat64", "CoerceFloat", "RaiseInvalidTypeError", "RaiseError")

			runAndCheck(&emptyString,
				"CoerceStringPtr", "CoerceString", "RaiseInvalidTypeError", "RaiseError")
			runAndCheck(&stringValue,
				"CoerceStringPtr", "CoerceString", "RaiseInvalidTypeError", "RaiseError")
		})
	})
})
