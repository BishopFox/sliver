package util

import (
	"fmt"
	"strings"
)

// Query 将Map序列化为Query参数
func Query(params map[string]interface{}) string {
	finalString := make([]string, 0)
	for key, value := range params {
		valueString := ""
		switch v := value.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			valueString = fmt.Sprintf("%d", v)
		case bool:
			valueString = fmt.Sprintf("%v", v)
		default:
			valueString = fmt.Sprintf("%s", v)
		}
		finalString = append(finalString, strings.Join([]string{key, valueString}, "="))
	}
	return strings.Join(finalString, "&")
}
