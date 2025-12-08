package text

import (
	"strings"
)

// WrapHard wraps a string to the given length using a newline. Handles strings
// with ANSI escape sequences (such as text color) without breaking the text
// formatting. Breaks all words that go beyond the line boundary.
//
// For examples, refer to the unit-tests or GoDoc examples.
func WrapHard(str string, wrapLen int) string {
	if wrapLen <= 0 {
		return ""
	}
	str = strings.Replace(str, "\t", "    ", -1)
	sLen := StringWidthWithoutEscSequences(str)
	if sLen <= wrapLen {
		return str
	}

	out := &strings.Builder{}
	out.Grow(sLen + (sLen / wrapLen))
	for idx, paragraph := range strings.Split(str, "\n\n") {
		if idx > 0 {
			out.WriteString("\n\n")
		}
		wrapHard(paragraph, wrapLen, out)
	}

	return out.String()
}

// WrapSoft wraps a string to the given length using a newline. Handles strings
// with ANSI escape sequences (such as text color) without breaking the text
// formatting. Tries to move words that go beyond the line boundary to the next
// line.
//
// For examples, refer to the unit-tests or GoDoc examples.
func WrapSoft(str string, wrapLen int) string {
	if wrapLen <= 0 {
		return ""
	}
	str = strings.Replace(str, "\t", "    ", -1)
	sLen := StringWidthWithoutEscSequences(str)
	if sLen <= wrapLen {
		return str
	}

	out := &strings.Builder{}
	out.Grow(sLen + (sLen / wrapLen))
	for idx, paragraph := range strings.Split(str, "\n\n") {
		if idx > 0 {
			out.WriteString("\n\n")
		}
		wrapSoft(paragraph, wrapLen, out)
	}

	return out.String()
}

// WrapText is very similar to WrapHard except for one minor difference. Unlike
// WrapHard which discards line-breaks and respects only paragraph-breaks, this
// function respects line-breaks too.
//
// For examples, refer to the unit-tests or GoDoc examples.
func WrapText(str string, wrapLen int) string {
	if wrapLen <= 0 {
		return ""
	}
	str = strings.Replace(str, "\t", "    ", -1)
	sLen := StringWidthWithoutEscSequences(str)
	if sLen <= wrapLen {
		return str
	}

	out := &strings.Builder{}
	out.Grow(sLen + (sLen / wrapLen))
	for idx, line := range strings.Split(str, "\n") {
		if idx > 0 {
			out.WriteString("\n")
		}
		wrapHard(line, wrapLen, out)
	}

	return out.String()
}

func appendChar(char rune, wrapLen int, lineLen *int, inEscSeq bool, lastSeenEscSeq string, out *strings.Builder) {
	// handle reaching the end of the line as dictated by wrapLen or by finding
	// a newline character
	if (*lineLen == wrapLen && !inEscSeq && char != '\n') || (char == '\n') {
		if lastSeenEscSeq != "" {
			// terminate escape sequence and the line; and restart the escape
			// sequence in the next line
			out.WriteString(EscapeReset)
			out.WriteRune('\n')
			out.WriteString(lastSeenEscSeq)
		} else {
			// just start a new line
			out.WriteRune('\n')
		}
		// reset line index to 0th character
		*lineLen = 0
	}

	// if the rune is not a new line, output it
	if char != '\n' {
		out.WriteRune(char)

		// increment the line index if not in the middle of an escape sequence
		if !inEscSeq {
			*lineLen += RuneWidth(char)
		}
	}
}

func appendWord(word string, lineIdx *int, lastSeenEscSeq string, wrapLen int, out *strings.Builder) {
	inEscSeq := false
	for _, char := range word {
		if char == EscapeStartRune {
			inEscSeq = true
			lastSeenEscSeq = ""
		}
		if inEscSeq {
			lastSeenEscSeq += string(char)
		}

		appendChar(char, wrapLen, lineIdx, inEscSeq, lastSeenEscSeq, out)

		if inEscSeq && char == EscapeStopRune {
			inEscSeq = false
		}
		if lastSeenEscSeq == EscapeReset {
			lastSeenEscSeq = ""
		}
	}
}

func terminateLine(wrapLen int, lineLen *int, lastSeenEscSeq string, out *strings.Builder) {
	if *lineLen < wrapLen {
		out.WriteString(strings.Repeat(" ", wrapLen-*lineLen))
	}
	// something is already on the line; terminate it
	if lastSeenEscSeq != "" {
		out.WriteString(EscapeReset)
	}
	out.WriteRune('\n')
	out.WriteString(lastSeenEscSeq)
	*lineLen = 0
}

func terminateOutput(lastSeenEscSeq string, out *strings.Builder) {
	if lastSeenEscSeq != "" && lastSeenEscSeq != EscapeReset && !strings.HasSuffix(out.String(), EscapeReset) {
		out.WriteString(EscapeReset)
	}
}

func wrapHard(paragraph string, wrapLen int, out *strings.Builder) {
	esp := EscSeqParser{}
	lineLen, lastSeenEscSeq := 0, ""
	words := strings.Fields(paragraph)
	for wordIdx, word := range words {
		if openEscSeq := esp.ParseString(word); openEscSeq != "" {
			lastSeenEscSeq = openEscSeq
		}
		if lineLen > 0 {
			out.WriteRune(' ')
			lineLen++
		}

		wordLen := StringWidthWithoutEscSequences(word)
		if lineLen+wordLen <= wrapLen { // word fits within the line
			out.WriteString(word)
			lineLen += wordLen
		} else { // word doesn't fit within the line; hard-wrap
			appendWord(word, &lineLen, lastSeenEscSeq, wrapLen, out)
		}

		// end of line; but more words incoming
		if lineLen == wrapLen && wordIdx < len(words)-1 {
			terminateLine(wrapLen, &lineLen, lastSeenEscSeq, out)
		}
	}
	terminateOutput(lastSeenEscSeq, out)
}

func wrapSoft(paragraph string, wrapLen int, out *strings.Builder) {
	esp := EscSeqParser{}
	lineLen, lastSeenEscSeq := 0, ""
	words := strings.Fields(paragraph)
	for wordIdx, word := range words {
		if openEscSeq := esp.ParseString(word); openEscSeq != "" {
			lastSeenEscSeq = openEscSeq
		}

		spacing, spacingLen := wrapSoftSpacing(lineLen)
		wordLen := StringWidthWithoutEscSequences(word)
		if lineLen+spacingLen+wordLen <= wrapLen { // word fits within the line
			out.WriteString(spacing)
			out.WriteString(word)
			lineLen += spacingLen + wordLen
		} else { // word doesn't fit within the line
			lineLen = wrapSoftLastWordInLine(wrapLen, lineLen, lastSeenEscSeq, wordLen, word, out)
		}

		// end of line; but more words incoming
		if lineLen == wrapLen && wordIdx < len(words)-1 {
			terminateLine(wrapLen, &lineLen, lastSeenEscSeq, out)
		}
	}
	terminateOutput(lastSeenEscSeq, out)
}

func wrapSoftLastWordInLine(wrapLen int, lineLen int, lastSeenEscSeq string, wordLen int, word string, out *strings.Builder) int {
	if lineLen > 0 { // something is already on the line; terminate it
		terminateLine(wrapLen, &lineLen, lastSeenEscSeq, out)
	}
	if wordLen <= wrapLen { // word fits within a single line
		out.WriteString(word)
		lineLen = wordLen
	} else { // word doesn't fit within a single line; hard-wrap
		appendWord(word, &lineLen, lastSeenEscSeq, wrapLen, out)
	}
	return lineLen
}

func wrapSoftSpacing(lineLen int) (string, int) {
	spacing, spacingLen := "", 0
	if lineLen > 0 {
		spacing, spacingLen = " ", 1
	}
	return spacing, spacingLen
}
