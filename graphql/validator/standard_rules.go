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

package validator

import (
	"fmt"
	"runtime"
	"strings"
)

// This set includes all validation rules defined by the GraphQL spec. Note that we cannot access
// "rules" package from here because this would result a import cycle. To resolve this issue, we
// expose a InitStandardRules to set up this variable and only allow the caller from init function
// in rules package.
var standardRules *rules

// InitStandardRules initializes standardRules. It can only be called from rules package.
func InitStandardRules(rules ...interface{}) {
	pc, _, _, ok := runtime.Caller(1)
	if ok {
		f := runtime.FuncForPC(pc)
		if f != nil && !strings.HasPrefix(f.Name(), "github.com/botobag/artemis/graphql/validator/rules.init") {
			panic(fmt.Sprintf(`validator.InitStandardRules is not allowed to be called from "%s".`, f.Name()))
		}
	}

	standardRules = buildRules(rules...)
}

// StandardRules returns rule set that required by specification for validating query documents.
func StandardRules() *rules {
	if standardRules == nil {
		pc, f, _, ok := runtime.Caller(1)
		if ok {
			// validator.Validate also calls StandardRules to obtain the standard rules. In this case,
			// skip one more frame to get the actual caller.
			if fu := runtime.FuncForPC(pc); fu != nil && fu.Name() == "github.com/botobag/artemis/graphql/validator.Validate" {
				_, f, _, ok = runtime.Caller(2)
			}
		}

		if ok {
			f = fmt.Sprintf(`in "%s"`, f)
		} else {
			f = "at where validator.StandardRules being called"
		}
		panic(fmt.Sprintf(`Please import "github.com/botobag/artemis/graphql/validator/rules" %s for loading standard validation rules:

import (
	...

	// Load standard rules required by specification for validating queries.
	_ "github.com/botobag/artemis/graphql/validator/rules"
)
`, f))
	}
	return standardRules
}
