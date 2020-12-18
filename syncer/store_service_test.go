package syncer

import (
	"testing"
	"time"
)

func TestStoreService(t *testing.T) {
	cfg := DefaultStoreConfig
	cfg.dbConfig.Dir = "./test"
	cfg.interval = 1
	cfg.models = map[string]interface{}{
		"SyncerTest": &syncerTest{},
		"StoreTest":  &storeTest{},
	}

	store := NewStore(cfg)

	// setUpStore(store)
	defer tearDownStore(store)

	quit := store.Schedule()
	time.AfterFunc(2*time.Second, func() {
		quit <- struct{}{}
	})
	<-quit
}
