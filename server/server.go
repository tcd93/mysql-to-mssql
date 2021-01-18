// Package server handle request validation & parsing, then passes to package API for processing
package server

import (
	db "gonnextor/db"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// Server can be embedded and used in a desktop application
type Server struct {
	*handler
}

// NewServer creates new instance of Server, default Log Store storage is "nutsdb"
func NewServer(dbConfig db.Options) *Server {
	return &Server{newHandler("nutsdb", &dbConfig)}
}

// StartServer starts listening for request, default address localhost:1323
func (s *Server) StartServer(address string) {
	if address == "" {
		address = ":1323"
	}

	e := echo.New()
	e.Validator = s.handler.validator
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// add/edit datamodels
	structGroup := e.Group("/struct")
	structGroup.GET("/get", s.getStruct)
	structGroup.GET("/get/:name", s.getStruct)
	structGroup.POST("/put", s.putStruct)

	parserGroup := e.Group("/parser")
	parserGroup.POST("/start", s.startParser)
	parserGroup.POST("/stop", s.stopParser)
	parserGroup.GET("/stream", s.streamStdout) // websocket

	syncerGroup := e.Group("/syncer")
	syncerGroup.POST("/start", s.startSyncer)
	syncerGroup.POST("/stop", s.stopSyncer)

	e.Logger.Fatal(e.Start(address))
}
