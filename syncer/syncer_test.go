package syncer

import (
	"reflect"
	"testing"
	"time"

	dcm "github.com/shopspring/decimal"
)

// Integration tests of basic operations to MSSQL DB
// Requirement:
// 	- MSSQL (2008+) on localhost with SSO enabled
//  - Enable TCP/IP in Configuration Manager
// 	- Set up database named "gonnextor"
//	- Create table [SyncerTest] as follow:

// CREATE TABLE [dbo].[SyncerTest](
// 	[id] int primary key,
// 	[name] nvarchar(50) NULL,
// 	[bo] bit NULL,
// 	[bi] bigint NULL,
// 	[bi_u] decimal(21, 0) NULL,
// 	[de] decimal(38, 5) NULL,
// 	[fl] smallmoney NULL,
// 	[do] float NULL,
// 	[bit] bigint NULL,
// 	[dtime] datetime NULL,
// 	[date] date NULL,
// 	[time] time(7) NULL,
// 	[blb] binary(1000) NULL,
// 	[bnr] varbinary(max) NULL,
//  [user] varchar(max) default system_user
// )

type syncerTest struct {
	ID     int         `gorm:"column:id;primaryKey"`   // BIGINT
	Name   string      `gorm:"column:name;primaryKey"` // CHAR/VARCHAR/NVARCHAR
	Bo     bool        `gorm:"column:bo"`              // BIT
	Bi     int         `gorm:"column:bi"`              // BIGINT, works with INT too, but INT is only 4 bytes in MSSQL, so overflow may be happens
	BiU    *uint       `gorm:"column:bi_u"`            // DECIMAL (custom convertion - performance penalty)
	De     dcm.Decimal `gorm:"column:de"`              // DECIMAL
	Fl     float32     `gorm:"column:fl"`              // SMALLMONEY
	Do     float64     `gorm:"column:do"`              // FLOAT
	Bit    int         `gorm:"column:bit"`             // BIGINT
	DTime  time.Time   `gorm:"column:dtime"`           // DATETIME
	Date   *time.Time  `gorm:"column:date"`            // DATE
	Time   *string     `gorm:"column:time"`            // TIME(7)
	Blob   []byte      `gorm:"column:blb"`             // BINARY/VARBINARY
	Binary *[]byte     `gorm:"column:bnr"`
}

func setUp(syncer *Syncer) {
	if _, err := syncer.truncate("SyncerTest"); err != nil {
		panic(err)
	}
}

func tearDown(syncer *Syncer) {
	syncer.Close()
}

func TestGetColumns(t *testing.T) {

	model := &syncerTest{
		ID:   1,
		Name: "中文 English Tiếng Việt",
		BiU:  nil,
	}
	expected := []column{
		{"id", true, "int"},
		{"name", true, "string"},
	}
	actual, _ := getColumns(model, true)

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Expected: \n\n%#v\n\n Actual: \n\n%#v\n\n", expected, actual)
	}
}

func TestGenerateInsertStatement(t *testing.T) {
	model := &syncerTest{
		ID:   1,
		Name: "中文 English Tiếng Việt",
	}
	cols, _ := getColumns(model, false)
	upt := buildInsertStatement("testtable", cols)
	if upt != "insert into testtable (id,name,bo,bi,bi_u,de,fl,do,bit,dtime,date,time,blb,bnr) values (?,?,?,?,?,?,?,?,?,?,?,?,?,CONVERT(VARBINARY(MAX),?))" {
		t.Errorf("Expected: \n\n%s\n\n Actual: \n\n%s\n\n", "insert into testtable (id,name,bo,bi,bi_u,de,fl,do,bit,dtime,date,time,blb,bnr) values (?,?,?,?,?,?,?,?,?,?,?,?,?,CONVERT(VARBINARY(MAX),?))", upt)
	}
}

func TestGenerateUpdateStatement(t *testing.T) {
	model := &syncerTest{
		ID:   1,
		Name: "中文 English Tiếng Việt",
	}
	cols, _ := getColumns(model, false)
	expected := "update testtable set id=?,name=?,bo=?,bi=?,bi_u=?,de=?,fl=?,do=?,bit=?,dtime=?,date=?,time=?,blb=?,bnr=CONVERT(VARBINARY(MAX),?) where id = 1"
	actual := buildUpdateStatement("testtable", cols, "id = 1")
	if actual != expected {
		t.Errorf("Expected: \n\n%s\n\n Actual: \n\n%s\n\n", expected, actual)
	}
}

func TestGenerateUpdateStatementUsingPK(t *testing.T) {

	model := &syncerTest{
		ID:   1,
		Name: "中文 English Tiếng Việt",
		BiU:  nil,
	}
	cols, _ := getColumns(model, false)
	expected := "update testtable set id=?,name=?,bo=?,bi=?,bi_u=?,de=?,fl=?,do=?,bit=?,dtime=?,date=?,time=?,blb=?,bnr=CONVERT(VARBINARY(MAX),?) where id=? AND name=?"
	actual := buildUpdateStatement("testtable", cols, "") // set 'where' empty
	if actual != expected {
		t.Errorf("Expected: \n\n%s\n\n Actual: \n\n%s\n\n", expected, actual)
	}
}

func TestSyncerInsertTable(t *testing.T) {
	dec, _ := dcm.NewFromString("11112345111899999999874444444313.11198")
	dtime, _ := time.Parse("2006-01-02 15:04:05", "2020-01-01 10:10:10")
	date, _ := time.Parse("2006-01-02", "1999-12-01")
	unsigned := uint(18446744073709551615)
	varbin := []byte("varbinary stuff")
	time := "23:59:59"

	model := &syncerTest{
		ID:     1,
		Name:   "中文 English Tiếng Việt",
		Bo:     true,
		Bi:     9223372036854775807,
		BiU:    &unsigned,
		De:     dec,
		Fl:     12.3457,
		Do:     56.789123456,
		Bit:    9223372036854775807,
		DTime:  dtime,
		Date:   &date,
		Time:   &time,
		Blob:   []byte("binary stuff"),
		Binary: &varbin,
	}
	cfg := TargetDbConfig{
		Server:   "127.0.0.1",
		Database: "gonnextor",
		Log:      63,
	}
	syncer := NewSyncer(cfg)
	setUp(syncer)
	defer tearDown(syncer)

	res, err := syncer.Insert("SyncerTest", model)
	if err != nil {
		t.Errorf("Exec failed: %v\n", err.Error())
	} else {
		rowsAffected, err := res.RowsAffected()
		if err != nil {
			t.Errorf("RowsAffected() failed: %v\n", err.Error())
		}
		if rowsAffected == 0 || rowsAffected != 1 {
			t.Error("rowsAffected is not 1")
		}
	}
}

func TestSyncerUpdateTable(t *testing.T) {

	cfg := TargetDbConfig{
		Server:   "127.0.0.1",
		Database: "gonnextor",
		Log:      63,
	}
	syncer := NewSyncer(cfg)
	setUp(syncer)
	defer tearDown(syncer)

	TestSyncerInsertTable(t)

	dtime, _ := time.Parse("2006-01-02 15:04:05", "2020-01-01 10:10:10")
	unsigned := uint(18446744073709551615)
	model := &syncerTest{
		ID:    1,
		Name:  "base de données",
		DTime: dtime,
		BiU:   &unsigned,
	}
	res, err := syncer.Update("SyncerTest", model, "id = 1")
	if err != nil {
		t.Errorf("Exec failed: %v\n", err.Error())
	} else {
		rowsAffected, err := res.RowsAffected()
		if err != nil {
			t.Errorf("RowsAffected() failed: %v\n", err.Error())
		}
		if rowsAffected == 0 || rowsAffected != 1 {
			t.Error("rowsAffected is not 1")
		}
	}
}

func TestSyncerUpdateTableWithPK(t *testing.T) {

	cfg := TargetDbConfig{
		Server:   "127.0.0.1",
		Database: "gonnextor",
		Log:      63,
	}
	syncer := NewSyncer(cfg)
	setUp(syncer)
	defer tearDown(syncer)

	TestSyncerInsertTable(t)

	dec, _ := dcm.NewFromString("11112345111899999999874444444313.11198")
	dtime, _ := time.Parse("2006-01-02 15:04:05", "2020-01-01 10:10:10")
	date, _ := time.Parse("2006-01-02", "1999-12-01")
	unsigned := uint(18446744073709551615)
	varbin := []byte("varbinary stuff")
	tm := "23:59:59"

	oldModel := &syncerTest{
		ID:     1,
		Name:   "中文 English Tiếng Việt",
		Bo:     true,
		Bi:     9223372036854775807,
		BiU:    &unsigned,
		De:     dec,
		Fl:     12.3457,
		Do:     56.789123456,
		Bit:    9223372036854775807,
		DTime:  dtime,
		Date:   &date,
		Time:   &tm,
		Blob:   []byte("binary stuff"),
		Binary: &varbin,
	}

	newdtime, _ := time.Parse("2006-01-02 15:04:05", "2020-12-31 12:00:00")
	newunsigned := uint(18446744073709551615)
	newvarbin := []byte("updated field - varbinary stuff")
	newModel := &syncerTest{
		ID:     1,
		Name:   "中文 English Tiếng Việt 2",
		DTime:  newdtime,
		BiU:    &newunsigned,
		Binary: &newvarbin,
	}
	res, err := syncer.UpdateOnPK("SyncerTest", oldModel, newModel)
	if err != nil {
		t.Errorf("Exec failed: %v\n", err.Error())
	} else {
		rowsAffected, err := res.RowsAffected()
		if err != nil {
			t.Errorf("RowsAffected() failed: %v\n", err.Error())
		}
		if rowsAffected == 0 || rowsAffected != 1 {
			t.Error("rowsAffected is not 1")
		}
	}
}

func TestSyncerDeleteTable(t *testing.T) {

	cfg := TargetDbConfig{
		Server:   "127.0.0.1",
		Database: "gonnextor",
		Log:      63,
	}
	syncer := NewSyncer(cfg)
	setUp(syncer)
	defer tearDown(syncer)

	TestSyncerInsertTable(t)

	res, err := syncer.Delete("SyncerTest", "id = 1")
	if err != nil {
		t.Errorf("Exec failed: %v\n", err.Error())
	} else {
		rowsAffected, err := res.RowsAffected()
		if err != nil {
			t.Errorf("RowsAffected() failed: %v\n", err.Error())
		}
		if rowsAffected == 0 || rowsAffected != 1 {
			t.Error("rowsAffected is not 1")
		}
	}
}

// test delete on non-matching pks
func TestSyncerDeleteTableWithPK(t *testing.T) {

	cfg := TargetDbConfig{
		Server:   "127.0.0.1",
		Database: "gonnextor",
		Log:      63,
	}
	syncer := NewSyncer(cfg)
	setUp(syncer)
	defer tearDown(syncer)

	TestSyncerInsertTable(t)

	model := &syncerTest{
		ID:   1,
		Name: "中文 English Tiếng Việt 2", // different key
	}

	res, err := syncer.DeleteOnPK("SyncerTest", model)
	if err != nil {
		t.Errorf("Exec failed: %v\n", err.Error())
	} else {
		rowsAffected, err := res.RowsAffected()
		if err != nil {
			t.Errorf("RowsAffected() failed: %v\n", err.Error())
		}
		if rowsAffected != 0 {
			t.Error("rowsAffected is not 0")
		}
	}
}

type syncerPointerTest struct {
	ID     int          `gorm:"column:id"`
	Name   *string      `gorm:"column:name"`
	Bo     *bool        `gorm:"column:bo"`
	Bi     *int         `gorm:"column:bi"`
	BiU    *uint        `gorm:"column:bi_u"`
	De     *dcm.Decimal `gorm:"column:de"`
	Fl     *float32     `gorm:"column:fl"`
	Do     *float64     `gorm:"column:do"`
	Bit    *int         `gorm:"column:bit"`
	DTime  *time.Time   `gorm:"column:dtime"`
	Date   *time.Time   `gorm:"column:date"`
	Time   *string      `gorm:"column:time"`
	Blob   *[]byte      `gorm:"column:blb"`
	Binary *[]byte      `gorm:"column:bnr"`
}

func TestSyncerInsertNullsToTable(t *testing.T) {
	model := &syncerPointerTest{
		ID: 1,
	}
	cfg := TargetDbConfig{
		Server:   "127.0.0.1",
		Database: "gonnextor",
		Log:      63,
	}
	syncer := NewSyncer(cfg)
	setUp(syncer)
	defer tearDown(syncer)

	res, err := syncer.Insert("SyncerTest", model)
	if err != nil {
		t.Errorf("Exec failed: %v\n", err.Error())
	} else {
		rowsAffected, err := res.RowsAffected()
		if err != nil {
			t.Errorf("RowsAffected() failed: %v\n", err.Error())
		}
		if rowsAffected == 0 || rowsAffected != 1 {
			t.Error("rowsAffected is not 1")
		}
	}
}
