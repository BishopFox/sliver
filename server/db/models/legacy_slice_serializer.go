package models

/*
	Sliver Implant Framework
	Copyright (C) 2026  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"gorm.io/gorm/schema"
)

const legacyJSONSliceSerializerName = "legacyjsonslice"

func init() {
	schema.RegisterSerializer(legacyJSONSliceSerializerName, legacyJSONSliceSerializer{})
}

// legacyJSONSliceSerializer writes slices as JSON while accepting the scalar
// values emitted by the historical GORM tags for one-element slices. This is
// intentionally limited to the legacy CrackCommand list fields.
type legacyJSONSliceSerializer struct{}

func (legacyJSONSliceSerializer) Value(_ context.Context, _ *schema.Field, _ reflect.Value, fieldValue interface{}) (interface{}, error) {
	encoded, err := json.Marshal(fieldValue)
	if err != nil {
		return nil, err
	}
	if string(encoded) == "null" {
		return nil, nil
	}
	return string(encoded), nil
}

func (legacyJSONSliceSerializer) Scan(ctx context.Context, field *schema.Field, dst reflect.Value, dbValue interface{}) error {
	value := reflect.New(field.FieldType).Elem()
	if dbValue == nil {
		field.ReflectValueOf(ctx, dst).Set(value)
		return nil
	}

	raw := strings.TrimSpace(databaseScalarString(dbValue))
	if raw == "null" {
		field.ReflectValueOf(ctx, dst).Set(value)
		return nil
	}
	if strings.HasPrefix(raw, "[") {
		decoded := reflect.New(field.FieldType)
		if err := json.Unmarshal([]byte(raw), decoded.Interface()); err != nil {
			return fmt.Errorf("decode %s JSON slice: %w", field.Name, err)
		}
		field.ReflectValueOf(ctx, dst).Set(decoded.Elem())
		return nil
	}

	legacyValues := []string{raw}
	if strings.HasPrefix(raw, "{") && strings.HasSuffix(raw, "}") {
		contents := strings.TrimSpace(raw[1 : len(raw)-1])
		if contents == "" {
			field.ReflectValueOf(ctx, dst).Set(reflect.MakeSlice(field.FieldType, 0, 0))
			return nil
		}
		legacyValues = strings.Split(contents, ",")
	}

	value = reflect.MakeSlice(field.FieldType, 0, len(legacyValues))
	for _, legacyValue := range legacyValues {
		element, err := parseLegacySliceElement(strings.TrimSpace(legacyValue), field.FieldType.Elem())
		if err != nil {
			return fmt.Errorf("decode legacy %s value: %w", field.Name, err)
		}
		value = reflect.Append(value, element)
	}
	field.ReflectValueOf(ctx, dst).Set(value)
	return nil
}

func databaseScalarString(value interface{}) string {
	switch typed := value.(type) {
	case []byte:
		return string(typed)
	case string:
		return typed
	default:
		return fmt.Sprint(typed)
	}
}

func parseLegacySliceElement(raw string, elementType reflect.Type) (reflect.Value, error) {
	element := reflect.New(elementType).Elem()
	switch elementType.Kind() {
	case reflect.String:
		if strings.HasPrefix(raw, `"`) && strings.HasSuffix(raw, `"`) {
			var decoded string
			if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
				return reflect.Value{}, err
			}
			raw = decoded
		}
		element.SetString(raw)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value, err := strconv.ParseInt(raw, 10, elementType.Bits())
		if err != nil {
			return reflect.Value{}, err
		}
		element.SetInt(value)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		value, err := strconv.ParseUint(raw, 10, elementType.Bits())
		if err != nil {
			return reflect.Value{}, err
		}
		element.SetUint(value)
	default:
		return reflect.Value{}, fmt.Errorf("unsupported slice element type %s", elementType)
	}
	return element, nil
}
