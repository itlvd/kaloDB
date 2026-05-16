package kalodb

import (
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
)

const (
	checksumSize    = 4
	keyLenFieldSize = 4
	valLenFieldSize = 4
	deletedFlagSize = 1
)

var (
	ErrBadSum = errors.New("bad checksum")
)

type Entry struct {
	key      []byte
	val      []byte
	deleted  bool
}

func (ent *Entry) Encode() []byte {
	headerSize := checksumSize + keyLenFieldSize + valLenFieldSize + deletedFlagSize
	keyOffset := headerSize
	valOffset := keyOffset + len(ent.key)

	data := make([]byte, headerSize+len(ent.key)+len(ent.val))

	binary.LittleEndian.PutUint32(data[checksumSize:checksumSize+keyLenFieldSize], uint32(len(ent.key)))
	binary.LittleEndian.PutUint32(data[checksumSize+keyLenFieldSize:checksumSize+keyLenFieldSize+valLenFieldSize], uint32(len(ent.val)))

	if ent.deleted {
		data[checksumSize+keyLenFieldSize+valLenFieldSize] = 1
	}

	copy(data[keyOffset:valOffset], ent.key)
	copy(data[valOffset:], ent.val)

	binary.LittleEndian.PutUint32(data[0:checksumSize], crc32.ChecksumIEEE(data[checksumSize:]))
	return data
}

func (ent *Entry) Decode(r io.Reader) (err error) {
	headerSize := checksumSize + keyLenFieldSize + valLenFieldSize + deletedFlagSize

	header := make([]byte, headerSize)
	_, err = io.ReadFull(r, header)
	if err != nil {
		return err
	}

	checksum := binary.LittleEndian.Uint32(header[0:checksumSize])
	keySize := binary.LittleEndian.Uint32(header[checksumSize : checksumSize+keyLenFieldSize])
	valSize := binary.LittleEndian.Uint32(header[checksumSize+keyLenFieldSize : checksumSize+keyLenFieldSize+valLenFieldSize])

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

	h := crc32.NewIEEE()
	h.Write(header[checksumSize:])
	h.Write(key)
	h.Write(val)

	if checksum != h.Sum32() {
		return ErrBadSum
	}

	if header[headerSize-deletedFlagSize] == 1 {
		ent.deleted = true
		ent.val = nil
	}

	return nil
}
