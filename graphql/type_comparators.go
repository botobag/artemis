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

// IsTypeSubTypeOf returns true if Provided a type and a super type, return true if the first type is either
// equal or a subset of the second super type (covariant).
func IsTypeSubTypeOf(schema Schema, maybeSubType Type, superType Type) bool {
	// Equivalent type is a valid subtype
	if maybeSubType == superType {
		return true
	}

	switch superType := superType.(type) {
	case NonNull:
		// If superType is non-null, maybeSubType must also be non-null.
		if maybeSubType, ok := maybeSubType.(NonNull); ok {
			return IsTypeSubTypeOf(schema, maybeSubType.InnerType(), superType.InnerType())
		}
		return false

	case List:
		// If superType type is a list, maybeSubType type must also be a list.
		if maybeSubType, ok := maybeSubType.(List); ok {
			return IsTypeSubTypeOf(schema, maybeSubType.ElementType(), superType.ElementType())
		}
		return false

	case AbstractType:
		// If superType type is an abstract type, maybeSubType type may be a currently possible object
		// type.
		if maybeSubType, ok := maybeSubType.(Object); ok {
			return schema.PossibleTypes(superType).Contains(maybeSubType)
		}
		return false

	default:
		if maybeSubType, ok := maybeSubType.(NonNull); ok {
			// If superType is nullable, maybeSubType may be non-null or nullable.
			return IsTypeSubTypeOf(schema, maybeSubType.InnerType(), superType)
		}

		// Otherwise, the child type is not a valid subtype of the parent type.
		return false
	}
}
