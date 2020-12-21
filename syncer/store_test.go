package syncer

import (
	"reflect"
	"testing"
	"time"

	dcm "github.com/shopspring/decimal"
)

func setUpStore(store *Store) {
	if err := store.truncate("SyncerTest"); err != nil {
		panic(err)
	}
	if err := store.truncate("StoreTest"); err != nil {
		panic(err)
	}
}

func tearDownStore(store *Store) {
	store.Close()
}

func TestStoreInsert(t *testing.T) {
	store := NewStore(DefaultStoreConfig)
	setUpStore(store)
	defer tearDownStore(store)

	// 1st insert

	dec, _ := dcm.NewFromString("11112345111899999999874444444313.11198")
	dtime, _ := time.Parse("2006-01-02 15:04:05", "2020-01-01 10:10:10")
	date, _ := time.Parse("2006-01-02", "1999-12-01")
	unsigned := uint(18446744073709551615)
	time := "23:59:59"
	varbin := []byte("Varbinary Stuff")
	insertedModel := &syncerTest{
		ID:     1,
		Name:   "中文 English Tiếng Việt",
		Bo:     true,
		Bi:     9223372036854775807,
		BiU:    &unsigned,
		De:     dec,
		Fl:     12.3457,
		Do:     56.789123456,
		Bit:    9223372036854775807,
		DTime:  dtime,
		Date:   &date,
		Time:   &time,
		Blob:   []byte("First Row"),
		Binary: &varbin,
	}

	err := store.LogInsert("SyncerTest", insertedModel)
	if err != nil {
		t.Errorf("Insert syncerTest failed: %v\n", err.Error())
		t.FailNow()
	}

	var b []interface{}
	var retrievedModel = &syncerTest{}
	err = store.GetAll("SyncerTest", retrievedModel, func(rec *Record) bool {
		b = append(b, rec.New)
		return true
	})
	if err != nil {
		t.Errorf("GetAll failed: %v\n", err.Error())
		t.FailNow()
	}
	if len(b) != 1 {
		t.Errorf("GetAll - len != 1, actual: %v", len(b))
		t.FailNow()
	}
	if reflect.DeepEqual(insertedModel, b[0].(*syncerTest)) == false {
		t.Errorf("Difference in inserted model & retrieved model: \n Expected: %v\n   Actual: %v", insertedModel, b[0].(*syncerTest))
	}

	// 2nd insert
	insertedModel = &syncerTest{
		ID:     2,
		Name:   "Dòng 2",
		Bo:     true,
		Bi:     9223372036854775807,
		BiU:    nil,
		De:     dec,
		Fl:     12.345666,
		Do:     56.789123456,
		Bit:    9223372036854775807,
		DTime:  dtime,
		Date:   nil,
		Time:   &time,
		Blob:   []byte("Sécond Rơw"),
		Binary: nil,
	}
	err = store.LogInsert("SyncerTest", insertedModel)
	if err != nil {
		t.Errorf("Insert syncerTest failed: %v\n", err.Error())
		t.FailNow()
	}

	b = nil
	err = store.GetAll("SyncerTest", retrievedModel, func(rec *Record) bool {
		b = append(b, rec.New)
		return true
	})
	if err != nil {
		t.Errorf("GetAll failed: %v\n", err.Error())
		t.FailNow()
	}
	if len(b) != 2 {
		t.Errorf("GetAll - len != 2, actual: %v", len(b))
		t.FailNow()
	}
	if reflect.DeepEqual(insertedModel, b[1].(*syncerTest)) == false {
		t.Errorf("Difference in inserted model & retrieved model: \n Expected: %v\n   Actual: %v", insertedModel, b[1].(*syncerTest))
	}

	// insert to another table

	secondInsertModel := &storeTest{1, []byte("Name")}
	var retrievedModel2 = &storeTest{}
	err = store.LogInsert("StoreTest", secondInsertModel)
	if err != nil {
		t.Errorf("Insert storeTest failed: %v\n", err.Error())
		t.FailNow()
	}

	b = nil
	err = store.GetAll("StoreTest", retrievedModel2, func(rec *Record) bool {
		b = append(b, rec.New)
		return true
	})
	if err != nil {
		t.Errorf("GetAll failed: %v\n", err.Error())
		t.FailNow()
	}
	if len(b) != 1 {
		t.Errorf("GetAll - len != 1, actual: %v", len(b))
		t.FailNow()
	}

	if reflect.DeepEqual(secondInsertModel, b[0]) == false {
		t.Errorf("Difference in inserted model & retrieved model: \n Expected: %v\n Actual: %v", secondInsertModel, b[0])
	}
}

func TestStoreUpdate(t *testing.T) {
	store := NewStore(DefaultStoreConfig)
	setUpStore(store)
	defer tearDownStore(store)

	// 1st update
	unsigned := uint(18446744073709551615)
	old := &syncerTest{
		ID:   1,
		Name: "~~Old~~",
		Bo:   true,
		Bi:   9223372036854775807,
		BiU:  &unsigned,
		Blob: []byte("Old Row"),
	}
	new := &syncerTest{
		ID:   1,
		Name: "^^New^^",
		Bo:   false,
		Bi:   9223372036854775807,
		BiU:  nil,
		Blob: []byte("New Row"),
	}

	err := store.LogUpdate("SyncerTest", old, new)
	if err != nil {
		t.Errorf("LogUpdate syncerTest failed: %v\n", err.Error())
		t.FailNow()
	}

	var retrievedModel = &syncerTest{}
	var count uint8
	err = store.GetAll("SyncerTest", retrievedModel, func(rec *Record) bool {
		if reflect.DeepEqual(rec.New, new) == false {
			t.Errorf("Difference in [new] updated model & retrieved model: \n Expected: %v\n   Actual: %v", new, rec.New)
		}
		if reflect.DeepEqual(rec.Old, old) == false {
			t.Errorf("Difference in [old] updated model & retrieved model: \n Expected: %v\n   Actual: %v", old, rec.Old)
		}
		count++
		return true
	})
	if err != nil {
		t.Errorf("GetAll failed: %v\n", err.Error())
		t.FailNow()
	}
	if count != 1 {
		t.Errorf("GetAll - len != 1, actual: %v", count)
		t.FailNow()
	}
}

func TestStoreDelete(t *testing.T) {
	store := NewStore(DefaultStoreConfig)
	setUpStore(store)
	defer tearDownStore(store)

	r := &storeTest{1, []byte("Name")}

	err := store.LogDelete("StoreTest", r)
	if err != nil {
		t.Errorf("LogDelete storeTest failed: %v\n", err.Error())
		t.FailNow()
	}

	var retrievedModel = &storeTest{}
	var count uint8
	err = store.GetAll("StoreTest", retrievedModel, func(rec *Record) bool {
		if reflect.DeepEqual(rec.Old, r) == false {
			t.Errorf("Difference in updated model & retrieved model: \n Expected: %v\n   Actual: %v", r, rec.Old)
		}
		count++
		return true
	})
	if err != nil {
		t.Errorf("GetAll failed: %v\n", err.Error())
		t.FailNow()
	}
	if count != 1 {
		t.Errorf("GetAll - len != 1, actual: %v", count)
		t.FailNow()
	}
}

type storeTest struct {
	ID   int    `gorm:"column:id;primaryKey"`
	Name []byte `gorm:"column:name"`
}
