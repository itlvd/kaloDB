package kalodb

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newTestDB(t *testing.T, filename string) *DB {
	t.Helper()
	os.Remove(filename)
	db := &DB{}
	db.KV.log.FileName = filename
	assert.Nil(t, db.Open())
	t.Cleanup(func() {
		db.Close()
		os.Remove(filename)
	})
	return db
}

func makeRow(ts int64, src, dst string) Row {
	return Row{
		Cell{Type: TypeI64, I64: ts},
		Cell{Type: TypeStr, Str: []byte(src)},
		Cell{Type: TypeStr, Str: []byte(dst)},
	}
}

func pkRow(src, dst string) Row {
	return Row{
		Cell{},
		Cell{Type: TypeStr, Str: []byte(src)},
		Cell{Type: TypeStr, Str: []byte(dst)},
	}
}

func TestTableSelectMissing(t *testing.T) {
	db := newTestDB(t, ".test_tbl_sel_miss")
	ok, err := db.Select(linkSchema(), makeRow(1, "a", "b"))
	assert.Nil(t, err)
	assert.False(t, ok)
}

func TestTableSelectAfterInsert(t *testing.T) {
	db := newTestDB(t, ".test_tbl_sel_ins")

	row := makeRow(123, "a", "b")
	db.Insert(linkSchema(), row)

	out := pkRow("a", "b")
	ok, err := db.Select(linkSchema(), out)
	assert.Nil(t, err)
	assert.True(t, ok)
	assert.Equal(t, row, out)
}

func TestTableInsertNew(t *testing.T) {
	db := newTestDB(t, ".test_tbl_ins_new")

	updated, err := db.Insert(linkSchema(), makeRow(1, "a", "b"))
	assert.Nil(t, err)
	assert.True(t, updated)
}

func TestTableInsertDuplicate(t *testing.T) {
	db := newTestDB(t, ".test_tbl_ins_dup")

	db.Insert(linkSchema(), makeRow(1, "a", "b"))

	// same PKey, different val → Insert must be a no-op
	updated, err := db.Insert(linkSchema(), makeRow(999, "a", "b"))
	assert.Nil(t, err)
	assert.False(t, updated)

	// original value must be unchanged
	out := pkRow("a", "b")
	db.Select(linkSchema(), out)
	assert.Equal(t, int64(1), out[0].I64)
}

func TestTableUpdateNonExistent(t *testing.T) {
	db := newTestDB(t, ".test_tbl_upd_miss")

	updated, err := db.Update(linkSchema(), makeRow(1, "a", "b"))
	assert.Nil(t, err)
	assert.False(t, updated)
}

func TestTableUpdateSameValue(t *testing.T) {
	db := newTestDB(t, ".test_tbl_upd_same")

	row := makeRow(1, "a", "b")
	db.Insert(linkSchema(), row)

	updated, err := db.Update(linkSchema(), row)
	assert.Nil(t, err)
	assert.False(t, updated)
}

func TestTableUpdateNewValue(t *testing.T) {
	db := newTestDB(t, ".test_tbl_upd_new")

	db.Insert(linkSchema(), makeRow(1, "a", "b"))

	updated, err := db.Update(linkSchema(), makeRow(42, "a", "b"))
	assert.Nil(t, err)
	assert.True(t, updated)

	out := pkRow("a", "b")
	db.Select(linkSchema(), out)
	assert.Equal(t, int64(42), out[0].I64)
}

func TestTableUpsertNew(t *testing.T) {
	db := newTestDB(t, ".test_tbl_ups_new")

	updated, err := db.Upsert(linkSchema(), makeRow(1, "a", "b"))
	assert.Nil(t, err)
	assert.True(t, updated)

	out := pkRow("a", "b")
	ok, err := db.Select(linkSchema(), out)
	assert.Nil(t, err)
	assert.True(t, ok)
	assert.Equal(t, int64(1), out[0].I64)
}

func TestTableUpsertOverwrite(t *testing.T) {
	db := newTestDB(t, ".test_tbl_ups_over")

	db.Insert(linkSchema(), makeRow(1, "a", "b"))

	updated, err := db.Upsert(linkSchema(), makeRow(99, "a", "b"))
	assert.Nil(t, err)
	assert.True(t, updated)

	out := pkRow("a", "b")
	db.Select(linkSchema(), out)
	assert.Equal(t, int64(99), out[0].I64)
}

func TestTableUpsertSameValue(t *testing.T) {
	db := newTestDB(t, ".test_tbl_ups_same")

	row := makeRow(1, "a", "b")
	db.Insert(linkSchema(), row)

	updated, err := db.Upsert(linkSchema(), row)
	assert.Nil(t, err)
	assert.False(t, updated)
}

func TestTableDeleteExisting(t *testing.T) {
	db := newTestDB(t, ".test_tbl_del_ok")

	row := makeRow(1, "a", "b")
	db.Insert(linkSchema(), row)

	deleted, err := db.Delete(linkSchema(), row)
	assert.Nil(t, err)
	assert.True(t, deleted)

	ok, _ := db.Select(linkSchema(), pkRow("a", "b"))
	assert.False(t, ok)
}

func TestTableDeleteNonExistent(t *testing.T) {
	db := newTestDB(t, ".test_tbl_del_miss")

	deleted, err := db.Delete(linkSchema(), makeRow(1, "a", "b"))
	assert.Nil(t, err)
	assert.False(t, deleted)
}

func TestTableMultipleRows(t *testing.T) {
	db := newTestDB(t, ".test_tbl_multi")

	rows := []Row{
		makeRow(1, "a", "b"),
		makeRow(2, "a", "c"),
		makeRow(3, "d", "e"),
	}
	for _, r := range rows {
		updated, err := db.Insert(linkSchema(), r)
		assert.Nil(t, err)
		assert.True(t, updated)
	}

	for _, r := range rows {
		src := string(r[1].Str)
		dst := string(r[2].Str)
		out := pkRow(src, dst)
		ok, err := db.Select(linkSchema(), out)
		assert.Nil(t, err)
		assert.True(t, ok)
		assert.Equal(t, r[0].I64, out[0].I64)
	}
}

func TestTablePersistence(t *testing.T) {
	fname := ".test_tbl_persist"
	os.Remove(fname)
	defer os.Remove(fname)

	db := &DB{}
	db.KV.log.FileName = fname
	assert.Nil(t, db.Open())

	db.Insert(linkSchema(), makeRow(10, "p", "q"))
	db.Insert(linkSchema(), makeRow(20, "r", "s"))
	db.Delete(linkSchema(), makeRow(20, "r", "s"))
	db.Close()

	assert.Nil(t, db.Open())
	defer db.Close()

	out := pkRow("p", "q")
	ok, err := db.Select(linkSchema(), out)
	assert.Nil(t, err)
	assert.True(t, ok)
	assert.Equal(t, int64(10), out[0].I64)

	ok, err = db.Select(linkSchema(), pkRow("r", "s"))
	assert.Nil(t, err)
	assert.False(t, ok)
}

func TestTableI64PrimaryKey(t *testing.T) {
	db := newTestDB(t, ".test_tbl_i64pk")

	schema := &Schema{
		Table: "event",
		Cols: []Column{
			{Name: "id", Type: TypeI64},
			{Name: "name", Type: TypeStr},
		},
		PKey: []int{0},
	}

	row := Row{
		Cell{Type: TypeI64, I64: 7},
		Cell{Type: TypeStr, Str: []byte("click")},
	}

	updated, err := db.Insert(schema, row)
	assert.Nil(t, err)
	assert.True(t, updated)

	out := Row{Cell{Type: TypeI64, I64: 7}, Cell{}}
	ok, err := db.Select(schema, out)
	assert.Nil(t, err)
	assert.True(t, ok)
	assert.Equal(t, []byte("click"), out[1].Str)
}

func TestTableInsertWriteError(t *testing.T) {
	db := newTestDB(t, ".test_tbl_ins_err")
	forceCloseLog(&db.KV)

	_, err := db.Insert(linkSchema(), makeRow(1, "a", "b"))
	assert.NotNil(t, err)
}

func TestTableUpsertWriteError(t *testing.T) {
	db := newTestDB(t, ".test_tbl_ups_err")
	forceCloseLog(&db.KV)

	_, err := db.Upsert(linkSchema(), makeRow(1, "a", "b"))
	assert.NotNil(t, err)
}

func TestTableUpdateWriteError(t *testing.T) {
	db := newTestDB(t, ".test_tbl_upd_err")

	// seed key directly in mem so the update path runs
	row := makeRow(1, "a", "b")
	key := row.EncodeKey(linkSchema())
	db.KV.mem[string(key)] = row.EncodeVal(linkSchema())

	forceCloseLog(&db.KV)

	// different value so the "same value" guard is bypassed
	_, err := db.Update(linkSchema(), makeRow(99, "a", "b"))
	assert.NotNil(t, err)
}

func TestTableDeleteWriteError(t *testing.T) {
	db := newTestDB(t, ".test_tbl_del_err")

	row := makeRow(1, "a", "b")
	key := row.EncodeKey(linkSchema())
	db.KV.mem[string(key)] = row.EncodeVal(linkSchema())

	forceCloseLog(&db.KV)

	_, err := db.Delete(linkSchema(), row)
	assert.NotNil(t, err)
}

func TestTableSelectCorruptedVal(t *testing.T) {
	db := newTestDB(t, ".test_tbl_corrupt")

	row := makeRow(1, "a", "b")
	db.Insert(linkSchema(), row)

	// corrupt the stored value directly in mem
	key := row.EncodeKey(linkSchema())
	db.KV.mem[string(key)] = []byte{0xde, 0xad} // bad / truncated

	out := pkRow("a", "b")
	ok, err := db.Select(linkSchema(), out)
	assert.False(t, ok)
	assert.NotNil(t, err)
}

func TestTableByPKey(t *testing.T) {
	db := newTestDB(t, ".test_tbl_full")

	row := makeRow(123, "a", "b")

	ok, err := db.Select(linkSchema(), row)
	assert.Nil(t, err)
	assert.False(t, ok)

	updated, err := db.Insert(linkSchema(), row)
	assert.Nil(t, err)
	assert.True(t, updated)

	out := pkRow("a", "b")
	ok, err = db.Select(linkSchema(), out)
	assert.Nil(t, err)
	assert.True(t, ok)
	assert.Equal(t, row, out)

	row[0].I64 = 456
	updated, err = db.Update(linkSchema(), row)
	assert.Nil(t, err)
	assert.True(t, updated)

	ok, err = db.Select(linkSchema(), out)
	assert.Nil(t, err)
	assert.True(t, ok)
	assert.Equal(t, row, out)

	deleted, err := db.Delete(linkSchema(), row)
	assert.Nil(t, err)
	assert.True(t, deleted)

	ok, err = db.Select(linkSchema(), row)
	assert.Nil(t, err)
	assert.False(t, ok)
}
