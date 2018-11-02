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

// This files implements 3 directives required by specification.
//
// Reference: https://facebook.github.io/graphql/June2018/#sec-Type-System.Directives

//===----------------------------------------------------------------------------------------====//
// @skip
//===----------------------------------------------------------------------------------------====//
// The @skip directive may be provided for fields, fragment spreads, and inline fragments, and
// allows for conditional exclusion during execution as described by the if argument.

var skipDirective = MustNewDirective(&DirectiveConfig{
	Name: "skip",
	Description: "Directs the executor to skip this field or fragment when the `if` " +
		"argument is true.",
	Locations: []DirectiveLocation{
		DirectiveLocationField,
		DirectiveLocationFragmentSpread,
		DirectiveLocationInlineFragment,
	},
	Args: ArgumentConfigMap{
		"if": {
			Type:        NonNullOfType(Boolean()),
			Description: "Skipped when true.",
		},
	},
})

// SkipDirective returns directive definition for @skip.
func SkipDirective() *Directive {
	return skipDirective
}

//===----------------------------------------------------------------------------------------====//
// @include
//===----------------------------------------------------------------------------------------====//
// The @include directive may be provided for fields, fragment spreads, and inline fragments, and
// allows for conditional inclusion during execution as described by the if argument.
//
// Reference: https://facebook.github.io/graphql/June2018/#sec--include

// IncludeDirective is used to conditionally include fields or fragments.
var includeDirective = MustNewDirective(&DirectiveConfig{
	Name: "include",
	Description: "Directs the executor to include this field or fragment only when " +
		"the `if` argument is true.",
	Locations: []DirectiveLocation{
		DirectiveLocationField,
		DirectiveLocationFragmentSpread,
		DirectiveLocationInlineFragment,
	},
	Args: ArgumentConfigMap{
		"if": {
			Type:        NonNullOfType(Boolean()),
			Description: "Included when true.",
		},
	},
})

// IncludeDirective returns directive definition for @include.
func IncludeDirective() *Directive {
	return includeDirective
}

//===----------------------------------------------------------------------------------------====//
// @deprecated
//===----------------------------------------------------------------------------------------====//
// The @deprecated directive is used within the type system definition language to indicate
// deprecated portions of a GraphQL service’s schema, such as deprecated fields on a type or
// deprecated enum values.
//
// Reference: https://facebook.github.io/graphql/June2018/#sec--deprecated

// DefaultDeprecationReason is a constant string used for default reason for a deprecation.
const DefaultDeprecationReason = "No longer supported"

// DeprecatedDirective  Used to declare element of a GraphQL schema as deprecated.
var deprecatedDirective = MustNewDirective(&DirectiveConfig{
	Name:        "deprecated",
	Description: "Marks an element of a GraphQL schema as no longer supported.",
	Locations: []DirectiveLocation{
		DirectiveLocationFieldDefinition,
		DirectiveLocationEnumValue,
	},
	Args: ArgumentConfigMap{
		"reason": {
			Type: T(String()),
			Description: "Explains why this element was deprecated, usually also including a " +
				"suggestion for how to access supported similar data. Formatted" +
				"in [Markdown](https://daringfireball.net/projects/markdown/).",
			DefaultValue: DefaultDeprecationReason,
		},
	},
})

// DeprecatedDirective returns directive definition for @deprecated.
func DeprecatedDirective() *Directive {
	return deprecatedDirective
}

// StandardDirectives returns list of directives that should be included in a standard GraphQL as
// per specification.
//
// Reference: https://facebook.github.io/graphql/June2018/#sec-Type-System.Directives
func StandardDirectives() []*Directive {
	return []*Directive{
		// @skip
		SkipDirective(),
		// @include
		IncludeDirective(),
		// @deprecated
		DeprecatedDirective(),
	}
}
