package parser

import (
	"crypto/tls"
	"fmt"

	cn "github.com/siddontang/go-mysql/canal"
)

// Config object for mysql parser
type Config struct {
	ServerID uint32
	Addr     string
	User     string
	Password string

	// IncludeTableRegex or ExcludeTableRegex should contain database name,
	// only a table which matches IncludeTableRegex and dismatches ExcludeTableRegex will be processed.
	// Example:
	//	IncludeTableRegex : [".*\\.canal"], ExcludeTableRegex : ["mysql\\..*"]
	// This will include all database's 'canal' table, except database 'mysql'.
	// Default of IncludeTableRegex and ExcludeTableRegex are empty, this will include all tables
	IncludeTableRegex []string
	ExcludeTableRegex []string

	// When set to true, go-mysql will use Decimal package for decimal types, it'll come with a performance penalty, but more precision.
	// When set to false (default), go-mysql will use native float64 type, whose max precision is at around 16-17 digits.
	UseDecimal bool

	// Charset is for MySQL client character set
	Charset string

	// If not nil, use the provided tls.Config to connect to the database using TLS/SSL.
	TLSConfig *tls.Config
}

// ModelMap maps the actual table name & the table schema structure
//
// Example:
//	var m = ModelMap{"User": &TableModel{}}
//	// "User" table on RDBMS has 2 column id & name, correspond to TableModel.ID & TableModel.Name
//	// struct tag ``gorm:"column:xxx"` is required
// 	type TableModel struct {
// 		ID     int       `gorm:"column:id"`
// 		Name   string    `gorm:"column:name"`
// 	}
//
// For a list of available mappings between SQL type & Go type, see https://gitlab.nexdev.net/jaden.tang/gonnextor/-/tree/master/mysql/parser#mappings
type ModelMap = map[string]interface{}

// EventHandlerWrapper is a wrapper for canal's event handler, implements `EventHandlerInterface`
type EventHandlerWrapper struct {
	baseHandler baseEventHandler
	cfg         Config
	EventHandlerInterface
}

// NewEventWrapper creates new instance of `EventHandlerWrapper`
func NewEventWrapper(models ModelMap, cfg Config, handler EventHandlerInterface) *EventHandlerWrapper {

	canal, err := cn.NewCanal(&cn.Config{
		ServerID:          cfg.ServerID,
		Addr:              cfg.Addr,
		User:              cfg.User,
		Password:          cfg.Password,
		IncludeTableRegex: cfg.IncludeTableRegex,
		ExcludeTableRegex: cfg.ExcludeTableRegex,
		UseDecimal:        cfg.UseDecimal,
		Flavor:            "mysql",
		Charset:           cfg.Charset,
		TLSConfig:         cfg.TLSConfig,
	})
	if err != nil {
		panic(fmt.Sprintf("Error during init: %v", err))
	}

	return &EventHandlerWrapper{
		baseEventHandler{
			cn.DummyEventHandler{},
			handler, // pass handler into baseEventHandler for processing, generate callback events
			models,
			canal,
		},
		cfg,
		handler, // keep a reference of handler exposed in EventHandlerWrapper so that other modules can listen to callback events
	}
}

// StartBinlogListener starts listening from the binlog position directly, ignore mysqldump
func (w *EventHandlerWrapper) StartBinlogListener() {

	canal := w.baseHandler.canal
	if canal == nil {
		panic(fmt.Sprint("canal is nil, make sure you have called NewEventWrapper() to create new canal instance"))
	}
	canal.SetEventHandler(&w.baseHandler)

	coords, err := canal.GetMasterPos()
	if err == nil {
		canal.RunFrom(coords)
	}
}

// Close event
func (w *EventHandlerWrapper) Close() {
	w.baseHandler.canal.Close()
	w.baseHandler.canal = nil
}
