// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package genai

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// Ptr returns a pointer to its argument.
// It can be used to initialize pointer fields:
//
//	genai.GenerateContentConfig{Temperature: genai.Ptr(0.5)}
func Ptr[T any](t T) *T { return &t }

//nolint:unused
type converterFuncWithClientWithRoot func(*apiClient, map[string]any, map[string]any, map[string]any) (map[string]any, error)

type converterFuncWithRoot func(map[string]any, map[string]any, map[string]any) (map[string]any, error)

type transformerFunc[T any] func(T) (T, error)

// setValueByPath handles setting values within nested maps, including handling array-like structures.
//
// Examples:
//
//	setValueByPath(map[string]any{}, []string{"a", "b"}, v)
//	  -> {"a": {"b": v}}
//
//	setValueByPath(map[string]any{}, []string{"a", "b[]", "c"}, []any{v1, v2})
//	  -> {"a": {"b": [{"c": v1}, {"c": v2}]}}
//
//	setValueByPath(map[string]any{"a": {"b": [{"c": v1}, {"c": v2}]}}, []string{"a", "b[]", "d"}, v3)
//	  -> {"a": {"b": [{"c": v1, "d": v3}, {"c": v2, "d": v3}]}}
func setValueByPath(data map[string]any, keys []string, value any) {
	if value == nil {
		return
	}
	for i, key := range keys[:len(keys)-1] {
		if strings.HasSuffix(key, "[]") {
			keyName := key[:len(key)-2]
			if _, ok := data[keyName]; !ok {
				if reflect.ValueOf(value).Kind() == reflect.Slice {
					data[keyName] = make([]map[string]any, reflect.ValueOf(value).Len())
				} else {
					data[keyName] = make([]map[string]any, 1)
				}
				for k := range data[keyName].([]map[string]any) {
					data[keyName].([]map[string]any)[k] = make(map[string]any)
				}
			}

			if reflect.ValueOf(value).Kind() == reflect.Slice {
				for j, d := range data[keyName].([]map[string]any) {
					if j >= reflect.ValueOf(value).Len() {
						continue
					}
					setValueByPath(d, keys[i+1:], reflect.ValueOf(value).Index(j).Interface())
				}
			} else {
				for _, d := range data[keyName].([]map[string]any) {
					setValueByPath(d, keys[i+1:], value)
				}
			}
			return
		} else if strings.HasSuffix(key, "[0]") {
			keyName := key[:len(key)-3]
			if _, ok := data[keyName]; !ok {
				data[keyName] = make([]map[string]any, 1)
				data[keyName].([]map[string]any)[0] = make(map[string]any)
			}
			setValueByPath(data[keyName].([]map[string]any)[0], keys[i+1:], value)
			return
		} else {
			if _, ok := data[key]; !ok {
				data[key] = make(map[string]any)
			}
			if _, ok := data[key].(map[string]any); !ok {
				data[key] = make(map[string]any)
			}
			data = data[key].(map[string]any)
		}
	}

	finalKey := keys[len(keys)-1]
	existingValue, exists := data[finalKey]

	if exists {
		// 1. Check if the new value is "empty" (nil, zero value, or empty collection)
		isNewValueEmpty := false
		if value == nil {
			isNewValueEmpty = true
		} else {
			valReflect := reflect.ValueOf(value)
			if valReflect.Kind() == reflect.Invalid { // Handles nil interface{}
				isNewValueEmpty = true
			} else if valReflect.IsZero() { // Covers zero values for primitives, nil for pointers/interfaces
				isNewValueEmpty = true
			} else if (valReflect.Kind() == reflect.Map || valReflect.Kind() == reflect.Slice) && valReflect.Len() == 0 {
				isNewValueEmpty = true
			}
		}

		if isNewValueEmpty {
			// If new value is empty, do not overwrite existing non-empty value.
			// This is triggered when handling tuning datasets.
			return
		}
		if reflect.DeepEqual(value, existingValue) {
			// Don't fail when overwriting value with same value
			return
		}
		if existingMap, ok1 := existingValue.(map[string]any); ok1 {
			if newMap, ok2 := value.(map[string]any); ok2 {
				// Instead of overwriting dictionary with another dictionary, merge them.
				// This is important for handling training and validation datasets in tuning.
				for k, v := range newMap {
					existingMap[k] = v
				}
				data[finalKey] = existingMap // Assign the updated map back
			}
		} else {
			log.Println("Error. Cannot set value for an existing key. Key: ", finalKey, "; Existing value: ", existingValue, "; New value: ", value)
		}
	} else {
		if finalKey == "_self" && reflect.TypeOf(value).Kind() == reflect.Map {
			// Iterate through the `value` map and copy its contents to `data`.
			if valMap, ok := value.(map[string]any); ok {
				for k, v := range valMap {
					data[k] = v
				}
			}
		} else {
			// If existing_data is None (or key doesn't exist), set the value directly.
			data[finalKey] = value
		}
	}
}

// getValueByPath retrieves a value from a nested map or slice or struct based on a path of keys.
//
// Examples:
//
//	getValueByPath(map[string]any{"a": {"b": "v"}}, []string{"a", "b"})
//	  -> "v"
//	getValueByPath(map[string]any{"a": {"b": [{"c": "v1"}, {"c": "v2"}]}}, []string{"a", "b[]", "c"})
//	  -> []any{"v1", "v2"}
func getValueByPath(data any, keys []string) any {
	if len(keys) == 1 && keys[0] == "_self" {
		return data
	}
	if len(keys) == 0 {
		return nil
	}
	var current any = data
	for i, key := range keys {
		if strings.HasSuffix(key, "[]") {

			keyName := key[:len(key)-2]
			switch v := current.(type) {
			case map[string]any:
				if sliceData, ok := v[keyName]; ok {
					var result []any
					switch concreteSliceData := sliceData.(type) {
					case []map[string]any:
						for _, d := range concreteSliceData {
							result = append(result, getValueByPath(d, keys[i+1:]))
						}
					case []any:
						for _, d := range concreteSliceData {
							result = append(result, getValueByPath(d, keys[i+1:]))
						}
					default:
						return nil
					}
					return result
				} else {
					return nil
				}
			default:
				return nil
			}
		} else {
			switch v := current.(type) {
			case map[string]any:
				current = v[key]
			default:
				return nil
			}
		}
	}
	return current
}

// getValueByPathOrDefault retrieves a value from a nested map or slice or struct based on a path of
// keys, or returns a default value.
func getValueByPathOrDefault(data any, keys []string, defaultValue any) any {
	if len(keys) == 1 && keys[0] == "_self" {
		return data
	}
	if len(keys) == 0 {
		return defaultValue
	}
	var current any = data
	for i, key := range keys {
		if strings.HasSuffix(key, "[]") {

			keyName := key[:len(key)-2]
			switch v := current.(type) {
			case map[string]any:
				if sliceData, ok := v[keyName]; ok {
					var result []any
					switch concreteSliceData := sliceData.(type) {
					case []map[string]any:
						for _, d := range concreteSliceData {
							result = append(result, getValueByPathOrDefault(d, keys[i+1:], defaultValue))
						}
					case []any:
						for _, d := range concreteSliceData {
							result = append(result, getValueByPathOrDefault(d, keys[i+1:], defaultValue))
						}
					default:
						return defaultValue
					}
					return result
				} else {
					return defaultValue
				}
			default:
				return defaultValue
			}
		} else {
			switch v := current.(type) {
			case map[string]any:
				var ok bool
				current, ok = v[key]
				if !ok {
					return defaultValue
				}
			default:
				return defaultValue
			}
		}
	}
	if current == nil {
		return defaultValue
	}
	return current
}

func formatMap(template string, variables map[string]any) (string, error) {
	var buffer bytes.Buffer
	for i := 0; i < len(template); i++ {
		if template[i] == '{' {
			j := i + 1
			for j < len(template) && template[j] != '}' {
				j++
			}
			if j < len(template) {
				key := template[i+1 : j]
				if value, ok := variables[key]; ok {
					switch val := value.(type) {
					case string:
						buffer.WriteString(val)
					default:
						return "", errors.New("formatMap: nested interface or unsupported type found")
					}
				}
				i = j
			}
		} else {
			buffer.WriteByte(template[i])
		}
	}
	return buffer.String(), nil
}

// applyConverterToSlice calls converter function (with API client) to each element of the slice.
//
//nolint:unused
func applyConverterToSliceWithClientWithRoot(ac *apiClient, inputs []any, converter converterFuncWithClientWithRoot, rootObject map[string]any) ([]map[string]any, error) {
	var outputs []map[string]any
	for _, object := range inputs {
		object, err := converter(ac, object.(map[string]any), nil, rootObject)
		if err != nil {
			return nil, err
		}
		outputs = append(outputs, object)
	}
	return outputs, nil
}

// applyConverterToSlice calls converter function to each element of the slice.
//
//nolint:unused
func applyConverterToSliceWithRoot(inputs []any, converter converterFuncWithRoot, rootObject map[string]any) ([]map[string]any, error) {
	var outputs []map[string]any
	for _, object := range inputs {
		object, err := converter(object.(map[string]any), nil, rootObject)
		if err != nil {
			return nil, err
		}
		outputs = append(outputs, object)
	}
	return outputs, nil
}

// applyItemTransformerToSlice calls item transformer function to each element of the slice.
func applyItemTransformerToSlice[T any](inputs []T, itemTransformer transformerFunc[T]) ([]T, error) {
	var outputs []T
	for _, input := range inputs {
		object, err := itemTransformer(input)
		if err != nil {
			return nil, err
		}
		outputs = append(outputs, object)
	}
	return outputs, nil
}

func deepMarshal(input any, output *map[string]any) error {
	if inputBytes, err := json.Marshal(input); err != nil {
		return fmt.Errorf("deepMarshal: unable to marshal input: %w", err)
	} else if err := json.Unmarshal(inputBytes, output); err != nil {
		return fmt.Errorf("deepMarshal: unable to unmarshal input: %w", err)
	}
	return nil
}

func deepCopy[T any](original T, copied *T) error {
	bytes, err := json.Marshal(original)
	if err != nil {
		return err
	}

	err = json.Unmarshal(bytes, copied)
	return err
}

// createURLQuery creates a URL query string from a map of key-value pairs.
// The keys are sorted alphabetically before being encoded.
// Supported value types are string, int, float64, bool, and []string.
// An error is returned if an unsupported type is encountered.
func createURLQuery(query map[string]any) (string, error) {
	v := url.Values{}
	keys := make([]string, 0, len(query))
	for k := range query {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := query[key]
		switch value := value.(type) {
		case string:
			v.Add(key, value)
		case int:
			v.Add(key, strconv.Itoa(value))
		case float64:
			v.Add(key, strconv.FormatFloat(value, 'f', -1, 64))
		case bool:
			v.Add(key, strconv.FormatBool(value))
		case []string:
			for _, item := range value {
				v.Add(key, item)
			}
		default:
			return "", fmt.Errorf("unsupported type: %T", value)
		}
	}
	return v.Encode(), nil
}

func yieldErrorAndEndIterator[T any](err error) iter.Seq2[*T, error] {
	return func(yield func(*T, error) bool) {
		if !yield(nil, err) {
			return
		}
	}
}

func mergeHTTPOptions(clientConfig *ClientConfig, configHTTPOptions *HTTPOptions) *HTTPOptions {
	var clientHTTPOptions *HTTPOptions
	if clientConfig != nil {
		clientHTTPOptions = &(clientConfig.HTTPOptions)
	}

	// TODO(b/422842863): Implement a more flexible HTTPOptions merger.
	result := HTTPOptions{}
	if clientHTTPOptions == nil && configHTTPOptions == nil {
		return nil
	} else if clientHTTPOptions == nil {
		result = HTTPOptions{
			BaseURL:               configHTTPOptions.BaseURL,
			APIVersion:            configHTTPOptions.APIVersion,
			ExtrasRequestProvider: configHTTPOptions.ExtrasRequestProvider,
		}
	} else {
		result = HTTPOptions{
			BaseURL:               clientHTTPOptions.BaseURL,
			APIVersion:            clientHTTPOptions.APIVersion,
			ExtrasRequestProvider: clientHTTPOptions.ExtrasRequestProvider,
		}
	}

	if configHTTPOptions != nil {
		if configHTTPOptions.BaseURL != "" {
			result.BaseURL = configHTTPOptions.BaseURL
		}
		if configHTTPOptions.APIVersion != "" {
			result.APIVersion = configHTTPOptions.APIVersion
		}
		if configHTTPOptions.ExtrasRequestProvider != nil {
			result.ExtrasRequestProvider = configHTTPOptions.ExtrasRequestProvider
		}
	}
	result.Headers = mergeHeaders(clientHTTPOptions, configHTTPOptions)
	return &result
}

func mergeHeaders(clientHTTPOptions *HTTPOptions, configHTTPOptions *HTTPOptions) http.Header {
	result := http.Header{}
	if clientHTTPOptions == nil && configHTTPOptions == nil {
		return result
	}

	if clientHTTPOptions != nil {
		doMergeHeaders(clientHTTPOptions.Headers, &result)
	}
	// configHTTPOptions takes precedence over clientHTTPOptions.
	if configHTTPOptions != nil {
		doMergeHeaders(configHTTPOptions.Headers, &result)
	}
	return result
}

func doMergeHeaders(input http.Header, output *http.Header) {
	for k, v := range input {
		for _, vv := range v {
			output.Add(k, vv)
		}
	}
}

// moveValueByPath moves values from source paths to destination paths.
//
// Examples:
//
//	moveValueByPath(
//	  map[string]any{"requests": []any{map[string]any{"content": "v1"}, map[string]any{"content": "v2"}}},
//	  map[string]string{"requests[].*": "requests[].request.*"}
//	)
//	  -> {"requests": [{"request": {"content": "v1"}}, {"request": {"content": "v2"}}]}
func moveValueByPath(data any, paths map[string]string) {
	for sourcePath, destPath := range paths {
		sourceKeys := strings.Split(sourcePath, ".")
		destKeys := strings.Split(destPath, ".")

		// Determine keys to exclude from wildcard to avoid cyclic references
		excludeKeys := make(map[string]bool)
		wildcardIdx := -1
		for i, key := range sourceKeys {
			if key == "*" {
				wildcardIdx = i
				break
			}
		}

		if wildcardIdx != -1 && len(destKeys) > wildcardIdx {
			// Extract the intermediate key between source and dest paths
			// Example: source=["requests[]", "*"], dest=["requests[]", "request", "*"]
			// We want to exclude "request"
			for i := wildcardIdx; i < len(destKeys); i++ {
				key := destKeys[i]
				if key != "*" && !strings.HasSuffix(key, "[]") && !strings.HasSuffix(key, "[0]") {
					excludeKeys[key] = true
				}
			}
		}

		moveValueRecursive(data, sourceKeys, destKeys, 0, excludeKeys)
	}
}

// moveValueRecursive recursively moves values from source path to destination path.
func moveValueRecursive(data any, sourceKeys []string, destKeys []string, keyIdx int, excludeKeys map[string]bool) {
	if keyIdx >= len(sourceKeys) {
		return
	}

	key := sourceKeys[keyIdx]

	if strings.HasSuffix(key, "[]") {
		keyName := key[:len(key)-2]
		if dataMap, ok := data.(map[string]any); ok {
			if sliceData, exists := dataMap[keyName]; exists {
				switch slice := sliceData.(type) {
				case []any:
					for _, item := range slice {
						moveValueRecursive(item, sourceKeys, destKeys, keyIdx+1, excludeKeys)
					}
				case []map[string]any:
					for _, item := range slice {
						moveValueRecursive(item, sourceKeys, destKeys, keyIdx+1, excludeKeys)
					}
				}
			}
		}
	} else if key == "*" {
		// Handle wildcard - move all fields
		if dataMap, ok := data.(map[string]any); ok {
			keysToMove := []string{}
			for k := range dataMap {
				if !strings.HasPrefix(k, "_") && !excludeKeys[k] {
					keysToMove = append(keysToMove, k)
				}
			}
			valuesToMove := make(map[string]any)
			for _, k := range keysToMove {
				valuesToMove[k] = dataMap[k]
			}

			// Set values at destination
			for k, v := range valuesToMove {
				newDestKeys := []string{}
				for _, dk := range destKeys[keyIdx:] {
					if dk == "*" {
						newDestKeys = append(newDestKeys, k)
					} else {
						newDestKeys = append(newDestKeys, dk)
					}
				}
				setValueByPath(dataMap, newDestKeys, v)
			}

			for _, k := range keysToMove {
				delete(dataMap, k)
			}
		}
	} else {
		// Navigate to next level
		if dataMap, ok := data.(map[string]any); ok {
			if nextData, exists := dataMap[key]; exists {
				moveValueRecursive(nextData, sourceKeys, destKeys, keyIdx+1, excludeKeys)
			}
		}
	}
}
