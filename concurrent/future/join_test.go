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

package future_test

import (
	"errors"

	"github.com/botobag/artemis/concurrent/future"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// completeOnNotify is the future that only completes when Complete or SetErr method is called.
type completeOnNotify struct {
	value     interface{}
	err       error
	waker     future.Waker
	completed bool
	polled    bool
}

func (f *completeOnNotify) Poll(waker future.Waker) (future.PollResult, error) {
	if !f.completed {
		f.waker = waker
		return future.PollResultPending, nil
	}

	// Completed future can only be polled once.
	Expect(f.polled).Should(BeFalse())
	f.polled = true

	if f.err != nil {
		return nil, f.err
	}

	return f.value, nil
}

func (f *completeOnNotify) Complete(value interface{}) error {
	Expect(f.completed).Should(BeFalse())
	f.completed = true
	f.value = value
	Expect(f.waker).ShouldNot(BeNil())
	return f.waker.Wake()
}

func (f *completeOnNotify) SetErr(err error) error {
	Expect(f.completed).Should(BeFalse())
	f.completed = true
	f.err = err
	Expect(f.waker).ShouldNot(BeNil())
	return f.waker.Wake()
}

var _ = Describe("Join: collect values from multiple futures", func() {
	It("creates future that contains no underlying futures", func() {
		f := future.Join([]future.Future{}...)
		Expect(future.BlockOn(f)).Should(BeEmpty())
	})

	It("creates future that collects values from multiple futures into an array", func() {
		f := future.Join(
			future.Ready(1),
			future.Ready(2),
			future.Ready(3),
		)
		Expect(future.BlockOn(f)).Should(Equal([]interface{}{1, 2, 3}))
	})

	It("failed if one of the input futures throws error", func() {
		expectErr := errors.New("an error value")
		f := future.Join(
			future.Ready(1),
			future.Err(expectErr),
			future.Ready(3),
		)
		_, err := future.BlockOn(f)
		Expect(err).Should(MatchError(err))
	})

	Describe("with more complex future which completes on notify", func() {
		var f1, f2, f3 *completeOnNotify

		BeforeEach(func() {
			f1 = &completeOnNotify{}
			f2 = &completeOnNotify{}
			f3 = &completeOnNotify{}
		})

		It("wakes join at most once on its completion", func() {
			f := future.Join(f1, f2, f3)

			waken := false
			waker := future.WakerFunc(func() error {
				Expect(waken).Should(BeFalse())
				waken = true
				return nil
			})

			Expect(f.Poll(waker)).Should(Equal(future.PollResultPending))
			Expect(waken).Should(BeFalse())

			Expect(f1.Complete(1)).Should(Succeed())
			Expect(f.Poll(waker)).Should(Equal(future.PollResultPending))
			Expect(waken).Should(BeFalse())

			Expect(f2.Complete(2)).Should(Succeed())
			Expect(f.Poll(waker)).Should(Equal(future.PollResultPending))
			Expect(waken).Should(BeFalse())

			Expect(f3.Complete(3)).Should(Succeed())
			Expect(f.Poll(waker)).Should(Equal([]interface{}{1, 2, 3}))
			Expect(waken).Should(BeTrue())
		})

		It("completes join with error if one of future is completed with an error", func() {
			f := future.Join(f1, f2, f3)

			waken := false
			waker := future.WakerFunc(func() error {
				Expect(waken).Should(BeFalse())
				waken = true
				return nil
			})

			Expect(f.Poll(waker)).Should(Equal(future.PollResultPending))
			Expect(waken).Should(BeFalse())

			Expect(f1.Complete(1)).Should(Succeed())
			Expect(f.Poll(waker)).Should(Equal(future.PollResultPending))
			Expect(waken).Should(BeFalse())

			Expect(f2.Complete(2)).Should(Succeed())
			Expect(f.Poll(waker)).Should(Equal(future.PollResultPending))
			Expect(waken).Should(BeFalse())

			Expect(f3.SetErr(errors.New("error"))).Should(Succeed())
			_, err := f.Poll(waker)
			Expect(err).Should(MatchError("error"))
			Expect(waken).Should(BeTrue())
		})
	})
})
