package generate

// import (
// 	"bytes"

// 	"github.com/Binject/go-donut/donut"
// )

// func getDonut(data []byte, params string, className string, method string, isDotNet bool) ([]byte, error) {
// 	buff := bytes.NewBuffer(data)
// 	config := &donut.DonutConfig{
// 		DotNetMode: isDotNet,
// 		Parameters: params,
// 		Class:      className,
// 		InstType:   donut.DONUT_INSTANCE_PIC,
// 		Method:     method,
// 		Bypass:     3, // continue on fail
// 		Compress:   1, // no compression - not yet supported in go-donut
// 		Format:     1, // raw shellcode
// 		ExitOpt:    1, // exit thread
// 	}
// 	payload, err := donut.ShellcodeFromBytes(buff, config)
// 	return payload.Bytes(), err
// }
