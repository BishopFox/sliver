package extension

import "errors"

var extensions map[string]Extension

type Extension interface {
	Load() error
	Call(exportName string, arguments []byte, callback func([]byte)) error
	GetID() string
	GetArch() string
}

func Add(e Extension) {
	extensions[e.GetID()] = e
}

func List() []string {
	var extList []string
	for id := range extensions {
		extList = append(extList, id)
	}
	return extList
}

func Run(extID string, funcName string, arguments []byte, callback func([]byte)) error {
	if ext, found := extensions[extID]; found {
		return ext.Call(funcName, arguments, callback)
	}
	return errors.New("{{if .Config.Debug}} extension not found{{end}}")
}

func init() {
	extensions = make(map[string]Extension)
}
