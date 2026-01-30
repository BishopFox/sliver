package types

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

const Version = "1.2"

// A NullString is a String that may be null. It can be encoded or decoded from
// JSON or the database.
type NullString struct {
	Valid  bool
	String string
}

func (ns *NullString) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		ns.Valid = false
		return nil
	}
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}
	ns.Valid = true
	ns.String = s
	return nil
}

func (ns NullString) MarshalJSON() ([]byte, error) {
	if !ns.Valid {
		return []byte("null"), nil
	}
	s, err := json.Marshal(ns.String)
	if err != nil {
		return []byte{}, err
	}
	return s, nil
}

// Scan implements the Scanner interface.
func (ns *NullString) Scan(value interface{}) error {
	if value == nil {
		ns.String, ns.Valid = "", false
		return nil
	}
	ns.String, ns.Valid = value.(string)
	return nil
}

// Value implements the driver Valuer interface.
func (ns NullString) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}
	return ns.String, nil
}

var errLeadingInt = errors.New("types: bad [0-9]*") // never printed

// leadingInt consumes the leading [0-9]* from s.
func leadingInt(s string) (x int64, rem string, err error) {
	i := 0
	for ; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			break
		}
		if x > (1<<63-1)/10 {
			// overflow
			return 0, "", errLeadingInt
		}
		x = x*10 + int64(c) - '0'
		if x < 0 {
			// overflow
			return 0, "", errLeadingInt
		}
	}
	return x, s[i:], nil
}

// leadingFraction consumes the leading [0-9]* from s.
// It is used only for fractions, so does not return an error on overflow,
// it just stops accumulating precision.
func leadingFraction(s string) (x int64, scale float64, rem string) {
	i := 0
	scale = 1
	overflow := false
	for ; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			break
		}
		if overflow {
			continue
		}
		if x > (1<<63-1)/10 {
			// It's possible for overflow to give a positive number, so take care.
			overflow = true
			continue
		}
		y := x*10 + int64(c) - '0'
		if y < 0 {
			overflow = true
			continue
		}
		x = y
		scale *= 10
	}
	return x, scale, s[i:]
}

// fmtInt formats v into the tail of buf.
// It returns the index where the output begins.
func fmtInt(buf []byte, v uint64) int {
	w := len(buf)
	if v == 0 {
		w--
		buf[w] = '0'
	} else {
		for v > 0 {
			w--
			buf[w] = byte(v%10) + '0'
			v /= 10
		}
	}
	return w
}

// fmtFrac formats the fraction of v/10**prec (e.g., ".12345") into the
// tail of buf, omitting trailing zeros. it omits the decimal
// point too when the fraction is 0. It returns the index where the
// output bytes begin and the value v/10**prec.
func fmtFrac(buf []byte, v uint64, prec int) (nw int, nv uint64) {
	// Omit trailing zeros up to and including decimal point.
	w := len(buf)
	print := false
	for i := 0; i < prec; i++ {
		digit := v % 10
		print = print || digit != 0
		if print {
			w--
			buf[w] = byte(digit) + '0'
		}
		v /= 10
	}
	if print {
		w--
		buf[w] = '.'
	}
	return w, v
}

var unitMap = map[string]int64{
	"bit": int64(Bit),
	"B":   int64(Byte),
	"kB":  int64(Kilobyte),
	"MB":  int64(Megabyte),
	"GB":  int64(Gigabyte),
	"TB":  int64(Terabyte),
	"PB":  int64(Petabyte),
	"EB":  int64(Exabyte),
}

const (
	Bit  Bits = 1
	Byte      = 8 * Bit
	// https://en.wikipedia.org/wiki/Orders_of_magnitude_(data)
	Kilobyte = 1000 * Byte
	Megabyte = 1000 * Kilobyte
	Gigabyte = 1000 * Megabyte
	Terabyte = 1000 * Gigabyte
	Petabyte = 1000 * Terabyte
	Exabyte  = 1000 * Petabyte
)

// Bits represents a quantity of bits, bytes, kilobytes or megabytes. Bits are
// parsed and formatted using the IEEE / SI standards, which use multiples of
// 1000 to represent kilobytes and megabytes (instead of multiples of 1024). For
// more information see https://en.wikipedia.org/wiki/Megabyte#Definitions.
type Bits int64

// Bytes returns the size as a floating point number of bytes.
func (b Bits) Bytes() float64 {
	bytes := b / Byte
	bits := b % Byte
	return float64(bytes) + float64(bits)/8
}

// Kilobytes returns the size as a floating point number of kilobytes.
func (b Bits) Kilobytes() float64 {
	bytes := b / Kilobyte
	bits := b % Kilobyte
	return float64(bytes) + float64(bits)/(8*1000)
}

// Megabytes returns the size as a floating point number of megabytes.
func (b Bits) Megabytes() float64 {
	bytes := b / Megabyte
	bits := b % Megabyte
	return float64(bytes) + float64(bits)/(8*1000*1000)
}

// Gigabytes returns the size as a floating point number of gigabytes.
func (b Bits) Gigabytes() float64 {
	bytes := b / Gigabyte
	bits := b % Gigabyte
	return float64(bytes) + float64(bits)/(8*1000*1000*1000)
}

// String returns a string representation of b using the largest unit that has a
// positive number before the decimal. At most three decimal places of precision
// are printed.
func (b Bits) String() string {
	if b == 0 {
		return "0"
	}
	// Largest value is "-123.150EB"
	var buf [10]byte
	w := len(buf) - 1
	u := uint64(b)
	neg := b < 0
	if neg {
		u = -u
	}
	if u < uint64(Byte) {
		w -= 2
		copy(buf[w:], "bit")
		w = fmtInt(buf[:w], u)
	} else {
		switch {
		case u < uint64(Kilobyte):
			w -= 0
			buf[w] = 'B'
			u = (u * 1e3 / 8)
		case u < uint64(Megabyte):
			w -= 1
			copy(buf[w:], "kB")
			u /= 8
		case u < uint64(Gigabyte):
			w -= 1
			copy(buf[w:], "MB")
			u /= 8 * 1e3
		case u < uint64(Terabyte):
			w -= 1
			copy(buf[w:], "GB")
			u /= 8 * 1e6
		case u < uint64(Petabyte):
			w -= 1
			copy(buf[w:], "TB")
			u /= 8 * 1e9
		case u < uint64(Exabyte):
			w -= 1
			copy(buf[w:], "PB")
			u /= 8 * 1e12
		case u >= uint64(Exabyte):
			w -= 1
			copy(buf[w:], "EB")
			u /= 8 * 1e15
		}
		w, u = fmtFrac(buf[:w], u, 3)
		w = fmtInt(buf[:w], u)
	}
	if neg {
		w--
		buf[w] = '-'
	}
	return string(buf[w:])
}

// ParseBits parses a quantity of bits. A bit size is a possibly signed sequence
// of decimal numbers, each with optional fraction and a unit suffix. Valid
// units are "bit", "B", "kB", "MB", "GB", "TB". Kilobytes are converted to
// bytes by dividing by 1000, not 1024, per the IEEE standard.
func ParseBits(s string) (Bits, error) {
	// Basically this is lifted from time.ParseDuration with a different unit
	// map.
	// [-+]?([0-9]*(\.[0-9]*)?[a-z]+)+
	orig := s
	var d int64
	neg := false

	// Consume [-+]?
	if s != "" {
		c := s[0]
		if c == '-' || c == '+' {
			neg = c == '-'
			s = s[1:]
		}
	}
	if s == "0" {
		return 0, nil
	}
	if s == "" {
		return 0, errors.New("types: invalid bits " + orig)
	}
	for s != "" {
		var (
			v, f  int64       // integers before, after decimal point
			scale float64 = 1 // value = v + f/scale
		)
		var err error

		// The next character must be [0-9.]
		if !(s[0] == '.' || '0' <= s[0] && s[0] <= '9') {
			return 0, errors.New("types: invalid bits " + orig)
		}
		// Consume [0-9]*
		pl := len(s)
		v, s, err = leadingInt(s)
		if err != nil {
			return 0, errors.New("types: invalid bits " + orig)
		}
		pre := pl != len(s) // whether we consumed anything before a period

		// Consume (\.[0-9]*)?
		post := false
		if s != "" && s[0] == '.' {
			s = s[1:]
			pl := len(s)
			f, scale, s = leadingFraction(s)
			post = pl != len(s)
		}
		if !pre && !post {
			// no digits (e.g. ".s" or "-.s")
			return 0, errors.New("types: invalid bits " + orig)
		}
		// Consume unit.
		i := 0
		for ; i < len(s); i++ {
			c := s[i]
			if c == '.' || '0' <= c && c <= '9' {
				break
			}
		}
		if i == 0 {
			return 0, errors.New("types: missing unit in input " + orig)
		}
		u := s[:i]
		s = s[i:]
		unit, ok := unitMap[u]
		if !ok {
			return 0, errors.New("types: unknown unit " + u + " in bit data " + orig)
		}
		if v > (1<<63-1)/unit {
			// overflow
			return 0, errors.New("types: invalid bits " + orig)
		}
		v *= unit
		if f > 0 {
			// float64 is needed to be nanosecond accurate for fractions of hours.
			// v >= 0 && (f*unit/scale) <= 3.6e+12 (ns/h, h is the largest unit)
			v += int64(float64(f) * (float64(unit) / scale))
			if v < 0 {
				// overflow
				return 0, errors.New("types: invalid bits " + orig)
			}
		}
		d += v
		if d < 0 {
			// overflow
			return 0, errors.New("types: invalid bits " + orig)
		}
	}
	if neg {
		d = -d
	}
	return Bits(d), nil
}
