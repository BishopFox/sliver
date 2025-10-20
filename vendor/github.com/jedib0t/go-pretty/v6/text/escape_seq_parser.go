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

type escSeqParser struct {
	codes map[int]bool

	// consume specific
	inEscSeq   bool
	escSeqKind escSeqKind
	escapeSeq  string
}

func (s *escSeqParser) Codes() []int {
	codes := make([]int, 0)
	for code, val := range s.codes {
		if val {
			codes = append(codes, code)
		}
	}
	sort.Ints(codes)
	return codes
}

func (s *escSeqParser) Consume(char rune) {
	if !s.inEscSeq && char == EscapeStartRune {
		s.inEscSeq = true
		s.escSeqKind = escSeqKindUnknown
		s.escapeSeq = ""
	} else if s.inEscSeq && s.escSeqKind == escSeqKindUnknown {
		if char == EscapeStartRuneCSI {
			s.escSeqKind = escSeqKindCSI
		} else if char == EscapeStartRuneOSI {
			s.escSeqKind = escSeqKindOSI
		}
	}

	if s.inEscSeq {
		s.escapeSeq += string(char)

		// --- FIX for OSC 8 hyperlinks (e.g. \x1b]8;;url\x07label\x1b]8;;\x07)
		if s.escSeqKind == escSeqKindOSI &&
			strings.HasPrefix(s.escapeSeq, escapeStartConcealOSI) &&
			char == '\a' { // BEL

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

func (s *escSeqParser) InSequence() bool {
	return s.inEscSeq
}

func (s *escSeqParser) IsOpen() bool {
	return len(s.codes) > 0
}

func (s *escSeqParser) Reset() {
	s.inEscSeq = false
	s.escSeqKind = escSeqKindUnknown
	s.escapeSeq = ""
}

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
)

func (s *escSeqParser) ParseSeq(seq string, seqKind escSeqKind) {
	if s.codes == nil {
		s.codes = make(map[int]bool)
	}

	if seqKind == escSeqKindOSI {
		seq = strings.Replace(seq, EscapeStartOSI, "", 1)
		seq = strings.Replace(seq, EscapeStopOSI, "", 1)
	} else { // escSeqKindCSI
		seq = strings.Replace(seq, EscapeStartCSI, "", 1)
		seq = strings.Replace(seq, EscapeStopCSI, "", 1)
	}

	codes := strings.Split(seq, ";")
	for _, code := range codes {
		code = strings.TrimSpace(code)
		if codeNum, err := strconv.Atoi(code); err == nil {
			switch codeNum {
			case escCodeResetAll:
				s.codes = make(map[int]bool) // clear everything
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
			default:
				s.codes[codeNum] = true
			}
		}
	}
}

func (s *escSeqParser) ParseString(str string) string {
	s.escapeSeq, s.inEscSeq, s.escSeqKind = "", false, escSeqKindUnknown
	for _, char := range str {
		s.Consume(char)
	}
	return s.Sequence()
}

func (s *escSeqParser) Sequence() string {
	out := strings.Builder{}
	if s.IsOpen() {
		out.WriteString(EscapeStart)
		for idx, code := range s.Codes() {
			if idx > 0 {
				out.WriteRune(';')
			}
			out.WriteString(fmt.Sprint(code))
		}
		out.WriteString(EscapeStop)
	}

	return out.String()
}

const (
	escapeStartConcealOSI = "\x1b]8;"
	escapeStopConcealOSI  = "\x1b\\"
)

func (s *escSeqParser) isEscapeStopRune(char rune) bool {
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
