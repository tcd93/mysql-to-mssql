package syncer

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/shopspring/decimal"
)

type column struct {
	name         string
	isPrimaryKey bool
	fieldType    string // reflect.Type.String()
}

type value = interface{}

func getColumns(model interface{}, onlyPrimary bool) (columns []column, values []value) {
	reflectedValue := reflect.Indirect(reflect.ValueOf(model))
	structType := reflectedValue.Type()
	columns = make([]column, 0, structType.NumField())
	values = make([]value, 0, structType.NumField())

	for k := 0; k < structType.NumField(); k++ {
		var colName string
		parsedTag := parseTagSetting(structType.Field(k).Tag)
		if len(parsedTag) < 2 {
			continue
		}

		if colName = parsedTag[1]; colName == "column" || colName == "" {
			continue
		}

		var isPrimaryKey bool
		if len(parsedTag) > 2 && parsedTag[2] == "primaryKey" {
			isPrimaryKey = true
		}

		if onlyPrimary && !isPrimaryKey {
			continue
		} else {
			field := reflectedValue.Field(k)
			fieldType := structType.Field(k).Type.String()

			columns = append(columns, column{
				name:         colName,
				isPrimaryKey: isPrimaryKey,
				fieldType:    fieldType,
			})

			if field.Kind() == reflect.Ptr && field.IsNil() {
				values = append(values, nil)
			} else {
				// MSSQL does not have a matching type for uint64, so we try to convert it to Decimal (21)
				// 	when the int is bigger than 8 bytes (this MSSQL driver does not do that implicitly)
				var fieldValue interface{}
				if (fieldType == "uint" || fieldType == "*uint") && reflect.Indirect(field).Uint() > 9223372036854775807 {
					dec, _ := decimal.NewFromString(fmt.Sprint(reflect.Indirect(field).Uint()))
					fieldValue = dec
				} else {
					fieldValue = field.Interface()
				}
				values = append(values, fieldValue)
			}
		}
	}
	return
}

// returns array of parsed tags, assuming input `tags` follow convention
// example:
//	`gorm:"column:pkCol;primaryKey"` // tags separator must be ";", first tag must be "column:...", second tag primaryKey is optional
func parseTagSetting(tags reflect.StructTag) []string {
	values := strings.SplitN(tags.Get("gorm"), ";", 2)
	colInfo := strings.SplitN(values[0], ":", 2)
	return append(colInfo, values[1:]...)
}
