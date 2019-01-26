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
})
