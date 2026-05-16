package kalodb

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
		newEntry := Entry{}
		eof, err := kv.log.Read(&newEntry)
		if err != nil {
			return err
		}

		if newEntry.deleted {
			delete(kv.mem, string(newEntry.key))
		} else {
			kv.mem[string(newEntry.key)] = newEntry.val
		}

		if eof {
			break
		}
	}
	return nil
}

func (kv *KV) Close() error { return nil }

func (kv *KV) Get(key []byte) (val []byte, ok bool, err error) {
	val, ok = kv.mem[string(key)]
	return
}

func (kv *KV) Set(key []byte, val []byte) (updated bool, err error) {
	prev, exist := kv.mem[string(key)]
	kv.mem[string(key)] = val

	err = kv.log.Write(&Entry{
		key:     key,
		val:     val,
		deleted: false,
	})

	updated = string(prev) != string(val) || !exist
	return updated, err
}

func (kv *KV) Del(key []byte) (deleted bool, err error) {
	_, deleted = kv.mem[string(key)]
	delete(kv.mem, string(key))

	err = kv.log.Write(&Entry{
		key:     key,
		val:     nil,
		deleted: true,
	})

	return
}
