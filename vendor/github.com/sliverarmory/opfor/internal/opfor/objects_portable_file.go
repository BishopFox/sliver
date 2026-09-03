package opfor

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
)

const portableJavaFileClass = "java.io.File"

type portableJavaFilePermission uint8

const (
	portableJavaFilePermissionRead portableJavaFilePermission = iota
	portableJavaFilePermissionWrite
	portableJavaFilePermissionExecute
)

type portableJavaFileSpaceKind uint8

const (
	portableJavaFileSpaceTotal portableJavaFileSpaceKind = iota
	portableJavaFileSpaceFree
	portableJavaFileSpaceUsable
)

// portableJavaFile is the bounded subset of java.io.File that can be
// represented without a JVM. The object itself remains immutable and owns
// only an abstract pathname; individual methods may query or mutate the
// filesystem without retaining an open descriptor.
type portableJavaFile struct {
	// pathname is authoritative. Java File identity is the exact sequence of
	// UTF-16 code units in its normalized abstract pathname; Value additionally
	// lets OPFOR retain byte provenance that a Go string alone cannot represent.
	pathname Value

	// path is the deliberate host spelling retained for the existing internal
	// ScriptLoader bridge. File methods never use it for identity.
	path string

	// OpenJDK File.toPath caches the successfully constructed default-provider
	// Path and returns that same object on later calls. Invalid paths are not
	// cached, so each attempt still follows the provider's validation path.
	pathMu  sync.Mutex
	nioPath *portableJavaPath
}

func (file *portableJavaFile) String() string {
	if file == nil {
		return ""
	}
	return file.pathValue().String()
}

func newPortableJavaFile(pathname Value) *portableJavaFile {
	pathname = sleepStringCoercion(pathname)
	return &portableJavaFile{pathname: pathname, path: portableJavaFileHostPath(pathname)}
}

func (file *portableJavaFile) pathValue() Value {
	if file != nil && file.pathname.Kind() == KindString {
		return file.pathname
	}
	if file == nil {
		return String("")
	}
	// Keep values created before pathname became authoritative usable by the
	// package's internal tests and serialized-runtime compatibility shims.
	return String(file.path)
}

func portableJavaFileConstruct(invocation ObjectInvocation) (Value, bool, error) {
	if resolvePortableClassName(invocation.Class) != portableJavaFileClass || invocation.Op != ObjectConstruct {
		return Null(), false, nil
	}

	var pathname Value
	switch len(invocation.Arguments) {
	case 1:
		var ok bool
		pathname, ok = portableJavaFileStringArgument(invocation.Arg(0))
		if !ok {
			return portableJavaFileConstructorWarning(invocation), true, nil
		}
		pathname = portableJavaFileNormalizeValue(pathname)
	case 2:
		child, ok := portableJavaFileStringArgument(invocation.Arg(1))
		if !ok {
			return portableJavaFileConstructorWarning(invocation), true, nil
		}
		child = portableJavaFileNormalizeValue(child)
		first := invocation.Arg(0)
		if first.IsNull() {
			// Sleep's reflection matcher selects File(File, String) for a null
			// parent, and Java treats that as new File(child).
			pathname = child
		} else if object, ok := first.Object(); ok {
			parent, ok := object.(*portableJavaFile)
			if !ok || parent == nil {
				return portableJavaFileConstructorWarning(invocation), true, nil
			}
			pathname = portableJavaFileResolveValue(parent.pathValue(), child)
		} else {
			parent, ok := portableJavaFileStringArgument(first)
			if !ok {
				return portableJavaFileConstructorWarning(invocation), true, nil
			}
			parent = portableJavaFileNormalizeValue(parent)
			// File(String, String) resolves an empty parent against the
			// platform default parent. File(File, String) deliberately does not.
			if len(sleepStringUnits(parent)) == 0 {
				parent = portableJavaFileDefaultParentValue(goruntime.GOOS)
			}
			pathname = portableJavaFileResolveValue(parent, child)
		}
	default:
		return portableJavaFileConstructorWarning(invocation), true, nil
	}
	return ObjectValue(newPortableJavaFile(pathname)), true, nil
}

func portableJavaFileStatic(ctx context.Context, invocation ObjectInvocation) (Value, bool, error) {
	if resolvePortableClassName(invocation.Class) != portableJavaFileClass || !invocation.Target.IsNull() {
		return Null(), false, nil
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), true, err
	}

	if (invocation.Op == ObjectGet || invocation.Op == ObjectInvoke) && len(invocation.Arguments) == 0 {
		switch invocation.Message {
		case "separator":
			return String(string(os.PathSeparator)), true, nil
		case "separatorChar":
			return sleepUTF16CharacterValue(uint16(os.PathSeparator)), true, nil
		case "pathSeparator":
			return String(string(os.PathListSeparator)), true, nil
		case "pathSeparatorChar":
			return sleepUTF16CharacterValue(uint16(os.PathListSeparator)), true, nil
		}
	}
	if invocation.Op != ObjectInvoke {
		return Null(), false, nil
	}

	switch invocation.Message {
	case "listRoots":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, portableJavaFileClass), true, nil
		}
		roots := portableJavaFileRootValues()
		array, err := newRuntimeArray(invocation.Runtime, roots...)
		if err != nil {
			return Null(), true, err
		}
		return ArrayValue(array), true, nil
	case "createTempFile":
		return portableJavaFileCreateTempFile(ctx, invocation)
	default:
		return Null(), false, nil
	}
}

func portableJavaFileRootValues() []Value {
	paths := portableJavaFileRootPaths()
	roots := make([]Value, len(paths))
	for index, path := range paths {
		roots[index] = ObjectValue(newPortableJavaFile(String(path)))
	}
	return roots
}

func portableJavaFileCreateTempFile(ctx context.Context, invocation ObjectInvocation) (Value, bool, error) {
	if len(invocation.Arguments) != 2 && len(invocation.Arguments) != 3 {
		return portableNoMatchingMethod(invocation, portableJavaFileClass), true, nil
	}

	prefixArgument := invocation.Arg(0)
	if prefixArgument.IsNull() {
		return Null(), true, errors.New(`java.lang.NullPointerException: Cannot invoke "String.length()" because "prefix" is null`)
	}
	prefix, ok := portableJavaFileStringArgument(prefixArgument)
	if !ok {
		return portableNoMatchingMethod(invocation, portableJavaFileClass), true, nil
	}
	if len(sleepStringUnits(prefix)) < 3 {
		return Null(), true, fmt.Errorf(
			`java.lang.IllegalArgumentException: Prefix string %q too short: length must be at least 3`,
			prefix.String(),
		)
	}

	suffix := invocation.Arg(1)
	if suffix.IsNull() {
		suffix = String(".tmp")
	} else {
		var suffixOK bool
		suffix, suffixOK = portableJavaFileStringArgument(suffix)
		if !suffixOK {
			return portableNoMatchingMethod(invocation, portableJavaFileClass), true, nil
		}
	}

	directory := newPortableJavaFile(portableJavaFileNormalizeValue(String(os.TempDir())))
	if len(invocation.Arguments) == 3 && !invocation.Arg(2).IsNull() {
		var directoryOK bool
		directory, directoryOK = portableJavaFileValue(invocation.Arg(2))
		if !directoryOK {
			return portableNoMatchingMethod(invocation, portableJavaFileClass), true, nil
		}
	}

	// OpenJDK validates the original prefix length and then discards any
	// directory component by taking new File(prefix).getName(). The suffix is
	// not stripped: a separator there makes the generated child fail the
	// getName identity check below unless normalization removes it.
	prefix = portableJavaFileNameValue(portableJavaFileNormalizeValue(prefix))
	for attempts := 0; attempts < 10_000; attempts++ {
		if err := executionContextError(ctx); err != nil {
			return Null(), true, err
		}
		randomPart, err := portableJavaFileTempRandomDecimal()
		if err != nil {
			return Null(), true, fmt.Errorf("java.io.IOException: %s", err)
		}
		name, err := portableJavaFileTempName(
			prefix,
			suffix,
			randomPart,
			portableJavaFileNameMax(portableJavaFileHostPath(directory.pathValue())),
		)
		if err != nil {
			return Null(), true, err
		}
		pathValue := portableJavaFileResolveValue(directory.pathValue(), name)
		if portableJavaFileInvalidValue(pathValue) || !sleepStringValuesEqual(name, portableJavaFileNameValue(pathValue)) {
			return Null(), true, fmt.Errorf("java.io.IOException: Unable to create temporary file, %s", name.String())
		}

		hostPath := portableJavaFileHostPath(pathValue)
		created, err := os.OpenFile(hostPath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o666)
		if err != nil {
			if errors.Is(err, os.ErrExist) {
				continue
			}
			return Null(), true, portableJavaFileCreateTempIOException(err)
		}
		if err := created.Close(); err != nil {
			_ = os.Remove(hostPath)
			return Null(), true, portableJavaFileCreateTempIOException(err)
		}
		return ObjectValue(newPortableJavaFile(pathValue)), true, nil
	}
	return Null(), true, errors.New("java.io.IOException: Unable to create temporary file")
}

func portableJavaFileTempName(prefix, suffix Value, randomPart string, nameMax int) (Value, error) {
	prefixLength := sleepStringLength(prefix)
	randomLength := len(randomPart)
	suffixLength := sleepStringLength(suffix)
	if nameMax <= 0 {
		nameMax = 255
	}

	excess := int64(prefixLength) + int64(randomLength) + int64(suffixLength) - int64(nameMax)
	if excess > 0 {
		// This intentionally follows the pinned OpenJDK TempDirectory algorithm,
		// including its two suffix-shortening passes with the same first excess.
		prefixLength = portableJavaFileShortenTempPart(prefixLength, excess, 3)
		excess = int64(prefixLength) + int64(randomLength) + int64(suffixLength) - int64(nameMax)
		if excess > 0 {
			minimumSuffix := 0
			if portableJavaFileValueHasPrefixUnit(suffix, '.') {
				minimumSuffix = 4
			}
			suffixLength = portableJavaFileShortenTempPart(suffixLength, excess, minimumSuffix)
			suffixLength = portableJavaFileShortenTempPart(suffixLength, excess, 3)
			excess = int64(prefixLength) + int64(randomLength) + int64(suffixLength) - int64(nameMax)
		}
		if excess > 0 && excess <= int64(randomLength-5) {
			randomLength = portableJavaFileShortenTempPart(randomLength, excess, 5)
		}
	}

	if int64(prefixLength)+int64(randomLength)+int64(suffixLength) > math.MaxInt32 {
		return Null(), errors.New("java.lang.OutOfMemoryError: Required length exceeds implementation limit")
	}
	name := sleepStringConcat(
		sleepStringValueSlice(prefix, 0, prefixLength),
		String(randomPart[:randomLength]),
		sleepStringValueSlice(suffix, 0, suffixLength),
	)
	// OpenJDK normalizes the completed component before constructing and
	// validating the child File. This notably permits a suffix consisting only
	// of trailing separators, which normalization removes.
	return portableJavaFileNormalizeValue(name), nil
}

func portableJavaFileShortenTempPart(length int, excess int64, minimum int) int {
	shortened := int64(length) - excess
	if shortened < int64(minimum) {
		shortened = int64(minimum)
	}
	if shortened >= 0 && shortened < int64(length) {
		return int(shortened)
	}
	return length
}

func portableJavaFileTempRandomDecimal() (string, error) {
	var source [8]byte
	if _, err := io.ReadFull(rand.Reader, source[:]); err != nil {
		return "", err
	}
	return strconv.FormatUint(binary.BigEndian.Uint64(source[:]), 10), nil
}

func portableJavaFileCreateTempIOException(err error) error {
	var pathError *os.PathError
	if errors.As(err, &pathError) && pathError != nil {
		err = pathError.Err
	}
	message := err.Error()
	if message != "" {
		characters := []rune(message)
		characters[0] = unicode.ToUpper(characters[0])
		message = string(characters)
	}
	return fmt.Errorf("java.io.IOException: %s", message)
}

// Sleep's ObjectUtilities coerces scalar numeric values when matching a Java
// String parameter. Compound values and the empty scalar do not match String.
func portableJavaFileStringArgument(value Value) (Value, bool) {
	switch value.Kind() {
	case KindString:
		return value, true
	case KindInt, KindLong, KindDouble:
		return String(value.String()), true
	default:
		return Null(), false
	}
}

func portableJavaFileConstructorWarning(invocation ObjectInvocation) Value {
	message := fmt.Sprintf("no constructor matching %s(%s)", portableJavaFileClass, portableObjectArgumentList(invocation))
	return portableObjectWarning(invocation, message)
}

func (file *portableJavaFile) invoke(invocation ObjectInvocation) (Value, bool, error) {
	return file.invokeContext(context.Background(), invocation)
}

func (file *portableJavaFile) invokeContext(ctx context.Context, invocation ObjectInvocation) (Value, bool, error) {
	if file == nil {
		return Null(), false, nil
	}
	if invocation.Op == ObjectTypeCheck {
		class := resolvePortableClassName(invocation.Class)
		switch class {
		case portableJavaFileClass, "java.lang.Object", "java.io.Serializable", "java.lang.Comparable":
			return Bool(true), true, nil
		default:
			return Bool(false), true, nil
		}
	}
	if invocation.Op != ObjectInvoke {
		return Null(), false, nil
	}
	switch invocation.Message {
	case "equals":
		if len(invocation.Arguments) != 1 {
			return Null(), false, nil
		}
		other, ok := portableJavaFileValue(invocation.Arg(0))
		return portableJavaFileBoolean(ok && portableJavaFileCompareValues(file.pathValue(), other.pathValue()) == 0), true, nil
	case "compareTo":
		if len(invocation.Arguments) != 1 {
			return Null(), false, nil
		}
		argument := invocation.Arg(0)
		if argument.IsNull() {
			// Current OpenJDK dereferences the second File inside the platform
			// FileSystem comparator. Sleep exposes that reflected exception via
			// checkError and returns the empty scalar from the object expression.
			return Null(), true, errors.New(`java.lang.NullPointerException: Cannot invoke "java.io.File.getPath()" because "f2" is null`)
		}
		other, ok := portableJavaFileValue(argument)
		if !ok {
			// java.io.File also has a synthetic compareTo(Object) bridge. Sleep
			// 2.1's reflection matcher selects it for otherwise incompatible
			// scalar values, and the bridge's cast fails before comparison.
			return Null(), true, portableJavaFileClassCast(argument)
		}
		return Int(portableJavaFileCompareValues(file.pathValue(), other.pathValue())), true, nil
	case "renameTo":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, portableJavaFileClass), true, nil
		}
		argument := invocation.Arg(0)
		if argument.IsNull() {
			return Null(), true, errors.New("java.lang.NullPointerException")
		}
		destination, ok := portableJavaFileValue(argument)
		if !ok {
			return portableNoMatchingMethod(invocation, portableJavaFileClass), true, nil
		}
		if portableJavaFileInvalidValue(file.pathValue()) || portableJavaFileInvalidValue(destination.pathValue()) {
			return portableJavaFileBoolean(false), true, nil
		}
		err := os.Rename(
			portableJavaFileFilesystemPathValue(file.pathValue()),
			portableJavaFileFilesystemPathValue(destination.pathValue()),
		)
		return portableJavaFileBoolean(err == nil), true, nil
	case "setLastModified":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, portableJavaFileClass), true, nil
		}
		modified, ok := portableJavaFileLongArgument(invocation.Arg(0))
		if !ok {
			return portableNoMatchingMethod(invocation, portableJavaFileClass), true, nil
		}
		if modified < 0 {
			return Null(), true, errors.New("java.lang.IllegalArgumentException: Negative time")
		}
		if portableJavaFileInvalidValue(file.pathValue()) {
			return portableJavaFileBoolean(false), true, nil
		}
		// A zero access time asks Go to leave that timestamp unchanged. This
		// mirrors UnixFileSystem's explicit stat/utimes sequence and Windows'
		// SetFileTime call, without introducing a user-space atime race.
		err := os.Chtimes(
			portableJavaFileFilesystemPathValue(file.pathValue()),
			time.Time{},
			time.UnixMilli(modified),
		)
		return portableJavaFileBoolean(err == nil), true, nil
	case "setWritable", "setReadable", "setExecutable":
		if len(invocation.Arguments) != 1 && len(invocation.Arguments) != 2 {
			return portableNoMatchingMethod(invocation, portableJavaFileClass), true, nil
		}
		enabled, ok := portableJavaFileBooleanArgument(invocation.Arg(0))
		if !ok {
			return portableNoMatchingMethod(invocation, portableJavaFileClass), true, nil
		}
		ownerOnly := true
		if len(invocation.Arguments) == 2 {
			ownerOnly, ok = portableJavaFileBooleanArgument(invocation.Arg(1))
			if !ok {
				return portableNoMatchingMethod(invocation, portableJavaFileClass), true, nil
			}
		}
		if portableJavaFileInvalidValue(file.pathValue()) {
			return portableJavaFileBoolean(false), true, nil
		}
		permission := portableJavaFilePermissionWrite
		switch invocation.Message {
		case "setReadable":
			permission = portableJavaFilePermissionRead
		case "setExecutable":
			permission = portableJavaFilePermissionExecute
		}
		return portableJavaFileBoolean(portableJavaFileSetPermission(
			portableJavaFileFilesystemPathValue(file.pathValue()), permission, enabled, ownerOnly,
		)), true, nil
	case "list", "listFiles":
		if len(invocation.Arguments) > 1 {
			return portableNoMatchingMethod(invocation, portableJavaFileClass), true, nil
		}
		files := invocation.Message == "listFiles"
		if len(invocation.Arguments) == 0 || invocation.Arg(0).IsNull() {
			return file.portableListContextForRuntime(ctx, invocation.Runtime, files, portableJavaFileFilterNone, nil)
		}
		filter, kind, ok := portableJavaFileFilterArgument(invocation, files)
		if !ok {
			// An opaque non-portable object may be a real JVM FilenameFilter or
			// FileFilter. The importer already had first refusal and remains the
			// only authority capable of invoking that object truthfully.
			if object, isObject := invocation.Arg(0).Object(); isObject {
				if _, isPortableProxy := object.(*portableJavaProxy); !isPortableProxy {
					return Null(), false, nil
				}
			}
			return portableNoMatchingMethod(invocation, portableJavaFileClass), true, nil
		}
		return file.portableListContextForRuntime(ctx, invocation.Runtime, files, kind, filter)
	}
	if len(invocation.Arguments) != 0 {
		return Null(), false, nil
	}
	pathname := file.pathValue()
	filesystemPath := portableJavaFileFilesystemPathValue(pathname)
	invalidPath := portableJavaFileInvalidValue(pathname)

	switch invocation.Message {
	case "exists":
		if invalidPath {
			return portableJavaFileBoolean(false), true, nil
		}
		_, err := os.Stat(filesystemPath)
		return portableJavaFileBoolean(err == nil), true, nil
	case "isFile":
		if invalidPath {
			return portableJavaFileBoolean(false), true, nil
		}
		info, err := os.Stat(filesystemPath)
		return portableJavaFileBoolean(err == nil && info.Mode().IsRegular()), true, nil
	case "isDirectory":
		if invalidPath {
			return portableJavaFileBoolean(false), true, nil
		}
		info, err := os.Stat(filesystemPath)
		return portableJavaFileBoolean(err == nil && info.IsDir()), true, nil
	case "canRead":
		if invalidPath {
			return portableJavaFileBoolean(false), true, nil
		}
		return portableJavaFileBoolean(portableJavaFileCanRead(filesystemPath)), true, nil
	case "canWrite":
		if invalidPath {
			return portableJavaFileBoolean(false), true, nil
		}
		return portableJavaFileBoolean(portableJavaFileCanWrite(filesystemPath)), true, nil
	case "canExecute":
		if invalidPath {
			return portableJavaFileBoolean(false), true, nil
		}
		return portableJavaFileBoolean(portableJavaFileCanExecute(filesystemPath)), true, nil
	case "isHidden":
		if invalidPath {
			return portableJavaFileBoolean(false), true, nil
		}
		name := portableJavaFileNameValue(pathname)
		if goruntime.GOOS != "windows" {
			// UnixFileSystem.isHidden is a pathname check and does not require the
			// target to exist.
			return portableJavaFileBoolean(portableJavaFileValueHasPrefixUnit(name, '.')), true, nil
		}
		// The Go standard library does not expose Windows hidden attributes.
		// Retain the portable dot-name convention for this bounded adapter.
		_, err := os.Stat(filesystemPath)
		return portableJavaFileBoolean(err == nil && portableJavaFileValueHasPrefixUnit(name, '.')), true, nil
	case "length":
		if invalidPath {
			return Long(0), true, nil
		}
		info, err := os.Stat(filesystemPath)
		if err != nil {
			return Long(0), true, nil
		}
		return Long(info.Size()), true, nil
	case "lastModified":
		if invalidPath {
			return Long(0), true, nil
		}
		info, err := os.Stat(filesystemPath)
		if err != nil {
			return Long(0), true, nil
		}
		return Long(info.ModTime().UnixMilli()), true, nil
	case "getTotalSpace":
		if invalidPath {
			return Long(0), true, nil
		}
		return Long(portableJavaFileSpace(filesystemPath, portableJavaFileSpaceTotal)), true, nil
	case "getFreeSpace":
		if invalidPath {
			return Long(0), true, nil
		}
		return Long(portableJavaFileSpace(filesystemPath, portableJavaFileSpaceFree)), true, nil
	case "getUsableSpace":
		if invalidPath {
			return Long(0), true, nil
		}
		return Long(portableJavaFileSpace(filesystemPath, portableJavaFileSpaceUsable)), true, nil
	case "createNewFile":
		if portableJavaFileInvalidValue(pathname) {
			return Null(), true, errors.New("java.io.IOException: Invalid file path")
		}
		// Unlike most File operations, OpenJDK passes the raw abstract
		// pathname to createFileExclusively. In particular File("") raises an
		// IOException instead of resolving to the process working directory.
		created, err := os.OpenFile(portableJavaFileHostPath(pathname), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o666)
		if err != nil {
			if errors.Is(err, os.ErrExist) {
				return portableJavaFileBoolean(false), true, nil
			}
			return Null(), true, fmt.Errorf("java.io.IOException: Could not open file: %w", err)
		}
		if err := created.Close(); err != nil {
			return Null(), true, fmt.Errorf("java.io.IOException: Could not close file: %w", err)
		}
		return portableJavaFileBoolean(true), true, nil
	case "delete":
		if portableJavaFileInvalidValue(pathname) {
			return portableJavaFileBoolean(false), true, nil
		}
		return portableJavaFileBoolean(os.Remove(filesystemPath) == nil), true, nil
	case "mkdir":
		if portableJavaFileInvalidValue(pathname) {
			return portableJavaFileBoolean(false), true, nil
		}
		return portableJavaFileBoolean(os.Mkdir(filesystemPath, 0o777) == nil), true, nil
	case "mkdirs":
		if portableJavaFileInvalidValue(pathname) {
			return portableJavaFileBoolean(false), true, nil
		}
		if _, err := os.Stat(filesystemPath); err == nil {
			return portableJavaFileBoolean(false), true, nil
		}
		return portableJavaFileBoolean(os.MkdirAll(filesystemPath, 0o777) == nil), true, nil
	case "getName":
		return portableJavaFileNameValue(pathname), true, nil
	case "getParent":
		parent, ok := portableJavaFileParentValue(pathname)
		if !ok {
			return Null(), true, nil
		}
		return parent, true, nil
	case "getPath", "toString":
		return pathname, true, nil
	case "getAbsolutePath":
		absolute, err := portableJavaFileAbsoluteValue(pathname)
		if err != nil {
			return Null(), true, err
		}
		return absolute, true, nil
	case "getCanonicalPath":
		canonical, err := portableJavaFileCanonicalValue(pathname)
		if err != nil {
			return Null(), true, err
		}
		return canonical, true, nil
	case "isAbsolute":
		return portableJavaFileBoolean(portableJavaFileIsAbsoluteValue(pathname, goruntime.GOOS)), true, nil
	case "getAbsoluteFile":
		absolute, err := portableJavaFileAbsoluteValue(pathname)
		if err != nil {
			return Null(), true, err
		}
		return ObjectValue(newPortableJavaFile(absolute)), true, nil
	case "getCanonicalFile":
		canonical, err := portableJavaFileCanonicalValue(pathname)
		if err != nil {
			return Null(), true, err
		}
		return ObjectValue(newPortableJavaFile(canonical)), true, nil
	case "toURI":
		uri, err := portableJavaURIFromFile(file)
		if err != nil {
			return Null(), true, err
		}
		return ObjectValue(uri), true, nil
	case "toURL":
		url, err := portableJavaURLFromFile(file)
		if err != nil {
			return Null(), true, err
		}
		return ObjectValue(url), true, nil
	case "toPath":
		path, err := file.portableJavaPath()
		if err != nil {
			return Null(), true, err
		}
		return ObjectValue(path), true, nil
	case "getParentFile":
		parent, ok := portableJavaFileParentValue(pathname)
		if !ok {
			return Null(), true, nil
		}
		return ObjectValue(newPortableJavaFile(parent)), true, nil
	case "setReadOnly":
		if invalidPath {
			return portableJavaFileBoolean(false), true, nil
		}
		return portableJavaFileBoolean(portableJavaFileSetReadOnly(filesystemPath)), true, nil
	case "hashCode":
		return Int(portableJavaFileHashValue(pathname)), true, nil
	default:
		return Null(), false, nil
	}
}

// Sleep 2.1's ObjectUtilities only matches primitive long parameters from
// numeric scalars, the empty scalar, or an exactly boxed java.lang.Long.
// Strings (including numeric strings), arrays, hashes, and other boxes do not
// reach File.setLastModified.
func portableJavaFileLongArgument(value Value) (int64, bool) {
	switch value.Kind() {
	case KindNull:
		return 0, true
	case KindInt, KindLong, KindDouble:
		return sleepInt64(value), true
	case KindObject:
		primitive, ok := value.data.(*portableJavaPrimitive)
		if ok && primitive != nil && primitive.className() == "java.lang.Long" {
			return sleepInt64(primitive.sleepValue()), true
		}
	}
	return 0, false
}

// Primitive booleans follow ObjectUtilities.buildArgument: numeric scalars
// become value.intValue()!=0, the empty scalar becomes false, and only an
// exactly boxed Boolean object is accepted from Java-object scalars.
func portableJavaFileBooleanArgument(value Value) (bool, bool) {
	switch value.Kind() {
	case KindNull:
		return false, true
	case KindInt, KindLong, KindDouble:
		return sleepInt32(value) != 0, true
	case KindObject:
		primitive, ok := value.data.(*portableJavaPrimitive)
		if ok && primitive != nil && primitive.className() == "java.lang.Boolean" {
			return primitive.sleepValue().Int32() != 0, true
		}
	}
	return false, false
}

// portableJavaFileCanonicalValue crosses the same deliberate host-path
// encoding boundary as filesystem effects. The returned canonical spelling is
// therefore textual Java UTF-16; raw-byte provenance remains attached to the
// immutable abstract pathname returned by getPath, not invented for an OS
// pathname returned from canonicalization.
func portableJavaFileCanonicalValue(path Value) (Value, error) {
	if portableJavaFileInvalidValue(path) {
		return Null(), errors.New("java.io.IOException: Invalid file path")
	}
	absolute, err := portableJavaFileAbsoluteValue(path)
	if err != nil {
		return Null(), fmt.Errorf("java.io.IOException: Bad pathname: %w", err)
	}
	canonical, err := portableJavaFileCanonicalHostPath(portableJavaFileHostPath(absolute))
	if err != nil {
		return Null(), fmt.Errorf("java.io.IOException: Bad pathname: %w", err)
	}
	return String(canonical), nil
}

// JDK_Canonicalize resolves every existing prefix, removes dot segments, and
// still returns a unique absolute spelling for a missing or dangling suffix.
// Resolve components from left to right so "symlink/.." applies .. to the
// symlink target, as it does in the JDK rather than as filepath.Clean would.
// Once a component is missing, no descendant can be an existing symlink; the
// remaining suffix can therefore be cleaned lexically. A dangling symlink is
// retained under its original name, matching JDK_Canonicalize rather than
// replacing it with its nonexistent target. Other errors stay I/O errors.
func portableJavaFileCanonicalHostPath(path string) (string, error) {
	volume := filepath.VolumeName(path)
	remainder := path[len(volume):]
	resolved := volume
	if len(remainder) != 0 && os.IsPathSeparator(remainder[0]) {
		resolved += string(filepath.Separator)
	}
	components := strings.FieldsFunc(remainder, func(character rune) bool {
		return character <= 0xff && os.IsPathSeparator(uint8(character))
	})
	for index, component := range components {
		switch component {
		case ".":
			continue
		case "..":
			resolved = filepath.Dir(resolved)
			continue
		}
		candidate := filepath.Join(resolved, component)
		evaluated, err := filepath.EvalSymlinks(candidate)
		if err == nil {
			resolved = evaluated
			continue
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		resolved = candidate
		for _, suffix := range components[index+1:] {
			resolved = filepath.Join(resolved, suffix)
		}
		return filepath.Clean(resolved), nil
	}
	return filepath.Clean(resolved), nil
}

type portableJavaFileFilterKind uint8

const (
	portableJavaFileFilterNone portableJavaFileFilterKind = iota
	portableJavaFilenameFilter
	portableJavaFileFilter
)

func portableJavaProxyImplements(proxy *portableJavaProxy, target string) bool {
	if proxy == nil {
		return false
	}
	target = resolvePortableClassName(target)
	for _, implemented := range proxy.interfaces {
		if portableJavaAssignable(resolvePortableClassName(implemented), target) {
			return true
		}
	}
	return false
}

// portableJavaFileFilterArgument mirrors Sleep 2.1's ObjectUtilities overload
// matching. A bare Sleep closure is an exact interface match; for overloaded
// listFiles the official JAR selects FileFilter, while list has only the
// FilenameFilter overload. Explicit proxies retain their declared interface.
func portableJavaFileFilterArgument(invocation ObjectInvocation, files bool) (*portableJavaProxy, portableJavaFileFilterKind, bool) {
	argument := invocation.Arg(0)
	if _, ok := argument.Function(); ok {
		kind := portableJavaFilenameFilter
		if files {
			kind = portableJavaFileFilter
		}
		return &portableJavaProxy{closure: argument, runtime: invocation.Runtime}, kind, true
	}
	object, ok := argument.Object()
	if !ok {
		return nil, portableJavaFileFilterNone, false
	}
	proxy, ok := object.(*portableJavaProxy)
	if !ok || proxy == nil {
		return nil, portableJavaFileFilterNone, false
	}
	if files && portableJavaProxyImplements(proxy, "java.io.FileFilter") {
		return proxy, portableJavaFileFilter, true
	}
	if portableJavaProxyImplements(proxy, "java.io.FilenameFilter") {
		return proxy, portableJavaFilenameFilter, true
	}
	return nil, portableJavaFileFilterNone, false
}

func (file *portableJavaFile) portableList(files bool) (Value, bool, error) {
	return file.portableListContext(context.Background(), files, portableJavaFileFilterNone, nil)
}

func (file *portableJavaFile) portableListChild(name Value) Value {
	child := portableJavaFileNormalizeValue(name)
	// File.listFiles special-cases an empty parent pathname and creates each
	// result with new File(name), retaining a relative child.
	if len(sleepStringUnits(file.pathValue())) != 0 {
		child = portableJavaFileResolveValue(file.pathValue(), child)
	}
	return ObjectValue(newPortableJavaFile(child))
}

func portableJavaFileFilterStep(ctx context.Context) error {
	if err := executionContextError(ctx); err != nil {
		return err
	}
	return consumeInstruction(ctx)
}

func portableJavaFileFilterSignature(kind portableJavaFileFilterKind) string {
	if kind == portableJavaFileFilter {
		return "public abstract boolean java.io.FileFilter.accept(java.io.File)"
	}
	return "public abstract boolean java.io.FilenameFilter.accept(java.io.File,java.lang.String)"
}

func portableJavaFileFilterError(ctx context.Context, err error, filter *portableJavaProxy, kind portableJavaFileFilterKind) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, ErrResourceLimit) || errors.Is(err, ErrScriptUnloaded) {
		return &portableObjectCallbackError{cause: err}
	}
	var thrown *scriptThrow
	if !errors.As(err, &thrown) || thrown == nil {
		// Cancellation, limits, unloads, and importer callback failures are Go
		// execution-boundary errors, not Sleep thrown values. Keep them
		// authoritative instead of manufacturing a reflected Java exception.
		return &portableObjectCallbackError{cause: err}
	}
	if filter != nil {
		thrown.addFrame(fmt.Sprintf("   <Java>:-1 %s as %s", describeTraceValue(filter.closure), portableJavaFileFilterSignature(kind)))
	}
	exception := newPortableJavaException(err)
	if exception != nil {
		// ProxyInterface records the interface method and closure origin before
		// reflection turns the thrown value into File's soft error. Preserve that
		// getStackTrace() state; the official bridge does not add the outer File
		// listing method to this particular proxy failure.
		caller := currentFiber(ctx)
		if caller != nil && caller.closure != nil && caller.closure.script != nil {
			caller.closure.script.setStackTrace(exception.frames)
		}
		return exception
	}
	return err
}

func (file *portableJavaFile) portableListContext(ctx context.Context, files bool, kind portableJavaFileFilterKind, filter *portableJavaProxy) (Value, bool, error) {
	return file.portableListContextForRuntime(ctx, nil, files, kind, filter)
}

func (file *portableJavaFile) portableListContextForRuntime(ctx context.Context, runtime *Runtime, files bool, kind portableJavaFileFilterKind, filter *portableJavaProxy) (Value, bool, error) {
	if err := executionContextError(ctx); err != nil {
		return Null(), true, err
	}
	if file == nil || portableJavaFileInvalidValue(file.pathValue()) {
		return Null(), true, nil
	}
	directory, err := os.Open(portableJavaFileFilesystemPathValue(file.pathValue()))
	if err != nil {
		return Null(), true, nil
	}
	defer directory.Close()
	// File.ReadDir preserves filesystem enumeration order. Read bounded batches
	// so a rejecting filter cannot force a proportional DirEntry/Value scratch
	// allocation before any accepted result entry is reserved.
	var values []Value
	for {
		entries, readErr := directory.ReadDir(128)
		for _, entry := range entries {
			if err := portableJavaFileFilterStep(ctx); err != nil {
				return Null(), true, err
			}
			name := String(entry.Name())
			switch kind {
			case portableJavaFilenameFilter:
				accepted, err := filter.call(ctx, "accept", []Argument{
					{Value: ObjectValue(file)},
					{Value: name},
				}, true)
				if err != nil {
					return Null(), true, portableJavaFileFilterError(ctx, err, filter, kind)
				}
				if sleepInt32(accepted) == 0 {
					continue
				}
			case portableJavaFileFilter:
				child := file.portableListChild(name)
				accepted, err := filter.call(ctx, "accept", []Argument{{Value: child}}, true)
				if err != nil {
					return Null(), true, portableJavaFileFilterError(ctx, err, filter, kind)
				}
				if sleepInt32(accepted) == 0 {
					continue
				}
				// OpenJDK adds the same File instance passed to FileFilter.accept.
				if err := reserveCollectionEntries(runtime, 1); err != nil {
					return Null(), true, err
				}
				values = append(values, child)
				continue
			}
			if err := reserveCollectionEntries(runtime, 1); err != nil {
				return Null(), true, err
			}
			if files {
				values = append(values, file.portableListChild(name))
			} else {
				values = append(values, name)
			}
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return Null(), true, nil
		}
	}
	// Sleep's ObjectUtilities.BuildScalar eagerly converts every Java
	// method-returned reference array into a fresh mutable ListContainer.
	return ArrayValue(NewArray(values...)), true, nil
}

func portableJavaFileInvalid(path string) bool {
	return portableJavaFileInvalidValue(String(path))
}

func portableJavaFileInvalidValue(path Value) bool {
	return portableJavaFileInvalidForGOOS(path, goruntime.GOOS)
}

// portableJavaFileInvalidForGOOS mirrors the pinned OpenJDK platform checks.
// WinNTFileSystem rejects NUL and any name component ending in a space. With
// its default ADS setting it deliberately leaves other Win32-invalid
// characters to the native operation. Keeping goos explicit makes the
// otherwise Windows-only rule testable on every host.
func portableJavaFileInvalidForGOOS(path Value, goos string) bool {
	units := sleepStringUnits(path)
	for _, unit := range units {
		if unit == 0 {
			return true
		}
	}
	if goos != "windows" || len(units) == 0 {
		return false
	}
	if units[len(units)-1] == ' ' {
		return true
	}
	for index := 0; index+1 < len(units); index++ {
		if units[index] == ' ' && units[index+1] == '\\' {
			return true
		}
	}
	return false
}

// OpenJDK retains an empty abstract pathname for equality, naming, and hash
// operations, but resolves it against the process working directory for File's
// metadata and access methods. Go's os.Stat("") instead fails, so make that
// inherited JVM boundary explicit without changing the stored pathname.
func portableJavaFileFilesystemPath(path string) string {
	return portableJavaFileFilesystemPathValue(String(path))
}

func portableJavaFileFilesystemPathValue(path Value) string {
	if len(sleepStringUnits(path)) != 0 {
		return portableJavaFileHostPath(path)
	}
	absolute, err := portableJavaFileAbsoluteValue(path)
	if err != nil {
		return portableJavaFileHostPath(path)
	}
	return portableJavaFileHostPath(absolute)
}

func portableJavaFileValue(value Value) (*portableJavaFile, bool) {
	object, ok := value.Object()
	if !ok {
		return nil, false
	}
	file, ok := object.(*portableJavaFile)
	return file, ok && file != nil
}

func portableJavaFileClassCast(value Value) error {
	actual := "java.lang.Object"
	switch value.Kind() {
	case KindArray:
		actual = "java.util.LinkedList"
	case KindHash:
		actual = "java.util.HashMap"
	default:
		if class, ok := portableObjectClass(value); ok {
			actual = class
		}
	}
	return fmt.Errorf("java.lang.ClassCastException: class %s cannot be cast to class %s", actual, portableJavaFileClass)
}

// portableJavaFileCompare follows the platform FileSystem comparator. The
// string wrapper remains for the I/O builtins; File values use the Value form
// below so binary provenance and unpaired UTF-16 units are never reparsed.
func portableJavaFileCompare(left, right string) int32 {
	return portableJavaFileCompareValues(String(left), String(right))
}

func portableJavaFileCompareValues(left, right Value) int32 {
	if goruntime.GOOS == "windows" {
		return portableJavaStringCompareIgnoreCaseValues(left, right)
	}
	return portableJavaStringCompareValues(left, right)
}

func portableJavaStringCompare(left, right string) int32 {
	return portableJavaStringCompareValues(String(left), String(right))
}

func portableJavaStringCompareValues(left, right Value) int32 {
	leftUnits := sleepStringUnits(left)
	rightUnits := sleepStringUnits(right)
	limit := len(leftUnits)
	if len(rightUnits) < limit {
		limit = len(rightUnits)
	}
	for index := 0; index < limit; index++ {
		if leftUnits[index] != rightUnits[index] {
			return int32(leftUnits[index]) - int32(rightUnits[index])
		}
	}
	return int32(len(leftUnits) - len(rightUnits))
}

func portableJavaStringCompareIgnoreCase(left, right string) int32 {
	return portableJavaStringCompareIgnoreCaseValues(String(left), String(right))
}

func portableJavaStringCompareIgnoreCaseValues(left, right Value) int32 {
	leftUnits, rightUnits := sleepStringUnits(left), sleepStringUnits(right)
	leftIndex, rightIndex := 0, 0
	for leftIndex < len(leftUnits) && rightIndex < len(rightUnits) {
		leftCodePoint, leftWidth := sleepUTF16CodePointAt(leftUnits, leftIndex)
		rightCodePoint, rightWidth := sleepUTF16CodePointAt(rightUnits, rightIndex)
		leftRune, rightRune := rune(leftCodePoint), rune(rightCodePoint)
		if leftRune == rightRune {
			leftIndex += leftWidth
			rightIndex += rightWidth
			continue
		}
		leftRune, rightRune = sleepJavaSimpleCase(leftRune, true), sleepJavaSimpleCase(rightRune, true)
		if leftRune == rightRune {
			leftIndex += leftWidth
			rightIndex += rightWidth
			continue
		}
		leftRune, rightRune = sleepJavaSimpleCase(leftRune, false), sleepJavaSimpleCase(rightRune, false)
		if leftRune != rightRune {
			return int32(leftRune - rightRune)
		}
		leftIndex += leftWidth
		rightIndex += rightWidth
	}
	return int32(len(leftUnits) - len(rightUnits))
}

func portableJavaFileHash(path string) int32 {
	return portableJavaFileHashValue(String(path))
}

func portableJavaFileHashValue(path Value) int32 {
	units := sleepStringUnits(path)
	if goruntime.GOOS == "windows" {
		// WinNTFileSystem uses full Locale.ENGLISH lower-casing. English and
		// Locale.ROOT share these Unicode mappings, including contextual and
		// one-to-many results, so reuse OPFOR's pinned Unicode 17 implementation.
		units = sleepStringUnits(sleepStringMapCase(path, false))
	}
	var hash int32
	for _, unit := range units {
		hash = 31*hash + int32(unit)
	}
	return hash ^ 1234321
}

func portableJavaFileBoolean(value bool) Value {
	if value {
		return Int(1)
	}
	return Int(0)
}

func portableJavaFileSpaceBytes(blocks, blockSize uint64) int64 {
	if blockSize != 0 && blocks > uint64(math.MaxInt64)/blockSize {
		return math.MaxInt64
	}
	return int64(blocks * blockSize)
}

// portableJavaFileNormalize mirrors FileSystem.normalize without resolving dot
// segments. Java File stores an abstract pathname, not a canonical pathname.
func portableJavaFileNormalize(path string) string {
	return portableJavaFileNormalizeValue(String(path)).String()
}

func portableJavaFileIsRoot(path string) bool {
	return portableJavaFileIsRootValue(String(path), goruntime.GOOS)
}

func portableJavaFileResolve(parent, child string) string {
	return portableJavaFileResolveValue(String(parent), String(child)).String()
}

func portableJavaFileAbsolute(path string) (string, error) {
	value, err := portableJavaFileAbsoluteValue(String(path))
	return value.String(), err
}

func portableJavaFileName(path string) string {
	return portableJavaFileNameValue(String(path)).String()
}

func portableJavaFileParent(path string) (string, bool) {
	parent, ok := portableJavaFileParentValue(String(path))
	return parent.String(), ok
}

func portableJavaFileNormalizeValue(path Value) Value {
	return portableJavaFileNormalizeValueForGOOS(sleepStringCoercion(path), goruntime.GOOS)
}

func portableJavaFileNormalizeValueForGOOS(path Value, goos string) Value {
	if goos == "windows" {
		return portableJavaFileNormalizeWindows(path)
	}
	units := sleepStringUnits(path)
	raw := sleepStringRawMask(path)
	doubleSlash := -1
	for index := 0; index+1 < len(units); index++ {
		if units[index] == '/' && units[index+1] == '/' {
			doubleSlash = index
			break
		}
	}
	if doubleSlash < 0 && (len(units) == 0 || units[len(units)-1] != '/') {
		return path
	}
	offset := doubleSlash
	if offset < 0 {
		offset = len(units) - 1
	}
	end := len(units)
	for end > offset && units[end-1] == '/' {
		end--
	}
	if end == 0 {
		if len(units) != 0 {
			return sleepStringValueFromUnits(units[:1], raw[:1])
		}
		return String("/")
	}
	if end == offset {
		return sleepStringValueFromUnits(units[:offset], raw[:offset])
	}
	resultUnits := append([]uint16(nil), units[:offset]...)
	resultRaw := append([]bool(nil), raw[:offset]...)
	var previous uint16
	for index := offset; index < end; index++ {
		unit := units[index]
		if previous == '/' && unit == '/' {
			continue
		}
		resultUnits = append(resultUnits, unit)
		resultRaw = append(resultRaw, raw[index])
		previous = unit
	}
	return sleepStringValueFromUnits(resultUnits, resultRaw)
}

func portableJavaFileNormalizeWindows(path Value) Value {
	units := sleepStringUnits(path)
	raw := sleepStringRawMask(path)
	if portableJavaFileUnitsHavePrefix(units, []uint16{'\\', '\\', '?', '\\'}) {
		if portableJavaFileUnitsHavePrefix(units[4:], []uint16{'U', 'N', 'C', '\\'}) {
			units = append([]uint16{'\\', '\\'}, units[8:]...)
			raw = append([]bool{false, false}, raw[8:]...)
		} else {
			units = units[4:]
			raw = raw[4:]
		}
	}
	if len(units) == 0 {
		return sleepStringValueFromUnits(units, raw)
	}

	// Preserve an already-normal pathname byte-for-byte, matching the JDK's
	// fast path and OPFOR's additional provenance contract.
	previous := uint16(0)
	normal := true
	for index, unit := range units {
		if unit == '/' || unit == '\\' && previous == '\\' && index > 1 || unit == ':' && index > 1 {
			normal = false
			break
		}
		previous = unit
	}
	if normal && units[len(units)-1] != '\\' {
		return sleepStringValueFromUnits(units, raw)
	}

	source := 0
	for source < len(units) && portableJavaFileWindowsSlash(units[source]) {
		source++
	}
	resultUnits := make([]uint16, 0, len(units))
	resultRaw := make([]bool, 0, len(raw))
	if len(units)-source >= 2 && portableJavaFileASCIIAlpha(units[source]) && units[source+1] == ':' {
		resultUnits = append(resultUnits, units[source], ':')
		resultRaw = append(resultRaw, raw[source], raw[source+1])
		source += 2
	} else {
		source = 0
		if len(units) >= 2 && portableJavaFileWindowsSlash(units[0]) && portableJavaFileWindowsSlash(units[1]) {
			source = 1
			resultUnits = append(resultUnits, '\\')
			resultRaw = append(resultRaw, units[0] == '\\' && raw[0])
		}
	}

	for source < len(units) {
		unit := units[source]
		unitRaw := raw[source]
		source++
		if portableJavaFileWindowsSlash(unit) {
			for source < len(units) && portableJavaFileWindowsSlash(units[source]) {
				source++
			}
			if source == len(units) {
				switch {
				case len(resultUnits) == 2 && resultUnits[1] == ':':
					resultUnits = append(resultUnits, '\\')
					resultRaw = append(resultRaw, false)
				case len(resultUnits) == 0:
					resultUnits = append(resultUnits, '\\')
					resultRaw = append(resultRaw, false)
				case len(resultUnits) == 1 && resultUnits[0] == '\\':
					resultUnits = append(resultUnits, '\\')
					resultRaw = append(resultRaw, false)
				}
				break
			}
			resultUnits = append(resultUnits, '\\')
			resultRaw = append(resultRaw, unit == '\\' && unitRaw)
			continue
		}
		resultUnits = append(resultUnits, unit)
		resultRaw = append(resultRaw, unitRaw)
	}
	return sleepStringValueFromUnits(resultUnits, resultRaw)
}

func portableJavaFileResolveValue(parent, child Value) Value {
	return portableJavaFileResolveValueForGOOS(parent, child, goruntime.GOOS)
}

func portableJavaFileResolveValueForGOOS(parent, child Value, goos string) Value {
	parentUnits, childUnits := sleepStringUnits(parent), sleepStringUnits(child)
	if goos != "windows" {
		if len(childUnits) == 0 {
			return parent
		}
		var result Value
		switch {
		case childUnits[0] == '/' && len(parentUnits) == 1 && parentUnits[0] == '/':
			result = child
		case childUnits[0] == '/' || len(parentUnits) == 1 && parentUnits[0] == '/':
			result = sleepStringConcat(parent, child)
		default:
			result = sleepStringConcat(parent, String("/"), child)
		}
		units := sleepStringUnits(result)
		if len(units) > 1 && units[len(units)-1] == '/' {
			return sleepStringValueSlice(result, 0, len(units)-1)
		}
		return result
	}
	if len(parentUnits) == 0 {
		return child
	}
	if len(childUnits) == 0 {
		return parent
	}
	childStart, parentEnd := 0, len(parentUnits)
	directoryRelative := len(parentUnits) == 2 && portableJavaFileASCIIAlpha(parentUnits[0]) && parentUnits[1] == ':'
	if len(childUnits) > 1 && childUnits[0] == '\\' {
		if childUnits[1] == '\\' {
			childStart = 2
		} else if !directoryRelative {
			childStart = 1
		}
		if len(childUnits) == childStart {
			if parentUnits[len(parentUnits)-1] == '\\' {
				return sleepStringValueSlice(parent, 0, len(parentUnits)-1)
			}
			return parent
		}
	}
	if parentUnits[len(parentUnits)-1] == '\\' {
		parentEnd--
	}
	left := sleepStringValueSlice(parent, 0, parentEnd)
	right := sleepStringValueSlice(child, childStart, len(childUnits))
	var result Value
	if childUnits[childStart] == '\\' || directoryRelative {
		result = sleepStringConcat(left, right)
	} else {
		result = sleepStringConcat(left, String("\\"), right)
	}
	resultUnits := sleepStringUnits(result)
	if len(resultUnits) > 1 && resultUnits[len(resultUnits)-1] == '\\' && resultUnits[len(resultUnits)-2] != ':' {
		return sleepStringValueSlice(result, 0, len(resultUnits)-1)
	}
	return result
}

func portableJavaFileAbsoluteValue(path Value) (Value, error) {
	if portableJavaFileIsAbsoluteValue(path, goruntime.GOOS) {
		return path, nil
	}
	workingDirectory, err := os.Getwd()
	if err != nil {
		return Null(), err
	}
	working := portableJavaFileNormalizeValue(String(workingDirectory))
	if goruntime.GOOS != "windows" {
		return portableJavaFileResolveValue(working, path), nil
	}
	units := sleepStringUnits(path)
	prefix := portableJavaFilePrefixLength(path, "windows")
	switch prefix {
	case 0:
		return portableJavaFileResolveValue(working, path), nil
	case 1:
		workingUnits := sleepStringUnits(working)
		if len(workingUnits) >= 2 && portableJavaFileASCIIAlpha(workingUnits[0]) && workingUnits[1] == ':' {
			return sleepStringConcat(sleepStringValueSlice(working, 0, 2), path), nil
		}
		return sleepStringConcat(working, path), nil
	case 2:
		workingUnits := sleepStringUnits(working)
		if len(workingUnits) >= 2 && portableJavaFileEqualFoldASCII(workingUnits[0], units[0]) && workingUnits[1] == ':' {
			return portableJavaFileResolveValue(working, sleepStringValueSlice(path, 2, len(units))), nil
		}
		return sleepStringConcat(sleepStringValueSlice(path, 0, 2), String("\\"), sleepStringValueSlice(path, 2, len(units))), nil
	default:
		return path, nil
	}
}

func portableJavaFileNameValue(path Value) Value {
	return portableJavaFileNameValueForGOOS(path, goruntime.GOOS)
}

func portableJavaFileNameValueForGOOS(path Value, goos string) Value {
	units := sleepStringUnits(path)
	separator := portableJavaFileSeparator(goos)
	index := portableJavaFileLastIndexUnit(units, separator)
	prefixLength := portableJavaFilePrefixLength(path, goos)
	if index < prefixLength {
		return sleepStringValueSlice(path, prefixLength, len(units))
	}
	return sleepStringValueSlice(path, index+1, len(units))
}

func portableJavaFileParentValue(path Value) (Value, bool) {
	units := sleepStringUnits(path)
	if len(units) == 0 || portableJavaFileIsRootValue(path, goruntime.GOOS) {
		return Null(), false
	}
	separator := portableJavaFileSeparator(goruntime.GOOS)
	index := portableJavaFileLastIndexUnit(units, separator)
	prefixLength := portableJavaFilePrefixLength(path, goruntime.GOOS)
	if index < prefixLength {
		if prefixLength != 0 && len(units) > prefixLength {
			return sleepStringValueSlice(path, 0, prefixLength), true
		}
		return Null(), false
	}
	return sleepStringValueSlice(path, 0, index), true
}

func portableJavaFileIsAbsoluteValue(path Value, goos string) bool {
	prefix := portableJavaFilePrefixLength(path, goos)
	if goos != "windows" {
		return prefix != 0
	}
	units := sleepStringUnits(path)
	return prefix == 3 || prefix == 2 && len(units) != 0 && units[0] == '\\'
}

func portableJavaFileIsRootValue(path Value, goos string) bool {
	units := sleepStringUnits(path)
	if goos != "windows" {
		return len(units) == 1 && units[0] == '/'
	}
	return len(units) == 3 && portableJavaFileASCIIAlpha(units[0]) && units[1] == ':' && units[2] == '\\'
}

func portableJavaFilePrefixLength(path Value, goos string) int {
	units := sleepStringUnits(path)
	if len(units) == 0 {
		return 0
	}
	if goos != "windows" {
		if units[0] == '/' {
			return 1
		}
		return 0
	}
	first := units[0]
	second := uint16(0)
	if len(units) > 1 {
		second = units[1]
	}
	if first == '\\' {
		if second == '\\' {
			return 2
		}
		return 1
	}
	if portableJavaFileASCIIAlpha(first) && second == ':' {
		if len(units) > 2 && units[2] == '\\' {
			return 3
		}
		return 2
	}
	return 0
}

func portableJavaFileDefaultParentValue(goos string) Value {
	if goos == "windows" {
		return String("\\")
	}
	return String("/")
}

// portableJavaFileHostPath deliberately encodes Java UTF-16 for Go's OS APIs.
// Provenance never changes a Java char: binary C3 A9 therefore becomes the two
// characters U+00C3 U+00A9, not textual U+00E9. Go cannot carry an unpaired
// surrogate through its string-based filesystem API. OpenJDK's Unix platform
// encoder replaces that malformed unit with '?'; Go's Windows conversion uses
// U+FFFD. File identity remains exact on either side of this explicit boundary.
func portableJavaFileHostPath(path Value) string {
	return portableJavaFileHostPathForGOOS(path, goruntime.GOOS)
}

func portableJavaFileHostPathForGOOS(path Value, goos string) string {
	units := sleepStringUnits(path)
	var result strings.Builder
	for index := 0; index < len(units); {
		codePoint, width := sleepUTF16CodePointAt(units, index)
		if width == 1 && units[index] >= 0xd800 && units[index] <= 0xdfff {
			if goos == "windows" {
				result.WriteRune(unicode.ReplacementChar)
			} else {
				result.WriteByte('?')
			}
		} else {
			result.WriteRune(rune(codePoint))
		}
		index += width
	}
	return result.String()
}

func portableJavaFileCanRead(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	if goruntime.GOOS == "windows" {
		// WinNTFileSystem.checkAccess0 reports read/execute access whenever
		// GetFileAttributes succeeds; it does not attempt a content open.
		return true
	}
	if info.IsDir() {
		directory, openErr := os.Open(path)
		if openErr != nil {
			return false
		}
		_, readErr := directory.Readdirnames(1)
		closeErr := directory.Close()
		return (readErr == nil || errors.Is(readErr, io.EOF)) && closeErr == nil
	}
	if info.Mode().IsRegular() {
		file, openErr := os.Open(path)
		if openErr != nil {
			return false
		}
		return file.Close() == nil
	}
	// Opening FIFOs and devices can block. For those uncommon targets the
	// portable adapter intentionally falls back to advertised permission bits.
	return info.Mode().Perm()&0o444 != 0
}

func portableJavaFileCanWrite(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	if goruntime.GOOS == "windows" {
		// Go's Windows FileMode exposes the read-only attribute as missing
		// write bits. OpenJDK ignores that attribute for directories.
		return info.IsDir() || info.Mode().Perm()&0o222 != 0
	}
	if info.Mode().IsRegular() {
		file, openErr := os.OpenFile(path, os.O_WRONLY, 0)
		if openErr != nil {
			return false
		}
		return file.Close() == nil
	}
	// The standard library has no non-mutating access(2) equivalent shared by
	// Unix and Windows. Opening a directory O_WRONLY always fails, so directory
	// and special-file checks remain a documented permission-bit approximation.
	return info.Mode().Perm()&0o222 != 0
}

func portableJavaFileValueHasPrefixUnit(value Value, prefix uint16) bool {
	units := sleepStringUnits(value)
	return len(units) != 0 && units[0] == prefix
}

func portableJavaFileSeparator(goos string) uint16 {
	if goos == "windows" {
		return '\\'
	}
	return '/'
}

func portableJavaFileWindowsSlash(unit uint16) bool { return unit == '\\' || unit == '/' }

func portableJavaFileASCIIAlpha(unit uint16) bool {
	return unit >= 'a' && unit <= 'z' || unit >= 'A' && unit <= 'Z'
}

func portableJavaFileEqualFoldASCII(left, right uint16) bool {
	if left >= 'a' && left <= 'z' {
		left -= 'a' - 'A'
	}
	if right >= 'a' && right <= 'z' {
		right -= 'a' - 'A'
	}
	return left == right
}

func portableJavaFileLastIndexUnit(units []uint16, target uint16) int {
	for index := len(units) - 1; index >= 0; index-- {
		if units[index] == target {
			return index
		}
	}
	return -1
}

func portableJavaFileUnitsHavePrefix(units, prefix []uint16) bool {
	if len(units) < len(prefix) {
		return false
	}
	for index := range prefix {
		if units[index] != prefix[index] {
			return false
		}
	}
	return true
}
