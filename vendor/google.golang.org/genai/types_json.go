// Copyright 2025 Google LLC
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
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"cloud.google.com/go/civil"
)

type int64SliceJSON []int64

func (s *int64SliceJSON) UnmarshalJSON(data []byte) error {
	var stringSlice []string
	if err := json.Unmarshal(data, &stringSlice); err != nil {
		// If both attempts fail, return a more informative error
		return fmt.Errorf("failed to unmarshal as []int64 or []string: %w", err)
	}

	if stringSlice == nil {
		return nil
	}

	// If successful as a []string, convert each element to int64
	result := make([]int64, 0, len(stringSlice))
	for _, str := range stringSlice {
		val, err := strconv.ParseInt(str, 10, 64)
		if err != nil {
			return err // Error during string-to-int conversion
		}
		result = append(result, val)
	}

	*s = result
	return nil
}

func (s *int64SliceJSON) MarshalJSON() ([]byte, error) {
	stringSlice := make([]string, 0)
	for _, val := range *s {
		stringSlice = append(stringSlice, strconv.FormatInt(val, 10))
	}
	return json.Marshal(stringSlice)
}

type durationJSON time.Duration

func (d *durationJSON) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("failed to unmarshal duration from string: %w", err)
	}
	if len(s) == 0 {
		return nil
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("failed to parse duration string %q: %w", s, err)
	}
	*d = durationJSON(dur)
	return nil
}

func (d *durationJSON) MarshalJSON() ([]byte, error) {
	return json.Marshal(fmt.Sprintf("%.0fs", time.Duration(*d).Seconds()))
}

type dateJSON civil.Date

func (d *dateJSON) UnmarshalJSON(data []byte) error {
	m := make(map[string]int)
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("failed to unmarshal date from map: %w", err)
	}

	if len(m) == 0 {
		return nil
	}
	if _, ok := m["year"]; !ok {
		return fmt.Errorf("key %q not found", "year")
	}
	d.Year = m["year"]

	if month, ok := m["month"]; ok {
		d.Month = time.Month(month)
	}
	if day, ok := m["day"]; ok {
		d.Day = day
	}
	return nil
}

func (d *dateJSON) MarshalJSON() ([]byte, error) {
	m := make(map[string]int)
	if d == nil || (civil.Date)(*d).IsZero() {
		return json.Marshal(nil)
	}
	if d.Year != 0 {
		m["year"] = d.Year
	}
	if d.Month != 0 {
		m["month"] = int(d.Month)
	}
	if d.Day != 0 {
		m["day"] = d.Day
	}
	return json.Marshal(m)
}
