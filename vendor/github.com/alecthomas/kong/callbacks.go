package kong

import (
	"fmt"
	"reflect"
	"strings"
)

type bindings map[reflect.Type]func() (reflect.Value, error)

func (b bindings) String() string {
	out := []string{}
	for k := range b {
		out = append(out, k.String())
	}
	return "bindings{" + strings.Join(out, ", ") + "}"
}

func (b bindings) add(values ...interface{}) bindings {
	for _, v := range values {
		v := v
		b[reflect.TypeOf(v)] = func() (reflect.Value, error) { return reflect.ValueOf(v), nil }
	}
	return b
}

func (b bindings) addTo(impl, iface interface{}) {
	valueOf := reflect.ValueOf(impl)
	b[reflect.TypeOf(iface).Elem()] = func() (reflect.Value, error) { return valueOf, nil }
}

func (b bindings) addProvider(provider interface{}) error {
	pv := reflect.ValueOf(provider)
	t := pv.Type()
	if t.Kind() != reflect.Func || t.NumIn() != 0 || t.NumOut() != 2 || t.Out(1) != reflect.TypeOf((*error)(nil)).Elem() {
		return fmt.Errorf("%T must be a function with the signature func()(T, error)", provider)
	}
	rt := pv.Type().Out(0)
	b[rt] = func() (reflect.Value, error) {
		out := pv.Call(nil)
		errv := out[1]
		var err error
		if !errv.IsNil() {
			err = errv.Interface().(error) // nolint
		}
		return out[0], err
	}
	return nil
}

// Clone and add values.
func (b bindings) clone() bindings {
	out := make(bindings, len(b))
	for k, v := range b {
		out[k] = v
	}
	return out
}

func (b bindings) merge(other bindings) bindings {
	for k, v := range other {
		b[k] = v
	}
	return b
}

func getMethod(value reflect.Value, name string) reflect.Value {
	method := value.MethodByName(name)
	if !method.IsValid() {
		if value.CanAddr() {
			method = value.Addr().MethodByName(name)
		}
	}
	return method
}

func callFunction(f reflect.Value, bindings bindings) error {
	if f.Kind() != reflect.Func {
		return fmt.Errorf("expected function, got %s", f.Type())
	}
	in := []reflect.Value{}
	t := f.Type()
	if t.NumOut() != 1 || !t.Out(0).Implements(callbackReturnSignature) {
		return fmt.Errorf("return value of %s must implement \"error\"", t)
	}
	for i := 0; i < t.NumIn(); i++ {
		pt := t.In(i)
		if argf, ok := bindings[pt]; ok {
			argv, err := argf()
			if err != nil {
				return err
			}
			in = append(in, argv)
		} else {
			return fmt.Errorf("couldn't find binding of type %s for parameter %d of %s(), use kong.Bind(%s)", pt, i, t, pt)
		}
	}
	out := f.Call(in)
	if out[0].IsNil() {
		return nil
	}
	return out[0].Interface().(error) // nolint
}

func callAnyFunction(f reflect.Value, bindings bindings) (out []any, err error) {
	if f.Kind() != reflect.Func {
		return nil, fmt.Errorf("expected function, got %s", f.Type())
	}
	in := []reflect.Value{}
	t := f.Type()
	for i := 0; i < t.NumIn(); i++ {
		pt := t.In(i)
		if argf, ok := bindings[pt]; ok {
			argv, err := argf()
			if err != nil {
				return nil, err
			}
			in = append(in, argv)
		} else {
			return nil, fmt.Errorf("couldn't find binding of type %s for parameter %d of %s(), use kong.Bind(%s)", pt, i, t, pt)
		}
	}
	outv := f.Call(in)
	out = make([]any, len(outv))
	for i, v := range outv {
		out[i] = v.Interface()
	}
	return out, nil
}
