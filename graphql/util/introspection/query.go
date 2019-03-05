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

package introspection

import (
	"fmt"
	"text/template"

	"github.com/botobag/artemis/internal/util"
)

// queryOptions configures the query string built from Query.
type queryOptions struct {
	// Whether to include descriptions in the introspection result
	OmitDescriptions bool
}

// QueryOption provides an option to Query.
type QueryOption func(options *queryOptions)

// OmitDescriptions sets options.OmitDescriptions.
func OmitDescriptions() QueryOption {
	return func(options *queryOptions) {
		options.OmitDescriptions = true
	}
}

var queryTemplate = template.Must(template.New("IntrospectionQuery").Parse(`
		{{define "description"}}{{if not .OmitDescriptions}}description{{end}}{{end}}
    query IntrospectionQuery {
      __schema {
        queryType { name }
        mutationType { name }
        subscriptionType { name }
        types {
          ...FullType
        }
        directives {
          name
          {{template "description" .}}
          locations
          args {
            ...InputValue
          }
        }
      }
    }

    fragment FullType on __Type {
      kind
      name
      {{template "description" .}}
      fields(includeDeprecated: true) {
        name
        {{template "description" .}}
        args {
          ...InputValue
        }
        type {
          ...TypeRef
        }
        isDeprecated
        deprecationReason
      }
      inputFields {
        ...InputValue
      }
      interfaces {
        ...TypeRef
      }
      enumValues(includeDeprecated: true) {
        name
        {{template "description" .}}
        isDeprecated
        deprecationReason
      }
      possibleTypes {
        ...TypeRef
      }
    }

    fragment InputValue on __InputValue {
      name
      {{template "description" .}}
      type { ...TypeRef }
      defaultValue
    }

    fragment TypeRef on __Type {
      kind
      name
      ofType {
        kind
        name
        ofType {
          kind
          name
          ofType {
            kind
            name
            ofType {
              kind
              name
              ofType {
                kind
                name
                ofType {
                  kind
                  name
                  ofType {
                    kind
                    name
                  }
                }
              }
            }
          }
        }
      }
    }
  `))

// Query constructs a GraphQL document for querying GraphQL schema introspection system.
func Query(options ...QueryOption) string {
	var config queryOptions
	// Apply options.
	for _, option := range options {
		option(&config)
	}

	var buf util.StringBuilder
	if err := queryTemplate.Execute(&buf, &config); err != nil {
		panic(fmt.Sprintf("failed to construct introspection query with option: %+v: %s", config, err))
	}
	return buf.String()
}
