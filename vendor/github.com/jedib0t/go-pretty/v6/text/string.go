package text

import (
	"strings"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
)

// Constants
const (
	EscapeReset     = EscapeStart + "0" + EscapeStop
	EscapeStart     = "\x1b["
	EscapeStartRune = rune(27) // \x1b
	EscapeStop      = "m"
	EscapeStopRune  = 'm'
)

// InsertEveryN inserts the rune every N characters in the string. For ex.:
//  InsertEveryN("Ghost", '-', 1) == "G-h-o-s-t"
//  InsertEveryN("Ghost", '-', 2) == "Gh-os-t"
//  InsertEveryN("Ghost", '-', 3) == "Gho-st"
//  InsertEveryN("Ghost", '-', 4) == "Ghos-t"
//  InsertEveryN("Ghost", '-', 5) == "Ghost"
func InsertEveryN(str string, runeToInsert rune, n int) string {
	if n <= 0 {
		return str
	}

	sLen := RuneCount(str)
	var out strings.Builder
	out.Grow(sLen + (sLen / n))
	outLen, isEscSeq := 0, false
	for idx, c := range str {
		if c == EscapeStartRune {
			isEscSeq = true
		}

		if !isEscSeq && outLen > 0 && (outLen%n) == 0 && idx != sLen {
			out.WriteRune(runeToInsert)
		}
		out.WriteRune(c)
		if !isEscSeq {
			outLen += RuneWidth(c)
		}

		if isEscSeq && c == EscapeStopRune {
			isEscSeq = false
		}
	}
	return out.String()
}

// LongestLineLen returns the length of the longest "line" within the
// argument string. For ex.:
//  LongestLineLen("Ghost!\nCome back here!\nRight now!") == 15
func LongestLineLen(str string) int {
	maxLength, currLength, isEscSeq := 0, 0, false
	for _, c := range str {
		if c == EscapeStartRune {
			isEscSeq = true
		} else if isEscSeq && c == EscapeStopRune {
			isEscSeq = false
			continue
		}

		if c == '\n' {
			if currLength > maxLength {
				maxLength = currLength
			}
			currLength = 0
		} else if !isEscSeq {
			currLength += RuneWidth(c)
		}
	}
	if currLength > maxLength {
		maxLength = currLength
	}
	return maxLength
}

// Pad pads the given string with as many characters as needed to make it as
// long as specified (maxLen). This function does not count escape sequences
// while calculating length of the string. Ex.:
//  Pad("Ghost", 0, ' ') == "Ghost"
//  Pad("Ghost", 3, ' ') == "Ghost"
//  Pad("Ghost", 5, ' ') == "Ghost"
//  Pad("Ghost", 7, ' ') == "Ghost  "
//  Pad("Ghost", 10, '.') == "Ghost....."
func Pad(str string, maxLen int, paddingChar rune) string {
	strLen := RuneCount(str)
	if strLen < maxLen {
		str += strings.Repeat(string(paddingChar), maxLen-strLen)
	}
	return str
}

// RepeatAndTrim repeats the given string until it is as long as maxRunes.
// For ex.:
//  RepeatAndTrim("", 5) == ""
//  RepeatAndTrim("Ghost", 0) == ""
//  RepeatAndTrim("Ghost", 5) == "Ghost"
//  RepeatAndTrim("Ghost", 7) == "GhostGh"
//  RepeatAndTrim("Ghost", 10) == "GhostGhost"
func RepeatAndTrim(str string, maxRunes int) string {
	if str == "" || maxRunes == 0 {
		return ""
	} else if maxRunes == utf8.RuneCountInString(str) {
		return str
	}
	repeatedS := strings.Repeat(str, int(maxRunes/utf8.RuneCountInString(str))+1)
	return Trim(repeatedS, maxRunes)
}

// RuneCount is similar to utf8.RuneCountInString, except for the fact that it
// ignores escape sequences while counting. For ex.:
//  RuneCount("") == 0
//  RuneCount("Ghost") == 5
//  RuneCount("\x1b[33mGhost\x1b[0m") == 5
//  RuneCount("\x1b[33mGhost\x1b[0") == 5
func RuneCount(str string) int {
	count, isEscSeq := 0, false
	for _, c := range str {
		if c == EscapeStartRune {
			isEscSeq = true
		} else if isEscSeq {
			if c == EscapeStopRune {
				isEscSeq = false
			}
		} else {
			count += RuneWidth(c)
		}
	}
	return count
}

// RuneWidth returns the mostly accurate character-width of the rune. This is
// not 100% accurate as the character width is usually dependant on the
// typeface (font) used in the console/terminal. For ex.:
//  RuneWidth('A') == 1
//  RuneWidth('ツ') == 2
//  RuneWidth('⊙') == 1
//  RuneWidth('︿') == 2
//  RuneWidth(0x27) == 0
func RuneWidth(r rune) int {
	return runewidth.RuneWidth(r)
}

// Snip returns the given string with a fixed length. For ex.:
//  Snip("Ghost", 0, "~") == "Ghost"
//  Snip("Ghost", 1, "~") == "~"
//  Snip("Ghost", 3, "~") == "Gh~"
//  Snip("Ghost", 5, "~") == "Ghost"
//  Snip("Ghost", 7, "~") == "Ghost  "
//  Snip("\x1b[33mGhost\x1b[0m", 7, "~") == "\x1b[33mGhost\x1b[0m  "
func Snip(str string, length int, snipIndicator string) string {
	if length > 0 {
		lenStr := RuneCount(str)
		if lenStr > length {
			lenStrFinal := length - RuneCount(snipIndicator)
			return Trim(str, lenStrFinal) + snipIndicator
		}
	}
	return str
}

// Trim trims a string to the given length while ignoring escape sequences. For
// ex.:
//  Trim("Ghost", 3) == "Gho"
//  Trim("Ghost", 6) == "Ghost"
//  Trim("\x1b[33mGhost\x1b[0m", 3) == "\x1b[33mGho\x1b[0m"
//  Trim("\x1b[33mGhost\x1b[0m", 6) == "\x1b[33mGhost\x1b[0m"
func Trim(str string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}

	var out strings.Builder
	out.Grow(maxLen)

	outLen, isEscSeq, lastEscSeq := 0, false, strings.Builder{}
	for _, sChr := range str {
		out.WriteRune(sChr)
		if sChr == EscapeStartRune {
			isEscSeq = true
			lastEscSeq.Reset()
			lastEscSeq.WriteRune(sChr)
		} else if isEscSeq {
			lastEscSeq.WriteRune(sChr)
			if sChr == EscapeStopRune {
				isEscSeq = false
			}
		} else {
			outLen++
			if outLen == maxLen {
				break
			}
		}
	}
	if lastEscSeq.Len() > 0 && lastEscSeq.String() != EscapeReset {
		out.WriteString(EscapeReset)
	}
	return out.String()
}
