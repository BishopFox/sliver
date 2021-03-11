package registry

import (
	"fmt"
	// {{if .Config.Debug}}
	"log"
	// {{end}}
	"strings"

	"golang.org/x/sys/windows/registry"
)

var hives = map[string]registry.Key{
	"HKCR": registry.CLASSES_ROOT,
	"HKCU": registry.CURRENT_USER,
	"HKLM": registry.LOCAL_MACHINE,
	"HKPD": registry.PERFORMANCE_DATA,
	"HKU":  registry.USERS,
	"HKCC": registry.CURRENT_CONFIG,
}

func ReadKey(hive string, path string, key string) (string, error) {
	var (
		buf    []byte
		result string
	)
	hiveKey, found := hives[hive]
	if !found {
		return "", fmt.Errorf("could not find hive %s", hive)
	}
	k, err := registry.OpenKey(hiveKey, path, registry.QUERY_VALUE)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("could not open key %s: %s\n", path, err.Error())
		// {{end}}
		return "", err
	}

	_, valType, err := k.GetValue(key, buf)
	if err != nil {
		return "", err
	}
	switch valType {
	case registry.BINARY:
	case registry.SZ:
		fallthrough
	case registry.EXPAND_SZ:
		val, _, err := k.GetStringValue(key)
		if err != nil {
			return "", err
		}
		result = val
	case registry.DWORD:
		fallthrough
	case registry.QWORD:
		val, _, err := k.GetIntegerValue(key)
		if err != nil {
			return "", err
		}
		result = fmt.Sprintf("0x%08x", val)
	case registry.MULTI_SZ:
		val, _, err := k.GetStringsValue(key)
		if err != nil {
			return "", err
		}
		result = strings.Join(val, "\n")
	default:
		return "", fmt.Errorf("unhandled type: %d", valType)
	}
	return result, nil
}

func WriteKey(hive string, path string, key string) error {
	hiveKey, ok := hives[hive]
	if !ok {
		return fmt.Errorf("could not find hive %s", hive)
	}
	keyPath := fmt.Sprintf(`%s\%s`, path, key)
	_, err := registry.OpenKey(hiveKey, keyPath, registry.QUERY_VALUE|registry.SET_VALUE|registry.WRITE)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("could not open key %s: %s\n", keyPath, err.Error())
		// {{end}}
		return err
	}

	return nil
}
