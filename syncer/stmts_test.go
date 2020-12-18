package syncer

import (
	"reflect"
	"testing"
)

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
