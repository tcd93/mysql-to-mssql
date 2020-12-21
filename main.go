package main

import (
	"fmt"
	"gonnextor/mysql/parser"
	"gonnextor/syncer"
)

/* -------------- this section should be generated automatically -------------- */

const (
	// local directory to store messages
	localDir = "D:/temp/nutsdb"
)

// define a which table name maps to which data model, table name is case sensitive
var dataModels = syncer.ModelDefinitions{
	"staff": &StaffModel{},
}

func getMySQLConfig() parser.Config {
	return parser.Config{
		ServerID:          1,
		Addr:              "127.0.0.1:3306",
		User:              "root",
		Password:          "root",
		IncludeTableRegex: []string{"sakila\\.staff"}, // We only care table staff
	}
}

func getSQLServerConfig() *syncer.TargetDbConfig {
	return &syncer.TargetDbConfig{
		Server:   "127.0.0.1",
		Database: "gonnextor",
		Log:      63,
	}
}

/* -------------------------------- end automated code generation --------------------------------*/

var store *syncer.Store

type mainHandler struct{}

func (*mainHandler) OnInsert(schemaName string, tableName string, rec interface{}) {
	err := store.LogInsert(tableName, rec)
	if err != nil {
		fmt.Printf("Error during insert: %v\n", err.Error())
	}
}

func (*mainHandler) OnUpdate(schemaName string, tableName string, oldRec interface{}, newRec interface{}) {
	err := store.LogUpdate(tableName, oldRec, newRec)
	if err != nil {
		fmt.Printf("Error during update: %v\n", err.Error())
	}
}

func (*mainHandler) OnDelete(schemaName string, tableName string, rec interface{}) {
	err := store.LogDelete(tableName, rec)
	if err != nil {
		fmt.Printf("Error during delete: %v\n", err.Error())
	}
}

func main() {
	storeCfg := syncer.DefaultStoreConfig
	storeCfg.Models = dataModels
	storeCfg.LocalDbConfig.Dir = localDir
	storeCfg.TargetDbConfig = getSQLServerConfig()

	store = syncer.NewStore(storeCfg)
	parser := parser.NewEventWrapper(dataModels, getMySQLConfig(), &mainHandler{})

	store.Schedule()

	defer parser.Close()
	defer store.Close()
	parser.StartBinlogListener()
}
