package util

import (
	"fmt"
	"strings"
)

// Template 对字符串中的和map的key相同的字符串进行模板替换 仅支持 形如: {name}
func Template(source string, data map[string]interface{}) string {
	sourceCopy := &source
	for k, val := range data {
		valStr := ""
		switch v := val.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			valStr = fmt.Sprintf("%d", v)
		case bool:
			valStr = fmt.Sprintf("%v", v)
		default:
			valStr = fmt.Sprintf("%s", v)
		}
		*sourceCopy = strings.Replace(*sourceCopy, strings.Join([]string{"{", k, "}"}, ""), valStr, 1)
	}
	return *sourceCopy
}
