package syntax

import (
	"sync"
	"unicode"
	"unicode/utf8"
)

// CaseMode selects the character comparison used by a regex instruction.
// CaseInsensitive retains regexp2's original unicode.ToLower behavior.
// CaseJavaASCII and CaseJavaUnicode implement java.util.regex.Pattern's two
// case-insensitive modes.
type CaseMode uint8

const (
	CaseSensitive CaseMode = iota
	CaseInsensitive
	CaseJavaASCII
	CaseJavaUnicode
)

const customCaseOptions = JavaASCII | JavaUnicode

func caseModeFromOptions(options RegexOptions) CaseMode {
	switch {
	case options&JavaASCII != 0:
		return CaseJavaASCII
	case options&JavaUnicode != 0:
		return CaseJavaUnicode
	case options&IgnoreCase != 0:
		return CaseInsensitive
	default:
		return CaseSensitive
	}
}

// FoldCase returns the comparison key for ch in mode.
func FoldCase(ch rune, mode CaseMode) rune {
	switch mode {
	case CaseInsensitive:
		return unicode.ToLower(ch)
	case CaseJavaASCII:
		if 'A' <= ch && ch <= 'Z' {
			return ch + ('a' - 'A')
		}
		return ch
	case CaseJavaUnicode:
		// OpenJDK's Pattern uses this upper-then-lower key for literals and
		// uses the equivalent pair of operations for backreferences. These
		// mappings are generated from UnicodeData 17.0.0 rather than the Go
		// toolchain's potentially different Unicode tables.
		return javaUnicode17ToLower(javaUnicode17ToUpper(ch))
	default:
		return ch
	}
}

var javaUnicodeClasses struct {
	sync.Once
	byKey map[rune][]rune
}

// javaUnicodeVariants returns the complete Unicode-17 upper-then-lower
// equivalence class for ch. Building the inverse table lazily
// keeps ordinary regexp2 users from paying for Java-specific class matching.
func javaUnicodeVariants(ch rune) []rune {
	javaUnicodeClasses.Do(buildJavaUnicodeClasses)
	key := FoldCase(ch, CaseJavaUnicode)
	if variants := javaUnicodeClasses.byKey[key]; len(variants) != 0 {
		return variants
	}
	return nil
}

func buildJavaUnicodeClasses() {
	classes := make(map[rune][]rune)
	for ch := rune(0); ch <= utf8.MaxRune; ch++ {
		key := FoldCase(ch, CaseJavaUnicode)
		if key == ch && javaUnicode17ToUpper(ch) == ch && javaUnicode17ToLower(ch) == ch {
			continue
		}
		classes[key] = append(classes[key], ch)
	}

	for key, variants := range classes {
		foundKey := false
		for _, ch := range variants {
			if ch == key {
				foundKey = true
				break
			}
		}
		if !foundKey {
			classes[key] = append(variants, key)
		}
	}
	javaUnicodeClasses.byKey = classes
}

type javaUnicode17CaseMapping struct {
	from rune
	to   rune
}

func javaUnicode17ToUpper(ch rune) rune {
	return javaUnicode17CaseLookup(ch, javaUnicode17UpperMappings)
}

func javaUnicode17ToLower(ch rune) rune {
	return javaUnicode17CaseLookup(ch, javaUnicode17LowerMappings)
}

func javaUnicode17CaseLookup(ch rune, mappings []javaUnicode17CaseMapping) rune {
	left, right := 0, len(mappings)
	for left < right {
		middle := left + (right-left)/2
		if mappings[middle].from < ch {
			left = middle + 1
		} else {
			right = middle
		}
	}
	if left < len(mappings) && mappings[left].from == ch {
		return mappings[left].to
	}
	return ch
}
