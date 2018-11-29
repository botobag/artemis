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
	"github.com/botobag/artemis/internal/util"
)

// createdTypes tracks Type instances created for each TypeDefinition instance.
var createdTypes util.SyncMap

// newTypeResult is the value type of createdTypes.
type newTypeResult struct {
	// The created type
	t Type

	// Thr creator that is responsible for creating the t
	creator typeCreator

	// Any error occurred during creation
	err error

	// Wait for other goroutine to complete the creation.
	done chan bool
}

func (result *newTypeResult) waitForCompletion() (Type, error) {
	// Wait on result.done for completion.
	select {
	case <-result.done:
		break
	}
	return result.t, result.err
}

func (result *newTypeResult) complete() {
	// Release the reference to creator so it can be garbage collected.
	result.creator = nil
	// Send completion notification.
	close(result.done)
}

func (result *newTypeResult) completeWithError(err error) {
	// Reset the result.
	result.t = nil
	// Release the reference to creator so it can be garbage collected.
	result.creator = nil
	result.err = err
	// This will wake up everyone that blocks on this result.
	close(result.done)
}

// typeDefinitionResolver resolves a TypeDefinition into a Type during type finalization.
type typeDefinitionResolver func(typeDef TypeDefinition) (Type, error)

// Resolve simply calls resolver(typeDef) to make typeDefinitionResolver looks like an object.
func (resolver typeDefinitionResolver) Resolve(typeDef TypeDefinition) (Type, error) {
	return resolver(typeDef)
}

// typeCreator defines interfaces to be required to work with newTypeImpl to create a type instance.
type typeCreator interface {
	// TypeDefinition returns the TypeDefinition instance processed by this creator.
	TypeDefinition() TypeDefinition

	// LoadDataAndNew loads type data from TypeDefinition and create a "semi-initialized" Type
	// instance for return.
	LoadDataAndNew() (Type, error)

	// Finalize completes type creation for t that was returned from LoadDataAndNew. Any type
	// reference resolution such as resolving field type when defining an Object type must be done
	// here otherwise deadlock may happen if two types directly/indirectly depend on each other. The
	// closure given in the 2nd argument can be used to resolve a type reference, turning a
	// TypeDefinition into Type. Because at this point, the given type instance t has been
	// "registered", we are safe to load any dependent type including the type we're defining.
	Finalize(t Type, typeDefResolver typeDefinitionResolver) error
}

// nilTypeCreator is an artificial type creator for dealing with "nil" TypeDefinition. It resolves
// to "nil" Type without causing any error.
//
// Note: In most cases, "nil" Type is abnormal and invalid. But nilTypeCreator is not the place to
// 			 raise the error. Caller or validator are.
type nilTypeCreator struct{}

var _ typeCreator = nilTypeCreator{}

// TypeDefinition implements typeCreator.
func (nilTypeCreator) TypeDefinition() TypeDefinition {
	return nil
}

// LoadDataAndNew implements typeCreator.
func (nilTypeCreator) LoadDataAndNew() (Type, error) {
	return nil, nil
}

// Finalize implements typeCreator.
func (nilTypeCreator) Finalize(t Type, typeDefResolver typeDefinitionResolver) error {
	return nil
}

func newCreatorFor(typeDef TypeDefinition) typeCreator {
	switch typeDef := typeDef.(type) {
	case ScalarTypeDefinition:
		return &scalarTypeCreator{typeDef}
	case ScalarAliasTypeDefinition:
		return &scalarAliasTypeCreator{typeDef}
	case EnumTypeDefinition:
		return &enumTypeCreator{typeDef}
	case ObjectTypeDefinition:
		return &objectTypeCreator{typeDef}
	case InterfaceTypeDefinition:
		return &interfaceTypeCreator{typeDef}
	case UnionTypeDefinition:
		return &unionTypeCreator{typeDef}
	case InputObjectTypeDefinition:
		return &inputObjectTypeCreator{typeDef}
	case ListTypeDefinition:
		return &listTypeCreator{typeDef}
	case NonNullTypeDefinition:
		return &nonNullTypeCreator{typeDef}
	case nil:
		return &nilTypeCreator{}
	}
	panic("unknown type of TypeDefinition")
}

// newTypeImpl is the internal implementation of NewType for creating a type instance from given
// TypeDefinition. Call NewType (or its variants such as NewScalar) instead of calling it directly.
func newTypeImpl(creator typeCreator) (Type, error) {
	// Check whether the requested typeDef have already created a Type instance.
	typeCreatedResult, ok := createdTypes.Load(creator.TypeDefinition())
	if ok {
		return typeCreatedResult.(*newTypeResult).waitForCompletion()
	}

	return newTypeImplInternal(creator, map[TypeDefinition]Type{})
}

// newTypeImplInternal should only be called from newTypeImpl and from itself (recusively).
// newTypeImplInternal is recursively called from itself if the type has dependent types.
// finalizingTypeDefs contains set of TypeDefinition's that are finalizing in the call stack of
// newTypeImplInternal.
func newTypeImplInternal(creator typeCreator, finalizingTypeDefs map[TypeDefinition]Type) (Type, error) {
	// newTypeImplInternal assumes the caller has tested the existence of defining type in
	// createdTypes. So it started by trying to insert an entry for the type in createdTypes map.
	typeDef := creator.TypeDefinition()

	// Call LoadDataAndNew to load data from TypeDefinition to creator and initialize a type instance.
	// This won't resolve any TypeDefinition's referenced in typeDef otherwise deadlock could happen.
	typeInstance, err := creator.LoadDataAndNew()
	if err != nil {
		return nil, err
	}

	// Prepare a result to insert into createdTypes.
	result := &newTypeResult{
		t:       typeInstance,
		creator: creator,
		done:    make(chan bool),
	}

	// Try to insert the typeInstance into createdTypes.
	typeCreatedResult, loaded := createdTypes.LoadOrStore(typeDef, result)
	if loaded {
		// Someone sneaked in and got ticket to create the type. Wait for the completion.
		return typeCreatedResult.(*newTypeResult).waitForCompletion()
	}

	// During type finalization, creator calls the typeDefResolver for turining a TypeDefinition into
	// a Type. This is very similar to how we solve current typeDef.
	typeDefResolver := typeDefinitionResolver(func(typeDef TypeDefinition) (Type, error) {
		// Handle pseudo-TypeDefinition specially here.
		switch typeDef := typeDef.(type) {
		case typeWrapperTypeDefinition:
			return typeDef.Type(), nil

		case interfaceTypeWrapperTypeDefinition:
			return typeDef.Type(), nil
		}

		// Chck finalizingTypeDefs. If typeDef is listed there, it is resolved by one of previous calls
		// to newTypeImplInternal on the stack. Stop recursion and return the semi-initialized type in
		// any way otherwise we'll create a loop.
		if t, exists := finalizingTypeDefs[typeDef]; exists {
			return t, nil
		}

		// Quick check on whether typeDef have already created a Type instance.
		typeCreatedResult, ok := createdTypes.Load(typeDef)
		if ok {
			return typeCreatedResult.(*newTypeResult).waitForCompletion()
		}

		// No luck. Get the creator for the typeDef and call newTypeImpl recursively.
		return newTypeImplInternal(newCreatorFor(typeDef), finalizingTypeDefs)
	})

	// Add typeDef to finalizingTypeDefs.
	finalizingTypeDefs[typeDef] = result.t
	// And remove on return.
	defer func() {
		delete(finalizingTypeDefs, typeDef)
	}()

	// If here, we're responsible for creating the type. Call creator's Finalize to complete the type.
	if err = creator.Finalize(result.t, typeDefResolver); err != nil {
		// Complete the type with error.
		result.completeWithError(err)
		return nil, err
	}

	// We're done. Someone may also wait for the result. Call complete() to Notify them.
	result.complete()

	// We're done!
	return result.t, nil
}
