// +build ignore

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

package main

import (
	"fmt"
	"go/importer"
	"go/types"
	"io"
	"log"
	"os"
	"reflect"
	"strings"
	"text/template"
)

const filename = "visitor.generated.go"

// List of "abstract" AST node we're interested. An AST node that is declared with a Go interface
// will be ignored if it is not listed. They're also used to filter the types in ast packages. Type
// that doesn't implements any of the abstracts in the list will be considered as non-AST node type
// and will be ignored.
var abstractASTNodeNames = []string{
	"Node",
	"Type",
	"Value",
	"Definition",
	"Selection",
}

// ASTNodeTypeInfo contains information for an AST node.
type ASTNodeTypeInfo struct {
	// The node name
	Name string

	// The type definition
	Type types.Type

	// The type ctx of the children AST nodes; It has different meanings based on Type:
	//
	//  * If this is an "abstract" AST node (see comments for abstractASTNodeNames), this holds
	//    possible nodes for the node.
	//
	//  * If this is an AST node that represents an array of other AST nodes, this holds the element
	//    type (and the len(Children) should be 1).
	//
	//  * If Type is a pointer type, it is expected to be pointer to a struct type and Children
	//    contains the AST nodes in the pointee struct fields.
	//
	//  * Otherwise, Type must be a struct type and Children contains the AST nodes in the struct
	//    fields.
	Children []*ASTNodeTypeInfo
}

// IsAbstract returns true if the type represented by ASTNodeTypeInfo is an abstract type.
func (ctx *ASTNodeTypeInfo) IsAbstract() bool {
	return types.IsInterface(ctx.Type)
}

// IsPointer returns true if the type represented by ASTNodeTypeInfo is a pointer type.
func (ctx *ASTNodeTypeInfo) IsPointer() bool {
	_, ok := ctx.Type.Underlying().(*types.Pointer)
	return ok
}

// IsArray returns true if the type represented by ASTNodeTypeInfo is an array type.
func (ctx *ASTNodeTypeInfo) IsArray() bool {
	_, ok := ctx.Type.Underlying().(*types.Slice)
	return ok
}

// IsStruct returns true if the type represented by ASTNodeTypeInfo is a struct type.
func (ctx *ASTNodeTypeInfo) IsStruct() bool {
	_, ok := ctx.Type.Underlying().(*types.Struct)
	return ok
}

// TypeExpr returns the string for referencing the type of AST node in generated Go code.
func (ctx *ASTNodeTypeInfo) TypeExpr() string {
	if ctx.IsPointer() {
		return "*ast." + ctx.Name
	}
	return "ast." + ctx.Name
}

// VisitActionInstance returns the name of the visitor instance for visiting the node.
func (ctx *ASTNodeTypeInfo) VisitActionInstance() string {
	return strings.ToLower(ctx.Name[:1]) + ctx.Name[1:] + "VisitAction"
}

// NilCheck generate tests for checking whether
func (ctx *ASTNodeTypeInfo) NilCheck(field string) string {
	if ctx.IsAbstract() || ctx.IsPointer() {
		return field + " != nil"
	} else if ctx.IsArray() {
		return "len(" + field + ") != 0"
	} else {
		switch ctx.Name {
		case "Name":
			return "!" + field + ".IsNil()"
		case "NamedType":
			return "!" + field + ".Name.IsNil()"
		default:
			panic(fmt.Sprintf(`unhandled nil check for optional field "%s" with type %s`,
				field, ctx.Name))
		}
	}
}

// PossibleTypes returns list of possible concrete types for the type.
func (ctx *ASTNodeTypeInfo) PossibleTypes() []*ASTNodeTypeInfo {
	var result []*ASTNodeTypeInfo
	possibleTypes := []*ASTNodeTypeInfo{ctx}
	possibleTypeMap := map[*ASTNodeTypeInfo]bool{ctx: true}
	for len(possibleTypes) > 0 {
		var t *ASTNodeTypeInfo
		t, possibleTypes = possibleTypes[len(possibleTypes)-1], possibleTypes[:len(possibleTypes)-1]
		if !t.IsAbstract() {
			result = append(result, t)
		} else {
			// Scan the child nodes.
			for _, child := range t.Children {
				if _, exists := possibleTypeMap[child]; !exists {
					possibleTypeMap[child] = true
					possibleTypes = append(possibleTypes, child)
				}
			}
		}
	}
	return result
}

var astNodes []*ASTNodeTypeInfo

func discoverASTNodeTypes() error {
	// Import ast package.
	pkg, err := importer.For("source", nil).Import("github.com/botobag/artemis/graphql/ast")
	if err != nil {
		return err
	}

	scope := pkg.Scope()

	// Map node type name to the ASTNodeTypeInfo.
	astNodeMap := map[string]*ASTNodeTypeInfo{}

	// Initialize abstract AST nodes (without setting possible node types in their Children).
	abstractASTNodes := make([]*ASTNodeTypeInfo, 0, len(abstractASTNodeNames))
	for _, abstractASTNodeName := range abstractASTNodeNames {
		obj := scope.Lookup(abstractASTNodeName)
		if obj == nil {
			return fmt.Errorf(`Abstract node "%s" cannot be found in ast package`, abstractASTNodeName)
		}

		typ := obj.Type().Underlying()
		if !types.IsInterface(typ) {
			return fmt.Errorf(`Abstract nod "%s" is expected to be a Go interface in ast package`, abstractASTNodeName)
		}

		typeInfo := &ASTNodeTypeInfo{
			Name: abstractASTNodeName,
			Type: typ,
		}
		abstractASTNodes = append(abstractASTNodes, typeInfo)

		// Add to astNodeMap.
		astNodeMap[abstractASTNodeName] = typeInfo

		// Add to astNodes.
		astNodes = append(astNodes, typeInfo)
	}

	// Discover node types from the pkg to initialize astNodes (without setting Children).
	names := scope.Names()
	for _, name := range names {
		obj := scope.Lookup(name)
		if obj == nil {
			return fmt.Errorf(`named entity "%s" is listed in package but cannot be found`, name)
		}

		// Only process exported symbols.
		if !obj.Exported() {
			continue
		}

		typeName, ok := obj.(*types.TypeName)
		// Only process entity that indicates a named type.
		if !ok {
			continue
		}

		typ := typeName.Type()

		// Skip abstract type.
		if types.IsInterface(typ) {
			continue
		}

		// Create type ctx object in advance.
		typeInfo := &ASTNodeTypeInfo{
			Name: name,
			Type: typ,
		}

		// Skip node type that implement none of abstractASTNodes.
		skipped := true
		for _, abstractASTNode := range abstractASTNodes {
			if types.Implements(typ, abstractASTNode.Type.(*types.Interface)) {
				skipped = false
				// Add to abstractASTNode possible types.
				abstractASTNode.Children = append(abstractASTNode.Children, typeInfo)
			}
		}
		if skipped {
			// See whether it's the pointer to type that implements one of abstractASTNodes.
			typ = types.NewPointer(typ)
			for _, abstractASTNode := range abstractASTNodes {
				if types.Implements(typ, abstractASTNode.Type.(*types.Interface)) {
					skipped = false
					typeInfo.Type = typ
					// Add to abstractASTNode possible types.
					abstractASTNode.Children = append(abstractASTNode.Children, typeInfo)
				}
			}

			// No luck.
			if skipped {
				continue
			}
		}

		// Register in astNodeMap.
		astNodeMap[name] = typeInfo
		// Add to astNodes.
		astNodes = append(astNodes, typeInfo)
	}

	// Scan node children.
	for _, node := range astNodes {
		// Get the underlying type.
		typ := node.Type.Underlying()

		switch typ := typ.(type) {
		case *types.Interface:
			// Here we handle the case where one abstract may be included by the other abstract. Take
			// ast.Value which is an abstract that implements ast.Node as example, the following add
			// ast.Value to ast.Node's children (i.e., possible nodes) and removes the children in
			// ast.Value from ast.Node.
			for _, abstractASTNode := range abstractASTNodes {
				if node != abstractASTNode {
					if types.Implements(typ, abstractASTNode.Type.(*types.Interface)) {
						// Construct new possible nodes set of abstractASTNode which excludes nodes that
						// implements node.Type.
						possibleNodes := make([]*ASTNodeTypeInfo, 0, len(abstractASTNode.Children)-len(node.Children)+1)
						for _, possibleNode := range abstractASTNode.Children {
							if !types.Implements(possibleNode.Type, node.Type.(*types.Interface)) {
								possibleNodes = append(possibleNodes, possibleNode)
							}
						}
						possibleNodes = append(possibleNodes, node)
						abstractASTNode.Children = possibleNodes
					}
				}
			}

		case *types.Pointer, *types.Struct:
			structType, ok := typ.(*types.Struct)
			if !ok {
				// Must be a pointer. Obtain its element type.
				elementType := typ.(*types.Pointer).Elem().Underlying()
				// The elementType must be a struct type.
				structType, ok = elementType.(*types.Struct)
				if !ok {
					return fmt.Errorf("%s is a pointer-type node but has unexpected non-struct pointee type", node.Name)
				}
			}

			// Scan fields.
			hasChildren := false
			numFields := structType.NumFields()
			for i := 0; i < numFields; i++ {
				field := structType.Field(i)

				// Figure out the type name of the field.
				fieldType, ok := field.Type().(*types.Named)
				if !ok {
					// The only case will be a pointer to a node.
					if pointerTyp, ok := field.Type().Underlying().(*types.Pointer); ok {
						fieldType, _ = pointerTyp.Elem().(*types.Named)
					}
				}

				var fieldTypeInfo *ASTNodeTypeInfo
				if fieldType != nil {
					fieldTypeName := fieldType.Obj().Name()
					fieldTypeInfo = astNodeMap[fieldTypeName]
					if fieldTypeInfo != nil {
						hasChildren = true
					}
				}
				node.Children = append(node.Children, fieldTypeInfo)
			}

			if !hasChildren {
				node.Children = nil
			}

		case *types.Slice:
			// Set children node to the element type.
			elementType := typ.Elem()
			if _, ok := elementType.(*types.Pointer); ok {
				elementType = elementType.(*types.Pointer).Elem()
			}
			elementTypeInfo := astNodeMap[elementType.(*types.Named).Obj().Name()]
			if elementTypeInfo == nil {
				return fmt.Errorf("%s is an array-type node but its element contains non-AST Node", node.Name)
			}
			node.Children = []*ASTNodeTypeInfo{elementTypeInfo}

		default:
			return fmt.Errorf(`unsupported Go type "%T" found for node "%s"`, typ, node.Name)
		}
	}

	return nil
}

func genHeader(w io.Writer) {
	fmt.Fprintln(w, `/**
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

package visitor
`)

	fmt.Fprintln(w, `// Code generated by running "go generate" in github.com/botobag/artemis/graphql/ast/visitor.`)
	fmt.Fprintln(w, `// DO NOT EDIT.`)
	fmt.Fprintln(w)
}

func genImports(w io.Writer) {
	imports := []string{
		"fmt",
		"github.com/botobag/artemis/graphql/ast",
	}

	fmt.Fprintln(w, "import (")
	for _, pkg := range imports {
		fmt.Fprintf(w, "\t\"%s\"\n", pkg)
	}
	fmt.Fprintln(w, ")")
}

func genVisitActionInterfaces(w io.Writer) {
	tmpl, err := template.New("visitor-interface").Parse(`
// {{.Node}}VisitAction implements visiting function for {{.Node}}.
type {{.Node}}VisitAction interface {
	Visit{{.Node}}(node {{.Type}}, ctx interface{}) Result
}

// {{.Node}}VisitActionFunc is an adapter to help define a {{.Node}}VisitAction from a function
// which specifies action when traversing a node.
type {{.Node}}VisitActionFunc func(node {{.Type}}, ctx interface{}) Result

var _ {{.Node}}VisitAction = ({{.Node}}VisitActionFunc)(nil)

// Visit{{.Node}} implements {{.Node}}VisitAction by calling f(node, ctx).
func (f {{.Node}}VisitActionFunc) Visit{{.Node}}(node {{.Type}}, ctx interface{}) Result {
	return f(node, ctx)
}
`)
	if err != nil {
		panic(err)
	}

	for _, node := range astNodes {
		err := tmpl.Execute(w, map[string]string{
			"Node": node.Name,
			"Type": node.TypeExpr(),
		})
		if err != nil {
			panic(err)
		}
	}
}

func genVisitor(w io.Writer) {
	fmt.Fprintf(w, `
// A Visitor is provided to Walk to apply actions during AST traversal. It contains a collection of
// actions to be executed for each type of node during the traversal.
type Visitor struct {`)

	for _, node := range astNodes {
		if !node.IsAbstract() {
			fmt.Fprintf(w, `
	%s %19sVisitAction`, node.VisitActionInstance(), node.Name)
		}
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "}")

	for _, node := range astNodes {
		if !node.IsAbstract() {
			fmt.Fprintf(w, `
// Visit%s applies actions on %s.
func (v *Visitor) Visit%s(node %s, ctx interface{}) Result {
	if v.%s != nil {
		return v.%s.Visit%s(node, ctx)
	}
	return Continue
}
`, node.Name, node.Name, node.Name, node.TypeExpr(), node.VisitActionInstance(),
				node.VisitActionInstance(), node.Name)
		}
	}
}

func genNewVisitor(w io.Writer) {
	for _, node := range astNodes {
		fmt.Fprintf(w, `
// New%sVisitor creates a visitor instance which performs the given action when encountering %s.
func New%sVisitor(action %sVisitAction) *Visitor {
	return &Visitor{`, node.Name, node.Name, node.Name, node.Name)
		if node.IsAbstract() {
			for _, possibleType := range node.PossibleTypes() {
				fmt.Fprintf(w, `
		%s: %sVisitActionFunc(func(node %s, ctx interface{}) Result {
			return action.Visit%s(node, ctx)
		}),`, possibleType.VisitActionInstance(), possibleType.Name, possibleType.TypeExpr(), node.Name)
			}
		} else {
			fmt.Fprintf(w, `
		%s: action,`, node.VisitActionInstance())
		}
		fmt.Fprintln(w, `
	}
}`)
	}
}

func genWalk(w io.Writer, preorder bool) {
	for _, node := range astNodes {
		fmt.Fprintf(w, `
func walk%s(node %s, ctx interface{}, v *Visitor) Result {`, node.Name, node.TypeExpr())
		// Enter a node.
		if !node.IsAbstract() && preorder {
			fmt.Fprintf(w, `
	if result := v.Visit%s(node, ctx); result != Continue {
		return result
	}
`, node.Name)
		}

		// Special case: ListValue
		if node.Name == "ListValue" {
			fmt.Fprintf(w, `
	for _, value := range node.Values() {
		if result := walkValue(value, ctx, v); result == Break {
			return result
		}
	}
`)
		}

		// Special case: ObjectValue
		if node.Name == "ObjectValue" {
			fmt.Fprintf(w, `
	for _, field := range node.Fields() {
		if result := walkObjectField(field, ctx, v); result == Break {
			return result
		}
	}
`)
		}

		// Special case: NonNullType
		if node.Name == "NonNullType" {
			fmt.Fprintf(w, `
	var result Result
	switch t := node.Type.(type) {
	case ast.NamedType:
		result = walkNamedType(t, ctx, v)
	case ast.ListType:
		result = walkListType(t, ctx, v)
	default:
		panic(fmt.Sprintf("unhandled nullable type \"%%T\"", node.Type))
	}
	if result == Break {
		return result
	}
`)
		}

		// Visit children.
		if len(node.Children) > 0 {
			if node.IsAbstract() {
				fmt.Fprintf(w, `
	var result Result
	switch node := node.(type) {`)
				for _, possibleType := range node.Children {
					fmt.Fprintf(w, `
	case %s:
		result = walk%s(node, ctx, v)`, possibleType.TypeExpr(), possibleType.Name)
				}

				fmt.Fprintf(w, `
	default:
		panic(fmt.Sprintf("unexpected node type %%T when visiting %s", node))
	}
	if result == Break {
		return result
	}`, node.Name)

			} else if node.IsArray() {
				elementTypeInfo := node.Children[0]
				fmt.Fprintf(w, `
	for _, childNode := range node {
		if result := walk%s(childNode, ctx, v); result == Break {
			return result
		}
	}`, elementTypeInfo.Name)

			} else if node.IsPointer() || node.IsStruct() {
				typ := node.Type.Underlying()
				structType, ok := typ.(*types.Struct)
				if !ok {
					// Must be a pointer. Obtain its element type and casting (should have been checked when
					// discovering types).
					structType = typ.(*types.Pointer).Elem().Underlying().(*types.Struct)
				}

				// Scan fields.
				numFields := structType.NumFields()
				for i := 0; i < numFields; i++ {
					field := structType.Field(i)
					fieldTag := reflect.StructTag(structType.Tag(i))
					fieldTypeInfo := node.Children[i]
					if fieldTypeInfo == nil {
						// Skip non-AST node field.
						continue
					}

					fmt.Fprintf(w, `
	// Visit %s.`, field.Name())

					isOptional := fieldTag.Get("ast") == "optional"
					if isOptional {
						fmt.Fprintf(w, `
	if %s {
		if result := walk%s(node.%s, ctx, v); result == Break {
			return result
		}
	}`, fieldTypeInfo.NilCheck("node."+field.Name()), fieldTypeInfo.Name, field.Name())
					} else {
						fmt.Fprintf(w, `
	if result := walk%s(node.%s, ctx, v); result == Break {
		return result
	}`, fieldTypeInfo.Name, field.Name())
					}

				}
			} else {
				panic("unknown children type in node " + node.Name)
			}
			fmt.Fprintln(w)
		}

		// Leave node.
		if !node.IsAbstract() && !preorder {
			fmt.Fprintf(w, `
	return v.Visit%s(node, ctx)
`, node.Name)
		} else {
			fmt.Fprintf(w, `
	return Continue
`)
		}

		fmt.Fprintln(w, `}`)
	}
}

func main() {
	w, err := os.Create(filename)
	if err != nil {
		log.Fatalln(err)
	}
	defer w.Close()

	if err := discoverASTNodeTypes(); err != nil {
		log.Fatalln(err)
	}

	genHeader(w)
	genImports(w)
	genVisitActionInterfaces(w)
	genVisitor(w)
	genNewVisitor(w)
	genWalk(w, true)
}
