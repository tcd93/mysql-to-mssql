# mysql/parser
`wrapper.go` provides an interface to read MYSQL binlog in real-time (based on [go-mysql's canal]("github.com/siddontang/go-mysql/canal"))

#### REQUIREMENT
- __Go 1.14 64-bit (:warning: don't install any 32-bit versions, or Go >= 1.15)__
---
### [EXAMPLE](./example/README.md)
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

See `parser_test.go` or `wrapper_test.go` for examples

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