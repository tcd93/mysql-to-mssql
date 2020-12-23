package parser

import (
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/shopspring/decimal"
	"github.com/siddontang/go-mysql/canal"
	"github.com/siddontang/go-mysql/schema"
)

// getBinLogData reads `RowsEvent` and parses into `element` (struct)
//
// Sample Model:
// 	{
//		ID         int       			`gorm:"column:id_column_name_in_mysql"`
// 		Varchar    string    			`gorm:"column:varchar_column_name"`
// 		Number     int       			`gorm:"column:number_column_name"`
// 		Blob       []byte    			`gorm:"column:blob_column_name"`
// 		DateTime   time.Time 			`gorm:"column:datetime_column_name"`
// 		StructData TestStruct           `gorm:"column:struct_column_name;fromJson"`
// 		MapData    map[string]string    `gorm:"column:map_column_name;fromJson"`
// 		SliceData  []int                `gorm:"column:slice_column_name;fromJson"`
//	}
func getBinLogData(e *canal.RowsEvent, rowNum int, placeHolder interface{}) interface{} {
	element := placeHolder
	reflectedValue := reflect.Indirect(reflect.ValueOf(element))
	structType := reflectedValue.Type() // the input element should be Struct type (as in later we parse the tags)

	for k := 0; k < structType.NumField(); k++ {
		field := reflectedValue.Field(k)
		fieldType := field.Type()
		parsedTag := parseTagSetting(structType.Field(k).Tag)
		if len(parsedTag) < 2 {
			continue
		}

		var colName string
		if colName = parsedTag[1]; colName == "column" || colName == "" {
			continue
		}
		columnID := getColumnIDByRealName(e, colName)

		// note: process* functions mutate (set) the field
		if processNillable(field, e, rowNum, columnID) == false {
			if processNonNil(field, e, rowNum, columnID) == false {
				// json
				if len(parsedTag) > 2 {
					if ok := parsedTag[2]; ok == "fromJson" {
						newObject := reflect.New(fieldType).Interface()
						json := getString(e, rowNum, columnID)
						if json != nil {
							jsoniter.Unmarshal([]byte(*json), &newObject)
							field.Set(reflect.ValueOf(newObject).Elem().Convert(fieldType))
						}
					}
				} else {
					log.Printf("[parser] %v is not supported", fieldType.String())
					return nil
				}
			}
		}
	}
	return reflectedValue.Interface()
}

// process NON-NULL values, can not return NULL type, for example: an MYSQL's NULL INT column will map to golang's 0 int value.
// should have higher performance than `processNillable` on INT fields
func processNonNil(field reflect.Value, event *canal.RowsEvent, rowNum int, columnID int) (processed bool) {

	processed = true
	fieldType := field.Type()

	switch fieldType.Name() {
	case "bool":
		boolVal := getBool(event, rowNum, columnID)
		if boolVal != nil {
			field.SetBool(*boolVal)
		}
	case "int":
		intVal := getInt(event, rowNum, columnID)
		if intVal != nil {
			field.SetInt(*intVal)
		}
	case "uint":
		uIntVal := getUint(event, rowNum, columnID)
		if uIntVal != nil {
			field.SetUint(*uIntVal)
		}
	case "string":
		sVal := getString(event, rowNum, columnID)
		if sVal != nil {
			field.SetString(*sVal)
		}
	case "Time":
		timeVal := getTime(event, rowNum, columnID)
		if timeVal != nil {
			field.Set(reflect.ValueOf(*timeVal))
		}
	case "float32":
		floatVal := getFloat32(event, rowNum, columnID)
		if floatVal != nil {
			field.Set(reflect.ValueOf(*floatVal))
		}
	case "float64":
		floatVal := getFloat64(event, rowNum, columnID)
		if floatVal != nil {
			field.SetFloat(*floatVal)
		}
	default:
		if fieldType.String() == "[]string" { // SET
			set := getSet(event, rowNum, columnID)
			if set != nil {
				field.Set(reflect.ValueOf(*set))
			}
		} else if fieldType.String() == "[]uint8" || fieldType.String() == "[]byte" { // BLOB/BINARY
			bt := getByte(event, rowNum, columnID)
			if bt != nil {
				field.SetBytes(*bt)
			}
		} else if fieldType.String() == "decimal.Decimal" {
			dc := getDecimal(event, rowNum, columnID)
			if dc != nil {
				field.Set(reflect.ValueOf(*dc))
			}
		} else {
			processed = false
		}
	}
	return
}

// process NULL values, can not return NULL type, for example: an MYSQL's NULL INT column will map to golang's nil value (*int).
func processNillable(field reflect.Value, event *canal.RowsEvent, rowNum int, columnID int) (processed bool) {

	processed = true
	fieldType := field.Type()

	switch fieldType.String() {
	case "*bool":
		boolVal := getBool(event, rowNum, columnID)
		field.Set(reflect.ValueOf(boolVal))
	case "*int":
		int64Val := getInt(event, rowNum, columnID)
		if int64Val != nil {
			intVal := int(*int64Val)
			field.Set(reflect.ValueOf(&intVal)) //`field.Set()` can't implicityly infer *int64 to *int
		}
	case "*uint":
		uIntVal := getUint(event, rowNum, columnID)
		if uIntVal != nil {
			field.Set(reflect.ValueOf(*uIntVal))
		}
	case "*string":
		sVal := getString(event, rowNum, columnID)
		field.Set(reflect.ValueOf(sVal))
	case "*time.Time":
		timeVal := getTime(event, rowNum, columnID)
		field.Set(reflect.ValueOf(timeVal))
	case "*float32":
		floatVal := getFloat32(event, rowNum, columnID)
		field.Set(reflect.ValueOf(floatVal))
	case "*float64":
		floatVal := getFloat64(event, rowNum, columnID)
		field.Set(reflect.ValueOf(floatVal))
	default:
		if fieldType.String() == "*[]string" { // SET
			set := getSet(event, rowNum, columnID)
			field.Set(reflect.ValueOf(set))
		} else if fieldType.String() == "*[]uint8" || fieldType.String() == "*[]byte" { // BLOB/BINARY
			bt := getByte(event, rowNum, columnID)
			field.Set(reflect.ValueOf(bt))
		} else if fieldType.String() == "*decimal.Decimal" {
			field.Set(reflect.ValueOf(getDecimal(event, rowNum, columnID)))
		} else {
			processed = false
		}
	}
	return
}

// getSet returns specific field's Time from `RowsEvent`.
// Use this method on MYSQL SET types.
// for Set, the returned value is the int64 representation of the reversed set order.
//
// For example: with Set('1','2','3','4','5'), if we select '1' & '3', then the parser would read the bits from back to front
// which is 00101, resulting in 5.
// To decode this, we'd convert the number back to binary, then reverse the string to get logical order ("10100");
// finally we map each bits to the `values`, 1 = selected, 0 = not selected
func getSet(event *canal.RowsEvent, rowNum int, columnID int) *[]string {

	if event.Rows[rowNum][columnID] == nil {
		return nil
	}

	if event.Table.Columns[columnID].Type == schema.TYPE_SET {
		values := event.Table.Columns[columnID].SetValues
		if len(values) == 0 {
			return nil
		}
		reversedIntRepresentation := event.Rows[rowNum][columnID].(int64)
		formatString := fmt.Sprintf("%%0%vb", len(values))
		logicalOrderedBits := reverse(fmt.Sprintf(formatString, reversedIntRepresentation)) //choosable values: [a b c d], chosen: [a b] => 1100
		result := make([]string, 0, len(logicalOrderedBits))
		for index, v := range values {
			// 49 = "1"
			if logicalOrderedBits[index] == 49 {
				result = append(result, v)
			}
		}
		return &result
	}
	return nil
}

// getByte returns specific field's binary data from `RowsEvent`.
// Use this method on MYSQL BLOB/BINARY/VARBINARY types.
// `RowsEvent` is a set of affected rows in a batch of insert/update/delete
func getByte(event *canal.RowsEvent, rowNum int, columnID int) *[]byte {

	if event.Rows[rowNum][columnID] == nil {
		return nil
	}

	var t []byte
	switch event.Table.Columns[columnID].Type {
	case schema.TYPE_STRING:
		t = event.Rows[rowNum][columnID].([]byte)
	case schema.TYPE_BINARY:
		t = []byte(event.Rows[rowNum][columnID].(string))
	}
	return &t
}

// getString returns specific field's string value (varchar/text... in MySQL) from `RowsEvent`.
// Supports CHAR, VARCHAR, TEXT, TIME and ENUM.
// `RowsEvent` is a set of affected rows in a batch of insert/update/delete
func getString(event *canal.RowsEvent, rowNum int, columnID int) *string {

	if event.Rows[rowNum][columnID] == nil {
		return nil
	}

	var t string
	switch event.Table.Columns[columnID].Type {
	case schema.TYPE_ENUM:
		values := event.Table.Columns[columnID].EnumValues
		if len(values) == 0 {
			return nil
		}
		t = values[event.Rows[rowNum][columnID].(int64)-1]
	case schema.TYPE_STRING, schema.TYPE_TIME, schema.TYPE_BINARY, schema.TYPE_JSON:
		var ok bool
		if t, ok = event.Rows[rowNum][columnID].(string); !ok {
			t = string(event.Rows[rowNum][columnID].([]byte)) // in case user mistakenly typed BLOB as string
		}
	}
	return &t
}

// getTime returns specific field's Time from `RowsEvent`.
// Use this method on MYSQL DATETIME/TIMESTAMP/DATE types (does not support TIME)
// `RowsEvent` is a set of affected rows in a batch of insert/update/delete
func getTime(event *canal.RowsEvent, rowNum int, columnID int) *time.Time {

	if event.Rows[rowNum][columnID] == nil {
		return nil
	}

	switch event.Table.Columns[columnID].Type {
	case schema.TYPE_TIMESTAMP, schema.TYPE_DATETIME:
		t, err := time.Parse("2006-01-02 15:04:05", event.Rows[rowNum][columnID].(string))
		if err != nil {
			panic(fmt.Sprintf("time.Parse failed: %v", err))
		}
		return &t
	case schema.TYPE_DATE:
		t, err := time.Parse("2006-01-02", event.Rows[rowNum][columnID].(string))
		if err != nil {
			panic(fmt.Sprintf("time.Parse failed: %v", err))
		}
		return &t
	default:
		panic(fmt.Sprintf("getTime failed, make sure you are converting DateTime/Timestamp/Date only"))
	}
}

// getInt returns specific field's int64 value from `RowsEvent`.
// `RowsEvent` is a set of affected rows in a batch of insert/update/delete
func getInt(event *canal.RowsEvent, rowNum int, columnID int) *int64 {

	if event.Rows[rowNum][columnID] == nil {
		return nil
	}

	var t int64
	switch event.Table.Columns[columnID].Type {
	case schema.TYPE_NUMBER, schema.TYPE_MEDIUM_INT, schema.TYPE_BIT:

		switch event.Rows[rowNum][columnID].(type) {
		case int8:
			t = int64(event.Rows[rowNum][columnID].(int8))
		case int16:
			t = int64(event.Rows[rowNum][columnID].(int16))
		case int32:
			t = int64(event.Rows[rowNum][columnID].(int32))
		case int64:
			t = event.Rows[rowNum][columnID].(int64)
		case int:
			t = int64(event.Rows[rowNum][columnID].(int))
		case uint8:
			t = int64(event.Rows[rowNum][columnID].(uint8))
		case uint16:
			t = int64(event.Rows[rowNum][columnID].(uint16))
		case uint32:
			t = int64(event.Rows[rowNum][columnID].(uint32))
		case uint64:
			t = int64(event.Rows[rowNum][columnID].(uint64))
		case uint:
			t = int64(event.Rows[rowNum][columnID].(uint))
		}
	}
	return &t
}

// getUint returns specific field's uint64 value from `RowsEvent`.
// Use this method on MYSQL UNSIGNED BIGINT types (because it would overflow with int)
// `RowsEvent` is a set of affected rows in a batch of insert/update/delete
func getUint(event *canal.RowsEvent, rowNum int, columnID int) *uint64 {

	if event.Rows[rowNum][columnID] == nil {
		return nil
	}

	var t uint64
	if event.Table.Columns[columnID].Type == schema.TYPE_NUMBER {
		t = event.Rows[rowNum][columnID].(uint64)
	}
	return &t
}

// getFloat32 returns specific field's float32 value from `RowsEvent`.
// Use this method on MYSQL FLOAT types.
// `RowsEvent` is a set of affected rows in a batch of insert/update/delete
func getFloat32(event *canal.RowsEvent, rowNum int, columnID int) *float32 {

	if event.Rows[rowNum][columnID] == nil {
		return nil
	}

	var t float32
	if event.Table.Columns[columnID].Type == schema.TYPE_FLOAT {
		t = event.Rows[rowNum][columnID].(float32)
	}
	return &t
}

// getFloat64 returns specific field's float64 value from `RowsEvent`.
// Use this method on MYSQL DOUBLE/DECIMAL types.
// `RowsEvent` is a set of affected rows in a batch of insert/update/delete
func getFloat64(event *canal.RowsEvent, rowNum int, columnID int) *float64 {

	if event.Rows[rowNum][columnID] == nil {
		return nil
	}

	var t float64
	switch event.Table.Columns[columnID].Type {
	case schema.TYPE_FLOAT, schema.TYPE_DECIMAL:
		t = event.Rows[rowNum][columnID].(float64)
	}
	return &t
}

func getDecimal(event *canal.RowsEvent, rowNum int, columnID int) *decimal.Decimal {

	if event.Rows[rowNum][columnID] == nil {
		return nil
	}

	var t decimal.Decimal
	if event.Table.Columns[columnID].Type == schema.TYPE_DECIMAL {
		t = event.Rows[rowNum][columnID].(decimal.Decimal)
	}
	return &t
}

// getBool returns specific field's boolean value (usually TINYINT(1) in MySQL) from `RowsEvent`.
// `RowsEvent` is a set of affected rows in a batch of insert/update/delete
func getBool(event *canal.RowsEvent, rowNum int, columnID int) *bool {

	if event.Rows[rowNum][columnID] == nil {
		return nil
	}

	var t bool
	if *getInt(event, rowNum, columnID) == 1 {
		t = true
	} else {
		t = false
	}
	return &t
}

func getColumnIDByRealName(event *canal.RowsEvent, name string) int {

	for id, value := range event.Table.Columns {
		if value.Name == name {
			return id
		}
	}
	panic(fmt.Errorf("There is no column %s in table %s.%s", name, event.Table.Schema, event.Table.Name))
}

// returns array of parsed tags, assuming input `tags` follow convention
// example:
//	`gorm:"column:responseObject;fromJson"` // tags separator must be ";", first tag must be "column:...", second tag fromJson is optional
func parseTagSetting(tags reflect.StructTag) []string {
	// Example:
	//	tag `gorm:"column:responseObject;fromJson"`

	//	=> [column:responseObject fromJson]
	values := strings.SplitN(tags.Get("gorm"), ";", 2)
	colInfo := strings.SplitN(values[0], ":", 2)
	//	=> [column responseObject fromJson]
	return append(colInfo, values[1:]...)
}

// reverse a string
func reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}
