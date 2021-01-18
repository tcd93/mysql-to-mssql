package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

var (
	requestJSON = `{
		"table":"Jon_Snow",
		"columns": [
			{
				"name": "col0",
				"type": 1,
				"is_primary": true
			},
			{
				"name": "Col1",
				"type": 9
			},
			{
				"name": "Col2",
				"type": 10
			},
			{
				"name": "Col3",
				"type": 17
			},
			{
				"name": "Col4",
				"type": 18
			}
		]
	}`
	strct = &struct {
		Col1 time.Time  `gorm:"column:Col1"`
		Col2 *time.Time `gorm:"column:Col2"`
		Col3 []byte     `gorm:"column:Col3"`
		Col4 *[]byte    `gorm:"column:Col4"`
	}{}
	binLogConfigJSON = fmt.Sprintf(`{
		"server_id": 1,
		"addr": "35.240.181.214:3306",
		"user": "root",
		"password": "root",
		"use_decimal": true,
		"include_table_regex": ["sakila\\.Staff"],
		"tls_config": {
			"server_name": "mysql-to-mssql-syncer:a1",
			"server_ca": "%s",
			"client_cert": "%s",
			"client_key": "%s"
		}
	}`, rootPem, clientCert, clientKey)
	storeConfigJSON = `{
		"server": "127.0.0.1",
		"database": "gonnextor",
		"log": 63,
		"appname": "Postman"
	}`
)

var rootPem = `-----BEGIN CERTIFICATE-----\nMIIDfzCCAmegAwIBAgIBADANBgkqhkiG9w0BAQsFADB3MS0wKwYDVQQuEyQ2MTY3\nOGNkNC00MGQ4LTQzMjEtODU2ZC03NThmMTg5ODI1NTcxIzAhBgNVBAMTGkdvb2ds\nZSBDbG91ZCBTUUwgU2VydmVyIENBMRQwEgYDVQQKEwtHb29nbGUsIEluYzELMAkG\nA1UEBhMCVVMwHhcNMjAxMjE4MTM1NjE3WhcNMzAxMjE2MTM1NzE3WjB3MS0wKwYD\nVQQuEyQ2MTY3OGNkNC00MGQ4LTQzMjEtODU2ZC03NThmMTg5ODI1NTcxIzAhBgNV\nBAMTGkdvb2dsZSBDbG91ZCBTUUwgU2VydmVyIENBMRQwEgYDVQQKEwtHb29nbGUs\nIEluYzELMAkGA1UEBhMCVVMwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIB\nAQCqKv9DsASx589YkeyVPqjsRsOIjMEVCfF906w3ZuaJE0pFueG2EzCto2S58RwD\n5XoPMetYJaDcyJryM26TNW9hkqngVZcu6O4QbhYGWMJ0abT1tFlpoi0zOpbfcNug\nNHxrCJURydTVmHVqzQhmIgUpBF7mpBftAoqphCaa7w99e3J5AxEv/6xRJpfZcfqA\netue0jH9jzrP0BKs5s9/JVISX6+pZQ8Z+NdQxV297yB01Um/v9ytdlzEsrKDnW9d\nFqcuX4+GxjmH2CK5HwNxqnMdw60X7vQ+gt+MBckYL7qDUY3ZALbwnn8Dt71Y2j5o\nZ2HF3t3MhuRy8L7HVnp/ZRNVAgMBAAGjFjAUMBIGA1UdEwEB/wQIMAYBAf8CAQAw\nDQYJKoZIhvcNAQELBQADggEBAEvDOxg+3kxWqODt13gGFXdC5IHf9vIHxgIB2Y4q\n4HewoY9KxgcvmMR+db85HUVmBmXUnTl46mrGlI04O7nDhPA6rt+11BJRhKJBomPI\nL3VkRyzfmEIqauc8Nmuma4G6NLNdytYPMp6VhpRoDPJMoHw7V11i7wNLyUUB3Ty8\n/uqjynVgXiQoml6f7mMqCG/QqslGKwol0p29V9bIg15wdq/p64cxsrM4mognfKw9\nenKVDQaoKA8YJNKzoH2O2IVt7zumbBmA3RW/ut98knm2nvAaYfIqIMq+uZYR958l\nCBhPjfLomz56lbqBIq1ILOpKiZr7OFvn177JW2KjIjLYRUw=\n-----END CERTIFICATE-----`
var clientCert = `-----BEGIN CERTIFICATE-----\nMIIDfjCCAmagAwIBAgIEeUV9HjANBgkqhkiG9w0BAQsFADCBijEtMCsGA1UELhMk\nN2U4NDY0MjQtYWY4NS00OGFhLWJhNDktMDc5ZmMyZTM2ZWViMTYwNAYDVQQDEy1H\nb29nbGUgQ2xvdWQgU1FMIENsaWVudCBDQSBteWxvY2FsbWFjaGluZWNlcnQxFDAS\nBgNVBAoTC0dvb2dsZSwgSW5jMQswCQYDVQQGEwJVUzAeFw0yMDEyMTgxMzU5MjFa\nFw0zMDEyMTYxNDAwMjFaMEAxGzAZBgNVBAMTEm15bG9jYWxtYWNoaW5lY2VydDEU\nMBIGA1UEChMLR29vZ2xlLCBJbmMxCzAJBgNVBAYTAlVTMIIBIjANBgkqhkiG9w0B\nAQEFAAOCAQ8AMIIBCgKCAQEAjemhf7Y5MIvCjTuuSkLSiYM2FWs1m103eBYGgyRp\nSCPHx2Y8RzLcB7RqLSt18itTxsUWTnSvcla2HCQTQ/bqwDHAuKqaejKXKZssV0+y\nL8jD1Ockfucdj1PdZn+PAayYYGGLSUlPHFRMrK/pSjnYL+/aqm0w0IuJg5XAKdMX\nLKBGy0qvBaKNuN1jJQem9FswDlLnrRRxUMme5VB8dq4OzaOmNHvPZ3Upujra/0uH\nKleURb1sZr5ZgbUa178fvbbFAeoet95JQfD3FrVOzeL2dGnwA+cIoQ91h0VW+2Zs\nlBXAJ8p9bseqQuUvQhV31lp+MpLoeRSmGhUFcKMSkoYufQIDAQABozUwMzAJBgNV\nHRMEAjAAMCYGA1UdEQQfMB2BG2Rldmlsc2RvbnRjcnkyMDExQGdtYWlsLmNvbTAN\nBgkqhkiG9w0BAQsFAAOCAQEAHleutSjBC6O+yB715QGowshw3kJTgMsQqQqzO5lS\n5ugXtX/NVB+DRfCEkW5kEWetcw4m0gCEBbtBLmmuUXptmMWmJgvUjQLQIRkLcBcw\nmt9G8LHVwofOPQT/XB7++9NyR+Efaq9Rs64y+4EzRqEr0FiCHqlxh+mFia85pysw\ncsbB61ZYeIfJk20TR5JcunAj5Jw6+kYMQBZmdU1AO3V9IZg2B56jszaVSkI59JoW\nLTCVYh1djMy4ofRPK5yRW6S5k9/jQ+lG2maVoUoqXna4MFg5WZjsZxmHx7aaS9IJ\nqlyPZ+PQQ57sEypm1y58786SJmXDdFlTWJgLoC8GTy3MBA==\n-----END CERTIFICATE-----`
var clientKey = `-----BEGIN RSA PRIVATE KEY-----\nMIIEowIBAAKCAQEAjemhf7Y5MIvCjTuuSkLSiYM2FWs1m103eBYGgyRpSCPHx2Y8\nRzLcB7RqLSt18itTxsUWTnSvcla2HCQTQ/bqwDHAuKqaejKXKZssV0+yL8jD1Ock\nfucdj1PdZn+PAayYYGGLSUlPHFRMrK/pSjnYL+/aqm0w0IuJg5XAKdMXLKBGy0qv\nBaKNuN1jJQem9FswDlLnrRRxUMme5VB8dq4OzaOmNHvPZ3Upujra/0uHKleURb1s\nZr5ZgbUa178fvbbFAeoet95JQfD3FrVOzeL2dGnwA+cIoQ91h0VW+2ZslBXAJ8p9\nbseqQuUvQhV31lp+MpLoeRSmGhUFcKMSkoYufQIDAQABAoIBAHNPl46yfp3XsmoY\nSHLHAVQDbfrRdmmbwOqu2vPMrk+T400+4VPpG6iXDH9PhTMVyakFlC6D2dvKYYdU\nOONMy0sIIlTrK0KHwRRppgn8FAmH1Lg2aQ1Etlw0BP64P3dYyyflmswd6U5XoUXg\nWmuZvPSWrNM2jiemekKVd+OERpxXFp7G3gm9hUdcdkWUN3uSt9meP9U/D93rskxH\ncz9u8NM9pSHZM/hDhnmklBpK1ZCTFKLmbBxLG5e44l2rEox7ZDk9hoGiUH2VZiWa\nffimgsbIWXLLQpuMGsaWnFEGg4IogkNO8QKnvfAWC6xjmpKLs1/gs8uGueMdMOJG\nQIVg+3ECgYEA3JTBTEo6dQwRxKywHe+evAVkjPOtPsoC08QQCNvmp2vwpfjU7OOM\nPRGYSATbDg5QKJhqIVuv+QtkhBm5Iil6OXmDceaKeAg8IykQTnZeRN94Z9ZFvQG6\nisC+IwpUmN9fSMziVHU7wvPb+Daqa11RBeA0oVe9aCjZxtfrtJLtr1cCgYEApLMd\nxkKXRUhkBdZNvg3SL2dgG6x3k02NWrwPA+glrxi2SY6OuVuC8CwNhO1Ufr8cGwUN\n13FSS5qeEq6m+jdPoumipw2MudLltcgRPsiaint88YVGH1fOCAx/Ol2zo6EeNNd8\n5bjf6eLj7jYhRVLWLRlBvjxOsoVXEcGfB1VosEsCgYEAk9jlAxSRwBhZ5IB2/2m3\n9ICM1+kQixBt+rDkqntyS2+O+kAhv7H5MomTj1op1W8EjWEzaa6B8aSQN/bh3yL7\n+IAY+YQz9aZXYJ3KfzzZjMJYewjk7320IgJ0rxnigCDgRfSGE2QMrWw0WVeSvKns\nf8q6nBYqLyGKbrwVEZCV3bsCgYBh+O7IRkqH+zUMx9t9J+mVK1Bfheuno2VnySDB\neTIZg4DEJto95vLv/bSZEzjFINgONqZyN0X2FWbcxCouBAMKbSLFbuj8jTj6NqYT\ni+9qW2UKovYApRG5df2k8aJvvuMiGeGBIcWI8uAVjvuhqlIfh7u091j1Fx6hQGVi\nTms1GwKBgFIu1oUMF6jH9wW90PoE0LjJtUfdNApYRHT60j+/wVDKd5sdMnH4IRon\n5EJm7OCopTvAy7Ymj5MhE17SNjgQakX+OMPYzo/ldv9mIjLjMhCXHP6J0QzTk6r9\nHqLxX5lwD1vt3WOCQaVrAA+XVhBzl9tCaAAWgNPC7YwSkzBCP8+N\n-----END RSA PRIVATE KEY-----`

func setUp(json string) (c echo.Context, h *handler, rec *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(json))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	h = newHandler("inmem", nil)
	e.Validator = h.validator

	return
}

// second http req, must call setup first
func newReq(json string, ctx echo.Context) (newCtx echo.Context, rec *httptest.ResponseRecorder) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(json))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	newCtx = ctx.Echo().NewContext(req, rec)
	return
}

func TestPutStruct(t *testing.T) {
	c, h, rec := setUp(requestJSON)

	if assert.NoError(t, h.putStruct(c)) {
		assert.Equal(t, http.StatusCreated, rec.Code)
		assert.Equal(t, 1, len(*h.DataModels))
		assert.ObjectsAreEqual(strct, (*h.DataModels)["Jon_Snow"])
	}
}

func TestGetStruct(t *testing.T) {
	c, h, _ := setUp(requestJSON)

	h.putStruct(c)

	if assert.NoError(t, h.getStruct(c)) {
		assert.Equal(t, 1, len(*h.DataModels))
		assert.ObjectsAreEqual(strct, (*h.DataModels)["Jon_Snow"])
	}
}

func TestStartBinlog(t *testing.T) {
	c, h, _ := setUp(requestJSON)
	h.putStruct(c)

	c, rec := newReq(binLogConfigJSON, c)
	if assert.NoError(t, h.startParser(c)) {
		assert.Equal(t, http.StatusAccepted, rec.Code)
	}
}

func TestStopBinlog(t *testing.T) {
	c, h, _ := setUp(requestJSON)
	h.putStruct(c)

	c, rec := newReq(binLogConfigJSON, c)
	h.startParser(c)
	<-time.After(time.Second)
	if assert.NoError(t, h.stopParser(c)) {
		assert.Equal(t, http.StatusAccepted, rec.Code)
	}
}

func TestStartStore(t *testing.T) {
	c, h, rec := setUp(requestJSON)
	h.putStruct(c)
	c, _ = newReq(binLogConfigJSON, c)
	h.startParser(c)
	c, rec = newReq(storeConfigJSON, c)

	if assert.NoError(t, h.startSyncer(c)) {
		assert.Equal(t, http.StatusAccepted, rec.Code)
	}
}

func TestStopStore(t *testing.T) {
	c, h, rec := setUp(requestJSON)
	h.putStruct(c)
	c, _ = newReq(binLogConfigJSON, c)
	h.startParser(c)
	c, _ = newReq(storeConfigJSON, c)
	h.startSyncer(c)
	c, rec = newReq("", c)

	if assert.NoError(t, h.stopSyncer(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
	}
}
