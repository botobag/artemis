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
	"encoding/json"
	"sync"

	"github.com/botobag/artemis/concurrent/future"
	"github.com/botobag/artemis/dataloader"
	"github.com/botobag/artemis/graphql/ast"
)

// An ArgumentValues contains argument values given to a field. It is immutable after it is created.
type ArgumentValues struct {
	values map[string]interface{}
}

var noArgumentValues = ArgumentValues{
	// Allocate an non-nil map to eliminate null-check for Lookup.
	values: map[string]interface{}{},
}

// NoArgumentValues represents an empty argument value set.
func NoArgumentValues() ArgumentValues {
	return noArgumentValues
}

// NewArgumentValues creates an ArgumentValues from given values.
func NewArgumentValues(values map[string]interface{}) ArgumentValues {
	if len(values) == 0 {
		return noArgumentValues
	}
	return ArgumentValues{values}
}

// Lookup returns argument value for the given name. If argument with the given name doesn't exist,
// returns nil. The second value (ok) is a bool that is true if the argument exists, and false if
// not.
func (args ArgumentValues) Lookup(name string) (value interface{}, ok bool) {
	value, ok = args.values[name]
	return
}

// Get returns argument value for the given name. It returns nil if no such argument was found.
func (args ArgumentValues) Get(name string) interface{} {
	return args.values[name]
}

// MarshalJSON implements json.Marshaler to serialize the internal map in ArgumentValues into JSON.
// This is primarily used by tests for verifying argument values.
func (args ArgumentValues) MarshalJSON() ([]byte, error) {
	return json.Marshal(args.values)
}

// VariableValues contains values for variables defined by the query. It is immutable after it is
// created.
//
// Reference: https://facebook.github.io/graphql/June2018/#sec-Language.Variables
type VariableValues struct {
	values map[string]interface{}
}

var noVariableValues = VariableValues{
	// Allocate an non-nil map to eliminate null-check for Lookup.
	values: map[string]interface{}{},
}

// NoVariableValues represents an empty variable value set.
func NoVariableValues() VariableValues {
	return noVariableValues
}

// NewVariableValues creates an VariableValues from given values.
func NewVariableValues(values map[string]interface{}) VariableValues {
	return VariableValues{values}
}

// Lookup returns variable value for the given name. If variable with the given name doesn't exist,
// returns nil. The second value (ok) is a bool that is true if the variable exists, and false if
// not.
func (vars VariableValues) Lookup(name string) (value interface{}, ok bool) {
	value, ok = vars.values[name]
	return
}

// Get returns variable value for the given name. It returns nil if no such variable was found.
func (vars VariableValues) Get(name string) interface{} {
	return vars.values[name]
}

// MarshalJSON implements json.Marshaler to serialize the internal map in VariableValues into JSON.
// This is primarily used by tests for verifying argument values.
func (vars VariableValues) MarshalJSON() ([]byte, error) {
	return json.Marshal(vars.values)
}

// FieldSelectionInfo is provided as part of ResolveInfo which contains the Selection [0] for the
// resolving field and its parent fields.
//
// Reference: https://facebook.github.io/graphql/June2018/#Field
type FieldSelectionInfo interface {
	// A link to the parent field whose selection set contains this field selection
	Parent() FieldSelectionInfo

	// AST definition of the field; See comments in ResolveInfo.FieldDefinitions for more details.
	FieldDefinitions() []*ast.Field

	// The corresponding Field definition in Schema
	Field() Field

	// Argument values that are given to the field
	Args() ArgumentValues

	// TODO: Also expose the field result (from executor.ResultNode).
}

// DataLoaderManager provides a way to,
//
//  1. Let user manage DataLoader instances being used during execution and access the loaders via
//     ResolveInfo;
//  2. Let executor know which DataLoader's are pending for batch data fetching and schedule for
//     their dispatch.
type DataLoaderManager interface {
	// HasPendingDataLoaders returns true if there's any data loaders waiting for dispatch.
	HasPendingDataLoaders() bool

	// GetAndResetPendingDataLoaders reports DataLoader's that are waiting for dispatching and resets
	// current list.
	GetAndResetPendingDataLoaders() map[*dataloader.DataLoader]bool
}

// DataLoaderManagerBase is useful to embed in a DataLoaderManager class to track the use of data
// loaders as required by DataLoaderManager.
type DataLoaderManagerBase struct {
	// Mutex that guards pendingDataLoaders
	mutex sync.Mutex

	// Data loaders that have pending batch load to perform
	pendingLoaders map[*dataloader.DataLoader]bool
}

// LoadWith requests given loader for data at given key. It also updates pendingLoaders as
// appropriated.
func (manager *DataLoaderManagerBase) LoadWith(loader *dataloader.DataLoader, key dataloader.Key) (future.Future, error) {
	// Acquire lock to update pendingLoaders. Note that have to be done before loader.Load.
	mutex := &manager.mutex
	mutex.Lock()

	f, err := loader.Load(key)
	if err != nil {
		mutex.Unlock()
		return nil, err
	}

	// Update pendingLoaders.
	pendingLoaders := manager.pendingLoaders
	if pendingLoaders == nil {
		pendingLoaders = map[*dataloader.DataLoader]bool{}
		manager.pendingLoaders = pendingLoaders
	}
	pendingLoaders[loader] = true

	mutex.Unlock()
	return f, nil
}

// LoadManyWith requests given loader for data at given keys. It also updates pendingLoaders as
// appropriated. It is very similar to LoadWith with just `loader.Load` replaced with
// `loader.LoadMany`.
func (manager *DataLoaderManagerBase) LoadManyWith(loader *dataloader.DataLoader, keys dataloader.Keys) (future.Future, error) {
	// Acquire lock to update pendingLoaders. Note that have to be done before loader.LoadMany.
	mutex := &manager.mutex
	mutex.Lock()

	f, err := loader.LoadMany(keys)
	if err != nil {
		mutex.Unlock()
		return nil, err
	}

	// Update pendingLoaders.
	pendingLoaders := manager.pendingLoaders
	if pendingLoaders == nil {
		pendingLoaders = map[*dataloader.DataLoader]bool{}
		manager.pendingLoaders = pendingLoaders
	}
	pendingLoaders[loader] = true

	mutex.Unlock()
	return f, nil
}

// HasPendingDataLoaders implements DataLoaderManager.HasPendingDataLoaders.
func (manager *DataLoaderManagerBase) HasPendingDataLoaders() bool {
	mutex := &manager.mutex
	mutex.Lock()
	result := len(manager.pendingLoaders) != 0
	mutex.Unlock()
	return result
}

// GetAndResetPendingDataLoaders implements DataLoaderManager.HasPendingDataLoaders.
func (manager *DataLoaderManagerBase) GetAndResetPendingDataLoaders() map[*dataloader.DataLoader]bool {
	mutex := &manager.mutex
	mutex.Lock()
	result := manager.pendingLoaders
	manager.pendingLoaders = nil
	mutex.Unlock()
	return result
}

// ResolveInfo exposes a collection of information about execution state for resolvers.
type ResolveInfo interface {
	//===----------------------------------------------------------------------------------------===//
	// GraphQL Operation
	//===----------------------------------------------------------------------------------------===//
	// The following states are related to AST of the operation being executed (provided by
	// executor.PreparedOperation)

	// Schema of the type system that is currently executing.
	Schema() Schema

	// Document that contains definitions for the operation.
	Document() ast.Document

	// Definition of this operation
	Operation() *ast.OperationDefinition

	//===----------------------------------------------------------------------------------------===//
	// Execution Context
	//===----------------------------------------------------------------------------------------===//
	// The following states are related to the contexts supplied with the execution request (provided
	// by executor.ExecutionContext)

	// DataLoaderManager that manages usage and dispatch of data loaders during execution.
	DataLoaderManager() DataLoaderManager

	// RootValue is an initial value corresponding to the root type being executed.
	RootValue() interface{}

	// AppContext contains an application-specific data. It is what you passed to the AppContext in
	// executor.ExecuteParams. It is commonly used to represent an authenticated user, or
	// request-specific caches.
	AppContext() interface{}

	// VariableValues contains values to the parameters in current query. The values has passed
	// through the input coercion.
	VariableValues() VariableValues

	//===----------------------------------------------------------------------------------------===//
	// Field Selection Info
	//===----------------------------------------------------------------------------------------===//
	// The following states are related to the field that is being resolving in the Selection Set
	// (provided by executor.ExecutionNode)

	// Link to the Selection that requests this field.
	ParentFieldSelection() FieldSelectionInfo

	// The Oject in which the Field belongs to
	Object() Object

	// AST definitions of the field that is being requested; Note that it is an array of ast.Field
	// because a field could e requested in a Selection Set multiple times (with the same
	// name/response key) with different or the same sub-selection set. For example:
	//
	//	{
	//	  foo {
	//	    bar
	//	  }
	//
	//	  foo {
	//	    bar
	//	    baz
	//	  }
	//	}
	//
	// The above operation specifies "foo" two times which is valid. FieldDefinitions will returns two
	// ast.Field each of which corresponding to one of "foo" in the query. The query result would
	// contain only one "foo" with all three fields merged in the field data.
	FieldDefinitions() []*ast.Field

	// The corresponding Field definition in Schema
	Field() Field

	// Path in the response to this field. This can be serialized to the "path" when there're errors
	// occurred on field. Note that this is created on request by traversing ResultNode and could be
	// expensive. Cache it if you want to it to be reusable.
	Path() ResponsePath

	// Argument values that are given to the field
	Args() ArgumentValues
}
