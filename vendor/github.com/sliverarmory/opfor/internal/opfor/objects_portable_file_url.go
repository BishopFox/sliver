package opfor

const portableJavaURLClass = "java.net.URL"

// portableJavaURL is the immutable legacy file URL produced by File.toURL.
// That deprecated API deliberately does not quote illegal URL characters;
// URL's component constructor consequently treats the first '#' as a ref and
// the last '?' before it as the query separator.
type portableJavaURL struct {
	external Value
	path     Value
	file     Value
	query    Value
	querySet bool
	ref      Value
	refSet   bool
}

func (url *portableJavaURL) String() string {
	if url == nil {
		return ""
	}
	return url.external.String()
}

func portableJavaURLFromFile(file *portableJavaFile) (*portableJavaURL, error) {
	if portableJavaFileInvalidValue(file.pathValue()) {
		return nil, portableJavaMalformedURLException("Invalid file path")
	}
	pathname, err := portableJavaFileSlashified(file)
	if err != nil {
		return nil, err
	}
	return newPortableJavaFileURL(pathname), nil
}

func newPortableJavaFileURL(file Value) *portableJavaURL {
	units := sleepStringUnits(file)
	fragmentIndex := portableJavaIndexUnit(units, '#', false)
	fileEnd := len(units)
	var ref Value
	refSet := fragmentIndex >= 0
	if refSet {
		fileEnd = fragmentIndex
		ref = sleepStringValueFromUnits(units[fragmentIndex+1:], nil)
	}
	fileValue := sleepStringValueFromUnits(units[:fileEnd], nil)
	fileUnits := sleepStringUnits(fileValue)
	queryIndex := portableJavaIndexUnit(fileUnits, '?', true)
	path := fileValue
	var query Value
	querySet := queryIndex >= 0
	if querySet {
		path = sleepStringValueFromUnits(fileUnits[:queryIndex], nil)
		query = sleepStringValueFromUnits(fileUnits[queryIndex+1:], nil)
	}
	return &portableJavaURL{
		external: sleepStringConcat(String("file:"), sleepStringValueFromUnits(units, nil)),
		path:     path, file: fileValue, query: query, querySet: querySet, ref: ref, refSet: refSet,
	}
}

func portableJavaIndexUnit(units []uint16, target uint16, reverse bool) int {
	if reverse {
		for index := len(units) - 1; index >= 0; index-- {
			if units[index] == target {
				return index
			}
		}
		return -1
	}
	for index, unit := range units {
		if unit == target {
			return index
		}
	}
	return -1
}

func portableJavaMalformedURLException(message string) error {
	return &portableJavaURLValueError{message: message}
}

type portableJavaURLValueError struct{ message string }

func (err *portableJavaURLValueError) Error() string {
	return "java.net.MalformedURLException: " + err.message
}

func (url *portableJavaURL) invoke(invocation ObjectInvocation) (Value, bool, error) {
	if url == nil {
		return Null(), false, nil
	}
	if invocation.Op == ObjectTypeCheck {
		class := resolvePortableClassName(invocation.Class)
		switch class {
		case portableJavaURLClass, "java.lang.Object", "java.io.Serializable":
			return Bool(true), true, nil
		default:
			return Bool(false), true, nil
		}
	}
	if invocation.Op != ObjectInvoke {
		return Null(), false, nil
	}
	if invocation.Message == "equals" && len(invocation.Arguments) == 1 {
		other, ok := portableJavaURLValue(invocation.Arg(0))
		return portableJavaFileBoolean(ok && url.equal(other, true)), true, nil
	}
	if invocation.Message == "sameFile" && len(invocation.Arguments) == 1 {
		other, ok := portableJavaURLValue(invocation.Arg(0))
		if !ok {
			return Null(), false, nil
		}
		return portableJavaFileBoolean(url.equal(other, false)), true, nil
	}
	if len(invocation.Arguments) != 0 {
		return Null(), false, nil
	}
	switch invocation.Message {
	case "toString", "toExternalForm":
		return url.external, true, nil
	case "getProtocol":
		return String("file"), true, nil
	case "getHost", "getAuthority":
		// File.toURL invokes new URL("file", "", file), so these are defined
		// empty strings rather than undefined components.
		return String(""), true, nil
	case "getUserInfo":
		return Null(), true, nil
	case "getPort", "getDefaultPort":
		return Int(-1), true, nil
	case "getPath":
		return url.path, true, nil
	case "getFile":
		return url.file, true, nil
	case "getQuery":
		if !url.querySet {
			return Null(), true, nil
		}
		return url.query, true, nil
	case "getRef":
		if !url.refSet {
			return Null(), true, nil
		}
		return url.ref, true, nil
	case "hashCode":
		return Int(url.hashCode()), true, nil
	default:
		return Null(), false, nil
	}
}

func portableJavaURLValue(value Value) (*portableJavaURL, bool) {
	object, ok := value.Object()
	if !ok {
		return nil, false
	}
	url, ok := object.(*portableJavaURL)
	return url, ok && url != nil
}

func (url *portableJavaURL) equal(other *portableJavaURL, includeRef bool) bool {
	if url == nil || other == nil || portableJavaStringCompareValues(url.file, other.file) != 0 {
		return false
	}
	if !includeRef {
		return true
	}
	if url.refSet != other.refSet {
		return false
	}
	return !url.refSet || portableJavaStringCompareValues(url.ref, other.ref) == 0
}

func (url *portableJavaURL) hashCode() int32 {
	// URLStreamHandler's file handler inherits the default value hash. The
	// generated empty host cannot trigger DNS resolution.
	hash := portableJavaStringHashValue(String("file"))
	hash += portableJavaStringHashValue(url.file)
	hash-- // file handler's default port is -1
	if url.refSet {
		hash += portableJavaStringHashValue(url.ref)
	}
	return hash
}
