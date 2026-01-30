package notifications

import (
	"os"
	"reflect"
	"strings"
	"unicode"
)

func resolveEnvString(value string) (string, bool, bool) {
	name, ok := parseEnvVar(value)
	if !ok {
		return value, false, false
	}
	expanded, found := os.LookupEnv(name)
	if !found {
		notificationsLog.Warnf("Notifications env var %q is not set", name)
		return "", true, false
	}
	if expanded == "" {
		notificationsLog.Debugf("Notifications env var %q is set but empty", name)
	}
	return expanded, true, true
}

func parseEnvVar(value string) (string, bool) {
	if len(value) < 2 || value[0] != '$' {
		return "", false
	}
	name := value[1:]
	if strings.HasPrefix(name, "{") && strings.HasSuffix(name, "}") {
		name = strings.TrimSuffix(strings.TrimPrefix(name, "{"), "}")
	}
	if !isValidEnvName(name) {
		return "", false
	}
	return name, true
}

func isValidEnvName(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		if i == 0 {
			if r != '_' && !unicode.IsLetter(r) {
				return false
			}
			continue
		}
		if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func expandEnv(target any) {
	if target == nil {
		return
	}
	value := reflect.ValueOf(target)
	if value.Kind() != reflect.Pointer {
		return
	}
	expandEnvValue(value)
}

func expandEnvValue(value reflect.Value) {
	if !value.IsValid() {
		return
	}

	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return
		}
		expandEnvValue(value.Elem())
		return
	}

	switch value.Kind() {
	case reflect.String:
		if !value.CanSet() {
			return
		}
		expanded, ok, _ := resolveEnvString(value.String())
		if ok {
			value.SetString(expanded)
		}
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			expandEnvValue(value.Field(i))
		}
	case reflect.Slice:
		for i := 0; i < value.Len(); i++ {
			expandEnvValue(value.Index(i))
		}
	case reflect.Map:
		if value.Type().Key().Kind() != reflect.String {
			return
		}
		if value.Type().Elem().Kind() == reflect.String {
			for _, key := range value.MapKeys() {
				val := value.MapIndex(key)
				expanded, ok, _ := resolveEnvString(val.String())
				if ok {
					value.SetMapIndex(key, reflect.ValueOf(expanded))
				}
			}
		}
	}
}
