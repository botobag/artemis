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

// DefaultFieldResolverOpt configures a DefaultFieldResolver instance.
type DefaultFieldResolverOpt func(*DefaultFieldResolver)

// DefaultFieldResolver is used when a resolve function is not given to a field. It takes the
// property of the source object of the same name as the field and returns it as the result, or if
// it's a function, returns the result of calling that function while passing along args and context
// value.
type DefaultFieldResolver struct {
	UnresolvedAsError   bool
	ScanAnonymousFields bool
	ScanMethods         bool
	FieldTagName        string
}

var _ = (*DefaultFieldResolver)(nil)

// Resolve implements graphql.FieldResolver.
func (resolver *DefaultFieldResolver) Resolve(ctx context.Context, source interface{}, info graphql.ResolveInfo) (interface{}, error) {
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

func (resolver *DefaultFieldResolver) unresolvedErrorWithMessage(message string) error {
	if !resolver.UnresolvedAsError {
		return nil
	}

	return graphql.NewError(message)
}

func (resolver *DefaultFieldResolver) unresolvedError(info graphql.ResolveInfo) error {
	if !resolver.UnresolvedAsError {
		return nil
	}

	return graphql.NewError(fmt.Sprintf(`default resolver cannot resolve value for "%s.%s"`,
		info.Object().Name(), info.Field().Name()))
}

func (resolver *DefaultFieldResolver) resolveFromFunc(
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

func (resolver *DefaultFieldResolver) resolveFromValueOrFunc(
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

func (resolver *DefaultFieldResolver) resolveFromStruct(
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

func (resolver *DefaultFieldResolver) resolveFromMap(
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
