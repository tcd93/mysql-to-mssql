package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"mysql2mssql/mysql/parser"
	"time"

	"github.com/siddontang/go-log/log"
)

// Remember, table name is case-sensitive, must match MySQL db exactly
const (
	schema     = "sakila"
	staffTable = "Staff"
)

var models = parser.ModelMap{staffTable: &StaffModel{}}

// StaffModel datamodel, note that fields must be public (to use with `reflect` package)
type StaffModel struct {
	StaffID     int       `gorm:"column:staff_id"`
	FirstName   string    `gorm:"column:first_name"`
	LastName    string    `gorm:"column:last_name"`
	AddressID   int       `gorm:"column:address_id"`
	Email       *string   `gorm:"column:email"`
	Picture     *[]byte   `gorm:"column:picture"`
	StoreID     int       `gorm:"column:store_id"`
	Active      bool      `gorm:"column:active"`
	LastUpdated time.Time `gorm:"column:last_update"`
}

func getDefaultConfig() parser.Config {
	cer, err := tls.LoadX509KeyPair("client-cert.pem", "client-key.pem")
	if err != nil {
		panic(err)
	}
	serverCA, _ := ioutil.ReadFile("server-ca.pem")
	rootCAPool := x509.NewCertPool()
	rootCAPool.AppendCertsFromPEM(serverCA)

	return parser.Config{
		ServerID:          1,
		Addr:              "35.240.181.214:3306",
		User:              "root",
		Password:          "root",
		IncludeTableRegex: []string{fmt.Sprintf("%s\\.%s", schema, staffTable)}, // We only care table staff
		UseDecimal:        true,
		TLSConfig: &tls.Config{
			ServerName:   "mysql-to-mssql-syncer:a1", // <gcp-project-id>:<cloud-sql-instance>
			Certificates: []tls.Certificate{cer},
			RootCAs:      rootCAPool,
		},
	}
}

type mainHandler struct{}

func (*mainHandler) OnInsert(schemaName string, tableName string, rec interface{}) {
	// in reality, there should be somesort of switch-case tableName check here...
	log.Infof("Inserting on table %s.%s\nvalues: %#v\n", schemaName, tableName, rec.(StaffModel))
}

func (*mainHandler) OnUpdate(schemaName string, tableName string, oldRec interface{}, newRec interface{}) {
	log.Infof("Updating on table %s.%s\nvalues: %#v\n", schemaName, tableName, newRec.(StaffModel))
}

func (*mainHandler) OnDelete(schemaName string, tableName string, rec interface{}) {
	log.Infof("Deleting on table %s.%s\nvalues: %#v\n", schemaName, tableName, rec.(StaffModel))
}

func main() {
	wrapper := parser.NewEventWrapper(models, getDefaultConfig(), &mainHandler{})

	defer wrapper.Close()
	wrapper.StartBinlogListener()
}
