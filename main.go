package main

import (
	"mysql2mssql/db"
	"mysql2mssql/server"
)

const (
	// local directory to store messages
	localDir = "D:/temp/nutsdb"
)

func main() {
	server.NewServer(db.Options{
		Dir:         localDir,
		SegmentSize: 1024 * 1024, // 1mb
	}).StartServer("")
}
