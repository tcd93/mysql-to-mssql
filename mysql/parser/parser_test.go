package parser

import (
	"testing"
	"time"

	"github.com/siddontang/go-mysql/canal"
	"github.com/siddontang/go-mysql/schema"
)

func Test_getBinLogData_Access(t *testing.T) {

	e, _ := mockInsertRowEvent(2)
	model := &binlogTestStruct{}
	var firstModel, secondModel binlogTestStruct

	for i := 0; i < 2; i++ {
		_ = getBinLogData(e, i, model).(binlogTestStruct)
		if i == 0 {
			firstModel = *model
		}
		if i == 1 {
			secondModel = *model
		}
	}
	firstModel.String = "changed now"
	// make sure each loop returns a new model, not just a reference to old one
	if firstModel.String == secondModel.String {
		t.Errorf("firstModel is the same as secondModel")
	}
}

func Test_getBinLogData_Insert(t *testing.T) {

	e, insertRows := mockInsertRowEvent(1)
	model := getBinLogData(e, 0, &binlogTestStruct{}).(binlogTestStruct)

	if model.Int != insertRows[0] {
		t.Errorf("Int value did not update.")
	}
	if model.Bool != true {
		t.Errorf("Bool value did not update.")
	}
	if model.Float != insertRows[2] {
		t.Errorf("Float value did not update.")
	}
	if model.Enum != "Active" {
		t.Errorf("Enum value did not update.")
	}
	if model.String != insertRows[4] {
		t.Errorf("String value did not update.")
	}
	timeValue, _ := time.Parse("2006-01-02 15:04:05", insertRows[5].(string))
	if model.TimeStamp.Unix() != timeValue.Unix() {
		t.Errorf("TimeStamp value did not update.")
	}
	timeValue, _ = time.Parse("2006-01-02 15:04:05", insertRows[6].(string))
	if model.DateTime.Unix() != timeValue.Unix() {
		t.Errorf("DateTime value did not update.")
	}
	if model.EnumNull != "" {
		t.Errorf("Null enum was not parsed as string.")
	}

}

func Test_getBinLogData_Update(t *testing.T) {

	// model := binlogTestStruct{}
	rows := make([][]interface{}, 2)
	insertRows := make([]interface{}, 9)
	insertRows[0] = 1
	insertRows[1] = 0
	insertRows[2] = 1.123
	insertRows[3] = int64(1)
	insertRows[4] = "test text"
	insertRows[5] = "2018-02-16 14:28:09"
	insertRows[6] = "2018-02-17 17:28:11"
	insertRows[7] = int64(1)
	insertRows[8] = []byte("test text")
	rows[0] = insertRows
	updateRows := make([]interface{}, 9)
	updateRows[0] = 3
	updateRows[1] = 1
	updateRows[2] = 2.234
	updateRows[3] = int64(2)
	updateRows[4] = "test2 text2"
	updateRows[5] = "2018-02-16 15:28:09"
	updateRows[6] = "2018-02-17 17:28:11"
	updateRows[7] = nil
	updateRows[8] = []byte("test2 text2")
	rows[1] = updateRows
	columns := make([]schema.TableColumn, 9)

	columns[0] = schema.TableColumn{Name: "int", Type: schema.TYPE_NUMBER}
	columns[1] = schema.TableColumn{Name: "bool", Type: schema.TYPE_NUMBER}
	columns[2] = schema.TableColumn{Name: "float", Type: schema.TYPE_FLOAT}
	columns[3] = schema.TableColumn{Name: "enum", Type: schema.TYPE_ENUM, EnumValues: []string{"Active", "Deleted"}}
	columns[4] = schema.TableColumn{Name: "string", Type: schema.TYPE_STRING}
	columns[5] = schema.TableColumn{Name: "timeStamp", Type: schema.TYPE_TIMESTAMP}
	columns[6] = schema.TableColumn{Name: "dateTime", Type: schema.TYPE_DATETIME}
	columns[7] = schema.TableColumn{Name: "enum_null", Type: schema.TYPE_ENUM, EnumValues: []string{"Active", "Deleted"}}
	columns[8] = schema.TableColumn{Name: "byte_text", Type: schema.TYPE_STRING}
	table := schema.Table{Schema: "test", Name: "test", Columns: columns}

	e := canal.RowsEvent{Table: &table, Action: canal.UpdateAction, Rows: rows}
	model := getBinLogData(&e, 1, &binlogTestStruct{}).(binlogTestStruct)

	if model.Int != updateRows[0] {
		t.Errorf("Int value did not update.")
	}
	if model.Bool != true {
		t.Errorf("Bool value did not update.")
	}
	if model.Float != updateRows[2] {
		t.Errorf("Float value did not update.")
	}
	if model.Enum != "Deleted" {
		t.Errorf("Enum value did not update.")
	}
	if model.String != updateRows[4] {
		t.Errorf("String value did not update.")
	}
	timeValue, _ := time.Parse("2006-01-02 15:04:05", updateRows[5].(string))
	if model.TimeStamp.Unix() != timeValue.Unix() {
		t.Errorf("TimeStamp value did not update.")
	}
	timeValue, _ = time.Parse("2006-01-02 15:04:05", updateRows[6].(string))
	if model.DateTime.Unix() != timeValue.Unix() {
		t.Errorf("DateTime value did not update.")
	}
	if model.String != updateRows[4].(string) {
		t.Errorf("String value did not update.")
	}
	if model.EnumNull != "" {
		t.Errorf("Enum nulled did not update.")
	}

}

func TestPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	rows := make([][]interface{}, 1)
	insertRows := make([]interface{}, 6)
	insertRows[0] = 1

	rows[0] = insertRows

	columns := make([]schema.TableColumn, 6)

	columns[0] = schema.TableColumn{Name: "int", Type: schema.TYPE_NUMBER}

	table := schema.Table{Schema: "test", Name: "test", Columns: columns}

	e := canal.RowsEvent{Table: &table, Action: canal.InsertAction, Rows: rows}
	_ = getBinLogData(&e, 0, &binlogInvalidStruct{}).(binlogInvalidStruct)
}

func TestJson(t *testing.T) {
	// model := JSONData{}
	rows := make([][]interface{}, 1)
	insertRows := make([]interface{}, 6)
	insertRows[0] = 1
	insertRows[1] = `{"int":1,"test":"test"}`
	insertRows[2] = `{"a":"a","b":"b"}`
	insertRows[3] = `[2,4,6]`
	rows[0] = insertRows

	columns := make([]schema.TableColumn, 6)

	columns[0] = schema.TableColumn{Name: "int", Type: schema.TYPE_NUMBER}
	columns[1] = schema.TableColumn{Name: "struct_data", Type: schema.TYPE_STRING}
	columns[2] = schema.TableColumn{Name: "map_data", Type: schema.TYPE_STRING}
	columns[3] = schema.TableColumn{Name: "slice_data", Type: schema.TYPE_STRING}
	table := schema.Table{Schema: "test", Name: "test", Columns: columns}

	e := canal.RowsEvent{Table: &table, Action: canal.InsertAction, Rows: rows}
	model := getBinLogData(&e, 0, &jSONData{}).(jSONData)
	if model.StructData.Test != "test" || model.StructData.Int != 1 {
		t.Errorf("Struct from json parsing failed.")
	}
	if val, ok := model.MapData["a"]; ok && val != "a" && len(model.MapData) != 2 {
		t.Errorf("Map json parsing failed.")
	}
	if len(model.SliceData) != 3 || model.SliceData[0] != 2 || model.SliceData[2] != 6 {
		t.Errorf("Sliced json parsing failed.")
	}
}

type binlogTestStruct struct {
	Int             int        `gorm:"column:int"`
	Bool            bool       `gorm:"column:bool"`
	Float           float64    `gorm:"column:float"`
	Enum            string     `gorm:"column:enum"`
	String          string     `gorm:"column:string"`
	TimeStamp       *time.Time `gorm:"column:timeStamp"`
	DateTime        *time.Time `gorm:"column:dateTime"`
	EnumNull        string     `gorm:"column:enum_null"`
	ByteText        []byte     `gorm:"column:byte_text"`
	WillNotParse    int
	WillNotParseAlt int `gorm:"column"`
}

type binlogInvalidStruct struct {
	Int int `gorm:"column:id"`
}

type jSONData struct {
	Int        int `gorm:"column:int"`
	StructData struct {
		Test string `json:"test"`
		Int  int    `json:"int"`
	} `gorm:"column:struct_data;fromJson"`
	MapData   map[string]string `gorm:"column:map_data;fromJson"`
	SliceData []int             `gorm:"column:slice_data;fromJson"`
}

// mock an n-row insert event
func mockInsertRowEvent(inserts int) (e *canal.RowsEvent, insertRows []interface{}) {
	rows := make([][]interface{}, inserts)

	insertRows = make([]interface{}, 10)
	for i := 0; i < inserts; i++ {
		insertRows = make([]interface{}, 10)
		insertRows[0] = 1
		insertRows[1] = 1
		insertRows[2] = 1.123
		insertRows[3] = int64(1)
		insertRows[4] = "test text"
		insertRows[5] = "2018-02-16 14:28:09"
		insertRows[6] = "2019-10-18 01:18:19"
		insertRows[7] = nil
		insertRows[8] = []byte("test text")
		rows[i] = insertRows
	}

	columns := make([]schema.TableColumn, 9)
	columns[0] = schema.TableColumn{Name: "int", Type: schema.TYPE_NUMBER}
	columns[1] = schema.TableColumn{Name: "bool", Type: schema.TYPE_NUMBER}
	columns[2] = schema.TableColumn{Name: "float", Type: schema.TYPE_FLOAT}
	columns[3] = schema.TableColumn{Name: "enum", Type: schema.TYPE_ENUM, EnumValues: []string{"Active", "Deleted"}}
	columns[4] = schema.TableColumn{Name: "string", Type: schema.TYPE_STRING}
	columns[5] = schema.TableColumn{Name: "timeStamp", Type: schema.TYPE_TIMESTAMP}
	columns[6] = schema.TableColumn{Name: "dateTime", Type: schema.TYPE_DATETIME}
	columns[7] = schema.TableColumn{Name: "enum_null", Type: schema.TYPE_ENUM, EnumValues: []string{"Active", "Deleted"}}
	columns[8] = schema.TableColumn{Name: "byte_text", Type: schema.TYPE_STRING}

	table := schema.Table{Schema: "test", Name: "test", Columns: columns}

	return &canal.RowsEvent{Table: &table, Action: canal.InsertAction, Rows: rows}, insertRows
}
