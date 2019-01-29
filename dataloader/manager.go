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

package dataloader

import (
	"context"
	"fmt"
	"sync"

	"github.com/botobag/artemis/internal/util"
)

// Factory creates a DataLoader.
type Factory interface {
	Create() (*DataLoader, error)
}

// The FactoryFunc type is an adapter to allow the use of ordinary functions as Factory. If f is a
// function with the appropriate signature, FactoryFunc(f) is a Factory that calls f.
type FactoryFunc func() (*DataLoader, error)

// Create implements Factory by simply calling f()
func (f FactoryFunc) Create() (*DataLoader, error) {
	return f()
}

// RegisterInfo provides necessary information to register a DataLoader.
type RegisterInfo struct {
	// A string key that uniquely identifies the DataLoader registered in a Manager by this Info.
	Key string

	// Factory that creates DataLoader
	Factory Factory
}

// Manager provides a way to register and dispatch a collection of DataLoaders.
type Manager struct {
	// A map from RegisterInfo.Key to the created DataLoader instance
	loaders util.SyncMap

	// Mutex that prevent multiple DispatchAll's to be executed concurrently.
	dispatchMutex sync.Mutex
}

// GetOrCreate creates and adds a new DataLoader if one does not already exist with the key given in
// info.Key.
func (manager *Manager) GetOrCreate(info *RegisterInfo) (*DataLoader, error) {
	loaders := &manager.loaders

	// Check whether the dataloader already exists.
	if loader, found := loaders.Load(info.Key); found {
		return loader.(*DataLoader), nil
	}

	if info.Factory == nil {
		return nil, fmt.Errorf(`DataLoader factory for "%s" is not provided`, info.Key)
	}

	// Create a new one.
	loader, err := info.Factory.Create()
	if err != nil {
		return nil, err
	}

	// Reject nil loader.
	if loader == nil {
		return nil, fmt.Errorf(`DataLoader factory for "%s" returns a nil instance which is not `+
			`valid for registration`, info.Key)
	}

	// Register loader.
	registeredLoader, registered := loaders.LoadOrStore(info.Key, loader)
	if registered {
		return registeredLoader.(*DataLoader), nil
	}

	return loader, nil
}

// DispatchAll dispatches all registered DataLoaders.
func (manager *Manager) DispatchAll(ctx context.Context) {
	mutex := &manager.dispatchMutex
	mutex.Lock()
	manager.loaders.Range(func(key, value interface{}) bool {
		value.(*DataLoader).Dispatch(ctx)
		// Return true to continue the iteration.
		return true
	})
	mutex.Unlock()
}
