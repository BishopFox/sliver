package opfor

import (
	"fmt"
	"strings"
)

func (box *portableSqueezeBox) doStuff(invocation ObjectInvocation) (Value, bool, error) {
	argument := invocation.Arg(0)
	rows := make([][]Value, 0)
	if javaArray := portableFixtureJavaArray(argument); javaArray != nil {
		if javaArray.typeInfo.descriptor != "D" || len(javaArray.dimensions) != 2 {
			return portableObjectWarning(invocation, "incorrect dimensions for conversion to class [[D"), true, nil
		}
		width := javaArray.dimensions[1]
		for row := 0; row < javaArray.dimensions[0]; row++ {
			start := row * width
			rows = append(rows, append([]Value(nil), javaArray.values[start:start+width]...))
		}
	} else if outer, ok := argument.Array(); ok && outer != nil {
		for _, value := range outer.Values() {
			inner, nested := value.Array()
			if !nested || inner == nil {
				return portableObjectWarning(invocation, "incorrect dimensions for conversion to class [[D"), true, nil
			}
			rows = append(rows, inner.Values())
		}
	} else {
		return portableObjectWarning(invocation, "incorrect dimensions for conversion to class [[D"), true, nil
	}

	if box != nil && box.runtime != nil && box.runtime.stdout != nil {
		if _, err := fmt.Fprintln(box.runtime.stdout, "Printing the table:"); err != nil {
			return Null(), true, err
		}
		for _, row := range rows {
			for _, value := range row {
				if _, err := fmt.Fprintf(box.runtime.stdout, "%s; ", portableJavaDoubleText(value.Float64())); err != nil {
					return Null(), true, err
				}
			}
			if _, err := fmt.Fprintln(box.runtime.stdout); err != nil {
				return Null(), true, err
			}
		}
	}
	return Null(), true, nil
}

// portableArrayTest1 is a handwritten representation of the pinned BSD test
// fixture used to specify Sleep's Java array-overload coercion. The archive is
// inspected before this type is authorized; its bytecode is never executed.
type portableArrayTest1 struct{}

func (*portableArrayTest1) String() string { return "sleep.ArrayTest1" }

func (*portableArrayTest1) invoke(invocation ObjectInvocation) (Value, bool, error) {
	if invocation.Op == ObjectTypeCheck {
		class := resolvePortableClassName(invocation.Class)
		return Bool(class == "sleep.ArrayTest1" || class == "java.lang.Object"), true, nil
	}
	if invocation.Op != ObjectInvoke || len(invocation.Arguments) != 1 {
		return Null(), false, nil
	}
	argument := invocation.Arg(0)
	switch invocation.Message {
	case "foo":
		return portableArrayTestFoo(invocation, argument), true, nil
	case "bar":
		value, err := portableArrayTestBar(invocation, argument)
		return value, true, err
	case "car":
		return portableArrayTestCar(invocation, argument), true, nil
	case "mar":
		return portableArrayTestMar(invocation, argument), true, nil
	case "tar":
		if _, ok := argument.Array(); ok {
			portableFixturePrintln(invocation, "List a: class java.util.LinkedList")
			return Null(), true, nil
		}
	}
	return Null(), false, nil
}

func portableArrayTestFoo(invocation ObjectInvocation, argument Value) Value {
	if javaArray := portableFixtureJavaArray(argument); javaArray != nil {
		portableFixturePrintln(invocation, portableFixtureArrayLabel(javaArray)+" a")
		return Null()
	}
	array, ok := argument.Array()
	if !ok || array == nil {
		return portableNoMatchingMethod(invocation, "sleep.ArrayTest1")
	}
	values := array.Values()
	label := "Object[]"
	if len(values) != 0 {
		switch values[0].Kind() {
		case KindString:
			label = "String[]"
		case KindInt:
			label = "int[]"
		case KindLong:
			label = "long[]"
		case KindDouble:
			label = "double[]"
		}
	}
	portableFixturePrintln(invocation, label+" a")
	return Null()
}

func portableArrayTestBar(invocation ObjectInvocation, argument Value) (Value, error) {
	if javaArray := portableFixtureJavaArray(argument); javaArray != nil {
		if strings.HasPrefix(javaArray.typeInfo.descriptor, "L") {
			if err := portableArrayTestPrintObjectArray(invocation, javaArray.className(), javaArray.values); err != nil {
				return Null(), err
			}
		} else {
			portableFixturePrintln(invocation, "Object a: class "+javaArray.className())
		}
		return Null(), nil
	}
	array, ok := argument.Array()
	if !ok || array == nil {
		return portableNoMatchingMethod(invocation, "sleep.ArrayTest1"), nil
	}
	values := array.Values()
	if len(values) == 0 {
		portableFixturePrintln(invocation, "Object a: class java.util.LinkedList")
		return Null(), nil
	}
	if boxed, ok := portableFixturePrimitive(values[0]); ok {
		converted := make([]Value, len(values))
		for index, value := range values {
			candidate, candidateOK := portableFixturePrimitive(value)
			if !candidateOK || candidate.class != boxed.class {
				// Sleep's filter assignment may unwrap the later Float cells back
				// to ordinary scalars. Java's Object[] coercion uses the first
				// explicit Float as the component hint and boxes those remaining
				// values, including the reference runtime's numeric conversion of
				// a non-number string to 0.0. Long and other wrappers do not get
				// this widening path.
				if boxed.class == "java.lang.Float" {
					coerced, err := coercePortableJavaValue(value, "float")
					if err == nil {
						converted[index] = ObjectValue(&portableJavaPrimitive{class: boxed.class, value: coerced})
						continue
					}
				}
				return portableObjectWarning(invocation, fmt.Sprintf("%s at %d is not compatible with %s", value.Describe(), index, boxed.class)), nil
			}
			converted[index] = value
		}
		if err := portableArrayTestPrintObjectArray(invocation, "[L"+boxed.class+";", converted); err != nil {
			return Null(), err
		}
		return Null(), nil
	}
	if values[0].Kind() == KindString {
		converted := make([]Value, len(values))
		for index, value := range values {
			converted[index] = String(value.String())
		}
		if err := portableArrayTestPrintObjectArray(invocation, "[Ljava.lang.String;", converted); err != nil {
			return Null(), err
		}
		return Null(), nil
	}
	if actual, ok := portableObjectClass(values[0]); ok && values[0].Kind() == KindObject {
		for index, value := range values[1:] {
			candidate, candidateOK := portableObjectClass(value)
			if !candidateOK || !portableJavaAssignable(candidate, actual) {
				return portableObjectWarning(invocation, fmt.Sprintf("%s at %d is not compatible with %s", value.Describe(), index+1, actual)), nil
			}
		}
	}
	portableFixturePrintln(invocation, "Object a: class java.util.LinkedList")
	return Null(), nil
}

func portableArrayTestCar(invocation ObjectInvocation, argument Value) Value {
	if array, ok := argument.Array(); ok && array != nil {
		values := array.Values()
		if len(values) == 0 || values[0].Kind() == KindInt {
			portableFixturePrintln(invocation, "int[] a")
			return Null()
		}
	}
	portableFixturePrintln(invocation, "Object a")
	return Null()
}

func portableArrayTestMar(invocation ObjectInvocation, argument Value) Value {
	if javaArray := portableFixtureJavaArray(argument); javaArray != nil {
		if javaArray.typeInfo.descriptor == "I" {
			portableFixturePrintln(invocation, "int[] a")
			return Null()
		}
		return portableNoMatchingMethod(invocation, "sleep.ArrayTest1")
	}
	if array, ok := argument.Array(); ok && array != nil {
		values := array.Values()
		if len(values) == 0 || values[0].Kind() == KindInt {
			portableFixturePrintln(invocation, "int[] a")
		} else {
			portableFixturePrintln(invocation, "Collection a")
		}
		return Null()
	}
	return portableNoMatchingMethod(invocation, "sleep.ArrayTest1")
}

func portableArrayTestPrintObjectArray(invocation ObjectInvocation, class string, values []Value) error {
	if err := portableFixturePrintln(invocation, "Object[] a: class "+class); err != nil {
		return err
	}
	for index, value := range values {
		name, ok := portableObjectClass(value)
		if !ok {
			name = "java.lang.Object"
		}
		if err := portableFixturePrintln(invocation, fmt.Sprintf("a[%d] - %s - class %s", index, value.String(), name)); err != nil {
			return err
		}
	}
	return nil
}

func portableFixtureArrayLabel(array *portableJavaArray) string {
	if array == nil {
		return "Object[]"
	}
	switch array.typeInfo.descriptor {
	case "Z":
		return "boolean[]"
	case "I":
		return "int[]"
	case "J":
		return "long[]"
	case "F":
		return "float[]"
	case "D":
		return "double[]"
	default:
		if array.typeInfo.name == "java.lang.String" {
			return "String[]"
		}
		return "Object[]"
	}
}

func portableFixtureJavaArray(value Value) *portableJavaArray {
	object, ok := value.Object()
	if !ok {
		return nil
	}
	array, _ := object.(*portableJavaArray)
	return array
}

func portableFixturePrimitive(value Value) (*portableJavaPrimitive, bool) {
	object, ok := value.Object()
	if !ok {
		return nil, false
	}
	primitive, ok := object.(*portableJavaPrimitive)
	return primitive, ok && primitive != nil
}

func portableFixturePrintln(invocation ObjectInvocation, line string) error {
	if invocation.Runtime == nil || invocation.Runtime.stdout == nil {
		return nil
	}
	_, _ = fmt.Fprintln(invocation.Runtime.stdout, line)
	// These compatibility fixtures deliberately ignore ordinary writer errors,
	// matching their historical println behavior. Return only the trusted local
	// family latch so a quota failure stops further formatting/traversal.
	return invocation.Runtime.outputLimitError()
}
