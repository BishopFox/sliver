package kong

import (
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/bits"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var (
	mapperValueType       = reflect.TypeOf((*MapperValue)(nil)).Elem()
	boolMapperValueType   = reflect.TypeOf((*BoolMapperValue)(nil)).Elem()
	jsonUnmarshalerType   = reflect.TypeOf((*json.Unmarshaler)(nil)).Elem()
	textUnmarshalerType   = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
	binaryUnmarshalerType = reflect.TypeOf((*encoding.BinaryUnmarshaler)(nil)).Elem()
)

// DecodeContext is passed to a Mapper's Decode().
//
// It contains the Value being decoded into and the Scanner to parse from.
type DecodeContext struct {
	// Value being decoded into.
	Value *Value
	// Scan contains the input to scan into Target.
	Scan *Scanner
}

// WithScanner creates a clone of this context with a new Scanner.
func (r *DecodeContext) WithScanner(scan *Scanner) *DecodeContext {
	return &DecodeContext{
		Value: r.Value,
		Scan:  scan,
	}
}

// MapperValue may be implemented by fields in order to provide custom mapping.
// Mappers may additionally implement PlaceHolderProvider to provide custom placeholder text.
type MapperValue interface {
	Decode(ctx *DecodeContext) error
}

// BoolMapperValue may be implemented by fields in order to provide custom mappings for boolean values.
type BoolMapperValue interface {
	MapperValue
	IsBool() bool
}

type mapperValueAdapter struct {
	isBool bool
}

func (m *mapperValueAdapter) Decode(ctx *DecodeContext, target reflect.Value) error {
	if target.Type().Implements(mapperValueType) {
		return target.Interface().(MapperValue).Decode(ctx) // nolint
	}
	return target.Addr().Interface().(MapperValue).Decode(ctx) // nolint
}

func (m *mapperValueAdapter) IsBool() bool {
	return m.isBool
}

type textUnmarshalerAdapter struct{}

func (m *textUnmarshalerAdapter) Decode(ctx *DecodeContext, target reflect.Value) error {
	var value string
	err := ctx.Scan.PopValueInto("value", &value)
	if err != nil {
		return err
	}
	if target.Type().Implements(textUnmarshalerType) {
		return target.Interface().(encoding.TextUnmarshaler).UnmarshalText([]byte(value)) // nolint
	}
	return target.Addr().Interface().(encoding.TextUnmarshaler).UnmarshalText([]byte(value)) // nolint
}

type binaryUnmarshalerAdapter struct{}

func (m *binaryUnmarshalerAdapter) Decode(ctx *DecodeContext, target reflect.Value) error {
	var value string
	err := ctx.Scan.PopValueInto("value", &value)
	if err != nil {
		return err
	}
	if target.Type().Implements(binaryUnmarshalerType) {
		return target.Interface().(encoding.BinaryUnmarshaler).UnmarshalBinary([]byte(value)) // nolint
	}
	return target.Addr().Interface().(encoding.BinaryUnmarshaler).UnmarshalBinary([]byte(value)) // nolint
}

type jsonUnmarshalerAdapter struct{}

func (j *jsonUnmarshalerAdapter) Decode(ctx *DecodeContext, target reflect.Value) error {
	var value string
	err := ctx.Scan.PopValueInto("value", &value)
	if err != nil {
		return err
	}
	if target.Type().Implements(jsonUnmarshalerType) {
		return target.Interface().(json.Unmarshaler).UnmarshalJSON([]byte(value)) // nolint
	}
	return target.Addr().Interface().(json.Unmarshaler).UnmarshalJSON([]byte(value)) // nolint
}

// A Mapper represents how a field is mapped from command-line values to Go.
//
// Mappers can be associated with concrete fields via pointer, reflect.Type, reflect.Kind, or via a "type" tag.
//
// Additionally, if a type implements the MapperValue interface, it will be used.
type Mapper interface {
	// Decode ctx.Value with ctx.Scanner into target.
	Decode(ctx *DecodeContext, target reflect.Value) error
}

// VarsContributor can be implemented by a Mapper to contribute Vars during interpolation.
type VarsContributor interface {
	Vars(ctx *Value) Vars
}

// A BoolMapper is a Mapper to a value that is a boolean.
//
// This is used solely for formatting help.
type BoolMapper interface {
	Mapper
	IsBool() bool
}

// BoolMapperExt allows a Mapper to dynamically determine if a value is a boolean.
type BoolMapperExt interface {
	Mapper
	IsBoolFromValue(v reflect.Value) bool
}

// A MapperFunc is a single function that complies with the Mapper interface.
type MapperFunc func(ctx *DecodeContext, target reflect.Value) error

func (m MapperFunc) Decode(ctx *DecodeContext, target reflect.Value) error { // nolint: revive
	return m(ctx, target)
}

// A Registry contains a set of mappers and supporting lookup methods.
type Registry struct {
	names  map[string]Mapper
	types  map[reflect.Type]Mapper
	kinds  map[reflect.Kind]Mapper
	values map[reflect.Value]Mapper
}

// NewRegistry creates a new (empty) Registry.
func NewRegistry() *Registry {
	return &Registry{
		names:  map[string]Mapper{},
		types:  map[reflect.Type]Mapper{},
		kinds:  map[reflect.Kind]Mapper{},
		values: map[reflect.Value]Mapper{},
	}
}

// ForNamedValue finds a mapper for a value with a user-specified name.
//
// Will return nil if a mapper can not be determined.
func (r *Registry) ForNamedValue(name string, value reflect.Value) Mapper {
	if mapper, ok := r.names[name]; ok {
		return mapper
	}
	return r.ForValue(value)
}

// ForValue looks up the Mapper for a reflect.Value.
func (r *Registry) ForValue(value reflect.Value) Mapper {
	if mapper, ok := r.values[value]; ok {
		return mapper
	}
	return r.ForType(value.Type())
}

// ForNamedType finds a mapper for a type with a user-specified name.
//
// Will return nil if a mapper can not be determined.
func (r *Registry) ForNamedType(name string, typ reflect.Type) Mapper {
	if mapper, ok := r.names[name]; ok {
		return mapper
	}
	return r.ForType(typ)
}

// ForType finds a mapper from a type, by type, then kind.
//
// Will return nil if a mapper can not be determined.
func (r *Registry) ForType(typ reflect.Type) Mapper {
	// Check if the type implements MapperValue.
	for _, impl := range []reflect.Type{typ, reflect.PtrTo(typ)} {
		if impl.Implements(mapperValueType) {
			// FIXME: This should pass in the bool mapper.
			return &mapperValueAdapter{impl.Implements(boolMapperValueType)}
		}
	}
	// Next, try explicitly registered types.
	var mapper Mapper
	var ok bool
	if mapper, ok = r.types[typ]; ok {
		return mapper
	}
	// Next try stdlib unmarshaler interfaces.
	for _, impl := range []reflect.Type{typ, reflect.PtrTo(typ)} {
		switch {
		case impl.Implements(textUnmarshalerType):
			return &textUnmarshalerAdapter{}
		case impl.Implements(binaryUnmarshalerType):
			return &binaryUnmarshalerAdapter{}
		case impl.Implements(jsonUnmarshalerType):
			return &jsonUnmarshalerAdapter{}
		}
	}
	// Finally try registered kinds.
	if mapper, ok = r.kinds[typ.Kind()]; ok {
		return mapper
	}
	return nil
}

// RegisterKind registers a Mapper for a reflect.Kind.
func (r *Registry) RegisterKind(kind reflect.Kind, mapper Mapper) *Registry {
	r.kinds[kind] = mapper
	return r
}

// RegisterName registers a mapper to be used if the value mapper has a "type" tag matching name.
//
// eg.
//
//			Mapper string `kong:"type='colour'`
//	  	registry.RegisterName("colour", ...)
func (r *Registry) RegisterName(name string, mapper Mapper) *Registry {
	r.names[name] = mapper
	return r
}

// RegisterType registers a Mapper for a reflect.Type.
func (r *Registry) RegisterType(typ reflect.Type, mapper Mapper) *Registry {
	r.types[typ] = mapper
	return r
}

// RegisterValue registers a Mapper by pointer to the field value.
func (r *Registry) RegisterValue(ptr interface{}, mapper Mapper) *Registry {
	key := reflect.ValueOf(ptr)
	if key.Kind() != reflect.Ptr {
		panic("expected a pointer")
	}
	key = key.Elem()
	r.values[key] = mapper
	return r
}

// RegisterDefaults registers Mappers for all builtin supported Go types and some common stdlib types.
func (r *Registry) RegisterDefaults() *Registry {
	return r.RegisterKind(reflect.Int, intDecoder(bits.UintSize)).
		RegisterKind(reflect.Int8, intDecoder(8)).
		RegisterKind(reflect.Int16, intDecoder(16)).
		RegisterKind(reflect.Int32, intDecoder(32)).
		RegisterKind(reflect.Int64, intDecoder(64)).
		RegisterKind(reflect.Uint, uintDecoder(bits.UintSize)).
		RegisterKind(reflect.Uint8, uintDecoder(8)).
		RegisterKind(reflect.Uint16, uintDecoder(16)).
		RegisterKind(reflect.Uint32, uintDecoder(32)).
		RegisterKind(reflect.Uint64, uintDecoder(64)).
		RegisterKind(reflect.Float32, floatDecoder(32)).
		RegisterKind(reflect.Float64, floatDecoder(64)).
		RegisterKind(reflect.String, MapperFunc(func(ctx *DecodeContext, target reflect.Value) error {
			return ctx.Scan.PopValueInto("string", target.Addr().Interface())
		})).
		RegisterKind(reflect.Bool, boolMapper{}).
		RegisterKind(reflect.Slice, sliceDecoder(r)).
		RegisterKind(reflect.Map, mapDecoder(r)).
		RegisterType(reflect.TypeOf(time.Time{}), timeDecoder()).
		RegisterType(reflect.TypeOf(time.Duration(0)), durationDecoder()).
		RegisterType(reflect.TypeOf(&url.URL{}), urlMapper()).
		RegisterType(reflect.TypeOf(&os.File{}), fileMapper(r)).
		RegisterName("path", pathMapper(r)).
		RegisterName("existingfile", existingFileMapper(r)).
		RegisterName("existingdir", existingDirMapper(r)).
		RegisterName("counter", counterMapper()).
		RegisterName("filecontent", fileContentMapper(r)).
		RegisterKind(reflect.Ptr, ptrMapper{r})
}

type boolMapper struct{}

func (boolMapper) Decode(ctx *DecodeContext, target reflect.Value) error {
	if ctx.Scan.Peek().Type == FlagValueToken {
		token := ctx.Scan.Pop()
		switch v := token.Value.(type) {
		case string:
			v = strings.ToLower(v)
			switch v {
			case "true", "1", "yes":
				target.SetBool(true)

			case "false", "0", "no":
				target.SetBool(false)

			default:
				return fmt.Errorf("bool value must be true, 1, yes, false, 0 or no but got %q", v)
			}

		case bool:
			target.SetBool(v)

		default:
			return fmt.Errorf("expected bool but got %q (%T)", token.Value, token.Value)
		}
	} else {
		target.SetBool(true)
	}
	return nil
}
func (boolMapper) IsBool() bool { return true }

func durationDecoder() MapperFunc {
	return func(ctx *DecodeContext, target reflect.Value) error {
		t, err := ctx.Scan.PopValue("duration")
		if err != nil {
			return err
		}
		var d time.Duration
		switch v := t.Value.(type) {
		case string:
			d, err = time.ParseDuration(v)
			if err != nil {
				return fmt.Errorf("expected duration but got %q: %v", v, err)
			}
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
			d = reflect.ValueOf(v).Convert(reflect.TypeOf(time.Duration(0))).Interface().(time.Duration) // nolint: forcetypeassert
		default:
			return fmt.Errorf("expected duration but got %q", v)
		}
		target.Set(reflect.ValueOf(d))
		return nil
	}
}

func timeDecoder() MapperFunc {
	return func(ctx *DecodeContext, target reflect.Value) error {
		format := time.RFC3339
		if ctx.Value.Format != "" {
			format = ctx.Value.Format
		}
		var value string
		if err := ctx.Scan.PopValueInto("time", &value); err != nil {
			return err
		}
		t, err := time.Parse(format, value)
		if err != nil {
			return err
		}
		target.Set(reflect.ValueOf(t))
		return nil
	}
}

func intDecoder(bits int) MapperFunc { // nolint: dupl
	return func(ctx *DecodeContext, target reflect.Value) error {
		t, err := ctx.Scan.PopValue("int")
		if err != nil {
			return err
		}
		var sv string
		switch v := t.Value.(type) {
		case string:
			sv = v

		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			sv = fmt.Sprintf("%v", v)

		case float32, float64:
			sv = fmt.Sprintf("%0.f", v)

		default:
			return fmt.Errorf("expected an int but got %q (%T)", t, t.Value)
		}
		n, err := strconv.ParseInt(sv, 10, bits)
		if err != nil {
			return fmt.Errorf("expected a valid %d bit int but got %q", bits, sv)
		}
		target.SetInt(n)
		return nil
	}
}

func uintDecoder(bits int) MapperFunc { // nolint: dupl
	return func(ctx *DecodeContext, target reflect.Value) error {
		t, err := ctx.Scan.PopValue("uint")
		if err != nil {
			return err
		}
		var sv string
		switch v := t.Value.(type) {
		case string:
			sv = v

		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			sv = fmt.Sprintf("%v", v)

		case float32, float64:
			sv = fmt.Sprintf("%0.f", v)

		default:
			return fmt.Errorf("expected an int but got %q (%T)", t, t.Value)
		}
		n, err := strconv.ParseUint(sv, 10, bits)
		if err != nil {
			return fmt.Errorf("expected a valid %d bit uint but got %q", bits, sv)
		}
		target.SetUint(n)
		return nil
	}
}

func floatDecoder(bits int) MapperFunc {
	return func(ctx *DecodeContext, target reflect.Value) error {
		t, err := ctx.Scan.PopValue("float")
		if err != nil {
			return err
		}
		switch v := t.Value.(type) {
		case string:
			n, err := strconv.ParseFloat(v, bits)
			if err != nil {
				return fmt.Errorf("expected a float but got %q (%T)", t, t.Value)
			}
			target.SetFloat(n)

		case float32:
			target.SetFloat(float64(v))

		case float64:
			target.SetFloat(v)

		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			target.Set(reflect.ValueOf(v))

		default:
			return fmt.Errorf("expected an int but got %q (%T)", t, t.Value)
		}
		return nil
	}
}

func mapDecoder(r *Registry) MapperFunc {
	return func(ctx *DecodeContext, target reflect.Value) error {
		if target.IsNil() {
			target.Set(reflect.MakeMap(target.Type()))
		}
		el := target.Type()
		mapsep := ctx.Value.Tag.MapSep
		var childScanner *Scanner
		if ctx.Value.Flag != nil {
			t := ctx.Scan.Pop()
			// If decoding a flag, we need an value.
			if t.IsEOL() {
				return fmt.Errorf("missing value, expecting \"<key>=<value>%c...\"", mapsep)
			}
			switch v := t.Value.(type) {
			case string:
				childScanner = ScanAsType(t.Type, SplitEscaped(v, mapsep)...)

			case []map[string]interface{}:
				for _, m := range v {
					err := jsonTranscode(m, target.Addr().Interface())
					if err != nil {
						return err
					}
				}
				return nil

			case map[string]interface{}:
				return jsonTranscode(v, target.Addr().Interface())

			default:
				return fmt.Errorf("invalid map value %q (of type %T)", t, t.Value)
			}
		} else {
			tokens := ctx.Scan.PopWhile(func(t Token) bool { return t.IsValue() })
			childScanner = ScanFromTokens(tokens...)
		}
		for !childScanner.Peek().IsEOL() {
			var token string
			err := childScanner.PopValueInto("map", &token)
			if err != nil {
				return err
			}
			parts := strings.SplitN(token, "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("expected \"<key>=<value>\" but got %q", token)
			}
			key, value := parts[0], parts[1]

			keyTypeName, valueTypeName := "", ""
			if typ := ctx.Value.Tag.Type; typ != "" {
				parts := strings.Split(typ, ":")
				if len(parts) != 2 {
					return errors.New("type:\"\" on map field must be in the form \"[<keytype>]:[<valuetype>]\"")
				}
				keyTypeName, valueTypeName = parts[0], parts[1]
			}

			keyScanner := ScanAsType(FlagValueToken, key)
			keyDecoder := r.ForNamedType(keyTypeName, el.Key())
			keyValue := reflect.New(el.Key()).Elem()
			if err := keyDecoder.Decode(ctx.WithScanner(keyScanner), keyValue); err != nil {
				return fmt.Errorf("invalid map key %q", key)
			}

			valueScanner := ScanAsType(FlagValueToken, value)
			valueDecoder := r.ForNamedType(valueTypeName, el.Elem())
			valueValue := reflect.New(el.Elem()).Elem()
			if err := valueDecoder.Decode(ctx.WithScanner(valueScanner), valueValue); err != nil {
				return fmt.Errorf("invalid map value %q", value)
			}

			target.SetMapIndex(keyValue, valueValue)
		}
		return nil
	}
}

func sliceDecoder(r *Registry) MapperFunc {
	return func(ctx *DecodeContext, target reflect.Value) error {
		el := target.Type().Elem()
		sep := ctx.Value.Tag.Sep
		var childScanner *Scanner
		if ctx.Value.Flag != nil {
			t := ctx.Scan.Pop()
			// If decoding a flag, we need an value.
			if t.IsEOL() {
				return fmt.Errorf("missing value, expecting \"<arg>%c...\"", sep)
			}
			switch v := t.Value.(type) {
			case string:
				childScanner = ScanAsType(t.Type, SplitEscaped(v, sep)...)

			case []interface{}:
				return jsonTranscode(v, target.Addr().Interface())

			default:
				v = []interface{}{v}
				return jsonTranscode(v, target.Addr().Interface())
			}
		} else {
			tokens := ctx.Scan.PopWhile(func(t Token) bool { return t.IsValue() })
			childScanner = ScanFromTokens(tokens...)
		}
		childDecoder := r.ForNamedType(ctx.Value.Tag.Type, el)
		if childDecoder == nil {
			return fmt.Errorf("no mapper for element type of %s", target.Type())
		}
		for !childScanner.Peek().IsEOL() {
			childValue := reflect.New(el).Elem()
			err := childDecoder.Decode(ctx.WithScanner(childScanner), childValue)
			if err != nil {
				return err
			}
			target.Set(reflect.Append(target, childValue))
		}
		return nil
	}
}

func pathMapper(r *Registry) MapperFunc {
	return func(ctx *DecodeContext, target reflect.Value) error {
		if target.Kind() == reflect.Slice {
			return sliceDecoder(r)(ctx, target)
		}
		if target.Kind() != reflect.String {
			return fmt.Errorf("\"path\" type must be applied to a string not %s", target.Type())
		}
		var path string
		err := ctx.Scan.PopValueInto("file", &path)
		if err != nil {
			return err
		}
		if path != "-" {
			path = ExpandPath(path)
		}
		target.SetString(path)
		return nil
	}
}

func fileMapper(r *Registry) MapperFunc {
	return func(ctx *DecodeContext, target reflect.Value) error {
		if target.Kind() == reflect.Slice {
			return sliceDecoder(r)(ctx, target)
		}
		var path string
		err := ctx.Scan.PopValueInto("file", &path)
		if err != nil {
			return err
		}
		var file *os.File
		if path == "-" {
			file = os.Stdin
		} else {
			path = ExpandPath(path)
			file, err = os.Open(path) // nolint: gosec
			if err != nil {
				return err
			}
		}
		target.Set(reflect.ValueOf(file))
		return nil
	}
}

func existingFileMapper(r *Registry) MapperFunc {
	return func(ctx *DecodeContext, target reflect.Value) error {
		if target.Kind() == reflect.Slice {
			return sliceDecoder(r)(ctx, target)
		}
		if target.Kind() != reflect.String {
			return fmt.Errorf("\"existingfile\" type must be applied to a string not %s", target.Type())
		}
		var path string
		err := ctx.Scan.PopValueInto("file", &path)
		if err != nil {
			return err
		}

		if !ctx.Value.Active || ctx.Value.Set {
			// early return to avoid checking extra files that may not exist;
			// this hack only works because the value provided on the cli is
			// checked before the default value is checked (if default is set).
			return nil
		}

		if path != "-" {
			path = ExpandPath(path)
			stat, err := os.Stat(path)
			if err != nil {
				return err
			}
			if stat.IsDir() {
				return fmt.Errorf("%q exists but is a directory", path)
			}
		}
		target.SetString(path)
		return nil
	}
}

func existingDirMapper(r *Registry) MapperFunc {
	return func(ctx *DecodeContext, target reflect.Value) error {
		if target.Kind() == reflect.Slice {
			return sliceDecoder(r)(ctx, target)
		}
		if target.Kind() != reflect.String {
			return fmt.Errorf("\"existingdir\" must be applied to a string not %s", target.Type())
		}
		var path string
		err := ctx.Scan.PopValueInto("file", &path)
		if err != nil {
			return err
		}

		if !ctx.Value.Active || ctx.Value.Set {
			// early return to avoid checking extra dirs that may not exist;
			// this hack only works because the value provided on the cli is
			// checked before the default value is checked (if default is set).
			return nil
		}

		path = ExpandPath(path)
		stat, err := os.Stat(path)
		if err != nil {
			return err
		}
		if !stat.IsDir() {
			return fmt.Errorf("%q exists but is not a directory", path)
		}
		target.SetString(path)
		return nil
	}
}

func fileContentMapper(r *Registry) MapperFunc {
	return func(ctx *DecodeContext, target reflect.Value) error {
		if target.Kind() != reflect.Slice && target.Elem().Kind() != reflect.Uint8 {
			return fmt.Errorf("\"filecontent\" must be applied to []byte not %s", target.Type())
		}
		var path string
		err := ctx.Scan.PopValueInto("file", &path)
		if err != nil {
			return err
		}

		if !ctx.Value.Active || ctx.Value.Set {
			// early return to avoid checking extra dirs that may not exist;
			// this hack only works because the value provided on the cli is
			// checked before the default value is checked (if default is set).
			return nil
		}

		var data []byte
		if path != "-" {
			path = ExpandPath(path)
			data, err = ioutil.ReadFile(path) //nolint:gosec
		} else {
			data, err = ioutil.ReadAll(os.Stdin)
		}
		if err != nil {
			return err
		}
		target.SetBytes(data)
		return nil
	}
}

type ptrMapper struct {
	r *Registry
}

var _ BoolMapperExt = (*ptrMapper)(nil)

// IsBoolFromValue implements BoolMapperExt
func (p ptrMapper) IsBoolFromValue(target reflect.Value) bool {
	elem := reflect.New(target.Type().Elem()).Elem()
	nestedMapper := p.r.ForValue(elem)
	if nestedMapper == nil {
		return false
	}
	if bm, ok := nestedMapper.(BoolMapper); ok && bm.IsBool() {
		return true
	}
	if bm, ok := nestedMapper.(BoolMapperExt); ok && bm.IsBoolFromValue(target) {
		return true
	}
	return target.Kind() == reflect.Ptr && target.Type().Elem().Kind() == reflect.Bool
}

func (p ptrMapper) Decode(ctx *DecodeContext, target reflect.Value) error {
	elem := reflect.New(target.Type().Elem()).Elem()
	nestedMapper := p.r.ForValue(elem)
	if nestedMapper == nil {
		return fmt.Errorf("cannot find mapper for %v", target.Type().Elem().String())
	}
	err := nestedMapper.Decode(ctx, elem)
	if err != nil {
		return err
	}
	target.Set(elem.Addr())
	return nil
}

func counterMapper() MapperFunc {
	return func(ctx *DecodeContext, target reflect.Value) error {
		if ctx.Scan.Peek().Type == FlagValueToken {
			t, err := ctx.Scan.PopValue("counter")
			if err != nil {
				return err
			}
			switch v := t.Value.(type) {
			case string:
				n, err := strconv.ParseInt(v, 10, 64)
				if err != nil {
					return fmt.Errorf("expected a counter but got %q (%T)", t, t.Value)
				}
				target.SetInt(n)

			case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
				target.Set(reflect.ValueOf(v))

			default:
				return fmt.Errorf("expected a counter but got %q (%T)", t, t.Value)
			}
			return nil
		}

		switch target.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			target.SetInt(target.Int() + 1)

		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			target.SetUint(target.Uint() + 1)

		case reflect.Float32, reflect.Float64:
			target.SetFloat(target.Float() + 1)

		default:
			return fmt.Errorf("type:\"counter\" must be used with a numeric field")
		}
		return nil
	}
}

func urlMapper() MapperFunc {
	return func(ctx *DecodeContext, target reflect.Value) error {
		var urlStr string
		err := ctx.Scan.PopValueInto("url", &urlStr)
		if err != nil {
			return err
		}
		url, err := url.Parse(urlStr)
		if err != nil {
			return err
		}
		target.Set(reflect.ValueOf(url))
		return nil
	}
}

// SplitEscaped splits a string on a separator.
//
// It differs from strings.Split() in that the separator can exist in a field by escaping it with a \. eg.
//
//	SplitEscaped(`hello\,there,bob`, ',') == []string{"hello,there", "bob"}
func SplitEscaped(s string, sep rune) (out []string) {
	if sep == -1 {
		return []string{s}
	}
	escaped := false
	token := ""
	for i, ch := range s {
		switch {
		case escaped:
			if ch != sep {
				token += `\`
			}
			token += string(ch)
			escaped = false
		case ch == '\\' && i < len(s)-1:
			escaped = true
		case ch == sep && !escaped:
			out = append(out, token)
			token = ""
			escaped = false
		default:
			token += string(ch)
		}
	}
	if token != "" {
		out = append(out, token)
	}
	return
}

// JoinEscaped joins a slice of strings on sep, but also escapes any instances of sep in the fields with \. eg.
//
//	JoinEscaped([]string{"hello,there", "bob"}, ',') == `hello\,there,bob`
func JoinEscaped(s []string, sep rune) string {
	escaped := []string{}
	for _, e := range s {
		escaped = append(escaped, strings.ReplaceAll(e, string(sep), `\`+string(sep)))
	}
	return strings.Join(escaped, string(sep))
}

// NamedFileContentFlag is a flag value that loads a file's contents and filename into its value.
type NamedFileContentFlag struct {
	Filename string
	Contents []byte
}

func (f *NamedFileContentFlag) Decode(ctx *DecodeContext) error { // nolint: revive
	var filename string
	err := ctx.Scan.PopValueInto("filename", &filename)
	if err != nil {
		return err
	}
	// This allows unsetting of file content flags.
	if filename == "" {
		*f = NamedFileContentFlag{}
		return nil
	}
	filename = ExpandPath(filename)
	data, err := ioutil.ReadFile(filename) // nolint: gosec
	if err != nil {
		return fmt.Errorf("failed to open %q: %v", filename, err)
	}
	f.Contents = data
	f.Filename = filename
	return nil
}

// FileContentFlag is a flag value that loads a file's contents into its value.
type FileContentFlag []byte

func (f *FileContentFlag) Decode(ctx *DecodeContext) error { // nolint: revive
	var filename string
	err := ctx.Scan.PopValueInto("filename", &filename)
	if err != nil {
		return err
	}
	// This allows unsetting of file content flags.
	if filename == "" {
		*f = nil
		return nil
	}
	filename = ExpandPath(filename)
	data, err := ioutil.ReadFile(filename) // nolint: gosec
	if err != nil {
		return fmt.Errorf("failed to open %q: %v", filename, err)
	}
	*f = data
	return nil
}

func jsonTranscode(in, out interface{}) error {
	data, err := json.Marshal(in)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("%#v -> %T: %v", in, out, err)
	}
	return nil
}
