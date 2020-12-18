#### REQUIREMENT
[TCP/IP enabled](http://www.manifold.net/doc/mfd9/enable_tcp_ip_for_sql_server.htm) in SQL Server Configuration Manager

# store

**`Store`** is a cronjob service that runs in short intervals to get "uncommitted" changes from MySQL to sync to MSSQL (via _syncer_)

Internally _store_ uses **nutsdb** to capture all changes from MySQL (should being coming from the __mysql/parser__)

**nutsdb pro:**
- Embedded
- Support other types of datastuctures that other dbs dont, _store_ uses `List` heavily

**nutsdb cons:**
- Default config for datafile size is 8MB, so we can't insert records that are bigger than this size, and we can't change this config after first run
- A instance of `List` is always kept in memory (see `DB.ListIdx`), **so the tool may eat up all RAM if kept running for too long?**
- `List` is not thread-safe, but it doesn't matter in our use-case (there's only one binlog & one running instance)

# syncer
`syncer.go` provides an interface to insert/update/delete MSSQL (based on mssql [driver]("https://github.com/denisenkom/go-mssqldb"))

---
### EXAMPLE
```
package main

import (
	"mssql/syncer"
	"github.com/shopspring/decimal"
)

// In MSSQL, create table User with 3 columns: 
// - id (int) (can be primary key) 
// - name (varchar)
// - val (decimal(19,4))


// Define the data model
//  - field "ID" => column "id"
//  - field "Name" => column "name"
// To see the data type mapping between MSSQL & Golang, refer to later section
type UserModel struct {
	ID       int       `gorm:"column:id"`
	Name     string    `gorm:"column:name"`
	Decimal	 decimal.Decimal `gorm:"column:name"`
}


// Create new syncer instance
// that connects to localhost, database "gonnextor".
// Log level 63 is for debugging purpose only
func createSyncer() *canal.Canal {
	return NewSyncer(syncer.Config{
		Server:   "127.0.0.1",
		Database: "gonnextor",
		Log:      63,
	})
}

// main function to execute
func main() {
	//create a new model with data
    model := &UserModel{
		ID:     1,
		Name:   "中文 English Tiếng Việt",
		Decimal: decimal.NewFromString("12345.987"),
	}

    syncer := createSyncer()
    // insert into table User with data from model
    syncer.Insert("User", model)

    // update table User where id = 1, the third param is a string to specify "where" condition to append to the prepared query statement (must use question marks)
	// followed by a list of variables to "fill" those question marks
    syncer.Update("User", model, "id = ? AND username = ?", 1, "username to delete")

    // delete 
    syncer.Delete("User", "id = ?", 1)
}
```
---
### MAPPINGS
In the following map table, MSSQL "**Numeric**" types include: bit, tinyint, smallint, int, bigint, float, decimal, smallmoney & money;  
"**Text**" types include: nvarchar/varchar/char/nchar;

|     Go    | Go (NULL)  |             MSSQL            |                    Remark                  |
|-----------|------------|------------------------------|--------------------------------------------|
|    int    |    *int    |            Numeric           |                                        	 |
|    int    |    *int    |            Numeric           |                                        	 |
|    bool   |    *bool   |            Numeric           |       1 = true; 0 (&other) = false     	 |
|    int    |    *int    |            Numeric           |                                        	 |
|    int    |    *int    |            Numeric           |                                        	 |
|    int    |    *int    |            Numeric           |                                       	 |
|    uint   |    *uint   |            Numeric           |                                     	     |
|  float32  |  *float32  |            Numeric           | To preserve display correctness, avoid mapping with FLOAT type (as MSSQL's float is 64-bit), use SMALLMONEY instead  |
|  float64  |  *float64  |            Numeric           |                                            |
|  float64  |  *float64  |            Numeric           |                                            |
|  Decimal  |  *Decimal  |            Numeric           |                                            |
| time.Time | *time.Time |    Text/Date/DateTime/Time   |                                            |
| time.Time | *time.Time |    Text/Date/DateTime/Time   |                                            |
|   string  |   *string  |             ALL              | A string is most versatile & can convert to anything if it is convertable, else this'd throw exception |
|   []byte  |   *[]byte  |     Text/Binary/Varbinary    |                                            |

Array & maps are not supported

### UNIT TESTING
1. Set up local MSSQL instance with Single-sign-on
3. Create a database named "_gonnextor_"
4. CD into _mssql\syncer_ & Execute `go test -v`
