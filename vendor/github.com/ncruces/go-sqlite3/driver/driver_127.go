//go:build go1.27

package driver

import (
	"database/sql"
	"database/sql/driver"
	"math"
	"time"

	"github.com/ncruces/go-sqlite3"
)

func (r *rows) ScanColumn(ctx driver.ScanContext, i int, dest any) error {
	typ := r.Stmt.ColumnType(i)

	// Fast path.
	var src any
	switch typ {
	case sqlite3.NULL:
		// src = nil
	case sqlite3.FLOAT:
		f := r.stmt.ColumnFloat(i)
		if r.scanFloat(f, dest) {
			return nil
		}
		src = f
	case sqlite3.INTEGER:
		i := r.stmt.ColumnInt64(i)
		if r.scanInt(i, dest) {
			return nil
		}
		if r.scanFloat(float64(i), dest) {
			return nil
		}
		src = i
	default:
		var b []byte
		if typ == sqlite3.TEXT {
			b = r.stmt.ColumnRawText(i)
		} else {
			b = r.stmt.ColumnRawBlob(i)
		}
		switch d := dest.(type) {
		case *sql.RawBytes:
			*d = b
			return nil
		case *string:
			*d = string(b)
			return nil
		case *sql.NullString:
			d.String = string(b)
			d.Valid = true
			return nil
		case *[]byte:
			*d = append((*d)[:0], b...)
			return nil
		}
		src = b
	}

	// Time handling.
	if typ != sqlite3.NULL && typ != sqlite3.BLOB {
		var ok bool
		switch d := dest.(type) {
		case *time.Time:
			*d, ok = r.scanTime(src)
		case *sql.NullTime:
			d.Time, ok = r.scanTime(src)
			d.Valid = ok
		}
		if ok {
			return nil
		}
	}

	// Fallback.
	return sql.ConvertAssign(ctx, dest, r.convert(i, src))
}

func (r *rows) scanTime(src any) (time.Time, bool) {
	if s, ok := src.([]byte); ok {
		if t, ok := r.maybeTime(s); ok {
			return t, true
		}
		src = string(s)
	}
	t, err := r.tmRead.Decode(src)
	return t, err == nil
}

func (r *rows) scanFloat(f float64, dest any) bool {
	switch d := dest.(type) {
	case *float64:
		*d = f
	case *float32:
		*d = float32(f)
	case *sql.NullFloat64:
		d.Float64 = f
		d.Valid = true
	case *sql.Null[float32]:
		d.V = float32(f)
		d.Valid = true
	default:
		return false
	}
	return true
}

func (r *rows) scanInt(i int64, dest any) bool {
	switch d := dest.(type) {
	case *int64:
		*d = i
	case *sql.NullInt64:
		d.Int64 = i
		d.Valid = true

	case *uint64:
		*d = uint64(i)
		return 0 <= i
	case *sql.Null[uint64]:
		d.V = uint64(i)
		d.Valid = true
		return 0 <= i

	case *int:
		*d = int(i)
		return math.MinInt <= i && i <= math.MaxInt
	case *sql.Null[int]:
		d.V = int(i)
		d.Valid = true
		return math.MinInt <= i && i <= math.MaxInt

	case *uint:
		*d = uint(i)
		return 0 <= i && uint64(i) <= math.MaxUint
	case *sql.Null[uint]:
		d.V = uint(i)
		d.Valid = true
		return 0 <= i && uint64(i) <= math.MaxUint

	default:
		return false
	}
	return true
}
