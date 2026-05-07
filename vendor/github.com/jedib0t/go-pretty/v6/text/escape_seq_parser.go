package text

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Constants
const (
	EscapeReset        = EscapeResetCSI
	EscapeResetCSI     = EscapeStartCSI + "0" + EscapeStopCSI
	EscapeResetOSI     = EscapeStartOSI + "0" + EscapeStopOSI
	EscapeStart        = EscapeStartCSI
	EscapeStartCSI     = "\x1b["
	EscapeStartOSI     = "\x1b]"
	EscapeStartRune    = rune(27) // \x1b
	EscapeStartRuneCSI = '['      // [
	EscapeStartRuneOSI = ']'      // ]
	EscapeStop         = EscapeStopCSI
	EscapeStopCSI      = "m"
	EscapeStopOSI      = "\\"
	EscapeStopRune     = EscapeStopRuneCSI
	EscapeStopRuneCSI  = 'm'
	EscapeStopRuneOSI  = '\\'
)

// Deprecated Constants
const (
	CSIStartRune = EscapeStartRuneCSI
	CSIStopRune  = EscapeStopRuneCSI
	OSIStartRune = EscapeStartRuneOSI
	OSIStopRune  = EscapeStopRuneOSI
)

type escSeqKind int

const (
	escSeqKindUnknown escSeqKind = iota
	escSeqKindCSI
	escSeqKindOSI
)

// private constants
const (
	escCodeResetAll        = 0
	escCodeResetIntensity  = 22
	escCodeResetItalic     = 23
	escCodeResetUnderline  = 24
	escCodeResetBlink      = 25
	escCodeResetReverse    = 27
	escCodeResetCrossedOut = 29
	escCodeBold            = 1
	escCodeDim             = 2
	escCodeItalic          = 3
	escCodeUnderline       = 4
	escCodeBlinkSlow       = 5
	escCodeBlinkRapid      = 6
	escCodeReverse         = 7
	escCodeConceal         = 8
	escCodeCrossedOut      = 9

	// conceal OSI sequences
	escapeStartConcealOSI = "\x1b]8;"
	escapeStopConcealOSI  = "\x1b\\"
)

// 256-color codes
const (
	escCode256FgStart = 38
	escCode256BgStart = 48
	escCode256Color   = 5
	escCodeResetFg    = 39
	escCodeResetBg    = 49
	escCode256Max     = 255
)

// Internal encoding for 256-color codes uses fg256Start and bg256Start from color.go
// Private constants initialized from private constants to avoid repeated casting in hot paths
// Foreground 256-color: fg256Start + colorIndex (1000-1255)
// Background 256-color: bg256Start + colorIndex (2000-2255)
const (
	escCode256FgBase = int(fg256Start) // 1000
	escCode256BgBase = int(bg256Start) // 2000
)

// Standard color code ranges
const (
	// Standard foreground colors (30-37)
	escCodeFgStdStart = 30
	escCodeFgStdEnd   = 37
	// Bright foreground colors (90-97)
	escCodeFgBrightStart = 90
	escCodeFgBrightEnd   = 97
	// Standard background colors (40-47)
	escCodeBgStdStart = 40
	escCodeBgStdEnd   = 47
	// Bright background colors (100-107)
	escCodeBgBrightStart = 100
	escCodeBgBrightEnd   = 107
)

// Special characters
const (
	escRuneBEL = '\a' // BEL character (ASCII 7)
)

// EscSeqParser parses ANSI escape sequences from text and tracks active formatting codes.
// It supports both CSI (Control Sequence Introducer) and OSI (Operating System Command)
// escape sequence formats.
type EscSeqParser struct {
	// codes tracks active escape sequence codes (e.g., 1 for bold, 3 for italic).
	codes map[int]bool

	// inEscSeq indicates whether the parser is currently inside an escape sequence.
	inEscSeq bool
	// escSeqKind identifies the type of escape sequence being parsed (CSI or OSI).
	escSeqKind escSeqKind
	// escapeSeq accumulates the current escape sequence being parsed.
	escapeSeq string
}

func (s *EscSeqParser) Codes() []int {
	codes := make([]int, 0)
	for code, val := range s.codes {
		if val {
			codes = append(codes, code)
		}
	}
	sort.Ints(codes)
	return codes
}

func (s *EscSeqParser) Consume(char rune) {
	if !s.inEscSeq && char == EscapeStartRune {
		s.inEscSeq = true
		s.escSeqKind = escSeqKindUnknown
		s.escapeSeq = ""
	} else if s.inEscSeq && s.escSeqKind == escSeqKindUnknown {
		switch char {
		case EscapeStartRuneCSI:
			s.escSeqKind = escSeqKindCSI
		case EscapeStartRuneOSI:
			s.escSeqKind = escSeqKindOSI
		}
	}

	if s.inEscSeq {
		s.escapeSeq += string(char)

		// --- FIX for OSC 8 hyperlinks (e.g. \x1b]8;;url\x07label\x1b]8;;\x07)
		if s.escSeqKind == escSeqKindOSI &&
			strings.HasPrefix(s.escapeSeq, escapeStartConcealOSI) &&
			char == escRuneBEL { // BEL

			s.ParseSeq(s.escapeSeq, s.escSeqKind)
			s.Reset()
			return
		}

		if s.isEscapeStopRune(char) {
			s.ParseSeq(s.escapeSeq, s.escSeqKind)
			s.Reset()
		}
	}
}

func (s *EscSeqParser) InSequence() bool {
	return s.inEscSeq
}

func (s *EscSeqParser) IsOpen() bool {
	return len(s.codes) > 0
}

func (s *EscSeqParser) ParseSeq(seq string, seqKind escSeqKind) {
	if s.codes == nil {
		s.codes = make(map[int]bool)
	}

	seq = s.stripEscapeSequence(seq, seqKind)
	codes := s.splitAndTrimCodes(seq)
	processed256ColorIndices := s.process256ColorSequences(codes)
	s.processRegularCodes(codes, processed256ColorIndices)
}

func (s *EscSeqParser) ParseString(str string) string {
	s.escapeSeq, s.inEscSeq, s.escSeqKind = "", false, escSeqKindUnknown
	for _, char := range str {
		s.Consume(char)
	}
	return s.Sequence()
}

func (s *EscSeqParser) Reset() {
	s.inEscSeq = false
	s.escSeqKind = escSeqKindUnknown
	s.escapeSeq = ""
}

func (s *EscSeqParser) Sequence() string {
	out := strings.Builder{}
	if s.IsOpen() {
		out.WriteString(EscapeStart)
		codes := s.Codes()
		for idx, code := range codes {
			if idx > 0 {
				out.WriteRune(';')
			}
			// Check if this is a 256-color foreground code (1000-1255)
			if code >= escCode256FgBase && code <= escCode256FgBase+escCode256Max {
				colorIndex := code - escCode256FgBase
				out.WriteString(fmt.Sprintf("%d;%d;%d", escCode256FgStart, escCode256Color, colorIndex))
			} else if code >= escCode256BgBase && code <= escCode256BgBase+escCode256Max {
				// 256-color background code (2000-2255)
				colorIndex := code - escCode256BgBase
				out.WriteString(fmt.Sprintf("%d;%d;%d", escCode256BgStart, escCode256Color, colorIndex))
			} else {
				// Regular code
				out.WriteString(fmt.Sprint(code))
			}
		}
		out.WriteString(EscapeStop)
	}

	return out.String()
}

// clearAllBackgroundColors clears all background color codes.
func (s *EscSeqParser) clearAllBackgroundColors() {
	for code := escCodeBgStdStart; code <= escCodeBgStdEnd; code++ {
		delete(s.codes, code)
	}
	for code := escCodeBgBrightStart; code <= escCodeBgBrightEnd; code++ {
		delete(s.codes, code)
	}
	for code := escCode256BgBase; code <= escCode256BgBase+escCode256Max; code++ {
		delete(s.codes, code)
	}
}

// clearAllForegroundColors clears all foreground color codes.
func (s *EscSeqParser) clearAllForegroundColors() {
	for code := escCodeFgStdStart; code <= escCodeFgStdEnd; code++ {
		delete(s.codes, code)
	}
	for code := escCodeFgBrightStart; code <= escCodeFgBrightEnd; code++ {
		delete(s.codes, code)
	}
	for code := escCode256FgBase; code <= escCode256FgBase+escCode256Max; code++ {
		delete(s.codes, code)
	}
}

// clearColorRange clears standard foreground or background colors.
func (s *EscSeqParser) clearColorRange(isForeground bool) {
	if isForeground {
		// Clear standard foreground colors (30-37, 90-97)
		for code := escCodeFgStdStart; code <= escCodeFgStdEnd; code++ {
			delete(s.codes, code)
		}
		for code := escCodeFgBrightStart; code <= escCodeFgBrightEnd; code++ {
			delete(s.codes, code)
		}
	} else {
		// Clear standard background colors (40-47, 100-107)
		for code := escCodeBgStdStart; code <= escCodeBgStdEnd; code++ {
			delete(s.codes, code)
		}
		for code := escCodeBgBrightStart; code <= escCodeBgBrightEnd; code++ {
			delete(s.codes, code)
		}
	}
}

func (s *EscSeqParser) isEscapeStopRune(char rune) bool {
	if strings.HasPrefix(s.escapeSeq, escapeStartConcealOSI) {
		if strings.HasSuffix(s.escapeSeq, escapeStopConcealOSI) {
			return true
		}
	} else if (s.escSeqKind == escSeqKindCSI && char == EscapeStopRuneCSI) ||
		(s.escSeqKind == escSeqKindOSI && char == EscapeStopRuneOSI) {
		return true
	}
	return false
}

// isRegularCode checks if a code is a regular code (not a 256-color encoded value).
func (s *EscSeqParser) isRegularCode(codeNum int) bool {
	return codeNum < escCode256FgBase || codeNum > escCode256BgBase+escCode256Max
}

// parse256ColorSequence attempts to parse a 256-color sequence starting at index i.
// Returns (colorIndex, base, true) if valid, or (0, 0, false) if not.
func (s *EscSeqParser) parse256ColorSequence(codes []string, i int) (colorIndex int, base int, ok bool) {
	if i+2 >= len(codes) {
		return 0, 0, false
	}

	codeNum, err := strconv.Atoi(codes[i])
	if err != nil {
		return 0, 0, false
	}

	var expectedBase int
	switch codeNum {
	case escCode256FgStart:
		expectedBase = escCode256FgBase
	case escCode256BgStart:
		expectedBase = escCode256BgBase
	default:
		return 0, 0, false
	}

	nextCode, err := strconv.Atoi(codes[i+1])
	if err != nil || nextCode != escCode256Color {
		return 0, 0, false
	}

	colorIndex, err = strconv.Atoi(codes[i+2])
	if err != nil || colorIndex < 0 || colorIndex > escCode256Max {
		return 0, 0, false
	}

	return colorIndex, expectedBase, true
}

// process256ColorSequences processes 256-color sequences (38;5;n or 48;5;n) and returns
// a map of indices that were part of valid 256-color sequences.
func (s *EscSeqParser) process256ColorSequences(codes []string) map[int]bool {
	processedIndices := make(map[int]bool)
	for i := 0; i < len(codes); i++ {
		if colorIndex, base, ok := s.parse256ColorSequence(codes, i); ok {
			s.set256Color(base, colorIndex)
			s.clearColorRange(base == escCode256FgBase)
			processedIndices[i] = true
			processedIndices[i+1] = true
			processedIndices[i+2] = true
			i += 2 // Skip i+1 and i+2 (loop will increment to i+3)
		}
	}
	return processedIndices
}

// processCode handles a single escape code.
func (s *EscSeqParser) processCode(codeNum int) {
	switch codeNum {
	case escCodeResetAll:
		s.codes = make(map[int]bool)
	case escCodeResetIntensity:
		delete(s.codes, escCodeBold)
		delete(s.codes, escCodeDim)
	case escCodeResetItalic:
		delete(s.codes, escCodeItalic)
	case escCodeResetUnderline:
		delete(s.codes, escCodeUnderline)
	case escCodeResetBlink:
		delete(s.codes, escCodeBlinkSlow)
		delete(s.codes, escCodeBlinkRapid)
	case escCodeResetReverse:
		delete(s.codes, escCodeReverse)
	case escCodeResetCrossedOut:
		delete(s.codes, escCodeCrossedOut)
	case escCodeResetFg:
		s.clearAllForegroundColors()
	case escCodeResetBg:
		s.clearAllBackgroundColors()
	default:
		if s.isRegularCode(codeNum) {
			s.codes[codeNum] = true
		}
	}
}

// processRegularCodes processes regular escape codes and reset codes.
func (s *EscSeqParser) processRegularCodes(codes []string, processedIndices map[int]bool) {
	for i, code := range codes {
		if processedIndices[i] {
			continue
		}

		codeNum, err := strconv.Atoi(code)
		if err != nil {
			continue
		}

		s.processCode(codeNum)
	}
}

// set256Color sets a 256-color code and clears conflicting colors.
func (s *EscSeqParser) set256Color(base int, colorIndex int) {
	encodedValue := base + colorIndex
	s.codes[encodedValue] = true

	// Clear other colors in the same range
	for code := base; code <= base+escCode256Max; code++ {
		if code != encodedValue {
			delete(s.codes, code)
		}
	}
}

// splitAndTrimCodes splits the sequence by semicolons and trims whitespace.
func (s *EscSeqParser) splitAndTrimCodes(seq string) []string {
	codes := strings.Split(seq, ";")
	for i := range codes {
		codes[i] = strings.TrimSpace(codes[i])
	}
	return codes
}

// stripEscapeSequence removes escape sequence markers from the input string.
func (s *EscSeqParser) stripEscapeSequence(seq string, seqKind escSeqKind) string {
	if seqKind == escSeqKindOSI {
		seq = strings.Replace(seq, EscapeStartOSI, "", 1)
		seq = strings.Replace(seq, EscapeStopOSI, "", 1)
	} else {
		seq = strings.Replace(seq, EscapeStartCSI, "", 1)
		seq = strings.Replace(seq, EscapeStopCSI, "", 1)
	}
	return seq
}
