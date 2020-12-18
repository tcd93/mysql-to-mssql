package parser

// Test receving NULLs from DB, to map MySQL's NULL to Golang' Nil, define data model fields in pointer types
// Requirement:
// 	- MySQL on localhost with user: root/root
// 	- set up schema named "test"

import (
	"fmt"
	"time"

	. "github.com/pingcap/check"
	d "github.com/shopspring/decimal"
	"github.com/siddontang/go-log/log"
	"github.com/siddontang/go-mysql/mysql"
)

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
