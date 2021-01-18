package parser

import (
	"fmt"
	"testing"
	"time"

	. "github.com/pingcap/check"
	d "github.com/shopspring/decimal"
	"github.com/siddontang/go-log/log"
	"github.com/siddontang/go-mysql/mysql"
)

// Test basic operations from DB
// Requirement:
// 	- MySQL on localhost 3306 with user: root/root
// 	- set up schema named "test"

func TestEventsFromDB(t *testing.T) {
	TestingT(t)
}

type wrapperTestSuite struct {
	*EventHandlerWrapper
}

var insertChannel = make(chan wrapperTest)
var updateChannel = make(chan updateTuple)
var deleteChannel = make(chan wrapperTest)

type wrapperTestHandler struct{}

func (*wrapperTestHandler) OnInsert(schemaName string, tableName string, rec interface{}) {
	insertChannel <- rec.(wrapperTest)
}
func (*wrapperTestHandler) OnUpdate(schemaName string, tableName string, oldRec interface{}, newRec interface{}) {
	updateChannel <- updateTuple{
		oldRec.(wrapperTest),
		newRec.(wrapperTest),
	}
}
func (*wrapperTestHandler) OnDelete(schemaName string, tableName string, rec interface{}) {
	deleteChannel <- rec.(wrapperTest)
}

var _ = SerialSuites(&wrapperTestSuite{
	NewEventWrapper(
		map[string]interface{}{"wrapper_test": &wrapperTest{}},
		Config{
			ServerID:          1,
			Addr:              "127.0.0.1:3306",
			User:              "root",
			Password:          "root",
			IncludeTableRegex: []string{"test\\.wrapper_test"},
			UseDecimal:        true,
		},
		&wrapperTestHandler{},
	),
})

func (s *wrapperTestSuite) execute(query string, args ...interface{}) (*mysql.Result, error) {
	return s.baseHandler.canal.Execute(query, args...)
}

func (s *wrapperTestSuite) SetUpSuite(c *C) {
	fmt.Println("--- WRAPPER : SetUpSuite ---")
	// start listening for callback, wait a second to make sure it is started up
	go s.StartBinlogListener()
	<-time.After(time.Second)

	s.execute("SET GLOBAL binlog_format = 'ROW'")
}

func (s *wrapperTestSuite) TearDownSuite(c *C) {
	// To test the heartbeat and read timeout,so need to sleep 1 seconds without data transmission
	c.Logf("Start testing the heartbeat and read timeout")
	time.Sleep(time.Second)

	s.Close()
	// s.baseHandler.canal = nil
}

type wrapperTest struct {
	ID     int            `gorm:"column:id"`
	Name   string         `gorm:"column:name"`
	Bo     bool           `gorm:"column:bo"`
	Mi     int            `gorm:"column:mi"`
	Umi    int            `gorm:"column:umi"`
	Si     int            `gorm:"column:si"`
	SiU    int            `gorm:"column:si_u"`
	Ti     int            `gorm:"column:ti"`
	TiU    int            `gorm:"column:ti_u"`
	Bi     int            `gorm:"column:bi"`   // BIGINT maps to int (alias for int64 in amd-64)
	BiU    uint           `gorm:"column:bi_u"` // UNSIGNED BIGINT maps to uint
	De     d.Decimal      `gorm:"column:de"`   // DECIMAL also maps to decimal.Decimal (when config's useDecimal = true, otherwise it is float64)
	Fl     float32        `gorm:"column:fl"`   // FLOAT maps to float32
	FlU    float32        `gorm:"column:fl_u"`
	Do     float64        `gorm:"column:do"`     // DOUBLE maps to float64
	Bit    int            `gorm:"column:bit"`    // BIT maps to int64 (as max length in MySQL BIT is 64)
	DTime  time.Time      `gorm:"column:dtime"`  // DATETIME maps to time.Time
	TStam  time.Time      `gorm:"column:tstamp"` // TIMESTAMP maps to time.Time
	Date   time.Time      `gorm:"column:date"`   // DATE maps to time.Time
	Time   string         `gorm:"column:time"`   // *** TIME maps to string ***
	Set    []string       `gorm:"column:st"`     // SET maps to an array of strings
	Enum   string         `gorm:"column:enum"`   // ENUM maps to a string
	Blob   []byte         `gorm:"column:blb"`    // BLOB maps to []byte
	Binary []byte         `gorm:"column:bnr"`    // BINARY maps to []byte
	JSON   map[string]int `gorm:"column:json;fromJson"`
	Array  []int          `gorm:"column:array;fromJson"`
}
type updateTuple struct {
	beforeUpdate wrapperTest
	afterUpdate  wrapperTest
}

// Executed before every tests
func (s *wrapperTestSuite) SetUpTest(c *C) {
	s.execute("DROP TABLE IF EXISTS test.wrapper_test")
	if _, err := s.execute(`
	CREATE TABLE IF NOT EXISTS test.wrapper_test (
		id 		int 			AUTO_INCREMENT,
		name 	varchar(100),
		bo		tinyint 		DEFAULT 0,
		mi 		mediumint 		NOT NULL DEFAULT 0,
		umi 	mediumint 		UNSIGNED NOT NULL DEFAULT 0,
		si		smallint 		DEFAULT 0,
		si_u	smallint		UNSIGNED DEFAULT 0,
		ti		tinyint			DEFAULT 0,
		ti_u	tinyint			UNSIGNED DEFAULT 0,
		bi		bigint			DEFAULT 0,
		bi_u	bigint			UNSIGNED DEFAULT 0,
		de		decimal(60,5) 	DEFAULT 0,
		fl		float			DEFAULT 0,
		fl_u	float			UNSIGNED DEFAULT 0,
		do		double			DEFAULT 0,
		bit		bit(64)			DEFAULT b'0',
		dtime	datetime		DEFAULT '2020-01-01 10:10:10',
		tstamp	timestamp		DEFAULT '2020-01-01 12:12:12',
		date	date			DEFAULT '2019-10-01',
		time	time			DEFAULT '12:59:59',
		st		set('x','y','z','t')	DEFAULT 'x,y',
		enum	enum('a','b')	DEFAULT 'a',			
		blb 	blob 			DEFAULT NULL,	
		bnr		binary(3)		DEFAULT NULL,
		json	json			,
		array	json			,
		PRIMARY KEY(id)
	) ENGINE=innodb;
	`); err != nil {
		log.Errorln(err)
	}
}

func (s *wrapperTestSuite) TestReadingNumbers(c *C) {

	// execute valid SQL statements here
	if _, err := s.execute(`INSERT INTO test.wrapper_test (bo,mi,umi,si,si_u,ti,ti_u,bi,bi_u,de,fl,fl_u,do,bit,dtime,tstamp,date,time,st,enum,blb,bnr,json, array)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		1, -8388608, 16777215, -32768, 65535, -128, 255, 9223372036854775807, uint(18446744073709551615),
		"1111111111111234511189999999987444444444444444444444313.11198", 12.345678, 9999.99, //fl_u
		56.789123456, 9223372036854775807,
		"2020/01/01 10:10:10", "2020-01-01 12:12:12", "1999-12-01", "23:59:59",
		"x,t", "b",
		[]byte("a byte"),
		[]byte("bin"),
		"{\"a\": 1, \"b\": 23}", //json - to map
		"[11, 12, 13]",          //json - to int array
	); err != nil {
		log.Fatal(err)
	}

	select {
	case model := <-insertChannel:
		log.Infof("[TestInsert] - Received insert event: %v\n", model)
		c.Assert(model.ID, Equals, 1)
		c.Assert(model.Bo, IsTrue)
		c.Assert(model.Mi, Equals, -8388608)
		c.Assert(model.Umi, Equals, 16777215)
		c.Assert(model.Si, Equals, -32768)
		c.Assert(model.SiU, Equals, 65535)
		c.Assert(model.Ti, Equals, -128)
		c.Assert(model.TiU, Equals, 255)
		c.Assert(model.Bi, Equals, 9223372036854775807)
		c.Assert(model.BiU, Equals, uint(18446744073709551615))
		obtainedDec, _ := model.De.Value()
		dec, _ := d.NewFromString("1111111111111234511189999999987444444444444444444444313.11198")
		expectedDec, _ := dec.Value()
		c.Assert(obtainedDec, Equals, expectedDec)
		c.Assert(model.Fl, Equals, float32(12.345678))
		c.Assert(model.FlU, Equals, float32(9999.99))
		c.Assert(model.Do, Equals, 56.789123456)
		c.Assert(model.Bit, Equals, 9223372036854775807)
		t, _ := time.Parse("2006-01-02 15:04:05", "2020-01-01 10:10:10")
		c.Assert(model.DTime, Equals, t)
		t, _ = time.Parse("2006-01-02 15:04:05", "2020-01-01 12:12:12")
		c.Assert(model.TStam, Equals, t)
		t, _ = time.Parse("2006-01-02", "1999-12-01")
		c.Assert(model.Date, Equals, t)
		c.Assert(model.Time, Equals, "23:59:59") // display as String for Time type
		c.Assert(model.Set, DeepEquals, []string{"x", "t"})
		c.Assert(model.Enum, Equals, "b")
		c.Assert(model.Blob, DeepEquals, []byte("a byte"))
		c.Assert(model.Binary, DeepEquals, []byte("bin"))
		c.Assert(model.JSON, DeepEquals, map[string]int{"a": 1, "b": 23})
		c.Assert(model.Array, DeepEquals, []int{11, 12, 13})
	}
}

func (s *wrapperTestSuite) TestInsert(c *C) {

	// execute valid SQL statements here
	if _, err := s.execute("INSERT INTO test.wrapper_test (name) VALUES (?)", "string generated from TestInsert 1"); err != nil {
		log.Fatal(err)
	}
	if _, err := s.execute("INSERT INTO test.wrapper_test (name,mi,umi) VALUES (?,?,?), (?,?,?)",
		"string generated from TestInsert 2", 1, 1,
		"string generated from TestInsert 3", -1, 16777215); err != nil {
		log.Fatal(err)
	}
	if _, err := s.execute("INSERT INTO test.wrapper_test (name,blb) VALUES (?,?)", "string generated from TestInsert 4", `\0\iamablob`); err != nil {
		log.Fatal(err)
	}

	var sumMi, sumUMi int
	// 4 inserts = 4 loops
	for i := 0; i < 4; i++ {
		select {
		case model := <-insertChannel:
			log.Infof("[TestInsert] - Received insert event: %v\n", model)
			c.Assert(model, NotNil)
			c.Assert(model.ID, NotNil)
			sumMi += model.Mi
			sumUMi += model.Umi
		}
	}

	c.Assert(sumMi, Equals, 0)         // 0 + 1 - 1 + 0
	c.Assert(sumUMi, Equals, 16777216) // 0 +1 + 16777215 + 0
}

func (s *wrapperTestSuite) TestUpdate(c *C) {

	// execute valid SQL statements here
	if _, err := s.execute("INSERT INTO test.wrapper_test (name,mi,umi) VALUES (?,?,?), (?,?,?)",
		"string generated from TestUpdate 1", 1, 1,
		"string generated from TestUpdate 2", -1, 16777215); err != nil {
		log.Fatal(err)
	}
	// discard all inserts to unblock channel
	go func() {
		for i := 0; i < 2; i++ {
			<-insertChannel
		}
	}()

	if _, err := s.execute("UPDATE test.wrapper_test SET name='RESULT STRING', mi=mi+1, umi=umi-1"); err != nil {
		log.Fatal(err)
	}

	var names []string
	// 2 updates
	for i := 0; i < 2; i++ {
		select {
		case tuple := <-updateChannel:
			log.Infof("[TestUpdate] - Received update event: %v\n", tuple)
			c.Assert(tuple, NotNil)
			c.Assert(tuple.beforeUpdate.ID, Equals, tuple.afterUpdate.ID)
			c.Assert(tuple.beforeUpdate.Mi+1, Equals, tuple.afterUpdate.Mi)
			c.Assert(tuple.beforeUpdate.Umi-1, Equals, tuple.afterUpdate.Umi)
			names = append(names, tuple.afterUpdate.Name)
		}
	}
	for _, n := range names {
		c.Assert(n, Equals, "RESULT STRING")
	}
}

func (s *wrapperTestSuite) TestDelete(c *C) {

	// execute valid SQL statements here
	if _, err := s.execute("INSERT INTO test.wrapper_test (name,mi,umi) VALUES (?,?,?), (?,?,?)",
		"string generated from TestDelete 1", 1, 1,
		"string generated from TestDelete 2", -1, 16777215); err != nil {
		log.Fatal(err)
	}
	// discard all inserts to unblock channel
	go func() {
		for i := 0; i < 2; i++ {
			<-insertChannel
		}
	}()

	if _, err := s.execute("DELETE FROM test.wrapper_test WHERE name='string generated from TestDelete 2'"); err != nil {
		log.Fatal(err)
	}

	// 1 delete
	select {
	case model := <-deleteChannel:
		log.Infof("[TestDelete] - Received delete event: %v\n", model)
		c.Assert(model, NotNil)
		c.Assert(model.Umi, Equals, 16777215)
	}
}

// Merge statement - basically will trigger an update event for the dupplicated rows
func (s *wrapperTestSuite) TestInsert_OnDuplicateUpdate(c *C) {

	// execute valid SQL statements here
	if _, err := s.execute("INSERT INTO test.wrapper_test (id,name) VALUES (?,?)", 1, "string generated from TestInsert_OnDup 1"); err != nil {
		log.Fatal(err)
	}
	// discard all inserts to unblock channel
	go func() {
		<-insertChannel
	}()

	if _, err := s.execute("INSERT INTO test.wrapper_test (id,name) VALUES (?,?) on duplicate key update id = id + 1, name = 'insert dup test'",
		1, "string generated from TestInsert_OnDup 2"); err != nil {
		log.Fatal(err)
	} // replace first row with different id

	// 1 update
	select {
	case model := <-updateChannel:
		log.Infof("[TestInsert_OnDup] - Received update event: %v\n", model)
		c.Assert(model, NotNil)
		c.Assert(model.beforeUpdate.ID+1, Equals, model.afterUpdate.ID)
		c.Assert(model.beforeUpdate.Name, Equals, "string generated from TestInsert_OnDup 1")
		c.Assert(model.afterUpdate.Name, Equals, "insert dup test")
	}
}

// ---------------------------NULL CHECK---------------------------

// https://github.com/go-check/check/issues/12

type wrapperNULLTestSuite struct {
	*EventHandlerWrapper
}

var insertNilChannel = make(chan wrapperTestNillable)

type wrapperNULLTestHandler struct{}

func (*wrapperNULLTestHandler) OnInsert(schemaName string, tableName string, rec interface{}) {
	log.Infof("Inserting on table %s.%s", schemaName, tableName)
	insertNilChannel <- rec.(wrapperTestNillable)
}
func (*wrapperNULLTestHandler) OnUpdate(schemaName string, tableName string, oldRec interface{}, newRec interface{}) {
}
func (*wrapperNULLTestHandler) OnDelete(schemaName string, tableName string, rec interface{}) {}

var _ = SerialSuites(&wrapperNULLTestSuite{
	NewEventWrapper(
		map[string]interface{}{"wrapper_test_nillable": &wrapperTestNillable{}},
		Config{
			ServerID:          1,
			Addr:              "127.0.0.1:3306",
			User:              "root",
			Password:          "root",
			IncludeTableRegex: []string{"test\\.wrapper_test_nillable"},
			UseDecimal:        true,
		},
		&wrapperNULLTestHandler{},
	),
})

func (s *wrapperNULLTestSuite) execute(query string, args ...interface{}) (*mysql.Result, error) {
	return s.baseHandler.canal.Execute(query, args...)
}

func (s *wrapperNULLTestSuite) SetUpSuite(c *C) {
	fmt.Println("--- WRAPPER NULL : SetUpSuite ---")
	// start listening for callback, wait a second to make sure it is started up
	go s.StartBinlogListener()
	<-time.After(time.Second)

	s.execute("SET GLOBAL binlog_format = 'ROW'")
}

func (s *wrapperNULLTestSuite) TearDownSuite(c *C) {
	// To test the heartbeat and read timeout,so need to sleep 1 seconds without data transmission
	c.Logf("Start testing the heartbeat and read timeout")
	time.Sleep(time.Second)

	s.Close()
	// s.baseHandler.canal = nil
}

type wrapperTestNillable struct {
	ID    int        `gorm:"column:id"`
	Name  *string    `gorm:"column:name"`
	Bo    *bool      `gorm:"column:bo"`
	Mi    *int       `gorm:"column:mi"`
	Umi   *int       `gorm:"column:umi"`
	Si    *int       `gorm:"column:si"`
	SiU   *int       `gorm:"column:si_u"`
	Ti    *int       `gorm:"column:ti"`
	TiU   *int       `gorm:"column:ti_u"`
	Bi    *int       `gorm:"column:bi"`     // BIGINT maps to int (alias for int64 in amd-64)
	BiU   *uint      `gorm:"column:bi_u"`   // UNSIGNED BIGINT maps to uint
	De    *d.Decimal `gorm:"column:de"`     // DECIMAL maps to Decimal (with config useDecimal = true)
	Fl    *float32   `gorm:"column:fl"`     // FLOAT maps to float32
	Do    *float64   `gorm:"column:do"`     // DOUBLE maps to float64
	Bit   *int       `gorm:"column:bit"`    // BIT maps to int64 (as max length in MySQL BIT is 64)
	DTime *time.Time `gorm:"column:dtime"`  // DATETIME maps to time.Time
	TStam *time.Time `gorm:"column:tstamp"` // TIMESTAMP maps to time.Time
	Date  *time.Time `gorm:"column:date"`   // DATE maps to time.Time
	Time  *string    `gorm:"column:time"`   // *** TIME maps to string ***
	Set   *[]string  `gorm:"column:st"`     // SET maps to an array of strings
	Enum  *string    `gorm:"column:enum"`   // ENUM maps to a string
	Blob  *[]byte    `gorm:"column:blb"`    // BLOB maps to []byte
}

// Executed before every tests
func (s *wrapperNULLTestSuite) SetUpTest(c *C) {
	s.execute("DROP TABLE IF EXISTS test.wrapper_test_nillable")
	if _, err := s.execute(`
	CREATE TABLE IF NOT EXISTS test.wrapper_test_nillable (
		id 		int 			AUTO_INCREMENT,
		name 	varchar(100),
		bo		tinyint,
		mi 		mediumint,
		umi 	mediumint UNSIGNED,
		si		smallint,
		si_u	smallint UNSIGNED,
		ti		tinyint,
		ti_u	tinyint UNSIGNED,
		bi		bigint,
		bi_u	bigint UNSIGNED,
		de		decimal(60,5),
		fl		float,
		do		double,
		bit		bit(64),
		dtime	datetime,
		tstamp	timestamp,
		date	date,
		time	time,
		st		set('x','y','z','t'),
		enum	enum('a','b'),			
		blb 	blob,	
		PRIMARY KEY(id)
	) ENGINE=innodb;
	`); err != nil {
		log.Errorln(err)
	}
}

func (s *wrapperNULLTestSuite) TestReadingNULLs(c *C) {

	// execute valid SQL statements here
	if _, err := s.execute(`INSERT INTO test.wrapper_test_nillable (bo,mi,umi,si,si_u,ti,ti_u,bi,bi_u,de,fl,do,bit,dtime,tstamp,date,time,st,enum,blb) 
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, //bi_u
		nil, nil, nil, nil, //bit
		nil, nil, nil, //date
		nil,      //time
		nil, nil, //enum
		nil,
	); err != nil {
		log.Errorln(err)
	}

	select {
	case model := <-insertNilChannel:
		log.Infof("[TestInsert] - Received insert event: %v\n", model)
		c.Assert(model.ID, Equals, 1)
		c.Assert(model.Name, IsNil)
		c.Assert(model.Bo, IsNil)
		c.Assert(model.Mi, IsNil)
		c.Assert(model.Umi, IsNil)
		c.Assert(model.Si, IsNil)
		c.Assert(model.SiU, IsNil)
		c.Assert(model.Ti, IsNil)
		c.Assert(model.TiU, IsNil)
		c.Assert(model.Bi, IsNil)
		c.Assert(model.BiU, IsNil)
		c.Assert(model.De, IsNil)
		c.Assert(model.Fl, IsNil)
		c.Assert(model.Do, IsNil)
		c.Assert(model.Bit, IsNil)
		c.Assert(model.DTime, IsNil)
		c.Assert(model.TStam, IsNil)
		c.Assert(model.Date, IsNil)
		c.Assert(model.Time, IsNil)
		c.Assert(model.Set, IsNil)
		c.Assert(model.Enum, IsNil)
		c.Assert(model.Blob, IsNil)
	}
}
