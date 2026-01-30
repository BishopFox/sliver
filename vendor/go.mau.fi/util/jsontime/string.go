// Copyright (c) 2023 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package jsontime

import (
	"encoding/json"
	"strconv"
	"time"
)

func parseTimeString(data []byte, unixConv func(int64) time.Time, into *time.Time) error {
	var strVal string
	err := json.Unmarshal(data, &strVal)
	if err != nil {
		return err
	}
	val, err := strconv.ParseInt(strVal, 10, 64)
	if err != nil {
		return err
	}
	if val == 0 {
		*into = time.Time{}
	} else {
		*into = unixConv(val)
	}
	return nil
}

type UnixMilliString struct {
	time.Time
}

func (um UnixMilliString) MarshalJSON() ([]byte, error) {
	if um.IsZero() {
		return []byte{'"', '0', '"'}, nil
	}
	return json.Marshal(strconv.FormatInt(um.UnixMilli(), 10))
}

func (um *UnixMilliString) UnmarshalJSON(data []byte) error {
	return parseTimeString(data, time.UnixMilli, &um.Time)
}

type UnixMicroString struct {
	time.Time
}

func (um UnixMicroString) MarshalJSON() ([]byte, error) {
	if um.IsZero() {
		return []byte{'"', '0', '"'}, nil
	}
	return json.Marshal(strconv.FormatInt(um.UnixMicro(), 10))
}

func (um *UnixMicroString) UnmarshalJSON(data []byte) error {
	return parseTimeString(data, time.UnixMicro, &um.Time)
}

type UnixNanoString struct {
	time.Time
}

func (um UnixNanoString) MarshalJSON() ([]byte, error) {
	if um.IsZero() {
		return []byte{'"', '0', '"'}, nil
	}
	return json.Marshal(strconv.FormatInt(um.UnixNano(), 10))
}

func (um *UnixNanoString) UnmarshalJSON(data []byte) error {
	return parseTimeString(data, func(i int64) time.Time {
		return time.Unix(0, i)
	}, &um.Time)
}

type UnixString struct {
	time.Time
}

func (u UnixString) MarshalJSON() ([]byte, error) {
	if u.IsZero() {
		return []byte{'"', '0', '"'}, nil
	}
	return json.Marshal(strconv.FormatInt(u.Unix(), 10))
}

func (u *UnixString) UnmarshalJSON(data []byte) error {
	return parseTimeString(data, func(i int64) time.Time {
		return time.Unix(i, 0)
	}, &u.Time)
}
