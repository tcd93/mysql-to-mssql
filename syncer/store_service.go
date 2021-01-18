package syncer

import (
	"fmt"
	"time"

	"github.com/siddontang/go-log/log"
)

// Schedule a cronjob that scan the store & try to perform logged action on connected database
func (s *Syncer) Schedule() {
	ticker := time.NewTicker(time.Duration(s.interval) * time.Second)
	s.syncQuitSignal = make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				s.SyncAllModels(false)
			case <-s.syncQuitSignal:
				ticker.Stop()
				s.Close()
				return
			}
		}
	}()
}

// Stop schedule
func (s *Syncer) Stop() error {
	if s.syncQuitSignal == nil {
		return fmt.Errorf("Syncer is closed, or has not been started")
	}
	s.syncQuitSignal <- struct{}{}
	s.syncQuitSignal = nil
	return nil
}

// SyncAllModels scan all active records in store & perform syncing actions,
// if `isTest` is false, then records will be deleted after successful sync
func (s Syncer) SyncAllModels(isTest bool) {
	for table, model := range s.store.Models {
		var size int
		var count int64
		var err error

		size, err = s.store.Size(table)
		if size == 0 {
			continue
		}

		// TODO: make GetAll async
		err = s.store.GetAll(table, model, func(rec *Record) error {
			if rec.Action == InsertAction {
				_, err = s.Insert(table, rec.New)
				if err != nil {
					return fmt.Errorf("Insert error: %v", err.Error())
				}
			}
			if rec.Action == UpdateAction {
				// TODO: currently support UpdateOnPK for now, meaning user MUST define a PK in the datamodel
				_, err = s.UpdateOnPK(table, rec.Old, rec.New)
				if err != nil {
					return fmt.Errorf("Update error: %v", err.Error())
				}
			}
			if rec.Action == DeleteAction {
				// TODO: currently support DeleteOnPK for now, meaning user MUST define a PK in the datamodel
				_, err = s.DeleteOnPK(table, rec.Old)
				if err != nil {
					return fmt.Errorf("Delete error: %v", err.Error())
				}
			}
			if err == nil {
				count++
				return nil // return nil continues the loop
			}
			return err
		})
		// delete from store once success
		if err == nil && count > 0 && !isTest {
			if err := s.store.LRem(table, int(count)); err != nil {
				log.Panicf("Error in removing synced records: %v", err)
			}
		} else if err != nil {
			log.Errorf("error: %v - stopping syncer...", err.Error())
			s.syncQuitSignal <- struct{}{}
		}
	}
}
