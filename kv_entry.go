package kalodb

import (
	"encoding/binary"
	"io"
)

const (
	keyLenFieldSize = 4
	valLenFieldSize = 4
	deletedFlagSize = 1
)

type Entry struct {
	key     []byte
	val     []byte
	deleted bool
}

func (ent *Entry) Encode() []byte {
	headerSize := keyLenFieldSize + valLenFieldSize + deletedFlagSize
	keyOffset := headerSize
	valOffset := keyOffset + len(ent.key)

	data := make([]byte, headerSize+len(ent.key)+len(ent.val))

	binary.LittleEndian.PutUint32(data[0:keyLenFieldSize], uint32(len(ent.key)))
	binary.LittleEndian.PutUint32(data[keyLenFieldSize:keyLenFieldSize+valLenFieldSize], uint32(len(ent.val)))

	if ent.deleted {
		data[keyLenFieldSize+valLenFieldSize] = 1
	}

	copy(data[keyOffset:valOffset], ent.key)
	copy(data[valOffset:], ent.val)
	return data
}

func (ent *Entry) Decode(r io.Reader) error {
	headerSize := keyLenFieldSize + valLenFieldSize + deletedFlagSize

	header := make([]byte, headerSize)
	_, err := io.ReadFull(r, header)
	if err != nil {
		return err
	}

	if header[headerSize-deletedFlagSize] == 1 {
		ent.deleted = true
	}

	keySize := binary.LittleEndian.Uint32(header[0:keyLenFieldSize])
	valSize := binary.LittleEndian.Uint32(header[keyLenFieldSize : keyLenFieldSize+valLenFieldSize])

	key := make([]byte, keySize)
	if _, err := io.ReadFull(r, key); err != nil {
		return err
	}

	val := make([]byte, valSize)
	if _, err := io.ReadFull(r, val); err != nil {
		return err
	}

	ent.key = key
	ent.val = val

	return nil
}
