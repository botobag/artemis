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

package graphql

import (
	"fmt"
	"math"
	"strconv"

	"github.com/botobag/artemis/graphql/ast"
	"github.com/botobag/artemis/graphql/typeutil"
)

// The "type of internal value" for each built-in scalar are listed as follows,
//
// +--------------+---------------------------------+
// | GraphQL Type | Go Type ("internal value type") |
// +--------------+---------------------------------+
// | Int          | int                             |
// | Float        | float64                         |
// | String       | string                          |
// | Boolean      | boolean                         |
// | ID           | string                          |
// +--------------+---------------------------------+
//
// That is, the type of underlying value behind the interface{} returned by CoerceLiteralValue and
// CoerceVariableValue are fixed to the one given in the table for each type. Therefore, for
// example, when you receive an Int argument, you can expect you got an "int" not int32 or others.

// Reasons for the error when coercing built-in scalar types
const (
	coercionErrorNonInteger               string = "not an integer"
	coercionErrorIntegerTooLarge                 = "value too large for 32-bit signed integer"
	coercionErrorIntegerTooSmall                 = "value too small for 32-bit signed integer"
	coercionErrorNonNumeric                      = "not a numeric value"
	coercionErrorIntegerToFloatOutOfRange        = "integer that cannot represent with float: out of range"
	coercionErrorNonBoolean                      = "not a boolean value"
)

// scalarCoercerBase is built on top of typeutil.CoercionHelperBase as a shared base to the coercers
// for built-in scalars below.
type scalarCoercerBase struct {
	typeutil.CoercionHelperBase
	typeName string
}

// scalarCoercerBase is a CoercionHelper implementation.
var _ typeutil.CoercionHelper = (*scalarCoercerBase)(nil)

// RaiseError overrides typeutil.CoercionHelperBase.
func (coercer *scalarCoercerBase) RaiseError(value interface{}, ctx *typeutil.CoercionContext, format string, a ...interface{}) error {
	if v, ok := value.(string); ok {
		// Quote the string for pretty printing.
		value = strconv.Quote(v)
	}
	return NewCoercionError("%s cannot represent %v: %s", coercer.typeName, value, fmt.Sprintf(format, a...))
}

// RaiseInvalidArgumentTypeError returns an error indicating an unexpected type in input argument
// coercion.
func (coercer *scalarCoercerBase) RaiseInvalidArgumentTypeError(value ast.Value) error {
	v := value.Interface()
	return NewCoercionError("%s cannot represent %v: unexpected argument node type `%T`", coercer.typeName, v, value)
}

func (coercer *scalarCoercerBase) init(typeName string, impl typeutil.CoercionHelper) {
	coercer.CoercionHelperBase.SetImpl(impl)
	coercer.typeName = typeName
}

//===-----------------------------------------------------------------------------------------===//
// Int
//===-----------------------------------------------------------------------------------------===//
// The Int scalar type represents a signed 32‐bit numeric non‐fractional value as per spec.
//
// Reference: https://facebook.github.io/graphql/June2018/#sec-Int

// intCoercer implements input coercion and result coercion for Int type.
type intCoercer struct {
	scalarCoercerBase
}

var (
	_ ScalarResultCoercer = (*intCoercer)(nil)
	_ ScalarInputCoercer  = (*intCoercer)(nil)
)

func (coercer *intCoercer) init() {
	coercer.scalarCoercerBase.init("Int", coercer)
}

// RaiseNonValue implements typeutil.CoercionHelper.
func (coercer *intCoercer) RaiseNonValue(value interface{}, ctx *typeutil.CoercionContext) error {
	// Use coercionErrorNonInteger for non-value.
	return coercer.RaiseError(value, ctx, coercionErrorNonInteger)
}

// CoerceBool overrides typeutil.CoercionHelperBase.
func (coercer *intCoercer) CoerceBool(value bool, ctx *typeutil.CoercionContext) (interface{}, error) {
	// Input mode only accepts integer values. See "Input Coercion" in [0].
	//
	// [0]: https://facebook.github.io/graphql/June2018/#sec-Int
	if ctx.Mode == typeutil.InputCoercionMode {
		return nil, coercer.RaiseInvalidTypeError(value, ctx)
	}

	if value {
		return 1, nil
	}
	return 0, nil
}

// CoerceSignedInteger overrides typeutil.CoercionHelperBase.
func (coercer *intCoercer) CoerceSignedInteger(value int64, ctx *typeutil.CoercionContext) (interface{}, error) {
	if value > int64(math.MaxInt32) {
		return nil, coercer.RaiseError(value, ctx, coercionErrorIntegerTooLarge)
	} else if value < int64(math.MinInt32) {
		return nil, coercer.RaiseError(value, ctx, coercionErrorIntegerTooSmall)
	}
	return int(value), nil
}

// CoerceUnsignedInteger overrides typeutil.CoercionHelperBase.
func (coercer *intCoercer) CoerceUnsignedInteger(value uint64, ctx *typeutil.CoercionContext) (interface{}, error) {
	if value > uint64(math.MaxInt32) {
		return nil, coercer.RaiseError(value, ctx, coercionErrorIntegerTooLarge)
	}
	return int(value), nil
}

// CoerceFloat overrides typeutil.CoercionHelperBase.
func (coercer *intCoercer) CoerceFloat(value float64, ctx *typeutil.CoercionContext) (interface{}, error) {
	// Input mode only accepts integer values. See "Input Coercion" in [0].
	//
	// [0]: https://facebook.github.io/graphql/June2018/#sec-Int
	if ctx.Mode == typeutil.InputCoercionMode {
		return nil, coercer.RaiseInvalidTypeError(value, ctx)
	}

	// Make sure the conversion is lossless.
	intValue := int32(value)
	if float64(intValue) != value {
		return nil, coercer.RaiseError(value, ctx, coercionErrorNonInteger)
	}
	return int(intValue), nil
}

func (coercer *intCoercer) coerceStringImpl(value string, ctx *typeutil.CoercionContext) (interface{}, error) {
	val, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return nil, coercer.RaiseError(value, ctx, coercionErrorNonInteger)
	}
	return int(val), nil
}

// CoerceString overrides typeutil.CoercionHelperBase.
func (coercer *intCoercer) CoerceString(value string, ctx *typeutil.CoercionContext) (interface{}, error) {
	// Input mode only accepts integer values. See "Input Coercion" in [0].
	//
	// [0]: https://facebook.github.io/graphql/June2018/#sec-Int
	if ctx.Mode == typeutil.InputCoercionMode {
		return nil, coercer.RaiseInvalidTypeError(value, ctx)
	}
	return coercer.coerceStringImpl(value, ctx)
}

// CoerceResultValue implements ScalarResultCoercer.
func (coercer *intCoercer) CoerceResultValue(value interface{}) (interface{}, error) {
	return coercer.Coerce(value, typeutil.CoercionContext{
		Mode: typeutil.ResultCoercionMode,
	})
}

// CoerceVariableValue implements ScalarInputCoercer.
func (coercer *intCoercer) CoerceVariableValue(value interface{}) (interface{}, error) {
	return coercer.Coerce(value, typeutil.CoercionContext{
		Mode: typeutil.InputCoercionMode,
	})
}

// CoerceLiteralValue implements ScalarInputCoercer.
func (coercer *intCoercer) CoerceLiteralValue(value ast.Value) (interface{}, error) {
	ctx := &typeutil.CoercionContext{
		Mode: typeutil.InputCoercionMode,
	}

	switch value := value.(type) {
	case ast.IntValue:
		return coercer.coerceStringImpl(value.String(), ctx)
	}

	// Return a CoercionError to yield a field error.
	return nil, coercer.RaiseInvalidArgumentTypeError(value)
}

// intType implements builtin Int type in GraphQL.
type intType struct {
	ThisIsScalarType
	coercer intCoercer
}

// Name implements TypeWithName.
func (i *intType) Name() string {
	return "Int"
}

// Description implements TypeWithDescription.
func (i *intType) Description() string {
	return "The `Int` scalar type represents non-fractional signed whole numeric " +
		"values. Int can represent values between -(2^31) and 2^31 - 1."
}

// String implements fmt.Stringer.
func (i *intType) String() string {
	return i.Name()
}

// CoerceResultValue implmenets LeafType.
func (i *intType) CoerceResultValue(value interface{}) (interface{}, error) {
	return i.coercer.CoerceResultValue(value)
}

// CoerceVariableValue implmenets Scalar.
func (i *intType) CoerceVariableValue(value interface{}) (interface{}, error) {
	return i.coercer.CoerceVariableValue(value)
}

// CoerceLiteralValue implmenets Scalar.
func (i *intType) CoerceLiteralValue(value ast.Value) (interface{}, error) {
	return i.coercer.CoerceLiteralValue(value)
}

var intTypeInstance = func() Scalar {
	i := &intType{}
	// Initialize coercer.
	i.coercer.init()
	return i
}()

// Int returns the GraphQL builtin Int type definition.
func Int() Scalar {
	return intTypeInstance
}

//===-----------------------------------------------------------------------------------------===//
// Float
//===-----------------------------------------------------------------------------------------===//
// The Float scalar type represents signed double‐precision fractional values as specified by IEEE
// 754.
//
// Reference: https://facebook.github.io/graphql/June2018/#sec-Float

// floatCoercer implements input coercion and result coercion for Float type.
type floatCoercer struct {
	scalarCoercerBase
}

var (
	_ ScalarResultCoercer = (*floatCoercer)(nil)
	_ ScalarInputCoercer  = (*floatCoercer)(nil)
)

func (coercer *floatCoercer) init() {
	coercer.scalarCoercerBase.init("Float", coercer)
}

// ensureValue ensures that the given floating point value is a valid IEEE 754 number. More
// specifically, not an NaN or Inf.
func (coercer *floatCoercer) ensureValue(value float64, ctx *typeutil.CoercionContext) (interface{}, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return nil, coercer.RaiseNonValue(value, ctx)
	}
	return value, nil
}

// RaiseNonValue implements typeutil.CoercionHelper.
func (coercer *floatCoercer) RaiseNonValue(value interface{}, ctx *typeutil.CoercionContext) error {
	// Use coercionErrorNonNumeric for non-value.
	return coercer.RaiseError(value, ctx, coercionErrorNonNumeric)
}

// CoerceBool overrides typeutil.CoercionHelperBase.
func (coercer *floatCoercer) CoerceBool(value bool, ctx *typeutil.CoercionContext) (interface{}, error) {
	// Input mode only accepts integer and float values. See "Input Coercion" in [0].
	//
	// [0]: https://facebook.github.io/graphql/June2018/#sec-Float
	if ctx.Mode == typeutil.InputCoercionMode {
		return nil, coercer.RaiseInvalidTypeError(value, ctx)
	}

	if value {
		return 1.0, nil
	}
	return 0.0, nil
}

// CoerceSignedInteger overrides typeutil.CoercionHelperBase.
func (coercer *floatCoercer) CoerceSignedInteger(value int64, ctx *typeutil.CoercionContext) (interface{}, error) {
	// int and int64 won't get here. They'll be processed specially with range checks.
	return coercer.ensureValue(float64(value), ctx)
}

// CoerceInt overrides typeutil.CoercionHelperBase.
func (coercer *floatCoercer) CoerceInt(value int, ctx *typeutil.CoercionContext) (interface{}, error) {
	return coercer.CoerceInt64(int64(value), ctx)
}

// CoerceInt64 overrides typeutil.CoercionHelperBase.
func (coercer *floatCoercer) CoerceInt64(value int64, ctx *typeutil.CoercionContext) (interface{}, error) {
	floatValue := float64(value)
	if int64(floatValue) != value {
		return nil, coercer.RaiseError(value, ctx, coercionErrorIntegerToFloatOutOfRange)
	}
	return coercer.ensureValue(floatValue, ctx)
}

// CoerceUnsignedInteger overrides typeutil.CoercionHelperBase.
func (coercer *floatCoercer) CoerceUnsignedInteger(value uint64, ctx *typeutil.CoercionContext) (interface{}, error) {
	// uint and uint64 won't get here. They'll be processed specially with range checks.
	return coercer.ensureValue(float64(value), ctx)
}

// CoerceUint overrides typeutil.CoercionHelperBase.
func (coercer *floatCoercer) CoerceUint(value uint, ctx *typeutil.CoercionContext) (interface{}, error) {
	return coercer.CoerceUint64(uint64(value), ctx)
}

// CoerceUint64 overrides typeutil.CoercionHelperBase.
func (coercer *floatCoercer) CoerceUint64(value uint64, ctx *typeutil.CoercionContext) (interface{}, error) {
	floatValue := float64(value)
	if uint64(floatValue) != value {
		return nil, coercer.RaiseError(value, ctx, coercionErrorIntegerToFloatOutOfRange)
	}
	return coercer.ensureValue(floatValue, ctx)
}

// CoerceFloat overrides typeutil.CoercionHelperBase.
func (coercer *floatCoercer) CoerceFloat(value float64, ctx *typeutil.CoercionContext) (interface{}, error) {
	return coercer.ensureValue(value, ctx)
}

func (coercer *floatCoercer) coerceStringImpl(value string, ctx *typeutil.CoercionContext) (interface{}, error) {
	s, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil, coercer.RaiseError(value, ctx, coercionErrorNonNumeric)
	}
	return coercer.ensureValue(s, ctx)
}

// CoerceString overrides typeutil.CoercionHelperBase.
func (coercer *floatCoercer) CoerceString(value string, ctx *typeutil.CoercionContext) (interface{}, error) {
	// Input mode only accepts integer and float values. See "Input Coercion" in [0].
	//
	// [0]: https://facebook.github.io/graphql/June2018/#sec-Float
	if ctx.Mode == typeutil.InputCoercionMode {
		return nil, coercer.RaiseInvalidTypeError(value, ctx)
	}
	return coercer.coerceStringImpl(value, ctx)
}

// CoerceResultValue implements Scalar.
func (coercer *floatCoercer) CoerceResultValue(value interface{}) (interface{}, error) {
	return coercer.Coerce(value, typeutil.CoercionContext{
		Mode: typeutil.ResultCoercionMode,
	})
}

// CoerceVariableValue implements ScalarInputCoercer.
func (coercer *floatCoercer) CoerceVariableValue(value interface{}) (interface{}, error) {
	return coercer.Coerce(value, typeutil.CoercionContext{
		Mode: typeutil.InputCoercionMode,
	})
}

// CoerceLiteralValue implements ScalarInputCoercer.
func (coercer *floatCoercer) CoerceLiteralValue(value ast.Value) (interface{}, error) {
	ctx := &typeutil.CoercionContext{
		Mode: typeutil.InputCoercionMode,
	}

	switch value := value.(type) {
	// Both integer and float are accepted as per spec..
	case ast.FloatValue:
		return coercer.coerceStringImpl(value.String(), ctx)

	case ast.IntValue:
		return coercer.coerceStringImpl(value.String(), ctx)
	}

	return nil, coercer.RaiseInvalidArgumentTypeError(value)
}

// floatType implements builtin Float type in GraphQL.
type floatType struct {
	ThisIsScalarType
	coercer floatCoercer
}

// Name implements TypeWithName.
func (f *floatType) Name() string {
	return "Float"
}

// Description implements TypeWithDescription.
func (f *floatType) Description() string {
	return "The `Float` scalar type represents signed double-precision fractional " +
		"values as specified by [IEEE 754](http://en.wikipedia.org/wiki/IEEE_floating_point). "
}

// String implements fmt.Stringer.
func (f *floatType) String() string {
	return f.Name()
}

// CoerceResultValue implmenets LeafType.
func (f *floatType) CoerceResultValue(value interface{}) (interface{}, error) {
	return f.coercer.CoerceResultValue(value)
}

// CoerceVariableValue implmenets Scalar.
func (f *floatType) CoerceVariableValue(value interface{}) (interface{}, error) {
	return f.coercer.CoerceVariableValue(value)
}

// CoerceLiteralValue implmenets Scalar.
func (f *floatType) CoerceLiteralValue(value ast.Value) (interface{}, error) {
	return f.coercer.CoerceLiteralValue(value)
}

var floatTypeInstance = func() Scalar {
	f := &floatType{}
	// Initialize coercer.
	f.coercer.init()
	return f
}()

// Float returns the GraphQL builtin Float type definition.
func Float() Scalar {
	return floatTypeInstance
}

//===-----------------------------------------------------------------------------------------===//
// String
//===-----------------------------------------------------------------------------------------===//
// Reference: https://facebook.github.io/graphql/June2018/#sec-String

// stringCoercer implements input coercion and result coercion for String type.
type stringCoercer struct {
	scalarCoercerBase
}

var (
	_ ScalarResultCoercer = (*stringCoercer)(nil)
	_ ScalarInputCoercer  = (*stringCoercer)(nil)
)

func (coercer *stringCoercer) init() {
	coercer.scalarCoercerBase.init("String", coercer)
}

// CoerceBool overrides typeutil.CoercionHelperBase.
func (coercer *stringCoercer) CoerceBool(value bool, ctx *typeutil.CoercionContext) (interface{}, error) {
	// Input mode only accepts string values. See "Input Coercion" in [0].
	//
	// [0]: https://facebook.github.io/graphql/June2018/#sec-String
	if ctx.Mode == typeutil.InputCoercionMode {
		return nil, coercer.RaiseInvalidTypeError(value, ctx)
	}
	if value {
		return "true", nil
	}
	return "false", nil
}

// CoerceSignedInteger overrides typeutil.CoercionHelperBase.
func (coercer *stringCoercer) CoerceSignedInteger(value int64, ctx *typeutil.CoercionContext) (interface{}, error) {
	// Input mode only accepts string values. See "Input Coercion" in [0].
	//
	// [0]: https://facebook.github.io/graphql/June2018/#sec-String
	if ctx.Mode == typeutil.InputCoercionMode {
		return nil, coercer.RaiseInvalidTypeError(value, ctx)
	}
	return strconv.FormatInt(value, 10), nil
}

// CoerceUnsignedInteger overrides typeutil.CoercionHelperBase.
func (coercer *stringCoercer) CoerceUnsignedInteger(value uint64, ctx *typeutil.CoercionContext) (interface{}, error) {
	// Input mode only accepts string values. See "Input Coercion" in [0].
	//
	// [0]: https://facebook.github.io/graphql/June2018/#sec-String
	if ctx.Mode == typeutil.InputCoercionMode {
		return nil, coercer.RaiseInvalidTypeError(value, ctx)
	}
	return strconv.FormatUint(value, 10), nil
}

// CoerceFloat overrides typeutil.CoercionHelperBase.
func (coercer *stringCoercer) CoerceFloat(value float64, ctx *typeutil.CoercionContext) (interface{}, error) {
	// Input mode only accepts string values. See "Input Coercion" in [0].
	//
	// [0]: https://facebook.github.io/graphql/June2018/#sec-String
	if ctx.Mode == typeutil.InputCoercionMode {
		return nil, coercer.RaiseInvalidTypeError(value, ctx)
	}
	return fmt.Sprintf("%v", value), nil
}

// CoerceString overrides typeutil.CoercionHelperBase.
func (coercer *stringCoercer) CoerceString(value string, ctx *typeutil.CoercionContext) (interface{}, error) {
	return value, nil
}

// CoerceResultValue implements ScalarResultCoercer.
func (coercer *stringCoercer) CoerceResultValue(value interface{}) (interface{}, error) {
	return coercer.Coerce(value, typeutil.CoercionContext{
		Mode: typeutil.ResultCoercionMode,
	})
}

// CoerceVariableValue implements ScalarInputCoercer.
func (coercer *stringCoercer) CoerceVariableValue(value interface{}) (interface{}, error) {
	return coercer.Coerce(value, typeutil.CoercionContext{
		Mode: typeutil.InputCoercionMode,
	})
}

// CoerceLiteralValue implements ScalarInputCoercer.
func (coercer *stringCoercer) CoerceLiteralValue(value ast.Value) (interface{}, error) {
	if value, ok := value.(ast.StringValue); ok {
		return value.Value(), nil
	}

	return nil, coercer.RaiseInvalidArgumentTypeError(value)
}

// stringType implements builtin String type in GraphQL.
type stringType struct {
	ThisIsScalarType
	coercer stringCoercer
}

// Name implements TypeWithName.
func (s *stringType) Name() string {
	return "String"
}

// Description implements TypeWithDescription.
func (s *stringType) Description() string {
	return "The `String` scalar type represents textual data, represented as UTF-8 character " +
		"sequences. The String type is most often used by GraphQL to represent free-form human-" +
		"readable text."
}

// String implements fmt.Stringer.
func (s *stringType) String() string {
	return s.Name()
}

// CoerceResultValue implmenets LeafType.
func (s *stringType) CoerceResultValue(value interface{}) (interface{}, error) {
	return s.coercer.CoerceResultValue(value)
}

// CoerceVariableValue implmenets Scalar.
func (s *stringType) CoerceVariableValue(value interface{}) (interface{}, error) {
	return s.coercer.CoerceVariableValue(value)
}

// CoerceLiteralValue implmenets Scalar.
func (s *stringType) CoerceLiteralValue(value ast.Value) (interface{}, error) {
	return s.coercer.CoerceLiteralValue(value)
}

var stringTypeInstance = func() Scalar {
	s := &stringType{}
	// Initialize coercer.
	s.coercer.init()
	return s
}()

// String returns the GraphQL builtin String type definition.
func String() Scalar {
	return stringTypeInstance
}

//===-----------------------------------------------------------------------------------------===//
// Boolean
//===-----------------------------------------------------------------------------------------===//
// Reference: https://facebook.github.io/graphql/June2018/#sec-Boolean

type booleanCoercer struct {
	scalarCoercerBase
}

// booleanCoercer implements input coercion and result coercion for Boolean type.
var (
	_ ScalarResultCoercer = (*booleanCoercer)(nil)
	_ ScalarInputCoercer  = (*booleanCoercer)(nil)
)

func (coercer *booleanCoercer) init() {
	coercer.scalarCoercerBase.init("Boolean", coercer)
}

// RaiseNonValue implements typeutil.CoercionHelper.
func (coercer *booleanCoercer) RaiseNonValue(value interface{}, ctx *typeutil.CoercionContext) error {
	// Use coercionErrorNonInteger for non-value.
	return coercer.RaiseError(value, ctx, coercionErrorNonBoolean)
}

// CoerceBool overrides typeutil.CoercionHelperBase.
func (coercer *booleanCoercer) CoerceBool(value bool, ctx *typeutil.CoercionContext) (interface{}, error) {
	return value, nil
}

// CoerceSignedInteger overrides typeutil.CoercionHelperBase.
func (coercer *booleanCoercer) CoerceSignedInteger(value int64, ctx *typeutil.CoercionContext) (interface{}, error) {
	// Input mode only accepts boolean values. See "Input Coercion" in [0].
	//
	// [0]: https://facebook.github.io/graphql/June2018/#sec-Boolean
	if ctx.Mode == typeutil.InputCoercionMode {
		return nil, coercer.RaiseInvalidTypeError(value, ctx)
	}
	return value != 0, nil
}

// CoerceUnsignedInteger overrides typeutil.CoercionHelperBase.
func (coercer *booleanCoercer) CoerceUnsignedInteger(value uint64, ctx *typeutil.CoercionContext) (interface{}, error) {
	// Input mode only accepts boolean values. See "Input Coercion" in [0].
	//
	// [0]: https://facebook.github.io/graphql/June2018/#sec-Boolean
	if ctx.Mode == typeutil.InputCoercionMode {
		return nil, coercer.RaiseInvalidTypeError(value, ctx)
	}
	return value != 0, nil
}

// graphql-ruby only considers boolean as valid type. graphql-js considers both numeric and boolean
// but not string. We graphql-go accepts the most types including string. It converts "false" and ""
// to false and otherwise to "true". We follow graphql-js.

// CoerceResultValue implements ScalarResultCoercer.
func (coercer *booleanCoercer) CoerceResultValue(value interface{}) (interface{}, error) {
	return coercer.Coerce(value, typeutil.CoercionContext{
		Mode: typeutil.ResultCoercionMode,
	})
}

// CoerceVariableValue implements ScalarInputCoercer.
func (coercer *booleanCoercer) CoerceVariableValue(value interface{}) (interface{}, error) {
	return coercer.Coerce(value, typeutil.CoercionContext{
		Mode: typeutil.InputCoercionMode,
	})
}

// CoerceLiteralValue implements ScalarInputCoercer.
func (coercer *booleanCoercer) CoerceLiteralValue(value ast.Value) (interface{}, error) {
	// Only boolean is accepted as per spec..
	if value, ok := value.(ast.BooleanValue); ok {
		return value.Value(), nil
	}

	return nil, coercer.RaiseInvalidArgumentTypeError(value)
}

// booleanType implements builtin Boolean type in GraphQL.
type booleanType struct {
	ThisIsScalarType
	coercer booleanCoercer
}

// Name implements TypeWithName.
func (b *booleanType) Name() string {
	return "Boolean"
}

// Description implements TypeWithDescription.
func (b *booleanType) Description() string {
	return "The `Boolean` scalar type represents `true` or `false`."
}

// String implements fmt.Stringer.
func (b *booleanType) String() string {
	return b.Name()
}

// CoerceResultValue implmenets LeafType.
func (b *booleanType) CoerceResultValue(value interface{}) (interface{}, error) {
	return b.coercer.CoerceResultValue(value)
}

// CoerceVariableValue implmenets Scalar.
func (b *booleanType) CoerceVariableValue(value interface{}) (interface{}, error) {
	return b.coercer.CoerceVariableValue(value)
}

// CoerceLiteralValue implmenets Scalar.
func (b *booleanType) CoerceLiteralValue(value ast.Value) (interface{}, error) {
	return b.coercer.CoerceLiteralValue(value)
}

var booleanTypeInstance = func() Scalar {
	b := &booleanType{}
	// Initialize coercer.
	b.coercer.init()
	return b
}()

// Boolean returns the GraphQL builtin Boolean type definition.
func Boolean() Scalar {
	return booleanTypeInstance
}

//===-----------------------------------------------------------------------------------------===//
// ID
//===-----------------------------------------------------------------------------------------===//
// Reference: https://facebook.github.io/graphql/June2018/#sec-ID

type idCoercer struct {
	scalarCoercerBase
}

// idCoercer implements input coercion and result coercion for ID type.
var (
	_ ScalarResultCoercer = (*idCoercer)(nil)
	_ ScalarInputCoercer  = (*idCoercer)(nil)
)

func (coercer *idCoercer) init() {
	coercer.scalarCoercerBase.init("ID", coercer)
}

// CoerceSignedInteger overrides typeutil.CoercionHelperBase.
func (coercer *idCoercer) CoerceSignedInteger(value int64, ctx *typeutil.CoercionContext) (interface{}, error) {
	return strconv.FormatInt(value, 10), nil
}

// CoerceUnsignedInteger overrides typeutil.CoercionHelperBase.
func (coercer *idCoercer) CoerceUnsignedInteger(value uint64, ctx *typeutil.CoercionContext) (interface{}, error) {
	return strconv.FormatUint(value, 10), nil
}

// CoerceString overrides typeutil.CoercionHelperBase.
func (coercer *idCoercer) CoerceString(value string, ctx *typeutil.CoercionContext) (interface{}, error) {
	return value, nil
}

// CoerceResultValue implements ScalarResultCoercer.
func (coercer *idCoercer) CoerceResultValue(value interface{}) (interface{}, error) {
	return coercer.Coerce(value, typeutil.CoercionContext{
		Mode: typeutil.ResultCoercionMode,
	})
}

// CoerceVariableValue implements ScalarInputCoercer.
func (coercer *idCoercer) CoerceVariableValue(value interface{}) (interface{}, error) {
	return coercer.Coerce(value, typeutil.CoercionContext{
		Mode: typeutil.InputCoercionMode,
	})
}

// CoerceLiteralValue implements ScalarInputCoercer.
func (coercer *idCoercer) CoerceLiteralValue(value ast.Value) (interface{}, error) {
	// CoerceLiteralValue implements ScalarInputCoercerInputParser.
	switch value := value.(type) {
	case ast.StringValue:
		return value.Value(), nil

	case ast.IntValue:
		return value.String(), nil
	}
	return nil, coercer.RaiseInvalidArgumentTypeError(value)
}

// idType implements builtin ID type in GraphQL.
type idType struct {
	ThisIsScalarType
	coercer idCoercer
}

// Name implements TypeWithName.
func (id *idType) Name() string {
	return "ID"
}

// Description implements TypeWithDescription.
func (id *idType) Description() string {
	return "The `ID` scalar type represents a unique identifier, often used to " +
		"refetch an object or as key for a cache. The ID type appears in a JSON " +
		"response as a String; however, it is not intended to be human-readable. " +
		"When expected as an input type, any string (such as `\"4\"`) or integer " +
		"(such as `4`) input value will be accepted as an ID."
}

// String implements fmt.Stringer.
func (id *idType) String() string {
	return id.Name()
}

// CoerceResultValue implmenets LeafType.
func (id *idType) CoerceResultValue(value interface{}) (interface{}, error) {
	return id.coercer.CoerceResultValue(value)
}

// CoerceVariableValue implmenets Scalar.
func (id *idType) CoerceVariableValue(value interface{}) (interface{}, error) {
	return id.coercer.CoerceVariableValue(value)
}

// CoerceLiteralValue implmenets Scalar.
func (id *idType) CoerceLiteralValue(value ast.Value) (interface{}, error) {
	return id.coercer.CoerceLiteralValue(value)
}

var idTypeInstance = func() Scalar {
	id := &idType{}
	// Initialize coercer.
	id.coercer.init()
	return id
}()

// ID returns the GraphQL builtin ID type definition.
func ID() Scalar {
	return idTypeInstance
}
