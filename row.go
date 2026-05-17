package kalodb

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
)

type Schema struct {
	Table string
	Cols  []Column
	PKey  []int
}

type Column struct {
	Name string
	Type CellType
}

type Row []Cell

func (schema *Schema) NewRow() Row {
	return make(Row, len(schema.Cols))
}

func (row Row) EncodeKey(schema *Schema) (key []byte) {
	key = append([]byte(schema.Table), 0x00)
	for i := range schema.Cols {
		if !slices.Contains(schema.PKey, i) {
			continue
		}

		key = row[i].Encode(key)
	}

	return key
}

func (row Row) EncodeVal(schema *Schema) (val []byte) {
	for i := range schema.Cols {
		if slices.Contains(schema.PKey, i) {
			continue
		}

		val = append(val, row[i].Encode(val)...)
	}
	return val
}

func (row Row) DecodeKey(schema *Schema, key []byte) (err error) {
	splitIndex := bytes.IndexByte(key, 0x00)
	if splitIndex == -1 {
		return errors.New("key bytes invalid")
	}

	tableName := string(key[0:splitIndex])
	if schema.Table != tableName {
		return errors.New("table name does not match")
	}

	keyData := key[splitIndex+1:]
	for _, i := range schema.PKey {
		row[i] = Cell{
			Type: schema.Cols[i].Type,
		}
		keyData, err = row[i].Decode(keyData)
		if err != nil {
			return fmt.Errorf("cannot decode key: %w", err)
		}
	}

	if len(keyData) != 0 {
		return errors.New("invalid number of key len")
	}

	return nil
}

func (row Row) DecodeVal(schema *Schema, val []byte) (err error) {
	for i, col := range schema.Cols {
		if slices.Contains(schema.PKey, i) {
			continue
		}

		row[i] = Cell{
			Type: col.Type,
		}

		val, err = row[i].Decode(val)
		if err != nil {
			return err
		}
	}

	if len(val) != 0 {
		return errors.New("invalid number of val len")
	}
	return err
}
