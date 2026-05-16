package kalodb

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEntryEncode(t *testing.T) {
	ent := Entry{key: []byte("k1"), val: []byte("xxx")}
	data := []byte{0xe9, 0xec, 0x4d, 0x9e, 2, 0, 0, 0, 3, 0, 0, 0, 0, 'k', '1', 'x', 'x', 'x'}

	assert.Equal(t, data, ent.Encode())

	decoded := Entry{}
	err := decoded.Decode(bytes.NewBuffer(data))
	assert.Nil(t, err)
	assert.Equal(t, ent, decoded)

	ent = Entry{key: []byte("k1"), deleted: true}
	data = []byte{0x4c, 0xd0, 0xfe, 0xe5, 2, 0, 0, 0, 0, 0, 0, 0, 1, 'k', '1'}

	assert.Equal(t, data, ent.Encode())

	decoded = Entry{}
	err = decoded.Decode(bytes.NewBuffer(data))
	assert.Nil(t, err)
	assert.Equal(t, ent, decoded)
}

func TestEntryDecode(t *testing.T) {
	original := Entry{
		key:     []byte("name"),
		val:     []byte("dong"),
		deleted: false,
	}

	data := original.Encode()

	var decoded Entry
	err := decoded.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Decode() returned error: %v", err)
	}

	if !bytes.Equal(decoded.key, original.key) {
		t.Fatalf("decoded key mismatch: want %q, got %q", original.key, decoded.key)
	}

	if !bytes.Equal(decoded.val, original.val) {
		t.Fatalf("decoded val mismatch: want %q, got %q", original.val, decoded.val)
	}
}

func TestEntryEncodeDecodeRoundTrip(t *testing.T) {
	tests := []Entry{
		{key: []byte("name"), val: []byte("dong"), deleted: false},
		{key: []byte(""), val: []byte("empty key"), deleted: false},
		{key: []byte("empty value"), val: []byte(""), deleted: false},
		{key: []byte("hello"), val: []byte("world"), deleted: false},
		{key: []byte("tombstone"), val: nil, deleted: true},
		{key: []byte("k1"), val: nil, deleted: true},
		{key: []byte(""), val: nil, deleted: true},
	}

	for _, original := range tests {
		data := original.Encode()

		var decoded Entry
		err := decoded.Decode(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("Decode() returned error: %v", err)
		}

		if !bytes.Equal(decoded.key, original.key) {
			t.Fatalf("key mismatch: want %q, got %q", original.key, decoded.key)
		}

		if !bytes.Equal(decoded.val, original.val) {
			t.Fatalf("val mismatch: want %q, got %q", original.val, decoded.val)
		}

		if decoded.deleted != original.deleted {
			t.Fatalf("deleted mismatch: want %v, got %v", original.deleted, decoded.deleted)
		}
	}
}

func TestEntryDecodeMultipleEntries(t *testing.T) {
	ent1 := Entry{key: []byte("name"), val: []byte("dong"), deleted: false}
	ent2 := Entry{key: []byte("lang"), val: nil, deleted: true} // tombstone, val = nil

	data := append(ent1.Encode(), ent2.Encode()...)
	r := bytes.NewReader(data)

	var decoded1 Entry
	if err := decoded1.Decode(r); err != nil {
		t.Fatalf("Decode first entry error: %v", err)
	}

	var decoded2 Entry
	if err := decoded2.Decode(r); err != nil {
		t.Fatalf("Decode second entry error: %v", err)
	}

	if !bytes.Equal(decoded1.key, ent1.key) ||
		!bytes.Equal(decoded1.val, ent1.val) ||
		decoded1.deleted != ent1.deleted {
		t.Fatalf("first entry mismatch: want %+v, got %+v", ent1, decoded1)
	}

	if !bytes.Equal(decoded2.key, ent2.key) ||
		!bytes.Equal(decoded2.val, ent2.val) ||
		decoded2.deleted != ent2.deleted {
		t.Fatalf("second entry mismatch: want %+v, got %+v", ent2, decoded2)
	}
}

func TestEntryDecodeShortHeader(t *testing.T) {
	data := []byte{
		0x04, 0x00, 0x00, 0x00,
		// short, missing rest of header
	}

	var ent Entry
	err := ent.Decode(bytes.NewReader(data))
	if err == nil {
		t.Fatal("expected error for short header, got nil")
	}
}

func TestEntryDecodeMissingDeletedFlag(t *testing.T) {
	data := []byte{
		0x00, 0x00, 0x00, 0x00, // checksum
		0x04, 0x00, 0x00, 0x00, // key length = 4
		0x04, 0x00, 0x00, 0x00, // value length = 4
		// missing deleted flag and the key/val bytes
	}

	var ent Entry
	err := ent.Decode(bytes.NewReader(data))
	if err == nil {
		t.Fatal("expected error for missing deleted flag, got nil")
	}
}

func TestEntryDecodeShortKey(t *testing.T) {
	original := Entry{key: []byte("name"), val: []byte("dong"), deleted: false}
	data := original.Encode()
	data = data[:10]

	var ent Entry
	err := ent.Decode(bytes.NewReader(data))
	if err == nil {
		t.Fatal("expected error for short key, got nil")
	}
}

func TestEntryDecodeShortValue(t *testing.T) {
	original := Entry{key: []byte("name"), val: []byte("dong"), deleted: false}
	data := original.Encode()
	data = data[:len(data)-2]

	var ent Entry
	err := ent.Decode(bytes.NewReader(data))
	if err == nil {
		t.Fatal("expected error for short value, got nil")
	}
}

type slowReader struct {
	r io.Reader
}

func (s slowReader) Read(p []byte) (int, error) {
	if len(p) > 1 {
		p = p[:1]
	}
	return s.r.Read(p)
}

func TestEntryDecodeWithSlowReader(t *testing.T) {
	tests := []Entry{
		{key: []byte("name"), val: []byte("dong"), deleted: false},
		{key: []byte("name"), val: nil, deleted: true},
	}

	for _, original := range tests {
		data := original.Encode()

		var decoded Entry
		err := decoded.Decode(slowReader{r: bytes.NewReader(data)})
		if err != nil {
			t.Fatalf("Decode() returned error: %v", err)
		}

		if !bytes.Equal(decoded.key, original.key) {
			t.Fatalf("decoded key mismatch: want %q, got %q", original.key, decoded.key)
		}

		if !bytes.Equal(decoded.val, original.val) {
			t.Fatalf("decoded val mismatch: want %q, got %q", original.val, decoded.val)
		}

		if decoded.deleted != original.deleted {
			t.Fatalf("decoded deleted mismatch: want %v, got %v", original.deleted, decoded.deleted)
		}
	}
}

func TestKVBasic(t *testing.T) {
	kv := KV{}
	kv.log.FileName = ".test_db"
	defer os.Remove(kv.log.FileName)

	os.Remove(kv.log.FileName)
	err := kv.Open()
	assert.Nil(t, err)
	defer kv.Close()

	updated, err := kv.Set([]byte("k1"), []byte("v1"))
	assert.True(t, updated && err == nil)

	val, ok, err := kv.Get([]byte("k1"))
	assert.True(t, string(val) == "v1" && ok && err == nil)

	_, ok, err = kv.Get([]byte("xxx"))
	assert.True(t, !ok && err == nil)

	updated, err = kv.Del([]byte("xxx"))
	assert.True(t, !updated && err == nil)

	updated, err = kv.Del([]byte("k1"))
	assert.True(t, updated && err == nil)

	_, ok, err = kv.Get([]byte("xxx"))
	assert.True(t, !ok && err == nil)

	updated, err = kv.Set([]byte("k2"), []byte("v2"))
	assert.True(t, updated && err == nil)

	kv.Close()
	err = kv.Open()
	assert.Nil(t, err)

	_, ok, err = kv.Get([]byte("k1"))
	assert.True(t, !ok && err == nil)
	val, ok, err = kv.Get([]byte("k2"))
	assert.True(t, string(val) == "v2" && ok && err == nil)
}

func TestKVRecovery(t *testing.T) {
	kv := KV{}
	kv.log.FileName = ".test_db"
	defer os.Remove(kv.log.FileName)

	prepare := func() {
		os.Remove(kv.log.FileName)

		err := kv.Open()
		assert.Nil(t, err)
		defer kv.Close()

		updated, err := kv.Set([]byte("k1"), []byte("v1"))
		assert.True(t, updated && err == nil)
		updated, err = kv.Set([]byte("k2"), []byte("v2"))
		assert.True(t, updated && err == nil)
	}

	prepare()
	fp, _ := os.OpenFile(kv.log.FileName, os.O_RDWR, 0o644)
	st, _ := fp.Stat()
	fp.Truncate(st.Size() - 1)
	fp.Close()
	err := kv.Open()
	assert.Nil(t, err)
	val, ok, err := kv.Get([]byte("k1"))
	assert.True(t, string(val) == "v1" && ok && err == nil)
	_, ok, err = kv.Get([]byte("k2")) // bad
	assert.True(t, !ok && err == nil)
	kv.Close()

	prepare()
	fp, _ = os.OpenFile(kv.log.FileName, os.O_RDWR, 0o644)
	st, _ = fp.Stat()
	fp.WriteAt([]byte{0}, st.Size()-1)
	fp.Close()
	err = kv.Open()
	assert.Nil(t, err)
	val, ok, err = kv.Get([]byte("k1"))
	assert.True(t, string(val) == "v1" && ok && err == nil)
	_, ok, err = kv.Get([]byte("k2")) // bad
	assert.True(t, !ok && err == nil)
	kv.Close()
}
