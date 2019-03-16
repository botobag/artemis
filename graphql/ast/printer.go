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

package ast

import (
	"fmt"
	"strings"

	"github.com/botobag/artemis/internal/util"
	"github.com/botobag/artemis/jsonwriter"
)

// Print uses a set of formatting rules (compatible with graphql-js) to convert an AST into a
// string.
func Print(node Node) string {
	var buf util.StringBuilder
	FPrint(&buf, node)
	return buf.String()
}

// FPrint "pretty-prints" an AST node to out.
func FPrint(out util.StringWriter, node Node) {
	(&printer{
		StringWriter: out,
	}).printNode(node)
}

type printer struct {
	util.StringWriter
	indentLevel int
}

func (p *printer) beginBlock() {
	p.WriteString("{\n")
	p.indentLevel++
}

func (p *printer) endBlock() {
	p.indentLevel--
	p.writeNewLineWithIndent()
	p.WriteString("}")
}

func (p *printer) writeNewLineWithIndent() {
	p.WriteString("\n")
	p.writeIndent()
}

func (p *printer) writeIndent() {
	p.WriteString(p.indentation())
}

func (p *printer) indentation() string {
	return strings.Repeat(" ", 2*p.indentLevel)
}

// Write implements io.Writer to allow p using jsonwriter.Stream to encode string values.
func (p *printer) Write(b []byte) (n int, err error) {
	return p.WriteString(string(b))
}

func (p *printer) printNode(node Node) {
	switch node := node.(type) {
	case *Argument:
		p.printArgument(node)
	case Arguments:
		p.printArguments("(", node, ", ", ")")
	case Definitions:
		p.printDefinitions(node)
	case *Directive:
		p.printDirective(node)
	case Directives:
		p.printDirectives(node)
	case Document:
		p.printDocument(node)
	case Name:
		p.printName(node)
	case *ObjectField:
		p.printObjectField(node)
	case SelectionSet:
		p.printSelectionSet(node)
	case *VariableDefinition:
		p.printVariableDefinition(node)
	case VariableDefinitions:
		p.printVariableDefinitions(node)
	case Type:
		p.printType(node)
	case Value:
		p.printValue(node)
	case Definition:
		p.printDefinition(node)
	case Selection:
		p.printSelection(node)
	default:
		panic(fmt.Sprintf("unsupported node type %T to print", node))
	}
}

func (p *printer) printName(name Name) {
	p.WriteString(name.Value())
}

//===----------------------------------------------------------------------------------------====//
// Document
//===----------------------------------------------------------------------------------------====//

func (p *printer) printDocument(doc Document) {
	p.printDefinitions(doc.Definitions)
	p.WriteString("\n")
}

func (p *printer) printDefinitions(definitions Definitions) {
	if len(definitions) > 0 {
		p.printDefinition(definitions[0])
		for _, definition := range definitions[1:] {
			p.WriteString("\n\n")
			p.printDefinition(definition)
		}
	}
}

func (p *printer) printDefinition(node Definition) {
	switch node := node.(type) {
	case *FragmentDefinition:
		p.printFragmentDefinition(node)
	case *OperationDefinition:
		p.printOperationDefinition(node)
	case Selection:
		p.printSelection(node)
	default:
		panic(fmt.Sprintf("unexpected node type %T when printing Definition", node))
	}
}

func (p *printer) printOperationDefinition(operation *OperationDefinition) {
	var (
		op           = operation.OperationType()
		name         = operation.Name
		varDefs      = operation.VariableDefinitions
		directives   = operation.Directives
		selectionSet = operation.SelectionSet
	)

	if name.IsNil() && len(directives) == 0 && len(varDefs) == 0 && op == OperationTypeQuery {
		p.printSelectionSet(selectionSet)
	} else {
		p.WriteString(string(op))

		if !name.IsNil() || len(varDefs) > 0 {
			p.WriteString(" ")
			if !name.IsNil() {
				p.printName(name)
			}
			if len(varDefs) > 0 {
				p.printVariableDefinitions(varDefs)
			}
		}

		if len(directives) > 0 {
			p.WriteString(" ")
			p.printDirectives(directives)
		}

		if len(selectionSet) > 0 {
			p.WriteString(" ")
			p.printSelectionSet(selectionSet)
		}
	}
}

func (p *printer) printVariableDefinitions(varDefs VariableDefinitions) {
	if len(varDefs) > 0 {
		p.WriteString("(")
		p.printVariableDefinition(varDefs[0])
		for _, varDef := range varDefs[1:] {
			p.WriteString(", ")
			p.printVariableDefinition(varDef)
		}
		p.WriteString(")")
	}
}

func (p *printer) printVariableDefinition(varDef *VariableDefinition) {
	p.printVariable(varDef.Variable)
	p.WriteString(": ")
	p.printType(varDef.Type)

	if varDef.DefaultValue != nil {
		p.WriteString(" = ")
		p.printValue(varDef.DefaultValue)
	}

	directives := varDef.Directives
	if len(directives) > 0 {
		p.WriteString(" ")
		p.printDirectives(directives)
	}
}

//===----------------------------------------------------------------------------------------====//
// Fragments
//===----------------------------------------------------------------------------------------====//

func (p *printer) printFragmentDefinition(fragmentDef *FragmentDefinition) {
	// Note: fragment variable definitions are experimental and may be changed or removed in the future.
	p.WriteString("fragment ")
	p.printName(fragmentDef.Name)
	p.printVariableDefinitions(fragmentDef.VariableDefinitions)
	p.WriteString(" on ")
	p.printNamedType(fragmentDef.TypeCondition)
	p.WriteString(" ")

	directives := fragmentDef.Directives
	if len(directives) > 0 {
		p.printDirectives(directives)
		p.WriteString(" ")
	}

	p.printSelectionSet(fragmentDef.SelectionSet)
}

func (p *printer) printFragmentSpread(fragment *FragmentSpread) {
	p.WriteString("...")
	p.printName(fragment.Name)

	directives := fragment.Directives
	if len(directives) > 0 {
		p.WriteString(" ")
		p.printDirectives(directives)
	}
}

func (p *printer) printInlineFragment(fragment *InlineFragment) {
	p.WriteString("...")

	if fragment.HasTypeCondition() {
		p.WriteString(" on ")
		p.printNamedType(fragment.TypeCondition)
	}

	directives := fragment.Directives
	if len(directives) > 0 {
		p.WriteString(" ")
		p.printDirectives(directives)
	}

	selectionSet := fragment.SelectionSet
	if len(selectionSet) > 0 {
		p.WriteString(" ")
		p.printSelectionSet(selectionSet)
	}
}

//===----------------------------------------------------------------------------------------====//
// SelectionSet
//===----------------------------------------------------------------------------------------====//

func (p *printer) printSelectionSet(selectionSet SelectionSet) {
	if len(selectionSet) > 0 {
		p.beginBlock()
		p.writeIndent()
		p.printSelection(selectionSet[0])
		for _, selection := range selectionSet[1:] {
			p.writeNewLineWithIndent()
			p.printSelection(selection)
		}
		p.endBlock()
	}
}

func (p *printer) printSelection(node Selection) {
	switch node := node.(type) {
	case *Field:
		p.printField(node)
	case *FragmentSpread:
		p.printFragmentSpread(node)
	case *InlineFragment:
		p.printInlineFragment(node)
	default:
		panic(fmt.Sprintf("unexpected node type %T when printing Selection", node))
	}
}

func (p *printer) printField(field *Field) {
	var (
		alias = field.Alias
	)
	if !alias.IsNil() {
		p.printName(alias)
		p.WriteString(": ")
	}

	p.printName(field.Name)
	p.printArguments("(", field.Arguments, ", ", ")")

	if len(field.Directives) > 0 {
		p.WriteString(" ")
		p.printDirectives(field.Directives)
	}

	if len(field.SelectionSet) > 0 {
		p.WriteString(" ")
		p.printSelectionSet(field.SelectionSet)
	}
}

func (p *printer) printArguments(start string, args Arguments, sep string, end string) {
	if len(args) > 0 {
		p.WriteString(start)
		p.printArgument(args[0])
		for _, arg := range args[1:] {
			p.WriteString(sep)
			p.printArgument(arg)
		}
		p.WriteString(end)
	}
}

func (p *printer) printArgument(arg *Argument) {
	p.printName(arg.Name)
	p.WriteString(": ")
	p.printValue(arg.Value)
}

//===----------------------------------------------------------------------------------------====//
// Value
//===----------------------------------------------------------------------------------------====//

func (p *printer) printValue(node Value) {
	switch node := node.(type) {
	case BooleanValue:
		p.printBooleanValue(node)
	case EnumValue:
		p.printEnumValue(node)
	case FloatValue:
		p.printFloatValue(node)
	case IntValue:
		p.printIntValue(node)
	case ListValue:
		p.printListValue(node)
	case NullValue:
		p.printNullValue(node)
	case ObjectValue:
		p.printObjectValue(node)
	case StringValue:
		p.printStringValue(node, "  ")
	case Variable:
		p.printVariable(node)
	default:
		panic(fmt.Sprintf("unexpected node type %T when printing Value", node))
	}
}

func (p *printer) printBooleanValue(value BooleanValue) {
	if value.Value() {
		p.WriteString("true")
	} else {
		p.WriteString("false")
	}
}

func (p *printer) printEnumValue(value EnumValue) {
	p.WriteString(value.Value())
}

func (p *printer) printFloatValue(value FloatValue) {
	p.WriteString(value.String())
}

func (p *printer) printIntValue(value IntValue) {
	p.WriteString(value.String())
}

func (p *printer) printListValue(value ListValue) {
	values := value.Values()
	p.WriteString("[")
	if len(values) > 0 {
		p.printValue(values[0])
		for _, value := range values[1:] {
			p.WriteString(", ")
			p.printValue(value)
		}
	}
	p.WriteString("]")
}

func (p *printer) printNullValue(value NullValue) {
	p.WriteString("null")
}

func (p *printer) printObjectValue(value ObjectValue) {
	p.WriteString("{")
	fields := value.Fields()
	if len(fields) > 0 {
		p.printObjectField(fields[0])
		for _, field := range fields[1:] {
			p.WriteString(", ")
			p.printObjectField(field)
		}
	}
	p.WriteString("}")
}

func (p *printer) printObjectField(field *ObjectField) {
	p.printName(field.Name)
	p.WriteString(": ")
	p.printValue(field.Value)
}

func (p *printer) printStringValue(value StringValue, blockStringIndent string) {
	if value.IsBlockString() {
		p.printBlockString(value.Value(), blockStringIndent)
	} else {
		// graphql-js: JSON.stringify(value)
		stream := jsonwriter.NewStream(p)
		stream.WriteString(value.Value())
		stream.Flush()
	}
}

// Print a block string in the indented block form by adding a leading and trailing blank line.
// However, if a block string starts with whitespace and is a single-line, adding a leading blank
// line would strip that whitespace.
func (p *printer) printBlockString(value string, indentation string) {
	var (
		isSingleLine         = !strings.ContainsRune(value, '\n')
		hasLeadingSpace      = len(value) > 0 && (value[0] == ' ' || value[0] == '\t')
		hasTrailingQuote     = len(value) > 0 && value[len(value)-1] == '"'
		printAsMultipleLines = !isSingleLine || hasTrailingQuote
	)

	p.WriteString(`"""`)

	// Format a multi-line block quote to account for leading space.
	if printAsMultipleLines && !(isSingleLine && hasLeadingSpace) {
		p.writeNewLineWithIndent()
		p.WriteString(indentation)
	}

	// Replace """ with \""".
	value = strings.Replace(value, `"""`, `\"""`, -1)
	if len(indentation) > 0 {
		value = strings.Replace(value, "\n", "\n"+p.indentation()+indentation, -1)
	}
	p.WriteString(value)

	if printAsMultipleLines {
		p.writeNewLineWithIndent()
	}

	p.WriteString(`"""`)
}

func (p *printer) printVariable(v Variable) {
	p.WriteString("$")
	p.printName(v.Name)
}

//===----------------------------------------------------------------------------------------====//
// Type
//===----------------------------------------------------------------------------------------====//

func (p *printer) printType(node Type) {
	switch node := node.(type) {
	case ListType:
		p.printListType(node)
	case NamedType:
		p.printNamedType(node)
	case NonNullType:
		p.printNonNullType(node)
	default:
		panic(fmt.Sprintf("unexpected node type %T when printing Type", node))
	}
}

func (p *printer) printListType(list ListType) {
	p.WriteString("[")
	p.printType(list.ItemType)
	p.WriteString("]")
}

func (p *printer) printNamedType(named NamedType) {
	p.printName(named.Name)
}

func (p *printer) printNonNullType(nonNull NonNullType) {
	p.printType(nonNull.Type)
	p.WriteString("!")
}

//===----------------------------------------------------------------------------------------====//
// Directive
//===----------------------------------------------------------------------------------------====//

func (p *printer) printDirectives(directives Directives) {
	if len(directives) > 0 {
		p.printDirective(directives[0])
		for _, directive := range directives[1:] {
			p.WriteString(" ")
			p.printDirective(directive)
		}
	}
}

func (p *printer) printDirective(directive *Directive) {
	p.WriteString("@")
	p.printName(directive.Name)
	p.printArguments("(", directive.Arguments, ", ", ")")
}
