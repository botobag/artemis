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

package parser

import (
	"fmt"

	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/graphql/ast"
	"github.com/botobag/artemis/graphql/lexer"
	"github.com/botobag/artemis/graphql/token"
)

// parser holds internal state during parsing.
type parser struct {
	// The lexer for tokenization
	lexer *lexer.Lexer

	// The configuration options
	options ParseOptions
}

func newParser(source *graphql.Source, options ParseOptions) (*parser, error) {
	if source == nil {
		return nil, graphql.NewError("Must provide Source. Received: nil")
	}
	return &parser{
		lexer:   lexer.New(source),
		options: options,
	}, nil
}

// If the next token is of the given kind, return true after advancing the lexer. Otherwise, do not
// change the parser state and return false.
func (p *parser) skip(tokenKind token.Kind) (bool, error) {
	if p.lexer.Token().Kind == tokenKind {
		if _, err := p.lexer.Advance(); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

// If the next token is of the given kind, return that token after advancing the lexer. Otherwise,
// do not change the parser state and throw an error.
func (p *parser) expect(tokenKind token.Kind) (*token.Token, error) {
	token := p.lexer.Token()
	if token.Kind == tokenKind {
		if _, err := p.lexer.Advance(); err != nil {
			return nil, err
		}
		return token, nil
	}
	return nil, graphql.NewSyntaxError(
		p.lexer.Source(),
		token.Location,
		fmt.Sprintf("Expected %v, found %s", tokenKind, token.Description()))
}

// If the next token is a keyword with the given value, return true after advancing
// the lexer. Otherwise, do not change the parser state and return false.
func (p *parser) skipKeyword(keyword string) (bool, error) {
	if tok := p.peek(); tok.Kind == token.KindName && tok.Value == keyword {
		_, err := p.lexer.Advance()
		if err != nil {
			return true, err
		}
		return true, nil
	}
	return false, nil
}

// If the next token is a keyword with the given value, return that token after
// advancing the lexer. Otherwise, do not change the parser state and throw
// an error.
func (p *parser) expectKeyword(keyword string) error {
	hasKeyword, err := p.skipKeyword(keyword)

	if err != nil {
		return err
	} else if !hasKeyword {
		tok := p.peek()
		return graphql.NewSyntaxError(p.lexer.Source(), tok.Location,
			fmt.Sprintf(`Expected "%s", found %s`, keyword, tok.Description()))
	}
	return nil
}

// Peek return current token without consume it.
func (p *parser) peek() *token.Token {
	return p.lexer.Token()
}

// Helper function for creating an error when an unexpected lexed token is encountered.
func (p *parser) unexpected() error {
	token := p.lexer.Token()
	return graphql.NewSyntaxError(
		p.lexer.Source(), token.Location, fmt.Sprintf("Unexpected %s", token.Description()))
}

// parseList returns a non-empty list of parse nodes, determined by the parseFunc. This list begins
// with a lex token of openKind and ends with a lex token of closeKind. Advances the parser to the
// next lex token after the closing token.
//
// The following example rewrites parseSelections with parseList:
//
//		func (p *parser) parseSelections() (ast.SelectionSet, error) {
//			selections, err := p.parseList(token.KindLeftBrace, p.parseSelection, token.KindRightBrace)
//			if err != nil {
//				return nil, err
//			}
//			return ast.SelectionSet(selections.([]ast.Selection)), nil
//		}
//
// It works like a charm but the use of reflection causes performance problem. A microbenchmark in
// parser_test which has 10k field selection in a query shows more than 1x slow compare with the
// "idiomatic" approach.
//
//	"idomatic" approach (current):
//		Parser
//  		parses query with 10k field selection
//
//  		Ran 10 samples:
//  		parse time:
//    		Fastest Time: 0.002s
//    		Slowest Time: 0.004s
//    		Average Time: 0.003s ± 0.001s
//
//	parseList:
//		Parser
//  		parses query with 10k field selection
//
//  		Ran 10 samples:
//  		parse time:
//    		Fastest Time: 0.007s
//    		Slowest Time: 0.009s
//    		Average Time: 0.007s ± 0.001s
//
//func (p *parser) parseList(
//	openKind token.Kind,
//	parseFunc interface{},
//	closeKind token.Kind) (interface{}, error) {
//
//	// Expect a token of openKind.
//	if _, err := p.expect(openKind); err != nil {
//		return nil, err
//	}
//
//	// Determine the result node type from type of the first return value of parseFunc.
//	parseFuncValue := reflect.ValueOf(parseFunc)
//	parseFuncType := parseFuncValue.Type()
//	nodeType := parseFuncType.Out(0)
//
//	// Create result which is an array of nodes with at least one element.
//	nodes := reflect.MakeSlice(reflect.SliceOf(nodeType), 0, 1)
//
//	for {
//		// Parse a node.
//		retValues := parseFuncValue.Call(nil)
//
//		// Check error.
//		errValue := retValues[1]
//		if errValue.IsValid() && !errValue.IsNil() {
//			return nil, errValue.Interface().(error)
//		}
//
//		// Append the result.
//		nodes = reflect.Append(nodes, retValues[0])
//
//		// Determine whether we should continue by check the current token with closeKind.
//		stop, err := p.skip(closeKind)
//		if err != nil {
//			return nil, err
//		}
//
//		if stop {
//			break
//		}
//	}
//
//	return nodes.Interface(), nil
//}

// Converts a name lex token into a name parse node.
func (p *parser) parseName() (ast.Name, error) {
	token, err := p.expect(token.KindName)
	if err != nil {
		return ast.Name{}, err
	}
	return ast.Name{
		Token: token,
	}, nil
}

// Implements the parsing rules in the Document section.

//	Document ::
//		Definition+
func (p *parser) parseDocument() (ast.Document, error) {
	// Expect SOF.
	if _, err := p.expect(token.KindSOF); err != nil {
		return ast.Document{}, err
	}

	definitions := make([]ast.Definition, 0, 1)
	for {
		definition, err := p.parseDefinition()
		if err != nil {
			return ast.Document{}, err
		}

		definitions = append(definitions, definition)

		// Stop on encountering an EOF token.
		stop, err := p.skip(token.KindEOF)
		if err != nil {
			return ast.Document{}, err
		}

		if stop {
			break
		}
	}

	return ast.Document{
		Definitions: definitions,
	}, nil
}

//	Definition ::
//		ExecutableDefinition
//		TypeSystemDefinition
//		TypeSystemExtension
func (p *parser) parseDefinition() (ast.Definition, error) {
	tok := p.peek()
	switch tok.Kind {
	case token.KindName:
		switch tok.Value {
		case "query", "mutation", "subscription":
			return p.parseOperationDefinition()
		case "fragment":
			return p.parseFragmentDefinition()
		}

	// TODO: TypeSystemDefinition
	// TODO: TypeSystemExtension

	case token.KindLeftBrace:
		// Should be parseExecutableDefinition and then parseOperationDefinition. But directly jump to
		// parseQueryShorthand to make it a slightly faster.
		return p.parseQueryShorthand()
	}

	return nil, p.unexpected()
}

//	ExecutableDefinition ::
//		OperationDefinition
//		FragmentDefinition
//
// This is supposed to be called from parseDefinition() but we instead directly jump to
// parse{OperationDefinition,FragmentDefinition,QueryShorthand} instead.
//
//func (p *parser) parseExecutableDefinition() (ast.ExecutableDefinition, error) {
//	tok := p.peek()
//	switch p.peek().Kind {
//	case token.KindName:
//		switch tok.Value {
//		case "query", "mutation", "subscription":
//			return p.parseOperationDefinition()
//		case "fragment":
//			return p.parseFragmentDefinition()
//		}
//
//	case token.KindLeftBrace:
//		return p.parseOperationDefinition()
//	}
//
//	return nil, p.unexpected()
//}

//	OperationDefinition ::
// 		OperationType Name? VariableDefinitions? Directives? SelectionSet
//		SelectionSet
//
// Note the second rule which is known as "Query Shorthand" is handled by parseQueryShorthand() not
// here. And we ensure that there's no one one call this function for handle that case (by direcly
// call parseQueryShorthand() from parseDefinition() when a left brace is seen.)
func (p *parser) parseOperationDefinition() (*ast.OperationDefinition, error) {
	var (
		name                ast.Name
		variableDefinitions []*ast.VariableDefinition
		directives          ast.Directives
		selectionSet        ast.SelectionSet
	)

	operationType, err := p.expect(token.KindName)
	if err != nil {
		return nil, err
	}

	if p.peek().Kind == token.KindName {
		if name, err = p.parseName(); err != nil {
			return nil, err
		}
	}

	if p.peek().Kind == token.KindLeftParen {
		if variableDefinitions, err = p.parseVariableDefinitions(); err != nil {
			return nil, err
		}
	}

	if p.peek().Kind == token.KindAt {
		if directives, err = p.parseDirectives(false /* isConst */); err != nil {
			return nil, err
		}
	}

	if selectionSet, err = p.parseSelectionSet(); err != nil {
		return nil, err
	}

	return &ast.OperationDefinition{
		DefinitionBase: ast.DefinitionBase{
			Directives: directives,
		},
		Type:                operationType,
		Name:                name,
		VariableDefinitions: variableDefinitions,
		SelectionSet:        selectionSet,
	}, nil
}

// Parse a "Query Shorthand" which is a query operation represented in a short‐hand form. It only
// specifies a SelectionSet, omitting the query keyword, query name and any others.
//
// For example: "{ field }"
//
// Reference: https://facebook.github.io/graphql/June2018/#sec-Language.Operations
func (p *parser) parseQueryShorthand() (*ast.OperationDefinition, error) {
	selectionSet, err := p.parseSelectionSet()
	if err != nil {
		return nil, err
	}

	return &ast.OperationDefinition{
		SelectionSet: selectionSet,
	}, nil
}

//	SelectionSet ::
//		{ Selection+ }
func (p *parser) parseSelectionSet() (ast.SelectionSet, error) {
	// Expect {.
	if _, err := p.expect(token.KindLeftBrace); err != nil {
		return nil, err
	}

	selections := make([]ast.Selection, 0, 1)
	for {
		selection, err := p.parseSelection()
		if err != nil {
			return nil, err
		}

		selections = append(selections, selection)

		// Stop on } token.
		stop, err := p.skip(token.KindRightBrace)
		if err != nil {
			return nil, err
		}

		if stop {
			break
		}
	}

	return ast.SelectionSet(selections), nil
}

//	Selection ::
//		Field
//		FragmentSpread
//		InlineFragment
//
//	FragmentSpread ::
//		... FragmentName Directives?
//
//	InlineFragment ::
//		... TypeCondition? Directives? SelectionSet
func (p *parser) parseSelection() (ast.Selection, error) {
	// Both FragmentSpread and InlineFragment start with "...".
	isFragment, err := p.skip(token.KindSpread)
	if err != nil {
		return nil, err
	} else if isFragment {
		// Peek the next token to determine which rule should we go.
		tok := p.peek()
		if tok.Kind != token.KindName || tok.Value == "on" {
			// Must be a InlineFragment.
			return p.parseInlineFragment()
		}
		return p.parseFragmentSpread()
	}
	return p.parseField()
}

//	Field ::
//		Alias? Name Arguments? Directives? SelectionSet?
//
//	Alias ::
//		Name :
func (p *parser) parseField() (*ast.Field, error) {
	var (
		alias        ast.Name
		name         ast.Name
		arguments    ast.Arguments
		directives   ast.Directives
		selectionSet ast.SelectionSet
	)

	nameOrAlias, err := p.parseName()
	if err != nil {
		return nil, err
	}

	hasColon, err := p.skip(token.KindColon)
	if err != nil {
		return nil, err
	}

	if !hasColon {
		name = nameOrAlias
	} else {
		alias = nameOrAlias
		name, err = p.parseName()
		if err != nil {
			return nil, err
		}
	}

	if p.peek().Kind == token.KindLeftParen {
		arguments, err = p.parseArguments(false)
		if err != nil {
			return nil, err
		}
	}

	if p.peek().Kind == token.KindAt {
		directives, err = p.parseDirectives(false)
		if err != nil {
			return nil, err
		}
	}

	if p.peek().Kind == token.KindLeftBrace {
		selectionSet, err = p.parseSelectionSet()
		if err != nil {
			return nil, err
		}
	}

	return &ast.Field{
		Alias:        alias,
		Name:         name,
		Arguments:    arguments,
		Directives:   directives,
		SelectionSet: selectionSet,
	}, nil
}

//	FragmentSpread
//		... FragmentName Directives?
//
// Note that this function assumes "..." has been consumed (see parseSelection, it needs a lookahead
// for distinguish between InlineFragment.)
func (p *parser) parseFragmentSpread() (*ast.FragmentSpread, error) {
	name, err := p.parseName()
	if err != nil {
		return nil, err
	}

	var directives ast.Directives
	if tok := p.peek(); tok.Kind == token.KindAt {
		if directives, err = p.parseDirectives(false /* isConst */); err != nil {
			return nil, err
		}
	}

	return &ast.FragmentSpread{
		Name:       name,
		Directives: directives,
	}, nil
}

//	FragmentDefinition ::
//		fragment FragmentName TypeCondition Directives? SelectionSet
func (p *parser) parseFragmentDefinition() (*ast.FragmentDefinition, error) {
	// "fragment" keyword must already been expected before here. Simply expect a Name token to
	// advance.
	if _, err := p.expect(token.KindName); err != nil {
		return nil, err
	}

	name, err := p.parseFragmentName()
	if err != nil {
		return nil, err
	}

	var variableDefinitions []*ast.VariableDefinition
	if p.options.ExperimentalFragmentVariables {
		if tok := p.peek(); tok.Kind == token.KindLeftParen {
			if variableDefinitions, err = p.parseVariableDefinitions(); err != nil {
				return nil, err
			}
		}
	}

	typeCondition, err := p.parseTypeCondition()
	if err != nil {
		return nil, err
	}

	var directives ast.Directives
	if tok := p.peek(); tok.Kind == token.KindAt {
		if directives, err = p.parseDirectives(false /* isConst */); err != nil {
			return nil, err
		}
	}

	selectionSet, err := p.parseSelectionSet()
	if err != nil {
		return nil, err
	}

	return &ast.FragmentDefinition{
		DefinitionBase: ast.DefinitionBase{
			Directives: directives,
		},
		Name:                name,
		VariableDefinitions: variableDefinitions,
		TypeCondition:       typeCondition,
		SelectionSet:        selectionSet,
	}, nil
}

//	FragmentName ::
//		Name but not on
func (p *parser) parseFragmentName() (ast.Name, error) {
	if tok := p.peek(); tok.Kind == token.KindName && tok.Value == "on" {
		return ast.Name{}, graphql.NewSyntaxError(p.lexer.Source(), tok.Location,
			`Expected a fragment name before "on"`)
	}

	return p.parseName()
}

//	TypeCondition ::
//		on NamedType
func (p *parser) parseTypeCondition() (ast.NamedType, error) {
	if err := p.expectKeyword("on"); err != nil {
		return ast.NamedType{}, err
	}
	return p.parseNamedType()
}

//	InlineFragment
//		... TypeCondition? Directives? SelectionSet
func (p *parser) parseInlineFragment() (*ast.InlineFragment, error) {
	var (
		typeCondition ast.NamedType
		directives    ast.Directives
		err           error
	)

	if tok := p.peek(); tok.Kind == token.KindName {
		if typeCondition, err = p.parseTypeCondition(); err != nil {
			return nil, err
		}
	}

	if tok := p.peek(); tok.Kind == token.KindAt {
		if directives, err = p.parseDirectives(false /* isConst */); err != nil {
			return nil, err
		}
	}

	selectionSet, err := p.parseSelectionSet()
	if err != nil {
		return nil, err
	}

	return &ast.InlineFragment{
		TypeCondition: typeCondition,
		Directives:    directives,
		SelectionSet:  selectionSet,
	}, nil
}

//	Arguments ::
//		( Argument+ )
func (p *parser) parseArguments(isConst bool) (ast.Arguments, error) {
	if _, err := p.expect(token.KindLeftParen); err != nil {
		return nil, err
	}

	arguments := make([]*ast.Argument, 0, 1)
	for {
		argument, err := p.parseArgument(isConst)
		if err != nil {
			return nil, err
		}

		arguments = append(arguments, argument)

		// Stop on } token.
		stop, err := p.skip(token.KindRightParen)
		if err != nil {
			return nil, err
		}

		if stop {
			break
		}
	}

	return ast.Arguments(arguments), nil
}

//	Argument ::
//		Name : Value
func (p *parser) parseArgument(isConst bool) (*ast.Argument, error) {
	name, err := p.parseName()
	if err != nil {
		return nil, err
	}

	if _, err := p.expect(token.KindColon); err != nil {
		return nil, err
	}

	value, err := p.parseValue(isConst)
	if err != nil {
		return nil, err
	}

	return &ast.Argument{
		Name:  name,
		Value: value,
	}, nil
}

//	Value ::
//		Variable
//		IntValue
//		FloatValue
//		StringValue
//		BooleanValue
//		NullValue
//		EnumValue
//		ListValueConst
//		ObjectValueConst
//
//	BooleanValue::
//		true or false
//
//	NullValue::
//		null
//
//	EnumValue ::
//		Name but not true or false or null
func (p *parser) parseValue(isConst bool) (ast.Value, error) {
	tok := p.peek()
	switch tok.Kind {
	case token.KindDollar:
		if !isConst {
			return p.parseVariable()
		}

	case token.KindInt:
		if _, err := p.lexer.Advance(); err != nil {
			return nil, err
		}
		return ast.IntValue{
			Token: tok,
		}, nil

	case token.KindFloat:
		if _, err := p.lexer.Advance(); err != nil {
			return nil, err
		}
		return ast.FloatValue{
			Token: tok,
		}, nil

	case token.KindString, token.KindBlockString:
		if _, err := p.lexer.Advance(); err != nil {
			return nil, err
		}
		return ast.StringValue{
			Token: tok,
		}, nil

	case token.KindName:
		if _, err := p.lexer.Advance(); err != nil {
			return nil, err
		}

		switch tok.Value {
		case "true", "false":
			return ast.BooleanValue{
				Token: tok,
			}, nil

		case "null":
			return ast.NullValue{
				Token: tok,
			}, nil

		default:
			return ast.EnumValue{
				Token: tok,
			}, nil
		}

	case token.KindLeftBracket:
		return p.parseListValue(isConst)

	case token.KindLeftBrace:
		return p.parseObjectValue(isConst)
	}

	return nil, p.unexpected()
}

//	ListValue ::
//		[ ]
//		[ Value+ ]
func (p *parser) parseListValue(isConst bool) (ast.ListValue, error) {
	startToken, err := p.expect(token.KindLeftBracket)
	if err != nil {
		return ast.ListValue{}, err
	}

	var values []ast.Value
	for {
		// Stop on ] token.
		stop, err := p.skip(token.KindRightBracket)
		if err != nil {
			return ast.ListValue{}, err
		}
		if stop {
			break
		}

		value, err := p.parseValue(isConst)
		if err != nil {
			return ast.ListValue{}, err
		}

		values = append(values, value)
	}

	if len(values) == 0 {
		// Store the start token for empty list value.
		return ast.ListValue{
			ValuesOrStartToken: startToken,
		}, nil
	}
	return ast.ListValue{
		ValuesOrStartToken: values,
	}, nil
}

//	ObjectValue ::
//		{ }
//		{ ObjectField+ }
func (p *parser) parseObjectValue(isConst bool) (ast.ObjectValue, error) {
	startToken, err := p.expect(token.KindLeftBrace)
	if err != nil {
		return ast.ObjectValue{}, err
	}

	var fields []*ast.ObjectField
	for {
		// Stop on } token.
		stop, err := p.skip(token.KindRightBrace)
		if err != nil {
			return ast.ObjectValue{}, err
		}
		if stop {
			break
		}

		// Parse a ObjectField.
		field, err := p.parseObjectField(isConst)
		if err != nil {
			return ast.ObjectValue{}, err
		}

		fields = append(fields, field)
	}

	if len(fields) == 0 {
		// Store the start token for empty list value.
		return ast.ObjectValue{
			FieldsOrStartToken: startToken,
		}, nil
	}
	return ast.ObjectValue{
		FieldsOrStartToken: fields,
	}, nil
}

//	ObjectField ::
//		Name : Value
func (p *parser) parseObjectField(isConst bool) (*ast.ObjectField, error) {
	name, err := p.parseName()
	if err != nil {
		return nil, err
	}

	if _, err := p.expect(token.KindColon); err != nil {
		return nil, err
	}

	value, err := p.parseValue(isConst)
	if err != nil {
		return nil, err
	}

	return &ast.ObjectField{
		Name:  name,
		Value: value,
	}, nil
}

//	Variable ::
//		$ Name
func (p *parser) parseVariable() (ast.Variable, error) {
	if _, err := p.expect(token.KindDollar); err != nil {
		return ast.Variable{}, err
	}

	name, err := p.parseName()
	if err != nil {
		return ast.Variable{}, err
	}

	return ast.Variable{
		Name: name,
	}, nil
}

//	VariableDefinitions ::
//		( VariableDefinition+ )
func (p *parser) parseVariableDefinitions() ([]*ast.VariableDefinition, error) {
	var variableDefinitions []*ast.VariableDefinition

	if _, err := p.expect(token.KindLeftParen); err != nil {
		return nil, err
	}

	for {
		variableDefinition, err := p.parseVariableDefinition()
		if err != nil {
			return nil, err
		}
		variableDefinitions = append(variableDefinitions, variableDefinition)

		stop, err := p.skip(token.KindRightParen)
		if err != nil {
			return nil, err
		} else if stop {
			break
		}

		// Continue parsing a VariableDefinition node.
	}

	return variableDefinitions, nil
}

//	VariableDefinition ::
//		Variable : Type DefaultValue? Directives?
func (p *parser) parseVariableDefinition() (*ast.VariableDefinition, error) {
	var (
		defaultValue ast.Value
		directives   ast.Directives
	)

	variable, err := p.parseVariable()
	if err != nil {
		return nil, err
	}

	if _, err := p.expect(token.KindColon); err != nil {
		return nil, err
	}

	variableType, err := p.parseType()
	if err != nil {
		return nil, err
	}

	if p.peek().Kind == token.KindEquals {
		if defaultValue, err = p.parseDefaultValue(); err != nil {
			return nil, err
		}
	}

	if p.peek().Kind == token.KindAt {
		if directives, err = p.parseDirectives(true /* isConst */); err != nil {
			return nil, err
		}
	}

	return &ast.VariableDefinition{
		Variable:     variable,
		Type:         variableType,
		DefaultValue: defaultValue,
		Directives:   directives,
	}, nil
}

//	Type ::
//		NamedType
//		ListType
//		NonNullType
//
//	NamedType ::
//		Name
//
//	ListType ::
//		[ Type ]
//
//	NonNullType ::
//		NamedType !
//		ListType !
func (p *parser) parseType() (ast.Type, error) {
	var t ast.Type

	// See how many level are the innermost named type nested in the list.
	listLevel := 0
	for {
		isOpeningList, err := p.skip(token.KindLeftBracket)
		if err != nil {
			return nil, err
		} else if isOpeningList {
			listLevel++
		} else {
			// Must be a Name.
			name, err := p.parseName()
			if err != nil {
				return nil, err
			}

			t = ast.NamedType{
				Name: name,
			}

			// Stop when innermost named type is reached. No opening list is allowed.
			break
		}
	}

	for listLevel > 0 {
		isNonNull, err := p.skip(token.KindBang)
		if err != nil {
			return nil, err
		} else if isNonNull {
			t = ast.NonNullType{
				// Must be a nullable type because we only allow at most one "!" when closing the list.
				Type: t.(ast.NullableType),
			}
		}

		if _, err := p.expect(token.KindRightBracket); err != nil {
			return nil, err
		}

		t = ast.ListType{
			ItemType: t,
		}
		listLevel--
	}

	// The result type could be further wrapped into a non-null type.
	isNonNull, err := p.skip(token.KindBang)
	if err != nil {
		return nil, err
	} else if isNonNull {
		t = ast.NonNullType{
			Type: t.(ast.NullableType),
		}
	}

	return t, nil
}

//	NamedType ::
//		Name
func (p *parser) parseNamedType() (ast.NamedType, error) {
	name, err := p.parseName()
	if err != nil {
		return ast.NamedType{}, err
	}

	return ast.NamedType{
		Name: name,
	}, nil
}

//	DefaultValue ::
//		= Value
func (p *parser) parseDefaultValue() (ast.Value, error) {
	if _, err := p.expect(token.KindEquals); err != nil {
		return nil, err
	}

	value, err := p.parseValue(true /* isConst */)
	if err != nil {
		return nil, err
	}

	return value, nil
}

//	Directives ::
//		Directive+
func (p *parser) parseDirectives(isConst bool) (ast.Directives, error) {
	var directives ast.Directives

	for {
		directive, err := p.parseDirective(isConst)
		if err != nil {
			return nil, err
		}
		directives = append(directives, directive)

		if p.peek().Kind != token.KindAt {
			break
		}

		// Continue parsing a Directive node.
	}

	return directives, nil
}

//	Directive ::
//		@ Name Arguments?
func (p *parser) parseDirective(isConst bool) (*ast.Directive, error) {
	if _, err := p.expect(token.KindAt); err != nil {
		return nil, err
	}

	name, err := p.parseName()
	if err != nil {
		return nil, err
	}

	var arguments ast.Arguments
	if p.peek().Kind == token.KindLeftParen {
		arguments, err = p.parseArguments(isConst)
		if err != nil {
			return nil, err
		}
	}

	return &ast.Directive{
		Name:      name,
		Arguments: arguments,
	}, nil
}
