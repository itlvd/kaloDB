package kalodb

import (
	"encoding/binary"
	"errors"
	"slices"
)

type CellType uint8

const (
	TypeI64 CellType = 1
	TypeStr CellType = 2
)

type Cell struct {
	Type CellType
	I64  int64
	Str  []byte
}

func (cell *Cell) Encode(toAppend []byte) []byte {
	switch cell.Type {
	case TypeI64:
		return binary.LittleEndian.AppendUint64(toAppend, uint64(cell.I64))
	case TypeStr:
		lenHeader := len(cell.Str)
		toAppend = binary.LittleEndian.AppendUint32(toAppend, uint32(lenHeader))
		return append(toAppend, cell.Str...)
	default:
		panic("type does not supported")
	}
}

func (cell *Cell) Decode(data []byte) (rest []byte, err error) {
	switch cell.Type {
	case TypeI64:
		if len(data) < 8 {
			err = errors.New("missing information TypeInt64")
			return data, err
		} else {
			cell.I64 = int64(binary.LittleEndian.Uint64(data[0:8]))
			return data[8:], nil
		}

	case TypeStr:
		if len(data) < 4 {
			err = errors.New("missing information TypeStr")
			return data, err
		}
		lenStr := binary.LittleEndian.Uint32(data[0:4])
		if len(data[4:]) < int(lenStr) {
			err = errors.New("length string and len string metadata does not match")
			return data, err
		}

		cell.Str = slices.Clone(data[4 : 4+lenStr])
		return data[4+lenStr:], nil
	default:
		err = errors.New("this type currently does not supported")
	}

	return data, err
}
