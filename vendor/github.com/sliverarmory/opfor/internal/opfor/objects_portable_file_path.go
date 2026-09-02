package opfor

import (
	"fmt"
	goruntime "runtime"
)

const (
	portableJavaPathInterface      = "java.nio.file.Path"
	portableJavaWatchableInterface = "java.nio.file.Watchable"
)

// portableJavaPath models only the immutable default-provider Path returned
// by File.toPath. Sleep 2.1 cannot reflect most public Path interface methods
// through the package-private UnixPath/WindowsPath implementation; toFile is
// the one value method the official bridge exposes consistently.
type portableJavaPath struct {
	pathname Value
}

func (path *portableJavaPath) String() string {
	if path == nil {
		return ""
	}
	return path.pathname.String()
}

func portableJavaPathClass() string {
	if goruntime.GOOS == "windows" {
		return "sun.nio.fs.WindowsPath"
	}
	return "sun.nio.fs.UnixPath"
}

func (file *portableJavaFile) portableJavaPath() (*portableJavaPath, error) {
	file.pathMu.Lock()
	defer file.pathMu.Unlock()
	if file.nioPath != nil {
		return file.nioPath, nil
	}
	pathname, err := portableJavaNIOPathValueForGOOS(file.pathValue(), goruntime.GOOS)
	if err != nil {
		return nil, err
	}
	file.nioPath = &portableJavaPath{pathname: pathname}
	return file.nioPath, nil
}

func (path *portableJavaPath) invoke(invocation ObjectInvocation) (Value, bool, error) {
	if path == nil {
		return Null(), false, nil
	}
	if invocation.Op == ObjectTypeCheck {
		class := resolvePortableClassName(invocation.Class)
		switch class {
		case portableJavaPathClass(), portableJavaPathInterface, portableJavaWatchableInterface,
			"java.lang.Object", "java.lang.Comparable", "java.lang.Iterable":
			return Bool(true), true, nil
		default:
			return Bool(false), true, nil
		}
	}
	if invocation.Op == ObjectInvoke && invocation.Message == "toFile" && len(invocation.Arguments) == 0 {
		return ObjectValue(newPortableJavaFile(path.pathname)), true, nil
	}
	return Null(), false, nil
}

func portableJavaNIOPathValueForGOOS(path Value, goos string) (Value, error) {
	path = sleepStringCoercion(path)
	if goos == "windows" {
		return portableJavaNIOWindowsPathValue(path)
	}
	return portableJavaNIOUnixPathValue(path)
}

func portableJavaNIOUnixPathValue(path Value) (Value, error) {
	units := sleepStringUnits(path)
	for index := 0; index < len(units); index++ {
		unit := units[index]
		if unit == 0 {
			return Null(), portableJavaInvalidPath(path, "Nul character not allowed", -1)
		}
		if unit >= 0xd800 && unit <= 0xdbff {
			if index+1 < len(units) && units[index+1] >= 0xdc00 && units[index+1] <= 0xdfff {
				index++
				continue
			}
			return Null(), portableJavaInvalidPath(path, "Malformed input or input contains unmappable characters", -1)
		}
		if unit >= 0xdc00 && unit <= 0xdfff {
			return Null(), portableJavaInvalidPath(path, "Malformed input or input contains unmappable characters", -1)
		}
	}
	if len(units) == 0 {
		return sleepStringValueFromUnits(nil, nil), nil
	}
	normalized := make([]uint16, 0, len(units))
	previousSlash := false
	for _, unit := range units {
		if unit == '/' {
			if previousSlash {
				continue
			}
			previousSlash = true
		} else {
			previousSlash = false
		}
		normalized = append(normalized, unit)
	}
	if len(normalized) > 1 && normalized[len(normalized)-1] == '/' {
		normalized = normalized[:len(normalized)-1]
	}
	return sleepStringValueFromUnits(normalized, nil), nil
}

func portableJavaNIOWindowsPathValue(original Value) (Value, error) {
	input := sleepStringUnits(original)
	expected := byte(0)
	if portableJavaUnitsHavePrefix(input, []uint16{'\\', '\\', '?', '\\'}) {
		if portableJavaUnitsHavePrefix(input, []uint16{'\\', '\\', '?', '\\', 'U', 'N', 'C', '\\'}) {
			expected = 'u'
			input = append([]uint16{'\\', '\\'}, input[8:]...)
		} else {
			expected = 'a'
			input = input[4:]
		}
	}

	typeCode := byte('r')
	root := []uint16(nil)
	offset := 0
	if len(input) > 1 && portableJavaWindowsSlash(input[0]) && portableJavaWindowsSlash(input[1]) {
		typeCode = 'u'
		hostStart := portableJavaWindowsNextNonSlash(input, 2)
		hostEnd, err := portableJavaWindowsNextSlashChecked(original, input, hostStart)
		if err != nil {
			return Null(), err
		}
		if hostStart == hostEnd {
			return Null(), portableJavaInvalidPath(original, "UNC path is missing hostname", -1)
		}
		shareStart := portableJavaWindowsNextNonSlash(input, hostEnd)
		shareEnd, err := portableJavaWindowsNextSlashChecked(original, input, shareStart)
		if err != nil {
			return Null(), err
		}
		if shareStart == shareEnd {
			return Null(), portableJavaInvalidPath(original, "UNC path is missing sharename", -1)
		}
		root = append(root, '\\', '\\')
		root = append(root, input[hostStart:hostEnd]...)
		root = append(root, '\\')
		root = append(root, input[shareStart:shareEnd]...)
		root = append(root, '\\')
		offset = shareEnd
	} else if len(input) > 1 && portableJavaWindowsLetter(input[0]) && input[1] == ':' {
		root = append(root, input[0], ':')
		offset = 2
		typeCode = 'd'
		if len(input) > 2 && portableJavaWindowsSlash(input[2]) {
			root = append(root, '\\')
			offset = 3
			typeCode = 'a'
		}
	} else if len(input) > 0 && portableJavaWindowsSlash(input[0]) {
		root = append(root, '\\')
		typeCode = 'x'
	}
	if expected != 0 && typeCode != expected {
		reason := "Long path prefix can only be used with an absolute path"
		if expected == 'u' {
			reason = "Long UNC path prefix can only be used with a UNC path"
		}
		return Null(), portableJavaInvalidPath(original, reason, -1)
	}

	result := append([]uint16(nil), root...)
	position := portableJavaWindowsNextNonSlash(input, offset)
	for position < len(input) {
		componentStart := position
		last := uint16(0)
		for position < len(input) && !portableJavaWindowsSlash(input[position]) {
			unit := input[position]
			if portableJavaWindowsInvalid(unit) {
				return Null(), portableJavaInvalidPath(original, fmt.Sprintf("Illegal char <%c>", unit), position)
			}
			last = unit
			position++
		}
		if last == ' ' {
			return Null(), portableJavaInvalidPath(original, "Trailing char < >", position-1)
		}
		result = append(result, input[componentStart:position]...)
		position = portableJavaWindowsNextNonSlash(input, position)
		if position < len(input) {
			result = append(result, '\\')
		}
	}
	return sleepStringValueFromUnits(result, nil), nil
}

func portableJavaWindowsNextSlashChecked(original Value, input []uint16, offset int) (int, error) {
	for offset < len(input) && !portableJavaWindowsSlash(input[offset]) {
		if portableJavaWindowsInvalid(input[offset]) {
			return offset, portableJavaInvalidPath(original, fmt.Sprintf("Illegal character [%c] in path", input[offset]), offset)
		}
		offset++
	}
	return offset, nil
}

func portableJavaWindowsNextNonSlash(input []uint16, offset int) int {
	for offset < len(input) && portableJavaWindowsSlash(input[offset]) {
		offset++
	}
	return offset
}

func portableJavaWindowsSlash(unit uint16) bool { return unit == '\\' || unit == '/' }

func portableJavaWindowsLetter(unit uint16) bool {
	return unit >= 'a' && unit <= 'z' || unit >= 'A' && unit <= 'Z'
}

func portableJavaWindowsInvalid(unit uint16) bool {
	return unit < 0x20 || unit == '<' || unit == '>' || unit == ':' || unit == '"' || unit == '|' || unit == '?' || unit == '*'
}

func portableJavaUnitsHavePrefix(value, prefix []uint16) bool {
	if len(value) < len(prefix) {
		return false
	}
	for index, unit := range prefix {
		if value[index] != unit {
			return false
		}
	}
	return true
}

func portableJavaInvalidPath(path Value, reason string, index int) error {
	if index >= 0 {
		return fmt.Errorf("java.nio.file.InvalidPathException: %s at index %d: %s", reason, index, path.String())
	}
	return fmt.Errorf("java.nio.file.InvalidPathException: %s: %s", reason, path.String())
}
