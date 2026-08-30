package opfor

import (
	"crypto/md5"
	cryptorand "crypto/rand"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
)

// portableJavaUUID is the immutable java.util.UUID value exposed by Sleep's
// object bridge. The two words retain Java's signed-long bit representation;
// computations use uint64 so shifts and masks match Java's unsigned operators.
type portableJavaUUID struct {
	most  uint64
	least uint64
}

func newPortableJavaUUID(most, least int64) *portableJavaUUID {
	return &portableJavaUUID{most: uint64(most), least: uint64(least)}
}

func portableJavaUUIDFromBytes(data []byte) *portableJavaUUID {
	return &portableJavaUUID{
		most:  binary.BigEndian.Uint64(data[:8]),
		least: binary.BigEndian.Uint64(data[8:16]),
	}
}

func (uuid *portableJavaUUID) String() string {
	if uuid == nil {
		return "null"
	}
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uint32(uuid.most>>32), uint16(uuid.most>>16), uint16(uuid.most),
		uint16(uuid.least>>48), uuid.least&0x0000ffffffffffff)
}

func portableJavaUUIDClass(invocation ObjectInvocation) (Value, bool, error) {
	class := resolvePortableClassName(invocation.Class)
	if class != "java.util.UUID" {
		return Null(), false, nil
	}
	if invocation.Op == ObjectConstruct {
		if len(invocation.Arguments) != 2 {
			message := fmt.Sprintf("no constructor matching java.util.UUID(%s)", portableObjectArgumentList(invocation))
			return portableObjectWarning(invocation, message), true, nil
		}
		return ObjectValue(newPortableJavaUUID(sleepInt64(invocation.Arg(0)), sleepInt64(invocation.Arg(1)))), true, nil
	}
	if invocation.Op != ObjectInvoke {
		return Null(), false, nil
	}

	switch invocation.Message {
	case "randomUUID":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, class), true, nil
		}
		data := make([]byte, 16)
		if _, err := cryptorand.Read(data); err != nil {
			return Null(), true, fmt.Errorf("java.lang.InternalError: unable to generate random UUID: %w", err)
		}
		data[6] = data[6]&0x0f | 0x40
		data[8] = data[8]&0x3f | 0x80
		return ObjectValue(portableJavaUUIDFromBytes(data)), true, nil
	case "nameUUIDFromBytes":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, class), true, nil
		}
		if invocation.Arg(0).IsNull() {
			return Null(), true, portableJavaUUIDException(
				"java.lang.NullPointerException", "Cannot read the array length because name is null",
				"public static java.util.UUID java.util.UUID.nameUUIDFromBytes(byte[])")
		}
		name, ok := portableJavaByteArray(invocation.Arg(0))
		if !ok {
			return portableNoMatchingMethod(invocation, class), true, nil
		}
		digest := md5.Sum(name)
		data := digest[:]
		data[6] = data[6]&0x0f | 0x30
		data[8] = data[8]&0x3f | 0x80
		return ObjectValue(portableJavaUUIDFromBytes(data)), true, nil
	case "fromString":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, class), true, nil
		}
		if invocation.Arg(0).IsNull() {
			return Null(), true, portableJavaUUIDException(
				"java.lang.NullPointerException", "Cannot invoke \"String.length()\" because name is null",
				"public static java.util.UUID java.util.UUID.fromString(java.lang.String)")
		}
		if invocation.Arg(0).Kind() != KindString {
			return portableNoMatchingMethod(invocation, class), true, nil
		}
		uuid, err := parsePortableJavaUUID(invocation.Arg(0).String())
		if err != nil {
			return Null(), true, err
		}
		return ObjectValue(uuid), true, nil
	}
	return Null(), false, nil
}

func parsePortableJavaUUID(name string) (*portableJavaUUID, error) {
	const frame = "public static java.util.UUID java.util.UUID.fromString(java.lang.String)"
	if len(name) > 36 {
		return nil, portableJavaUUIDException("java.lang.IllegalArgumentException", "UUID string too large", frame)
	}
	parts := strings.Split(name, "-")
	if len(parts) != 5 {
		return nil, portableJavaUUIDException("java.lang.IllegalArgumentException", "Invalid UUID string: "+name, frame)
	}
	values := [5]uint64{}
	for index, part := range parts {
		if part == "" {
			return nil, portableJavaUUIDException(
				"java.lang.NumberFormatException", `For input string: "" under radix 16`, frame)
		}
		value, err := strconv.ParseInt(part, 16, 64)
		if err != nil {
			errorIndex := portableJavaUUIDHexErrorIndex(part)
			message := fmt.Sprintf("Error at index %d in: %q", errorIndex, part)
			return nil, portableJavaUUIDException("java.lang.NumberFormatException", message, frame)
		}
		values[index] = uint64(value)
	}
	most := values[0] & 0xffffffff
	most = most<<16 | (values[1] & 0xffff)
	most = most<<16 | (values[2] & 0xffff)
	least := values[3] & 0xffff
	least = least<<48 | (values[4] & 0x0000ffffffffffff)
	return &portableJavaUUID{most: most, least: least}, nil
}

func portableJavaUUIDHexErrorIndex(value string) int {
	start := 0
	if value != "" && (value[0] == '+' || value[0] == '-') {
		start = 1
	}
	if start == len(value) {
		return len(value)
	}
	for index := start; index < len(value); index++ {
		character := value[index]
		if character >= '0' && character <= '9' || character >= 'a' && character <= 'f' || character >= 'A' && character <= 'F' {
			continue
		}
		return index
	}
	if len(value) == 0 {
		return 0
	}
	return len(value) - 1
}

func portableJavaUUIDException(class, message, frame string) *portableJavaException {
	text := class
	if message != "" {
		text += ": " + message
	}
	return &portableJavaException{class: class, message: message, text: text, frame: frame}
}

func (uuid *portableJavaUUID) invoke(invocation ObjectInvocation) (Value, bool, error) {
	if invocation.Op == ObjectTypeCheck {
		class := resolvePortableClassName(invocation.Class)
		return Bool(class == "java.util.UUID" || class == "java.lang.Object" ||
			class == "java.lang.Comparable" || class == "java.io.Serializable"), true, nil
	}
	if invocation.Op != ObjectInvoke {
		return Null(), false, nil
	}
	if uuid == nil {
		return Null(), true, portableJavaUUIDException(
			"java.lang.NullPointerException", "java.util.UUID state is null", "")
	}

	switch invocation.Message {
	case "toString":
		if len(invocation.Arguments) == 0 {
			return String(uuid.String()), true, nil
		}
	case "getMostSignificantBits":
		if len(invocation.Arguments) == 0 {
			return Long(int64(uuid.most)), true, nil
		}
	case "getLeastSignificantBits":
		if len(invocation.Arguments) == 0 {
			return Long(int64(uuid.least)), true, nil
		}
	case "version":
		if len(invocation.Arguments) == 0 {
			return Int(int32(uuid.version())), true, nil
		}
	case "variant":
		if len(invocation.Arguments) == 0 {
			return Int(int32(uuid.variant())), true, nil
		}
	case "timestamp":
		if len(invocation.Arguments) == 0 {
			if err := uuid.requireTimeBased("timestamp"); err != nil {
				return Null(), true, err
			}
			value := (uuid.most&0x0fff)<<48 | ((uuid.most>>16)&0xffff)<<32 | uuid.most>>32
			return Long(int64(value)), true, nil
		}
	case "clockSequence":
		if len(invocation.Arguments) == 0 {
			if err := uuid.requireTimeBased("clockSequence"); err != nil {
				return Null(), true, err
			}
			return Int(int32((uuid.least & 0x3fff000000000000) >> 48)), true, nil
		}
	case "node":
		if len(invocation.Arguments) == 0 {
			if err := uuid.requireTimeBased("node"); err != nil {
				return Null(), true, err
			}
			return Long(int64(uuid.least & 0x0000ffffffffffff)), true, nil
		}
	case "hashCode":
		if len(invocation.Arguments) == 0 {
			highLow := uuid.most ^ uuid.least
			return Int(int32(highLow ^ highLow>>32)), true, nil
		}
	case "equals":
		if len(invocation.Arguments) == 1 {
			other, ok := portableJavaUUIDValue(invocation.Arg(0))
			return Bool(ok && uuid.most == other.most && uuid.least == other.least), true, nil
		}
	case "compareTo":
		if len(invocation.Arguments) == 1 {
			other, ok := portableJavaUUIDValue(invocation.Arg(0))
			if !ok {
				if invocation.Arg(0).IsNull() {
					return Null(), true, portableJavaUUIDException(
						"java.lang.NullPointerException", "Cannot read field \"mostSigBits\" because val is null",
						"public int java.util.UUID.compareTo(java.util.UUID)")
				}
				class := "null"
				if actual, known := portableObjectClass(invocation.Arg(0)); known {
					class = actual
				}
				return Null(), true, portableJavaUUIDException(
					"java.lang.ClassCastException", class+" cannot be cast to java.util.UUID",
					"public int java.util.UUID.compareTo(java.util.UUID)")
			}
			return Int(comparePortableJavaUUID(uuid, other)), true, nil
		}
	}
	return portableNoMatchingMethod(invocation, "java.util.UUID"), true, nil
}

func portableJavaUUIDValue(value Value) (*portableJavaUUID, bool) {
	object, ok := value.Object()
	if !ok {
		return nil, false
	}
	uuid, ok := object.(*portableJavaUUID)
	return uuid, ok && uuid != nil
}

func (uuid *portableJavaUUID) version() int {
	return int((uuid.most >> 12) & 0x0f)
}

func (uuid *portableJavaUUID) variant() int {
	switch uuid.least >> 61 {
	case 4, 5:
		return 2
	case 6:
		return 6
	case 7:
		return 7
	default:
		return 0
	}
}

func (uuid *portableJavaUUID) requireTimeBased(method string) error {
	if uuid.version() == 1 {
		return nil
	}
	result := "long"
	if method == "clockSequence" {
		result = "int"
	}
	return portableJavaUUIDException(
		"java.lang.UnsupportedOperationException", "Not a time-based UUID",
		fmt.Sprintf("public %s java.util.UUID.%s()", result, method))
}

func comparePortableJavaUUID(left, right *portableJavaUUID) int32 {
	if int64(left.most) < int64(right.most) {
		return -1
	}
	if int64(left.most) > int64(right.most) {
		return 1
	}
	if int64(left.least) < int64(right.least) {
		return -1
	}
	if int64(left.least) > int64(right.least) {
		return 1
	}
	return 0
}
