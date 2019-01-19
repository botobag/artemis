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

var _ = Describe("Ready: Future that is immediately ready with a value", func() {
	It("creates future that is ready with a value", func() {
		Expect(future.Ready(1).Poll(nil)).Should(Equal(1))
	})

	It("creates future that is ready with an error", func() {
		testErr := errors.New("ready with an error")
		_, err := future.Err(testErr).Poll(nil)
		Expect(err).Should(MatchError(testErr))

		_, err = future.Err(nil).Poll(nil)
		Expect(err).Should(MatchError(""))
	})
})
