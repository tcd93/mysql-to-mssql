package param

import "mysql2mssql/db"

type (
	// StructRequest is the request to add/edit "Datamodels", which represent the table structure in source / target DBs
	StructRequest struct {
		Table   string   `json:"table" validate:"required"`
		Columns []Column `json:"columns" validate:"required,dive"`
	}
	// Column metadata for a column in a table
	Column struct {
		Name      string       `json:"name" validate:"required"`
		Type      db.MySQLType `json:"type" validate:"required,numeric,lte=20"`
		IsPrimary bool         `json:"is_primary,omitempty"`
	}
	// StartParserRequest is the request for starting the sourceDB Parser
	// & log changes to an embedded Log Store (defaults to "nutsdb")
	//
	// IncludeTableRegex or ExcludeTableRegex should contain database name,
	// only a table which matches IncludeTableRegex and dismatches ExcludeTableRegex will be processed.
	//
	// Example: IncludeTableRegex = [".*\\.canal"], ExcludeTableRegex : ["mysql\\..*"].
	// This will include all database's 'canal' table, except database 'mysql'
	//
	// UseDecimal: When set to true, go-mysql will use Decimal package for decimal types
	StartParserRequest struct {
		ServerID          uint32   `json:"server_id" validate:"required,numeric"`
		Addr              string   `json:"addr" validate:"required,hostname_port"`
		User              string   `json:"user" validate:"required,alphanum"`
		Password          string   `json:"password" validate:"required,alphanum"`
		IncludeTableRegex []string `json:"include_table_regex,omitempty"`
		ExcludeTableRegex []string `json:"exclude_table_regex,omitempty"`
		UseDecimal        bool     `json:"use_decimal" validate:"required"`
		Charset           string   `json:"charset,omitempty"`
		TLSConfig         struct {
			ServerName string `json:"server_name,omitempty"`
			ServerCA   string `json:"server_ca,omitempty"`
			ClientCert string `json:"client_cert,omitempty"`
			ClientKey  string `json:"client_key,omitempty"`
		} `json:"tls_config,omitempty"`
	}
	// StartSyncerRequest for starting targetDB Syncer, there'll be a scheduled job to scan Log Store
	// for unsynced changes and perform changes immediately
	//
	// Interval: run scheduler every X second(s), note that if the job is still running when next Interval arrives, it'll be blocked
	//
	// Server: server IP (example "127.0.0.1")
	//
	// Userid: The SQL Server Authentication user id or the Windows Authentication user id in the DOMAIN\User format.
	// On Windows, if user id is empty or missing Single-Sign-On is used.
	//
	// Log: default 0/no logging, 63 for full logging
	// 	* 1 log errors
	// 	* 2 log messages
	// 	* 4 log rows affected
	// 	* 8 trace sql statements
	// 	* 16 log statement parameters
	// 	* 32 log transaction begin/end
	//
	// Encrypt:
	// 	* "disable" - Data send between client and server is not encrypted.
	// 	* "false" - Data sent between client and server is not encrypted beyond the login packet. (Default)
	// 	* "true" - Data sent between client and server is encrypted.
	//
	// Appname: the programe_name in dm_exec_sessions (default is go-mssqldb)
	StartSyncerRequest struct {
		Interval int64  `json:"interval,omitempty" validate:"numeric"`
		Server   string `json:"server" validate:"required,ip"`
		Database string `json:"database" validate:"required"`
		Userid   string `json:"user_id,omitempty"`
		Password string `json:"password,omitempty"`
		Log      uint8  `json:"log,omitempty" validate:"numeric"`
		Encrypt  string `json:"encrypt,omitempty"`
		Appname  string `json:"app_name,omitempty"`
	}
)
