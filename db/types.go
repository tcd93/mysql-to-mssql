package db

import (
	"time"

	"github.com/shopspring/decimal"
)

// MySQLType represents MySQL's data type
type MySQLType uint8

// NOTE: type enum must start from 1 to avoid conflicts with validator library
const (
	// Int includes int, tinyint, mediumint, signed bigint; default value is 0.
	//
	// Warning: don't use this type on mysql unsinged bigint, use Uint instead
	Int MySQLType = iota + 1

	// NullableInt includes int, tinyint, mediumint, signed bigint; default value is nil.
	//
	// Warning: don't use this type on mysql unsinged bigint, use NullableUInt intead
	NullableInt

	// UInt includes unsigned integer types including unsinged bigint; default is 0
	UInt

	// NullableUInt includes unsigned integer types including unsinged bigint; default is nil
	NullableUInt

	// String includes char, varchar, text, time, enum, also support blob, binary, varbinary; default value is ""
	String

	// NullableString includes char, varchar, text, time, enum, also support blob, binary, varbinary; default value is nil
	NullableString

	// Bool includes tinyint (1 bit) type; default to false
	Bool

	// NullableBool includes tinyint (1 biy) type; default to nil
	NullableBool

	// DateTime includes date, datetime & timestamp; default to zero-value of Go's time.Time type (0001-01-01 00:00:00 +0000 UTC)
	DateTime

	// NullableDateTime includes date, datetime & timestamp; default to nil
	NullableDateTime

	// Float includes float (32 bit); default to 0.0
	Float

	// Float includes float (32 bit); default to nil
	NullableFloat

	// Double includes double (64 bit); default to 0.0
	Double

	// Double includes double (64 bit); default to nil
	NullableDouble

	// Decimal includes decimal; default to 0, under the hood, Go'd map it to Shopspring's decimal type
	Decimal

	// NullableDecimal includes decimal; default to nil
	NullableDecimal

	// Blob includes blob, binary, varbinary; default to empty array of bytes
	Blob

	// NullableBlob includes blob, binary, varbinary; default to nil
	NullableBlob

	// Set includes set; default to empty array of string
	Set

	// NullableSet includes set; default to nil
	NullableSet
)

// Entry is a record stored in database
type Entry struct {
	Key   string
	Value []byte
}

// Options records params for creating DB object.
type Options struct {
	// Dir represents Open the database located in which dir.
	Dir         string
	SegmentSize int64
}

// Interface for the storage engine. "bucket" can be viewed as a namespace for storage,
// inside it contains many key/value pairs data
type Interface interface {
	// Release & close database
	Release() error
	// Dir returns storage directory (empty for inmem)
	Dir() string
	SetDir(dir string)
	// GetAll entries in a bucket
	GetAll(bucket string) ([]*Entry, error)
	// GetAllKey returns all data for a key
	GetAllKey(bucket string, key string) ([][]byte, error)
	// Put or override an entry in a bucket
	Put(bucket string, key string, value []byte, ttl uint32) error
	// Push inserts the value at the tail of the list stored in the bucket at given key
	Push(bucket string, key string, value []byte) error
	// Rem remove `count` elements from List from left
	Rem(bucket string, key string, count int) error
	// Size get current "sync-pending" records from local database
	Size(bucket string, key string) (int, error)
	// Type: nutsdb or inmem
	Type() string
	// Truncate key array at bucket
	Truncate(bucket string, key string) error
}

// Convert MySQLType to a Golang compatible value
func Convert(mType MySQLType) interface{} {
	var t interface{}
	switch mType {
	case Int:
		t = int(0)
	case NullableInt:
		t = new(int)
	case UInt:
		t = uint(0)
	case NullableUInt:
		t = new(uint)
	case String:
		t = ""
	case NullableString:
		t = new(string)
	case Bool:
		t = false
	case NullableBool:
		t = new(bool)
	case DateTime:
		t = time.Time{}
	case NullableDateTime:
		t = new(time.Time)
	case Float:
		t = float32(0)
	case NullableFloat:
		t = new(float32)
	case Double:
		t = float64(0)
	case NullableDouble:
		t = new(float64)
	case Decimal:
		t = decimal.Zero
	case NullableDecimal:
		t = new(decimal.Decimal)
	case Blob:
		t = []byte{}
	case NullableBlob:
		t = new([]byte)
	case Set:
		t = []string{}
	case NullableSet:
		t = new([]string)
	}
	return t
}
