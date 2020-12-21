package syncer

import (
	"log"
	"time"
)

// Schedule a cronjob that scan the store & try to perform logged action on connected database
func (s *Store) Schedule() (quit chan struct{}) {
	ticker := time.NewTicker(time.Duration(s.config.Interval) * time.Second)
	quit = make(chan struct{})

	syncer := NewSyncer(*s.config.TargetDbConfig)

	go func() {
		for {
			select {
			case <-ticker.C:
				s.SyncAllModels(syncer, false)
			case <-quit:
				ticker.Stop()
				syncer.Close()
				return
			}
		}
	}()
	return
}

// SyncAllModels scan all active records in store & perform syncing actions,
// if `isTest` is false, then records will be deleted after successful sync
func (s *Store) SyncAllModels(syncer *Syncer, isTest bool) {

	for table, model := range s.config.Models {
		var count int64
		var err error
		// TODO: make GetAll async
		err = s.GetAll(table, model, func(rec *Record) bool {
			if rec.Action == InsertAction {
				_, err = syncer.Insert(table, rec.New)
				if err != nil {
					// TODO: log into file, logging to console is too much
					// log.Printf("Store service - Insert error: %v\n", err)
					return false
				}
			}
			if rec.Action == UpdateAction {
				// TODO: currently support UpdateOnPK for now, meaning user MUST define a PK in the datamodel
				_, err = syncer.UpdateOnPK(table, rec.Old, rec.New)
				if err != nil {
					// log.Printf("Store service - Insert error: %v\n", err)
					return false
				}
			}
			if rec.Action == DeleteAction {
				// TODO: currently support DeleteOnPK for now, meaning user MUST define a PK in the datamodel
				_, err = syncer.DeleteOnPK(table, rec.Old)
				if err != nil {
					// log.Printf("Store service - Delete error: %v\n", err)
					return false
				}
			}
			if err == nil {
				count++
				return true // return "true" continues the loop
			}
			return false
		})
		// log.Printf("%v - Affect rows: %v\n", table, count)
		// delete from store once success
		if err == nil && count > 0 && !isTest {
			if err := s.LRem(table, int(count)); err != nil {
				log.Panicf("Error in removing synced records: %v", err)
			}
		}
	}
}
