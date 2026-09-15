package opfor

import (
	"context"
	"errors"
	"math"
	"strings"
	"unicode/utf16"
)

// portableJavaLocale is the immutable base-field Locale shape needed by
// String's explicit case overloads. The two legacy constructor compatibility
// extensions are retained; Locale.Builder, arbitrary extensions, display-name
// providers, and machine-default mutation remain outside this bounded model.
type portableJavaLocale struct {
	language  string
	country   string
	variant   string
	extension string
}

var portableJavaLocaleConstants = func() map[string]*portableJavaLocale {
	locales := map[string]*portableJavaLocale{
		"ROOT":                {},
		"ENGLISH":             {language: "en"},
		"FRENCH":              {language: "fr"},
		"GERMAN":              {language: "de"},
		"ITALIAN":             {language: "it"},
		"JAPANESE":            {language: "ja"},
		"KOREAN":              {language: "ko"},
		"CHINESE":             {language: "zh"},
		"SIMPLIFIED_CHINESE":  {language: "zh", country: "CN"},
		"TRADITIONAL_CHINESE": {language: "zh", country: "TW"},
		"FRANCE":              {language: "fr", country: "FR"},
		"GERMANY":             {language: "de", country: "DE"},
		"ITALY":               {language: "it", country: "IT"},
		"JAPAN":               {language: "ja", country: "JP"},
		"KOREA":               {language: "ko", country: "KR"},
		"UK":                  {language: "en", country: "GB"},
		"US":                  {language: "en", country: "US"},
		"CANADA":              {language: "en", country: "CA"},
		"CANADA_FRENCH":       {language: "fr", country: "CA"},
	}
	locales["CHINA"] = locales["SIMPLIFIED_CHINESE"]
	locales["PRC"] = locales["SIMPLIFIED_CHINESE"]
	locales["TAIWAN"] = locales["TRADITIONAL_CHINESE"]
	return locales
}()

func (locale *portableJavaLocale) String() string {
	if locale == nil {
		return "null"
	}
	var result strings.Builder
	result.WriteString(locale.language)
	if locale.country != "" || locale.language != "" && locale.variant != "" {
		result.WriteByte('_')
		result.WriteString(locale.country)
	}
	if locale.variant != "" && (locale.language != "" || locale.country != "") {
		result.WriteByte('_')
		result.WriteString(locale.variant)
	}
	if locale.extension != "" && (locale.language != "" || locale.country != "") {
		result.WriteString("_#")
		result.WriteString(locale.extension)
	}
	return result.String()
}

func (locale *portableJavaLocale) languageTag() string {
	if locale == nil {
		return ""
	}
	parts := make([]string, 0, 3)
	if !portableJavaLocaleLanguageSubtag(locale.language) {
		parts = append(parts, "und")
	} else {
		parts = append(parts, locale.language)
	}
	if portableJavaLocaleRegionSubtag(locale.country) {
		parts = append(parts, locale.country)
	}
	var private []string
	if locale.variant != "" {
		variants := strings.Split(locale.variant, "_")
		index := 0
		for index < len(variants) && portableJavaLocaleVariantSubtag(variants[index]) {
			parts = append(parts, variants[index])
			index++
		}
		if index < len(variants) {
			private = variants[index:]
			valid := true
			for _, subtag := range private {
				if len(subtag) < 1 || len(subtag) > 8 || !portableJavaLocaleASCIIAlnum(subtag) {
					valid = false
					break
				}
			}
			if !valid {
				private = nil
			}
		}
	}
	if locale.extension != "" {
		parts = append(parts, strings.Split(locale.extension, "-")...)
	}
	if len(private) != 0 {
		parts = append(parts, "x", "lvariant")
		parts = append(parts, private...)
	}
	return strings.Join(parts, "-")
}

func portableJavaLocaleLanguageSubtag(value string) bool {
	return len(value) >= 2 && len(value) <= 8 && portableJavaLocaleASCIIAlpha(value)
}

func portableJavaLocaleRegionSubtag(value string) bool {
	if len(value) == 2 {
		return portableJavaLocaleASCIIAlpha(value)
	}
	if len(value) != 3 {
		return false
	}
	for index := 0; index < len(value); index++ {
		if value[index] < '0' || value[index] > '9' {
			return false
		}
	}
	return true
}

func portableJavaLocaleASCIIAlpha(value string) bool {
	for index := 0; index < len(value); index++ {
		character := value[index]
		if character < 'A' || character > 'Z' {
			if character < 'a' || character > 'z' {
				return false
			}
		}
	}
	return true
}

func portableJavaLocaleVariantSubtag(value string) bool {
	if len(value) >= 5 && len(value) <= 8 {
		return portableJavaLocaleASCIIAlnum(value)
	}
	return len(value) == 4 && value[0] >= '0' && value[0] <= '9' && portableJavaLocaleASCIIAlnum(value)
}

func portableJavaLocaleASCIIAlnum(value string) bool {
	for index := 0; index < len(value); index++ {
		character := value[index]
		if character < '0' || character > '9' {
			if character < 'A' || character > 'Z' {
				if character < 'a' || character > 'z' {
					return false
				}
			}
		}
	}
	return true
}

func (locale *portableJavaLocale) hashCode() int32 {
	if locale == nil {
		return 0
	}
	var hash int32
	for fieldIndex, field := range []string{locale.language, "", locale.country, locale.variant} {
		for _, unit := range utf16.Encode([]rune(field)) {
			character := unit
			if fieldIndex != 3 && character >= 'A' && character <= 'Z' {
				character += 'a' - 'A'
			}
			hash = hash*31 + int32(character)
		}
	}
	if locale.extension != "" {
		hash ^= portableJavaStringHash(locale.extension)
	}
	return hash
}

func portableJavaStringHash(value string) int32 {
	var hash int32
	for _, unit := range utf16.Encode([]rune(value)) {
		hash = hash*31 + int32(unit)
	}
	return hash
}

func (locale *portableJavaLocale) invoke(invocation ObjectInvocation) (Value, bool, error) {
	if locale == nil {
		return Null(), false, nil
	}
	if invocation.Op == ObjectTypeCheck {
		return Bool(portableJavaAssignable("java.util.Locale", invocation.Class)), true, nil
	}
	if invocation.Op != ObjectInvoke {
		return Null(), false, nil
	}
	switch invocation.Message {
	case "toString":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.util.Locale"), true, nil
		}
		return String(locale.String()), true, nil
	case "toLanguageTag":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.util.Locale"), true, nil
		}
		return String(locale.languageTag()), true, nil
	case "getLanguage":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.util.Locale"), true, nil
		}
		return String(locale.language), true, nil
	case "getCountry":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.util.Locale"), true, nil
		}
		return String(locale.country), true, nil
	case "getVariant":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.util.Locale"), true, nil
		}
		return String(locale.variant), true, nil
	case "getScript":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.util.Locale"), true, nil
		}
		return String(""), true, nil
	case "hasExtensions":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.util.Locale"), true, nil
		}
		return portableJavaBooleanValue(locale.extension != ""), true, nil
	case "hashCode":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.util.Locale"), true, nil
		}
		return Int(locale.hashCode()), true, nil
	case "equals":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "java.util.Locale"), true, nil
		}
		otherObject, ok := invocation.Arg(0).Object()
		other, ok := otherObject.(*portableJavaLocale)
		return portableJavaBooleanValue(ok && other != nil && *locale == *other), true, nil
	}
	return Null(), false, nil
}

func portableJavaLocaleClass(invocation ObjectInvocation) (Value, bool, error) {
	if portableJavaClassName(invocation.Class) != "Locale" {
		return Null(), false, nil
	}
	if invocation.Op == ObjectConstruct {
		if len(invocation.Arguments) < 1 || len(invocation.Arguments) > 3 {
			message := "no constructor matching java.util.Locale(" + portableObjectArgumentList(invocation) + ")"
			return portableObjectWarning(invocation, message), true, nil
		}
		parts := make([]string, len(invocation.Arguments))
		for index := range invocation.Arguments {
			value := invocation.Arg(index)
			if value.IsNull() {
				return Null(), true, errors.New("java.lang.NullPointerException")
			}
			parts[index] = value.String()
		}
		locale := &portableJavaLocale{language: portableJavaLocaleLanguage(parts[0])}
		if len(parts) >= 2 {
			locale.country = portableJavaLocaleUpperASCII(parts[1])
		}
		if len(parts) == 3 {
			locale.variant = parts[2]
		}
		if locale.language == "ja" && locale.country == "JP" && locale.variant == "JP" {
			locale.extension = "u-ca-japanese"
		}
		if locale.language == "th" && locale.country == "TH" && locale.variant == "TH" {
			locale.extension = "u-nu-thai"
		}
		return ObjectValue(locale), true, nil
	}
	if invocation.Op != ObjectGet && invocation.Op != ObjectInvoke || len(invocation.Arguments) != 0 {
		return Null(), false, nil
	}
	locale := portableJavaLocaleConstants[invocation.Message]
	if locale == nil {
		return Null(), false, nil
	}
	return ObjectValue(locale), true, nil
}

func portableJavaLocaleLanguage(value string) string {
	value = portableJavaLocaleLowerASCII(value)
	switch value {
	case "iw":
		return "he"
	case "ji":
		return "yi"
	case "in":
		return "id"
	default:
		return value
	}
}

func portableJavaLocaleLowerASCII(value string) string {
	result := []byte(value)
	for index, character := range result {
		if character >= 'A' && character <= 'Z' {
			result[index] += 'a' - 'A'
		}
	}
	return string(result)
}

func portableJavaLocaleUpperASCII(value string) string {
	result := []byte(value)
	for index, character := range result {
		if character >= 'a' && character <= 'z' {
			result[index] -= 'a' - 'A'
		}
	}
	return string(result)
}

func portableJavaStringCase(
	ctx context.Context,
	invocation ObjectInvocation,
	target Value,
	upper bool,
) (Value, bool, error) {
	if len(invocation.Arguments) > 1 {
		return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
	}
	language := ""
	if len(invocation.Arguments) == 1 {
		value := invocation.Arg(0)
		if value.IsNull() {
			return Null(), true, errors.New("java.lang.NullPointerException")
		}
		object, ok := value.Object()
		locale, ok := object.(*portableJavaLocale)
		if !ok || locale == nil {
			return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
		}
		language = locale.language
	}
	result, err := sleepStringMapCaseLocale(ctx, target, upper, language)
	return result, true, err
}

func sleepStringMapCaseLocale(ctx context.Context, value Value, upper bool, language string) (Value, error) {
	value = sleepStringCoercion(value)
	units := sleepStringUnits(value)
	raw := sleepStringRawMask(value)
	resultUnits := make([]uint16, 0, len(units))
	resultRaw := make([]bool, 0, len(raw))
	work := &portableJavaStringWork{ctx: ctx}
	changed := false
	for index := 0; index < len(units); {
		codePoint, width := sleepUTF16CodePointAt(units, index)
		if err := work.advance(width); err != nil {
			return Null(), err
		}
		mapped, ok := sleepJavaLocaleCaseMapping(units, index, width, rune(codePoint), upper, language)
		if !ok {
			mapped, ok = sleepJavaRootCaseMapping(rune(codePoint), upper)
			if !upper && codePoint == 0x03a3 && sleepJavaFinalSigma(units, index, width) {
				mapped, ok = "\u03c2", true
			}
		}
		if !ok {
			resultUnits = append(resultUnits, units[index:index+width]...)
			resultRaw = append(resultRaw, raw[index:index+width]...)
			index += width
			continue
		}
		mappedUnits := utf16.Encode([]rune(mapped))
		if equalUTF16Units(mappedUnits, units[index:index+width]) {
			resultUnits = append(resultUnits, units[index:index+width]...)
			resultRaw = append(resultRaw, raw[index:index+width]...)
		} else {
			changed = true
			if int64(len(resultUnits))+int64(len(mappedUnits)) > math.MaxInt32 {
				return Null(), errors.New("java.lang.OutOfMemoryError: Required length exceeds implementation limit")
			}
			resultUnits = append(resultUnits, mappedUnits...)
			resultRaw = append(resultRaw, make([]bool, len(mappedUnits))...)
		}
		index += width
	}
	if err := work.finish(); err != nil {
		return Null(), err
	}
	if !changed {
		return value, nil
	}
	return sleepStringValueFromUnits(resultUnits, resultRaw), nil
}

func sleepJavaLocaleCaseMapping(
	units []uint16,
	index, width int,
	codePoint rune,
	upper bool,
	language string,
) (string, bool) {
	switch language {
	case "tr", "az":
		if upper && codePoint == 0x0069 {
			return "\u0130", true
		}
		if !upper {
			switch codePoint {
			case 0x0130:
				return "i", true
			case 0x0307:
				if sleepJavaLocaleAfterI(units, index) {
					return "", true
				}
			case 0x0049:
				if !sleepJavaLocaleBeforeDot(units, index, width) {
					return "\u0131", true
				}
			}
		}
	case "lt":
		if upper && codePoint == 0x0307 && sleepJavaLocaleAfterSoftDotted(units, index) {
			return "", true
		}
		if !upper {
			switch codePoint {
			case 0x0049:
				if sleepJavaLocaleMoreAbove(units, index, width) {
					return "i\u0307", true
				}
			case 0x004a:
				if sleepJavaLocaleMoreAbove(units, index, width) {
					return "j\u0307", true
				}
			case 0x012e:
				if sleepJavaLocaleMoreAbove(units, index, width) {
					return "\u012f\u0307", true
				}
			case 0x00cc:
				return "i\u0307\u0300", true
			case 0x00cd:
				return "i\u0307\u0301", true
			case 0x0128:
				return "i\u0307\u0303", true
			}
		}
	}
	return "", false
}

func sleepJavaLocaleCombiningClass(codePoint rune) int {
	if javaRegexRuneInRanges(codePoint, javaStringCombiningClassAboveRanges) {
		return 230
	}
	if javaRegexRuneInRanges(codePoint, javaStringCombiningClassNonZeroRanges) {
		return 1
	}
	return 0
}

func sleepJavaLocaleAfterI(units []uint16, index int) bool {
	for current := index; current > 0; {
		codePoint, width := sleepUTF16CodePointBefore(units, current)
		current -= width
		if codePoint == 'I' {
			return true
		}
		combiningClass := sleepJavaLocaleCombiningClass(rune(codePoint))
		if combiningClass == 0 || combiningClass == 230 {
			return false
		}
	}
	return false
}

func sleepJavaLocaleAfterSoftDotted(units []uint16, index int) bool {
	for current := index; current > 0; {
		codePoint, width := sleepUTF16CodePointBefore(units, current)
		current -= width
		if javaRegexRuneInRanges(rune(codePoint), javaStringSoftDottedRanges) {
			return true
		}
		combiningClass := sleepJavaLocaleCombiningClass(rune(codePoint))
		if combiningClass == 0 || combiningClass == 230 {
			return false
		}
	}
	return false
}

func sleepJavaLocaleMoreAbove(units []uint16, index, width int) bool {
	for current := index + width; current < len(units); {
		codePoint, codePointWidth := sleepUTF16CodePointAt(units, current)
		combiningClass := sleepJavaLocaleCombiningClass(rune(codePoint))
		if combiningClass == 230 {
			return true
		}
		if combiningClass == 0 {
			return false
		}
		current += codePointWidth
	}
	return false
}

func sleepJavaLocaleBeforeDot(units []uint16, index, width int) bool {
	for current := index + width; current < len(units); {
		codePoint, codePointWidth := sleepUTF16CodePointAt(units, current)
		if codePoint == 0x0307 {
			return true
		}
		combiningClass := sleepJavaLocaleCombiningClass(rune(codePoint))
		if combiningClass == 0 || combiningClass == 230 {
			return false
		}
		current += codePointWidth
	}
	return false
}
