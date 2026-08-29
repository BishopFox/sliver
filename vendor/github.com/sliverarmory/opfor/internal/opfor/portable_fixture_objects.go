package opfor

import (
	"archive/zip"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

type portableFixtureObjectState struct {
	mu         sync.RWMutex
	authorized map[ScriptID]map[string]struct{}
	squeezeBox portableSqueezeBoxClass
}

func newPortableFixtureObjectState() *portableFixtureObjectState {
	return &portableFixtureObjectState{
		authorized: make(map[ScriptID]map[string]struct{}),
		squeezeBox: portableSqueezeBoxClass{
			aStringField: "this is a string field",
			aDoubleField: 3,
		},
	}
}

func (runtime *Runtime) portableFixtureState() *portableFixtureObjectState {
	if runtime == nil {
		return nil
	}
	runtime.mu.Lock()
	if runtime.fixtureObjects == nil {
		runtime.fixtureObjects = newPortableFixtureObjectState()
	}
	state := runtime.fixtureObjects
	runtime.mu.Unlock()
	return state
}

func (state *portableFixtureObjectState) authorize(script ScriptID, classes []string) {
	if state == nil || script == 0 {
		return
	}
	state.mu.Lock()
	allowed := state.authorized[script]
	if allowed == nil {
		allowed = make(map[string]struct{}, len(classes))
		state.authorized[script] = allowed
	}
	for _, class := range classes {
		allowed[class] = struct{}{}
	}
	state.mu.Unlock()
}

func (state *portableFixtureObjectState) allows(script ScriptID, class string) bool {
	if state == nil || script == 0 {
		return false
	}
	state.mu.RLock()
	_, ok := state.authorized[script][class]
	state.mu.RUnlock()
	return ok
}

type portableFixtureImportSpec struct {
	classes []string
	entries []string
}

var portableFixtureImports = map[string]portableFixtureImportSpec{
	"org.hick.blah.SqueezeBox": {
		classes: []string{"org.hick.blah.SqueezeBox"},
		entries: []string{"org/hick/blah/SqueezeBox.class"},
	},
	"org.hick.blah.*": {
		classes: []string{"org.hick.blah.SqueezeBox"},
		entries: []string{"org/hick/blah/SqueezeBox.class"},
	},
	"org.hick.tests.*": {
		classes: []string{"org.hick.tests.TestLoadable"},
		entries: []string{"org/hick/tests/TestLoadable.class"},
	},
	"org.hick.tests.TestLoadable": {
		classes: []string{"org.hick.tests.TestLoadable"},
		entries: []string{"org/hick/tests/TestLoadable.class"},
	},
	"sleep.*": {
		classes: []string{"sleep.ArrayTest1"},
		entries: []string{"sleep/ArrayTest1.class"},
	},
	"com.eric.*": {
		classes: []string{"com.eric.Eric", "com.eric.Person"},
		entries: []string{"com/eric/Eric.class", "com/eric/Person.class"},
	},
}

// portableFixtureImport recognizes only the pinned, source-audited classes
// represented by this file. Every other import remains at the importer Host
// boundary; OPFOR never interprets arbitrary JAR bytecode.
func portableFixtureImport(runtime *Runtime, script ScriptID, target, archive string) bool {
	spec, ok := portableFixtureImports[target]
	if !ok || runtime == nil || archive == "" {
		return false
	}
	path := archive
	if resolver := runtime.concreteFileSourceResolver(); resolver != nil {
		// ImportManager and BasicUtilities both resolve explicit source
		// containers through ParserConfig.findJarFile. Use the same direct-first
		// classpath lookup for the bounded fixture adapter; otherwise validation
		// can find a classpath JAR which this second, authorization-only check then
		// incorrectly reopens beneath the runtime base directory.
		path = resolver.resolveContainer(path)
	} else if !filepath.IsAbs(path) {
		path = filepath.Clean(path)
	}
	reader, err := zip.OpenReader(path)
	if err != nil {
		return false
	}
	defer reader.Close()
	found := make(map[string]bool, len(spec.entries))
	for _, file := range reader.File {
		for _, entry := range spec.entries {
			if file.Name == entry {
				found[entry] = true
			}
		}
	}
	for _, entry := range spec.entries {
		if !found[entry] {
			return false
		}
	}
	runtime.portableFixtureState().authorize(script, spec.classes)
	return true
}

var portableFixtureIdentity atomic.Uint64

type portableFieldError struct {
	message string
}

func (err *portableFieldError) Error() string {
	if err == nil {
		return ""
	}
	return err.message
}

type portableSqueezeBoxClass struct {
	mu           sync.RWMutex
	aStringField string
	aDoubleField float64
}

type portableSqueezeBox struct {
	mu                   sync.RWMutex
	class                *portableSqueezeBoxClass
	runtime              *Runtime
	instanceStringField  string
	instanceBooleanField bool
	sq                   int32
	identity             uint64
}

func newPortableSqueezeBox(runtime *Runtime, class *portableSqueezeBoxClass) *portableSqueezeBox {
	return &portableSqueezeBox{
		class:                class,
		runtime:              runtime,
		instanceStringField:  "this is also a string field",
		instanceBooleanField: true,
		sq:                   33,
		identity:             portableFixtureIdentity.Add(1),
	}
}

func (box *portableSqueezeBox) String() string {
	if box == nil {
		return "null"
	}
	return fmt.Sprintf("org.hick.blah.SqueezeBox@%x", box.identity)
}

func (box *portableSqueezeBox) invoke(invocation ObjectInvocation) (Value, bool, error) {
	if box == nil {
		return Null(), false, nil
	}
	if invocation.Op == ObjectTypeCheck {
		class := resolvePortableClassName(invocation.Class)
		return Bool(class == "org.hick.blah.SqueezeBox" || class == "java.lang.Object"), true, nil
	}
	if invocation.Op == ObjectInvoke && invocation.Message == "doStuff" && len(invocation.Arguments) == 1 {
		return box.doStuff(invocation)
	}
	if len(invocation.Arguments) == 0 && (invocation.Op == ObjectInvoke || invocation.Op == ObjectGet) {
		switch invocation.Message {
		case "printValues":
			return box.printValues()
		case "squeeze":
			box.mu.Lock()
			box.sq++
			value := box.sq
			box.mu.Unlock()
			return Int(value), true, nil
		case "toString":
			return String(box.String()), true, nil
		}
		if value, ok := box.getField(invocation.Message); ok {
			return value, true, nil
		}
	}
	if invocation.Op == ObjectSet && len(invocation.Arguments) == 1 {
		return Null(), true, box.setField(invocation.Message, invocation.Arg(0))
	}
	return Null(), false, nil
}

func (box *portableSqueezeBox) printValues() (Value, bool, error) {
	box.class.mu.RLock()
	staticString := box.class.aStringField
	staticDouble := box.class.aDoubleField
	box.class.mu.RUnlock()
	box.mu.RLock()
	instanceString := box.instanceStringField
	instanceBoolean := box.instanceBooleanField
	box.mu.RUnlock()
	if box.runtime == nil || box.runtime.stdout == nil {
		return Null(), true, nil
	}
	_, err := fmt.Fprintf(box.runtime.stdout,
		"static members:\naStringField '%s' and aDoubleField = %s\ninstance members:\ninstanceStringField '%s' instanceBooleanField = %t\n",
		staticString, portableJavaDoubleText(staticDouble), instanceString, instanceBoolean)
	return Null(), true, err
}

func (box *portableSqueezeBox) getField(name string) (Value, bool) {
	switch name {
	case "aStringField", "aDoubleField":
		return box.class.getField(name)
	}
	box.mu.RLock()
	defer box.mu.RUnlock()
	switch name {
	case "instanceStringField":
		return String(box.instanceStringField), true
	case "instanceBooleanField":
		return Bool(box.instanceBooleanField), true
	case "sq":
		return Int(box.sq), true
	default:
		return Null(), false
	}
}

func (box *portableSqueezeBox) setField(name string, value Value) error {
	switch name {
	case "aStringField", "aDoubleField":
		return box.class.setField(name, value)
	}
	box.mu.Lock()
	defer box.mu.Unlock()
	switch name {
	case "instanceStringField":
		converted, ok := portableFixtureString(value)
		if !ok {
			return portableFixtureConversionError(value, "class java.lang.String")
		}
		box.instanceStringField = converted
	case "instanceBooleanField":
		converted, ok := portableFixtureBoolean(value)
		if !ok {
			return portableFixtureConversionError(value, "boolean")
		}
		box.instanceBooleanField = converted
	case "sq":
		converted, ok := portableFixtureInt(value)
		if !ok {
			return portableFixtureConversionError(value, "int")
		}
		box.sq = converted
	default:
		return &portableFieldError{message: "no field named " + name + " in class org.hick.blah.SqueezeBox"}
	}
	return nil
}

func (class *portableSqueezeBoxClass) getField(name string) (Value, bool) {
	if class == nil {
		return Null(), false
	}
	class.mu.RLock()
	defer class.mu.RUnlock()
	switch name {
	case "aStringField":
		return String(class.aStringField), true
	case "aDoubleField":
		return Double(class.aDoubleField), true
	default:
		return Null(), false
	}
}

func (class *portableSqueezeBoxClass) setField(name string, value Value) error {
	if class == nil {
		return &portableFieldError{message: "no field named " + name + " in class org.hick.blah.SqueezeBox"}
	}
	class.mu.Lock()
	defer class.mu.Unlock()
	switch name {
	case "aStringField":
		converted, ok := portableFixtureString(value)
		if !ok {
			return portableFixtureConversionError(value, "class java.lang.String")
		}
		class.aStringField = converted
	case "aDoubleField":
		converted, ok := portableFixtureDouble(value)
		if !ok {
			return portableFixtureConversionError(value, "double")
		}
		class.aDoubleField = converted
	default:
		return &portableFieldError{message: "no field named " + name + " in class org.hick.blah.SqueezeBox"}
	}
	return nil
}

type portableEric struct {
	mu       sync.RWMutex
	fname    string
	lname    string
	identity uint64
}

// portableTestLoadable is an inert representation of the pinned BSD fixture
// class. Importing the class authorizes its name, but OPFOR never executes its
// bytecode. The observable direct-object behavior used by impfrom4 is simply a
// successfully constructed object followed by Sleep's ordinary missing-method
// warning for scriptUnloaded() with the wrong arity.
type portableTestLoadable struct {
	identity uint64
}

func newPortableTestLoadable() *portableTestLoadable {
	return &portableTestLoadable{identity: portableFixtureIdentity.Add(1)}
}

func (loadable *portableTestLoadable) String() string {
	if loadable == nil {
		return "null"
	}
	return fmt.Sprintf("org.hick.tests.TestLoadable@%x", loadable.identity)
}

func newPortableEric(fname, lname string) *portableEric {
	return &portableEric{fname: fname, lname: lname, identity: portableFixtureIdentity.Add(1)}
}

func (eric *portableEric) String() string {
	if eric == nil {
		return "null"
	}
	return fmt.Sprintf("com.eric.Eric@%x", eric.identity)
}

func (eric *portableEric) invoke(invocation ObjectInvocation) (Value, bool, error) {
	if eric == nil {
		return Null(), false, nil
	}
	if invocation.Op == ObjectTypeCheck {
		class := resolvePortableClassName(invocation.Class)
		return Bool(class == "com.eric.Eric" || class == "com.eric.Person" || class == "java.lang.Object"), true, nil
	}
	if len(invocation.Arguments) == 0 && (invocation.Op == ObjectInvoke || invocation.Op == ObjectGet) {
		eric.mu.RLock()
		defer eric.mu.RUnlock()
		switch invocation.Message {
		case "fname":
			return String(eric.fname), true, nil
		case "lname":
			return String(eric.lname), true, nil
		case "toString":
			return String(eric.String()), true, nil
		}
	}
	if invocation.Op == ObjectSet && len(invocation.Arguments) == 1 {
		converted, ok := portableFixtureString(invocation.Arg(0))
		if !ok {
			return Null(), true, portableFixtureConversionError(invocation.Arg(0), "class java.lang.String")
		}
		eric.mu.Lock()
		switch invocation.Message {
		case "fname":
			eric.fname = converted
		case "lname":
			eric.lname = converted
		default:
			eric.mu.Unlock()
			return Null(), true, &portableFieldError{message: "no field named " + invocation.Message + " in class com.eric.Eric"}
		}
		eric.mu.Unlock()
		return Null(), true, nil
	}
	return Null(), false, nil
}

func portableFixtureClass(invocation ObjectInvocation) (Value, bool, error) {
	if invocation.Runtime == nil {
		return Null(), false, nil
	}
	class := resolvePortableClassName(invocation.Class)
	state := invocation.Runtime.portableFixtureState()
	if !state.allows(invocation.Script, class) {
		return Null(), false, nil
	}
	switch class {
	case "org.hick.blah.SqueezeBox":
		if invocation.Op == ObjectConstruct && len(invocation.Arguments) == 0 {
			return ObjectValue(newPortableSqueezeBox(invocation.Runtime, &state.squeezeBox)), true, nil
		}
		if len(invocation.Arguments) == 0 && (invocation.Op == ObjectGet || invocation.Op == ObjectInvoke) {
			if value, ok := state.squeezeBox.getField(invocation.Message); ok {
				return value, true, nil
			}
		}
		if invocation.Op == ObjectSet && len(invocation.Arguments) == 1 {
			return Null(), true, state.squeezeBox.setField(invocation.Message, invocation.Arg(0))
		}
	case "com.eric.Eric":
		if invocation.Op == ObjectConstruct && len(invocation.Arguments) == 2 {
			fname, fnameOK := portableFixtureString(invocation.Arg(0))
			lname, lnameOK := portableFixtureString(invocation.Arg(1))
			if fnameOK && lnameOK {
				return ObjectValue(newPortableEric(fname, lname)), true, nil
			}
		}
	case "org.hick.tests.TestLoadable":
		if invocation.Op == ObjectConstruct && len(invocation.Arguments) == 0 {
			return ObjectValue(newPortableTestLoadable()), true, nil
		}
	case "sleep.ArrayTest1":
		if invocation.Op == ObjectConstruct && len(invocation.Arguments) == 0 {
			return ObjectValue(&portableArrayTest1{}), true, nil
		}
	}
	return Null(), false, nil
}

func portableFixtureTarget(target any, invocation ObjectInvocation) (Value, bool, error) {
	switch object := target.(type) {
	case *portableSqueezeBox:
		return object.invoke(invocation)
	case *portableEric:
		return object.invoke(invocation)
	case *portableTestLoadable:
		if invocation.Op == ObjectTypeCheck {
			class := resolvePortableClassName(invocation.Class)
			return Bool(class == "org.hick.tests.TestLoadable" || class == "java.lang.Object"), true, nil
		}
		return Null(), false, nil
	case *portableArrayTest1:
		return object.invoke(invocation)
	default:
		return Null(), false, nil
	}
}

func portableFixtureObjectClass(object any) (string, bool) {
	switch value := object.(type) {
	case *portableSqueezeBox:
		return "org.hick.blah.SqueezeBox", value != nil
	case *portableEric:
		return "com.eric.Eric", value != nil
	case *portableTestLoadable:
		return "org.hick.tests.TestLoadable", value != nil
	case *portableArrayTest1:
		return "sleep.ArrayTest1", value != nil
	default:
		return "", false
	}
}

func portableFixtureString(value Value) (string, bool) {
	if value.IsNull() {
		return "", true
	}
	switch value.Kind() {
	case KindString, KindInt, KindLong, KindDouble:
		return value.String(), true
	default:
		return "", false
	}
}

func portableFixtureBoolean(value Value) (bool, bool) {
	switch value.Kind() {
	case KindInt, KindLong, KindDouble:
		return value.Int64() != 0, true
	case KindObject:
		if primitive, ok := value.data.(*portableJavaPrimitive); ok && primitive != nil && primitive.className() == "java.lang.Boolean" {
			return primitive.sleepValue().Truth(), true
		}
	}
	return false, false
}

func portableFixtureInt(value Value) (int32, bool) {
	switch value.Kind() {
	case KindInt, KindLong, KindDouble:
		return value.Int32(), true
	case KindObject:
		if primitive, ok := value.data.(*portableJavaPrimitive); ok && primitive != nil && primitive.className() == "java.lang.Integer" {
			return primitive.sleepValue().Int32(), true
		}
	}
	return 0, false
}

func portableFixtureDouble(value Value) (float64, bool) {
	switch value.Kind() {
	case KindInt, KindLong, KindDouble:
		return value.Float64(), true
	case KindObject:
		if primitive, ok := value.data.(*portableJavaPrimitive); ok && primitive != nil && primitive.className() == "java.lang.Double" {
			return primitive.sleepValue().Float64(), true
		}
	}
	return 0, false
}

func portableFixtureConversionError(value Value, target string) error {
	return &portableFieldError{message: "unable to convert " + value.Describe() + " to a " + target}
}

func portableJavaDoubleText(value float64) string {
	text := strconv.FormatFloat(value, 'g', -1, 64)
	if !strings.ContainsAny(text, ".eE") {
		text += ".0"
	}
	return text
}
