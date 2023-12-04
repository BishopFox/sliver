package sqlite3

import (
	"encoding/json"
	"strconv"
	"time"
	"unsafe"

	"github.com/ncruces/go-sqlite3/internal/util"
)

// JSON returns a value that can be used as an argument to
// [database/sql.DB.Exec], [database/sql.Row.Scan] and similar methods to
// store value as JSON, or decode JSON into value.
func JSON(value any) any {
	return jsonValue{value}
}

type jsonValue struct{ any }

func (j jsonValue) JSON() any { return j.any }

func (j jsonValue) Scan(value any) error {
	var buf []byte

	switch v := value.(type) {
	case []byte:
		buf = v
	case string:
		buf = unsafe.Slice(unsafe.StringData(v), len(v))
	case int64:
		buf = strconv.AppendInt(nil, v, 10)
	case float64:
		buf = strconv.AppendFloat(nil, v, 'g', -1, 64)
	case time.Time:
		buf = append(buf, '"')
		buf = v.AppendFormat(buf, time.RFC3339Nano)
		buf = append(buf, '"')
	case nil:
		buf = append(buf, "null"...)
	default:
		panic(util.AssertErr())
	}

	return json.Unmarshal(buf, j.any)
}
