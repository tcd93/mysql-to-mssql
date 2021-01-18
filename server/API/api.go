// Package API provides interface for interacting with core modules such as listening changes, syncing...
package API

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"gonnextor/db"
	"gonnextor/mysql/parser"
	"gonnextor/server/param"
	"gonnextor/syncer"
	"io"
	"os"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	dynamicstruct "github.com/ompluscator/dynamic-struct"
	"github.com/siddontang/go-log/log"
)

// The API for interacting with source/target databases
type API struct {
	DataModels   *parser.ModelMap
	DBInterface  db.Interface
	eventWrapper *parser.EventHandlerWrapper
	syncer       *syncer.Syncer
	logStore     *syncer.Store
}

var json = jsoniter.ConfigCompatibleWithStandardLibrary

const bucket = "API"

// Put defines what table/columns to Parse & Sync
func (a *API) Put(p param.StructRequest) (strct interface{}, err error) {
	strct = generateStruct(p.Columns)
	(*a.DataModels)[p.Table] = strct
	err = a.storeToDB(p)
	return
}

// Get retrieves a saved table structure
func (a *API) Get(tabName string) (strct interface{}) {
	return (*a.DataModels)[tabName]
}

// StartParser inits Parser to listen to changes on source db & log changes to Log Store
func (a *API) StartParser(p param.StartParserRequest) {
	a.logStore = syncer.NewStore(a.DBInterface, *a.DataModels)
	w := parser.NewEventWrapper(*a.DataModels, createParserConfig(p), a)
	go w.StartBinlogListener()
	a.eventWrapper = w
}

// StopParser stops the Parser listener
func (a *API) StopParser() (err error) {
	if a.eventWrapper == nil {
		return errors.New("Parser is closed, or has not been started")
	}
	a.eventWrapper.Close()
	a.eventWrapper = nil
	return
}

// LogChan streams the logged contents from Parser & Syncer
func (a *API) LogChan(stream chan string, quit chan struct{}) {
	ticker := time.NewTicker(500 * time.Millisecond) // send log messages to client at an interval

	// following code switch out old logger (os.Stdout) to a new one which is able to stream output
	// to both console & a new buffer to send back to client

	b := &bytes.Buffer{}
	mw := io.MultiWriter(os.Stdout, b)

	stdHandler, _ := log.NewStreamHandler(os.Stdout)
	oldLogger := log.NewDefault(stdHandler)

	multiplexer, _ := log.NewStreamHandler(mw)
	logger := log.NewDefault(multiplexer)

	logger.SetLevel(0) // must set level to "trace" for `print` func (used by Syncer's db driver) to work
	log.SetDefaultLogger(logger)
	a.syncer.SetLogger(logger)

	for {
		select {
		case <-ticker.C:
			if b.Len() > 0 {
				data := strings.Split(b.String(), "\n")
				for _, line := range data {
					stream <- line
				}
				b.Reset()
			}
		case <-quit:
			logger.SetLevel(2)
			log.SetDefaultLogger(oldLogger)
			a.syncer.SetLogger(oldLogger)
			b.Reset()
			return
		}
	}
}

// LoadDataModels is called during init to load saved table structures to memory
func (a *API) LoadDataModels() (err error) {
	entries, err := a.DBInterface.GetAll(bucket)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		p := &param.StructRequest{}
		err = json.Unmarshal(entry.Value, p)
		if err != nil {
			return err
		}
		(*a.DataModels)[string(entry.Key)] = generateStruct(p.Columns)
	}
	return nil
}

// StartSyncer to periodically sync changes recorded in log store to target DB
func (a *API) StartSyncer(p param.StartSyncerRequest) {
	a.syncer = a.createSyncer(p)
	a.syncer.Schedule()
}

// StopSyncer stops the syncing job
func (a *API) StopSyncer() (err error) {
	if err = a.syncer.Stop(); err != nil {
		return err
	}
	a.syncer.Close()
	return
}

////////////////////////////////////////////////////////////////

func createParserConfig(param param.StartParserRequest) parser.Config {
	return parser.Config{
		ServerID:          param.ServerID,
		Addr:              param.Addr,
		Charset:           param.Charset,
		IncludeTableRegex: param.IncludeTableRegex,
		ExcludeTableRegex: param.ExcludeTableRegex,
		User:              param.User,
		Password:          param.Password,
		UseDecimal:        param.UseDecimal,
		TLSConfig:         createTLSConfig(param.TLSConfig.ServerName, param.TLSConfig.ServerCA, param.TLSConfig.ClientCert, param.TLSConfig.ClientKey),
	}
}

func createTLSConfig(serverName string, serverCA string, clientCert string, clientKey string) *tls.Config {
	if serverName == "" {
		return nil
	}
	rootCAPool := x509.NewCertPool()
	ok := rootCAPool.AppendCertsFromPEM([]byte(serverCA))
	if !ok {
		panic("Failed to parse root certificate, please check validity of serverCA string")
	}
	cert, err := tls.X509KeyPair([]byte(clientCert), []byte(clientKey))
	if err != nil {
		panic(err)
	}
	return &tls.Config{
		ServerName:   serverName,
		Certificates: []tls.Certificate{cert},
		RootCAs:      rootCAPool,
	}
}

func (a *API) createSyncer(param param.StartSyncerRequest) *syncer.Syncer {
	if param.Interval == 0 {
		param.Interval = 1
	}
	if a.DataModels == nil || len(*a.DataModels) == 0 {
		panic("DataModel is not yet defined, please call /struct/put first")
	}
	if a.logStore == nil {
		panic("Log Store is not initialized, please call /parser/start first")
	}
	var tDBConf syncer.TargetDbConfig
	tDBConf.Server = param.Server
	tDBConf.Database = param.Database
	if param.Appname != "" {
		tDBConf.Appname = param.Appname
	}
	if param.Encrypt != "" {
		tDBConf.Encrypt = param.Encrypt
	}
	if param.Userid != "" {
		tDBConf.Userid = param.Userid
	}
	if param.Password != "" {
		tDBConf.Password = param.Password
	}
	if param.Log != 0 {
		tDBConf.Log = param.Log
	}
	return syncer.NewSyncer(tDBConf, param.Interval, a.logStore)
}

func generateStruct(cols []param.Column) interface{} {
	d := dynamicstruct.NewStruct()
	for _, c := range cols {
		t := db.Convert(c.Type)
		var prm string
		if c.IsPrimary {
			prm = ";primaryKey"
		}
		// capitalize first letter to create exported field name for reflection access
		d.AddField(strings.Title(c.Name), t, fmt.Sprintf(`gorm:"column:%s%s"`, c.Name, prm))
	}
	return d.Build().New()
}

func (a *API) storeToDB(param param.StructRequest) (err error) {
	bytes, err := json.Marshal(param)
	return a.DBInterface.Put(bucket, param.Table, bytes, 0)
}

////////////////////////////////////////////////////////////////
// TODO: use schema

// OnInsert implements EventHandlerInterface
func (a *API) OnInsert(schemaName string, tableName string, rec interface{}) {
	err := a.logStore.LogInsert(tableName, rec)
	if err != nil {
		log.Errorf("Error during insert: %v\n", err.Error())
	}
}

// OnUpdate implements EventHandlerInterface
func (a *API) OnUpdate(schemaName string, tableName string, oldRec interface{}, newRec interface{}) {
	err := a.logStore.LogUpdate(tableName, oldRec, newRec)
	if err != nil {
		log.Errorf("Error during update: %v\n", err.Error())
	}
}

// OnDelete implements EventHandlerInterface
func (a *API) OnDelete(schemaName string, tableName string, rec interface{}) {
	err := a.logStore.LogDelete(tableName, rec)
	if err != nil {
		log.Printf("Error during delete: %v\n", err.Error())
	}
}

////////////////////////////////////////////////////////////////
