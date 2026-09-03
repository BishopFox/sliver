package opfor

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
)

// javaRegexRuneRange is the compact, generated representation of a Unicode
// predicate that Go's standard unicode tables do not expose. The generated
// tables are pinned to the Unicode version used by the audited OpenJDK source.
type javaRegexRuneRange struct {
	lo rune
	hi rune
}

type javaRegexUnicodeBlock struct {
	lo       rune
	hi       rune
	javaName string
}

type javaRegexFullCaseMapping struct {
	from rune
	to   string
}

// javaRegexEmojiRanges retains the focused table view used by compatibility
// tests while the generated property registry also carries these predicates.
var javaRegexEmojiRanges = map[string][]javaRegexRuneRange{
	"EMOJI":                 javaRegexPropertyRanges["EMOJI"],
	"EMOJI_PRESENTATION":    javaRegexPropertyRanges["EMOJI_PRESENTATION"],
	"EMOJI_MODIFIER":        javaRegexPropertyRanges["EMOJI_MODIFIER"],
	"EMOJI_MODIFIER_BASE":   javaRegexPropertyRanges["EMOJI_MODIFIER_BASE"],
	"EMOJI_COMPONENT":       javaRegexPropertyRanges["EMOJI_COMPONENT"],
	"EXTENDED_PICTOGRAPHIC": javaRegexPropertyRanges["EXTENDED_PICTOGRAPHIC"],
}

func javaRegexRangesClass(ranges []javaRegexRuneRange) string {
	if len(ranges) == 0 {
		return `[^\s\S]`
	}
	var result strings.Builder
	result.Grow(2 + len(ranges)*18)
	result.WriteByte('[')
	for _, current := range ranges {
		fmt.Fprintf(&result, `\x{%x}`, current.lo)
		if current.hi != current.lo {
			fmt.Fprintf(&result, `-\x{%x}`, current.hi)
		}
	}
	result.WriteByte(']')
	return result.String()
}

func javaRegexGeneratedBlockProperty(name string) (string, error) {
	// Character.UnicodeBlock.forName is case-insensitive but otherwise accepts
	// only the identifier, canonical, and no-space aliases registered by the
	// JDK. Do not apply Unicode property's broader loose-name normalization.
	index, ok := javaRegexUnicodeBlockAliases[javaRegexRootUpper(name)]
	if !ok {
		return "", fmt.Errorf("unsupported Java Unicode block %q", name)
	}
	if index < 0 {
		// Java retains SURROGATES_AREA as an accepted historical block name,
		// but UnicodeBlock.of never returns that aggregate block.
		return `[^\s\S]`, nil
	}
	block := javaRegexUnicodeBlocks[index]
	return javaRegexRangesClass([]javaRegexRuneRange{{lo: block.lo, hi: block.hi}}), nil
}

func javaRegexGeneratedBinaryProperty(name string) (string, bool) {
	key := javaRegexRootUpper(name)
	switch key {
	case "HEXDIGIT", "HEX_DIGIT":
		key = "HEXDIGIT"
	case "JOINCONTROL", "JOIN_CONTROL":
		key = "JOIN_CONTROL"
	case "NONCHARACTERCODEPOINT", "NONCHARACTER_CODE_POINT":
		key = "NONCHARACTER_CODE_POINT"
	case "WHITESPACE", "WHITE_SPACE":
		key = "WHITE_SPACE"
	}
	switch key {
	case "ALPHABETIC", "ASSIGNED", "CONTROL", "EMOJI", "EMOJI_PRESENTATION",
		"EMOJI_MODIFIER", "EMOJI_MODIFIER_BASE", "EMOJI_COMPONENT",
		"EXTENDED_PICTOGRAPHIC", "HEXDIGIT", "IDEOGRAPHIC", "JOIN_CONTROL",
		"LETTER", "LOWERCASE", "NONCHARACTER_CODE_POINT", "TITLECASE",
		"PUNCTUATION", "UPPERCASE", "WHITE_SPACE", "WORD", "ALNUM",
		"BLANK", "GRAPH", "PRINT", "DIGIT":
		// Public CharPredicates Unicode/binary and Unicode POSIX names.
	default:
		return "", false
	}
	ranges, ok := javaRegexPropertyRanges[key]
	if !ok {
		return "", false
	}
	return javaRegexRangesClass(ranges), true
}

// javaRegexGeneratedUnicodeProperty mirrors CharPredicates.forUnicodeProperty:
// the name is full-uppercased with Locale.ROOT semantics, underscores remain
// significant, and Unicode POSIX names are consulted after binary names.
func javaRegexGeneratedUnicodeProperty(name string, caseInsensitive bool) (string, bool) {
	key := javaRegexRootUpper(name)
	if caseInsensitive {
		switch key {
		case "LOWERCASE", "UPPERCASE", "TITLECASE", "LOWER", "UPPER":
			return javaRegexRangesClass(javaRegexPropertyRanges["CASED"]), true
		}
	}
	switch key {
	case "ALPHA":
		key = "ALPHABETIC"
	case "LOWER":
		key = "LOWERCASE"
	case "UPPER":
		key = "UPPERCASE"
	case "SPACE":
		key = "WHITE_SPACE"
	case "PUNCT":
		key = "PUNCTUATION"
	case "XDIGIT":
		key = "HEXDIGIT"
	case "CNTRL":
		key = "CONTROL"
	}
	return javaRegexGeneratedBinaryProperty(key)
}

// javaRegexGeneratedCategoryProperty performs OpenJDK forProperty's exact
// general-category lookup. It deliberately does not accept UCD loose aliases.
func javaRegexGeneratedCategoryProperty(name string, caseInsensitive bool) (string, error) {
	if caseInsensitive && (name == "Ll" || name == "Lu" || name == "Lt") {
		name = "LC"
	}
	ranges, ok := javaRegexCategoryRanges[name]
	if !ok {
		return "", fmt.Errorf("unknown Java Unicode category %q", name)
	}
	return javaRegexRangesClass(ranges), nil
}

// javaRegexGeneratedScriptProperty follows Character.UnicodeScript.forName:
// exact enum identifiers and ISO 15924 aliases are case-insensitive, but
// spaces, hyphens, and underscores are not interchangeable.
func javaRegexGeneratedScriptProperty(name string) (string, error) {
	canonical, ok := javaRegexScriptAliases[javaRegexRootUpper(name)]
	if !ok {
		return "", fmt.Errorf("unsupported Java Unicode script %q", name)
	}
	return javaRegexRangesClass(javaRegexScriptRanges[canonical]), nil
}

func javaRegexGeneratedWordClass() string {
	return javaRegexRangesClass(javaRegexPropertyRanges["WORD"])
}

// javaRegexGeneratedJavaProperty returns the fixed Unicode-17 predicates
// behind java.lang.Character's boolean-property family.
func javaRegexGeneratedJavaProperty(name string, caseInsensitive bool) (string, bool) {
	key := ""
	switch name {
	case "javaLowerCase":
		key = "LOWERCASE"
	case "javaUpperCase":
		key = "UPPERCASE"
	case "javaTitleCase":
		key = "TITLECASE"
	case "javaWhitespace":
		key = "JAVA_WHITESPACE"
	case "javaAlphabetic":
		key = "ALPHABETIC"
	case "javaIdeographic":
		key = "IDEOGRAPHIC"
	case "javaDigit":
		key = "DIGIT"
	case "javaDefined":
		key = "ASSIGNED"
	case "javaLetter":
		key = "LETTER"
	case "javaLetterOrDigit":
		key = "LD"
	case "javaJavaIdentifierStart":
		key = "JAVA_IDENTIFIER_START"
	case "javaJavaIdentifierPart":
		key = "JAVA_IDENTIFIER_PART"
	case "javaUnicodeIdentifierStart":
		key = "UNICODE_IDENTIFIER_START"
	case "javaUnicodeIdentifierPart":
		key = "UNICODE_IDENTIFIER_PART"
	case "javaIdentifierIgnorable":
		key = "IDENTIFIER_IGNORABLE"
	case "javaSpaceChar":
		key = "SPACECHAR"
	case "javaISOControl":
		key = "ISOCONTROL"
	case "javaMirrored":
		return javaRegexRangesClass(javaRegexMirroredRanges), true
	default:
		return "", false
	}
	if caseInsensitive && (key == "LOWERCASE" || key == "UPPERCASE" || key == "TITLECASE") {
		key = "CASED"
	}
	if key == "LD" {
		return javaRegexRangesClass(javaRegexCategoryRanges["LD"]), true
	}
	ranges, ok := javaRegexPropertyRanges[key]
	if !ok {
		return "", false
	}
	return javaRegexRangesClass(ranges), true
}

// javaRegexRootUpper implements the unconditional full uppercase mapping used
// by String.toUpperCase(Locale.ROOT), including multi-code-point expansions.
// The generated table is fixed at Unicode 17.0.0 and is independent of Go's
// toolchain Unicode version.
func javaRegexRootUpper(value string) string {
	var result strings.Builder
	result.Grow(len(value))
	for _, current := range value {
		if mapped, ok := javaRegexRootUpperMapping(current); ok {
			result.WriteString(mapped)
		} else {
			result.WriteRune(current)
		}
	}
	return result.String()
}

func javaRegexRootUpperMapping(value rune) (string, bool) {
	left, right := 0, len(javaRegexRootUpperMappings)
	for left < right {
		middle := left + (right-left)/2
		if javaRegexRootUpperMappings[middle].from < value {
			left = middle + 1
		} else {
			right = middle
		}
	}
	if left < len(javaRegexRootUpperMappings) && javaRegexRootUpperMappings[left].from == value {
		return javaRegexRootUpperMappings[left].to, true
	}
	return "", false
}

var javaRegexCharacterNames struct {
	sync.Once
	byName map[string]rune
	named  map[rune]struct{}
	err    error
}

func loadJavaRegexCharacterNames() (map[string]rune, map[rune]struct{}, error) {
	javaRegexCharacterNames.Do(func() {
		compressed, err := base64.StdEncoding.DecodeString(javaRegexCharacterNamesZlibBase64)
		if err != nil {
			javaRegexCharacterNames.err = fmt.Errorf("decode generated Java character names: %w", err)
			return
		}
		reader, err := zlib.NewReader(bytes.NewReader(compressed))
		if err != nil {
			javaRegexCharacterNames.err = fmt.Errorf("open generated Java character names: %w", err)
			return
		}
		plain, readErr := io.ReadAll(io.LimitReader(reader, 8<<20+1))
		closeErr := reader.Close()
		if readErr != nil {
			javaRegexCharacterNames.err = fmt.Errorf("read generated Java character names: %w", readErr)
			return
		}
		if closeErr != nil {
			javaRegexCharacterNames.err = fmt.Errorf("close generated Java character names: %w", closeErr)
			return
		}
		if len(plain) > 8<<20 || len(plain) < 4 {
			javaRegexCharacterNames.err = fmt.Errorf("generated Java character-name table has invalid size %d", len(plain))
			return
		}

		count := int(binary.BigEndian.Uint32(plain[:4]))
		byName := make(map[string]rune, count)
		named := make(map[rune]struct{}, count)
		offset := 4
		for range count {
			if offset+6 > len(plain) {
				javaRegexCharacterNames.err = fmt.Errorf("generated Java character-name table is truncated")
				return
			}
			cp := rune(binary.BigEndian.Uint32(plain[offset : offset+4]))
			nameLength := int(binary.BigEndian.Uint16(plain[offset+4 : offset+6]))
			offset += 6
			if nameLength == 0 || offset+nameLength > len(plain) || cp < 0 || cp > 0x10ffff {
				javaRegexCharacterNames.err = fmt.Errorf("generated Java character-name record is invalid")
				return
			}
			name := string(plain[offset : offset+nameLength])
			offset += nameLength
			if _, duplicate := byName[name]; duplicate {
				javaRegexCharacterNames.err = fmt.Errorf("generated Java character name %q is duplicated", name)
				return
			}
			byName[name] = cp
			named[cp] = struct{}{}
		}
		if offset != len(plain) {
			javaRegexCharacterNames.err = fmt.Errorf("generated Java character-name table has trailing data")
			return
		}
		javaRegexCharacterNames.byName = byName
		javaRegexCharacterNames.named = named
	})
	return javaRegexCharacterNames.byName, javaRegexCharacterNames.named, javaRegexCharacterNames.err
}

// javaRegexCodePointOf mirrors Character.codePointOf for the Unicode 17 data
// bundled by the audited OpenJDK revision. Direct UnicodeData names (including
// OpenJDK's control-name choices) are checked first. Assigned characters that
// lack such a name use Character.getName's "BLOCK HEX" fallback.
func javaRegexCodePointOf(value string) (rune, error) {
	name := javaRegexRootUpper(javaRegexJavaTrim(value))
	for _, current := range name {
		if current > 0x7f {
			return 0, fmt.Errorf("unknown Unicode character name %q", value)
		}
	}
	byName, named, err := loadJavaRegexCharacterNames()
	if err != nil {
		return 0, err
	}
	if cp, ok := byName[name]; ok {
		return cp, nil
	}

	separator := strings.LastIndexByte(name, ' ')
	if separator < 0 || separator+1 == len(name) {
		return 0, fmt.Errorf("unknown Unicode character name %q", value)
	}
	parsed, parseErr := strconv.ParseUint(name[separator+1:], 16, 21)
	if parseErr != nil || parsed > 0x10ffff {
		return 0, fmt.Errorf("unknown Unicode character name %q", value)
	}
	cp := rune(parsed)
	if _, hasDirectName := named[cp]; hasDirectName || !javaRegexRuneInRanges(cp, javaRegexAssignedRanges) {
		return 0, fmt.Errorf("unknown Unicode character name %q", value)
	}
	block, ok := javaRegexBlockForRune(cp)
	if !ok {
		return 0, fmt.Errorf("unknown Unicode character name %q", value)
	}
	want := strings.ReplaceAll(block.javaName, "_", " ") + " " + strings.ToUpper(strconv.FormatInt(int64(cp), 16))
	if name != want {
		return 0, fmt.Errorf("unknown Unicode character name %q", value)
	}
	return cp, nil
}

// String.trim, used by Character.codePointOf, removes only code units at or
// below U+0020. It is deliberately narrower than strings.TrimSpace.
func javaRegexJavaTrim(value string) string {
	start, end := 0, len(value)
	for start < end && value[start] <= 0x20 {
		start++
	}
	for end > start && value[end-1] <= 0x20 {
		end--
	}
	return value[start:end]
}

func javaRegexRuneInRanges(value rune, ranges []javaRegexRuneRange) bool {
	left, right := 0, len(ranges)
	for left < right {
		middle := left + (right-left)/2
		if ranges[middle].hi < value {
			left = middle + 1
		} else {
			right = middle
		}
	}
	return left < len(ranges) && ranges[left].lo <= value
}

func javaRegexBlockForRune(value rune) (javaRegexUnicodeBlock, bool) {
	left, right := 0, len(javaRegexUnicodeBlocks)
	for left < right {
		middle := left + (right-left)/2
		if javaRegexUnicodeBlocks[middle].hi < value {
			left = middle + 1
		} else {
			right = middle
		}
	}
	if left >= len(javaRegexUnicodeBlocks) || javaRegexUnicodeBlocks[left].lo > value {
		return javaRegexUnicodeBlock{}, false
	}
	return javaRegexUnicodeBlocks[left], true
}
