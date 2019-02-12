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

package jsonwriter_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"unicode"

	"github.com/botobag/artemis/internal/util"
	"github.com/botobag/artemis/jsonwriter"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func typeSwitchEncode(stream *jsonwriter.Stream, value interface{}) {
	switch value := value.(type) {
	case map[string]interface{}:
		if len(value) == 0 {
			stream.WriteEmptyObject()
		} else {
			first := true
			stream.WriteObjectStart()
			for k, v := range value {
				if first {
					first = false
				} else {
					stream.WriteMore()
				}
				stream.WriteObjectField(k)
				typeSwitchEncode(stream, v)
			}
			stream.WriteObjectEnd()
		}

	case []interface{}:
		if len(value) == 0 {
			stream.WriteEmptyArray()
		} else {
			stream.WriteArrayStart()
			typeSwitchEncode(stream, value[0])
			for i := 1; i < len(value); i++ {
				stream.WriteMore()
				typeSwitchEncode(stream, value[i])
			}
			stream.WriteArrayEnd()
		}

	default:
		stream.WriteInterface(value)
	}
}

func testStreamWithExpected(value interface{}, expected string) {
	var (
		buf    util.StringBuilder
		stream = jsonwriter.NewStream(&buf)
	)

	typeSwitchEncode(stream, value)

	// Check output.
	Expect(stream.Flush()).ShouldNot(HaveOccurred())
	Expect(buf.String()).Should(MatchJSON(expected), "value = %+v", value)
}

func testStream(value interface{}) {
	// Compare output with the one from encoding/json.Marshal.
	expected, err := json.Marshal(value)
	Expect(err).ShouldNot(HaveOccurred())
	testStreamWithExpected(value, string(expected))
}

type (
	BoolAlias    bool
	IntAlias     int
	Int8Alias    int8
	Int16Alias   int16
	Int32Alias   int32
	Int64Alias   int64
	UintAlias    uint
	Uint8Alias   uint8
	Uint16Alias  uint16
	Uint32Alias  uint32
	Uint64Alias  uint64
	Float32Alias float32
	Float64Alias float64
	StringAlias  string
)

// https://go.googlesource.com/go/+/5fae09b/src/encoding/json/example_marshaling_test.go
type Animal int

const (
	Gopher Animal = iota
	Zebra
)

func (a Animal) MarshalJSONTo(stream *jsonwriter.Stream) error {
	switch a {
	case Gopher:
		stream.WriteString("gopher")
	case Zebra:
		stream.WriteString("zebra")
	default:
		return fmt.Errorf("unknown animal: %d", a)
	}
	return nil
}

type NilMarshaler struct{}

func (*NilMarshaler) MarshalJSONTo(stream *jsonwriter.Stream) error {
	panic("unreachable")
}

// MarshalJSON implements encoding/json.Marshaler.
func (a Animal) MarshalJSON() ([]byte, error) {
	return jsonwriter.Marshal(a)
}

type GoJSONMarshaler struct {
	b   []byte
	err error
}

func (marshaler *GoJSONMarshaler) MarshalJSON() ([]byte, error) {
	return marshaler.b, marshaler.err
}

var _ = Describe("Stream", func() {
	It("encodes simple but special string with single byte", func() {
		// Tests from https://go.googlesource.com/go/+/5fae09b/src/encoding/json/encode_test.go.
		var encodeStringTests = []struct {
			in  string
			out string
		}{
			{"\x00", `"\u0000"`},
			{"\x01", `"\u0001"`},
			{"\x02", `"\u0002"`},
			{"\x03", `"\u0003"`},
			{"\x04", `"\u0004"`},
			{"\x05", `"\u0005"`},
			{"\x06", `"\u0006"`},
			{"\x07", `"\u0007"`},
			{"\x08", `"\u0008"`},
			{"\x09", `"\t"`},
			{"\x0a", `"\n"`},
			{"\x0b", `"\u000b"`},
			{"\x0c", `"\u000c"`},
			{"\x0d", `"\r"`},
			{"\x0e", `"\u000e"`},
			{"\x0f", `"\u000f"`},
			{"\x10", `"\u0010"`},
			{"\x11", `"\u0011"`},
			{"\x12", `"\u0012"`},
			{"\x13", `"\u0013"`},
			{"\x14", `"\u0014"`},
			{"\x15", `"\u0015"`},
			{"\x16", `"\u0016"`},
			{"\x17", `"\u0017"`},
			{"\x18", `"\u0018"`},
			{"\x19", `"\u0019"`},
			{"\x1a", `"\u001a"`},
			{"\x1b", `"\u001b"`},
			{"\x1c", `"\u001c"`},
			{"\x1d", `"\u001d"`},
			{"\x1e", `"\u001e"`},
			{"\x1f", `"\u001f"`},
			{"\x22", `"\""`},
			{"\x27", `"'"`},
		}

		for _, tt := range encodeStringTests {
			testStreamWithExpected(tt.in, tt.out)
		}
	})

	It("encodes nil", func() {
		testStream(nil)
	})

	It("encodes integers", func() {
		testStream(int8(math.MaxInt8))
		testStream(int8(math.MinInt8))
		testStream(int16(math.MaxInt16))
		testStream(int16(math.MinInt16))
		testStream(int32(math.MaxInt32))
		testStream(int32(math.MinInt32))
		testStream(int64(math.MaxInt64))
		testStream(int64(math.MinInt64))

		testStream(uint8(math.MaxUint8))
		testStream(uint16(math.MaxUint16))
		testStream(uint32(math.MaxUint32))
		testStream(uint64(math.MaxUint64))

		testStream(uint8(0))
		testStream(uint16(0))
		testStream(uint32(0))
		testStream(uint64(0))

		// https://groups.google.com/forum/#!msg/golang-nuts/a9PitPAHSSU/ziQw1-QHw3EJ
		const MaxUint = ^uint(0)
		const MinUint = 0
		const MaxInt = int(MaxUint >> 1)
		const MinInt = -MaxInt - 1

		testStream(int(MaxInt))
		testStream(int(MinInt))
		testStream(uint(MaxUint))
		testStream(uint(MinUint))

		testStream(IntAlias(-1))
		testStream(Int8Alias(-12))
		testStream(Int16Alias(-123))
		testStream(Int32Alias(-1234))
		testStream(Int64Alias(-12345))

		testStream(UintAlias(1))
		testStream(Uint8Alias(12))
		testStream(Uint16Alias(123))
		testStream(Uint32Alias(1234))
		testStream(Uint64Alias(12345))
	})

	It("encodes floats", func() {
		testStream(float32(0.1))
		testStream(float64(0.1))
		testStream(float32(3.14))
		testStream(float64(3.14))
		testStream(float32(1e-9))
		testStream(float64(1e-9))

		testStream(float32(math.MaxFloat32))
		testStream(float32(math.SmallestNonzeroFloat32))

		testStream(float64(math.MaxFloat64))
		testStream(float64(math.SmallestNonzeroFloat64))

		testStream(Float32Alias(math.E))
		testStream(Float64Alias(math.Pi))
	})

	It("encodes string-like type", func() {
		testStream(StringAlias("hello"))
	})

	It("encodes pointers", func() {
		testStream((*bool)(nil))
		testStream((*int)(nil))
		testStream((*int8)(nil))
		testStream((*int16)(nil))
		testStream((*int32)(nil))
		testStream((*int64)(nil))
		testStream((*uint)(nil))
		testStream((*uint8)(nil))
		testStream((*uint16)(nil))
		testStream((*uint32)(nil))
		testStream((*uint64)(nil))
		testStream((*float32)(nil))
		testStream((*float64)(nil))
		testStream((*string)(nil))

		testStream((*BoolAlias)(nil))
		testStream((*IntAlias)(nil))
		testStream((*Int8Alias)(nil))
		testStream((*Int16Alias)(nil))
		testStream((*Int32Alias)(nil))
		testStream((*Int64Alias)(nil))
		testStream((*UintAlias)(nil))
		testStream((*Uint8Alias)(nil))
		testStream((*Uint16Alias)(nil))
		testStream((*Uint32Alias)(nil))
		testStream((*Uint64Alias)(nil))
		testStream((*Float32Alias)(nil))
		testStream((*Float64Alias)(nil))
		testStream((*StringAlias)(nil))

		var (
			b   bool
			i   int
			i8  int8
			i16 int16
			i32 int32
			i64 int64
			u   uint
			u8  uint8
			u16 uint16
			u32 uint32
			u64 uint64
			s   string
			f32 float32
			f64 float64
		)

		testStream(&b)
		testStream(&i)
		testStream(&i8)
		testStream(&i16)
		testStream(&i32)
		testStream(&i64)
		testStream(&u)
		testStream(&u8)
		testStream(&u16)
		testStream(&u32)
		testStream(&u64)
		testStream(&s)
		testStream(&f32)
		testStream(&f64)

		var (
			bAlias   BoolAlias
			iAlias   IntAlias
			i8Alias  Int8Alias
			i16Alias Int16Alias
			i32Alias Int32Alias
			i64Alias Int64Alias
			uAlias   UintAlias
			u8Alias  Uint8Alias
			u16Alias Uint16Alias
			u32Alias Uint32Alias
			u64Alias Uint64Alias
			sAlias   StringAlias
			f32Alias Float32Alias
			f64Alias Float64Alias
		)

		testStream(&bAlias)
		testStream(&iAlias)
		testStream(&i8Alias)
		testStream(&i16Alias)
		testStream(&i32Alias)
		testStream(&i64Alias)
		testStream(&uAlias)
		testStream(&u8Alias)
		testStream(&u16Alias)
		testStream(&u32Alias)
		testStream(&u64Alias)
		testStream(&sAlias)
		testStream(&f32Alias)
		testStream(&f64Alias)
	})

	It("encodes arrays", func() {
		// Empty array
		testStream([]interface{}{})

		testStream([]interface{}{
			"a",
			1,
			"c",
			nil,
		})
	})

	It("encodes object", func() {
		// Empty object
		testStream(map[string]interface{}{})

		testStream(map[string]interface{}{
			"K": "Kelvin",
			"ß": "long s",
		})
	})

	It("encodes bool", func() {
		testStream(true)
		testStream(false)
		testStream(BoolAlias(true))
		testStream(BoolAlias(false))
	})

	It("escapes HTML characters", func() {
		// https://go.googlesource.com/go/+/5fae09b/src/encoding/json/encode_test.go#635
		testStreamWithExpected(
			`<html>foo &`+"\xe2\x80\xa8 \xe2\x80\xa9"+`</html>`,
			`"\u003chtml\u003efoo \u0026\u2028 \u2029\u003c/html\u003e"`,
		)
	})

	It("encodes value that implements json.Marshaler", func() {
		testStream(&GoJSONMarshaler{
			b: []byte{'"', 'h', 'e', 'l', 'l', 'o', '"'},
		})

		stream := jsonwriter.NewStream(&util.StringBuilder{})
		stream.WriteInterface(&GoJSONMarshaler{
			err: errors.New("test marshaler error"),
		})

		err := stream.Flush()
		Expect(err).Should(HaveOccurred())
		Expect(err.Error()).Should(ContainSubstring("test marshaler error"))
	})

	It("accepts invalid utf8", func() {
		var r []rune
		for i := '\u0000'; i <= unicode.MaxRune; i++ {
			r = append(r, i)
		}
		s := string(r) + "\xff\xff\xffhello" // some invalid UTF-8 too
		testStream(s)
	})

	It("encodes arbitrary types with encoding.json", func() {
		// https://go.googlesource.com/go/+/5fae09b/src/encoding/json/encode_test.go#58
		type Optionals struct {
			Sr  string                 `json:"sr"`
			So  string                 `json:"so,omitempty"`
			Sw  string                 `json:"-"`
			Ir  int                    `json:"omitempty"` // actually named omitempty, not an option
			Io  int                    `json:"io,omitempty"`
			Slr []string               `json:"slr,random"`
			Slo []string               `json:"slo,omitempty"`
			Mr  map[string]interface{} `json:"mr"`
			Mo  map[string]interface{} `json:",omitempty"`
			Fr  float64                `json:"fr"`
			Fo  float64                `json:"fo,omitempty"`
			Br  bool                   `json:"br"`
			Bo  bool                   `json:"bo,omitempty"`
			Ur  uint                   `json:"ur"`
			Uo  uint                   `json:"uo,omitempty"`
			Str struct{}               `json:"str"`
			Sto struct{}               `json:"sto,omitempty"`
		}

		const optionalsExpected = `{
 "sr": "",
 "omitempty": 0,
 "slr": null,
 "mr": {},
 "fr": 0,
 "br": false,
 "ur": 0,
 "str": {},
 "sto": {}
}`

		var o Optionals
		o.Sw = "something"
		o.Mr = map[string]interface{}{}
		o.Mo = map[string]interface{}{}

		testStreamWithExpected(o, strings.Map(func(r rune) rune {
			if unicode.IsSpace(r) {
				return -1
			}
			return r
		}, optionalsExpected))
	})

	It("fails to encode unsupported values", func() {
		// https://go.googlesource.com/go/+/5fae09b/src/encoding/json/encode_test.go#137
		var unsupportedValues = []interface{}{
			math.NaN(),
			math.Inf(-1),
			math.Inf(1),
			float32(math.NaN()),
			float32(math.Inf(-1)),
			float32(math.Inf(1)),
		}

		for _, v := range unsupportedValues {
			stream := jsonwriter.NewStream(&util.StringBuilder{})
			stream.WriteInterface(v)
			Expect(stream.Flush()).Should(HaveOccurred(), "value: %+v, type: %T", v, v)
		}
	})

	Context("encodes values that implements custom marshaler", func() {
		It("simple values", func() {
			// https://go.googlesource.com/go/+/5fae09b/src/encoding/json/example_marshaling_test.go
			zoo := []interface{}{
				Gopher,
				Zebra,
				Gopher,
				Gopher,
				Zebra,
			}
			testStream(zoo)
		})

		It("encodes to null", func() {
			var nilMarshaler *NilMarshaler
			testStreamWithExpected(nilMarshaler, "null")
		})

		It("returns error when custom marshaler failed", func() {
			var (
				zoo = []interface{}{
					Gopher,
					Animal(123),
					Zebra,
					Gopher,
					Gopher,
					Zebra,
				}
				stream = jsonwriter.NewStream(&util.StringBuilder{})
			)

			typeSwitchEncode(stream, zoo)
			Expect(stream.Flush()).ShouldNot(Succeed())

			streamErr := stream.Error()
			Expect(streamErr.Error()).Should(ContainSubstring("unknown animal: 123"))

			_, err := json.Marshal(zoo)
			Expect(err).Should(MatchError(streamErr))
		})
	})
})
