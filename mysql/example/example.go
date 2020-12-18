package example

import (
	"gonnextor/mysql/parser"
	"time"

	"github.com/siddontang/go-log/log"
)

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
	return parser.Config{
		ServerID:          1,
		Addr:              "127.0.0.1:3306",
		User:              "root",
		Password:          "root",
		IncludeTableRegex: []string{"sakila\\.staff"}, // We only care table staff
	}
}

type mainHandler struct{}

func (*mainHandler) OnInsert(schemaName string, tableName string, rec interface{}) {
	log.Infof("Inserting on table %s.%s\nvalues: %#v\n", schemaName, tableName, rec.(StaffModel))
}

func (*mainHandler) OnUpdate(schemaName string, tableName string, oldRec interface{}, newRec interface{}) {
	log.Infof("Updating on table %s.%s\nvalues: %#v\n", schemaName, tableName, newRec.(StaffModel))
}

func (*mainHandler) OnDelete(schemaName string, tableName string, rec interface{}) {
	log.Infof("Deleting on table %s.%s\nvalues: %#v\n", schemaName, tableName, rec.(StaffModel))
}

func main() {
	wrapper := parser.NewEventWrapper(parser.ModelMap{"Staff": &StaffModel{}}, getDefaultConfig(), &mainHandler{})

	defer wrapper.Close()
	wrapper.StartBinlogListener()
}
