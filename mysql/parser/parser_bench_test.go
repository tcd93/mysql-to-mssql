package parser

import (
	"testing"
	"time"

	"github.com/siddontang/go-mysql/canal"
	"github.com/siddontang/go-mysql/schema"
)

type handler struct{}

func (h *handler) OnInsert(schemaName string, tableName string, rec interface{})                   {}
func (h *handler) OnUpdate(schemaName string, tableName string, rec interface{}, rec2 interface{}) {}
func (h *handler) OnDelete(schemaName string, tableName string, rec interface{})                   {}

func Benchmark_EventHandler(b *testing.B) {
	e, _ := mockRowEventBench(b.N)
	base := &baseEventHandler{
		DummyEventHandler: canal.DummyEventHandler{},
		models: map[string]interface{}{
			"test_bench": &benchNonNullStruct{},
		},
		EventHandlerInterface: &handler{},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		base.OnRow(e)
	}
}

type benchNonNullStruct struct {
	Int       int       `gorm:"column:int"`
	Int1      int       `gorm:"column:int1"`
	Int2      int       `gorm:"column:int2"`
	Int3      int       `gorm:"column:int3"`
	BigInt    int       `gorm:"column:bigint"`
	Bool      bool      `gorm:"column:bool"`
	Float     float64   `gorm:"column:float"`
	Enum      string    `gorm:"column:enum"`
	String    string    `gorm:"column:string"`
	TimeStamp time.Time `gorm:"column:timeStamp"`
	DateTime  time.Time `gorm:"column:dateTime"`
	DateTime2 time.Time `gorm:"column:dateTime2"`
	EnumNull  string    `gorm:"column:enum_null"`
	Set       []string  `gorm:"column:st"`
	ByteText  []byte    `gorm:"column:byte_text"`
}

// mock an n-row insert event
func mockRowEventBench(inserts int) (e *canal.RowsEvent, insertRows []interface{}) {
	rows := make([][]interface{}, inserts)

	insertRows = make([]interface{}, inserts)
	for i := 0; i < inserts; i++ {
		insertRows = make([]interface{}, 15)
		insertRows[0] = 1
		insertRows[1] = 2
		insertRows[2] = 3
		insertRows[3] = 4
		insertRows[4] = 5
		insertRows[5] = 1000000
		insertRows[6] = 1.123
		insertRows[7] = int64(1)
		insertRows[8] = "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat"
		insertRows[9] = "2018-02-16 14:28:09"
		insertRows[10] = "2019-10-18 01:18:19"
		insertRows[11] = "2019-10-18 01:18:19"
		insertRows[12] = nil
		insertRows[13] = "Set3,Set4"
		insertRows[14] = []byte("Lorem ipsum dolor sit amet")
		rows[i] = insertRows
	}

	columns := make([]schema.TableColumn, 15)
	columns[0] = schema.TableColumn{Name: "int", Type: schema.TYPE_NUMBER}
	columns[1] = schema.TableColumn{Name: "int1", Type: schema.TYPE_NUMBER}
	columns[2] = schema.TableColumn{Name: "int2", Type: schema.TYPE_NUMBER}
	columns[3] = schema.TableColumn{Name: "int3", Type: schema.TYPE_NUMBER}
	columns[4] = schema.TableColumn{Name: "bigint", Type: schema.TYPE_NUMBER}
	columns[5] = schema.TableColumn{Name: "bool", Type: schema.TYPE_NUMBER}
	columns[6] = schema.TableColumn{Name: "float", Type: schema.TYPE_FLOAT}
	columns[7] = schema.TableColumn{Name: "enum", Type: schema.TYPE_ENUM, EnumValues: []string{"Active", "Deleted"}}
	columns[8] = schema.TableColumn{Name: "string", Type: schema.TYPE_STRING}
	columns[9] = schema.TableColumn{Name: "timeStamp", Type: schema.TYPE_TIMESTAMP}
	columns[10] = schema.TableColumn{Name: "dateTime", Type: schema.TYPE_DATETIME}
	columns[11] = schema.TableColumn{Name: "dateTime2", Type: schema.TYPE_DATETIME}
	columns[12] = schema.TableColumn{Name: "enum_null", Type: schema.TYPE_ENUM, EnumValues: []string{"Active", "Deleted"}}
	columns[13] = schema.TableColumn{Name: "st", Type: schema.TYPE_ENUM, SetValues: []string{"Set1", "Set2", "Set3", "Set4", "Set5"}}
	columns[14] = schema.TableColumn{Name: "byte_text", Type: schema.TYPE_STRING}

	table := schema.Table{Schema: "test", Name: "test_bench", Columns: columns}

	return &canal.RowsEvent{Table: &table, Action: canal.InsertAction, Rows: rows}, insertRows
}
