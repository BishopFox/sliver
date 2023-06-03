package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/rsteube/carapace/pkg/xdg"
)

type configMap map[string]interface{}

func (c configMap) Keys() []string {
	keys := make([]string, 0)
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}

type Field struct {
	Name        string
	Description string
	Style       string
	Tag         string
}

func (c configMap) Fields(name string) ([]Field, error) {
	if i, ok := c[name]; ok {
		fields := make([]Field, 0)
		t := reflect.TypeOf(i).Elem()
		v := reflect.ValueOf(i).Elem()
		for index := 0; index < t.NumField(); index++ {
			field := t.Field(index)
			if field.Type.Name() != "string" {
				return nil, fmt.Errorf("invalid field type [name: '%v', type: '%v']", field.Name, field.Type.Name())
			}
			fields = append(fields, Field{field.Name, field.Tag.Get("description"), v.FieldByName(field.Name).String(), field.Tag.Get("tag")})
		}
		return fields, nil
	}
	return nil, fmt.Errorf("unknown config: '%v'", name)
}

var config = struct {
	Styles configMap
}{
	Styles: make(configMap),
}

func RegisterStyle(name string, i interface{}) {
	config.Styles[name] = i
}

func Load() error {
	if err := load("styles", config.Styles); err != nil {
		return err
	}
	return nil
}

func load(name string, c configMap) error {
	if dir, err := xdg.UserConfigDir(); err == nil {
		content, err := os.ReadFile(fmt.Sprintf("%v/carapace/%v.json", dir, name))
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}

		var unmarshalled map[string]map[string]string
		if err := json.Unmarshal(content, &unmarshalled); err != nil {
			return err
		}

		for key, value := range unmarshalled {
			if s, ok := c[key]; ok {
				elem := reflect.ValueOf(s).Elem()
				for k, v := range value {
					if field := elem.FieldByName(k); field != (reflect.Value{}) {
						field.SetString(v)
					}
				}
			}
		}
	}
	return nil
}

func GetStyleConfigs() []string                   { return config.Styles.Keys() }
func GetStyleFields(name string) ([]Field, error) { return config.Styles.Fields(name) }
func SetStyle(key, value string) error {
	return set("styles", key, strings.Replace(value, ",", " ", -1))
}

func set(name, key, value string) error {
	dir, err := xdg.UserConfigDir()
	if err != nil {
		return err
	}

	file := fmt.Sprintf("%v/carapace/%v.json", dir, name)
	content, err := os.ReadFile(file)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(file), os.ModePerm); err != nil {
			return err
		}
		content = []byte("{}")
	}

	var config map[string]map[string]string
	if err := json.Unmarshal(content, &config); err != nil {
		return err
	}

	if splitted := strings.Split(key, "."); len(splitted) != 2 {
		return errors.New("invalid key")
	} else {
		if _, ok := config[splitted[0]]; !ok {
			config[splitted[0]] = make(map[string]string, 0)
		}
		if strings.TrimSpace(value) == "" {
			delete(config[splitted[0]], splitted[1])
		} else {
			config[splitted[0]][splitted[1]] = value
		}
	}

	marshalled, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(file, marshalled, os.ModePerm)
}
