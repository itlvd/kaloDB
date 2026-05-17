package kalodb

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func linkSchema() *Schema {
	return &Schema{
		Table: "link",
		Cols: []Column{
			{Name: "time", Type: TypeI64},
			{Name: "src", Type: TypeStr},
			{Name: "dst", Type: TypeStr},
		},
		PKey: []int{1, 2},
	}
}

func linkRow() Row {
	return Row{
		Cell{Type: TypeI64, I64: 123},
		Cell{Type: TypeStr, Str: []byte("a")},
		Cell{Type: TypeStr, Str: []byte("b")},
	}
}

func TestNewRow(t *testing.T) {
	schema := linkSchema()
	row := schema.NewRow()

	assert.Equal(t, len(schema.Cols), len(row))
	for _, c := range row {
		assert.Equal(t, Cell{}, c)
	}
}

func TestRowEncode(t *testing.T) {
	schema := linkSchema()
	row := linkRow()

	key := []byte{'l', 'i', 'n', 'k', 0, 1, 0, 0, 0, 'a', 1, 0, 0, 0, 'b'}
	val := []byte{123, 0, 0, 0, 0, 0, 0, 0}

	assert.Equal(t, key, row.EncodeKey(schema))
	assert.Equal(t, val, row.EncodeVal(schema))

	decoded := schema.NewRow()
	assert.Nil(t, decoded.DecodeKey(schema, key))
	assert.Nil(t, decoded.DecodeVal(schema, val))
	assert.Equal(t, row, decoded)
}

func TestEncodeKey_FollowsColumnOrderNotPKeyOrder(t *testing.T) {
	schema := linkSchema()
	schema.PKey = []int{2, 1}

	row := linkRow()
	key := []byte{'l', 'i', 'n', 'k', 0, 1, 0, 0, 0, 'a', 1, 0, 0, 0, 'b'}

	assert.Equal(t, key, row.EncodeKey(schema))
}

func TestRowEncode_LongerStrings(t *testing.T) {
	schema := linkSchema()
	row := Row{
		Cell{Type: TypeI64, I64: 0},
		Cell{Type: TypeStr, Str: []byte("hello")},
		Cell{Type: TypeStr, Str: []byte("")},
	}

	key := []byte{
		'l', 'i', 'n', 'k', 0,
		5, 0, 0, 0, 'h', 'e', 'l', 'l', 'o',
		0, 0, 0, 0,
	}
	val := []byte{0, 0, 0, 0, 0, 0, 0, 0}

	assert.Equal(t, key, row.EncodeKey(schema))
	assert.Equal(t, val, row.EncodeVal(schema))

	decoded := schema.NewRow()
	assert.Nil(t, decoded.DecodeKey(schema, key))
	assert.Nil(t, decoded.DecodeVal(schema, val))
	assert.Equal(t, row, decoded)
}

func TestRowEncode_NegativeInt(t *testing.T) {
	schema := linkSchema()
	row := Row{
		Cell{Type: TypeI64, I64: -1},
		Cell{Type: TypeStr, Str: []byte("x")},
		Cell{Type: TypeStr, Str: []byte("y")},
	}

	encKey := row.EncodeKey(schema)
	encVal := row.EncodeVal(schema)

	decoded := schema.NewRow()
	assert.Nil(t, decoded.DecodeKey(schema, encKey))
	assert.Nil(t, decoded.DecodeVal(schema, encVal))
	assert.Equal(t, row, decoded)
}

func TestRowEncode_SinglePK(t *testing.T) {
	schema := &Schema{
		Table: "kv",
		Cols: []Column{
			{Name: "k", Type: TypeStr},
			{Name: "v", Type: TypeStr},
		},
		PKey: []int{0},
	}
	row := Row{
		Cell{Type: TypeStr, Str: []byte("hello")},
		Cell{Type: TypeStr, Str: []byte("world")},
	}

	key := []byte{
		'k', 'v', 0,
		5, 0, 0, 0, 'h', 'e', 'l', 'l', 'o',
	}
	val := []byte{
		5, 0, 0, 0, 'w', 'o', 'r', 'l', 'd',
	}

	assert.Equal(t, key, row.EncodeKey(schema))
	assert.Equal(t, val, row.EncodeVal(schema))

	decoded := schema.NewRow()
	assert.Nil(t, decoded.DecodeKey(schema, key))
	assert.Nil(t, decoded.DecodeVal(schema, val))
	assert.Equal(t, row, decoded)
}

func TestDecodeKey_DoesNotTouchNonPKColumns(t *testing.T) {
	schema := linkSchema()
	key := []byte{'l', 'i', 'n', 'k', 0, 1, 0, 0, 0, 'a', 1, 0, 0, 0, 'b'}

	decoded := schema.NewRow()
	assert.Nil(t, decoded.DecodeKey(schema, key))

	assert.Equal(t, Cell{}, decoded[0])
	assert.Equal(t, Cell{Type: TypeStr, Str: []byte("a")}, decoded[1])
	assert.Equal(t, Cell{Type: TypeStr, Str: []byte("b")}, decoded[2])
}

func TestDecodeVal_DoesNotTouchPKColumns(t *testing.T) {
	schema := linkSchema()
	val := []byte{123, 0, 0, 0, 0, 0, 0, 0}

	decoded := schema.NewRow()
	assert.Nil(t, decoded.DecodeVal(schema, val))

	assert.Equal(t, Cell{Type: TypeI64, I64: 123}, decoded[0])
	assert.Equal(t, Cell{}, decoded[1])
	assert.Equal(t, Cell{}, decoded[2])
}

func TestDecodeKey_BadPrefix(t *testing.T) {
	schema := linkSchema()

	cases := map[string][]byte{
		"empty":        nil,
		"too short":    []byte("lnk"),
		"wrong table":  []byte("node\x00rest"),
		"missing null": []byte("linkXrest"),
	}
	for name, key := range cases {
		t.Run(name, func(t *testing.T) {
			row := schema.NewRow()
			err := row.DecodeKey(schema, key)
			assert.Error(t, err)
		})
	}
}

func TestDecodeKey_TrailingGarbage(t *testing.T) {
	schema := linkSchema()
	key := []byte{'l', 'i', 'n', 'k', 0, 1, 0, 0, 0, 'a', 1, 0, 0, 0, 'b', 0xFF}

	row := schema.NewRow()
	err := row.DecodeKey(schema, key)
	assert.Error(t, err)
}

func TestDecodeVal_TrailingGarbage(t *testing.T) {
	schema := linkSchema()
	val := []byte{123, 0, 0, 0, 0, 0, 0, 0, 0xFF}

	row := schema.NewRow()
	err := row.DecodeVal(schema, val)
	assert.Error(t, err)
}

func TestDecodeVal_Truncated(t *testing.T) {
	schema := linkSchema()
	val := []byte{123, 0, 0, 0, 0, 0, 0}

	row := schema.NewRow()
	err := row.DecodeVal(schema, val)
	assert.Error(t, err)
}
