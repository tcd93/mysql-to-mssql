package syncer

import (
	"bytes"
	"fmt"
	"gonnextor/db"
	"reflect"

	"encoding/gob"
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

const bucket = "store"

// ModelDefinitions defines table structure, see package server.StructRequest
type ModelDefinitions = map[string]interface{}

// Store stores the logs in a database, see package server.StartSyncerRequest
type Store struct {
	LocalDb db.Interface
	Models  ModelDefinitions
}

// DefaultStore use inmemdb
var DefaultStore = &Store{
	LocalDb: db.UseInmemDB(),
}

// NewStore inits new instance of Store
func NewStore(db db.Interface, models ModelDefinitions) *Store {
	if len(models) == 0 {
		panic("Please define model definitions in config")
	}
	if db.Type() == "nutsdb" && db.Dir() == "" {
		panic("Please define local database's directory")
	}

	return &Store{
		LocalDb: db,
		Models:  models,
	}
}

// Close closes database connection
func (s *Store) Close() {
	s.LocalDb.Release()
}

func forEach(list [][]byte, model interface{}, callback func(rec *Record) error) (err error) {
	for i := range list {
		rec := &Record{New: model, Old: model}
		err := decodeBytes(list[i], rec)
		if err != nil {
			return err
		}
		if err = callback(rec); err != nil {
			return fmt.Errorf("%v - forEach loop forcibly stopped", err)
		}
	}
	return
}

// GetAll returns all values (decoded into mappingModel) in targetTable, callback is fired once per record
func (s *Store) GetAll(targetTable string, mappingModel interface{}, callback func(rec *Record) error) (err error) {
	list, err := s.LocalDb.GetAllKey(bucket, targetTable)
	if err != nil {
		return err
	}
	return forEach(list, mappingModel, callback)
}

// Size get current "sync-pending" records from local database
func (s *Store) Size(targetTable string) (size int, err error) {
	return s.LocalDb.Size(bucket, targetTable)
}

// LRem remove List from left
func (s *Store) LRem(targetTable string, count int) (err error) {
	return s.LocalDb.Rem(bucket, targetTable, count)
}

// LogInsert records the insert event into Store
func (s *Store) LogInsert(targetTable string, model interface{}) error {
	rec := &Record{Action: InsertAction, New: model}
	b, err := encodeBytes(rec)
	if err != nil {
		return fmt.Errorf("Marshal error: %v", err)
	}
	return s.LocalDb.Push(bucket, targetTable, b)
}

// LogUpdate records the update event into Store
func (s *Store) LogUpdate(targetTable string, oldModel interface{}, newModel interface{}) error {
	rec := &Record{Action: UpdateAction, Old: oldModel, New: newModel}
	b, err := encodeBytes(rec)
	if err != nil {
		return fmt.Errorf("Marshal error: %v", err)
	}
	return s.LocalDb.Push(bucket, targetTable, b)
}

// LogDelete records the delete event into Store
func (s *Store) LogDelete(targetTable string, model interface{}) error {
	rec := &Record{Action: DeleteAction, Old: model}
	b, err := encodeBytes(rec)
	if err != nil {
		return fmt.Errorf("Marshal error: %v", err)
	}
	return s.LocalDb.Push(bucket, targetTable, b)
}

// truncate target bucket (targetTable)
func (s *Store) truncate(targetTable string) error {
	return s.LocalDb.Truncate(bucket, targetTable)
}

func encode(act Action, models ...interface{}) ([]byte, error) {
	buffer := &bytes.Buffer{}
	enc := gob.NewEncoder(buffer)
	// first byte: action
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

func encodeBytes(rec *Record) ([]byte, error) {
	return encode(rec.Action, rec.New, rec.Old)
}

// decode into type i, return a new copy of type i
func decode(dec *gob.Decoder, i interface{}) (newValue interface{}, err error) {
	t := reflect.TypeOf(i).Elem()
	newValue = reflect.New(t).Interface()
	err = dec.Decode(newValue)
	if err != nil {
		return nil, fmt.Errorf("Decode error: %v", err)
	}
	return
}

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
