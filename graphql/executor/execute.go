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

package executor

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"sync"
	"sync/atomic"

	"github.com/botobag/artemis/concurrent/future"
	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/graphql/ast"
	values "github.com/botobag/artemis/graphql/internal/value"
	"github.com/botobag/artemis/iterator"
	"github.com/botobag/artemis/jsonwriter"
)

// ExecutionResult contains result from running an Executor.
type ExecutionResult struct {
	Data   *ResultNode
	Errors graphql.Errors
}

// MarshalJSONTo writes the JSON encoding of result to the w. It makes use of jsonwriter
// implementation which offers better performance compared to Go's built-in encoding/json. Using
// this API to write result is preferred rather than encoding/json.Marshal.
func (result *ExecutionResult) MarshalJSONTo(w io.Writer) error {
	stream := jsonwriter.NewStream(w)
	stream.WriteValue(NewExecutionResultMarshaler(result))
	stream.WriteRawString("\n")
	return stream.Flush()
}

// MarshalJSON implements json.Marshaler interface for ExecutionResult.
func (result ExecutionResult) MarshalJSON() ([]byte, error) {
	return jsonwriter.Marshal(NewExecutionResultMarshaler(&result))
}

// Given a selectionSet, adds all of the fields in that selection to the passed in map of fields,
// and returns it at the end.
//
// CollectFields requires the "runtime type" of an object. For a field which returns an Interface or
// Union type, the "runtime type" will be the actual Object type returned by that field.
func collectFields(
	ctx *ExecutionContext,
	node *ExecutionNode,
	runtimeType graphql.Object) ([]*ExecutionNode, error) {
	// Look up nodes for the Selection Set with the given runtime type in node's child nodes.
	var childNodes []*ExecutionNode

	if node.Children == nil {
		// Initialize the children node map.
		node.Children = map[graphql.Object][]*ExecutionNode{}
	} else {
		// See whether we have built one before.
		childNodes = node.Children[runtimeType]
	}

	if childNodes == nil {
		// Load selection set into ExecutionNode's.
		var err error
		childNodes, err = buildChildExecutionNodesForSelectionSet(ctx, node, runtimeType)
		if err != nil {
			return nil, err
		}
	}

	// Store the result before return.
	node.Children[runtimeType] = childNodes

	return childNodes, nil
}

// Build ExecutionNode's for the selection set of given node.
func buildChildExecutionNodesForSelectionSet(
	ctx *ExecutionContext,
	parentNode *ExecutionNode,
	runtimeType graphql.Object) ([]*ExecutionNode, error) {
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

	// Initialize the stack. Find the selection sets in parentNode to process.
	if parentNode.IsRoot() {
		stack = []taskData{
			{ctx.Operation().Definition().SelectionSet, 0},
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
		var (
			data = &stack[len(stack)-1]

			selectionSet  = data.selectionSet
			numSelections = len(selectionSet)
			interrupted   = false
		)

		for data.selectionIndex < numSelections && !interrupted {
			selection := selectionSet[data.selectionIndex]
			data.selectionIndex++
			if data.selectionIndex >= numSelections {
				// No more selections in the selection set. Pop it from the stack.
				stack = stack[:len(stack)-1]
			}

			// Check @skip and @include.
			shouldInclude, err := shouldIncludeNode(ctx, selection)
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
					fieldDef := findFieldDef(
						ctx.Operation().Schema(),
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
					args, err := values.ArgumentValues(fieldDef, selection, ctx.VariableValues())
					if err != nil {
						return nil, err
					}

					// Build a node.
					field = &ExecutionNode{
						Parent:      parentNode,
						Definitions: []*ast.Field{selection},
						Field:       fieldDef,
						Args:        args,
					}

					// Add to result.
					childNodes = append(childNodes, field)

					// Insert a map entry.
					fields[name] = field
				}

			case *ast.InlineFragment:
				// Apply fragment only if the runtime type satisfied the type condition.
				if selection.HasTypeCondition() {
					if !doesTypeConditionSatisfy(ctx, selection.TypeCondition, runtimeType) {
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
				fragmentDef := ctx.Operation().FragmentDef(fragmentName)
				if fragmentDef == nil {
					break
				}

				if !doesTypeConditionSatisfy(ctx, fragmentDef.TypeCondition, runtimeType) {
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

// Determines if a field should be included based on the @include and @skip directives, where @skip
// has higher precedence than @include.
//
// Reference: https://facebook.github.io/graphql/June2018/#sec--include
func shouldIncludeNode(ctx *ExecutionContext, node ast.Selection) (bool, error) {
	// Neither @skip nor @include has precedence over the other. In the case that both the @skip and
	// @include directives are provided in on the same the field or fragment, it must be queried only
	// if the @skip condition is false and the @include condition is true. Stated conversely, the
	// field or fragment must not be queried if either the @skip condition is true or the @include
	// condition is false.
	skip, err := values.DirectiveValues(
		graphql.SkipDirective(), node.GetDirectives(), ctx.VariableValues())
	if err != nil {
		return false, err
	}
	shouldSkip := skip.Get("if")
	if shouldSkip != nil && shouldSkip.(bool) {
		return false, nil
	}

	include, err := values.DirectiveValues(
		graphql.IncludeDirective(), node.GetDirectives(), ctx.VariableValues())
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
func findFieldDef(
	schema graphql.Schema,
	parentType graphql.Object,
	fieldName string) graphql.Field {
	if schema.Query() == parentType {
		// Deal with special introspection fields.
		if fieldName == schemaMetaFieldName {
			return schemaMetaField{}
		} else if fieldName == typeMetaFieldName {
			return typeMetaField{}
		}
	}
	return parentType.Fields()[fieldName]
}

// Determines if a type condition is satisfied with the given type.
func doesTypeConditionSatisfy(
	ctx *ExecutionContext,
	typeCondition ast.NamedType,
	t graphql.Object) bool {
	schema := ctx.Operation().Schema()

	conditionalType := schema.TypeFromAST(typeCondition)
	if conditionalType == t {
		return true
	}

	if abstractType, ok := conditionalType.(graphql.AbstractType); ok {
		possibleTypes := schema.PossibleTypes(abstractType)
		return possibleTypes.Contains(t)
	}

	return false
}

func collectAndDispatchRootTasks(ctx *ExecutionContext, executor executor) (*ResultNode, error) {
	rootType := ctx.Operation().RootType()
	// Root node is a special node which behaves like a field with nil parent and definition.
	rootNode := &ExecutionNode{
		Parent:      nil,
		Definitions: nil,
	}

	// Collect fields in the top-level selection set.
	nodes, err := collectFields(ctx, rootNode, rootType)
	if err != nil {
		return nil, err
	}

	// Allocate result node.
	result := &ResultNode{}

	// Create tasks for executing root nodes.
	dispatchTasksForObject(
		ctx,
		executor,
		result,
		nodes,
		ctx.RootValue())

	return result, nil
}

// Dispatch tasks for evaluating an object value comprised of the fields specified in childNodes.
// Return the ResultNode that contains
func dispatchTasksForObject(
	ctx *ExecutionContext,
	executor executor,
	result *ResultNode,
	childNodes []*ExecutionNode,
	value interface{}) {

	numChildNodes := len(childNodes)

	// Allocate ResultNode's for each nodes.
	nodeResults := make([]ResultNode, numChildNodes)

	// Setup result value.
	result.Kind = ResultKindObject
	result.Value = &ObjectResultValue{
		ExecutionNodes: childNodes,
		FieldValues:    nodeResults,
	}

	// Create tasks to resolve object fields.
	for i := 0; i < numChildNodes; i++ {
		nodeResult := &nodeResults[i]
		nodeResult.Parent = result
		childNode := childNodes[i]

		// Set the flag so field can reject nil value on error.
		if graphql.IsNonNullType(childNode.Field.Type()) {
			nodeResult.SetToRejectNull()
		}

		// Create a task and dispatch it with given dispatcher.
		task := newExecuteNodeTask(executor, ctx, childNode, nodeResult, value)
		executor.Dispatch(task)
	}
}

//===----------------------------------------------------------------------------------------====//
// ExecuteNodeTask
//===----------------------------------------------------------------------------------------====//

var executeNodeTaskFreeList = sync.Pool{
	New: func() interface{} {
		return &ExecuteNodeTask{}
	},
}

func newExecuteNodeTask(
	executor executor,
	ctx *ExecutionContext,
	node *ExecutionNode,
	result *ResultNode,
	source interface{},
) *ExecuteNodeTask {

	// Find one from the free list.
	task := executeNodeTaskFreeList.Get().(*ExecuteNodeTask)
	task.executor = executor
	task.ctx = ctx
	task.node = node
	task.result = result
	task.source = source
	// Initialze reference count to 1.
	task.refCount = 1

	return task
}

// ExecuteNodeTask executes a field (represented by ExecutionNode). It is scheduled and is run by
// an executor.
//
// ExecuteNodeTask is a temporary object used extensively during execution. Its allocation is
// managed by a sync.Pool (i.e., executeNodeTaskFreeList) to improve the allocation rate. A field
// "refCount" is added to track the number of references to this task object. Once the count reaches
// 0, the task is put back to the free list automatically.
type ExecuteNodeTask struct {
	// Executor that runs this task
	executor executor

	// Context for execution
	ctx *ExecutionContext

	// The node to evaluate
	node *ExecutionNode

	// The ResultNode for writing the field value. It is allocated by the one that prepares the
	// ExecuteNodeTask for execution.
	result *ResultNode

	// Source value which is passed to the field resolver; This is the field value of the parent.
	source interface{}

	// Track the number of references to this object. See retain and release.
	refCount int64
}

// retain increment the reference count of the task.
func (task *ExecuteNodeTask) retain() *ExecuteNodeTask {
	atomic.AddInt64(&task.refCount, 1)
	return task
}

// release decrement the reference count of the task. If the count reaches the task is considered
// unused (and should not be used thereafter) and will be put to the free list for later reuse by
// others (for another task).
func (task *ExecuteNodeTask) release() {
	if atomic.AddInt64(&task.refCount, -1) == 0 {
		executeNodeTaskFreeList.Put(task)
	}
}

// run implements Task. It executes the task to value for the field corresponding to the
// ExecutionNode. The execution result is written to the task.result and errors are added to
// executor (via task.executor.AppendErrors) so nothing is returned from this method.
func (task *ExecuteNodeTask) run() {
	var (
		ctx    = task.ctx
		node   = task.node
		result = task.result
		field  = node.Field
	)

	// Get field resolver to execute.
	resolver := field.Resolver()
	if resolver == nil {
		resolver = ctx.Operation().DefaultFieldResolver()
	}

	// Execute resolver to retrieve the field value
	value, err := resolver.Resolve(ctx.Context(), task.source, task.newResolveInfoFor(result))
	if err != nil {
		task.handleNodeError(err, result)
		task.release()
		return
	}

	// Complete subfields with value.
	task.completeValue(field.Type(), task.result, value)

	// Decrement reference count.
	task.release()

	return
}

// handleNodeError first creates a graphql.Error for an error value (which includes additional
// information such as field location) to be included in the GraphQL response and then adds the
// error to the ctx (using ctx.AppendErrors) to indicate a failed field execution.
func (task *ExecuteNodeTask) handleNodeError(err error, result *ResultNode) {
	node := task.node

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

	// Append error to task.errs.
	task.executor.AppendError(e, result)
}

// completeValue implements "Value Completion" [0]. It ensures the value resolved from the field
// resolver adheres to the expected return type.
//
// [0]: https://facebook.github.io/graphql/June2018/#sec-Value-Completion
func (task *ExecuteNodeTask) completeValue(
	returnType graphql.Type,
	result *ResultNode,
	value interface{}) {

	if wrappingType, isWrappingType := returnType.(graphql.WrappingType); isWrappingType {
		task.completeWrappingValue(wrappingType, result, value)
	} else {
		task.completeNonWrappingValue(returnType, result, value)
	}
}

func (task *ExecuteNodeTask) completeValuePrologue(
	returnType graphql.Type,
	result *ResultNode,
	value interface{}) (completed bool) {

	// Resolvers can return error to signify failure. See https://github.com/graphql/graphql-js/commit/f62c0a25.
	if err, ok := value.(*graphql.Error); ok && err != nil {
		task.handleNodeError(err, result)
		return true
	}

	// Resolves can return a Future whose value is generated by an asynchronous computation and may
	// not be ready yet. Dispatch a task to poll its result.
	if value, ok := value.(future.Future); ok {
		task.executor.Dispatch(&AsyncValueTask{
			// Increment the reference count because the task is now referenced by the AsyncValueTask.
			nodeTask:        task.retain(),
			dataLoaderCycle: task.executor.DataLoaderCycle(),
			returnType:      returnType,
			result:          result,
			value:           value,
		})
		return true
	}

	return false
}

// completeWrappingValue completes value for NonNull and List type.
func (task *ExecuteNodeTask) completeWrappingValue(
	returnType graphql.WrappingType,
	result *ResultNode,
	value interface{}) {

	if task.completeValuePrologue(returnType, result, value) {
		return
	}

	type ValueNode struct {
		returnType graphql.WrappingType
		result     *ResultNode
		value      interface{}
	}
	queue := []ValueNode{
		{
			returnType: returnType,
			result:     result,
			value:      value,
		},
	}

	for len(queue) > 0 {
		var valueNode *ValueNode
		// Pop one value node from queue.
		valueNode, queue = &queue[0], queue[1:]

		var (
			returnType graphql.Type = valueNode.returnType
			result                  = valueNode.result
			value                   = valueNode.value
		)

		// If the parent was resolved to nil, stop processing this node.
		if result.Parent.IsNil() {
			continue
		}

		// Handle non-null.
		nonNullType, isNonNullType := returnType.(graphql.NonNull)

		if isNonNullType {
			// For non-null type, continue on its unwrapped type.
			returnType = nonNullType.InnerType()
		}

		// Handle nil value.
		if values.IsNullish(value) {
			// Check for non-nullability.
			if isNonNullType {
				node := task.node
				task.handleNodeError(
					graphql.NewError(fmt.Sprintf("Cannot return null for non-nullable field %v.%s.",
						parentFieldType(task.ctx, node).Name(), node.Field.Name())),
					result)
			} else {
				// Resolve the value to nil without error.
				result.Kind = ResultKindNil
				result.Value = nil
			}

			// Continue to the next value.
			continue
		} // if values.IsNullish(value)

		listType, isListType := returnType.(graphql.List)
		if !isListType {
			task.completeNonWrappingValue(returnType, result, value)
			continue
		}

		// Complete a list value by completing each item in the list with the inner type.
		elementType := listType.ElementType()
		elementWrappingType, isWrappingElementType := elementType.(graphql.WrappingType)

		// The following code is a bit mess. If the value implements Iterable interfaces, we want to
		// enumerates the its item values via its custom iterator. Otherwise, we fallback to use
		// reflect.Value.Index to obtain item values. It's possible to implement an Iterable for the
		// fallback path and merges the control flow. But we choose to avoid indirection to minimize
		// overheads.
		//
		// Invariants for the former case (value implements Iterable interfaces):
		//
		//  - iterable != nil
		//  - v.IsValid() returns false
		//  - numElements is undefined
		//
		// Invariants for the latter case (use reflection to get item values):
		//
		//  - iterable is nil
		//  - v.Kind() returns reflect.Array or reflect.Slice
		//  - numElements is defined
		//
		// We check "iterable" to see which case being dealt with as needed.
		var (
			iterable    graphql.Iterable
			v           reflect.Value
			resultNodes ResultNodeList
			numElements int
		)

		// Setup iterable and v.
		if iterableValue, ok := value.(graphql.Iterable); ok {
			iterable = iterableValue
			if sizedIterable, ok := iterable.(graphql.SizedIterable); ok {
				// Make use of size hint to avoid list grow as possible.
				resultNodes = NewFixedSizeResultNodeList(sizedIterable.Size())
			} else {
				resultNodes = NewResultNodeList()
			}
		} else {
			v = reflect.ValueOf(value)
			if v.Kind() == reflect.Ptr {
				v = v.Elem()
			}

			if v.Kind() != reflect.Array && v.Kind() != reflect.Slice {
				node := task.node
				task.handleNodeError(
					graphql.NewError(
						fmt.Sprintf("Expected Iterable, but did not find one for field %s.%s.",
							parentFieldType(task.ctx, node).Name(), node.Field.Name())),
					result)
				continue
			}

			numElements = v.Len()
			resultNodes = NewFixedSizeResultNodeList(numElements)
		}

		// Complete result.
		result.Kind = ResultKindList
		result.Value = resultNodes

		// The following control flow diverage into 4 paths:
		//
		//	if iterable != nil {
		//		if isWrappingElementType {
		//			...
		//		} else {
		//			...
		//		}
		//	} else { // iterable == nil
		//		// v must be a valid reflect.Value.
		//		if isWrappingElementType {
		//			...
		//		} else {
		//			...
		//		}
		//	}
		if iterable != nil {
			// Invariants: iterable != nil
			iter := iterable.Iterator()

			for {
				value, err := iter.Next()
				if err == iterator.Done {
					break
				} else if err != nil {
					node := task.node
					task.handleNodeError(
						graphql.NewError(
							fmt.Sprintf("Error occurred while enumerates values in the list field %s.%s.",
								parentFieldType(task.ctx, node).Name(), node.Field.Name()), err),
						result)
					break
				} else {
					// Prepare resultNode for element.
					resultNode := resultNodes.EmplaceBack(result, !isNonNullType)

					if isWrappingElementType {
						queue = append(queue, ValueNode{
							returnType: elementWrappingType,
							result:     resultNode,
							value:      value,
						})
					} else { // !isWrappingElementType
						if !task.completeNonWrappingValue(elementType, resultNode, value) {
							// If the err causes the parent to be nil'ed, stop procsessing the remaining elements.
							if result.IsNil() {
								break
							}
						}
					}
				}
			}
		} else { // iterable == nil
			// Invariants: v.IsValid() and numElements is defined

			if isWrappingElementType {
				for i := 0; i < numElements; i++ {
					resultNode := resultNodes.EmplaceBack(result, !isNonNullType)
					queue = append(queue, ValueNode{
						returnType: elementWrappingType,
						result:     resultNode,
						value:      v.Index(i).Interface(),
					})
				}
			} else { // !isWrappingElementType
				for i := 0; i < numElements; i++ {
					resultNode := resultNodes.EmplaceBack(result, !isNonNullType)
					value := v.Index(i).Interface()
					if !task.completeNonWrappingValue(elementType, resultNode, value) {
						// If the err causes the parent to be nil'ed, stop procsessing the remaining elements.
						if result.IsNil() {
							break
						}
					}
				}
			} // if isWrappingElementType
		} // if iterable != nil
	}
}

func (task *ExecuteNodeTask) completeNonWrappingValue(
	returnType graphql.Type,
	result *ResultNode,
	value interface{}) (ok bool) {

	if task.completeValuePrologue(returnType, result, value) {
		return true
	}

	// Chack for nullish. Non-null type should already be handled by completeWrappingValue.
	if values.IsNullish(value) {
		result.Value = nil
		result.Kind = ResultKindNil
		return true
	}

	switch returnType := returnType.(type) {
	// Scalar and Enum
	case graphql.LeafType:
		return task.completeLeafValue(returnType, result, value)

	case graphql.Object:
		return task.completeObjectValue(returnType, result, value)

	// Union and Interface
	case graphql.AbstractType:
		return task.completeAbstractValue(returnType, result, value)
	}

	task.handleNodeError(
		graphql.NewError(fmt.Sprintf(`Cannot complete value of unexpected type "%v".`, returnType)),
		result)

	return false
}

func (task *ExecuteNodeTask) completeLeafValue(
	returnType graphql.LeafType,
	result *ResultNode,
	value interface{}) (ok bool) {

	coercedValue, err := returnType.CoerceResultValue(value)
	if err != nil {
		// See comments in graphql.NewCoercionError for the rules of handling error.
		if e, ok := err.(*graphql.Error); !ok || e.Kind != graphql.ErrKindCoercion {
			// Wrap the error in our own.
			err = graphql.NewDefaultResultCoercionError(returnType.Name(), value, err)
		}
		task.handleNodeError(err, result)
		return false
	}

	// Setup result and return.
	result.Kind = ResultKindLeaf
	result.Value = coercedValue
	return true
}

func (task *ExecuteNodeTask) completeObjectValue(
	returnType graphql.Object,
	result *ResultNode,
	value interface{}) (ok bool) {

	ctx := task.ctx

	// Collect fields in the selection set.
	childNodes, err := collectFields(ctx, task.node, returnType)
	if err != nil {
		task.handleNodeError(err, result)
		return false
	}

	// Dispatch tasks to execute subfields.
	dispatchTasksForObject(task.ctx, task.executor, result, childNodes, value)

	return true
}

func (task *ExecuteNodeTask) completeAbstractValue(
	returnType graphql.AbstractType,
	result *ResultNode,
	value interface{}) (ok bool) {

	var (
		ctx  = task.ctx
		node = task.node
	)

	resolver := returnType.TypeResolver()
	if resolver == nil {
		task.handleNodeError(
			graphql.NewError(
				fmt.Sprintf("Abstract type %s must provide resolver to resolve to an Object type at "+
					"runtime for field %s.%s with value %s",
					returnType, parentFieldType(ctx, node).Name(), node.Field.Name(),
					graphql.Inspect(value))), result)
		return false
	}

	runtimeType, err := resolver.Resolve(ctx.Context(), value, task.newResolveInfoFor(result))
	if err != nil {
		task.handleNodeError(err, result)
		return false
	}

	if runtimeType == nil {
		task.handleNodeError(
			graphql.NewError(
				fmt.Sprintf("Abstract type %s must resolve to an Object type at runtime for field %s.%s "+
					"with value %s, received nil.",
					returnType, parentFieldType(ctx, node).Name(), node.Field.Name(),
					graphql.Inspect(value))), result)
		return false
	}

	possibleTypes := task.ctx.Schema().PossibleTypes(returnType)
	if !possibleTypes.Contains(runtimeType) {
		task.handleNodeError(
			graphql.NewError(
				fmt.Sprintf(`Runtime Object type "%s" is not a possible type for "%s".`,
					runtimeType, returnType)), result)
		return false
	}

	return task.completeObjectValue(runtimeType, result, value)
}

// newResolveInfoFor creates a ResolveInfo to resolve result with current task context.
func (task *ExecuteNodeTask) newResolveInfoFor(result *ResultNode) graphql.ResolveInfo {
	if result == task.result {
		return task
	}

	return &ResolveInfo{
		ExecutionContext: task.ctx,
		ExecutionNode:    task.node,
		ResultNode:       result,
	}
}

// The following implements graphql.ResolveInfo for ExecuteNodeTask. This is a memory optimization.
// When resolving value for task.result (that's the case for ExecuteNodeTask.run), we can pass:
//
//	info := &ResolveInfo{
//		ExecutionContext: task.ctx,
//		ExecutionNode:    task.node,
//		ResultNode:       task.result,
//	}
//
// But a better way is to use "task" as an ResolveInfo object to save allocation overheads.

// Schema implements graphql.ResolveInfo.
func (task *ExecuteNodeTask) Schema() graphql.Schema {
	return task.ctx.Operation().Schema()
}

// Document implements graphql.ResolveInfo.
func (task *ExecuteNodeTask) Document() ast.Document {
	return task.ctx.Operation().Document()
}

// Operation implements graphql.ResolveInfo.
func (task *ExecuteNodeTask) Operation() *ast.OperationDefinition {
	return task.ctx.Operation().Definition()
}

// DataLoaderManager implements graphql.ResolveInfo.
func (task *ExecuteNodeTask) DataLoaderManager() graphql.DataLoaderManager {
	return task.ctx.DataLoaderManager()
}

// RootValue implements graphql.ResolveInfo.
func (task *ExecuteNodeTask) RootValue() interface{} {
	return task.ctx.RootValue()
}

// AppContext implements graphql.ResolveInfo.
func (task *ExecuteNodeTask) AppContext() interface{} {
	return task.ctx.AppContext()
}

// VariableValues implements graphql.ResolveInfo.
func (task *ExecuteNodeTask) VariableValues() graphql.VariableValues {
	return task.ctx.VariableValues()
}

// ParentFieldSelection implements graphql.ResolveInfo.
func (task *ExecuteNodeTask) ParentFieldSelection() graphql.FieldSelectionInfo {
	return fieldSelectionInfo{task.node.Parent}
}

// Object implements graphql.ResolveInfo.
func (task *ExecuteNodeTask) Object() graphql.Object {
	return parentFieldType(task.ctx, task.node)
}

// FieldDefinitions implements graphql.ResolveInfo.
func (task *ExecuteNodeTask) FieldDefinitions() []*ast.Field {
	return task.node.Definitions
}

// Field implements graphql.ResolveInfo.
func (task *ExecuteNodeTask) Field() graphql.Field {
	return task.node.Field
}

// Path implements graphql.ResolveInfo.
func (task *ExecuteNodeTask) Path() graphql.ResponsePath {
	return task.result.Path()
}

// Args implements graphql.ResolveInfo.
func (task *ExecuteNodeTask) Args() graphql.ArgumentValues {
	return task.node.Args
}

//===----------------------------------------------------------------------------------------====//
// AsyncValueTask
//===----------------------------------------------------------------------------------------====//

// AsyncValueTask polls a Future to get a value from an asynchronous computation. The value will be
// used to complete node execution (by calling completeValue with the value).
type AsyncValueTask struct {
	// Node that requires the value to complete
	nodeTask *ExecuteNodeTask

	// dataLoaderCycle specifies which cycle of data loaders dispatching this task is waiting for. See
	// comments for DataLoaderCycle type in executor.go for details.
	dataLoaderCycle DataLoaderCycle

	// The value to wait for calling completeValue
	value future.Future

	// Corresponding parameters to call completeValue
	returnType graphql.Type
	result     *ResultNode
}

// AsyncValueTask implements Task.
var _ Task = (*AsyncValueTask)(nil)

// run implements Task.
func (task *AsyncValueTask) run() {
	// Poll task.value to see whether it is ready.
	value, err := task.value.Poll(future.WakerFunc(task.wake))
	if err != nil {
		task.nodeTask.handleNodeError(err, task.result)
	} else if value != future.PollResultPending {
		task.nodeTask.completeValue(task.returnType, task.result, value)
		task.nodeTask.release()
	} else {
		// Value is not available at the time. Someone will perform the computation and notifies us via
		// wake when the value is ready.
		task.nodeTask.executor.Yield(task)

		// Dispatch data loaders if there's any pending .
		tryDispatchDataLoaders(task.nodeTask.ctx, task.nodeTask.executor, task.dataLoaderCycle)
	}
}

// wake dispatch the task to the executor (again) to poll its result.
func (task *AsyncValueTask) wake() error {
	task.nodeTask.executor.Resume(task)
	return nil
}

// tryDispatchDataLoaders dispatches data loaders if the dispatch hasn't occurred in the given
// taskCycle.
func tryDispatchDataLoaders(
	ctx *ExecutionContext,
	executor executor,
	taskCycle DataLoaderCycle) (newCycle DataLoaderCycle) {

	dataLoaderManager := ctx.DataLoaderManager()
	if dataLoaderManager == nil || !dataLoaderManager.HasPendingDataLoaders() {
		// Quick return if data loader is not enabled or there's no any loaders pending for dispatch.
		return
	}

	for {
		// Obtain current data loader cycle.
		curCycle := executor.DataLoaderCycle()

		if taskCycle == curCycle {
			// The task depends on the dispatch of data loaders in given cycle which hasn't happened.
			// Increment the cycle to obtain the permit to run dispatch for the cycle. The increment may
			// fail. For example, concurrent executor performs a CAS to ensure only one successfully
			// increment the counter. In such case, restart the loop to reload executor's cycle counter.
			if executor.IncDataLoaderCycle(taskCycle + 1) {
				// Successfully increment the cycle counter. Perform the actual data loader dispatch.
				dispatchDataLoaders(ctx.Context(), dataLoaderManager)
				return taskCycle + 1
			}
		} else {
			// Someone has dispatched the data loaders.
			return curCycle
		}
	}
}

func dispatchDataLoaders(ctx context.Context, manager graphql.DataLoaderManager) {
	// Dispatching a DataLoader may request more data which generate a new set of loaders that is
	// waiting for dispatch.
	for {
		pendingLoaders := manager.GetAndResetPendingDataLoaders()
		if len(pendingLoaders) == 0 {
			break
		}

		for loader := range pendingLoaders {
			loader.Dispatch(ctx)
		}
	}
}
