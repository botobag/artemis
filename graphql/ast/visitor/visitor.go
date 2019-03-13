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

//go:generate go run gen.go

package visitor

// Result contains the return value for visitor function. The behavior of the visitor can be altered
// based on the value, including skipping over a sub-tree of AST (by returning SkipSubTree) or to
// stop the whole traversal (by returning Break).
type Result interface {
	// result puts a special mark for a Result type. The unexpored field forbids external package to
	// use our defined result constants.
	result()
}

// resultConstant is a constant to tell visitor of the next action to take
type resultConstant int

// result implements Result.
func (resultConstant) result() {}

// Enumeration of resultConstant.
const (
	// No action, continue the traversal.
	Continue resultConstant = iota

	// Skip over the sub-tree of AST
	SkipSubTree

	// Stop the traversal on return
	Break
)
