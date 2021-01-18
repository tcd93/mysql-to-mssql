package parser

import (
	"crypto/tls"
	"fmt"

	cn "github.com/siddontang/go-mysql/canal"
)

// Config object, see https://pkg.go.dev/github.com/siddontang/go-mysql@v1.1.0/canal
type Config struct {
	ServerID          uint32
	Addr              string
	User              string
	Password          string
	IncludeTableRegex []string
	ExcludeTableRegex []string
	UseDecimal        bool
	Charset           string
	TLSConfig         *tls.Config
}

// ModelMap maps the actual table name & the table structure
//
// Example:
// "User" table on RDBMS has 2 column id & name, correspond to TableModel.ID & TableModel.Name
//	var m = ModelMap{"User": &struct{
// 		ID     int       `gorm:"column:id"`
// 		Name   string    `gorm:"column:name"` // struct tag ``gorm:"column:xxx"` is required
// }}
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
			handler,
			models,
			canal,
		},
		cfg,
		handler,
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
