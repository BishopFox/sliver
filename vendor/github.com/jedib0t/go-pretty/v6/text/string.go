package text

import (
	"strings"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
	"golang.org/x/text/width"
)

// RuneWidth stuff
var (
	rwCondition = runewidth.NewCondition()
)

// InsertEveryN inserts the rune every N characters in the string. For ex.:
//
//	InsertEveryN("Ghost", '-', 1) == "G-h-o-s-t"
//	InsertEveryN("Ghost", '-', 2) == "Gh-os-t"
//	InsertEveryN("Ghost", '-', 3) == "Gho-st"
//	InsertEveryN("Ghost", '-', 4) == "Ghos-t"
//	InsertEveryN("Ghost", '-', 5) == "Ghost"
func InsertEveryN(str string, runeToInsert rune, n int) string {
	if n <= 0 {
		return str
	}

	sLen := StringWidthWithoutEscSequences(str)
	var out strings.Builder
	out.Grow(sLen + (sLen / n))
	outLen, esp := 0, escSeqParser{}
	for idx, c := range str {
		if esp.InSequence() {
			esp.Consume(c)
			out.WriteRune(c)
			continue
		}
		esp.Consume(c)
		if !esp.InSequence() && outLen > 0 && (outLen%n) == 0 && idx != sLen {
			out.WriteRune(runeToInsert)
		}
		out.WriteRune(c)
		if !esp.InSequence() {
			outLen += RuneWidth(c)
		}
	}
	return out.String()
}

// LongestLineLen returns the length of the longest "line" within the
// argument string. For ex.:
//
//	LongestLineLen("Ghost!\nCome back here!\nRight now!") == 15
func LongestLineLen(str string) int {
	maxLength, currLength, esp := 0, 0, escSeqParser{}
	//fmt.Println(str)
	for _, c := range str {
		//fmt.Printf("%03d | %03d | %c | %5v | %v | %#v\n", idx, c, c, esp.inEscSeq, esp.Codes(), esp.escapeSeq)
		if esp.InSequence() {
			esp.Consume(c)
			continue
		}
		esp.Consume(c)
		if c == '\n' {
			if currLength > maxLength {
				maxLength = currLength
			}
			currLength = 0
		} else if !esp.InSequence() {
			currLength += RuneWidth(c)
		}
	}
	if currLength > maxLength {
		maxLength = currLength
	}
	return maxLength
}

// OverrideRuneWidthEastAsianWidth can *probably* help with alignment, and
// length calculation issues when dealing with Unicode character-set and a
// non-English language set in the LANG variable.
//
// Set this to 'false' to force the "runewidth" library to pretend to deal with
// English character-set. Be warned that if the text/content you are dealing
// with contains East Asian character-set, this may result in unexpected
// behavior.
//
// References:
// * https://github.com/mattn/go-runewidth/issues/64#issuecomment-1221642154
// * https://github.com/jedib0t/go-pretty/issues/220
// * https://github.com/jedib0t/go-pretty/issues/204
func OverrideRuneWidthEastAsianWidth(val bool) {
	rwCondition.EastAsianWidth = val
}

// Pad pads the given string with as many characters as needed to make it as
// long as specified (maxLen). This function does not count escape sequences
// while calculating length of the string. Ex.:
//
//	Pad("Ghost", 0, ' ') == "Ghost"
//	Pad("Ghost", 3, ' ') == "Ghost"
//	Pad("Ghost", 5, ' ') == "Ghost"
//	Pad("Ghost", 7, ' ') == "Ghost  "
//	Pad("Ghost", 10, '.') == "Ghost....."
func Pad(str string, maxLen int, paddingChar rune) string {
	strLen := StringWidthWithoutEscSequences(str)
	if strLen < maxLen {
		str += strings.Repeat(string(paddingChar), maxLen-strLen)
	}
	return str
}

// ProcessCRLF converts "\r\n" to "\n", and processes lone "\r" by moving the
// cursor/carriage to the start of the line and overwrites the contents
// accordingly. Ex.:
//
// ProcessCRLF("abc") == "abc"
// ProcessCRLF("abc\r\ndef") == "abc\ndef"
// ProcessCRLF("abc\r\ndef\rghi") == "abc\nghi"
// ProcessCRLF("abc\r\ndef\rghi\njkl") == "abc\nghi\njkl"
// ProcessCRLF("abc\r\ndef\rghi\njkl\r") == "abc\nghi\njkl"
// ProcessCRLF("abc\r\ndef\rghi\rjkl\rmn") == "abc\nmnl"
func ProcessCRLF(str string) string {
	str = strings.ReplaceAll(str, "\r\n", "\n")
	if !strings.Contains(str, "\r") {
		return str
	}

	lines := strings.Split(str, "\n")
	for lineIdx, line := range lines {
		if !strings.Contains(line, "\r") {
			continue
		}

		lineRunes, newLineRunes := []rune(line), make([]rune, 0)
		for idx, realIdx := 0, 0; idx < len(lineRunes); idx++ {
			// if a CR, move "cursor" back to beginning of line
			if lineRunes[idx] == '\r' {
				realIdx = 0
				continue
			}

			// if cursor is not at end, overwrite
			if realIdx < len(newLineRunes) {
				newLineRunes[realIdx] = lineRunes[idx]
			} else { // else append
				newLineRunes = append(newLineRunes, lineRunes[idx])
			}
			realIdx++
		}
		lines[lineIdx] = string(newLineRunes)
	}
	return strings.Join(lines, "\n")
}

// RepeatAndTrim repeats the given string until it is as long as maxRunes.
// For ex.:
//
//	RepeatAndTrim("", 5) == ""
//	RepeatAndTrim("Ghost", 0) == ""
//	RepeatAndTrim("Ghost", 5) == "Ghost"
//	RepeatAndTrim("Ghost", 7) == "GhostGh"
//	RepeatAndTrim("Ghost", 10) == "GhostGhost"
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
//
//	RuneCount("") == 0
//	RuneCount("Ghost") == 5
//	RuneCount("\x1b[33mGhost\x1b[0m") == 5
//	RuneCount("\x1b[33mGhost\x1b[0") == 5
//
// Deprecated: in favor of RuneWidthWithoutEscSequences
func RuneCount(str string) int {
	return StringWidthWithoutEscSequences(str)
}

// RuneWidth returns the mostly accurate character-width of the rune. This is
// not 100% accurate as the character width is usually dependent on the
// typeface (font) used in the console/terminal. For ex.:
//
//	RuneWidth('A') == 1
//	RuneWidth('ツ') == 2
//	RuneWidth('⊙') == 1
//	RuneWidth('︿') == 2
//	RuneWidth(0x27) == 0
func RuneWidth(r rune) int {
	return rwCondition.RuneWidth(r)
}

// RuneWidthWithoutEscSequences is similar to RuneWidth, except for the fact
// that it ignores escape sequences while counting. For ex.:
//
//	RuneWidthWithoutEscSequences("") == 0
//	RuneWidthWithoutEscSequences("Ghost") == 5
//	RuneWidthWithoutEscSequences("\x1b[33mGhost\x1b[0m") == 5
//	RuneWidthWithoutEscSequences("\x1b[33mGhost\x1b[0") == 5
//
// deprecated: use StringWidthWithoutEscSequences instead
func RuneWidthWithoutEscSequences(str string) int {
	return StringWidthWithoutEscSequences(str)
}

// Snip returns the given string with a fixed length. For ex.:
//
//	Snip("Ghost", 0, "~") == "Ghost"
//	Snip("Ghost", 1, "~") == "~"
//	Snip("Ghost", 3, "~") == "Gh~"
//	Snip("Ghost", 5, "~") == "Ghost"
//	Snip("Ghost", 7, "~") == "Ghost  "
//	Snip("\x1b[33mGhost\x1b[0m", 7, "~") == "\x1b[33mGhost\x1b[0m  "
func Snip(str string, length int, snipIndicator string) string {
	if length > 0 {
		lenStr := StringWidthWithoutEscSequences(str)
		if lenStr > length {
			lenStrFinal := length - StringWidthWithoutEscSequences(snipIndicator)
			return Trim(str, lenStrFinal) + snipIndicator
		}
	}
	return str
}

// StringWidth is similar to RuneWidth, except it works on a string. For
// ex.:
//
//	StringWidth("Ghost 生命"): 10
//	StringWidth("\x1b[33mGhost 生命\x1b[0m"): 19
func StringWidth(str string) int {
	return rwCondition.StringWidth(str)
}

// StringWidthWithoutEscSequences is similar to RuneWidth, except for the fact
// that it ignores escape sequences while counting. For ex.:
//
//	StringWidthWithoutEscSequences("") == 0
//	StringWidthWithoutEscSequences("Ghost") == 5
//	StringWidthWithoutEscSequences("\x1b[33mGhost\x1b[0m") == 5
//	StringWidthWithoutEscSequences("\x1b[33mGhost\x1b[0") == 5
//	StringWidthWithoutEscSequences("Ghost 生命"): 10
//	StringWidthWithoutEscSequences("\x1b[33mGhost 生命\x1b[0m"): 10
func StringWidthWithoutEscSequences(str string) int {
	count, esp := 0, escSeqParser{}
	for _, c := range str {
		if esp.InSequence() {
			esp.Consume(c)
			continue
		}
		esp.Consume(c)
		if !esp.InSequence() {
			count += RuneWidth(c)
		}
	}
	return count
}

// Trim trims a string to the given length while ignoring escape sequences. For
// ex.:
//
//	Trim("Ghost", 3) == "Gho"
//	Trim("Ghost", 6) == "Ghost"
//	Trim("\x1b[33mGhost\x1b[0m", 3) == "\x1b[33mGho\x1b[0m"
//	Trim("\x1b[33mGhost\x1b[0m", 6) == "\x1b[33mGhost\x1b[0m"
func Trim(str string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}

	var out strings.Builder
	out.Grow(maxLen)

	outLen, esp := 0, escSeqParser{}
	for _, sChr := range str {
		if esp.InSequence() {
			esp.Consume(sChr)
			out.WriteRune(sChr)
			continue
		}
		esp.Consume(sChr)
		if esp.InSequence() {
			out.WriteRune(sChr)
			continue
		}
		if outLen < maxLen {
			outLen++
			out.WriteRune(sChr)
			continue
		}
	}
	return out.String()
}

// Widen is like width.Widen.String() but ignores escape sequences. For ex:
//
//	Widen("Ghost 生命"): "Ｇｈｏｓｔ\u3000生命"
//	Widen("\x1b[33mGhost 生命\x1b[0m"): "\x1b[33mＧｈｏｓｔ\u3000生命\x1b[0m"
func Widen(str string) string {
	sb := strings.Builder{}
	sb.Grow(len(str))

	esp := escSeqParser{}
	for _, c := range str {
		if esp.InSequence() {
			sb.WriteRune(c)
			esp.Consume(c)
			continue
		}
		esp.Consume(c)
		if !esp.InSequence() {
			sb.WriteString(width.Widen.String(string(c)))
		} else {
			sb.WriteRune(c)
		}
	}
	return sb.String()
}
