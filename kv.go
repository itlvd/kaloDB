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
	prev, exist := kv.mem[string(key)]
	switch mode {
	case ModeUpsert:
		updated = !exist || string(prev) != string(val)
	case ModeUpdate:
		updated = exist && string(prev) != string(val)
	case ModeInsert:
		updated = !exist
	default:
		return false, errors.New("mode does not supported")
	}

	if updated {
		if err = kv.log.Write(&Entry{
			key:     key,
			val:     val,
			deleted: false,
		}); err != nil {
			return false, err
		}

		kv.mem[string(key)] = val
	}

	return updated, nil
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
