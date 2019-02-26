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
)

// DirectiveLocation specifies a valid location for a directive to be used.
type DirectiveLocation string

// Reference: https://facebook.github.io/graphql/June2018/#DirectiveLocations
const (
	// Executable directive location
	DirectiveLocationQuery              DirectiveLocation = "QUERY"
	DirectiveLocationMutation                             = "MUTATION"
	DirectiveLocationSubscription                         = "SUBSCRIPTION"
	DirectiveLocationField                                = "FIELD"
	DirectiveLocationFragmentDefinition                   = "FRAGMENT_DEFINITION"
	DirectiveLocationFragmentSpread                       = "FRAGMENT_SPREAD"
	DirectiveLocationInlineFragment                       = "INLINE_FRAGMENT"
	DirectiveLocationVariableDefinition                   = "VARIABLE_DEFINITION"

	// Type system directive location
	DirectiveLocationSchema               = "SCHEMA"
	DirectiveLocationScalar               = "SCALAR"
	DirectiveLocationObject               = "OBJECT"
	DirectiveLocationFieldDefinition      = "FIELD_DEFINITION"
	DirectiveLocationArgumentDefinition   = "ARGUMENT_DEFINITION"
	DirectiveLocationInterface            = "INTERFACE"
	DirectiveLocationUnion                = "UNION"
	DirectiveLocationEnum                 = "ENUM"
	DirectiveLocationEnumValue            = "ENUM_VALUE"
	DirectiveLocationInputObject          = "INPUT_OBJECT"
	DirectiveLocationInputFieldDefinition = "INPUT_FIELD_DEFINITION"
)

// DirectiveConfig provides definition for creating a Directive.
type DirectiveConfig struct {
	// Name of the defining Directive
	Name string

	// Description for the Directive type
	Description string

	// Locations in the schema where the defining directive can appear
	Locations []DirectiveLocation

	// Arguments to be provided when using the directive
	Args ArgumentConfigMap
}

// DeepCopy makes a copy of receiver.
func (config *DirectiveConfig) DeepCopy() *DirectiveConfig {
	if config == nil {
		return nil
	}
	out := new(DirectiveConfig)
	*out = *config

	if len(config.Locations) == 0 {
		out.Locations = nil
	} else {
		out.Locations = make([]DirectiveLocation, len(config.Locations))
		copy(out.Locations, config.Locations)
	}
	return out
}

// Directive are used by the GraphQL runtime as a way of modifying a validator, execution or client
// tool behavior.
//
// Reference: https://facebook.github.io/graphql/June2018/#sec-Type-System.Directives
type Directive interface {
	fmt.Stringer

	// Name of the directive
	Name() string

	// Description provides documentation for the directive.
	Description() string

	// Locations specifies the places where the directive must only be used.
	Locations() []DirectiveLocation

	// Args indicates the arguments taken by the directive.
	Args() []Argument
}

// directive provides an implementation to Schema which creates schema from a SchemaConfig.
type directive struct {
	config DirectiveConfig
	args   []Argument
	// notation is cached value for returning from String() and is initialized in constructor.
	notation string
}

var (
	_ Directive = (*directive)(nil)
)

// NewDirective creates a Directive from a DirectiveConfig.
func NewDirective(config *DirectiveConfig) (Directive, error) {
	if len(config.Name) == 0 {
		return nil, NewError("Must provide name for Directive.")
	}

	// Build arguments with NewType as resolver for TypeDefinition.
	args, err := buildArguments(config.Args, typeDefinitionResolver(NewType))
	if err != nil {
		return nil, err
	}

	return &directive{
		config:   *config.DeepCopy(),
		args:     args,
		notation: fmt.Sprintf("@%s", config.Name),
	}, nil
}

// MustNewDirective is a convenience function equivalent to NewDirective but panics on failure
// instead of returning an error.
func MustNewDirective(config *DirectiveConfig) Directive {
	directive, err := NewDirective(config)
	if err != nil {
		panic(err)
	}
	return directive
}

// Name implements Directive.
func (d *directive) Name() string {
	return d.config.Name
}

// Description implements Directive.
func (d *directive) Description() string {
	return d.config.Description
}

// Locations implements Directive.
func (d *directive) Locations() []DirectiveLocation {
	return d.config.Locations
}

// Args implements Directive.
func (d *directive) Args() []Argument {
	return d.args
}

// String implemennts fmt.Stringer.
func (d *directive) String() string {
	return d.notation
}
