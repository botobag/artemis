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

	// The type info of the children AST nodes; It has different meanings based on Type:
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
func (info *ASTNodeTypeInfo) IsAbstract() bool {
	return types.IsInterface(info.Type)
}

// IsPointer returns true if the type represented by ASTNodeTypeInfo is a pointer type.
func (info *ASTNodeTypeInfo) IsPointer() bool {
	_, ok := info.Type.Underlying().(*types.Pointer)
	return ok
}

// IsArray returns true if the type represented by ASTNodeTypeInfo is an array type.
func (info *ASTNodeTypeInfo) IsArray() bool {
	_, ok := info.Type.Underlying().(*types.Slice)
	return ok
}

// IsStruct returns true if the type represented by ASTNodeTypeInfo is a struct type.
func (info *ASTNodeTypeInfo) IsStruct() bool {
	_, ok := info.Type.Underlying().(*types.Struct)
	return ok
}

// TypeExpr returns the string for referencing the type of AST node in generated Go code.
func (info *ASTNodeTypeInfo) TypeExpr() string {
	if info.IsPointer() {
		return "*ast." + info.Name
	}
	return "ast." + info.Name
}

// VisitorInstance returns the name of the visitor instance for visiting the node.
func (info *ASTNodeTypeInfo) VisitorInstance() string {
	return strings.ToLower(info.Name[:1]) + info.Name[1:] + "Visitor"
}

// NilCheck generate tests for checking whether
func (info *ASTNodeTypeInfo) NilCheck(field string) string {
	if info.IsAbstract() || info.IsPointer() {
		return field + " != nil"
	} else if info.IsArray() {
		return "len(" + field + ") != 0"
	} else {
		switch info.Name {
		case "Name":
			return "!" + field + ".IsNil()"
		case "NamedType":
			return "!" + field + ".Name.IsNil()"
		default:
			panic(fmt.Sprintf(`unhandled nil check for optional field "%s" with type %s`,
				field, info.Name))
		}
	}
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

		// Create type info object in advance.
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

func genVisitorInterfaces(w io.Writer) {
	tmpl, err := template.New("visitor-interface").Parse(`
// {{.Node}}Visitor implements visiting function for {{.Node}}.
type {{.Node}}Visitor interface {
	Enter{{.Node}}(node {{.Type}}, info *Info) Result
	Leave{{.Node}}(node {{.Type}}, info *Info) Result
}

// {{.Node}}VisitorFunc is an adapter to help define a {{.Node}}Visitor from a function which
// specifies action when entering a node.
type {{.Node}}VisitorFunc func(node {{.Type}}, info *Info) Result

var _ {{.Node}}Visitor = ({{.Node}}VisitorFunc)(nil)

// Enter{{.Node}} implements {{.Node}}Visitor by calling f(node, info).
func (f {{.Node}}VisitorFunc) Enter{{.Node}}(node {{.Type}}, info *Info) Result {
	return f(node, info)
}

// Leave{{.Node}} implements {{.Node}}Visitor which takes no actions.
func ({{.Node}}VisitorFunc) Leave{{.Node}}(node {{.Type}}, info *Info) Result {
	return Continue
}

// {{.Node}}VisitorFuncs is an adapter to help define a {{.Node}}Visitor from functions.
type {{.Node}}VisitorFuncs struct {
	Enter func(node {{.Type}}, info *Info) Result
	Leave func(node {{.Type}}, info *Info) Result
}

var _ {{.Node}}Visitor = (*{{.Node}}VisitorFuncs)(nil)

// Enter{{.Node}} implements {{.Node}}Visitor by calling f.Enter.
func (f *{{.Node}}VisitorFuncs) Enter{{.Node}}(node {{.Type}}, info *Info) Result {
	return f.Enter(node, info)
}

// Leave{{.Node}} implements {{.Node}}Visitor by calling f.Leave.
func (f *{{.Node}}VisitorFuncs) Leave{{.Node}}(node {{.Type}}, info *Info) Result {
	return f.Leave(node, info)
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

func genDefaultVisitors(w io.Writer) {
	fmt.Fprintf(w, `
// defaultVisitor takes no action when visiting a node.
type defaultVisitor uint

// Instance of default visitor that is shared among all newly created visitor.
const defaultVisitorInstance defaultVisitor = 0
`)

	fmt.Fprintf(w, `
var (`)

	for _, node := range astNodes {
		fmt.Fprintf(w, `
	_ %-26s = defaultVisitorInstance`, node.Name+"Visitor")
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, ")")

	for _, node := range astNodes {
		fmt.Fprintf(w, `
func (defaultVisitor) Enter%s(node %s, info *Info) Result {
	return Continue
}`, node.Name, node.TypeExpr())
	}

	for _, node := range astNodes {
		fmt.Fprintf(w, `
func (defaultVisitor) Leave%s(node %s, info *Info) Result {
	return Continue
}`, node.Name, node.TypeExpr())
	}

	fmt.Fprintln(w)
}

func genVisitor(w io.Writer) {
	fmt.Fprintf(w, `
// A Visitor is provided to visit an AST, it contains the collection of visitor to be executed
// during the visitor's traversal.
type Visitor interface {`)

	for _, node := range astNodes {
		// Use "Visit" instead of "VisitNode".
		if node.Name == "Node" {
			fmt.Fprintf(w, `
	Visit(node ast.Node, ctx interface{})`)
		} else {
			fmt.Fprintf(w, `
	Visit%s(node %s, ctx interface{})`, node.Name, node.TypeExpr())
		}
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "}")
}

func genVisitorImpl(w io.Writer) {
	fmt.Fprintf(w, `
// A visitor is the actual implementation of Visitor.
type visitor struct {`)

	for _, node := range astNodes {
		if !node.IsAbstract() {
			fmt.Fprintf(w, `
	%s %19sVisitor`, node.VisitorInstance(), node.Name)
		}
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "}")

	// VisitorNode initializes an Info object and calls visitNodeInternal.
	for _, node := range astNodes {
		if node.Name == "Node" {
			fmt.Fprintf(w, `
func (v *visitor) Visit(node ast.Node, ctx interface{}) {`)
		} else {
			fmt.Fprintf(w, `
func (v *visitor) Visit%s(node %s, ctx interface{}) {`, node.Name, node.TypeExpr())
		}
		fmt.Fprintf(w, `
	v.visit%sInternal(node, &Info{
		context: ctx,
	})
}`, node.Name)
	}

	fmt.Fprintf(w, `
func newVisitor() *visitor {
	return &visitor{`)

	for _, node := range astNodes {
		if !node.IsAbstract() {
			fmt.Fprintf(w, `
		%-27s defaultVisitorInstance,`, node.VisitorInstance()+":")
		}
	}

	fmt.Fprintf(w, `
	}
}`)

	for _, node := range astNodes {
		fmt.Fprintf(w, `
func (v *visitor) visit%sInternal(node %s, info *Info) Result {`, node.Name, node.TypeExpr())

		// Enter a node.
		if !node.IsAbstract() {
			fmt.Fprintf(w, `
	result := v.%s.Enter%s(node, info)
	if result != Continue {
		return result
	}`, node.VisitorInstance(), node.Name)

			fmt.Fprintln(w)
		}

		// Special case: ListValue
		if node.Name == "ListValue" {
			fmt.Fprintf(w, `
	childInfo := info.withParent(node)
	for _, value := range node.Values() {
		if result := v.visitValueInternal(value, childInfo); result == Break {
			return result
		}
	}
`)
		}

		// Special case: ObjectValue
		if node.Name == "ObjectValue" {
			fmt.Fprintf(w, `
	childInfo := info.withParent(node)
	for _, field := range node.Fields() {
		if result := v.visitObjectFieldInternal(field, childInfo); result == Break {
			return result
		}
	}
`)
		}

		// Special case: NonNullType
		if node.Name == "NonNullType" {
			fmt.Fprintf(w, `
	childInfo := info.withParent(node)
	switch t := node.Type.(type) {
	case ast.NamedType:
		result = v.visitNamedTypeInternal(t, childInfo)
	case ast.ListType:
		result = v.visitListTypeInternal(t, childInfo)
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
	switch node := node.(type) {`)
				for _, possibleType := range node.Children {
					fmt.Fprintf(w, `
	case %s:
		return v.visit%sInternal(node, info)`, possibleType.TypeExpr(), possibleType.Name)
				}

				fmt.Fprintf(w, `
	default:
		panic(fmt.Sprintf("unexpected node type %%T when visiting %s", node))
	}`, node.Name)

			} else if node.IsArray() {
				elementTypeInfo := node.Children[0]
				fmt.Fprintf(w, `
	childInfo := info.withParent(node)
	for _, childNode := range node {
		if result := v.visit%sInternal(childNode, childInfo); result == Break {
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

				fmt.Fprintf(w, `
	childInfo := info.withParent(node)`)

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
		if result := v.visit%sInternal(node.%s, childInfo); result == Break {
			return result
		}
	}`, fieldTypeInfo.NilCheck("node."+field.Name()), fieldTypeInfo.Name, field.Name())
					} else {
						fmt.Fprintf(w, `
	if result := v.visit%sInternal(node.%s, childInfo); result == Break {
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
		if !node.IsAbstract() {
			fmt.Fprintf(w, `
	return v.%s.Leave%s(node, info)
`, node.VisitorInstance(), node.Name)
		}

		fmt.Fprintln(w, `}`)
	}
}

func genVisitorBuilder(w io.Writer) {
	fmt.Fprintf(w, `
// Builder creates visitor.
type Builder struct {
	v *visitor
}

// NewBuilder creates a builder to builds a visitor.
func NewBuilder() Builder {
	return Builder{
		v: newVisitor(),
	}
}

// Build returns the visitor that is being built. Builder should not be used on return.
func (builder Builder) Build() Visitor {
	return builder.v
}
`)

	for _, node := range astNodes {
		fmt.Fprintf(w, `
// Visit%sWith set a visitor for %s. Note that this will override the one that is set previously
// silently.
func (builder Builder) Visit%sWith(visitor %sVisitor) Builder {`,
			node.Name, node.Name, node.Name, node.Name)
		if node.IsAbstract() {
			possibleTypes := make([]*ASTNodeTypeInfo, len(node.Children))
			copy(possibleTypes, node.Children)

			for len(possibleTypes) > 0 {
				possibleType := possibleTypes[len(possibleTypes)-1]
				possibleTypes = possibleTypes[:len(possibleTypes)-1]
				if possibleType.IsAbstract() {
					possibleTypes = append(possibleTypes, possibleType.Children...)
					continue
				}

				fmt.Fprintf(w, `
	if builder.v.%s == defaultVisitorInstance {
		builder.v.%s = &%sVisitorFuncs{
			Enter: func(node %s, info *Info) Result {
				return visitor.Enter%s(node, info)
			},
			Leave: func(node %s, info *Info) Result {
				return visitor.Leave%s(node, info)
			},
		}
	}`, possibleType.VisitorInstance(), possibleType.VisitorInstance(), possibleType.Name,
					possibleType.TypeExpr(), node.Name, possibleType.TypeExpr(), node.Name)
			}
		} else {
			fmt.Fprintf(w, `
	builder.v.%s = visitor`, node.VisitorInstance())
		}
		fmt.Fprintln(w, `
	return builder
}`)
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
	genVisitorInterfaces(w)
	genDefaultVisitors(w)
	genVisitor(w)
	genVisitorImpl(w)
	genVisitorBuilder(w)
}
