package opfor

import (
	"os"
	goruntime "runtime"
	"unicode"
	"unicode/utf8"
)

const portableJavaURIClass = "java.net.URI"

// portableJavaURI is the immutable, hierarchical file URI constructed by
// java.io.File.toURI. It intentionally does not pretend to be a general URI
// parser; every instance has scheme "file", no authority/query/fragment, and
// an absolute path quoted by URI's component constructor.
type portableJavaURI struct {
	path     Value
	rawPath  Value
	external Value
}

func (uri *portableJavaURI) String() string {
	if uri == nil {
		return ""
	}
	return uri.external.String()
}

func portableJavaURIFromFile(file *portableJavaFile) (*portableJavaURI, error) {
	path, err := portableJavaFileSlashified(file)
	if err != nil {
		return nil, err
	}
	units := sleepStringUnits(path)
	if len(units) >= 2 && units[0] == '/' && units[1] == '/' {
		// File.toURI protects a UNC pathname from being parsed as an authority.
		path = sleepStringConcat(String("//"), path)
	}
	rawPath := portableJavaURIQuotePath(path)
	return &portableJavaURI{
		path:     path,
		rawPath:  rawPath,
		external: sleepStringConcat(String("file:"), rawPath),
	}, nil
}

// portableJavaFileSlashified mirrors the pinned OpenJDK File.slashify helper:
// make the absolute pathname use '/', ensure a leading slash, and preserve a
// trailing slash only for a directory that exists at conversion time.
func portableJavaFileSlashified(file *portableJavaFile) (Value, error) {
	absolute, err := portableJavaFileAbsoluteValue(file.pathValue())
	if err != nil {
		return Null(), err
	}
	units := sleepStringUnits(absolute)
	if goruntime.GOOS == "windows" {
		for index, unit := range units {
			if unit == '\\' {
				units[index] = '/'
			}
		}
	}
	if len(units) == 0 || units[0] != '/' {
		units = append([]uint16{'/'}, units...)
	}
	info, statErr := os.Stat(portableJavaFileFilesystemPathValue(absolute))
	if statErr == nil && info.IsDir() && units[len(units)-1] != '/' {
		units = append(units, '/')
	}
	return sleepStringValueFromUnits(units, nil), nil
}

func portableJavaURIQuotePath(path Value) Value {
	units := sleepStringUnits(path)
	quoted := make([]uint16, 0, len(units))
	for _, unit := range units {
		if unit < 0x80 {
			if portableJavaURIPathASCIIAllowed(byte(unit)) {
				quoted = append(quoted, unit)
			} else {
				quoted = portableJavaURIAppendEscapedByte(quoted, byte(unit))
			}
			continue
		}
		character := rune(unit)
		if portableJavaURIIsSpaceChar(character) || portableJavaURIIsISOControl(unit) {
			var encoded [utf8.UTFMax]byte
			width := utf8.EncodeRune(encoded[:], character)
			for _, octet := range encoded[:width] {
				quoted = portableJavaURIAppendEscapedByte(quoted, octet)
			}
			continue
		}
		// URI.quote operates on Java chars. Ordinary non-ASCII chars, including
		// each half of a supplementary pair, remain present in the raw URI.
		quoted = append(quoted, unit)
	}
	return sleepStringValueFromUnits(quoted, nil)
}

func portableJavaURIPathASCIIAllowed(character byte) bool {
	if character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || character >= '0' && character <= '9' {
		return true
	}
	return containsASCII("_-!.~'()*:@&=+$,;/", character)
}

func containsASCII(characters string, target byte) bool {
	for index := 0; index < len(characters); index++ {
		if characters[index] == target {
			return true
		}
	}
	return false
}

func portableJavaURIAppendEscapedByte(destination []uint16, octet byte) []uint16 {
	const hexadecimal = "0123456789ABCDEF"
	return append(destination, '%', uint16(hexadecimal[octet>>4]), uint16(hexadecimal[octet&0x0f]))
}

func portableJavaURIIsSpaceChar(character rune) bool {
	return unicode.Is(unicode.Zs, character) || unicode.Is(unicode.Zl, character) || unicode.Is(unicode.Zp, character)
}

func portableJavaURIIsISOControl(unit uint16) bool {
	return unit <= 0x1f || unit >= 0x7f && unit <= 0x9f
}

func (uri *portableJavaURI) invoke(invocation ObjectInvocation) (Value, bool, error) {
	if uri == nil {
		return Null(), false, nil
	}
	if invocation.Op == ObjectTypeCheck {
		class := resolvePortableClassName(invocation.Class)
		switch class {
		case portableJavaURIClass, "java.lang.Object", "java.lang.Comparable", "java.io.Serializable":
			return Bool(true), true, nil
		default:
			return Bool(false), true, nil
		}
	}
	if invocation.Op != ObjectInvoke {
		return Null(), false, nil
	}
	if invocation.Message == "equals" && len(invocation.Arguments) == 1 {
		other, ok := portableJavaURIValue(invocation.Arg(0))
		return portableJavaFileBoolean(ok && portableJavaURIPathEqual(uri.rawPath, other.rawPath)), true, nil
	}
	if invocation.Message == "compareTo" && len(invocation.Arguments) == 1 {
		other, ok := portableJavaURIValue(invocation.Arg(0))
		if !ok {
			return Null(), false, nil
		}
		return Int(portableJavaURIPathCompare(uri.rawPath, other.rawPath)), true, nil
	}
	if len(invocation.Arguments) != 0 {
		return Null(), false, nil
	}
	switch invocation.Message {
	case "toString":
		return uri.external, true, nil
	case "toASCIIString":
		// OpenJDK NFC-normalizes before encoding non-ASCII. Go's standard
		// library has no Unicode normalization implementation, so only the
		// exact ASCII branch is supplied; an importer may own the other case.
		for _, unit := range sleepStringUnits(uri.external) {
			if unit >= 0x80 {
				return Null(), false, nil
			}
		}
		return uri.external, true, nil
	case "getScheme":
		return String("file"), true, nil
	case "getRawSchemeSpecificPart", "getRawPath":
		return uri.rawPath, true, nil
	case "getSchemeSpecificPart", "getPath":
		return uri.path, true, nil
	case "getRawAuthority", "getAuthority", "getRawUserInfo", "getUserInfo", "getHost",
		"getRawQuery", "getQuery", "getRawFragment", "getFragment":
		return Null(), true, nil
	case "getPort":
		return Int(-1), true, nil
	case "isAbsolute":
		return portableJavaFileBoolean(true), true, nil
	case "isOpaque":
		return portableJavaFileBoolean(false), true, nil
	case "hashCode":
		return Int(portableJavaURIHash(uri.rawPath)), true, nil
	default:
		return Null(), false, nil
	}
}

func portableJavaURIValue(value Value) (*portableJavaURI, bool) {
	object, ok := value.Object()
	if !ok {
		return nil, false
	}
	uri, ok := object.(*portableJavaURI)
	return uri, ok && uri != nil
}

func portableJavaURIPathEqual(left, right Value) bool {
	return portableJavaURIPathCompare(left, right) == 0
}

// File-created URIs always use uppercase hex in their generated escapes, so
// UTF-16 comparison is the exact percent-normalized comparison for this closed
// object family.
func portableJavaURIPathCompare(left, right Value) int32 {
	return portableJavaStringCompareValues(left, right)
}

func portableJavaURIHash(rawPath Value) int32 {
	var hash int32
	for _, character := range "file" {
		hash = 31*hash + int32(character)
	}
	return portableJavaURIComponentHash(hash, rawPath)
}

func portableJavaURIComponentHash(initial int32, value Value) int32 {
	units := sleepStringUnits(value)
	hasEscape := false
	for _, unit := range units {
		if unit == '%' {
			hasEscape = true
			break
		}
	}
	if !hasEscape {
		return initial*127 + portableJavaStringHashValue(value)
	}
	var component int32
	for index := 0; index < len(units); index++ {
		unit := units[index]
		component = 31*component + int32(unit)
		if unit == '%' && index+2 < len(units) {
			for offset := 1; offset <= 2; offset++ {
				hex := units[index+offset]
				if hex >= 'a' && hex <= 'f' {
					hex -= 'a' - 'A'
				}
				component = 31*component + int32(hex)
			}
			index += 2
		}
	}
	return initial*127 + component
}

func portableJavaStringHashValue(value Value) int32 {
	var hash int32
	for _, unit := range sleepStringUnits(value) {
		hash = 31*hash + int32(unit)
	}
	return hash
}
