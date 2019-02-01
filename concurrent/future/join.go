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

import (
	"sync/atomic"
)

// join implements Future returned by Join.
type join struct {
	values []interface{}
	// If one of the value is completed with an error, this will set to the error for return.
	err   atomic.Value /*error */
	waker Waker
	// Number of values currently wait for completing the join
	pendingCount int64
}

// joinPendingValue is created for each future within a join that is waiting for result. It
// implements Waker interface which re-poll the associated future on notify and if the future
// completes with a result, update the value to the containing join. Furthermore, if the update
// clears all pending values in containing join (i.e., j.pendingCount reaches 0), also notify
// j.waker.
type joinPendingValue struct {
	// The join that depends on this value
	j *join

	// The future to poll for value
	f Future

	// The index of value within containing join
	i int
}

func (value *joinPendingValue) poll() (interface{}, error) {
	return value.f.Poll(value)
}

func (value *joinPendingValue) Wake() error {
	// Poll the future.
	result, err := value.poll()
	if err != nil {
		// Store error and wake the join to return.
		j := value.j
		j.err.Store(err)
		return j.waker.Wake()
	} else if result != PollResultPending {
		// The future is completed. Set the result.
		j := value.j
		j.values[value.i] = result

		// Decrement pendingCount.
		if atomic.AddInt64(&j.pendingCount, -1) == 0 {
			// All values are ready. Notify join's waker.
			return j.waker.Wake()
		}
	}

	return nil
}

// Poll implements future.Future.
func (j *join) Poll(waker Waker) (PollResult, error) {
	var (
		done   int64
		values = j.values
	)

	if err := j.err.Load(); err != nil {
		return nil, err.(error)
	}

	// Update waker.
	j.waker = waker

	for i, value := range values {
		if value, ok := value.(*joinPendingValue); ok {
			// Poll the future for value.
			result, err := value.poll()
			if err != nil {
				return nil, err
			} else if result != PollResultPending {
				values[i] = interface{}(result)
				done++
			}
		}
	}

	if atomic.AddInt64(&j.pendingCount, -done) == 0 {
		return j.values, nil
	}

	return PollResultPending, nil
}

// Join creates a Future which aggregates values from a collection of Futures.
//
// The returned Future drives execution of the input futures and collect the results into an
// []interface{} in the same order as they're given.
func Join(f ...Future) Future {
	// Initialize storage for result values.
	values := make([]interface{}, len(f))
	j := &join{
		values:       values,
		waker:        NopWaker,
		pendingCount: int64(len(f)),
	}

	for i, f := range f {
		values[i] = &joinPendingValue{
			j: j,
			f: f,
			i: i,
		}
	}

	return j
}
