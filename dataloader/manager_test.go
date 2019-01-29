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

package dataloader_test

import (
	"context"
	"errors"

	"github.com/botobag/artemis/concurrent/future"
	"github.com/botobag/artemis/dataloader"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Manager", func() {
	var idLoaderFactory = dataloader.FactoryFunc(func() (*dataloader.DataLoader, error) {
		idLoader := newIdentityLoader(dataloader.Config{})
		return idLoader.DataLoader, nil
	})

	It("memoizes DataLoader instances", func() {
		info := &dataloader.RegisterInfo{
			Key:     "Test",
			Factory: idLoaderFactory,
		}

		manager := &dataloader.Manager{}

		// Register dataloader.
		loader, err := manager.GetOrCreate(info)
		Expect(err).ShouldNot(HaveOccurred())

		// Register the 2nd time. It should return the loader instance that was created before.
		loader2, err := manager.GetOrCreate(info)
		Expect(err).ShouldNot(HaveOccurred())

		Expect(loader2).Should(BeIdenticalTo(loader))
	})

	It("rejects invalid DataLoader factory", func() {
		// Nil factory
		manager := &dataloader.Manager{}
		_, err := manager.GetOrCreate(&dataloader.RegisterInfo{
			Key:     "NilFactory",
			Factory: nil,
		})
		Expect(err).Should(MatchError(`DataLoader factory for "NilFactory" is not provided`))

		// Factory that returns nil DataLoader instance without error.
		_, err = manager.GetOrCreate(&dataloader.RegisterInfo{
			Key: "FactoryReturnsNil",
			Factory: dataloader.FactoryFunc(func() (*dataloader.DataLoader, error) {
				return nil, nil
			}),
		})
		Expect(err).Should(MatchError(ContainSubstring(`DataLoader factory for "FactoryReturnsNil" returns a nil instance`)))

		// Factory that returns error.
		factoryErr := errors.New("factory error")
		_, err = manager.GetOrCreate(&dataloader.RegisterInfo{
			Key: "ErrorFactory",
			Factory: dataloader.FactoryFunc(func() (*dataloader.DataLoader, error) {
				return nil, factoryErr
			}),
		})
		Expect(err).Should(MatchError(factoryErr))
	})

	It("can dispatch all managed DataLoader instances", func() {
		manager := &dataloader.Manager{}

		// Register dataloader.
		aLoader, err := manager.GetOrCreate(&dataloader.RegisterInfo{
			Key:     "LoaderA",
			Factory: idLoaderFactory,
		})
		Expect(err).ShouldNot(HaveOccurred())

		bLoader, err := manager.GetOrCreate(&dataloader.RegisterInfo{
			Key:     "LoaderB",
			Factory: idLoaderFactory,
		})
		Expect(err).ShouldNot(HaveOccurred())

		a, err := aLoader.Load("A")
		Expect(err).ShouldNot(HaveOccurred())

		b, err := bLoader.Load("B")
		Expect(err).ShouldNot(HaveOccurred())

		go manager.DispatchAll(context.Background())
		Expect(future.BlockOn(a)).Should(Equal("A"))
		Expect(future.BlockOn(b)).Should(Equal("B"))
	})
})
