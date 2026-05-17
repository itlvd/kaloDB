package kalodb

import (
	"bytes"
	"encoding/binary"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTableCell(t *testing.T) {
	cell := Cell{Type: TypeI64, I64: -2}
	data := []byte{0xfe, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	assert.Equal(t, data, cell.Encode(nil))
	decoded := Cell{Type: TypeI64}
	rest, err := decoded.Decode(data)
	assert.True(t, len(rest) == 0 && err == nil)
	assert.Equal(t, cell, decoded)

	cell = Cell{Type: TypeStr, Str: []byte("asdf")}
	data = []byte{4, 0, 0, 0, 'a', 's', 'd', 'f'}
	assert.Equal(t, data, cell.Encode(nil))
	decoded = Cell{Type: TypeStr}
	rest, err = decoded.Decode(data)
	assert.True(t, len(rest) == 0 && err == nil)
	assert.Equal(t, cell, decoded)
}

func TestCell_Encode_I64(t *testing.T) {
	tests := []struct {
		name string
		val  int64
	}{
		{"zero", 0},
		{"positive", 42},
		{"negative", -42},
		{"max", math.MaxInt64},
		{"min", math.MinInt64},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := &Cell{Type: TypeI64, I64: tc.val}
			got := c.Encode(nil)

			if len(got) != 8 {
				t.Fatalf("expected 8 bytes, got %d", len(got))
			}
			want := int64(binary.LittleEndian.Uint64(got))
			if want != tc.val {
				t.Errorf("encoded value = %d, want %d", want, tc.val)
			}
		})
	}
}

func TestCell_Encode_Str(t *testing.T) {
	tests := []struct {
		name string
		val  []byte
	}{
		{"empty", []byte("")},
		{"short", []byte("hello")},
		{"unicode", []byte("xin chào 🌏")},
		{"long", bytes.Repeat([]byte("a"), 10_000)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := &Cell{Type: TypeStr, Str: tc.val}
			got := c.Encode(nil)

			if len(got) < 4 {
				t.Fatalf("encoded output too small: %d bytes", len(got))
			}
			gotLen := binary.LittleEndian.Uint32(got[:4])
			if int(gotLen) != len(tc.val) {
				t.Errorf("length header = %d, want %d", gotLen, len(tc.val))
			}
			if !bytes.Equal(got[4:], tc.val) {
				t.Errorf("payload mismatch: got %q, want %q", got[4:], tc.val)
			}
		})
	}
}

func TestCell_Encode_AppendsToExisting(t *testing.T) {
	prefix := []byte{0xDE, 0xAD, 0xBE, 0xEF}

	c := &Cell{Type: TypeI64, I64: 1}
	got := c.Encode(prefix)

	if !bytes.HasPrefix(got, prefix) {
		t.Fatalf("expected output to start with prefix %x, got %x", prefix, got)
	}
	if len(got) != len(prefix)+8 {
		t.Errorf("expected length %d, got %d", len(prefix)+8, len(got))
	}
}

func TestCell_Encode_UnsupportedTypePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic for unsupported type, got none")
		}
	}()

	c := &Cell{Type: CellType(99)}
	_ = c.Encode(nil)
}

func TestCell_Decode_I64(t *testing.T) {
	in := &Cell{Type: TypeI64, I64: -1234567890}
	encoded := in.Encode(nil)

	out := &Cell{Type: TypeI64}
	rest, err := out.Decode(encoded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rest) != 0 {
		t.Errorf("expected empty rest, got %d bytes", len(rest))
	}
	if out.I64 != in.I64 {
		t.Errorf("got %d, want %d", out.I64, in.I64)
	}
}

func TestCell_Decode_I64_LeavesTrailingBytes(t *testing.T) {
	in := &Cell{Type: TypeI64, I64: 7}
	encoded := in.Encode(nil)
	encoded = append(encoded, 0x01, 0x02, 0x03)

	out := &Cell{Type: TypeI64}
	rest, err := out.Decode(encoded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(rest, []byte{0x01, 0x02, 0x03}) {
		t.Errorf("unexpected rest: %x", rest)
	}
}

func TestCell_Decode_I64_TooShort(t *testing.T) {
	out := &Cell{Type: TypeI64}
	short := []byte{1, 2, 3}

	rest, err := out.Decode(short)
	if err == nil {
		t.Fatal("expected error for short input, got nil")
	}
	if !bytes.Equal(rest, short) {
		t.Errorf("on error, rest should equal input; got %x, want %x", rest, short)
	}
}

func TestCell_Decode_Str(t *testing.T) {
	in := &Cell{Type: TypeStr, Str: []byte("kalodb")}
	encoded := in.Encode(nil)

	out := &Cell{Type: TypeStr}
	rest, err := out.Decode(encoded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rest) != 0 {
		t.Errorf("expected empty rest, got %d bytes", len(rest))
	}
	if !bytes.Equal(out.Str, in.Str) {
		t.Errorf("got %q, want %q", out.Str, in.Str)
	}
}

func TestCell_Decode_Str_Empty(t *testing.T) {
	in := &Cell{Type: TypeStr, Str: []byte("")}
	encoded := in.Encode(nil)

	out := &Cell{Type: TypeStr}
	rest, err := out.Decode(encoded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rest) != 0 {
		t.Errorf("expected empty rest, got %d", len(rest))
	}
	if len(out.Str) != 0 {
		t.Errorf("expected empty string, got %q", out.Str)
	}
}

func TestCell_Decode_Str_HeaderTooShort(t *testing.T) {
	out := &Cell{Type: TypeStr}
	short := []byte{1, 2}

	rest, err := out.Decode(short)
	if err == nil {
		t.Fatal("expected error for short header, got nil")
	}
	if !bytes.Equal(rest, short) {
		t.Errorf("on error, rest should equal input; got %x", rest)
	}
}

func TestCell_Decode_Str_PayloadTooShort(t *testing.T) {
	buf := make([]byte, 0, 7)
	buf = binary.LittleEndian.AppendUint32(buf, 10)
	buf = append(buf, []byte("abc")...)

	out := &Cell{Type: TypeStr}
	rest, err := out.Decode(buf)
	if err == nil {
		t.Fatal("expected error for truncated payload, got nil")
	}
	if !bytes.Equal(rest, buf) {
		t.Errorf("on error, rest should equal input")
	}
}

func TestCell_Decode_Str_IsClonedNotAliased(t *testing.T) {
	in := &Cell{Type: TypeStr, Str: []byte("hello")}
	encoded := in.Encode(nil)

	out := &Cell{Type: TypeStr}
	if _, err := out.Decode(encoded); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for i := range encoded {
		encoded[i] = 0xFF
	}
	if string(out.Str) != "hello" {
		t.Errorf("decoded Str was aliased to input buffer; got %q", out.Str)
	}
}

func TestCell_Decode_UnsupportedType(t *testing.T) {
	out := &Cell{Type: CellType(99)}
	data := []byte{1, 2, 3, 4}

	rest, err := out.Decode(data)
	if err == nil {
		t.Fatal("expected error for unsupported type, got nil")
	}
	if !bytes.Equal(rest, data) {
		t.Errorf("rest should equal input on error")
	}
}

func TestCell_RoundTrip(t *testing.T) {
	cells := []Cell{
		{Type: TypeI64, I64: 0},
		{Type: TypeI64, I64: math.MaxInt64},
		{Type: TypeI64, I64: math.MinInt64},
		{Type: TypeStr, Str: []byte("")},
		{Type: TypeStr, Str: []byte("hello world")},
		{Type: TypeStr, Str: []byte(strings.Repeat("x", 1024))},
	}

	var buf []byte
	for i := range cells {
		buf = cells[i].Encode(buf)
	}

	rest := buf
	for i, want := range cells {
		got := &Cell{Type: want.Type}
		var err error
		rest, err = got.Decode(rest)
		if err != nil {
			t.Fatalf("cell %d: decode error: %v", i, err)
		}
		switch want.Type {
		case TypeI64:
			if got.I64 != want.I64 {
				t.Errorf("cell %d: I64 got %d, want %d", i, got.I64, want.I64)
			}
		case TypeStr:
			if !bytes.Equal(got.Str, want.Str) {
				t.Errorf("cell %d: Str got %q, want %q", i, got.Str, want.Str)
			}
		}
	}
	if len(rest) != 0 {
		t.Errorf("expected fully consumed buffer, %d bytes remaining", len(rest))
	}
}
