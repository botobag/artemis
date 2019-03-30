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

package executor

import (
	"github.com/botobag/artemis/graphql"
	"github.com/botobag/artemis/graphql/ast"
)

// An ExecutionNode represents a field to be evaluated (executed) in ExecutionGraph. It is computed
// at the first time a field evaluates its selection set where variable values and field's runtime
// type are both known.
//
// In this graph, every node represents a field to be executed. This intermediate data stores
// information computed in CollectFields [0]. That is, field arguments were coerced, @include/@skip
// was evaluated, selection set was flatten in node, field selections in the set with the same
// response key were coalesced.
//
// Storing these computation results have 2 benefits. First, it saves the overheads when a node is
// revisited during execution. This is a common case when resolving a List value. For example,
//
// Given a schema:
//
//	type Query {
//	  hero: Character
//	}
//
//	type Character {
//	  name: String
//	  friends: [Character]
//	}
//
// and a query document:
//
//	{
//	  hero {
//	    name
//	    friends { # The Selection Set of this node would be evaluated multiple time
//	      name
//	    }
//	  }
//	}
//
// A possible result would look like,
//
//	{
//	  "data": {
//	    "hero": {
//	      "name": "R2-D2",
//	      "friends": [
//	        {
//	          "name": "Luke Skywalker"
//	        },
//	        {
//	          "name": "Han Solo"
//	        },
//	        {
//	          "name": "Leia Organa"
//	        }
//	      ]
//	    }
//	  }
//	}
//
// During the execution, the Selection Set of the node "friends" (so as its sub fields) would be
// used and executed many times depends on the list size.
//
// Secondly, we also make this information accessible from field resolvers, allowing them to have
// more context about resolving field. For example, the field resolver now knows the parent node and
// from there one can know what are the siblings by looking at the parent's child nodes.
//
// [0]: https://graphql.github.io/graphql-spec/June2018/#CollectFields()
type ExecutionNode struct {
	// Parent of this node in the graph; This is nil for root node.
	Parent *ExecutionNode

	// Field definitions for this node; Note that this is an array because a field could be requested
	// multiple times in the documents. Validator already ensures that they don't have conflict
	// definitions (e.g., fields with different argument values). Their results
	// are merged into one field in the response. This is nil for root node.
	Definitions []*ast.Field

	// The corresponding Field definition in the schema; This is nil for root node.
	Field graphql.Field

	// Arguments to this field; Note that the argument value is coerced unless it is a variable which
	// will remain as an ast.Variable.
	Args graphql.ArgumentValues

	// The child nodes of this node; Note that this is a map where key is the concrete type of the
	// node. Selection Sets in a field may vary subject to its runtime type.
	Children map[graphql.Object][]*ExecutionNode
}

// IsRoot returns true if this node represents a root node.
func (node *ExecutionNode) IsRoot() bool {
	return node.Parent == nil
}

// ResponseKey is the field alias name if defined, otherwise the field name.
func (node *ExecutionNode) ResponseKey() string {
	return node.Definitions[0].ResponseKey()
}
