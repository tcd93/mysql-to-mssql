# mysql/parser
`wrapper.go` provides an interface to read MYSQL binlog in real-time (based on [go-mysql's canal]("github.com/siddontang/go-mysql/canal"))

#### REQUIREMENT
- __Go 1.14 64-bit (:warning: don't install any 32-bit versions, or Go >= 1.15)__
---
### EXAMPLE
```
package main

import (
	"mysql/parser"
	"github.com/shopspring/decimal"
)

// In MYSQL, create table User with 3 columns: 
// - id (int) 
// - name (varchar)
// - val (decimal(19,4))


// Define the data model (correspond to the 'watched' table in MYSQL)
//  - field "ID" => column "id"
//  - field "Name" => column "name"
// Note that first letter in field name (of the struct) must be in Uppercase;
// the tags must be in format: `gorm:column:...`
// To see the data type mapping between MYSQL & Golang, refer to later section
type UserModel struct {
	ID       int       `gorm:"column:id"`
	Name     string    `gorm:"column:name"`
	Decimal	 decimal.Decimal `gorm:"column:name"`
}


// Define the handler & 3 methods (must be named OnInsert, OnUpdate & OnDelete respectively)
type eventHandler struct{}

func (*eventHandler) OnInsert(schemaName string, tableName string, rec interface{}) {
    dataModel := rec.(UserModel) // "casts" the generic interface{} to our pre-defined data model
    dataModel.Name... // now we can access the fields on model

	// action on insert event
	// log.Infof("Received insert event: %v\n", dataModel)
}
func (*eventHandler) OnUpdate(schemaName string, tableName string, oldRec interface{}, newRec interface{}) {
    tableInfo := fmt.Sprintf("%s.%s", schemaName, tableName) // retrieve info on the affected table

	// action on update event
}
func (*eventHandler) OnDelete(schemaName string, tableName string, rec interface{}) {
	// action on delete event
}


// Create database config
func config() parser.Config {
	// The "IncludeTableRegex" specify the list of regex strings of table names that canal should listen to
	// Read more: https://pkg.go.dev/github.com/siddontang/go-mysql@v1.1.0/canal
	return parser.Config{
		ServerID:          1,
		Addr:              "127.0.0.1:3306",
		User:              "root",
		Password:          "root",
		IncludeTableRegex: []string{"test\\.User"}, // In this example only include "User" table in schema "test"
		UseDecimal:        true, // when set to true, go-mysql will use Decimal package for decimal types, it'll come with a performance penalty, but more precision
								 // when set to false (default), go-mysql will use native float64 type, whose max precision is at around 16-17 digits
	}
}

// main function to execute
func main() {
	// create the wrapper object 
	wrapper := parser.NewEventWrapper(&UserModel{}, config(), &eventHandler{})

	defer wrapper.Close() // close canal after done
	wrapper.StartBinlogListener() // start listening events on the canal
}
```
Execute `go run .` to start listening

---
### MAPPINGS
If MYSQL column is __NULLABLE__, map it to the type pointer instead of actual type

For example, if the mapping is specified as _int => smallint_, when the field on MYSQL is set as NULL, Go will view it as number 0

If the mapping is _*int => smallint_, then Go would view it as nil

|     Go    | Go (NULL)  |         MySQL         |       Signed       |      Unsigned      |                    Remark                	  |
|-----------|------------|-----------------------|--------------------|--------------------|----------------------------------------------|
|    int    |    *int    |          int          | :heavy_check_mark: | :heavy_check_mark: |                                        	  |
|    int    |    *int    |        tinyint        | :heavy_check_mark: | :heavy_check_mark: |                                        	  |
|    bool   |    *bool   |        tinyint        | :heavy_check_mark: | :heavy_check_mark: |       1 = true; 0 (&other) = false     	  |
|    int    |    *int    |       mediumint       | :heavy_check_mark: | :heavy_check_mark: |                                        	  |
|    int    |    *int    |        smallint       | :heavy_check_mark: | :heavy_check_mark: |                                        	  |
|    int    |    *int    |         bigint        | :heavy_check_mark: |        :x:         |                                       	  |
|    uint   |    *uint   |         bigint        |         :x:        | :heavy_check_mark: |                                        	  |
|  float32  |  *float32  |         float         | :heavy_check_mark: | :heavy_check_mark: |      precision is at around 6 digits   	  |
|  float64  |  *float64  |         double        | :heavy_check_mark: | :heavy_check_mark: |      precision is at 15 - 17 digits    	  |
|  float64  |  *float64  |        decimal        | :heavy_check_mark: | :heavy_check_mark: |   :warning: precision is at 15 - 17 digits!  |
|  Decimal  |  *Decimal  |        decimal        | :heavy_check_mark: | :heavy_check_mark: |   :warning: [slower performance!](https://github.com/shopspring/decimal#why-isnt-the-api-similar-to-bigints)			  |
| time.Time | *time.Time |   datetime/timestamp  |                    |                    |                                           	  |
| time.Time | *time.Time |          date         |                    |                    |                                           	  |
|   string  |   *string  |          time         |                    |                    |                   "00:59:59"              	  |
|   string  |   *string  |   char/varchar/text   |                    |                    |      also support blob, binary, varbinary 	  |
|   []byte  |   *[]byte  | blob/binary/varbinary |                    |                    |                                           	  |
|  []string |  *[]string |          set          |                    |                    |        return the set's string literals   	  |
|   string  |   *string  |          enum         |                    |                    |        return the value's string literal  	  |

Also support advanced mapping from MYSQL JSON type, we just need to add `fromjson` to the struct tag

See `parser_test.go` or `wrapper_test.go`[line 114-172] for examples

|         Go        | MySQL |        Example       |
|-------------------|-------|----------------------|
|   map[string]int  |  json |   {"a": 1, "b": 2}   |
| map[string]string |  json | {"a": "z", "b": "y"} |
|       []int       |  json |       "[1,2,3]"      |

### UNIT TESTING
1. Set up local MYSQL instance on port 3306 (tested on 8.0)
2. Create a user with username/pass: __root/root__
3. Create a schema named "_test_"
4. CD into _mysql\parser_ & Execute `go test -v`