package db

import (
	"github.com/xujiajun/nutsdb"
)

type localdb struct {
	*nutsdb.DB
	dbConfig Options
}

// UseNutsDB uses nutsdb v0.5 underneath dbInterface
func UseNutsDB(dbConfig Options) Interface {
	db, err := nutsdb.Open(nutsdb.Options{
		Dir:         dbConfig.Dir,
		SegmentSize: dbConfig.SegmentSize,
		SyncEnable:  true,
	})
	if err != nil {
		panic(err)
	}
	return &localdb{db, dbConfig}
}

func (ldb localdb) Release() error {
	return ldb.Close()
}

func (ldb localdb) Dir() string {
	return ldb.dbConfig.Dir
}

func (ldb localdb) SetDir(dir string) {
	ldb.dbConfig.Dir = dir
}

func (ldb localdb) GetAll(bucket string) (entries []*Entry, err error) {
	err = ldb.View(func(tx *nutsdb.Tx) error {
		txEntries, err := tx.GetAll(bucket)
		// ignore empty bucket
		if err != nil && err != nutsdb.ErrBucketEmpty {
			return err
		}
		for _, e := range txEntries {
			entries = append(entries, &Entry{
				Key:   string(e.Key),
				Value: e.Value,
			})
		}
		return nil
	})
	return
}

func (ldb localdb) GetAllKey(bucket string, key string) (list [][]byte, err error) {
	err = ldb.View(func(tx *nutsdb.Tx) error {
		list, err = tx.LRange(bucket, []byte(key), 0, -1)
		// ignore empty bucket
		if err != nil && err != nutsdb.ErrBucketEmpty {
			return err
		}
		return nil
	})
	return
}

func (ldb localdb) Put(bucket string, key string, value []byte, ttl uint32) error {
	return ldb.Update(func(tx *nutsdb.Tx) error {
		return tx.Put(bucket, []byte(key), value, ttl)
	})
}

func (ldb localdb) Push(bucket string, key string, value []byte) error {
	return ldb.Update(func(tx *nutsdb.Tx) error {
		return tx.RPush(bucket, []byte(key), value)
	})
}

func (ldb localdb) Rem(bucket string, key string, count int) (err error) {
	return ldb.Update(func(tx *nutsdb.Tx) error {
		return tx.LRem(bucket, []byte(key), count)
	})
}

func (ldb localdb) Size(bucket string, key string) (size int, err error) {
	ldb.View(func(tx *nutsdb.Tx) error {
		size, err = tx.LSize(bucket, []byte(key))
		if err != nil {
			return err
		}
		return nil
	})
	return
}

func (ldb localdb) Type() string {
	return "nutsdb"
}

func (ldb localdb) Truncate(bucket string, key string) error {
	return ldb.Update(func(tx *nutsdb.Tx) error {
		if err := tx.LRem(bucket, []byte(key), 0); err != nil {
			// ignore "the list not found" error
			if err.Error() == "the list not found" || err.Error() == "err bucket" {
				return nil
			}
			return err
		}
		return nil
	})
}
