// Copyright 2017 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package firestore

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	pb "cloud.google.com/go/firestore/apiv1/firestorepb"
	"cloud.google.com/go/internal/fields"
)

func setFromProtoValue(dest interface{}, vprotoSrc *pb.Value, c *Client) error {
	destV := reflect.ValueOf(dest)
	if destV.Kind() != reflect.Ptr || destV.IsNil() {
		return errors.New("firestore: nil or not a pointer")
	}
	return setReflectFromProtoValue(destV.Elem(), vprotoSrc, c)
}

// setReflectFromProtoValue sets vDest from a Firestore Value.
// vDest must be a settable value.
func setReflectFromProtoValue(vDest reflect.Value, vprotoSrc *pb.Value, c *Client) error {
	typeErr := func() error {
		return fmt.Errorf("firestore: cannot set type %s to %s", vDest.Type(), typeString(vprotoSrc))
	}

	valTypeSrc := vprotoSrc.ValueType
	// A Null value sets anything nullable to nil, and has no effect
	// on anything else.
	if _, ok := valTypeSrc.(*pb.Value_NullValue); ok {
		switch vDest.Kind() {
		case reflect.Interface, reflect.Ptr, reflect.Map, reflect.Slice:
			vDest.Set(reflect.Zero(vDest.Type()))
		}
		return nil
	}

	// Handle special types first.
	switch vDest.Type() {
	case typeOfByteSlice:
		x, ok := valTypeSrc.(*pb.Value_BytesValue)
		if !ok {
			return typeErr()
		}
		vDest.SetBytes(x.BytesValue)
		return nil

	case typeOfGoTime:
		x, ok := valTypeSrc.(*pb.Value_TimestampValue)
		if !ok {
			return typeErr()
		}
		if err := x.TimestampValue.CheckValid(); err != nil {
			return err
		}
		vDest.Set(reflect.ValueOf(x.TimestampValue.AsTime()))
		return nil

	case typeOfProtoTimestamp:
		x, ok := valTypeSrc.(*pb.Value_TimestampValue)
		if !ok {
			return typeErr()
		}
		vDest.Set(reflect.ValueOf(x.TimestampValue))
		return nil

	case typeOfLatLng:
		x, ok := valTypeSrc.(*pb.Value_GeoPointValue)
		if !ok {
			return typeErr()
		}
		vDest.Set(reflect.ValueOf(x.GeoPointValue))
		return nil

	case typeOfDocumentRef:
		x, ok := valTypeSrc.(*pb.Value_ReferenceValue)
		if !ok {
			return typeErr()
		}
		dr, err := pathToDoc(x.ReferenceValue, c)
		if err != nil {
			return err
		}
		vDest.Set(reflect.ValueOf(dr))
		return nil

	case typeOfVector32:
		val, err := vector32FromProtoValue(vprotoSrc)
		if err != nil {
			return err
		}
		vDest.Set(reflect.ValueOf(val))
		return nil
	case typeOfVector64:
		val, err := vector64FromProtoValue(vprotoSrc)
		if err != nil {
			return err
		}
		vDest.Set(reflect.ValueOf(val))
		return nil
	}

	switch vDest.Kind() {
	case reflect.Bool:
		x, ok := valTypeSrc.(*pb.Value_BooleanValue)
		if !ok {
			return typeErr()
		}
		vDest.SetBool(x.BooleanValue)

	case reflect.String:
		x, ok := valTypeSrc.(*pb.Value_StringValue)
		if !ok {
			return typeErr()
		}
		vDest.SetString(x.StringValue)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var i int64
		switch x := valTypeSrc.(type) {
		case *pb.Value_IntegerValue:
			i = x.IntegerValue
		case *pb.Value_DoubleValue:
			f := x.DoubleValue
			i = int64(f)
			if float64(i) != f {
				return fmt.Errorf("firestore: float %f does not fit into %s", f, vDest.Type())
			}
		default:
			return typeErr()
		}
		if vDest.OverflowInt(i) {
			return overflowErr(vDest, i)
		}
		vDest.SetInt(i)

	case reflect.Uint8, reflect.Uint16, reflect.Uint32:
		var u uint64
		switch x := valTypeSrc.(type) {
		case *pb.Value_IntegerValue:
			u = uint64(x.IntegerValue)
		case *pb.Value_DoubleValue:
			f := x.DoubleValue
			u = uint64(f)
			if float64(u) != f {
				return fmt.Errorf("firestore: float %f does not fit into %s", f, vDest.Type())
			}
		default:
			return typeErr()
		}
		if vDest.OverflowUint(u) {
			return overflowErr(vDest, u)
		}
		vDest.SetUint(u)

	case reflect.Float32, reflect.Float64:
		var f float64
		switch x := valTypeSrc.(type) {
		case *pb.Value_DoubleValue:
			f = x.DoubleValue
		case *pb.Value_IntegerValue:
			f = float64(x.IntegerValue)
			if int64(f) != x.IntegerValue {
				return overflowErr(vDest, x.IntegerValue)
			}
		default:
			return typeErr()
		}
		if vDest.OverflowFloat(f) {
			return overflowErr(vDest, f)
		}
		vDest.SetFloat(f)

	case reflect.Slice:
		x, ok := valTypeSrc.(*pb.Value_ArrayValue)
		if !ok {
			return typeErr()
		}
		vals := x.ArrayValue.Values
		vlen := vDest.Len()
		xlen := len(vals)
		// Make a slice of the right size, avoiding allocation if possible.
		switch {
		case vlen < xlen:
			vDest.Set(reflect.MakeSlice(vDest.Type(), xlen, xlen))
		case vlen > xlen:
			vDest.SetLen(xlen)
		}
		return populateRepeated(vDest, vals, xlen, c)

	case reflect.Array:
		x, ok := valTypeSrc.(*pb.Value_ArrayValue)
		if !ok {
			return typeErr()
		}
		vals := x.ArrayValue.Values
		xlen := len(vals)
		vlen := vDest.Len()
		minlen := vlen
		// Set extra elements to their zero value.
		if vlen > xlen {
			z := reflect.Zero(vDest.Type().Elem())
			for i := xlen; i < vlen; i++ {
				vDest.Index(i).Set(z)
			}
			minlen = xlen
		}
		return populateRepeated(vDest, vals, minlen, c)

	case reflect.Map:
		x, ok := valTypeSrc.(*pb.Value_MapValue)
		if !ok {
			return typeErr()
		}
		return populateMap(vDest, x.MapValue.Fields, c)

	case reflect.Ptr:
		// If the pointer is nil, set it to a zero value.
		if vDest.IsNil() {
			vDest.Set(reflect.New(vDest.Type().Elem()))
		}
		return setReflectFromProtoValue(vDest.Elem(), vprotoSrc, c)

	case reflect.Struct:
		x, ok := valTypeSrc.(*pb.Value_MapValue)
		if !ok {
			return typeErr()
		}
		return populateStruct(vDest, x.MapValue.Fields, c)

	case reflect.Interface:
		if vDest.NumMethod() == 0 { // empty interface
			// If v holds a pointer, set the pointer.
			if !vDest.IsNil() && vDest.Elem().Kind() == reflect.Ptr {
				return setReflectFromProtoValue(vDest.Elem(), vprotoSrc, c)
			}
			// Otherwise, create a fresh value.
			x, err := createFromProtoValue(vprotoSrc, c)
			if err != nil {
				return err
			}
			vDest.Set(reflect.ValueOf(x))
			return nil
		}
		// Any other kind of interface is an error.
		fallthrough

	default:
		return fmt.Errorf("firestore: cannot set type %s", vDest.Type())
	}
	return nil
}

// populateRepeated sets the first n elements of vr, which must be a slice or
// array, to the corresponding elements of vals.
func populateRepeated(vr reflect.Value, vals []*pb.Value, n int, c *Client) error {
	for i := 0; i < n; i++ {
		if err := setReflectFromProtoValue(vr.Index(i), vals[i], c); err != nil {
			return err
		}
	}
	return nil
}

// populateMap sets the elements of destValueMap, which must be a map, from the
// corresponding elements of srcPropMap.
//
// Since a map value is not settable, this function always creates a new
// element for each corresponding map key. Existing values of destValueMap are
// overwritten. This happens even if the map value is something like a pointer
// to a struct, where we could in theory populate the existing struct value
// instead of discarding it. This behavior matches encoding/json.
func populateMap(destValueMap reflect.Value, srcPropMap map[string]*pb.Value, c *Client) error {
	destValueMapType := destValueMap.Type()
	if destValueMapType.Key().Kind() != reflect.String {
		return errors.New("firestore: map key type is not string")
	}
	if destValueMap.IsNil() {
		destValueMap.Set(reflect.MakeMap(destValueMapType))
	}
	et := destValueMapType.Elem()
	for srcKey, srcVProto := range srcPropMap {
		el := reflect.New(et).Elem()
		if err := setReflectFromProtoValue(el, srcVProto, c); err != nil {
			return err
		}
		keyToSet := reflect.ValueOf(srcKey)
		if reflect.ValueOf(srcKey).CanConvert(destValueMapType.Key()) {
			keyToSet = reflect.ValueOf(srcKey).Convert(destValueMapType.Key())
		}
		destValueMap.SetMapIndex(keyToSet, el)
	}
	return nil
}

// createMapFromValueMap creates a fresh map and populates it with pm.
func createMapFromValueMap(pm map[string]*pb.Value, c *Client) (map[string]interface{}, error) {
	m := map[string]interface{}{}
	for k, pv := range pm {
		v, err := createFromProtoValue(pv, c)
		if err != nil {
			return nil, err
		}
		m[k] = v
	}
	return m, nil
}

// populateStruct sets the fields of vs, which must be a struct, from
// the matching elements of pm.
func populateStruct(vs reflect.Value, pm map[string]*pb.Value, c *Client) error {
	fs, err := fieldCache.Fields(vs.Type())
	if err != nil {
		return err
	}

	type match struct {
		vproto *pb.Value
		f      *fields.Field
	}
	// Find best field matches
	matched := make(map[string]match)
	for k, vproto := range pm {
		f := fs.Match(k)
		if f == nil {
			continue
		}
		if _, ok := matched[f.Name]; ok {
			// If multiple case insensitive fields match, the exact match
			// should win.
			if f.Name == k {
				matched[k] = match{vproto: vproto, f: f}
			}
		} else {
			matched[f.Name] = match{vproto: vproto, f: f}
		}
	}

	// Reflect values
	for _, v := range matched {
		f := v.f
		vproto := v.vproto

		if err := setReflectFromProtoValue(vs.FieldByIndex(f.Index), vproto, c); err != nil {
			return fmt.Errorf("%s.%s: %w", vs.Type(), f.Name, err)
		}
	}
	return nil
}

func createFromProtoValue(vproto *pb.Value, c *Client) (interface{}, error) {
	switch v := vproto.ValueType.(type) {
	case *pb.Value_NullValue:
		return nil, nil
	case *pb.Value_BooleanValue:
		return v.BooleanValue, nil
	case *pb.Value_IntegerValue:
		return v.IntegerValue, nil
	case *pb.Value_DoubleValue:
		return v.DoubleValue, nil
	case *pb.Value_TimestampValue:
		if err := v.TimestampValue.CheckValid(); err != nil {
			return nil, err
		}
		return v.TimestampValue.AsTime(), nil
	case *pb.Value_StringValue:
		return v.StringValue, nil
	case *pb.Value_BytesValue:
		return v.BytesValue, nil
	case *pb.Value_ReferenceValue:
		return pathToDoc(v.ReferenceValue, c)
	case *pb.Value_GeoPointValue:
		return v.GeoPointValue, nil
	case *pb.Value_ArrayValue:
		vals := v.ArrayValue.Values
		ret := make([]interface{}, len(vals))
		for i, v := range vals {
			r, err := createFromProtoValue(v, c)
			if err != nil {
				return nil, err
			}
			ret[i] = r
		}
		return ret, nil

	case *pb.Value_MapValue:
		fields := v.MapValue.Fields
		ret := make(map[string]interface{}, len(fields))
		for k, v := range fields {
			r, err := createFromProtoValue(v, c)
			if err != nil {
				return nil, err
			}
			ret[k] = r
		}

		typeVal, ok := ret[typeKey]
		if !ok || typeVal != typeValVector {
			// Map is not a vector. Return the map
			return ret, nil
		}

		// Special handling for vector
		return vector64FromProtoValue(vproto)
	default:
		return nil, fmt.Errorf("firestore: unknown value type %T", v)
	}
}

// Convert a document path to a DocumentRef.
func pathToDoc(docPath string, c *Client) (*DocumentRef, error) {
	projID, dbID, docIDs, err := parseDocumentPath(docPath)
	if err != nil {
		return nil, err
	}
	parentResourceName := fmt.Sprintf("projects/%s/databases/%s", projID, dbID)
	_, doc := c.idsToRef(docIDs, parentResourceName)
	return doc, nil
}

// A document path should be of the form "projects/P/databases/D/documents/coll1/doc1/coll2/doc2/...".
func parseDocumentPath(path string) (projectID, databaseID string, docPath []string, err error) {
	parts := strings.Split(path, "/")
	if len(parts) < 6 || parts[0] != "projects" || parts[2] != "databases" || parts[4] != "documents" {
		return "", "", nil, fmt.Errorf("firestore: malformed document path %q", path)
	}
	docp := parts[5:]
	if len(docp)%2 != 0 {
		return "", "", nil, fmt.Errorf("firestore: path %q refers to collection, not document", path)
	}
	return parts[1], parts[3], docp, nil
}

func typeString(vproto *pb.Value) string {
	switch vproto.ValueType.(type) {
	case *pb.Value_NullValue:
		return "null"
	case *pb.Value_BooleanValue:
		return "bool"
	case *pb.Value_IntegerValue:
		return "int"
	case *pb.Value_DoubleValue:
		return "float"
	case *pb.Value_TimestampValue:
		return "timestamp"
	case *pb.Value_StringValue:
		return "string"
	case *pb.Value_BytesValue:
		return "bytes"
	case *pb.Value_ReferenceValue:
		return "reference"
	case *pb.Value_GeoPointValue:
		return "GeoPoint"
	case *pb.Value_MapValue:
		return "map"
	case *pb.Value_ArrayValue:
		return "array"
	default:
		return "<unknown Value type>"
	}
}

func overflowErr(v reflect.Value, x interface{}) error {
	return fmt.Errorf("firestore: value %v overflows type %s", x, v.Type())
}
