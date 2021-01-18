package syncer

import (
	"database/sql"
	"fmt"
	"strings"

	mssql "github.com/denisenkom/go-mssqldb" //driver for MSSQL, to work with "database/sql" package
)

// TargetDbConfig object for mssql syncer, see package server.StartParserRequest
type TargetDbConfig struct {
	Server   string
	Database string
	Userid   string
	Password string
	Log      uint8
	Encrypt  string
	Appname  string
}

// Syncer wrapper, uses go-mssqldb underneath
type Syncer struct {
	store          *Store
	interval       int64
	cfg            TargetDbConfig
	db             *sql.DB
	insertStmts    map[string]*sql.Stmt
	updateStmts    map[string]*sql.Stmt
	deleteStmts    map[string]*sql.Stmt
	syncQuitSignal chan struct{}
}

// Insert a single row to `targetTable`
func (s *Syncer) Insert(targetTable string, model interface{}) (sql.Result, error) {
	cols, newVals := getColumns(model, false)

	if s.insertStmts[targetTable] == nil {
		stmt, err := s.db.Prepare(buildInsertStatement(targetTable, cols))
		if err != nil {
			return nil, err
		}
		s.insertStmts[targetTable] = stmt
	}

	return s.insertStmts[targetTable].Exec(newVals...)
}

// Update a single row to `targetTable`.
// `where` specify the string to append to update statement
// followed by the condition parameters.
// Example:
// 	Update("table_name", model, "id = ? AND name = ?", 1, "username")
func (s *Syncer) Update(targetTable string, model interface{}, where string, conditions ...interface{}) (sql.Result, error) {
	cols, newVals := getColumns(model, false)

	if s.updateStmts[targetTable] == nil {
		stmt, err := s.db.Prepare(buildUpdateStatement(targetTable, cols, where))
		if err != nil {
			return nil, err
		}
		s.updateStmts[targetTable] = stmt
	}

	return s.updateStmts[targetTable].Exec(append(newVals, conditions...)...)
}

// UpdateOnPK updates a single row to `targetTable` based on `primaryKey` tag defined on model struct.
// Expects `newModel` & `oldModel` are of same struct type
// Example:
// 	UpdateOnPK("table_name", oldModel, newModel)
func (s *Syncer) UpdateOnPK(targetTable string, oldModel interface{}, newModel interface{}) (sql.Result, error) {
	cols, newVals := getColumns(newModel, false)

	if s.updateStmts[targetTable] == nil {
		// since data structure of oldModel & newModel is the same
		// so the result of `buildUpdateStatement` is indifferent of the new or old model we pass in
		stmt, err := s.db.Prepare(buildUpdateStatement(targetTable, cols, ""))
		if err != nil {
			return nil, err
		}
		s.updateStmts[targetTable] = stmt
	}

	// get the values of primary columns to map to "where" part in statement
	_, pks := getColumns(oldModel, true)
	if len(pks) == 0 {
		return nil, fmt.Errorf("primaryKey tag not defined in model of %v", targetTable)
	}

	return s.updateStmts[targetTable].Exec(append(newVals, pks...)...)
}

// Delete a single row from `targetTable`.
// `where` specify the string to append to update statement
// followed by the condition parameters.
// Example:
// 	Delete("table_name", "id = ? AND name = ?", 1, "username")
func (s *Syncer) Delete(targetTable string, where string, conditions ...interface{}) (sql.Result, error) {
	if s.deleteStmts[targetTable] == nil {
		stmt, err := s.db.Prepare(buildDeleteStatement(targetTable, where))
		if err != nil {
			return nil, err
		}
		s.deleteStmts[targetTable] = stmt
	}
	return s.deleteStmts[targetTable].Exec(conditions...)
}

// DeleteOnPK deletes a single row from `targetTable` based on `primaryKey` tag defined on model struct.
// Example:
// 	DeleteOnPK("table_name", model)
func (s *Syncer) DeleteOnPK(targetTable string, model interface{}) (sql.Result, error) {
	cols, pks := getColumns(model, true)
	if len(pks) == 0 {
		return nil, fmt.Errorf("primaryKey tag not defined in model of %v", targetTable)
	}

	if s.deleteStmts[targetTable] == nil {
		stmt, err := s.db.Prepare(buildDeleteStatementFromPK(targetTable, cols))
		if err != nil {
			return nil, err
		}
		s.deleteStmts[targetTable] = stmt
	}
	return s.deleteStmts[targetTable].Exec(pks...)
}

// Close connection pool
func (s *Syncer) Close() {
	for _, stmts := range s.insertStmts {
		stmts.Close()
	}
	s.db.Close()
}

// SetLogger set custom logger for database driver
func (s *Syncer) SetLogger(logger interface{}) {
	mssql.SetLogger(logger.(mssql.Logger))
}

// NewSyncer returns new instance of Syncer, should be called only once.
// intv is the interval frequency (second) between each log store scan
func NewSyncer(cfg TargetDbConfig, intv int64, s *Store) *Syncer {
	conn := buildConn(cfg)
	return &Syncer{
		store:       s,
		interval:    intv,
		db:          conn,
		cfg:         cfg,
		insertStmts: make(map[string]*sql.Stmt, 0),
		updateStmts: make(map[string]*sql.Stmt, 0),
		deleteStmts: make(map[string]*sql.Stmt, 0),
	}
}

///////////////////////////////private methods////////////////////////////////

func buildConn(cfg TargetDbConfig) (conn *sql.DB) {
	var connectStringBuilder strings.Builder
	connectStringBuilder.Grow(50)
	if cfg.Server != "" {
		fmt.Fprintf(&connectStringBuilder, "server=%s;", cfg.Server)
	}
	if cfg.Database != "" {
		fmt.Fprintf(&connectStringBuilder, "database=%s;", cfg.Database)
	}
	if cfg.Userid != "" {
		fmt.Fprintf(&connectStringBuilder, "user id=%s;password=%s;", cfg.Userid, cfg.Password)
	}
	fmt.Fprintf(&connectStringBuilder, "log=%v;", cfg.Log)
	if cfg.Encrypt != "" {
		fmt.Fprintf(&connectStringBuilder, "encrypt=%s;", cfg.Encrypt)
	}
	if cfg.Appname != "" {
		fmt.Fprintf(&connectStringBuilder, "app name=%s;", cfg.Appname)
	}
	fmt.Printf("connection string: %s\n", connectStringBuilder.String())
	conn, err := sql.Open("mssql", connectStringBuilder.String())
	if err != nil {
		panic(fmt.Sprintf("Open connection failed: %v", err.Error()))
	}
	return
}

func (s *Syncer) truncate(targetTable string) (sql.Result, error) {
	return s.db.Exec(fmt.Sprintf("truncate table %s", targetTable))
}

func buildInsertStatement(targetTable string, columns []column) string {
	length := len(columns)
	var sBuilder strings.Builder
	sBuilder.Grow(length * 10)

	fmt.Fprintf(&sBuilder, "insert into %s (", targetTable)
	for i, c := range columns {
		fmt.Fprint(&sBuilder, c.name)
		if i < length-1 {
			sBuilder.WriteByte(44) // append comma ","
		} else {
			sBuilder.WriteByte(41) // append closing bracket ")"
		}
	}

	sBuilder.WriteString(" values (")
	for i, c := range columns {
		if c.fieldType == "*[]uint8" {
			// https://github.com/denisenkom/go-mssqldb/issues/530
			fmt.Fprint(&sBuilder, "CONVERT(VARBINARY(MAX),?)")
		} else {
			fmt.Fprint(&sBuilder, "?")
		}
		if i < length-1 {
			sBuilder.WriteByte(44) // append comma ","
		} else {
			sBuilder.WriteByte(41) // append closing bracket ")"
		}
	}

	return sBuilder.String()
}

func buildUpdateStatement(targetTable string, columns []column, where string) string {
	length := len(columns)
	var sBuilder strings.Builder
	sBuilder.Grow(length * 20)

	fmt.Fprintf(&sBuilder, "update %s set ", targetTable)
	for i, c := range columns {
		if c.fieldType == "*[]uint8" {
			// https://github.com/denisenkom/go-mssqldb/issues/530
			fmt.Fprintf(&sBuilder, "%s=CONVERT(VARBINARY(MAX),?)", c.name)
		} else {
			fmt.Fprintf(&sBuilder, "%s=?", c.name)
		}
		if i < length-1 {
			sBuilder.WriteByte(44) // append comma ","
		}
	}

	if where != "" {
		fmt.Fprintf(&sBuilder, " where %s", where)
	} else if len(columns) > 0 { // update based on PK columns
		fmt.Fprintf(&sBuilder, " where %s", where)
		for _, c := range columns {
			if c.isPrimaryKey {
				fmt.Fprintf(&sBuilder, "%s=? AND ", c.name)
			}
		}
	}

	return strings.TrimRight(sBuilder.String(), " AND ")
}

func buildDeleteStatement(targetTable string, where string) string {
	var sBuilder strings.Builder
	sBuilder.Grow(13 + len(targetTable) + len(where))

	fmt.Fprintf(&sBuilder, "delete from %s", targetTable)
	if where != "" {
		fmt.Fprintf(&sBuilder, " where %s", where)
	}

	return sBuilder.String()
}

func buildDeleteStatementFromPK(targetTable string, columns []column) string {
	var sBuilder strings.Builder
	sBuilder.Grow(13 + len(targetTable) + 30)

	fmt.Fprintf(&sBuilder, "delete from %s", targetTable)
	if len(columns) > 0 {
		fmt.Fprint(&sBuilder, " where ")
		for _, c := range columns {
			if c.isPrimaryKey {
				fmt.Fprintf(&sBuilder, "%s=? AND ", c.name)
			}
		}
	}

	return strings.TrimRight(sBuilder.String(), " AND ")
}
