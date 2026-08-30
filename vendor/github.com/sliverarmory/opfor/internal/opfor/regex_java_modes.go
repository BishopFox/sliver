package opfor

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/sliverarmory/opfor/internal/regexp2"
)

// javaRegexModes are the embedded java.util.regex.Pattern flags whose
// semantics are observable without access to Pattern.compile's integer flag
// argument. RegexBridge only supplies a pattern string, so these modes all
// start disabled.
type javaRegexModes struct {
	caseInsensitive bool
	comments        bool
	dotAll          bool
	multiline       bool
	unicodeCase     bool
	unicodeClass    bool
	unixLines       bool
}

// Java property escapes are predicates, not ordinary character-class ranges.
// Case-insensitive Pattern modes widen only Java's explicitly cased
// properties; they do not case-fold arbitrary scripts, blocks, or POSIX
// predicates. The class-algebra pass needs property escapes to look like
// single-character atoms, so the mode pass substitutes an absent private-use
// character for each occurrence. Once class algebra has decomposed unions and
// intersections, expand restores the predicate under a case-sensitive private
// regexp2 scope.
type javaRegexPropertyMarker struct {
	character   rune
	atom        string
	replacement string
}

type javaRegexPropertyMarkers struct {
	reserved         map[rune]struct{}
	rangeIndex       int
	next             rune
	replacementBytes int
	markers          []javaRegexPropertyMarker
}

// Property predicates can expand into generated range classes. Bound the
// translated expression before handing it to regexp2 so a small source
// pattern cannot manufacture an unbounded parser allocation. One MiB leaves
// ample room for ordinary Aggressor patterns and dozens of the largest
// generated Unicode predicates.
const javaRegexTranslatedPatternLimit = 1 << 20

// javaRegexTranslationGuardError marks OPFOR safety limits and translator
// integrity failures. These are not java.util.regex.PatternSyntaxException
// equivalents and must never be recovered as Sleep script warnings.
type javaRegexTranslationGuardError struct {
	message string
}

func (err *javaRegexTranslationGuardError) Error() string {
	if err == nil {
		return ""
	}
	return err.message
}

func newJavaRegexTranslationGuardError(message string) error {
	return &javaRegexTranslationGuardError{message: message}
}

var javaRegexPrivateUseRanges = [...]struct{ first, last rune }{
	{first: 0xe000, last: 0xf8ff},
	{first: 0xf0000, last: 0xffffd},
	{first: 0x100000, last: 0x10fffd},
}

func newJavaRegexPropertyMarkers(source string) *javaRegexPropertyMarkers {
	markers := &javaRegexPropertyMarkers{reserved: make(map[rune]struct{})}
	for _, character := range source {
		if javaRegexPrivateUse(character) {
			markers.reserved[character] = struct{}{}
		}
	}
	return markers
}

func javaRegexPrivateUse(character rune) bool {
	for _, current := range javaRegexPrivateUseRanges {
		if character >= current.first && character <= current.last {
			return true
		}
	}
	return false
}

func (m *javaRegexPropertyMarkers) add(fragment string) (string, error) {
	replacement := `(?-aj:` + fragment + `)`
	if len(replacement) > javaRegexTranslatedPatternLimit-m.replacementBytes {
		return "", javaRegexTranslatedPatternLimitError()
	}
	for m.rangeIndex < len(javaRegexPrivateUseRanges) {
		current := javaRegexPrivateUseRanges[m.rangeIndex]
		if m.next < current.first {
			m.next = current.first
		}
		for m.next <= current.last {
			candidate := m.next
			m.next++
			if _, unavailable := m.reserved[candidate]; unavailable {
				continue
			}
			atom := "[" + string(candidate) + "]"
			m.markers = append(m.markers, javaRegexPropertyMarker{
				character:   candidate,
				atom:        atom,
				replacement: replacement,
			})
			m.replacementBytes += len(replacement)
			return atom, nil
		}
		m.rangeIndex++
		m.next = 0
	}
	return "", newJavaRegexTranslationGuardError("regular expression exhausts private-use property markers")
}

func (m *javaRegexPropertyMarkers) expand(pattern string) (string, error) {
	if err := validateJavaRegexTranslatedPatternSize(pattern); err != nil {
		return "", err
	}
	byCharacter := make(map[rune]int, len(m.markers))
	for index, marker := range m.markers {
		byCharacter[marker.character] = index
	}

	seen := make([]bool, len(m.markers))
	var result strings.Builder
	result.Grow(len(pattern))
	for index := 0; index < len(pattern); {
		if pattern[index] == '[' && index+1 < len(pattern) {
			character, size := utf8.DecodeRuneInString(pattern[index+1:])
			markerIndex, marked := byCharacter[character]
			end := index + 1 + size
			if marked && end < len(pattern) && pattern[end] == ']' {
				if seen[markerIndex] {
					return "", newJavaRegexTranslationGuardError("regular-expression property marker was duplicated during class translation")
				}
				replacement := m.markers[markerIndex].replacement
				if result.Len() > javaRegexTranslatedPatternLimit-len(replacement) {
					return "", javaRegexTranslatedPatternLimitError()
				}
				result.WriteString(replacement)
				seen[markerIndex] = true
				index = end + 1
				continue
			}
		}
		if result.Len() == javaRegexTranslatedPatternLimit {
			return "", javaRegexTranslatedPatternLimitError()
		}
		result.WriteByte(pattern[index])
		index++
	}
	for _, present := range seen {
		if !present {
			return "", newJavaRegexTranslationGuardError("regular-expression property marker was lost during class translation")
		}
	}
	return result.String(), nil
}

func javaRegexTranslatedPatternLimitError() error {
	return newJavaRegexTranslationGuardError(fmt.Sprintf(
		"regular-expression translation exceeds %d-byte limit",
		javaRegexTranslatedPatternLimit,
	))
}

func validateJavaRegexTranslatedPatternSize(pattern string) error {
	if len(pattern) > javaRegexTranslatedPatternLimit {
		return javaRegexTranslatedPatternLimitError()
	}
	return nil
}

// translateJavaRegexModes lowers the Java-only line model, predefined Unicode
// classes, and embedded d/u/U flags to regexp2 constructs. regexp2 otherwise
// follows .NET/RE2 rules: dot and anchors only recognize LF, RE2 \s omits VT,
// \b is Unicode-aware even when \w is ASCII, and d/U have unrelated or no
// meanings. Keeping those tokens intact would therefore silently mis-match.
func translateJavaRegexModes(pattern string) (string, *javaRegexPropertyMarkers, error) {
	var result strings.Builder
	result.Grow(len(pattern))

	current := javaRegexModes{}
	properties := newJavaRegexPropertyMarkers(pattern)
	groups := make([]javaRegexModes, 0, 8)
	for index := 0; index < len(pattern); {
		if current.comments {
			if size := javaRegexIgnoredSpace(pattern[index:]); size > 0 {
				index += size
				continue
			}
			if pattern[index] == '#' {
				end := index + 1
				for end < len(pattern) {
					character, size := utf8.DecodeRuneInString(pattern[end:])
					if javaRegexCommentTerminator(character) {
						break
					}
					end += size
				}
				index = end
				continue
			}
		}

		switch pattern[index] {
		case '\\':
			fragment, next, err := translateJavaRegexEscape(pattern, index, current, false, properties)
			if err != nil {
				return "", nil, err
			}
			result.WriteString(fragment)
			index = next
		case '[':
			end, err := javaRegexClassEndForModes(pattern, index, current)
			if err != nil {
				return "", nil, err
			}
			content, err := translateJavaRegexClassEscapes(pattern[index+1:end], current, properties)
			if err != nil {
				return "", nil, err
			}
			result.WriteByte('[')
			result.WriteString(content)
			result.WriteByte(']')
			index = end + 1
		case '.':
			result.WriteString(javaRegexDot(current))
			index++
		case '^':
			result.WriteString(javaRegexBeginning(current))
			index++
		case '$':
			result.WriteString(javaRegexDollar(current))
			index++
		case '(':
			if flag, private := javaRegexPrivateInlineFlag(pattern, index); private {
				return "", nil, fmt.Errorf("regular-expression flag %q is reserved for internal Java case modes", flag)
			}
			flags, ok := parseJavaRegexInlineFlags(pattern, index)
			if !ok {
				groups = append(groups, current)
				result.WriteByte('(')
				index++
				continue
			}
			next := applyJavaRegexInlineFlags(current, flags)
			kept := translatedJavaRegexCaseFlags(current, next)
			if flags.scoped {
				groups = append(groups, current)
				if kept == "" {
					result.WriteString("(?:")
				} else {
					result.WriteString("(?")
					result.WriteString(kept)
					result.WriteByte(':')
				}
			} else if kept != "" {
				result.WriteString("(?")
				result.WriteString(kept)
				result.WriteByte(')')
			}
			current = next
			index = flags.end
		case ')':
			result.WriteByte(')')
			index++
			if len(groups) > 0 {
				current = groups[len(groups)-1]
				groups = groups[:len(groups)-1]
			}
		default:
			_, size := utf8.DecodeRuneInString(pattern[index:])
			result.WriteString(pattern[index : index+size])
			index += size
		}
	}
	return result.String(), properties, nil
}

type javaRegexInlineFlags struct {
	on, off string
	scoped  bool
	end     int
}

func parseJavaRegexInlineFlags(pattern string, start int) (javaRegexInlineFlags, bool) {
	if start+2 >= len(pattern) || pattern[start:start+2] != "(?" {
		return javaRegexInlineFlags{}, false
	}
	index := start + 2
	off := false
	sawFlag := false
	var enabled, disabled strings.Builder
	for index < len(pattern) {
		current := pattern[index]
		switch current {
		case 'i', 'd', 'm', 's', 'u', 'x', 'U':
			sawFlag = true
			if off {
				disabled.WriteByte(current)
			} else {
				enabled.WriteByte(current)
			}
			index++
		case '-':
			if off {
				return javaRegexInlineFlags{}, false
			}
			sawFlag = true
			off = true
			index++
		case ':', ')':
			// OpenJDK accepts the empty inline-modifier spellings (?) and
			// (?-) as zero-width flag updates. A colon without a flag remains
			// the ordinary (?:...) non-capturing group prefix.
			if current == ':' && !sawFlag {
				return javaRegexInlineFlags{}, false
			}
			return javaRegexInlineFlags{
				on: enabled.String(), off: disabled.String(),
				scoped: current == ':', end: index + 1,
			}, true
		default:
			return javaRegexInlineFlags{}, false
		}
	}
	return javaRegexInlineFlags{}, false
}

// a and j are private regexp2 flags used only after Java-mode translation.
// Reject their uppercase spellings too, even though the fork intentionally
// exposes only lowercase forms. A source pattern must not bypass Java's public
// idmsuxU flag set through a future engine alias.
func javaRegexPrivateInlineFlag(pattern string, start int) (byte, bool) {
	if start+2 >= len(pattern) || pattern[start:start+2] != "(?" {
		return 0, false
	}
	for index := start + 2; index < len(pattern); index++ {
		switch pattern[index] {
		case 'a', 'A', 'j', 'J':
			return pattern[index], true
		case 'i', 'd', 'm', 's', 'u', 'x', 'U', '-':
			continue
		case ':', ')':
			return 0, false
		default:
			return 0, false
		}
	}
	return 0, false
}

func applyJavaRegexInlineFlags(current javaRegexModes, flags javaRegexInlineFlags) javaRegexModes {
	set := func(flag byte, enabled bool) {
		switch flag {
		case 'i':
			current.caseInsensitive = enabled
		case 'd':
			current.unixLines = enabled
		case 'm':
			current.multiline = enabled
		case 's':
			current.dotAll = enabled
		case 'u':
			current.unicodeCase = enabled
		case 'x':
			current.comments = enabled
		case 'U':
			current.unicodeClass = enabled
			if enabled {
				// Pattern.UNICODE_CHARACTER_CLASS implies UNICODE_CASE.
				current.unicodeCase = true
			} else {
				// Pattern's subFlag clears both U and the u that U implies.
				// A later scoped restoration reinstates the saved outer state.
				current.unicodeCase = false
			}
		}
	}
	for index := range flags.on {
		set(flags.on[index], true)
	}
	for index := range flags.off {
		set(flags.off[index], false)
	}
	return current
}

// The private a mode is Java's ASCII-only CASE_INSENSITIVE behavior; j is its
// UNICODE_CASE behavior. Emit exclusive mode transitions so u can change the
// active folding implementation even when i itself remains enabled.
func translatedJavaRegexCaseFlags(current, next javaRegexModes) string {
	effective := func(modes javaRegexModes) byte {
		if !modes.caseInsensitive {
			return 0
		}
		if modes.unicodeCase {
			return 'j'
		}
		return 'a'
	}
	before, after := effective(current), effective(next)
	if before == after {
		return ""
	}
	switch after {
	case 'a':
		return "a-j"
	case 'j':
		return "j-a"
	default:
		return "-aj"
	}
}

func javaRegexIgnoredSpace(value string) int {
	character, size := utf8.DecodeRuneInString(value)
	switch character {
	case '\t', '\n', '\v', '\f', '\r', ' ':
		return size
	default:
		return 0
	}
}

func javaRegexCommentTerminator(character rune) bool {
	switch character {
	case '\n', '\r', '\u0085', '\u2028', '\u2029':
		return true
	default:
		return false
	}
}

func javaRegexDot(modes javaRegexModes) string {
	if modes.dotAll {
		return `[\s\S]`
	}
	if modes.unixLines {
		return `[^\n]`
	}
	return `[^\n\r\u0085\u2028\u2029]`
}

func javaRegexBeginning(modes javaRegexModes) string {
	if !modes.multiline {
		return `\A`
	}
	if modes.unixLines {
		return `(?:\A|(?<=\n)(?!\z))`
	}
	return `(?:\A|(?:(?<=[\n\u0085\u2028\u2029])|(?<=\r)(?!\n))(?!\z))`
}

func javaRegexDollar(modes javaRegexModes) string {
	if modes.multiline {
		if modes.unixLines {
			return `(?:\z|(?=\n))`
		}
		return `(?:\z|(?=\r)|(?<!\r)(?=[\n\u0085\u2028\u2029]))`
	}
	return javaRegexEndBeforeFinalTerminator(modes)
}

func javaRegexEndBeforeFinalTerminator(modes javaRegexModes) string {
	if modes.unixLines {
		return `(?:\z|(?=\n\z))`
	}
	return `(?:\z|(?=\r(?:\n)?\z)|(?<!\r)(?=[\n\u0085\u2028\u2029]\z))`
}

func javaRegexWordClass(modes javaRegexModes) string {
	if modes.unicodeClass {
		return javaRegexGeneratedWordClass()
	}
	return `[A-Za-z0-9_]`
}

func javaRegexWordBoundary(modes javaRegexModes, negated bool) string {
	// UNICODE_CASE changes literal folding, not the definition of Java's \w
	// predicate. Keep the generated word predicate out of the surrounding a/j
	// mode just as property markers do after class algebra.
	word := javaRegexWordClass(modes)
	if modes.caseInsensitive {
		word = `(?-aj:` + word + `)`
	}
	if negated {
		return `(?:(?<=` + word + `)(?=` + word + `)|(?<!` + word + `)(?!` + word + `))`
	}
	return `(?:(?<=` + word + `)(?!` + word + `)|(?<!` + word + `)(?=` + word + `))`
}

func translateJavaRegexClassEscapes(
	content string,
	modes javaRegexModes,
	properties *javaRegexPropertyMarkers,
) (string, error) {
	var result strings.Builder
	for index := 0; index < len(content); {
		if modes.comments {
			if size := javaRegexIgnoredSpace(content[index:]); size > 0 {
				index += size
				continue
			}
			if content[index] == '#' {
				index++
				for index < len(content) {
					character, size := utf8.DecodeRuneInString(content[index:])
					index += size
					if javaRegexCommentTerminator(character) {
						break
					}
				}
				continue
			}
		}
		if content[index] == '\\' {
			fragment, next, err := translateJavaRegexEscape(content, index, modes, true, properties)
			if err != nil {
				return "", err
			}
			result.WriteString(fragment)
			index = next
			continue
		}
		if content[index] == '[' {
			end, err := javaRegexClassEndForModes(content, index, modes)
			if err != nil {
				return "", err
			}
			nested, err := translateJavaRegexClassEscapes(content[index+1:end], modes, properties)
			if err != nil {
				return "", err
			}
			result.WriteByte('[')
			result.WriteString(nested)
			result.WriteByte(']')
			index = end + 1
			continue
		}
		_, size := utf8.DecodeRuneInString(content[index:])
		result.WriteString(content[index : index+size])
		index += size
	}
	return result.String(), nil
}

// COMMENTS mode is active inside Java character classes as well. In
// particular, unescaped whitespace disappears and # can hide brackets until
// the next line. The ordinary class scanner deliberately has no flag state,
// so the mode pass needs this matching variant before it strips that text.
func javaRegexClassEndForModes(pattern string, start int, modes javaRegexModes) (int, error) {
	type classFrame struct {
		firstElement    bool
		leadingNegation bool
	}
	frames := []classFrame{{firstElement: true}}
	for index := start + 1; index < len(pattern); {
		if pattern[index] == '\\' {
			frames[len(frames)-1].firstElement = false
			index++
			if index < len(pattern) {
				_, size := utf8.DecodeRuneInString(pattern[index:])
				index += size
			}
			continue
		}
		if modes.comments {
			if size := javaRegexIgnoredSpace(pattern[index:]); size > 0 {
				index += size
				continue
			}
			if pattern[index] == '#' {
				index++
				for index < len(pattern) {
					character, size := utf8.DecodeRuneInString(pattern[index:])
					index += size
					if javaRegexCommentTerminator(character) {
						break
					}
				}
				continue
			}
		}
		switch pattern[index] {
		case '[':
			frames[len(frames)-1].firstElement = false
			frames = append(frames, classFrame{firstElement: true})
			index++
		case ']':
			if frames[len(frames)-1].firstElement {
				frames[len(frames)-1].firstElement = false
				index++
				continue
			}
			frames = frames[:len(frames)-1]
			if len(frames) == 0 {
				return index, nil
			}
			index++
		case '^':
			frame := &frames[len(frames)-1]
			if frame.firstElement && !frame.leadingNegation {
				frame.leadingNegation = true
			} else {
				frame.firstElement = false
			}
			index++
		default:
			frames[len(frames)-1].firstElement = false
			_, size := utf8.DecodeRuneInString(pattern[index:])
			index += size
		}
	}
	return 0, fmt.Errorf("unterminated character class")
}

func translateJavaRegexEscape(
	pattern string,
	start int,
	modes javaRegexModes,
	inClass bool,
	properties *javaRegexPropertyMarkers,
) (string, int, error) {
	if start+1 >= len(pattern) {
		return "", 0, fmt.Errorf("escape at end of pattern")
	}
	kind := pattern[start+1]
	predefined := func(positive string, negated bool) string {
		if negated {
			return negateJavaRegexClass(positive)
		}
		return positive
	}
	switch kind {
	case 'c':
		if start+2 >= len(pattern) {
			return "", 0, fmt.Errorf("missing control character")
		}
		character, size := utf8.DecodeRuneInString(pattern[start+2:])
		return regexp2.Escape(string(character ^ 64)), start + 2 + size, nil
	case 'd', 'D':
		value := `[0-9]`
		if modes.unicodeClass {
			value = `[\p{Nd}]`
		}
		return predefined(value, kind == 'D'), start + 2, nil
	case 's', 'S':
		value := `[\x09-\x0d\x20]`
		if modes.unicodeClass {
			value = `[\p{White_Space}]`
		}
		return predefined(value, kind == 'S'), start + 2, nil
	case 'w', 'W':
		fragment := predefined(javaRegexWordClass(modes), kind == 'W')
		if !modes.caseInsensitive {
			return fragment, start + 2, nil
		}
		marker, err := properties.add(fragment)
		if err != nil {
			return "", 0, err
		}
		return marker, start + 2, nil
	case 'h', 'H':
		value := `[\t\x20\u00a0\u1680\u180e\u2000-\u200a\u202f\u205f\u3000]`
		return predefined(value, kind == 'H'), start + 2, nil
	case 'v', 'V':
		value := `[\n\x0b\f\r\u0085\u2028\u2029]`
		return predefined(value, kind == 'V'), start + 2, nil
	case 'R':
		if inClass {
			return "", 0, fmt.Errorf("illegal \\R escape in character class")
		}
		return `(?:\r\n|[\n\x0b\f\r\u0085\u2028\u2029])`, start + 2, nil
	case 'X':
		if inClass {
			return "", 0, fmt.Errorf("illegal \\X escape in character class")
		}
		return `\X`, start + 2, nil
	case 'b':
		if inClass {
			return `\b`, start + 2, nil
		}
		if start+3 < len(pattern) && pattern[start+2:start+4] == "{g" {
			if start+4 >= len(pattern) || pattern[start+4] != '}' {
				return "", 0, fmt.Errorf("malformed Unicode grapheme boundary \\b{g}")
			}
			return `\b{g}`, start + 5, nil
		}
		return javaRegexWordBoundary(modes, false), start + 2, nil
	case 'B':
		if inClass {
			return `\B`, start + 2, nil
		}
		return javaRegexWordBoundary(modes, true), start + 2, nil
	case 'Z':
		if inClass {
			return `\Z`, start + 2, nil
		}
		return javaRegexEndBeforeFinalTerminator(modes), start + 2, nil
	case 'p', 'P':
		if start+2 >= len(pattern) {
			return "", 0, fmt.Errorf("malformed \\%c Unicode property", kind)
		}
		nameStart := start + 2
		end := nameStart
		if pattern[nameStart] == '{' {
			nameStart++
			closing := strings.IndexByte(pattern[nameStart:], '}')
			if closing < 0 {
				return "", 0, fmt.Errorf("unterminated \\%c Unicode property", kind)
			}
			end = nameStart + closing
			if end == nameStart {
				return "", 0, fmt.Errorf("empty \\%c Unicode property", kind)
			}
		} else {
			_, size := utf8.DecodeRuneInString(pattern[nameStart:])
			end = nameStart + size
		}
		fragment, err := javaRegexProperty(pattern[nameStart:end], modes)
		if err != nil {
			return "", 0, err
		}
		if kind == 'P' {
			fragment = negateJavaRegexClass(fragment)
		}
		marker, err := properties.add(fragment)
		if err != nil {
			return "", 0, err
		}
		if pattern[start+2] == '{' {
			end++
		}
		return marker, end, nil
	case '0':
		end := start + 2
		limit := 2
		if end < len(pattern) && pattern[end] >= '0' && pattern[end] <= '3' {
			limit = 3
		}
		value := 0
		digits := 0
		for end < len(pattern) && digits < limit && pattern[end] >= '0' && pattern[end] <= '7' {
			value = value*8 + int(pattern[end]-'0')
			end++
			digits++
		}
		if digits == 0 {
			return "", 0, fmt.Errorf("octal escape requires a digit")
		}
		return fmt.Sprintf(`\x{%x}`, value), end, nil
	case 'N':
		if start+2 >= len(pattern) || pattern[start+2] != '{' {
			return "", 0, fmt.Errorf("illegal Unicode character-name escape")
		}
		end := strings.IndexByte(pattern[start+3:], '}')
		if end < 0 {
			return "", 0, fmt.Errorf("unterminated Unicode character-name escape")
		}
		end += start + 3
		codePoint, err := javaRegexCodePointOf(pattern[start+3 : end])
		if err != nil {
			return "", 0, err
		}
		return fmt.Sprintf(`\x{%x}`, codePoint), end + 1, nil
	default:
		return pattern[start : start+2], start + 2, nil
	}
}

func negateJavaRegexClass(fragment string) string {
	if len(fragment) >= 2 && fragment[0] == '[' && fragment[len(fragment)-1] == ']' {
		body := fragment[1 : len(fragment)-1]
		if strings.HasPrefix(body, "^") {
			return "[" + body[1:] + "]"
		}
		return "[^" + body + "]"
	}
	return `(?:(?!` + fragment + `)[\s\S])`
}

func javaRegexProperty(name string, modes javaRegexModes) (string, error) {
	if separator := strings.IndexByte(name, '='); separator >= 0 {
		// Pattern.family lowercases only the property key. It does not apply
		// Unicode loose-name normalization to either side of the equals sign.
		key, validKey := javaRegexASCIIPropertyKey(name[:separator])
		if !validKey {
			return "", fmt.Errorf("unknown Java Unicode property %q", name)
		}
		value := name[separator+1:]
		switch key {
		case "sc", "script":
			return javaRegexScriptProperty(value)
		case "blk", "block":
			return javaRegexBlockProperty(value)
		case "gc", "general_category":
			return javaRegexExactProperty(value, modes.caseInsensitive)
		default:
			return "", fmt.Errorf("unknown Java Unicode property %q", name)
		}
	}

	if strings.HasPrefix(name, "In") {
		return javaRegexBlockProperty(name[2:])
	}
	if strings.HasPrefix(name, "Is") {
		value := name[2:]
		if property, ok := javaRegexBinaryProperty(value, modes.caseInsensitive); ok {
			return property, nil
		}
		if property, err := javaRegexExactProperty(value, modes.caseInsensitive); err == nil {
			return property, nil
		}
		if property, err := javaRegexScriptProperty(value); err == nil {
			return property, nil
		}
		return "", fmt.Errorf("unknown Java Unicode property %q", name)
	}
	if modes.unicodeClass {
		if property, ok := javaRegexUnicodePOSIXProperty(name, modes.caseInsensitive); ok {
			return property, nil
		}
	}
	if property, err := javaRegexExactProperty(name, modes.caseInsensitive); err == nil {
		return property, nil
	}
	return "", fmt.Errorf("unknown Java Unicode property %q", name)
}

func javaRegexASCIIPropertyKey(value string) (string, bool) {
	var result strings.Builder
	result.Grow(len(value))
	for index := range len(value) {
		current := value[index]
		switch {
		case current >= 'A' && current <= 'Z':
			result.WriteByte(current + ('a' - 'A'))
		case current >= 'a' && current <= 'z' || current == '_':
			result.WriteByte(current)
		default:
			return "", false
		}
	}
	return result.String(), true
}

func javaRegexCategoryProperty(name string, caseInsensitive bool) (string, error) {
	return javaRegexGeneratedCategoryProperty(name, caseInsensitive)
}

func javaRegexCasedCategoryProperty() string {
	property, _ := javaRegexGeneratedCategoryProperty("LC", false)
	return property
}

func javaRegexCasedBinaryProperty() string {
	property, _ := javaRegexGeneratedUnicodeProperty("LOWERCASE", true)
	return property
}

func javaRegexScriptProperty(name string) (string, error) {
	return javaRegexGeneratedScriptProperty(name)
}

func javaRegexBlockProperty(name string) (string, error) {
	return javaRegexGeneratedBlockProperty(name)
}

func javaRegexBinaryProperty(name string, caseInsensitive bool) (string, bool) {
	switch javaRegexRootUpper(name) {
	case "ALPHABETIC", "ASSIGNED", "CONTROL", "EMOJI", "EMOJI_PRESENTATION", "EMOJI_MODIFIER",
		"EMOJI_MODIFIER_BASE", "EMOJI_COMPONENT", "EXTENDED_PICTOGRAPHIC", "HEXDIGIT", "HEX_DIGIT",
		"IDEOGRAPHIC", "JOINCONTROL", "JOIN_CONTROL", "LETTER", "LOWERCASE", "NONCHARACTERCODEPOINT",
		"NONCHARACTER_CODE_POINT", "TITLECASE", "PUNCTUATION", "UPPERCASE", "WHITESPACE", "WHITE_SPACE",
		"WORD", "ALPHA", "LOWER", "UPPER", "SPACE", "PUNCT", "XDIGIT", "ALNUM", "CNTRL", "DIGIT",
		"BLANK", "GRAPH", "PRINT":
		return javaRegexGeneratedUnicodeProperty(name, caseInsensitive)
	default:
		return "", false
	}
}

func javaRegexUnicodePOSIXProperty(name string, caseInsensitive bool) (string, bool) {
	switch javaRegexRootUpper(name) {
	case "ALPHA", "LOWER", "UPPER", "SPACE", "PUNCT", "XDIGIT", "ALNUM", "CNTRL", "DIGIT", "BLANK", "GRAPH", "PRINT":
		return javaRegexGeneratedUnicodeProperty(name, caseInsensitive)
	default:
		return "", false
	}
}

func javaRegexExactProperty(name string, caseInsensitive bool) (string, error) {
	if category, err := javaRegexCategoryProperty(name, caseInsensitive); err == nil {
		return category, nil
	}
	if property, ok := javaRegexASCIIPOSIXProperty(name, caseInsensitive); ok {
		return property, nil
	}
	if property, ok := javaRegexJavaProperty(name, caseInsensitive); ok {
		return property, nil
	}
	switch name {
	case "L1":
		return `[\x00-\xff]`, nil
	case "all":
		return `[\s\S]`, nil
	default:
		return "", fmt.Errorf("unknown Java Unicode property %q", name)
	}
}

func javaRegexASCIIPOSIXProperty(name string, caseInsensitive bool) (string, bool) {
	switch name {
	case "Lower":
		if caseInsensitive {
			return `[A-Za-z]`, true
		}
		return `[a-z]`, true
	case "Upper":
		if caseInsensitive {
			return `[A-Za-z]`, true
		}
		return `[A-Z]`, true
	case "ASCII":
		return `[\x00-\x7f]`, true
	case "Alpha":
		return `[A-Za-z]`, true
	case "Digit":
		return `[0-9]`, true
	case "Alnum":
		return `[A-Za-z0-9]`, true
	case "Punct":
		return `[\x21-\x2f\x3a-\x40\x5b-\x60\x7b-\x7e]`, true
	case "Graph":
		return `[\x21-\x7e]`, true
	case "Print":
		return `[\x20-\x7e]`, true
	case "Blank":
		return `[\t\x20]`, true
	case "Cntrl":
		return `[\x00-\x1f\x7f]`, true
	case "XDigit":
		return `[0-9A-Fa-f]`, true
	case "Space":
		return `[\x09-\x0d\x20]`, true
	default:
		return "", false
	}
}

func javaRegexJavaProperty(name string, caseInsensitive bool) (string, bool) {
	return javaRegexGeneratedJavaProperty(name, caseInsensitive)
}
