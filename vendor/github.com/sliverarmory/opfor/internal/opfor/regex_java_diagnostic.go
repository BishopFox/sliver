package opfor

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// validateJavaRegexPattern catches Java syntax that the regexp2 parser accepts
// as a different construct. Most malformed patterns still flow through the
// translator and are classified by javaRegexEngineDiagnostic; keeping this
// preflight narrow avoids maintaining a second regular-expression compiler.
func validateJavaRegexPattern(pattern string) error {
	// COMMENTS mode changes how braces and group prefixes are read. The mode
	// lowering pass removes ignored text before regexp2 sees it, so defer those
	// uncommon spellings to that parser rather than rejecting valid comments.
	commentsMayBeEnabled := strings.Contains(pattern, "(?x")
	quoted := false
	for index := 0; index < len(pattern); {
		if quoted {
			if strings.HasPrefix(pattern[index:], `\E`) {
				quoted = false
				index += 2
				continue
			}
			_, size := utf8.DecodeRuneInString(pattern[index:])
			index += size
			continue
		}
		if pattern[index] == '\\' {
			if strings.HasPrefix(pattern[index:], `\Q`) {
				quoted = true
				index += 2
				continue
			}
			if index+2 < len(pattern) && strings.ContainsRune("pPNxb", rune(pattern[index+1])) && pattern[index+2] == '{' {
				if relative := strings.IndexByte(pattern[index+3:], '}'); relative >= 0 {
					index += 3 + relative + 1
					continue
				}
				// Let the escape translator provide the precise unclosed-family
				// diagnostic instead of mistaking its brace for a quantifier.
				return nil
			}
			index++
			if index < len(pattern) {
				_, size := utf8.DecodeRuneInString(pattern[index:])
				index += size
			}
			continue
		}
		if pattern[index] == '[' {
			end, err := javaRegexClassEnd(pattern, index)
			if err != nil {
				// The ordinary compiler reports the source-compatible unclosed
				// class location, including nested-class behavior.
				return nil
			}
			index = end + 1
			continue
		}
		if !commentsMayBeEnabled && pattern[index] == '(' && index+1 < len(pattern) && pattern[index+1] == '?' {
			if err := validateJavaRegexGroupPrefix(pattern, index); err != nil {
				return err
			}
		}
		if !commentsMayBeEnabled && pattern[index] == '{' {
			if end, err := validateJavaRegexCountedClosure(pattern, index); err != nil {
				return err
			} else {
				index = end
				continue
			}
		}
		_, size := utf8.DecodeRuneInString(pattern[index:])
		index += size
	}
	return nil
}

func validateJavaRegexGroupPrefix(pattern string, start int) error {
	index := start + 2
	if index >= len(pattern) {
		return newJavaRegexPatternSyntaxError(pattern, "Unknown inline modifier", index)
	}
	switch pattern[index] {
	case ':', '=', '!', '>':
		return nil
	case '<':
		// Lookbehind and named-capture validation are handled by the normal
		// translator, which knows the complete group registry.
		return nil
	case '$', '@':
		return newJavaRegexPatternSyntaxError(pattern, "Unknown group type", index)
	}

	sawMinus := false
	for index < len(pattern) {
		switch pattern[index] {
		case 'i', 'd', 'm', 's', 'u', 'x', 'U':
			index++
		case 'a', 'A', 'j', 'J':
			// Preserve the translator's explicit private-flag guard for direct
			// API callers; the bridge maps that guard to Java's public warning.
			return nil
		case '-':
			if sawMinus {
				return newJavaRegexPatternSyntaxError(pattern, "Unknown inline modifier", index)
			}
			sawMinus = true
			index++
		case ')', ':':
			return nil
		default:
			return newJavaRegexPatternSyntaxError(pattern, "Unknown inline modifier", index)
		}
	}
	return newJavaRegexPatternSyntaxError(pattern, "Unknown inline modifier", len(pattern))
}

func validateJavaRegexCountedClosure(pattern string, start int) (int, error) {
	index := start + 1
	if index >= len(pattern) || pattern[index] < '0' || pattern[index] > '9' {
		return 0, newJavaRegexPatternSyntaxError(pattern, "Illegal repetition", index)
	}

	minimum, next, overflowAt := javaRegexDecimal(pattern, index)
	if overflowAt >= 0 {
		return 0, newJavaRegexPatternSyntaxError(pattern, "Illegal repetition range", overflowAt)
	}
	index = next
	maximum := minimum
	if index < len(pattern) && pattern[index] == ',' {
		index++
		if index < len(pattern) && pattern[index] == '}' {
			return index + 1, nil
		}
		maximum, next, overflowAt = javaRegexDecimal(pattern, index)
		if overflowAt >= 0 {
			return 0, newJavaRegexPatternSyntaxError(pattern, "Illegal repetition range", overflowAt)
		}
		index = next
	}
	if index >= len(pattern) || pattern[index] != '}' {
		return 0, newJavaRegexPatternSyntaxError(pattern, "Unclosed counted closure", index)
	}
	if maximum < minimum {
		return 0, newJavaRegexPatternSyntaxError(pattern, "Illegal repetition range", index)
	}
	return index + 1, nil
}

func javaRegexDecimal(pattern string, start int) (int64, int, int) {
	const maxInt = int64(1<<31 - 1)
	value := int64(0)
	index := start
	for index < len(pattern) && pattern[index] >= '0' && pattern[index] <= '9' {
		digit := int64(pattern[index] - '0')
		if value > (maxInt-digit)/10 {
			return 0, index, index
		}
		value = value*10 + digit
		index++
	}
	return value, index, -1
}

func javaRegexEngineDiagnostic(pattern, detail string) (string, int, bool) {
	atByte := func(description string, byteIndex int) (string, int, bool) {
		return description, javaRegexPatternCodePointLength(pattern[:min(max(byteIndex, 0), len(pattern))]), true
	}

	switch {
	case strings.Contains(detail, "unmatched closing parenthesis"), strings.Contains(detail, "unexpected )"):
		if index := javaRegexUnmatchedClosing(pattern); index >= 0 {
			// OpenJDK notices the extra ')' after expr returns, so its cursor
			// remains one position before the actual closing parenthesis.
			return "Unmatched closing ')'", javaRegexPatternCodePointLength(pattern[:index]) - 1, true
		}
	case strings.Contains(detail, "missing argument to repetition operator"), strings.Contains(detail, "invalid nested repetition operator"):
		if index := javaRegexDanglingMeta(pattern); index >= 0 {
			return atByte(fmt.Sprintf("Dangling meta character '%c'", pattern[index]), index)
		}
	case strings.Contains(detail, "range in reverse order"), strings.Contains(detail, "invalid character class range"):
		if index := javaRegexReverseRangeEnd(pattern); index >= 0 {
			return atByte("Illegal character range", index)
		}
	case strings.Contains(detail, "escape at end of pattern"), strings.Contains(detail, "illegal \\ at end of pattern"):
		return "Unescaped trailing backslash", javaRegexPatternCodePointLength(pattern), true
	case strings.Contains(detail, "insufficient hexadecimal digits"):
		if index := strings.Index(pattern, `\x`); index >= 0 {
			return "Illegal hexadecimal escape sequence", javaRegexPatternCodePointLength(pattern), true
		}
		if index := strings.Index(pattern, `\u`); index >= 0 {
			return "Illegal Unicode escape sequence", javaRegexPatternCodePointLength(pattern), true
		}
	case strings.Contains(detail, "missing closing }"):
		if index := strings.Index(pattern, `\x{`); index >= 0 {
			return atByte("Illegal hexadecimal escape sequence", index+2)
		}
	case strings.Contains(detail, "hex values may not be larger than"):
		if start := strings.Index(pattern, `\x{`); start >= 0 {
			end := strings.IndexByte(pattern[start+3:], '}')
			if end >= 0 {
				return atByte("Hexadecimal codepoint is too big", start+3+end-1)
			}
		}
	case strings.Contains(detail, "missing control character"):
		if index := strings.Index(pattern, `\c`); index >= 0 {
			return atByte("Illegal control escape sequence", index+1)
		}
	case strings.Contains(detail, "malformed \\p"):
		return fmt.Sprintf("Unknown character property name {%c}", 0), javaRegexPatternCodePointLength(pattern), true
	case strings.Contains(detail, "unterminated \\p"):
		return "Unclosed character family", javaRegexPatternCodePointLength(pattern), true
	case strings.Contains(detail, "empty \\p"):
		if index := strings.IndexByte(pattern, '}'); index >= 0 {
			return atByte("Empty character family", index)
		}
	case strings.Contains(detail, "unknown Java Unicode property"),
		strings.Contains(detail, "unsupported Java Unicode block"),
		strings.Contains(detail, "unsupported Java Unicode script"),
		strings.Contains(detail, "unknown Java Unicode category"):
		if name, end, ok := javaRegexPropertySpelling(pattern); ok {
			return atByte("Unknown character property name {"+name+"}", end)
		}
	case strings.Contains(detail, "illegal Unicode character-name escape"):
		return "Illegal character name escape sequence", javaRegexPatternCodePointLength(pattern), true
	case strings.Contains(detail, "unterminated Unicode character-name escape"):
		return "Unclosed character name escape sequence", javaRegexPatternCodePointLength(pattern), true
	case strings.Contains(detail, "unknown Unicode character name"):
		if name, end, ok := javaRegexCharacterNameSpelling(pattern); ok {
			return atByte("Unknown character name ["+name+"]", end)
		}
	case strings.Contains(detail, "invalid named capture"):
		if description, index, ok := javaRegexInvalidNamedCapture(pattern); ok {
			return atByte(description, index)
		}
	case strings.Contains(detail, "duplicate named capture"):
		if name := javaRegexQuotedErrorValue(detail); name != "" {
			needle := "(?<" + name + ">"
			first := strings.Index(pattern, needle)
			if first >= 0 {
				second := strings.Index(pattern[first+len(needle):], needle)
				if second >= 0 {
					return atByte("Named capturing group <"+name+"> is already defined", first+len(needle)+second+len(needle)-1)
				}
			}
		}
	case strings.Contains(detail, "unknown named backreference"):
		if name, end, ok := javaRegexNamedBackreference(pattern); ok {
			return atByte("named capturing group <"+name+"> does not exist", end)
		}
	case strings.Contains(detail, "unterminated named backreference"):
		if start := strings.Index(pattern, `\k<`); start >= 0 {
			name := pattern[start+3:]
			if name == "" || !asciiRegexLetter(name[0]) {
				return atByte("capturing group name does not start with a Latin letter", len(pattern))
			}
			return atByte("named capturing group is missing trailing '>'", len(pattern))
		}
	case strings.Contains(detail, "malformed \\k<"):
		if start := strings.Index(pattern, `\k`); start >= 0 {
			return atByte(`\k is not followed by '<' for named capturing group`, start+2)
		}
	case strings.Contains(detail, "reserved for internal Java case modes"):
		if start := strings.Index(pattern, "(?"); start >= 0 {
			for index := start + 2; index < len(pattern); index++ {
				if strings.ContainsRune("aAjJ", rune(pattern[index])) {
					return atByte("Unknown inline modifier", index)
				}
			}
		}
	case strings.Contains(detail, "unrecognized escape sequence"),
		strings.Contains(detail, "illegal \\R escape"),
		strings.Contains(detail, "illegal \\X escape"):
		if index := javaRegexUnsupportedEscape(pattern); index >= 0 {
			return atByte("Illegal/unsupported escape sequence", index+1)
		}
	case strings.Contains(detail, "empty character-class intersection"):
		if index := strings.Index(pattern, "&&"); index >= 0 {
			return atByte("Bad class syntax", index+1)
		}
	}
	return "", 0, false
}

func javaRegexUnmatchedClosing(pattern string) int {
	depth := 0
	for index := 0; index < len(pattern); index++ {
		if pattern[index] == '\\' {
			index++
			continue
		}
		if pattern[index] == '[' {
			if end, err := javaRegexClassEnd(pattern, index); err == nil {
				index = end
				continue
			}
		}
		switch pattern[index] {
		case '(':
			depth++
		case ')':
			if depth == 0 {
				return index
			}
			depth--
		}
	}
	return -1
}

func javaRegexDanglingMeta(pattern string) int {
	for index := 0; index < len(pattern); index++ {
		if !strings.ContainsRune("*+?", rune(pattern[index])) {
			continue
		}
		if pattern[index] == '?' && index > 0 && pattern[index-1] == '(' {
			continue
		}
		if index == 0 || strings.ContainsRune("(|*+?", rune(pattern[index-1])) {
			// + and ? immediately following a quantifier are its legal
			// possessive/reluctant modifier; another identical quantifier is not.
			if index > 0 && strings.ContainsRune("*+?}", rune(pattern[index-1])) &&
				(pattern[index] == '+' || pattern[index] == '?') {
				continue
			}
			return index
		}
	}
	return -1
}

func javaRegexReverseRangeEnd(pattern string) int {
	for index := 1; index+1 < len(pattern); index++ {
		if pattern[index] == '-' && pattern[index-1] != '\\' && pattern[index+1] != ']' {
			return index + 1
		}
	}
	return -1
}

func javaRegexPropertySpelling(pattern string) (string, int, bool) {
	for _, prefix := range []string{`\p{`, `\P{`} {
		if start := strings.Index(pattern, prefix); start >= 0 {
			nameStart := start + len(prefix)
			if relative := strings.IndexByte(pattern[nameStart:], '}'); relative >= 0 {
				end := nameStart + relative
				return pattern[nameStart:end], end, true
			}
		}
	}
	return "", 0, false
}

func javaRegexCharacterNameSpelling(pattern string) (string, int, bool) {
	start := strings.Index(pattern, `\N{`)
	if start < 0 {
		return "", 0, false
	}
	nameStart := start + 3
	relative := strings.IndexByte(pattern[nameStart:], '}')
	if relative < 0 {
		return "", 0, false
	}
	end := nameStart + relative
	return pattern[nameStart:end], end, true
}

func javaRegexInvalidNamedCapture(pattern string) (string, int, bool) {
	start := strings.Index(pattern, "(?<")
	if start < 0 || start+3 >= len(pattern) {
		return "", 0, false
	}
	nameStart := start + 3
	end := strings.IndexByte(pattern[nameStart:], '>')
	if end < 0 {
		end = len(pattern) - nameStart
	}
	name := pattern[nameStart : nameStart+end]
	if name == "" || !asciiRegexLetter(name[0]) {
		return "capturing group name does not start with a Latin letter", nameStart, true
	}
	for index := 1; index < len(name); index++ {
		if !asciiRegexLetter(name[index]) && (name[index] < '0' || name[index] > '9') {
			return "named capturing group is missing trailing '>'", nameStart + index, true
		}
	}
	return "named capturing group is missing trailing '>'", nameStart + len(name), true
}

func javaRegexNamedBackreference(pattern string) (string, int, bool) {
	start := strings.Index(pattern, `\k<`)
	if start < 0 {
		return "", 0, false
	}
	nameStart := start + 3
	relative := strings.IndexByte(pattern[nameStart:], '>')
	if relative < 0 {
		return "", 0, false
	}
	end := nameStart + relative
	return pattern[nameStart:end], end, true
}

func javaRegexQuotedErrorValue(detail string) string {
	first := strings.IndexByte(detail, '"')
	if first < 0 {
		return ""
	}
	second := strings.IndexByte(detail[first+1:], '"')
	if second < 0 {
		return ""
	}
	return detail[first+1 : first+1+second]
}

func javaRegexUnsupportedEscape(pattern string) int {
	for index := 0; index+1 < len(pattern); index++ {
		if pattern[index] != '\\' {
			continue
		}
		kind := pattern[index+1]
		if (kind >= 'A' && kind <= 'Z') || (kind >= 'a' && kind <= 'z') {
			return index
		}
		index++
	}
	return -1
}
