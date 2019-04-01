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
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/internal/util"
)

// DefaultFieldResolverOption specifies an option to configure field resolver instance created by
// NewDefaultFieldResolver.
type DefaultFieldResolverOption func(*defaultFieldResolver)

// defaultFieldResolver is used when no resolve function is given to a field. It resolves the field
// value to the value of field in source object value whose name, or if it's a function,
// returns the result of calling that function while passing along args and context value.
type defaultFieldResolver struct {
	UnresolvedAsError   bool   // default: true
	ScanAnonymousFields bool   // default: true
	ScanMethods         bool   // default: true
	FieldTagName        string // default: "graphql"
}

// NewDefaultFieldResolver configures a field resolver which is useful as "default" resolver for
// fields without resolve function. A default field resolver can be provided to a prepare operation
// via DefaultFieldResolver option which will be used when encountering a field without custom
// resolver during execution.
//
// When source value is an object, the created resolver takes the value from the field with the
// matching name in the object as the field result. Additional capabilities can be enabled via
// various options.
func NewDefaultFieldResolver(opts ...DefaultFieldResolverOption) graphql.FieldResolver {
	resolver := &defaultFieldResolver{
		UnresolvedAsError:   true,
		ScanAnonymousFields: true,
		ScanMethods:         true,
		FieldTagName:        "graphql",
	}

	// Configure resolver with options.
	for _, opt := range opts {
		opt(resolver)
	}

	return resolver
}

// UnresolvedAsError specifies whether error should be returned for fields that cannot be
// successfully resolved by the resolver. The feature is enabled by default.
// UnresolvedAsError(false) has to be explicitly specified to disable the feature.
func UnresolvedAsError(enabled bool) DefaultFieldResolverOption {
	return func(resolver *defaultFieldResolver) {
		resolver.UnresolvedAsError = enabled
	}
}

// ScanAnonymousFields specifies whether anonymous fields contained the object should be inspected
// further to find matching field. The feature is enabled by default. Therefore,
// ScanAnonymousFields(false) has to be explicitly specified to disable the feature.
func ScanAnonymousFields(enabled bool) DefaultFieldResolverOption {
	return func(resolver *defaultFieldResolver) {
		resolver.ScanAnonymousFields = enabled
	}
}

// ScanMethods specifies whether public methods exposed by the source object value should also be
// taken into consideration to search for field value. If enabled, it matches method named with the
// field name in camel case. For example, "FooBar()" will be used for field named "foo_bar". The
// matching method will be invoked with the context and ResolveInfo and the return value is used as
// field result. The feature is enabled by default. Therefore, ScanMethods(false) has to be
// explicitly specified to disable the feature.
func ScanMethods(enabled bool) DefaultFieldResolverOption {
	return func(resolver *defaultFieldResolver) {
		resolver.ScanMethods = enabled
	}
}

// FieldTagName specifies the struct field tag that is used to specify custom name in source object
// field for matching targeting field. For example,
//
//	type Foo struct {
//		Bar string `graphql:"baz"`
//	}
//
// value in field Bar could be returned as result for fields named "bar" and "baz".
//
// The feature is enabled by default and can be disabled by FieldTagName("").
func FieldTagName(name string) DefaultFieldResolverOption {
	return func(resolver *defaultFieldResolver) {
		resolver.FieldTagName = name
	}
}

// Resolve implements graphql.FieldResolver.
func (resolver *defaultFieldResolver) Resolve(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
	value := reflect.ValueOf(source)
	if !value.IsValid() {
		return nil, resolver.unresolvedError(info)
	}

	// It source is a pointer, resolve value from what it points to.
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
		if !value.IsValid() {
			return nil, resolver.unresolvedError(info)
		}
	}

	if value.Kind() == reflect.Struct {
		return resolver.resolveFromStruct(ctx, source, value, info)
	}

	if value.Kind() == reflect.Map {
		return resolver.resolveFromMap(ctx, source, value, info)
	}

	return nil, resolver.unresolvedError(info)
}

func (resolver *defaultFieldResolver) unresolvedErrorWithMessage(message string) error {
	if !resolver.UnresolvedAsError {
		return nil
	}

	return graphql.NewError(message)
}

func (resolver *defaultFieldResolver) unresolvedError(info graphql.ResolveInfo) error {
	if !resolver.UnresolvedAsError {
		return nil
	}

	return graphql.NewError(fmt.Sprintf(`default resolver cannot resolve value for "%s.%s"`,
		info.Object().Name(), info.Field().Name()))
}

func (resolver *defaultFieldResolver) resolveFromFunc(
	ctx context.Context,
	source interface{},
	methodName string,
	f interface{},
	info graphql.ResolveInfo) (interface{}, error) {

	switch f := f.(type) {
	case func(ctx context.Context) (interface{}, error):
		return f(ctx)

	case func(ctx context.Context, source interface{}) (interface{}, error):
		return f(ctx, source)

	case func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error):
		return f(ctx, source, info)

	default:
		return nil, resolver.unresolvedErrorWithMessage(fmt.Sprintf(
			`default resolver found method %s but is unable to call for resolving %s.%s because of `+
				`unexpected type. Must be one of:
	func(ctx context.Context) (interface{}, error)
	func(ctx context.Context, source interface{}) (interface{}, error)
	func(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error), but got
	%T`, methodName, info.Object().Name(), info.Field().Name(), f))
	}
}

func (resolver *defaultFieldResolver) resolveFromValueOrFunc(
	ctx context.Context,
	source interface{},
	valueName string,
	value reflect.Value,
	info graphql.ResolveInfo) (interface{}, error) {

	// value could be a function.
	if value.Kind() == reflect.Func {
		return resolver.resolveFromFunc(ctx, source, valueName, value.Interface(), info)
	}
	return value.Interface(), nil
}

func (resolver *defaultFieldResolver) resolveFromStruct(
	ctx context.Context,
	source interface{},
	sourceValue reflect.Value,
	info graphql.ResolveInfo) (interface{}, error) {

	targetFieldName := info.Field().Name()
	camelTargetFieldName := util.CamelCase(targetFieldName)
	queue := []reflect.Value{sourceValue}
	tagName := resolver.FieldTagName

	for len(queue) > 0 {
		source := queue[0]
		queue = queue[1:]

		sourceType := source.Type()
		numFields := source.NumField()
		for i := 0; i < numFields; i++ {
			field := sourceType.Field(i)

			// Handle anonymous contained structs.
			if resolver.ScanAnonymousFields &&
				field.Anonymous &&
				field.Type.Kind() == reflect.Struct {
				queue = append(queue, source.Field(i))
				continue
			}

			// Match name specified by tag.
			if len(tagName) > 0 {
				tagOptions := strings.Split(field.Tag.Get(tagName), ",")
				if len(tagOptions) > 0 && tagOptions[0] == targetFieldName {
					return resolver.resolveFromValueOrFunc(
						ctx, source, fmt.Sprintf("%s.%s", sourceType.Name(), field.Name), source.Field(i), info)
				}
			}
		}

		// Try finding the field that matches field name in CamelCase.
		fieldValue := source.FieldByName(camelTargetFieldName)
		if fieldValue.IsValid() {
			return resolver.resolveFromValueOrFunc(
				ctx, source, fmt.Sprintf("%s.%s", sourceType.Name(), camelTargetFieldName), fieldValue, info)
		}
	}

	// Try finding the method that matches field name in CamelCase. Note that this is not in the loop.
	if resolver.ScanMethods {
		if sourceValue.CanAddr() {
			sourceValue = sourceValue.Addr()
		}

		method := sourceValue.MethodByName(camelTargetFieldName)
		if method.IsValid() {
			return resolver.resolveFromFunc(
				ctx, source, fmt.Sprintf("%s.%s", sourceValue.Type().Name(), camelTargetFieldName),
				method.Interface(), info)
		}
	}

	return nil, resolver.unresolvedError(info)
}

func (resolver *defaultFieldResolver) resolveFromMap(
	ctx context.Context,
	source interface{},
	sourceValue reflect.Value,
	info graphql.ResolveInfo) (interface{}, error) {

	fieldName := info.Field().Name()
	value := sourceValue.MapIndex(reflect.ValueOf(fieldName))
	if value.IsValid() {
		return resolver.resolveFromValueOrFunc(ctx, source, fmt.Sprintf("map[%s]", fieldName), value, info)
	}
	return nil, resolver.unresolvedError(info)
}
