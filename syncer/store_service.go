package syncer

import (
	"log"
	"time"
)

// Schedule a cronjob that scan the store & try to perform logged action on connected database
func (s *Store) Schedule() (quit chan struct{}) {
	ticker := time.NewTicker(time.Duration(s.config.interval) * time.Second)
	quit = make(chan struct{})

	syncer := NewSyncer(*s.config.syncConfig)

	go func() {
		for {
			select {
			case <-ticker.C:
				syncAllModels(s, syncer)
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
	return
}

func syncAllModels(s *Store, syncer *Syncer) {
	var count int64
	for table, model := range s.config.models {
		// TODO: make GetAll async
		err := s.GetAll(table, model, func(rec *Record) bool {
			if rec.Action == InsertAction {
				res, err := syncer.Insert(table, rec.New)
				if err != nil {
					log.Printf("Store service - Insert error: %v\n", err)
					// TODO: log or something
					return false
				}
				ar, err := res.RowsAffected()
				if err == nil {
					count += ar
				}
				return true // return "true" continues the loop
			}
			return false
		})
		// delete from store once success
		if err != nil && count > 0 {
			s.db.ListIdx[bucket].LRem(table, int(count))
		}
	}
	log.Printf("\nAffect rows: %v\n", count)
}
