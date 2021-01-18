package syncer

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

// clean all test-generated data files under ./test
func clearDir(fileDir string) {
	files, _ := ioutil.ReadDir(fileDir)
	for _, f := range files {
		name := f.Name()
		if name != "" {
			err := os.RemoveAll(fileDir + "/" + name)
			if err != nil {
				panic(err)
			}
		}
	}
}

func setUpData(store *Store, t *testing.T) {
	old := &storeTest{1, []byte(`Lorem ipsum dolor sit amet`)}

	// add 50 rows
	for i := 0; i < 50; i++ {
		old.ID = i
		err := store.LogInsert("StoreTest", old)
		if err != nil {
			t.Errorf("Insert storeTest failed: %v\n", err.Error())
			t.FailNow()
		}
	}

	// update row 20 - 24
	new := &storeTest{1, []byte(`updated row: `)}
	for i := 20; i < 25; i++ {
		old.ID = i
		new.ID = i
		new.Name = append(new.Name, []byte(fmt.Sprintf("%d", i))...)
		err := store.LogUpdate("StoreTest", old, new)
		if err != nil {
			t.Errorf("Update storeTest failed: %v\n", err.Error())
			t.FailNow()
		}
	}

	// delete row 5 - 7
	del := &storeTest{1, []byte("delete row")}
	for i := 5; i < 8; i++ {
		del.ID = i
		err := store.LogDelete("StoreTest", del)
		if err != nil {
			t.Errorf("Delete storeTest failed: %v\n", err.Error())
			t.FailNow()
		}
	}
}

func TestStoreService(t *testing.T) {
	// clearDir(dataFileTestDir)

	store := DefaultStore
	// store.LocalDb.SetDir(dataFileTestDir)

	store.Models = map[string]interface{}{
		"StoreTest": &storeTest{},
	}

	syncer := NewSyncer(TargetDbConfig{
		Server:   "127.0.0.1",
		Log:      63,
		Database: "mysql2mssql",
	}, 1, store)
	setUpData(store, t)
	defer tearDownStore(store)
	defer syncer.Close()

	syncer.SyncAllModels(true)
}
