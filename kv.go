package kalodb

import (
	"errors"
)

type KV struct {
	log Log
	mem map[string][]byte
}

func (kv *KV) Open() error {
	if err := kv.log.Open(); err != nil {
		return err
	}

	kv.mem = map[string][]byte{}
	for {
		newEntry := &Entry{}
		eof, err := kv.log.Read(newEntry)
		if err != nil {
			return err
		}

		if eof {
			break
		}

		if newEntry.deleted {
			delete(kv.mem, string(newEntry.key))
		} else {
			kv.mem[string(newEntry.key)] = newEntry.val
		}
	}
	return nil
}

func (kv *KV) Close() error { return kv.log.Close() }

func (kv *KV) Get(key []byte) (val []byte, ok bool, err error) {
	val, ok = kv.mem[string(key)]
	return
}

type UpdateMode int

const (
	ModeUpsert UpdateMode = 0 // insert or update
	ModeInsert UpdateMode = 1 // insert new
	ModeUpdate UpdateMode = 2 // update existing
)

func (kv *KV) SetEx(key []byte, val []byte, mode UpdateMode) (updated bool, err error) {
	switch mode {
	case ModeUpsert:
		return kv.setUpsert(key, val)
	case ModeUpdate:
		return kv.setUpdate(key, val)
	case ModeInsert:
		return kv.setInsert(key, val)
	default:
		return false, errors.New("mode does not supported")
	}
}

func (kv *KV) setUpsert(key []byte, val []byte) (updated bool, err error) {
	prev, exist := kv.mem[string(key)]
	if exist && string(prev) == string(val) {
		return false, nil
	}

	if err = kv.log.Write(&Entry{
		key:     key,
		val:     val,
		deleted: false,
	}); err != nil {
		return false, err
	}

	kv.mem[string(key)] = val
	updated = string(prev) != string(val) || !exist
	return updated, err
}

func (kv *KV) setInsert(key []byte, val []byte) (updated bool, err error) {
	_, exist := kv.mem[string(key)]
	if exist {
		return false, nil
	}

	if err = kv.log.Write(&Entry{
		key:     key,
		val:     val,
		deleted: false,
	}); err != nil {
		return false, err
	}

	kv.mem[string(key)] = val
	return true, nil
}

func (kv *KV) setUpdate(key []byte, val []byte) (updated bool, err error) {
	prev, exist := kv.mem[string(key)]
	if !exist || string(prev) == string(val) {
		return false, nil
	}

	if err = kv.log.Write(&Entry{
		key:     key,
		val:     val,
		deleted: false,
	}); err != nil {
		return false, err
	}

	kv.mem[string(key)] = val
	return true, nil
}

func (kv *KV) Set(key []byte, val []byte) (updated bool, err error) {
	return kv.SetEx(key, val, ModeUpsert)
}

func (kv *KV) Del(key []byte) (deleted bool, err error) {
	_, exist := kv.mem[string(key)]
	if !exist {
		return false, nil
	}

	if err = kv.log.Write(&Entry{
		key:     key,
		val:     nil,
		deleted: true,
	}); err != nil {
		return false, err
	}

	delete(kv.mem, string(key))
	return exist, err
}
