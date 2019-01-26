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

package future

// join implements Future returned by Join.
type join struct {
	inputs  []Future
	results []interface{}
}

// Poll implements future.Future.
func (f *join) Poll(waker Waker) (PollResult, error) {
	var (
		done    = true
		results = f.results
	)

	for i, input := range f.inputs {
		if results[i] != PollResultPending {
			continue
		}

		// Check input future.
		result, err := input.Poll(waker)
		if err != nil {
			return nil, err
		}

		if result == PollResultPending {
			done = false
		} else {
			results[i] = interface{}(result)
		}
	}

	if done {
		return f.results, nil
	}

	return PollResultPending, nil
}

// Join creates a Future which aggregates values from a collection of Futures.
//
// The returned Future drives execution of the input futures and collect the results into an
// []interface{} in the same order as they're given.
func Join(f ...Future) Future {
	// Initialize storage for result values.
	results := make([]interface{}, len(f))
	for i := range results {
		results[i] = PollResultPending
	}

	return &join{
		inputs:  f,
		results: results,
	}
}
