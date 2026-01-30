package text

import "strings"

// ANSICodesSupported will be true on consoles where ANSI Escape Codes/Sequences
// are supported.
var ANSICodesSupported = areANSICodesSupported()

// Escape encodes the string with the ANSI Escape Sequence.
// For ex.:
//
//	Escape("Ghost", "") == "Ghost"
//	Escape("Ghost", "\x1b[91m") == "\x1b[91mGhost\x1b[0m"
//	Escape("\x1b[94mGhost\x1b[0mLady", "\x1b[91m") == "\x1b[94mGhost\x1b[0m\x1b[91mLady\x1b[0m"
//	Escape("Nymeria\x1b[94mGhost\x1b[0mLady", "\x1b[91m") == "\x1b[91mNymeria\x1b[94mGhost\x1b[0m\x1b[91mLady\x1b[0m"
//	Escape("Nymeria \x1b[94mGhost\x1b[0m Lady", "\x1b[91m") == "\x1b[91mNymeria \x1b[94mGhost\x1b[0m\x1b[91m Lady\x1b[0m"
func Escape(str string, escapeSeq string) string {
	var out strings.Builder
	// Estimate capacity: original string + escape sequences
	out.Grow(len(str) + len(escapeSeq)*3 + len(EscapeReset)*2)

	if !strings.HasPrefix(str, EscapeStart) {
		out.WriteString(escapeSeq)
	}
	out.WriteString(strings.ReplaceAll(str, EscapeReset, EscapeReset+escapeSeq))
	if !strings.HasSuffix(out.String(), EscapeReset) {
		out.WriteString(EscapeReset)
	}
	result := out.String()
	if strings.Contains(result, escapeSeq+EscapeReset) {
		result = strings.ReplaceAll(result, escapeSeq+EscapeReset, "")
	}
	return result
}

// StripEscape strips all ANSI Escape Sequence from the string.
// For ex.:
//
//	StripEscape("Ghost") == "Ghost"
//	StripEscape("\x1b[91mGhost\x1b[0m") == "Ghost"
//	StripEscape("\x1b[94mGhost\x1b[0m\x1b[91mLady\x1b[0m") == "GhostLady"
//	StripEscape("\x1b[91mNymeria\x1b[94mGhost\x1b[0m\x1b[91mLady\x1b[0m") == "NymeriaGhostLady"
//	StripEscape("\x1b[91mNymeria \x1b[94mGhost\x1b[0m\x1b[91m Lady\x1b[0m") == "Nymeria Ghost Lady"
func StripEscape(str string) string {
	var out strings.Builder
	out.Grow(StringWidthWithoutEscSequences(str))

	isEscSeq := false
	for _, sChr := range str {
		if sChr == EscapeStartRune {
			isEscSeq = true
		}
		if !isEscSeq {
			out.WriteRune(sChr)
		}
		if isEscSeq && sChr == EscapeStopRune {
			isEscSeq = false
		}
	}
	return out.String()
}
