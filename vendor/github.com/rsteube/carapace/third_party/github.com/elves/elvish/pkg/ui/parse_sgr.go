package ui

import (
	"strconv"
	"strings"
)

const sgrPrefix = "\033["

var sgrStyling = map[int]Styling{
	0: Reset,
	1: Bold,
	2: Dim,
	4: Underlined,
	5: Blink,
	7: Inverse,
}

// StyleFromSGR builds a Style from an SGR sequence.
func StyleFromSGR(s string) Style {
	var ret Style
	StylingFromSGR(s).transform(&ret)
	return ret
}

// StylingFromSGR builds a Style from an SGR sequence.
func StylingFromSGR(s string) Styling {
	styling := jointStyling{}
	codes := getSGRCodes(s)
	if len(codes) == 0 {
		return Reset
	}
	for len(codes) > 0 {
		code := codes[0]
		consume := 1
		var moreStyling Styling

		switch {
		case sgrStyling[code] != nil:
			moreStyling = sgrStyling[code]
		case 30 <= code && code <= 37:
			moreStyling = Fg(ansiColor(code - 30))
		case 40 <= code && code <= 47:
			moreStyling = Bg(ansiColor(code - 40))
		case 90 <= code && code <= 97:
			moreStyling = Fg(ansiBrightColor(code - 90))
		case 100 <= code && code <= 107:
			moreStyling = Bg(ansiBrightColor(code - 100))
		case code == 38 && len(codes) >= 3 && codes[1] == 5:
			moreStyling = Fg(xterm256Color(codes[2]))
			consume = 3
		case code == 48 && len(codes) >= 3 && codes[1] == 5:
			moreStyling = Bg(xterm256Color(codes[2]))
			consume = 3
		case code == 38 && len(codes) >= 5 && codes[1] == 2:
			moreStyling = Fg(trueColor{
				uint8(codes[2]), uint8(codes[3]), uint8(codes[4])})
			consume = 5
		case code == 48 && len(codes) >= 5 && codes[1] == 2:
			moreStyling = Bg(trueColor{
				uint8(codes[2]), uint8(codes[3]), uint8(codes[4])})
			consume = 5
		default:
			// Do nothing; skip this code
		}
		codes = codes[consume:]
		if moreStyling != nil {
			styling = append(styling, moreStyling)
		}
	}
	return styling
}

func getSGRCodes(s string) []int {
	var codes []int
	for _, part := range strings.Split(s, ";") {
		if part == "" {
			codes = append(codes, 0)
		} else {
			code, err := strconv.Atoi(part)
			if err == nil {
				codes = append(codes, code)
			}
		}
	}
	return codes
}
