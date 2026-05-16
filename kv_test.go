package kalodb

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

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

	// reopen
	kv.Close()
	err = kv.Open()
	assert.Nil(t, err)

	_, ok, err = kv.Get([]byte("k1"))
	assert.True(t, !ok && err == nil)
	val, ok, err = kv.Get([]byte("k2"))
	assert.True(t, string(val) == "v2" && ok && err == nil)
}

func TestEntryEncode(t *testing.T) {
	ent := Entry{
		key:     []byte("name"),
		val:     []byte("dong"),
		deleted: false,
	}

	got := ent.Encode()

	want := []byte{
		0x04, 0x00, 0x00, 0x00, // key length = 4
		0x04, 0x00, 0x00, 0x00, // value length = 4
		0x00, // deleted = false
		'n', 'a', 'm', 'e',
		'd', 'o', 'n', 'g',
	}

	if !bytes.Equal(got, want) {
		t.Fatalf("Encode() mismatch\nwant: %v\ngot:  %v", want, got)
	}
}

func TestEntryEncodeDeleted(t *testing.T) {
	ent := Entry{
		key:     []byte("name"),
		val:     []byte("dong"),
		deleted: true,
	}

	got := ent.Encode()

	want := []byte{
		0x04, 0x00, 0x00, 0x00, // key length = 4
		0x04, 0x00, 0x00, 0x00, // value length = 4
		0x01, // deleted = true
		'n', 'a', 'm', 'e',
		'd', 'o', 'n', 'g',
	}

	if !bytes.Equal(got, want) {
		t.Fatalf("Encode() mismatch\nwant: %v\ngot:  %v", want, got)
	}
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
		{
			key:     []byte("name"),
			val:     []byte("dong"),
			deleted: false,
		},
		{
			key:     []byte(""),
			val:     []byte("empty key"),
			deleted: false,
		},
		{
			key:     []byte("empty value"),
			val:     []byte(""),
			deleted: false,
		},
		{
			key:     []byte("hello"),
			val:     []byte("world"),
			deleted: false,
		},
		{
			key:     []byte("tombstone"),
			val:     []byte(""),
			deleted: true,
		},
		{
			key:     []byte("deleted-with-value"),
			val:     []byte("old"),
			deleted: true,
		},
		{
			key:     []byte(""),
			val:     []byte(""),
			deleted: true,
		},
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
	}
}

func TestEntryDecodeMultipleEntries(t *testing.T) {
	ent1 := Entry{
		key:     []byte("name"),
		val:     []byte("dong"),
		deleted: false,
	}

	ent2 := Entry{
		key:     []byte("lang"),
		val:     []byte("go"),
		deleted: true,
	}

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

	if !bytes.Equal(decoded1.key, ent1.key) || !bytes.Equal(decoded1.val, ent1.val) {
		t.Fatalf("first entry mismatch")
	}

	if !bytes.Equal(decoded2.key, ent2.key) || !bytes.Equal(decoded2.val, ent2.val) {
		t.Fatalf("second entry mismatch")
	}
}

func TestEntryDecodeShortHeader(t *testing.T) {
	data := []byte{
		0x04, 0x00, 0x00, 0x00,
		// missing value length
	}

	var ent Entry
	err := ent.Decode(bytes.NewReader(data))
	if err == nil {
		t.Fatal("expected error for short header, got nil")
	}
}

func TestEntryDecodeMissingDeletedFlag(t *testing.T) {
	data := []byte{
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
	original := Entry{
		key:     []byte("name"),
		val:     []byte("dong"),
		deleted: false,
	}

	data := original.Encode()

	data = data[:10]

	var ent Entry
	err := ent.Decode(bytes.NewReader(data))
	if err == nil {
		t.Fatal("expected error for short key, got nil")
	}
}

func TestEntryDecodeShortValue(t *testing.T) {
	original := Entry{
		key:     []byte("name"),
		val:     []byte("dong"),
		deleted: false,
	}

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
	original := Entry{
		key:     []byte("name"),
		val:     []byte("dong"),
		deleted: true,
	}

	data := original.Encode()

	var decoded Entry
	err := decoded.Decode(slowReader{
		r: bytes.NewReader(data),
	})
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
