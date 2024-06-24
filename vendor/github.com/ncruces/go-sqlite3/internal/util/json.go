package util

import (
	"encoding/json"
	"strconv"
	"time"
	"unsafe"
)

type JSON struct{ Value any }

func (j JSON) Scan(value any) error {
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
		buf = []byte("null")
	default:
		panic(AssertErr())
	}

	return json.Unmarshal(buf, j.Value)
}
