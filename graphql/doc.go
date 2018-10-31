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

// Package graphql provides an implementation of GraphQL. It provides foundation to build GraphQL
// type schema and to serve queries against that type schema.
//
// TypeDefinition-NewType-Type Design
//
// Each Type has corresponding TypeDefinition which provides a set of interfaces for NewType to get
// required data to initialize instance for the Type.
//
// Instead of providing data (or definition) for creating Type via a concrete object (i.e.,
// `struct`), TypeDefinition provides the data by requiring type creator to implement an object that
// fulfills a set of interfaces. NewType will read data through the interfaces. Because referencing
// data in function never causes "initialization loop" like global variables, types that depends on
// each other and even depends on itself can be achieved without additional work.
//
// The added TypeDefinition abstraction and the ability to provide complete type dependency graph
// (through the way mentioned above) enables us to deal with circular type creation in the library.
//
// Note that every instance of TypeDefinition creates at most one Type instance. NewType tracks the
// Type instantiation by mapping the Type instance to its TypeDefinition instance. This implies that
// once the Type instance corresponded to the TypeDefinition is created, changes that made to
// TypeDefinition won't reflect to the created Type. Type instance for a TypeDefinition is created
// when NewType is called with the TypeDefinition instance or any types that reference to the
// TypeDefinition instance.
package graphql
