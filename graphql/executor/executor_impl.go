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
	goctx "context"
	"fmt"
	"reflect"

	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/graphql/ast"
	values "github.com/botobag/artemis/graphql/internal/value"
)

// Common includes common functions shared between Executor implementations. You might find it
// useful when implementing custom Executor.
type Common struct{}

// BuildRootResultNode returns a node to start execution of an operation.
func (executor Common) BuildRootResultNode(context *ExecutionContext) (*ResultNode, error) {
	rootType := context.Operation().RootType()
	// Root node is a special node which behaves like a field with nil parent and definition.
	rootNode := &ExecutionNode{}
	rootResult := &ResultNode{
		Kind: ResultKindUnresolved,
		Value: &UnresolvedResultValue{
			ExecutionNode: rootNode,
			ParentType:    rootType,
			Source:        context.RootValue(),
		},
	}

	err := executor.completeObjectValue(context, rootType, &ResolveInfo{
		ExecutionContext: context,
		ExecutionNode:    rootNode,
		ResultNode:       rootResult,
		ParentType:       rootType,
		ctx:              goctx.Background(),
	}, context.RootValue())
	if err != nil {
		return nil, err
	}

	return rootResult, nil
}

// Given a selectionSet, adds all of the fields in that selection to the passed in map of fields,
// and returns it at the end.
//
// CollectFields requires the "runtime type" of an object. For a field which returns an Interface or
// Union type, the "runtime type" will be the actual Object type returned by that field.
func (executor Common) collectFields(
	context *ExecutionContext,
	node *ExecutionNode,
	runtimeType *graphql.Object) ([]*ExecutionNode, error) {
	// Look up nodes for the Selection Set with the given runtime type in node's child nodes.
	var childNodes []*ExecutionNode

	if node.Children == nil {
		// Initialize the children node map.
		node.Children = map[*graphql.Object][]*ExecutionNode{}
	} else {
		// See whether we have built one before.
		childNodes = node.Children[runtimeType]
	}

	if childNodes == nil {
		// Load selection set into ExecutionNode's.
		var err error
		childNodes, err = executor.buildChildExecutionNodesForSelectionSet(context, node, runtimeType)
		if err != nil {
			return nil, err
		}
	}

	// Store the result before return.
	node.Children[runtimeType] = childNodes

	return childNodes, nil
}

// Build ExecutionNode's for the selection set of given node.
func (executor Common) buildChildExecutionNodesForSelectionSet(
	context *ExecutionContext,
	parentNode *ExecutionNode,
	runtimeType *graphql.Object) ([]*ExecutionNode, error) {
	// Boolean set to prevent named fragment to be applied twice or more in a selection set.
	visitedFragmentNames := map[string]bool{}

	// Map field response key to its corresponding node; This is used to group field definitions when
	// two fields corresponding to the same response key was requested in the selection set.
	fields := map[string]*ExecutionNode{}

	// The result nodes
	childNodes := []*ExecutionNode{}

	type taskData struct {
		// The Selection Set that is being processed into childNodes
		selectionSet ast.SelectionSet

		// The index of Selection to be resumed when restarting the task.
		selectionIndex int
	}

	// Stack contains task to be processed.
	var stack []taskData

	// Initialize the stack. Find the selection sets in parentNode to processed.
	if parentNode.IsRoot() {
		stack = []taskData{
			{context.Operation().Definition().SelectionSet, 0},
		}
	} else {
		definitions := parentNode.Definitions
		numDefinitions := len(definitions)
		stack = make([]taskData, numDefinitions)
		// stack is LIFO so place the selection sets in reverse order.
		for i, definition := range definitions {
			stack[numDefinitions-i-1].selectionSet = definition.SelectionSet
		}
	}

	for len(stack) > 0 {
		task := &stack[len(stack)-1]

		selectionSet := task.selectionSet
		numSelections := len(selectionSet)
		interrupted := false

		for task.selectionIndex < numSelections && !interrupted {
			selection := selectionSet[task.selectionIndex]
			task.selectionIndex++
			if task.selectionIndex >= numSelections {
				// No more selections in the selection set. Pop it from the stack.
				stack = stack[:len(stack)-1]
			}

			// Check @skip and @include.
			shouldInclude, err := executor.shouldIncludeNode(context, selection)
			if err != nil {
				return nil, err
			} else if !shouldInclude {
				continue
			}

			switch selection := selection.(type) {
			case *ast.Field:
				// Find existing fields.
				name := selection.ResponseKey()
				field := fields[name]
				if field != nil {
					// The field with the same name has been added to the selection set before. Append the
					// definition to the same node to coalesce their selection sets.
					field.Definitions = append(field.Definitions, selection)
				} else {
					// Find corresponding runtime Field definition in current schema.
					fieldDef := executor.findFieldDef(
						context.Operation().Schema(),
						runtimeType,
						selection.Name.Value())
					if fieldDef == nil {
						// Schema doesn't contains the field. Note that we should skip the field without an
						// error as per specification.
						//
						// Reference: 3.c. in https://facebook.github.io/graphql/June2018/#ExecuteSelectionSet().
						break
					}

					// Get argument values.
					arguments, err := values.ArgumentValues(fieldDef, selection, context.VariableValues())
					if err != nil {
						return nil, err
					}

					// Build a node.
					field = &ExecutionNode{
						Parent:         parentNode,
						Definitions:    []*ast.Field{selection},
						Field:          fieldDef,
						ArgumentValues: arguments,
					}

					// Add to result.
					childNodes = append(childNodes, field)

					// Insert a map entry.
					fields[name] = field
				}

			case *ast.InlineFragment:
				// Apply fragment only if the runtime type satisfied the type condition.
				if selection.HasTypeCondition() {
					if !executor.doesTypeConditionSatisfy(context, selection.TypeCondition, runtimeType) {
						break
					}
				}

				// Push a task to process selection set in the fragment.
				stack = append(stack, taskData{
					selectionSet: selection.SelectionSet,
				})

				// Interrupt current loop to start processing the selection set in the fragment.
				// Specification requires fields to be sorted in DFS order.
				interrupted = true

			case *ast.FragmentSpread:
				fragmentName := selection.Name.Value()
				if visited := visitedFragmentNames[fragmentName]; visited {
					break
				}
				visitedFragmentNames[fragmentName] = true

				// Find fragment definition to get type condition and selection set.
				fragmentDef := context.Operation().FragmentDef(fragmentName)
				if fragmentDef == nil {
					break
				}

				if !executor.doesTypeConditionSatisfy(context, fragmentDef.TypeCondition, runtimeType) {
					break
				}

				// Push a task to process selection set in the fragment.
				stack = append(stack, taskData{
					selectionSet: fragmentDef.SelectionSet,
				})

				interrupted = true
			}
		}
	} // for len(stack) > 0 {

	return childNodes, nil
}

// ExecuteNode implements "Executing Fields" [0]. It resolves the field on the given source object.
// In particular, this figures out the value that the field returns by calling its resolve function,
// then calls completeValue to complete promises, serialize scalars, or execute the
// sub-selection-set for objects.
//
// [0]: https://facebook.github.io/graphql/June2018/#sec-Executing-Fields
func (executor Common) ExecuteNode(
	ctx goctx.Context,
	context *ExecutionContext,
	result *ResultNode) graphql.Errors {

	unresolvedValue := result.UnresolvedValue()
	node := unresolvedValue.ExecutionNode
	parentType := unresolvedValue.ParentType
	source := unresolvedValue.Source

	// If parent becomes a "Invalid Nil" result, one of our sibling or decensant nodes came before us
	// and failed the execution. No need to proceed with execution for this node because the result
	// will always discarded.
	if result.Parent != nil && result.Parent.IsNil() {
		return graphql.NoErrors()
	}

	info := &ResolveInfo{
		ExecutionContext: context,
		ExecutionNode:    node,
		ResultNode:       result,
		ParentType:       parentType,
		ctx:              ctx,
	}

	// Get the field resolver.
	field := node.Field
	resolver := field.Resolver()
	if resolver == nil {
		resolver = context.Operation().DefaultFieldResolver()
	}

	// Call resolver to resolve the field value.
	value, err := resolver.Resolve(ctx, source, info)
	if err != nil {
		return graphql.ErrorsOf(executor.handleFieldError(err, result, node))
	}

	return executor.completeValue(context, field.Type(), info, value)
}

func (executor Common) handleFieldError(err error, result *ResultNode, node *ExecutionNode) error {
	// Attach location info.
	locations := make([]graphql.ErrorLocation, len(node.Definitions))
	for i := range node.Definitions {
		locations[i] = graphql.ErrorLocationOfASTNode(node.Definitions[i])
	}

	// Compute response path.
	path := result.Path()

	// Wrap it as a graphql.Error to ensure a consistent Error interface.
	e, ok := err.(*graphql.Error)
	if !ok {
		e = graphql.NewError(err.Error(), locations, path).(*graphql.Error)
	} else {
		e.Locations = locations
		e.Path = path
	}

	// Set result value to a nil value.
	result.Kind = ResultKindNil
	result.Value = nil

	// Impelement "Errors and Non-Nullability". Propagate the field error until a nullable field was
	// encountered.
	//
	// Reference: https://facebook.github.io/graphql/June2018/#sec-Errors-and-Non-Nullability
	for result != nil && result.IsNonNull() {
		result = result.Parent
		result.Kind = ResultKindNil
		result.Value = nil
	}

	return e
}

// completeValue implements "Value Completion" [0]. It ensures the value resolved from the field
// resolver adheres to the expected return type.
//
// [0]: https://facebook.github.io/graphql/June2018/#sec-Value-Completion
func (executor Common) completeValue(
	context *ExecutionContext,
	returnType graphql.Type,
	info *ResolveInfo,
	value interface{}) graphql.Errors {

	if wrappingType, isWrappingType := returnType.(graphql.WrappingType); isWrappingType {
		return executor.completeWrappingValue(context, wrappingType, info, value)
	}

	err := executor.completeNonWrappingValue(context, returnType, info, value)
	if err != nil {
		return graphql.ErrorsOf(err)
	}

	return graphql.NoErrors()
}

// completeWrappingValue completes value for NonNull and List type.
func (executor Common) completeWrappingValue(
	context *ExecutionContext,
	returnType graphql.WrappingType,
	info *ResolveInfo,
	value interface{}) graphql.Errors {
	var errs graphql.Errors

	// Resolvers can return error to signify failure. See https://github.com/graphql/graphql-js/commit/f62c0a25.
	if err, ok := value.(*graphql.Error); ok && err != nil {
		return graphql.ErrorsOf(
			executor.handleFieldError(err, info.ResultNode, info.ExecutionNode))
	}

	type taskData struct {
		returnType graphql.WrappingType
		result     *ResultNode
		value      interface{}
	}
	queue := []taskData{
		{
			returnType: returnType,
			result:     info.ResultNode,
			value:      value,
		},
	}
	node := info.ExecutionNode
	field := node.Field

	for len(queue) > 0 {
		var task *taskData
		// Pop one task from queue.
		task, queue = &queue[0], queue[1:]

		var returnType graphql.Type = task.returnType
		result := task.result
		value := task.value

		// If the parent was resolved to nil, stop processing this node.
		if result.Parent.IsNil() {
			continue
		}

		// Handle non-null.
		nonNullType, isNonNullType := returnType.(*graphql.NonNull)

		if isNonNullType {
			// For non-null type, continue on its unwrapped type.
			returnType = nonNullType.InnerType()
		}

		// Handle nil value.
		if values.IsNullish(value) {
			// Check for non-nullability.
			if isNonNullType {
				err := executor.handleFieldError(
					graphql.NewError(fmt.Sprintf("Cannot return null for non-nullable field %v.%s.",
						info.ParentType.Name(), node.Field.Name())),
					result, node)
				errs.Append(err)
			} else {
				// Resolve the value to nil without error.
				result.Kind = ResultKindNil
				result.Value = nil
			}

			// Continue to the next value.
			continue
		} // if values.IsNullish(value)

		listType, isListType := returnType.(*graphql.List)
		if !isListType {
			info.ResultNode = result
			err := executor.completeNonWrappingValue(context, returnType, info, value)
			if err != nil {
				errs.Append(err)
			}
			continue
		}

		// Complete a list value by completing each item in the list with the inner type.
		v := reflect.ValueOf(value)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}

		if v.Kind() != reflect.Array && v.Kind() != reflect.Slice {
			err := executor.handleFieldError(
				graphql.NewError(
					fmt.Sprintf("Expected Iterable, but did not find one for field %s.%s.",
						info.ParentType.Name(), field.Name())),
				result, node)
			errs.Append(err)
			continue
		}

		elementType := listType.ElementType()
		elementWrappingType, isWrappingElementType := elementType.(graphql.WrappingType)

		// Setup result nodes for elements.
		numElements := v.Len()
		resultNodes := make([]ResultNode, numElements)

		// Set child results to reject nil value if it is unwrapped from a non-null type.
		if isNonNullType {
			for i := range resultNodes {
				resultNodes[i].SetIsNonNull()
			}
		}

		// Complete result.
		result.Kind = ResultKindList
		result.Value = resultNodes

		if isWrappingElementType {
			for i := range resultNodes {
				resultNode := &resultNodes[i]
				resultNode.Parent = result
				queue = append(queue, taskData{
					returnType: elementWrappingType,
					result:     resultNode,
					value:      v.Index(i).Interface(),
				})
			}
		} else {
			for i := range resultNodes {
				resultNode := &resultNodes[i]
				resultNode.Parent = result
				info.ResultNode = resultNode
				value := v.Index(i).Interface()
				err := executor.completeNonWrappingValue(context, elementType, info, value)
				if err != nil {
					errs.Append(err)
				}
			}
		}
	}

	return errs
}

func (executor Common) completeNonWrappingValue(
	context *ExecutionContext,
	returnType graphql.Type,
	info *ResolveInfo,
	value interface{}) error {

	// Non-null and List type should already be handled in completeWrappingValue.
	result := info.ResultNode

	// Check for nullish.
	if values.IsNullish(value) {
		result.Value = nil
		result.Kind = ResultKindNil
		return nil
	}

	// Resolvers can return error to signify failure. See https://github.com/graphql/graphql-js/commit/f62c0a25.
	if err, ok := value.(*graphql.Error); ok {
		return executor.handleFieldError(err, result, info.ExecutionNode)
	}

	switch returnType := returnType.(type) {
	// Scalar and Enum.
	case graphql.LeafType:
		return executor.completeLeafValue(context, returnType, info, value)

	case *graphql.Object:
		return executor.completeObjectValue(context, returnType, info, value)

	// Union and Interface
	case graphql.AbstractType:
		return executor.completeAbstractValue(context, returnType, info, value)
	}

	return executor.handleFieldError(
		graphql.NewError(fmt.Sprintf(`Cannot complete value of unexpected type "%v".`, returnType)),
		result, info.ExecutionNode)
}

func (executor Common) completeLeafValue(
	context *ExecutionContext,
	returnType graphql.LeafType,
	info *ResolveInfo,
	value interface{}) error {

	result := info.ResultNode
	coercedValue, err := returnType.CoerceResultValue(value)
	if err != nil {
		// See comments in graphql.NewCoercionError for the rules of handling error.
		if e, ok := err.(*graphql.Error); !ok || e.Kind != graphql.ErrKindCoercion {
			// Wrap the error in our own.
			err = graphql.NewDefaultResultCoercionError(returnType.Name(), value, err)
		}
		return executor.handleFieldError(err, result, info.ExecutionNode)
	}

	// Setup result and return.
	result.Kind = ResultKindLeaf
	result.Value = coercedValue
	return nil
}

func (executor Common) completeObjectValue(
	context *ExecutionContext,
	returnType *graphql.Object,
	info *ResolveInfo,
	value interface{}) error {

	node := info.ExecutionNode
	result := info.ResultNode

	// Collect fields in the selection set.
	childNodes, err := executor.collectFields(context, node, returnType)
	if err != nil {
		return executor.handleFieldError(err, result, node)
	}

	// Setup an unresolved ResultNode for each child ExecutionNode.
	numChildNodes := len(childNodes)
	fieldResults := make([]ResultNode, numChildNodes)
	for i := 0; i < numChildNodes; i++ {
		fieldResult := &fieldResults[i]
		childNode := childNodes[i]
		fieldResult.Parent = result
		fieldResult.Kind = ResultKindUnresolved
		fieldResult.Value = &UnresolvedResultValue{
			ExecutionNode: childNode,
			ParentType:    info.ParentType,
			Source:        value,
		}
		// Set the flag so field can reject nil value on error.
		if graphql.IsNonNullType(childNode.Field.Type()) {
			fieldResult.SetIsNonNull()
		}
	}

	// Setup result.
	result.Kind = ResultKindObject
	result.Value = &ObjectResultValue{
		ExecutionNodes: childNodes,
		FieldValues:    fieldResults,
	}

	return nil
}

func (executor Common) completeAbstractValue(
	context *ExecutionContext,
	returnType graphql.AbstractType,
	info *ResolveInfo,
	value interface{}) error {
	panic("unimplemented")
}

// ExecutionQueue manages ExecutionNode's that are waiting for processing.
type ExecutionQueue interface {
	// Push adds a ResultNode to the queue for processing. The given node must be an unresolved result
	// (i.e., node.IsUnresolved() returns true.)
	Push(node *ResultNode)
}

// EnqueueChildNodes finds any unresolved child nodes of the given node and adds them to queue.
func (executor Common) EnqueueChildNodes(queue ExecutionQueue, node *ResultNode) {
	stack := []*ResultNode{node}
	for len(stack) > 0 {
		node, stack = stack[len(stack)-1], stack[:len(stack)-1]

		var childNodes []ResultNode
		if node.IsUnresolved() {
			queue.Push(node)
		} else if node.IsList() {
			childNodes = node.ListValue()
		} else if node.IsObject() {
			childNodes = node.ObjectValue().FieldValues
		}

		for i := len(childNodes) - 1; i >= 0; i-- {
			node := &childNodes[i]
			if node.IsUnresolved() {
				queue.Push(node)
			} else if node.IsList() || node.IsObject() {
				stack = append(stack, node)
			}
			// Skip nodes with other kinds. They don't have child nodes.
		}
	}
}

// Determines if a field should be included based on the @include and @skip directives, where @skip
// has higher precedence than @include.
//
// Reference: https://facebook.github.io/graphql/June2018/#sec--include
func (executor Common) shouldIncludeNode(context *ExecutionContext, node ast.Selection) (bool, error) {
	// Neither @skip nor @include has precedence over the other. In the case that both the @skip and
	// @include directives are provided in on the same the field or fragment, it must be queried only
	// if the @skip condition is false and the @include condition is true. Stated conversely, the
	// field or fragment must not be queried if either the @skip condition is true or the @include
	// condition is false.
	skip, err := values.DirectiveValues(
		graphql.SkipDirective(), node.GetDirectives(), context.VariableValues())
	if err != nil {
		return false, err
	}
	shouldSkip := skip.Get("if")
	if shouldSkip != nil && shouldSkip.(bool) {
		return false, nil
	}

	include, err := values.DirectiveValues(
		graphql.IncludeDirective(), node.GetDirectives(), context.VariableValues())
	if err != nil {
		return false, err
	}
	shouldInclude := include.Get("if")
	if shouldInclude != nil && !shouldInclude.(bool) {
		return false, nil
	}

	return true, nil
}

// This method looks up the field on the given type definition. It has special casing for the two
// introspection fields, __schema and __typename. __typename is special because it can always be
// queried as a field, even in situations where no other fields are allowed, like on a Union.
// __schema could get automatically added to the query type, but that would require mutating type
// definitions, which would cause issues.
func (executor Common) findFieldDef(
	schema *graphql.Schema,
	parentType *graphql.Object,
	fieldName string) *graphql.Field {
	// TODO: Deal with special introspection fields.
	return parentType.Fields()[fieldName]
}

// Determines if a type condition is satisfied with the given type.
func (executor Common) doesTypeConditionSatisfy(
	context *ExecutionContext,
	typeCondition ast.NamedType,
	t graphql.Type) bool {
	schema := context.Operation().Schema()

	conditionalType := schema.TypeFromAST(typeCondition)
	if conditionalType == t {
		return true
	}

	if abstractType, ok := t.(graphql.AbstractType); ok {
		for _, possibleType := range schema.PossibleTypes(abstractType) {
			if possibleType == t {
				return true
			}
		}
	}

	return false
}
