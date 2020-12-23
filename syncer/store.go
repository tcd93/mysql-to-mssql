package syncer

import (
	"bytes"
	"fmt"
	"log"
	"reflect"

	"encoding/gob"

	nutsdb "github.com/xujiajun/nutsdb"
)

// Action records the event type such as insert/update/delete
type Action uint8

const (
	// InsertAction - 1 log record in .dat file stores encoded bytes in following format: [action | inserted data]
	InsertAction Action = iota + 1
	// UpdateAction - 1 log record in .dat file stores encoded bytes in following format: [action | new data | old data]
	UpdateAction
	// DeleteAction - 1 log record in .dat file stores encoded bytes in following format: [action | deleted data]
	DeleteAction
)

// Record contains old & new data
type Record struct {
	Action Action
	// only available in update events
	Old interface{}
	New interface{}
}

// global bucket (namespace) for nutsdb
const bucket = "store"

// ModelDefinitions maps actual table name to the data structure it represents
type ModelDefinitions = map[string]interface{}

// StoreConfig config for the Log Store
type StoreConfig struct {
	LocalDbConfig nutsdb.Options
	// run cronjob every X amount of time, note that if the job is still running
	// when next Interval arrives, it'll be put on hold until job is done
	Interval       int64
	Models         ModelDefinitions
	TargetDbConfig *TargetDbConfig
}

// DefaultStoreConfig default values
var DefaultStoreConfig = &StoreConfig{
	LocalDbConfig: nutsdb.Options{
		EntryIdxMode:         nutsdb.HintKeyValAndRAMIdxMode,
		SegmentSize:          8 * 1024 * 1024, //set to 8mb
		NodeNum:              1,
		RWMode:               nutsdb.FileIO,
		SyncEnable:           true,
		StartFileLoadingMode: nutsdb.MMap,
	},
	Interval: 1,
	TargetDbConfig: &TargetDbConfig{
		Server: "127.0.0.1",
	},
}

// Store stores the logs in an embedded database (nutsdb)
type Store struct {
	db     *nutsdb.DB
	config *StoreConfig
}

// NewStore inits new instance of Store with default options.
func NewStore(cfg *StoreConfig) *Store {
	if len(cfg.Models) == 0 {
		panic("Please define model definitions in config (StoreConfig.Models)")
	}
	if cfg.LocalDbConfig.Dir == "" {
		panic("Please define local database's directory (StoreConfig.LocalDbConfig.Dir)")
	}
	if cfg.TargetDbConfig.Database == "" {
		panic("Please define target database (StoreConfig.TargetDbConfig.Database)'")
	}
	if cfg.TargetDbConfig.Server == "127.0.0.1" {
		log.Println("Setting target database to localhost...")
	}

	db, err := nutsdb.Open(cfg.LocalDbConfig)
	if err != nil {
		log.Fatal("NewStore error: ", err)
	}
	return &Store{db, cfg}
}

// Close closes connection
func (s *Store) Close() {
	s.db.Close()
}

func forEach(tx *nutsdb.Tx, targetTable string, model interface{}, callback func(rec *Record) bool) (err error) {
	var list [][]byte
	list, err = tx.LRange(bucket, []byte(targetTable), 0, -1)
	if err != nil {
		return err
	}
	for i := range list {
		rec := &Record{New: model, Old: model}
		err := decodeBytes(list[i], rec)
		if err != nil {
			return err
		}
		if callback(rec) == false {
			return fmt.Errorf("forEach loop forcibly stopped")
		}
	}
	return
}

// GetAll returns all values (decode into `mappingModel`) in `targetTable`, `callback` fired once per record
func (s *Store) GetAll(targetTable string, mappingModel interface{}, callback func(rec *Record) bool) (err error) {
	return s.db.View(func(tx *nutsdb.Tx) error {
		return forEach(tx, targetTable, mappingModel, callback)
	})
}

// Size get current "sync-pending" records from local database
func (s *Store) Size(targetTable string) (size int, err error) {
	s.db.View(func(tx *nutsdb.Tx) error {
		size, err = tx.LSize(bucket, []byte(targetTable))
		if err != nil {
			return err
		}
		return nil
	})
	return
}

// LRem remove List from left
func (s *Store) LRem(targetTable string, count int) (err error) {
	return s.db.Update(func(tx *nutsdb.Tx) error {
		return tx.LRem(bucket, []byte(targetTable), count)
	})
}

// LogInsert records the insert event into Store
func (s *Store) LogInsert(targetTable string, model interface{}) error {

	return s.db.Update(func(tx *nutsdb.Tx) error {
		rec := &Record{Action: InsertAction, New: model}
		return rpush(encodeBytes(rec))(tx, targetTable)
	})
}

// LogUpdate records the update event into Store
func (s *Store) LogUpdate(targetTable string, oldModel interface{}, newModel interface{}) error {

	return s.db.Update(func(tx *nutsdb.Tx) error {
		rec := &Record{Action: UpdateAction, Old: oldModel, New: newModel}
		return rpush(encodeBytes(rec))(tx, targetTable)
	})
}

// LogDelete records the delete event into Store
func (s *Store) LogDelete(targetTable string, model interface{}) error {

	return s.db.Update(func(tx *nutsdb.Tx) error {
		rec := &Record{Action: DeleteAction, Old: model}
		return rpush(encodeBytes(rec))(tx, targetTable)
	})
}

// truncate target bucket (targetTable), note that this does not empty .dat files
func (s *Store) truncate(targetTable string) error {

	return s.db.Update(func(tx *nutsdb.Tx) error {
		if err := tx.LRem(bucket, []byte(targetTable), 0); err != nil {
			// ignore "the list not found" error
			if err.Error() == "the list not found" || err.Error() == "err bucket" {
				// log.Printf("[warning] the bucket/list not found")
				return nil
			}
			return fmt.Errorf("truncate error: %v", err.Error())
		}
		return nil
	})
}

func rpush(buf []byte, err error) (with func(tx *nutsdb.Tx, targetTable string) error) {
	return func(tx *nutsdb.Tx, targetTable string) error {
		if err != nil {
			return fmt.Errorf("Marshal error: %v", err)
		}
		if err := tx.RPush(bucket, []byte(targetTable), buf); err != nil {
			return fmt.Errorf("Insert error: %v", err)
		}
		return nil
	}
}

func encode(act Action, models ...interface{}) ([]byte, error) {
	// note that a new encoder state has to be created for each op, as these ops
	// are not "streamable"
	buffer := &bytes.Buffer{}
	enc := gob.NewEncoder(buffer)
	// first bit: action
	err := enc.Encode(act)
	if err != nil {
		return nil, err
	}
	// following data: models
	for _, model := range models {
		if model == nil {
			continue
		}
		err = enc.Encode(model)
		if err != nil {
			return nil, err
		}
	}
	return buffer.Bytes(), nil
}

// serialize `model` into bytes
func encodeBytes(rec *Record) ([]byte, error) {
	return encode(rec.Action, rec.New, rec.Old)
}

// decode into type i, return a new copy of type i
func decode(dec *gob.Decoder, i interface{}) (newValue interface{}, err error) {
	// get concrete type (from pointer)
	t := reflect.TypeOf(i).Elem()
	// create new instance of the concrete type
	newValue = reflect.New(t).Interface()
	err = dec.Decode(newValue)
	if err != nil {
		return nil, fmt.Errorf("Decode error: %v", err)
	}
	return
}

// deserialize bytes into record
func decodeBytes(input []byte, rec *Record) (err error) {
	dec := gob.NewDecoder(bytes.NewBuffer(input))

	var action Action
	err = dec.Decode(&action)
	if err != nil {
		return fmt.Errorf("Decode error: %v", err)
	}
	rec.Action = action

	var v interface{}
	if action == InsertAction || action == UpdateAction {
		v, err = decode(dec, rec.New)
		rec.New = v
	}
	if action == UpdateAction {
		v, _ = decode(dec, rec.Old)
		rec.Old = v
	}
	if action == DeleteAction {
		v, err = decode(dec, rec.Old)
		rec.Old = v
	}
	return err
}
