package kalodb

import (
	"encoding/binary"
	"io"
)

type Entry struct {
	key []byte
	val []byte
}

func (ent *Entry) Encode() []byte {
	data := make([]byte, 4 + 4 + len(ent.key) + len(ent.val))

	binary.LittleEndian.PutUint32(data[0:4], uint32(len(ent.key)))
	binary.LittleEndian.PutUint32(data[4:8], uint32(len(ent.val)))

	copy(data[8:], ent.key)
	copy(data[8 + len(ent.key):], ent.val)
	return data
}

func (ent *Entry) Decode(r io.Reader) error {
	header := make([]byte, 8)

	_, err := io.ReadFull(r, header)
	if err != nil {
		return err
	}

	sizeKey := binary.LittleEndian.Uint32(header[0:4])
	sizeVal := binary.LittleEndian.Uint32(header[4:8])

	key := make([]byte, sizeKey)
	val := make([]byte, sizeVal)

	if _, err := io.ReadFull(r, key); err != nil {
		return err
	}

	if _, err := io.ReadFull(r, val); err != nil {
		return err
	}

	ent.key = key
	ent.val = val

	return nil
}
