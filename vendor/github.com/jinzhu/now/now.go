package now

import (
	"errors"
	"regexp"
	"time"
)

// BeginningOfMinute beginning of minute
func (now *Now) BeginningOfMinute() time.Time {
	return now.Truncate(time.Minute)
}

// BeginningOfHour beginning of hour
func (now *Now) BeginningOfHour() time.Time {
	y, m, d := now.Date()
	return time.Date(y, m, d, now.Time.Hour(), 0, 0, 0, now.Time.Location())
}

// BeginningOfDay beginning of day
func (now *Now) BeginningOfDay() time.Time {
	y, m, d := now.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, now.Time.Location())
}

// BeginningOfWeek beginning of week
func (now *Now) BeginningOfWeek() time.Time {
	t := now.BeginningOfDay()
	weekday := int(t.Weekday())

	if now.WeekStartDay != time.Sunday {
		weekStartDayInt := int(now.WeekStartDay)

		if weekday < weekStartDayInt {
			weekday = weekday + 7 - weekStartDayInt
		} else {
			weekday = weekday - weekStartDayInt
		}
	}
	return t.AddDate(0, 0, -weekday)
}

// BeginningOfMonth beginning of month
func (now *Now) BeginningOfMonth() time.Time {
	y, m, _ := now.Date()
	return time.Date(y, m, 1, 0, 0, 0, 0, now.Location())
}

// BeginningOfQuarter beginning of quarter
func (now *Now) BeginningOfQuarter() time.Time {
	month := now.BeginningOfMonth()
	offset := (int(month.Month()) - 1) % 3
	return month.AddDate(0, -offset, 0)
}

// BeginningOfHalf beginning of half year
func (now *Now) BeginningOfHalf() time.Time {
	month := now.BeginningOfMonth()
	offset := (int(month.Month()) - 1) % 6
	return month.AddDate(0, -offset, 0)
}

// BeginningOfYear BeginningOfYear beginning of year
func (now *Now) BeginningOfYear() time.Time {
	y, _, _ := now.Date()
	return time.Date(y, time.January, 1, 0, 0, 0, 0, now.Location())
}

// EndOfMinute end of minute
func (now *Now) EndOfMinute() time.Time {
	return now.BeginningOfMinute().Add(time.Minute - time.Nanosecond)
}

// EndOfHour end of hour
func (now *Now) EndOfHour() time.Time {
	return now.BeginningOfHour().Add(time.Hour - time.Nanosecond)
}

// EndOfDay end of day
func (now *Now) EndOfDay() time.Time {
	y, m, d := now.Date()
	return time.Date(y, m, d, 23, 59, 59, int(time.Second-time.Nanosecond), now.Location())
}

// EndOfWeek end of week
func (now *Now) EndOfWeek() time.Time {
	return now.BeginningOfWeek().AddDate(0, 0, 7).Add(-time.Nanosecond)
}

// EndOfMonth end of month
func (now *Now) EndOfMonth() time.Time {
	return now.BeginningOfMonth().AddDate(0, 1, 0).Add(-time.Nanosecond)
}

// EndOfQuarter end of quarter
func (now *Now) EndOfQuarter() time.Time {
	return now.BeginningOfQuarter().AddDate(0, 3, 0).Add(-time.Nanosecond)
}

// EndOfHalf end of half year
func (now *Now) EndOfHalf() time.Time {
	return now.BeginningOfHalf().AddDate(0, 6, 0).Add(-time.Nanosecond)
}

// EndOfYear end of year
func (now *Now) EndOfYear() time.Time {
	return now.BeginningOfYear().AddDate(1, 0, 0).Add(-time.Nanosecond)
}

// Monday monday
/*
func (now *Now) Monday() time.Time {
	t := now.BeginningOfDay()
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	return t.AddDate(0, 0, -weekday+1)
}
*/

func (now *Now) Monday(strs ...string) time.Time {
	var parseTime time.Time
	var err error
	if len(strs) > 0 {
		parseTime, err = now.Parse(strs...)
		if err != nil {
			panic(err)
		}
	} else {
		parseTime = now.BeginningOfDay()
	}
	weekday := int(parseTime.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	return parseTime.AddDate(0, 0, -weekday+1)
}

func (now *Now) Sunday(strs ...string) time.Time {
	var parseTime time.Time
	var err error
	if len(strs) > 0 {
		parseTime, err = now.Parse(strs...)
		if err != nil {
			panic(err)
		}
	} else {
		parseTime = now.BeginningOfDay()
	}
	weekday := int(parseTime.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	return parseTime.AddDate(0, 0, (7 - weekday))
}

// EndOfSunday end of sunday
func (now *Now) EndOfSunday() time.Time {
	return New(now.Sunday()).EndOfDay()
}

// Quarter returns the yearly quarter
func (now *Now) Quarter() uint {
	return (uint(now.Month())-1)/3 + 1
}

func (now *Now) parseWithFormat(str string, location *time.Location) (t time.Time, err error) {
	for _, format := range now.TimeFormats {
		t, err = time.ParseInLocation(format, str, location)

		if err == nil {
			return
		}
	}
	err = errors.New("Can't parse string as time: " + str)
	return
}

var hasTimeRegexp = regexp.MustCompile(`(\s+|^\s*|T)\d{1,2}((:\d{1,2})*|((:\d{1,2}){2}\.(\d{3}|\d{6}|\d{9})))(\s*$|[Z+-])`) // match 15:04:05, 15:04:05.000, 15:04:05.000000 15, 2017-01-01 15:04, 2021-07-20T00:59:10Z, 2021-07-20T00:59:10+08:00, 2021-07-20T00:00:10-07:00 etc
var onlyTimeRegexp = regexp.MustCompile(`^\s*\d{1,2}((:\d{1,2})*|((:\d{1,2}){2}\.(\d{3}|\d{6}|\d{9})))\s*$`)            // match 15:04:05, 15, 15:04:05.000, 15:04:05.000000, etc

// Parse parse string to time
func (now *Now) Parse(strs ...string) (t time.Time, err error) {
	var (
		setCurrentTime  bool
		parseTime       []int
		currentLocation = now.Location()
		onlyTimeInStr   = true
		currentTime     = formatTimeToList(now.Time)
	)

	for _, str := range strs {
		hasTimeInStr := hasTimeRegexp.MatchString(str) // match 15:04:05, 15
		onlyTimeInStr = hasTimeInStr && onlyTimeInStr && onlyTimeRegexp.MatchString(str)
		if t, err = now.parseWithFormat(str, currentLocation); err == nil {
			location := t.Location()
			parseTime = formatTimeToList(t)

			for i, v := range parseTime {
				// Don't reset hour, minute, second if current time str including time
				if hasTimeInStr && i <= 3 {
					continue
				}

				// If value is zero, replace it with current time
				if v == 0 {
					if setCurrentTime {
						parseTime[i] = currentTime[i]
					}
				} else {
					setCurrentTime = true
				}

				// if current time only includes time, should change day, month to current time
				if onlyTimeInStr {
					if i == 4 || i == 5 {
						parseTime[i] = currentTime[i]
						continue
					}
				}
			}

			t = time.Date(parseTime[6], time.Month(parseTime[5]), parseTime[4], parseTime[3], parseTime[2], parseTime[1], parseTime[0], location)
			currentTime = formatTimeToList(t)
		}
	}
	return
}

// MustParse must parse string to time or it will panic
func (now *Now) MustParse(strs ...string) (t time.Time) {
	t, err := now.Parse(strs...)
	if err != nil {
		panic(err)
	}
	return t
}

// Between check time between the begin, end time or not
func (now *Now) Between(begin, end string) bool {
	beginTime := now.MustParse(begin)
	endTime := now.MustParse(end)
	return now.After(beginTime) && now.Before(endTime)
}
