// Copyright (c) 2025 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package jsontime

import (
	"database/sql/driver"
	"strconv"
	"time"
)

func unmarshalDuration(into *time.Duration, jsonData []byte, unit time.Duration) error {
	val, err := strconv.ParseInt(string(jsonData), 10, 64)
	if err != nil {
		return err
	}
	*into = time.Duration(val) * unit
	return nil
}

func anyIntegerToDuration(src any, unit time.Duration, into *time.Duration) error {
	i64, err := anyIntegerTo64(src)
	if err != nil {
		return err
	}
	*into = time.Duration(i64) * unit
	return nil
}

type Seconds struct {
	time.Duration
}

func S(dur time.Duration) Seconds {
	return Seconds{Duration: dur}
}

func SInt(dur int) Seconds {
	return Seconds{Duration: time.Duration(dur) * time.Second}
}

func (s Seconds) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatInt(int64(s.Seconds()), 10)), nil
}

func (s Seconds) Value() (driver.Value, error) {
	return int64(s.Seconds()), nil
}

func (s *Seconds) UnmarshalJSON(data []byte) error {
	return unmarshalDuration(&s.Duration, data, time.Second)
}

func (s *Seconds) Scan(src interface{}) error {
	return anyIntegerToDuration(src, time.Second, &s.Duration)
}

func (s *Seconds) Get() time.Duration {
	if s == nil {
		return 0
	}
	return s.Duration
}

type Milliseconds struct {
	time.Duration
}

func MS(dur time.Duration) Milliseconds {
	return Milliseconds{Duration: dur}
}

func MSInt(dur int64) Milliseconds {
	return Milliseconds{Duration: time.Duration(dur) * time.Millisecond}
}

func (s Milliseconds) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatInt(s.Milliseconds(), 10)), nil
}

func (s Milliseconds) Value() (driver.Value, error) {
	return s.Milliseconds(), nil
}

func (s *Milliseconds) UnmarshalJSON(data []byte) error {
	return unmarshalDuration(&s.Duration, data, time.Millisecond)
}

func (s *Milliseconds) Scan(src interface{}) error {
	return anyIntegerToDuration(src, time.Millisecond, &s.Duration)
}

func (s *Milliseconds) Get() time.Duration {
	if s == nil {
		return 0
	}
	return s.Duration
}

type Microseconds struct {
	time.Duration
}

func (s Microseconds) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatInt(s.Microseconds(), 10)), nil
}

func (s Microseconds) Value() (driver.Value, error) {
	return s.Microseconds(), nil
}

func (s *Microseconds) UnmarshalJSON(data []byte) error {
	return unmarshalDuration(&s.Duration, data, time.Microsecond)
}

func (s *Microseconds) Scan(src interface{}) error {
	return anyIntegerToDuration(src, time.Microsecond, &s.Duration)
}

func (s *Microseconds) Get() time.Duration {
	if s == nil {
		return 0
	}
	return s.Duration
}

type Nanoseconds struct {
	time.Duration
}

func (s Nanoseconds) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatInt(s.Nanoseconds(), 10)), nil
}

func (s Nanoseconds) Value() (driver.Value, error) {
	return s.Nanoseconds(), nil
}

func (s *Nanoseconds) UnmarshalJSON(data []byte) error {
	return unmarshalDuration(&s.Duration, data, time.Nanosecond)
}

func (s *Nanoseconds) Scan(src interface{}) error {
	return anyIntegerToDuration(src, time.Nanosecond, &s.Duration)
}

func (s *Nanoseconds) Get() time.Duration {
	if s == nil {
		return 0
	}
	return s.Duration
}
