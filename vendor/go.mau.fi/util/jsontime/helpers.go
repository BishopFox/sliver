// Copyright (c) 2023 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package jsontime

import (
	"time"
)

func zeroSafeUnixToTime(val int64, fn func(int64) time.Time) time.Time {
	if val == 0 {
		return time.Time{}
	}
	return fn(val)
}

func UM(time time.Time) UnixMilli {
	return UnixMilli{Time: time}
}

func UMInt(ts int64) UnixMilli {
	return UM(zeroSafeUnixToTime(ts, time.UnixMilli))
}

func UnixMilliNow() UnixMilli {
	return UM(time.Now())
}

func UMicro(time time.Time) UnixMicro {
	return UnixMicro{Time: time}
}

func UMicroInt(ts int64) UnixMicro {
	return UMicro(zeroSafeUnixToTime(ts, time.UnixMicro))
}

func UnixMicroNow() UnixMicro {
	return UMicro(time.Now())
}

func UN(time time.Time) UnixNano {
	return UnixNano{Time: time}
}

func UNInt(ts int64) UnixNano {
	return UN(zeroSafeUnixToTime(ts, func(i int64) time.Time {
		return time.Unix(0, i)
	}))
}

func UnixNanoNow() UnixNano {
	return UN(time.Now())
}

func U(time time.Time) Unix {
	return Unix{Time: time}
}

func UInt(ts int64) Unix {
	return U(zeroSafeUnixToTime(ts, func(i int64) time.Time {
		return time.Unix(i, 0)
	}))
}

func UnixNow() Unix {
	return U(time.Now())
}
