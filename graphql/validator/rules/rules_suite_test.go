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

package rules_test

import (
	"fmt"
	"testing"

	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/graphql/ast"
	"github.com/botobag/artemis/graphql/parser"
	"github.com/botobag/artemis/graphql/token"
	"github.com/botobag/artemis/graphql/validator"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestGraphQLValidatorRules(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "GraphQL Validator Rules Suite")
}

// graphql-js/src/validation/__tests__/harness.js@8c96dc8
var Being = &graphql.InterfaceConfig{
	Name: "Being",
	Fields: graphql.Fields{
		"name": {
			Type: graphql.T(graphql.String()),
			Args: graphql.ArgumentConfigMap{
				"surname": {
					Type: graphql.T(graphql.Boolean()),
				},
			},
		},
	},
}

var Pet = &graphql.InterfaceConfig{
	Name: "Pet",
	Fields: graphql.Fields{
		"name": {
			Type: graphql.T(graphql.String()),
			Args: graphql.ArgumentConfigMap{
				"surname": {
					Type: graphql.T(graphql.Boolean()),
				},
			},
		},
	},
}

var Canine = &graphql.InterfaceConfig{
	Name: "Canine",
	Fields: graphql.Fields{
		"name": {
			Type: graphql.T(graphql.String()),
			Args: graphql.ArgumentConfigMap{
				"surname": {
					Type: graphql.T(graphql.Boolean()),
				},
			},
		},
	},
}

var DogCommand = &graphql.EnumConfig{
	Name: "DogCommand",
	Values: graphql.EnumValueDefinitionMap{
		"SIT": {
			Value: 0,
		},
		"HEEL": {
			Value: 1,
		},
		"DOWN": {
			Value: 2,
		},
	},
}

var Dog = &graphql.ObjectConfig{
	Name: "Dog",
	Fields: graphql.Fields{
		"name": {
			Type: graphql.T(graphql.String()),
			Args: graphql.ArgumentConfigMap{
				"surname": {
					Type: graphql.T(graphql.Boolean()),
				},
			},
		},
		"nickname": {
			Type: graphql.T(graphql.String()),
		},
		"barkVolume": {
			Type: graphql.T(graphql.Int()),
		},
		"barks": {
			Type: graphql.T(graphql.Boolean()),
		},
		"doesKnowCommand": {
			Type: graphql.T(graphql.Boolean()),
			Args: graphql.ArgumentConfigMap{
				"dogCommand": {
					Type: DogCommand,
				},
			},
		},
		"isHousetrained": {
			Type: graphql.T(graphql.Boolean()),
			Args: graphql.ArgumentConfigMap{
				"atOtherHomes": {
					Type:         graphql.T(graphql.Boolean()),
					DefaultValue: true,
				},
			},
		},
		"isAtLocation": {
			Type: graphql.T(graphql.Boolean()),
			Args: graphql.ArgumentConfigMap{
				"x": {
					Type: graphql.T(graphql.Int()),
				},
				"y": {
					Type: graphql.T(graphql.Int()),
				},
			},
		},
	},
	Interfaces: []graphql.InterfaceTypeDefinition{
		Being,
		Pet,
		Canine,
	},
}

var FurColor = &graphql.EnumConfig{
	Name: "FurColor",
	Values: graphql.EnumValueDefinitionMap{
		"BROWN": {
			Value: 0,
		},
		"BLACK": {
			Value: 1,
		},
		"TAN": {
			Value: 2,
		},
		"SPOTTED": {
			Value: 3,
		},
		"NO_FUR": {
			Value: graphql.NilEnumInternalValue,
		},
		"UNKNOWN": {
			Value: nil,
		},
	},
}

var Cat = &graphql.ObjectConfig{
	Name: "Cat",
	Fields: graphql.Fields{
		"name": {
			Type: graphql.T(graphql.String()),
			Args: graphql.ArgumentConfigMap{
				"surname": {
					Type: graphql.T(graphql.Boolean()),
				},
			},
		},
		"nickname": {
			Type: graphql.T(graphql.String()),
		},
		"meows": {
			Type: graphql.T(graphql.Boolean()),
		},
		"meowVolume": {
			Type: graphql.T(graphql.Int()),
		},
		"furColor": {
			Type: FurColor,
		},
	},
	Interfaces: []graphql.InterfaceTypeDefinition{
		Being,
		Pet,
	},
}

var CatOrDog = &graphql.UnionConfig{
	Name: "CatOrDog",
	PossibleTypes: []graphql.ObjectTypeDefinition{
		Cat,
		Dog,
	},
}

var Intelligent = &graphql.InterfaceConfig{
	Name: "Intelligent",
	Fields: graphql.Fields{
		"iq": {
			Type: graphql.T(graphql.Int()),
		},
	},
}

var Human = &graphql.ObjectConfig{
	Name: "Human",
	Interfaces: []graphql.InterfaceTypeDefinition{
		Being,
		Intelligent,
	},
}

var Alien = &graphql.ObjectConfig{
	Name: "Alien",
	Interfaces: []graphql.InterfaceTypeDefinition{
		Being,
		Intelligent,
	},
	Fields: graphql.Fields{
		"iq": {
			Type: graphql.T(graphql.Int()),
		},
		"name": {
			Type: graphql.T(graphql.String()),
			Args: graphql.ArgumentConfigMap{
				"surname": {
					Type: graphql.T(graphql.Boolean()),
				},
			},
		},
		"numEyes": {
			Type: graphql.T(graphql.Int()),
		},
	},
}

var DogOrHuman = &graphql.UnionConfig{
	Name: "DogOrHuman",
	PossibleTypes: []graphql.ObjectTypeDefinition{
		Dog,
		Human,
	},
}

var HumanOrAlien = &graphql.UnionConfig{
	Name: "HumanOrAlien",
	PossibleTypes: []graphql.ObjectTypeDefinition{
		Human,
		Alien,
	},
}

var ComplexInput = &graphql.InputObjectConfig{
	Name: "ComplexInput",
	Fields: graphql.InputFields{
		"requiredField": {
			Type: graphql.NonNullOfType(graphql.Boolean()),
		},
		"nonNullField": {
			Type:         graphql.NonNullOfType(graphql.Boolean()),
			DefaultValue: false,
		},
		"intField": {
			Type: graphql.T(graphql.Int()),
		},
		"stringField": {
			Type: graphql.T(graphql.String()),
		},
		"booleanField": {
			Type: graphql.T(graphql.Boolean()),
		},
		"stringListField": {
			Type: graphql.ListOfType(graphql.String()),
		},
	},
}

var ComplicatedArgs = &graphql.ObjectConfig{
	Name: "ComplicatedArgs",
	Fields: graphql.Fields{
		"intArgField": {
			Type: graphql.T(graphql.String()),
			Args: graphql.ArgumentConfigMap{
				"intArg": {
					Type: graphql.T(graphql.Int()),
				},
			},
		},
		"nonNullIntArgField": {
			Type: graphql.T(graphql.String()),
			Args: graphql.ArgumentConfigMap{
				"nonNullIntArg": {
					Type: graphql.NonNullOfType(graphql.Int()),
				},
			},
		},
		"stringArgField": {
			Type: graphql.T(graphql.String()),
			Args: graphql.ArgumentConfigMap{
				"stringArg": {
					Type: graphql.T(graphql.String()),
				},
			},
		},
		"booleanArgField": {
			Type: graphql.T(graphql.String()),
			Args: graphql.ArgumentConfigMap{
				"booleanArg": {
					Type: graphql.T(graphql.Boolean()),
				},
			},
		},
		"enumArgField": {
			Type: graphql.T(graphql.String()),
			Args: graphql.ArgumentConfigMap{
				"enumArg": {
					Type: FurColor,
				},
			},
		},
		"floatArgField": {
			Type: graphql.T(graphql.String()),
			Args: graphql.ArgumentConfigMap{
				"floatArg": {
					Type: graphql.T(graphql.Float()),
				},
			},
		},
		"idArgField": {
			Type: graphql.T(graphql.String()),
			Args: graphql.ArgumentConfigMap{
				"idArg": {
					Type: graphql.T(graphql.ID()),
				},
			},
		},
		"stringListArgField": {
			Type: graphql.T(graphql.String()),
			Args: graphql.ArgumentConfigMap{
				"stringListArg": {
					Type: graphql.ListOfType(graphql.String()),
				},
			},
		},
		"stringListNonNullArgField": {
			Type: graphql.T(graphql.String()),
			Args: graphql.ArgumentConfigMap{
				"stringListNonNullArg": {
					Type: graphql.ListOf(graphql.NonNullOfType(graphql.String())),
				},
			},
		},
		"complexArgField": {
			Type: graphql.T(graphql.String()),
			Args: graphql.ArgumentConfigMap{
				"complexArg": {
					Type: ComplexInput,
				},
			},
		},
		"multipleReqs": {
			Type: graphql.T(graphql.String()),
			Args: graphql.ArgumentConfigMap{
				"req1": {
					Type: graphql.NonNullOfType(graphql.Int()),
				},
				"req2": {
					Type: graphql.NonNullOfType(graphql.Int()),
				},
			},
		},
		"nonNullFieldWithDefault": {
			Type: graphql.T(graphql.String()),
			Args: graphql.ArgumentConfigMap{
				"arg": {
					Type:         graphql.NonNullOfType(graphql.Int()),
					DefaultValue: 0,
				},
			},
		},
		"multipleOpts": {
			Type: graphql.T(graphql.String()),
			Args: graphql.ArgumentConfigMap{
				"opt1": {
					Type:         graphql.T(graphql.Int()),
					DefaultValue: 0,
				},
				"opt2": {
					Type:         graphql.T(graphql.Int()),
					DefaultValue: 0,
				},
			},
		},
		"multipleOptAndReq": {
			Type: graphql.T(graphql.String()),
			Args: graphql.ArgumentConfigMap{
				"req1": {
					Type: graphql.NonNullOfType(graphql.Int()),
				},
				"req2": {
					Type: graphql.NonNullOfType(graphql.Int()),
				},
				"opt1": {
					Type:         graphql.T(graphql.Int()),
					DefaultValue: 0,
				},
				"opt2": {
					Type:         graphql.T(graphql.Int()),
					DefaultValue: 0,
				},
			},
		},
	},
}

var InvalidScalar = &graphql.ScalarConfig{
	Name: "Invalid",
	ResultCoercer: graphql.ScalarResultCoercerFunc(func(value interface{}) (interface{}, error) {
		return value, nil
	}),
	InputCoercer: graphql.ScalarInputCoercerFuncs{
		CoerceVariableValueFunc: func(value interface{}) (interface{}, error) {
			return nil, fmt.Errorf("Invalid scalar is always invalid: %s", graphql.Inspect(value))
		},
		CoerceLiteralValueFunc: func(value ast.Value) (interface{}, error) {
			return nil, fmt.Errorf("Invalid scalar is always invalid: %s", ast.Print(value))
		},
	},
}

var AnyScalar = &graphql.ScalarConfig{
	Name: "Any",
	ResultCoercer: graphql.ScalarResultCoercerFunc(func(value interface{}) (interface{}, error) {
		return value, nil
	}),
	InputCoercer: graphql.ScalarInputCoercerFuncs{
		CoerceVariableValueFunc: func(value interface{}) (interface{}, error) {
			// Allow any value.
			return value, nil
		},
		CoerceLiteralValueFunc: func(value ast.Value) (interface{}, error) {
			// Allow any value.
			return value, nil
		},
	},
}

var QueryRoot = &graphql.ObjectConfig{
	Name: "QueryRoot",
	Fields: graphql.Fields{
		"human": {
			Type: Human,
			Args: graphql.ArgumentConfigMap{
				"id": {
					Type: graphql.T(graphql.ID()),
				},
			},
		},
		"alien": {
			Type: Alien,
		},
		"dog": {
			Type: Dog,
		},
		"cat": {
			Type: Cat,
		},
		"pet": {
			Type: Pet,
		},
		"catOrDog": {
			Type: CatOrDog,
		},
		"dogOrHuman": {
			Type: DogOrHuman,
		},
		"humanOrAlien": {
			Type: HumanOrAlien,
		},
		"complicatedArgs": {
			Type: ComplicatedArgs,
		},
		"invalidArg": {
			Args: graphql.ArgumentConfigMap{
				"arg": {
					Type: InvalidScalar,
				},
			},
			Type: graphql.T(graphql.String()),
		},
		"anyArg": {
			Args: graphql.ArgumentConfigMap{
				"arg": {
					Type: AnyScalar,
				},
			},
			Type: graphql.T(graphql.String()),
		},
	},
}

var testSchema graphql.Schema

func expectValidationErrors(rule interface{}, queryStr string) GomegaAssertion {
	return expectValidationErrorsWithSchema(testSchema, rule, queryStr)
}

func expectValidationErrorsWithSchema(schema graphql.Schema, rule interface{}, queryStr string) GomegaAssertion {
	// Parse queryStr.
	doc := parser.MustParse(token.NewSource(queryStr))
	return Expect(validator.ValidateWithRules(schema, doc, rule))
}

func init() {
	Human.Fields = graphql.Fields{
		"name": {
			Type: graphql.T(graphql.String()),
			Args: graphql.ArgumentConfigMap{
				"surname": {
					Type: graphql.T(graphql.Boolean()),
				},
			},
		},
		"pets": {
			Type: graphql.ListOf(Pet),
		},
		"relatives": {
			Type: graphql.ListOf(Human),
		},
		"iq": {
			Type: graphql.T(graphql.Int()),
		},
	}

	testSchema = graphql.MustNewSchema(&graphql.SchemaConfig{
		Query: graphql.MustNewObject(QueryRoot),
		Types: []graphql.Type{
			graphql.MustNewObject(Cat),
			graphql.MustNewObject(Dog),
			graphql.MustNewObject(Human),
			graphql.MustNewObject(Alien),
		},
		Directives: graphql.DirectiveList{
			graphql.IncludeDirective(),
			graphql.SkipDirective(),
			graphql.MustNewDirective(&graphql.DirectiveConfig{
				Name:      "onQuery",
				Locations: []graphql.DirectiveLocation{graphql.DirectiveLocationQuery},
			}),
			graphql.MustNewDirective(&graphql.DirectiveConfig{
				Name:      "onMutation",
				Locations: []graphql.DirectiveLocation{graphql.DirectiveLocationMutation},
			}),
			graphql.MustNewDirective(&graphql.DirectiveConfig{
				Name:      "onSubscription",
				Locations: []graphql.DirectiveLocation{graphql.DirectiveLocationSubscription},
			}),
			graphql.MustNewDirective(&graphql.DirectiveConfig{
				Name:      "onField",
				Locations: []graphql.DirectiveLocation{graphql.DirectiveLocationField},
			}),
			graphql.MustNewDirective(&graphql.DirectiveConfig{
				Name:      "onFragmentDefinition",
				Locations: []graphql.DirectiveLocation{graphql.DirectiveLocationFragmentDefinition},
			}),
			graphql.MustNewDirective(&graphql.DirectiveConfig{
				Name:      "onFragmentSpread",
				Locations: []graphql.DirectiveLocation{graphql.DirectiveLocationFragmentSpread},
			}),
			graphql.MustNewDirective(&graphql.DirectiveConfig{
				Name:      "onInlineFragment",
				Locations: []graphql.DirectiveLocation{graphql.DirectiveLocationInlineFragment},
			}),
			graphql.MustNewDirective(&graphql.DirectiveConfig{
				Name:      "onVariableDefinition",
				Locations: []graphql.DirectiveLocation{graphql.DirectiveLocationVariableDefinition},
			}),
		},
	})
}
