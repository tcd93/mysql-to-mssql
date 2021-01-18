module mysql2mssql

go 1.14

require (
	github.com/dave/jennifer v1.4.1
	github.com/denisenkom/go-mssqldb v0.9.0
	github.com/go-playground/universal-translator v0.17.0 // indirect
	github.com/go-playground/validator v9.31.0+incompatible // indirect
	github.com/go-playground/validator/v10 v10.4.1
	github.com/json-iterator/go v1.1.10
	github.com/labstack/echo/v4 v4.1.17
	github.com/labstack/gommon v0.3.0
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/ompluscator/dynamic-struct v1.2.0
	github.com/pingcap/check v0.0.0-20200212061837-5e12011dc712
	github.com/shopspring/decimal v1.2.0
	github.com/siddontang/go v0.0.0-20180604090527-bdc77568d726
	github.com/siddontang/go-log v0.0.0-20190221022429-1e957dd83bed
	github.com/siddontang/go-mysql v1.1.0
	github.com/stretchr/testify v1.6.1
	github.com/xujiajun/nutsdb v0.5.0
	golang.org/x/net v0.0.0-20201224014010-6772e930b67b
	golang.org/x/sys v0.0.0-20210105210732-16f7687f5001 // indirect
)

replace github.com/xujiajun/nutsdb => github.com/tcd93/nutsdb v0.5.1
