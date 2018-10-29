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

// This file is copied from https://github.com/golang/go/blob/b2e66f1/src/strings/builder_test.go
// for testing the StringBuilder implementation we copied from Go's upstream. There are two changes
// we've made:
//
//  1. Rename `Builder` to `util.StringBuilder`
//  2. Comment out a call to `testing.T.Helper()` in `check()` because Go pre-1.9 doesn't support
//     it.
//
// The Go's license is reproduced below.

/**
 * Copyright (c) 2009 The Go Authors. All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions are
 * met:
 *
 *    * Redistributions of source code must retain the above copyright
 * notice, this list of conditions and the following disclaimer.
 *    * Redistributions in binary form must reproduce the above
 * copyright notice, this list of conditions and the following disclaimer
 * in the documentation and/or other materials provided with the
 * distribution.
 *    * Neither the name of Google Inc. nor the names of its
 * contributors may be used to endorse or promote products derived from
 * this software without specific prior written permission.
 *
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
 * "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
 * LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
 * A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
 * OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
 * SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
 * LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
 * DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
 * THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 * (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
 * OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 */

package util_test

import (
	"bytes"
	"testing"

	"github.com/botobag/artemis/internal/util"
)

func check(t *testing.T, b *util.StringBuilder, want string) {
	// Artemis: comment out as it doesn't exist in Go 1.8.
	// t.Helper()
	got := b.String()
	if got != want {
		t.Errorf("String: got %#q; want %#q", got, want)
		return
	}
	if n := b.Len(); n != len(got) {
		t.Errorf("Len: got %d; but len(String()) is %d", n, len(got))
	}
}

func TestBuilder(t *testing.T) {
	var b util.StringBuilder
	check(t, &b, "")
	n, err := b.WriteString("hello")
	if err != nil || n != 5 {
		t.Errorf("WriteString: got %d,%s; want 5,nil", n, err)
	}
	check(t, &b, "hello")
	if err = b.WriteByte(' '); err != nil {
		t.Errorf("WriteByte: %s", err)
	}
	check(t, &b, "hello ")
	n, err = b.WriteString("world")
	if err != nil || n != 5 {
		t.Errorf("WriteString: got %d,%s; want 5,nil", n, err)
	}
	check(t, &b, "hello world")
}

func TestBuilderString(t *testing.T) {
	var b util.StringBuilder
	b.WriteString("alpha")
	check(t, &b, "alpha")
	s1 := b.String()
	b.WriteString("beta")
	check(t, &b, "alphabeta")
	s2 := b.String()
	b.WriteString("gamma")
	check(t, &b, "alphabetagamma")
	s3 := b.String()

	// Check that subsequent operations didn't change the returned strings.
	if want := "alpha"; s1 != want {
		t.Errorf("first String result is now %q; want %q", s1, want)
	}
	if want := "alphabeta"; s2 != want {
		t.Errorf("second String result is now %q; want %q", s2, want)
	}
	if want := "alphabetagamma"; s3 != want {
		t.Errorf("third String result is now %q; want %q", s3, want)
	}
}

func TestBuilderReset(t *testing.T) {
	var b util.StringBuilder
	check(t, &b, "")
	b.WriteString("aaa")
	s := b.String()
	check(t, &b, "aaa")
	b.Reset()
	check(t, &b, "")

	// Ensure that writing after Reset doesn't alter
	// previously returned strings.
	b.WriteString("bbb")
	check(t, &b, "bbb")
	if want := "aaa"; s != want {
		t.Errorf("previous String result changed after Reset: got %q; want %q", s, want)
	}
}

func TestBuilderGrow(t *testing.T) {
	for _, growLen := range []int{0, 100, 1000, 10000, 100000} {
		p := bytes.Repeat([]byte{'a'}, growLen)
		allocs := testing.AllocsPerRun(100, func() {
			var b util.StringBuilder
			b.Grow(growLen) // should be only alloc, when growLen > 0
			b.Write(p)
			if b.String() != string(p) {
				t.Fatalf("growLen=%d: bad data written after Grow", growLen)
			}
		})
		wantAllocs := 1
		if growLen == 0 {
			wantAllocs = 0
		}
		if g, w := int(allocs), wantAllocs; g != w {
			t.Errorf("growLen=%d: got %d allocs during Write; want %v", growLen, g, w)
		}
	}
}

func TestBuilderWrite2(t *testing.T) {
	const s0 = "hello 世界"
	for _, tt := range []struct {
		name string
		fn   func(b *util.StringBuilder) (int, error)
		n    int
		want string
	}{
		{
			"Write",
			func(b *util.StringBuilder) (int, error) { return b.Write([]byte(s0)) },
			len(s0),
			s0,
		},
		{
			"WriteRune",
			func(b *util.StringBuilder) (int, error) { return b.WriteRune('a') },
			1,
			"a",
		},
		{
			"WriteRuneWide",
			func(b *util.StringBuilder) (int, error) { return b.WriteRune('世') },
			3,
			"世",
		},
		{
			"WriteString",
			func(b *util.StringBuilder) (int, error) { return b.WriteString(s0) },
			len(s0),
			s0,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var b util.StringBuilder
			n, err := tt.fn(&b)
			if err != nil {
				t.Fatalf("first call: got %s", err)
			}
			if n != tt.n {
				t.Errorf("first call: got n=%d; want %d", n, tt.n)
			}
			check(t, &b, tt.want)

			n, err = tt.fn(&b)
			if err != nil {
				t.Fatalf("second call: got %s", err)
			}
			if n != tt.n {
				t.Errorf("second call: got n=%d; want %d", n, tt.n)
			}
			check(t, &b, tt.want+tt.want)
		})
	}
}

func TestBuilderWriteByte(t *testing.T) {
	var b util.StringBuilder
	if err := b.WriteByte('a'); err != nil {
		t.Error(err)
	}
	if err := b.WriteByte(0); err != nil {
		t.Error(err)
	}
	check(t, &b, "a\x00")
}

func TestBuilderAllocs(t *testing.T) {
	var b util.StringBuilder
	const msg = "hello"
	b.Grow(len(msg) * 2) // because AllocsPerRun does an extra "warm-up" iteration
	var s string
	allocs := int(testing.AllocsPerRun(1, func() {
		b.WriteString("hello")
		s = b.String()
	}))
	if want := msg + msg; s != want {
		t.Errorf("String: got %#q; want %#q", s, want)
	}
	if allocs > 0 {
		t.Fatalf("got %d alloc(s); want 0", allocs)
	}

	// Issue 23382; verify that copyCheck doesn't force the
	// util.StringBuilder to escape and be heap allocated.
	n := testing.AllocsPerRun(10000, func() {
		var b util.StringBuilder
		b.Grow(5)
		b.WriteString("abcde")
		_ = b.String()
	})
	if n != 1 {
		t.Errorf("util.StringBuilder allocs = %v; want 1", n)
	}
}

func TestBuilderCopyPanic(t *testing.T) {
	tests := []struct {
		name      string
		fn        func()
		wantPanic bool
	}{
		{
			name:      "String",
			wantPanic: false,
			fn: func() {
				var a util.StringBuilder
				a.WriteByte('x')
				b := a
				_ = b.String() // appease vet
			},
		},
		{
			name:      "Len",
			wantPanic: false,
			fn: func() {
				var a util.StringBuilder
				a.WriteByte('x')
				b := a
				b.Len()
			},
		},
		{
			name:      "Reset",
			wantPanic: false,
			fn: func() {
				var a util.StringBuilder
				a.WriteByte('x')
				b := a
				b.Reset()
				b.WriteByte('y')
			},
		},
		{
			name:      "Write",
			wantPanic: true,
			fn: func() {
				var a util.StringBuilder
				a.Write([]byte("x"))
				b := a
				b.Write([]byte("y"))
			},
		},
		{
			name:      "WriteByte",
			wantPanic: true,
			fn: func() {
				var a util.StringBuilder
				a.WriteByte('x')
				b := a
				b.WriteByte('y')
			},
		},
		{
			name:      "WriteString",
			wantPanic: true,
			fn: func() {
				var a util.StringBuilder
				a.WriteString("x")
				b := a
				b.WriteString("y")
			},
		},
		{
			name:      "WriteRune",
			wantPanic: true,
			fn: func() {
				var a util.StringBuilder
				a.WriteRune('x')
				b := a
				b.WriteRune('y')
			},
		},
		{
			name:      "Grow",
			wantPanic: true,
			fn: func() {
				var a util.StringBuilder
				a.Grow(1)
				b := a
				b.Grow(2)
			},
		},
	}
	for _, tt := range tests {
		didPanic := make(chan bool)
		go func() {
			defer func() { didPanic <- recover() != nil }()
			tt.fn()
		}()
		if got := <-didPanic; got != tt.wantPanic {
			t.Errorf("%s: panicked = %v; want %v", tt.name, got, tt.wantPanic)
		}
	}
}

var someBytes = []byte("some bytes sdljlk jsklj3lkjlk djlkjw")

var sinkS string

func benchmarkBuilder(b *testing.B, f func(b *testing.B, numWrite int, grow bool)) {
	b.Run("1Write_NoGrow", func(b *testing.B) {
		b.ReportAllocs()
		f(b, 1, false)
	})
	b.Run("3Write_NoGrow", func(b *testing.B) {
		b.ReportAllocs()
		f(b, 3, false)
	})
	b.Run("3Write_Grow", func(b *testing.B) {
		b.ReportAllocs()
		f(b, 3, true)
	})
}

func BenchmarkBuildString_Builder(b *testing.B) {
	benchmarkBuilder(b, func(b *testing.B, numWrite int, grow bool) {
		for i := 0; i < b.N; i++ {
			var buf util.StringBuilder
			if grow {
				buf.Grow(len(someBytes) * numWrite)
			}
			for i := 0; i < numWrite; i++ {
				buf.Write(someBytes)
			}
			sinkS = buf.String()
		}
	})
}

func BenchmarkBuildString_ByteBuffer(b *testing.B) {
	benchmarkBuilder(b, func(b *testing.B, numWrite int, grow bool) {
		for i := 0; i < b.N; i++ {
			var buf bytes.Buffer
			if grow {
				buf.Grow(len(someBytes) * numWrite)
			}
			for i := 0; i < numWrite; i++ {
				buf.Write(someBytes)
			}
			sinkS = buf.String()
		}
	})
}
