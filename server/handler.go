package server

import (
	"mysql2mssql/db"
	"mysql2mssql/mysql/parser"
	"mysql2mssql/server/API"
	"mysql2mssql/server/param"
	"net/http"

	validator "github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"golang.org/x/net/websocket"
)

type (
	customValidator struct {
		validator *validator.Validate
	}
)

func (cv *customValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

type handler struct {
	*API.API
	validator *customValidator
}

// create new handler for the Server that manages storage type, data models & request validations
func newHandler(dbType string, dbConfig *db.Options) *handler {
	var dbEngine db.Interface
	if dbType == "nutsdb" || dbType == "" {
		dbEngine = db.UseNutsDB(*dbConfig)
	} else {
		dbEngine = db.UseInmemDB()
	}

	h := &handler{
		&API.API{
			DataModels:  &parser.ModelMap{},
			DBInterface: dbEngine,
		},
		&customValidator{validator.New()},
	}
	if err := h.LoadDataModels(); err != nil {
		panic(err)
	}
	return h
}

func (h *handler) putStruct(c echo.Context) (err error) {
	p := &param.StructRequest{}
	if err = c.Bind(p); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err = c.Validate(p); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	strct, err := h.Put(*p)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, strct)
}

func (h *handler) getStruct(c echo.Context) (err error) {
	tabName := c.Param("name")
	if tabName == "" {
		return c.JSON(http.StatusOK, h.DataModels)
	}
	return c.JSON(http.StatusOK, h.Get(tabName))
}

func (h *handler) startParser(c echo.Context) (err error) {
	defer func() {
		e := recover()
		if e != nil {
			err = echo.NewHTTPError(http.StatusInternalServerError, e)
		}
	}()

	p := &param.StartParserRequest{}
	if err = c.Bind(p); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "bind: "+err.Error())
	}
	if err = c.Validate(p); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "validate: "+err.Error())
	}

	h.StartParser(*p)

	return c.String(http.StatusAccepted, "OK")
}

func (h *handler) stopParser(c echo.Context) (err error) {
	if err = h.StopParser(); err != nil {
		return echo.NewHTTPError(http.StatusConflict, "Parser is closed, or has not been started")
	}
	return c.String(http.StatusOK, "OK")
}

// streams the log contents to client using Websocket
func (h *handler) streamStdout(c echo.Context) (err error) {
	strm := make(chan string)
	q := make(chan struct{})
	go h.LogChan(strm, q)
	defer func() { q <- struct{}{} }()

	websocket.Handler(func(ws *websocket.Conn) {
		defer ws.Close()
		for {
			select {
			case line := <-strm:
				websocket.Message.Send(ws, line)
			}
		}
	}).ServeHTTP(c.Response(), c.Request())
	return nil
}

func (h *handler) startSyncer(c echo.Context) (err error) {
	defer func() {
		e := recover()
		if e != nil {
			err = echo.NewHTTPError(http.StatusInternalServerError, e)
		}
	}()

	p := &param.StartSyncerRequest{}
	if err = c.Bind(p); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "bind: "+err.Error())
	}
	if err = c.Validate(p); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "validate: "+err.Error())
	}

	h.StartSyncer(*p)

	return c.String(http.StatusAccepted, "OK")
}

func (h *handler) stopSyncer(c echo.Context) (err error) {
	if err = h.StopSyncer(); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.String(http.StatusOK, "OK")
}
