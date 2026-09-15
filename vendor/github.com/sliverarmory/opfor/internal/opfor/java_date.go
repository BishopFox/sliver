package opfor

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/sliverarmory/opfor/internal/lexer"
)

// javaDateToken is one SimpleDateFormat pattern field or literal. OPFOR uses
// an explicit formatter/parser instead of time.Format so Java-only calendar
// fields and SimpleDateFormat's lenient prefix parsing remain representable.
type javaDateToken struct {
	field   byte
	count   int
	literal string
}

// javaDatePatternError is the IllegalArgumentException raised by
// SimpleDateFormat construction. Keeping it distinct lets parseDate preserve
// the separate null-result path used for syntactically valid patterns whose
// input cannot be parsed.
type javaDatePatternError struct {
	message string
}

func (err *javaDatePatternError) Error() string {
	if err == nil {
		return ""
	}
	return err.message
}

func tokenizeJavaDatePattern(pattern string) ([]javaDateToken, error) {
	var tokens []javaDateToken
	var literal strings.Builder
	flushLiteral := func() {
		if literal.Len() == 0 {
			return
		}
		tokens = append(tokens, javaDateToken{literal: literal.String()})
		literal.Reset()
	}
	quoted := false
	for index := 0; index < len(pattern); {
		if pattern[index] == '\'' {
			if index+1 < len(pattern) && pattern[index+1] == '\'' {
				literal.WriteByte('\'')
				index += 2
				continue
			}
			quoted = !quoted
			index++
			continue
		}
		if quoted || !asciiLetter(pattern[index]) {
			literal.WriteByte(pattern[index])
			index++
			continue
		}
		flushLiteral()
		start := index
		for index < len(pattern) && pattern[index] == pattern[start] {
			index++
		}
		field := pattern[start]
		if !strings.ContainsRune("GyYMLwWDdFEuaHkKhmsSzZX", rune(field)) {
			return nil, &javaDatePatternError{message: fmt.Sprintf("Illegal pattern character '%c'", field)}
		}
		count := index - start
		if field == 'X' && count > 3 {
			return nil, &javaDatePatternError{message: fmt.Sprintf("invalid ISO 8601 format: length=%d", count)}
		}
		tokens = append(tokens, javaDateToken{field: field, count: count})
	}
	if quoted {
		return nil, &javaDatePatternError{message: "Unterminated quote"}
	}
	flushLiteral()
	return tokens, nil
}

func formatJavaDate(instant time.Time, pattern string) (string, error) {
	tokens, err := tokenizeJavaDatePattern(pattern)
	if err != nil {
		return "", err
	}
	var output strings.Builder
	for _, token := range tokens {
		if token.field == 0 {
			output.WriteString(token.literal)
			continue
		}
		output.WriteString(formatJavaDateField(instant, token.field, token.count))
	}
	return output.String(), nil
}

func formatJavaDateField(instant time.Time, field byte, count int) string {
	civil := javaHybridCivilDateForTime(instant)
	year, month, day := civil.year, time.Month(civil.month), civil.day
	yearOfEra := year
	if yearOfEra <= 0 {
		yearOfEra = 1 - yearOfEra
	}
	hour := instant.Hour()
	switch field {
	case 'G':
		if year <= 0 {
			return "BC"
		}
		return "AD"
	case 'y':
		if count == 2 {
			return fmt.Sprintf("%02d", yearOfEra%100)
		}
		return padJavaNumber(yearOfEra, count)
	case 'Y':
		weekYear, _ := javaUSWeek(instant)
		if count == 2 {
			return fmt.Sprintf("%02d", positiveMod(weekYear, 100))
		}
		return padJavaNumber(weekYear, count)
	case 'M', 'L':
		if count >= 4 {
			return month.String()
		}
		if count == 3 {
			return month.String()[:3]
		}
		return padJavaNumber(int(month), count)
	case 'w':
		_, week := javaUSWeek(instant)
		return padJavaNumber(week, count)
	case 'W':
		return padJavaNumber(javaUSWeekOfMonth(instant), count)
	case 'D':
		return padJavaNumber(javaHybridYearDay(civil), count)
	case 'd':
		return padJavaNumber(day, count)
	case 'F':
		return padJavaNumber(javaHybridDayOfWeekInMonth(civil), count)
	case 'E':
		name := instant.Weekday().String()
		if count >= 4 {
			return name
		}
		return name[:3]
	case 'u':
		weekday := int(instant.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		return padJavaNumber(weekday, count)
	case 'a':
		if hour < 12 {
			return "AM"
		}
		return "PM"
	case 'H':
		return padJavaNumber(hour, count)
	case 'k':
		if hour == 0 {
			hour = 24
		}
		return padJavaNumber(hour, count)
	case 'K':
		return padJavaNumber(hour%12, count)
	case 'h':
		hour %= 12
		if hour == 0 {
			hour = 12
		}
		return padJavaNumber(hour, count)
	case 'm':
		return padJavaNumber(instant.Minute(), count)
	case 's':
		return padJavaNumber(instant.Second(), count)
	case 'S':
		return padJavaNumber(instant.Nanosecond()/int(time.Millisecond), count)
	case 'z':
		name, offset := instant.Zone()
		if count >= 4 {
			if longName := javaLongTimeZoneName(instant, name); longName != "" {
				return longName
			}
		}
		if name != "" {
			return name
		}
		return javaGMTOffset(offset, true)
	case 'Z':
		_, offset := instant.Zone()
		return javaNumericOffset(offset, false)
	case 'X':
		_, offset := instant.Zone()
		if offset == 0 {
			return "Z"
		}
		switch count {
		case 1:
			return javaNumericOffset(offset, false)[:3]
		case 2:
			return javaNumericOffset(offset, false)
		default:
			return javaNumericOffset(offset, true)
		}
	}
	return ""
}

func padJavaNumber(value, width int) string {
	if value < 0 {
		magnitude := strconv.FormatInt(-int64(value), 10)
		if width > len(magnitude) {
			magnitude = strings.Repeat("0", width-len(magnitude)) + magnitude
		}
		return "-" + magnitude
	}
	text := strconv.Itoa(value)
	if width <= len(text) {
		return text
	}
	return strings.Repeat("0", width-len(text)) + text
}

func positiveMod(value, divisor int) int {
	result := value % divisor
	if result < 0 {
		result += divisor
	}
	return result
}

func javaNumericOffset(offset int, colon bool) string {
	sign := byte('+')
	if offset < 0 {
		sign = '-'
		offset = -offset
	}
	hours := offset / 3600
	minutes := offset % 3600 / 60
	if colon {
		return fmt.Sprintf("%c%02d:%02d", sign, hours, minutes)
	}
	return fmt.Sprintf("%c%02d%02d", sign, hours, minutes)
}

func javaGMTOffset(offset int, colon bool) string {
	return "GMT" + javaNumericOffset(offset, colon)
}

// javaLongTimeZoneName supplies the Locale.US display names emitted by
// SimpleDateFormat for the common North American abbreviations returned by
// Go's IANA locations. Unknown abbreviations retain their short form.
func javaLongTimeZoneName(instant time.Time, name string) string {
	_, offset := instant.Zone()
	return javaLongTimeZoneNameForLocation(instant.Location().String(), name, offset)
}

func javaLongTimeZoneNameForLocation(locationName, abbreviation string, offset int) string {
	switch locationName {
	case "UTC":
		return "Coordinated Universal Time"
	case "GMT":
		return "Greenwich Mean Time"
	case "America/New_York", "US/Eastern", "EST5EDT":
		if abbreviation == "EDT" {
			return "Eastern Daylight Time"
		}
		return "Eastern Standard Time"
	case "America/Chicago", "US/Central", "CST6CDT":
		if abbreviation == "CDT" {
			return "Central Daylight Time"
		}
		return "Central Standard Time"
	case "America/Denver", "US/Mountain", "MST7MDT":
		if abbreviation == "MDT" {
			return "Mountain Daylight Time"
		}
		return "Mountain Standard Time"
	case "America/Los_Angeles", "US/Pacific", "PST8PDT":
		if abbreviation == "PDT" {
			return "Pacific Daylight Time"
		}
		return "Pacific Standard Time"
	case "America/Anchorage", "US/Alaska":
		if abbreviation == "AKDT" {
			return "Alaska Daylight Time"
		}
		return "Alaska Standard Time"
	case "Pacific/Honolulu", "US/Hawaii":
		return "Hawaii Standard Time"
	case "Local":
		// time.Local deliberately hides its backing IANA identifier. Pair the
		// abbreviation with its canonical US offset so an ambiguous positive-
		// offset name such as Asia/Shanghai's CST is not mislabeled.
		switch {
		case abbreviation == "EST" && offset == -5*60*60:
			return "Eastern Standard Time"
		case abbreviation == "EDT" && offset == -4*60*60:
			return "Eastern Daylight Time"
		case abbreviation == "CST" && offset == -6*60*60:
			return "Central Standard Time"
		case abbreviation == "CDT" && offset == -5*60*60:
			return "Central Daylight Time"
		case abbreviation == "MST" && offset == -7*60*60:
			return "Mountain Standard Time"
		case abbreviation == "MDT" && offset == -6*60*60:
			return "Mountain Daylight Time"
		case abbreviation == "PST" && offset == -8*60*60:
			return "Pacific Standard Time"
		case abbreviation == "PDT" && offset == -7*60*60:
			return "Pacific Daylight Time"
		case abbreviation == "AKST" && offset == -9*60*60:
			return "Alaska Standard Time"
		case abbreviation == "AKDT" && offset == -8*60*60:
			return "Alaska Daylight Time"
		case abbreviation == "HST" && offset == -10*60*60:
			return "Hawaii Standard Time"
		default:
			return ""
		}
	default:
		return ""
	}
}

// javaUSWeek reproduces Calendar's en-US defaults: Sunday is the first day
// and one day is sufficient for week one. A boundary Sunday can therefore be
// week one of the following week-year.
func javaUSWeek(instant time.Time) (int, int) {
	civil := javaHybridCivilDateForTime(instant)
	fixed := javaHybridFixedDay(civil.year, civil.month, civil.day)
	weekStart := fixed - int64(javaWeekdayFromFixedDay(fixed))
	year := civil.year
	start := javaUSWeekYearStartFixed(year)
	if weekStart < start {
		year--
		start = javaUSWeekYearStartFixed(year)
	} else if next := javaUSWeekYearStartFixed(year + 1); weekStart >= next {
		year++
		start = next
	}
	return year, int((weekStart-start)/7) + 1
}

func javaUSWeekOfMonth(instant time.Time) int {
	civil := javaHybridCivilDateForTime(instant)
	fixed := javaHybridFixedDay(civil.year, civil.month, civil.day)
	first := javaHybridFixedDay(civil.year, civil.month, 1)
	start := first - int64(javaWeekdayFromFixedDay(first))
	weekStart := fixed - int64(javaWeekdayFromFixedDay(fixed))
	return int((weekStart-start)/7) + 1
}

func javaUSWeekYearStartFixed(year int) int64 {
	first := javaHybridFixedDay(year, 1, 1)
	return first - int64(javaWeekdayFromFixedDay(first))
}

// javaCivilDate is an astronomical-year civil date: year 0 is 1 BC, year -1
// is 2 BC. GregorianCalendar uses the Julian calendar before its default
// 1582-10-15 Gregorian cutover, while Go's time package is proleptic
// Gregorian. Fixed days let the shim preserve the Java calendar fields while
// still using time.Time for instants and time-zone rules.
type javaCivilDate struct {
	year  int
	month int
	day   int
}

const javaGregorianCutoverFixedDay int64 = -141427 // 1582-10-15 Gregorian / 1582-10-05 Julian.

func javaHybridCivilDateForTime(instant time.Time) javaCivilDate {
	year, month, day := instant.Date()
	fixed := javaGregorianFixedDay(year, int(month), day)
	return javaHybridCivilDateFromFixed(fixed)
}

func javaHybridFixedDay(year, month, day int) int64 {
	year, month = javaNormalizeYearMonth(year, month)
	if year > 1582 || year == 1582 && (month > 10 || month == 10 && day >= 15) {
		return javaGregorianFixedDay(year, month, day)
	}
	return javaJulianFixedDay(year, month, day)
}

func javaHybridCivilDateFromFixed(fixed int64) javaCivilDate {
	if fixed >= javaGregorianCutoverFixedDay {
		return javaGregorianCivilDateFromFixed(fixed)
	}
	return javaJulianCivilDateFromFixed(fixed)
}

func javaTimeAtFixedDay(fixed int64, location *time.Location) time.Time {
	civil := javaGregorianCivilDateFromFixed(fixed)
	return time.Date(civil.year, time.Month(civil.month), civil.day, 0, 0, 0, 0, location)
}

func javaNormalizeYearMonth(year, month int) (int, int) {
	monthIndex := int64(month) - 1
	yearOffset := javaFloorDiv(monthIndex, 12)
	return year + int(yearOffset), int(monthIndex-yearOffset*12) + 1
}

func javaGregorianFixedDay(year, month, day int) int64 {
	year, month = javaNormalizeYearMonth(year, month)
	y := int64(year)
	if month <= 2 {
		y--
	}
	era := javaFloorDiv(y, 400)
	yearOfEra := y - era*400
	shiftedMonth := int64(month)
	if month > 2 {
		shiftedMonth -= 3
	} else {
		shiftedMonth += 9
	}
	dayOfYear := (153*shiftedMonth+2)/5 + int64(day) - 1
	dayOfEra := yearOfEra*365 + yearOfEra/4 - yearOfEra/100 + dayOfYear
	return era*146097 + dayOfEra - 719468
}

func javaGregorianCivilDateFromFixed(fixed int64) javaCivilDate {
	shifted := fixed + 719468
	era := javaFloorDiv(shifted, 146097)
	dayOfEra := shifted - era*146097
	yearOfEra := (dayOfEra - dayOfEra/1460 + dayOfEra/36524 - dayOfEra/146096) / 365
	year := yearOfEra + era*400
	dayOfYear := dayOfEra - (365*yearOfEra + yearOfEra/4 - yearOfEra/100)
	monthPart := (5*dayOfYear + 2) / 153
	day := dayOfYear - (153*monthPart+2)/5 + 1
	month := monthPart + 3
	if monthPart >= 10 {
		month -= 12
		year++
	}
	return javaCivilDate{year: int(year), month: int(month), day: int(day)}
}

func javaJulianFixedDay(year, month, day int) int64 {
	year, month = javaNormalizeYearMonth(year, month)
	y := int64(year)
	days := 365*y + javaFloorDiv(y+3, 4)
	monthDays := [...]int64{0, 31, 59, 90, 120, 151, 181, 212, 243, 273, 304, 334}
	days += monthDays[month-1]
	if month > 2 && positiveMod(year, 4) == 0 {
		days++
	}
	return days + int64(day) - 1 - 719530
}

func javaJulianCivilDateFromFixed(fixed int64) javaCivilDate {
	shifted := fixed + 719530
	era := javaFloorDiv(shifted, 1461)
	dayOfEra := shifted - era*1461
	yearOfEra := int64(0)
	dayOfYear := dayOfEra
	if dayOfEra >= 366 {
		yearOfEra = 1 + (dayOfEra-366)/365
		dayOfYear = dayOfEra - 366 - (yearOfEra-1)*365
	}
	year := era*4 + yearOfEra
	monthLengths := [...]int64{31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
	if year%4 == 0 {
		monthLengths[1] = 29
	}
	month := 1
	for dayOfYear >= monthLengths[month-1] {
		dayOfYear -= monthLengths[month-1]
		month++
	}
	return javaCivilDate{year: int(year), month: month, day: int(dayOfYear) + 1}
}

func javaFloorDiv(dividend, divisor int64) int64 {
	quotient := dividend / divisor
	if dividend%divisor < 0 {
		quotient--
	}
	return quotient
}

func javaWeekdayFromFixedDay(fixed int64) int {
	return int(positiveMod64(fixed+4, 7))
}

func positiveMod64(value, divisor int64) int64 {
	result := value % divisor
	if result < 0 {
		result += divisor
	}
	return result
}

func javaHybridYearDay(civil javaCivilDate) int {
	fixed := javaHybridFixedDay(civil.year, civil.month, civil.day)
	return int(fixed-javaHybridFixedDay(civil.year, 1, 1)) + 1
}

func javaHybridDayOfWeekInMonth(civil javaCivilDate) int {
	fixed := javaHybridFixedDay(civil.year, civil.month, civil.day)
	first := javaHybridFixedDay(civil.year, civil.month, 1)
	return int((fixed-first)/7) + 1
}

type javaDateParseState struct {
	era, year, weekYear, month, day, dayOfYear, weekOfYear, weekOfMonth, dayInMonth int
	weekday, hour24, hour12, minute, second, millisecond, ampm                      int
	hasEra, hasYear, hasWeekYear, hasMonth, hasDay, hasDayOfYear                    bool
	hasWeekOfYear, hasWeekOfMonth, hasDayInMonth, hasWeekday                        bool
	hasHour24, hasHour12, hasAMPM                                                   bool
	twoDigitYear, twoDigitWeekYear                                                  bool
	isoWeekday, yearStamp, weekYearStamp, nextStamp                                 int
	hasISOWeekday                                                                   bool
	location                                                                        *time.Location
}

func parseJavaDate(input, pattern string, now time.Time) (time.Time, error) {
	tokens, err := tokenizeJavaDatePattern(pattern)
	if err != nil {
		return time.Time{}, err
	}
	state := javaDateParseState{
		era: 1, year: 1970, weekYear: 1970, month: 1, day: 1,
		location: now.Location(),
	}
	position := 0
	for index, token := range tokens {
		if token.field == 0 {
			if !strings.HasPrefix(input[position:], token.literal) {
				return time.Time{}, fmt.Errorf("date does not match literal %q at offset %d", token.literal, position)
			}
			position += len(token.literal)
			continue
		}
		next := javaDateFollowingLiteral(tokens, index+1)
		adjacentNumeric := index+1 < len(tokens) && tokens[index+1].field != 0
		leadingSpace := javaDateLeadingFieldSpace(input[position:])
		consumed, parseErr := parseJavaDateField(input[position+leadingSpace:], token, next, adjacentNumeric, now, &state)
		if parseErr != nil {
			return time.Time{}, fmt.Errorf("field %s at offset %d: %w", strings.Repeat(string(token.field), token.count), position, parseErr)
		}
		position += leadingSpace + consumed
	}
	return state.time(now), nil
}

func javaDateLeadingFieldSpace(input string) int {
	position := 0
	for position < len(input) && (input[position] == ' ' || input[position] == '\t') {
		position++
	}
	return position
}

func javaDateFollowingLiteral(tokens []javaDateToken, start int) string {
	for index := start; index < len(tokens); index++ {
		if tokens[index].field == 0 {
			if tokens[index].literal != "" {
				return tokens[index].literal
			}
			continue
		}
		break
	}
	return ""
}

func parseJavaDateField(input string, token javaDateToken, followingLiteral string, adjacentNumeric bool, now time.Time, state *javaDateParseState) (int, error) {
	if input == "" {
		return 0, fmt.Errorf("missing value")
	}
	switch token.field {
	case 'G':
		if width := matchJavaText(input, []string{"AD", "Anno Domini"}); width != 0 {
			state.era, state.hasEra = 1, true
			return width, nil
		}
		if width := matchJavaText(input, []string{"BC", "Before Christ"}); width != 0 {
			state.era, state.hasEra = 0, true
			return width, nil
		}
		return 0, fmt.Errorf("expected era")
	case 'M', 'L':
		if token.count >= 3 {
			month, width := matchJavaMonth(input)
			if width == 0 {
				return 0, fmt.Errorf("expected month name")
			}
			state.month, state.hasMonth = month, true
			return width, nil
		}
	case 'E':
		weekday, width := matchJavaWeekday(input)
		if width == 0 {
			return 0, fmt.Errorf("expected weekday name")
		}
		state.weekday, state.hasWeekday = weekday, true
		state.hasISOWeekday = false
		return width, nil
	case 'a':
		if len(input) >= 2 && strings.EqualFold(input[:2], "AM") {
			state.ampm, state.hasAMPM = 0, true
			return 2, nil
		}
		if len(input) >= 2 && strings.EqualFold(input[:2], "PM") {
			state.ampm, state.hasAMPM = 1, true
			return 2, nil
		}
		return 0, fmt.Errorf("expected AM or PM")
	case 'z':
		width := len(input)
		if followingLiteral != "" {
			if offset := strings.Index(input, followingLiteral); offset >= 0 {
				width = offset
			}
		}
		location, zoneErr := parseJavaTimeZone(input[:width])
		if zoneErr != nil {
			return 0, zoneErr
		}
		state.location = location
		return width, nil
	case 'Z', 'X':
		location, width, zoneErr := parseJavaNumericTimeZone(input, token.field, token.count)
		if zoneErr != nil {
			return 0, zoneErr
		}
		state.location = location
		return width, nil
	}

	maxDigits := 0
	if adjacentNumeric {
		maxDigits = token.count
	}
	value, width, err := parseJavaNumber(input, maxDigits)
	if err != nil {
		return 0, err
	}
	switch token.field {
	case 'y':
		if token.count <= 2 && javaYearIsExactlyTwoDigits(input, width) {
			value = javaTwoDigitYear(value, now)
			state.twoDigitYear = true
		}
		state.year, state.hasYear = value, true
		state.nextStamp++
		state.yearStamp = state.nextStamp
	case 'Y':
		if token.count <= 2 && javaYearIsExactlyTwoDigits(input, width) {
			value = javaTwoDigitYear(value, now)
			state.twoDigitWeekYear = true
		}
		state.weekYear, state.hasWeekYear = value, true
		state.nextStamp++
		state.weekYearStamp = state.nextStamp
	case 'M', 'L':
		state.month, state.hasMonth = value, true
	case 'w':
		state.weekOfYear, state.hasWeekOfYear = value, true
	case 'W':
		state.weekOfMonth, state.hasWeekOfMonth = value, true
	case 'D':
		state.dayOfYear, state.hasDayOfYear = value, true
	case 'd':
		state.day, state.hasDay = value, true
	case 'F':
		state.dayInMonth, state.hasDayInMonth = value, true
	case 'u':
		state.isoWeekday, state.hasISOWeekday = value, true
		state.weekday, state.hasWeekday = javaStandaloneISOWeekday(value), true
	case 'H':
		state.hour24, state.hasHour24 = value, true
	case 'k':
		if value == 24 {
			value = 0
		}
		state.hour24, state.hasHour24 = value, true
	case 'K':
		state.hour12, state.hasHour12 = value, true
	case 'h':
		if value == 12 {
			value = 0
		}
		state.hour12, state.hasHour12 = value, true
	case 'm':
		state.minute = value
	case 's':
		state.second = value
	case 'S':
		state.millisecond = value
	default:
		return 0, fmt.Errorf("unsupported numeric field")
	}
	return width, nil
}

func parseJavaNumber(input string, maxUnits int) (int, int, error) {
	position, units := 0, 0
	negative := false
	if value, width := utf8.DecodeRuneInString(input); javaUSLenientMinusSign(value) {
		negative = true
		position, units = width, 1
	}
	digits := 0
	magnitude := uint64(0)
	overflow := false
	for position < len(input) {
		value, width := utf8.DecodeRuneInString(input[position:])
		if value == utf8.RuneError && width == 1 || value > '\uFFFF' || maxUnits > 0 && units+1 > maxUnits {
			break
		}
		digit, ok := javaBMPDecimalDigit(value)
		if !ok {
			break
		}
		if magnitude > (^uint64(0)-uint64(digit))/10 {
			overflow = true
		} else if !overflow {
			magnitude = magnitude*10 + uint64(digit)
		}
		position += width
		units++
		digits++
	}
	if digits == 0 {
		return 0, 0, fmt.Errorf("expected number")
	}
	const maxInt64 = uint64(^uint64(0) >> 1)
	if overflow || !negative && magnitude > maxInt64 || negative && magnitude > maxInt64+1 {
		if negative {
			return int(int32(-1 << 31)), position, nil
		}
		return int(int32(1<<31 - 1)), position, nil
	}
	var signedValue int64
	if negative {
		if magnitude == maxInt64+1 {
			signedValue = -1 << 63
		} else {
			signedValue = -int64(magnitude)
		}
	} else {
		signedValue = int64(magnitude)
	}
	return int(int32(signedValue)), position, nil
}

func javaUSLenientMinusSign(value rune) bool {
	switch value {
	case '-', '\u2010', '\u2011', '\u2012', '\u2013', '\u207B', '\u208B', '\u2212', '\u2796', '\uFE63', '\uFF0D':
		return true
	default:
		return false
	}
}

func javaBMPDecimalDigit(value rune) (int, bool) {
	digit := lexer.JavaDigit(value, 10)
	return digit, digit >= 0
}

func javaYearIsExactlyTwoDigits(input string, width int) bool {
	if width <= 0 || width > len(input) {
		return false
	}
	count := 0
	for _, value := range input[:width] {
		if _, ok := javaBMPDecimalDigit(value); !ok {
			return false
		}
		count++
	}
	return count == 2
}

func javaStandaloneISOWeekday(value int) int {
	if value >= 1 && value <= 7 {
		return value % 7
	}
	return positiveMod(value-1, 7)
}

func matchJavaText(input string, candidates []string) int {
	best := 0
	for _, candidate := range candidates {
		if len(candidate) > best && len(input) >= len(candidate) && strings.EqualFold(input[:len(candidate)], candidate) {
			best = len(candidate)
		}
	}
	return best
}

func matchJavaMonth(input string) (int, int) {
	bestMonth, bestWidth := 0, 0
	for month := time.January; month <= time.December; month++ {
		name := month.String()
		for _, candidate := range []string{name, name[:3]} {
			if len(candidate) > bestWidth && len(input) >= len(candidate) && strings.EqualFold(input[:len(candidate)], candidate) {
				bestMonth, bestWidth = int(month), len(candidate)
			}
		}
	}
	return bestMonth, bestWidth
}

func matchJavaWeekday(input string) (int, int) {
	bestDay, bestWidth := 0, 0
	for day := time.Sunday; day <= time.Saturday; day++ {
		name := day.String()
		for _, candidate := range []string{name, name[:3]} {
			if len(candidate) > bestWidth && len(input) >= len(candidate) && strings.EqualFold(input[:len(candidate)], candidate) {
				bestDay, bestWidth = int(day), len(candidate)
			}
		}
	}
	return bestDay, bestWidth
}

func parseJavaTimeZone(text string) (*time.Location, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("expected time zone")
	}
	upper := strings.ToUpper(text)
	if upper == "UTC" || upper == "GMT" || upper == "UT" {
		return time.UTC, nil
	}
	if offset, ok := map[string]int{
		"EST": -5 * 3600, "EDT": -4 * 3600,
		"CST": -6 * 3600, "CDT": -5 * 3600,
		"MST": -7 * 3600, "MDT": -6 * 3600,
		"PST": -8 * 3600, "PDT": -7 * 3600,
	}[upper]; ok {
		return time.FixedZone(upper, offset), nil
	}
	if strings.HasPrefix(upper, "GMT+") || strings.HasPrefix(upper, "GMT-") {
		location, _, err := parseJavaFlexibleNumericTimeZone(text[3:])
		return location, err
	}
	location, err := time.LoadLocation(text)
	if err != nil {
		return nil, fmt.Errorf("unknown time zone %q", text)
	}
	return location, nil
}

func parseJavaNumericTimeZone(input string, field byte, count int) (*time.Location, int, error) {
	if field == 'X' && strings.HasPrefix(input, "Z") {
		return time.UTC, 1, nil
	}
	width, colon := 0, false
	switch {
	case field == 'X' && count == 1:
		width = 3
	case field == 'X' && count == 2:
		width = 5
	case field == 'X' && count == 3:
		width, colon = 6, true
	case field == 'Z':
		width = 5
	default:
		return nil, 0, fmt.Errorf("unsupported numeric time-zone pattern")
	}
	if len(input) < width || input[0] != '+' && input[0] != '-' {
		return nil, 0, fmt.Errorf("expected numeric time zone")
	}
	if input[1] < '0' || input[1] > '9' || input[2] < '0' || input[2] > '9' {
		return nil, 0, fmt.Errorf("invalid time-zone hour")
	}
	sign := 1
	if input[0] == '-' {
		sign = -1
	}
	hour, err := strconv.Atoi(input[1:3])
	if err != nil {
		return nil, 0, fmt.Errorf("invalid time-zone hour")
	}
	minute := 0
	if width > 3 {
		minuteStart := 3
		if colon {
			if input[3] != ':' {
				return nil, 0, fmt.Errorf("expected colon in numeric time zone")
			}
			minuteStart = 4
		}
		if input[minuteStart] < '0' || input[minuteStart] > '9' || input[minuteStart+1] < '0' || input[minuteStart+1] > '9' {
			return nil, 0, fmt.Errorf("invalid time-zone minute")
		}
		minute, err = strconv.Atoi(input[minuteStart : minuteStart+2])
	}
	if err != nil || hour > 23 || minute > 59 {
		return nil, 0, fmt.Errorf("invalid numeric time zone")
	}
	offset := sign * (hour*3600 + minute*60)
	return time.FixedZone(javaNumericOffset(offset, true), offset), width, nil
}

// parseJavaFlexibleNumericTimeZone retains the general GMT offset shapes used
// by the z field. X and Z fields use parseJavaNumericTimeZone's strict widths.
func parseJavaFlexibleNumericTimeZone(input string) (*time.Location, int, error) {
	if len(input) < 3 || input[0] != '+' && input[0] != '-' {
		return nil, 0, fmt.Errorf("expected numeric time zone")
	}
	sign := 1
	if input[0] == '-' {
		sign = -1
	}
	hour, err := strconv.Atoi(input[1:3])
	if err != nil {
		return nil, 0, fmt.Errorf("invalid time-zone hour")
	}
	minute, width := 0, 3
	if len(input) >= 6 && input[3] == ':' {
		minute, err = strconv.Atoi(input[4:6])
		width = 6
	} else if len(input) >= 5 && input[3] >= '0' && input[3] <= '9' {
		minute, err = strconv.Atoi(input[3:5])
		width = 5
	}
	if err != nil || hour > 23 || minute > 59 {
		return nil, 0, fmt.Errorf("invalid numeric time zone")
	}
	offset := sign * (hour*3600 + minute*60)
	return time.FixedZone(javaNumericOffset(offset, true), offset), width, nil
}

func javaTwoDigitYear(value int, now time.Time) int {
	pivot := now.Year() - 80
	century := pivot - positiveMod(pivot, 100)
	result := century + positiveMod(value, 100)
	if result < pivot {
		result += 100
	}
	return result
}

func (state javaDateParseState) time(now time.Time) time.Time {
	instant := state.unadjustedTime(now)
	if (state.twoDigitYear || state.twoDigitWeekYear) && instant.Before(now.AddDate(-80, 0, 0)) {
		if state.twoDigitYear {
			state.year += 100
		}
		if state.twoDigitWeekYear {
			state.weekYear += 100
		}
		instant = state.unadjustedTime(now)
	}
	return instant
}

func (state javaDateParseState) unadjustedTime(now time.Time) time.Time {
	year := state.year
	if state.hasEra && state.era == 0 {
		year = 1 - year
	}
	hour := state.hour24
	if state.hasHour12 {
		hour = state.hour12
		if state.hasAMPM && state.ampm != 0 {
			hour += 12
		}
	} else if state.hasAMPM && state.ampm != 0 {
		hour = 12
	}
	location := state.location
	if location == nil {
		location = now.Location()
	}
	var date time.Time
	useWeekYear := state.hasWeekYear && (!state.hasYear || state.weekYearStamp > state.yearStamp)
	switch {
	case useWeekYear:
		week, weekday := 1, state.weekday
		if state.hasWeekOfYear {
			week = state.weekOfYear
		}
		if state.hasISOWeekday {
			week, weekday = javaISOWeekDateFields(week, state.isoWeekday)
		}
		date = javaTimeAtFixedDay(javaUSWeekYearStartFixed(state.weekYear)+int64((week-1)*7+weekday), location)
	case state.hasWeekOfYear:
		date = javaTimeAtFixedDay(javaUSWeekYearStartFixed(year)+int64((state.weekOfYear-1)*7+state.weekday), location)
	case state.hasDayOfYear:
		date = javaTimeAtFixedDay(javaHybridFixedDay(year, 1, 1)+int64(state.dayOfYear-1), location)
	case state.hasWeekOfMonth:
		first := javaHybridFixedDay(year, state.month, 1)
		start := first - int64(javaWeekdayFromFixedDay(first))
		date = javaTimeAtFixedDay(start+int64((state.weekOfMonth-1)*7+state.weekday), location)
	case state.hasDayInMonth || state.hasWeekday && !state.hasDay:
		occurrence := state.dayInMonth
		if !state.hasDayInMonth {
			occurrence = 1
		}
		date = javaUSDayOfWeekInMonth(year, state.month, occurrence, state.weekday, location)
	default:
		date = javaTimeAtFixedDay(javaHybridFixedDay(year, state.month, state.day), location)
	}
	return time.Date(date.Year(), date.Month(), date.Day(), hour, state.minute, state.second, state.millisecond*int(time.Millisecond), location)
}

func javaISOWeekDateFields(week, isoWeekday int) (int, int) {
	if isoWeekday >= 1 && isoWeekday <= 7 {
		return week, isoWeekday % 7
	}
	if isoWeekday >= 8 {
		adjusted := isoWeekday - 1
		week += adjusted / 7
		isoWeekday = adjusted%7 + 1
	} else {
		for isoWeekday <= 0 {
			isoWeekday += 7
			week--
		}
	}
	return week, isoWeekday % 7
}

func javaUSDayOfWeekInMonth(year, month, occurrence, weekday int, location *time.Location) time.Time {
	if occurrence >= 0 {
		first := javaHybridFixedDay(year, month, 1)
		offset := positiveMod(weekday-javaWeekdayFromFixedDay(first), 7)
		return javaTimeAtFixedDay(first+int64(offset+(occurrence-1)*7), location)
	}
	last := javaHybridFixedDay(year, month+1, 1) - 1
	offset := positiveMod(javaWeekdayFromFixedDay(last)-weekday, 7)
	return javaTimeAtFixedDay(last+int64(-offset+(occurrence+1)*7), location)
}
