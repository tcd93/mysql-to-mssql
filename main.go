package main

import (
	"fmt"
	"gonnextor/mysql/parser"
	"gonnextor/syncer"
)

// define a which table name maps to which data model.
// table name is case sensitive
var dataModels = parser.ModelMap{
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

func getSQLServerConfig() syncer.Config {
	return syncer.Config{
		Server:   "127.0.0.1",
		Database: "gonnextor",
		Log:      63,
	}
}

var sync *syncer.Syncer

type mainHandler struct{}

func (*mainHandler) OnInsert(schemaName string, tableName string, rec interface{}) {
	model := rec.(StaffModel)
	// log.Infof("Inserting on table %s.%s\nvalues: %#v\n", table.Schema, table.Name, model)

	// perform mssql sync on same table name
	sync.Insert(tableName, model)
}

func (*mainHandler) OnUpdate(schemaName string, tableName string, oldRec interface{}, newRec interface{}) {
	// oldModel := oldRec.(StaffModel)
	newModel := newRec.(StaffModel)
	oldModel := oldRec.(StaffModel)
	// log.Infof("Updating on table %s.%s\nvalues: %#v\n", table.Schema, table.Name, newModel)

	// _, err := sync.Update(tableName, newModel, "staff_id = ?", oldModel.StaffID)
	_, err := sync.UpdateOnPK(tableName, oldModel, newModel)
	if err != nil {
		fmt.Printf("Error during update: %v\n", err.Error())
	}
}

func (*mainHandler) OnDelete(schemaName string, tableName string, rec interface{}) {
	model := rec.(StaffModel)
	// log.Infof("Deleting on table %s.%s\nvalues: %#v\n", table.Schema, table.Name, model)

	// _, err := sync.Delete(tableName, "staff_id = ?", model.StaffID)
	_, err := sync.DeleteOnPK(tableName, model)
	if err != nil {
		fmt.Printf("Error during delete: %v\n", err.Error())
	}
}

func main() {
	wrapper := parser.NewEventWrapper(dataModels, getMySQLConfig(), &mainHandler{})

	sync = syncer.NewSyncer(getSQLServerConfig())

	defer wrapper.Close()
	wrapper.StartBinlogListener()
}
