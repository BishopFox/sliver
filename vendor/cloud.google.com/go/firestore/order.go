// Copyright 2018 Google LLC
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
	"bytes"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	pb "cloud.google.com/go/firestore/apiv1/firestorepb"
	tspb "google.golang.org/protobuf/types/known/timestamppb"
)

const (
	typeOrderNull = iota
	typeOrderMinKey
	typeOrderBoolean
	typeOrderNumber
	typeOrderTimestamp
	typeOrderBSONTimestamp
	typeOrderString
	typeOrderBlob
	typeOrderBSONBinary
	typeOrderRef
	typeOrderBSONObjectID
	typeOrderGeoPoint
	typeOrderRegex
	typeOrderArray
	typeOrderVector
	typeOrderObject
	typeOrderMaxKey
)

// Returns a negative number, zero, or a positive number depending on whether a is
// less than, equal to, or greater than b according to Firestore's ordering of
// values.
func compareValues(a, b *pb.Value) int {
	ta := typeOrder(a)
	tb := typeOrder(b)
	if ta != tb {
		return compareInt64s(int64(ta), int64(tb))
	}
	if ta == typeOrderNumber {
		return compareNumbers(extractNumber(a), extractNumber(b))
	}
	if ta == typeOrderBSONTimestamp {
		return compareBSONTimestamps(a, b)
	}
	if ta == typeOrderRegex {
		return compareBSONRegexes(a, b)
	}
	switch a := a.ValueType.(type) {
	case *pb.Value_NullValue:
		return 0 // nulls are equal

	case *pb.Value_BooleanValue:
		av := a.BooleanValue
		bv := b.GetBooleanValue()
		switch {
		case av && !bv:
			return 1
		case bv && !av:
			return -1
		default:
			return 0
		}

	case *pb.Value_IntegerValue:
		return compareNumbers(float64(a.IntegerValue), toFloat(b))

	case *pb.Value_DoubleValue:
		return compareNumbers(a.DoubleValue, toFloat(b))

	case *pb.Value_TimestampValue:
		return compareTimestamps(a.TimestampValue, b.GetTimestampValue())

	case *pb.Value_StringValue:
		return strings.Compare(a.StringValue, b.GetStringValue())

	case *pb.Value_BytesValue:
		return bytes.Compare(a.BytesValue, b.GetBytesValue())

	case *pb.Value_ReferenceValue:
		return compareReferences(a.ReferenceValue, b.GetReferenceValue())

	case *pb.Value_GeoPointValue:
		ag := a.GeoPointValue
		bg := b.GetGeoPointValue()
		if ag.Latitude != bg.Latitude {
			return compareFloat64s(ag.Latitude, bg.Latitude)
		}
		return compareFloat64s(ag.Longitude, bg.Longitude)

	case *pb.Value_ArrayValue:
		return compareArrays(a.ArrayValue.Values, b.GetArrayValue().Values)

	case *pb.Value_MapValue:
		return compareMaps(a.MapValue.Fields, b.GetMapValue().Fields)

	default:
		panic(fmt.Sprintf("bad value type: %v", a))
	}
}

// Treats NaN as less than any non-NaN.
func compareNumbers(a, b float64) int {
	switch {
	case math.IsNaN(a):
		if math.IsNaN(b) {
			return 0
		}
		return -1
	case math.IsNaN(b):
		return 1
	default:
		return compareFloat64s(a, b)
	}
}

// Return v as a float64, assuming it's an Integer or Double.
func toFloat(v *pb.Value) float64 {
	if x, ok := v.ValueType.(*pb.Value_IntegerValue); ok {
		return float64(x.IntegerValue)
	}
	return v.GetDoubleValue()
}

func compareTimestamps(a, b *tspb.Timestamp) int {
	if c := compareInt64s(a.Seconds, b.Seconds); c != 0 {
		return c
	}
	return compareInt64s(int64(a.Nanos), int64(b.Nanos))
}

func compareReferences(a, b string) int {
	// Compare path components lexicographically.
	pa := strings.Split(a, "/")
	pb := strings.Split(b, "/")
	return compareSequences(len(pa), len(pb), func(i int) int {
		return strings.Compare(pa[i], pb[i])
	})
}

func compareArrays(a, b []*pb.Value) int {
	return compareSequences(len(a), len(b), func(i int) int {
		return compareValues(a[i], b[i])
	})
}

func compareMaps(a, b map[string]*pb.Value) int {
	sortedKeys := func(m map[string]*pb.Value) []string {
		var ks []string
		for k := range m {
			ks = append(ks, k)
		}
		sort.Strings(ks)
		return ks
	}

	aks := sortedKeys(a)
	bks := sortedKeys(b)
	return compareSequences(len(aks), len(bks), func(i int) int {
		if c := strings.Compare(aks[i], bks[i]); c != 0 {
			return c
		}
		k := aks[i]
		return compareValues(a[k], b[k])
	})
}

func compareSequences(len1, len2 int, compare func(int) int) int {
	for i := 0; i < len1 && i < len2; i++ {
		if c := compare(i); c != 0 {
			return c
		}
	}
	return compareInt64s(int64(len1), int64(len2))
}

func compareFloat64s(a, b float64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

func compareInt64s(a, b int64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

// Return an integer corresponding to the type of value stored in v, such that
// comparing the resulting integers gives the Firestore ordering for types.
func typeOrder(v *pb.Value) int {
	switch v.ValueType.(type) {
	case *pb.Value_NullValue:
		return typeOrderNull
	case *pb.Value_BooleanValue:
		return typeOrderBoolean
	case *pb.Value_IntegerValue, *pb.Value_DoubleValue:
		return typeOrderNumber
	case *pb.Value_TimestampValue:
		return typeOrderTimestamp
	case *pb.Value_StringValue:
		return typeOrderString
	case *pb.Value_BytesValue:
		return typeOrderBlob
	case *pb.Value_ReferenceValue:
		return typeOrderRef
	case *pb.Value_GeoPointValue:
		return typeOrderGeoPoint
	case *pb.Value_ArrayValue:
		return typeOrderArray
	case *pb.Value_MapValue:
		if isBSONMinKey(v) {
			return typeOrderMinKey
		}
		if isBSONMaxKey(v) {
			return typeOrderMaxKey
		}
		if isBSONInt32(v) || isBSONDecimal128(v) {
			return typeOrderNumber
		}
		if isBSONTimestamp(v) {
			return typeOrderBSONTimestamp
		}
		if isBSONBinary(v) {
			return typeOrderBSONBinary
		}
		if isBSONObjectID(v) {
			return typeOrderBSONObjectID
		}
		if isBSONRegex(v) {
			return typeOrderRegex
		}
		if isVector(v) {
			return typeOrderVector
		}
		return typeOrderObject
	default:
		panic(fmt.Sprintf("bad value type: %v", v))
	}
}

// byReferenceValue implements sort.Interface for []*firestorepb.Value
type byFirestoreValue []*pb.Value

func (a byFirestoreValue) Len() int           { return len(a) }
func (a byFirestoreValue) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byFirestoreValue) Less(i, j int) bool { return compareValues(a[i], a[j]) < 0 }

func isMapWithSingleKey(v *pb.Value, key string) (*pb.Value, bool) {
	mv, ok := v.ValueType.(*pb.Value_MapValue)
	if !ok {
		return nil, false
	}
	fields := mv.MapValue.Fields
	if len(fields) != 1 {
		return nil, false
	}
	val, ok := fields[key]
	return val, ok
}

func isBSONMinKey(v *pb.Value) bool {
	val, ok := isMapWithSingleKey(v, "__min__")
	if !ok {
		return false
	}
	_, ok = val.ValueType.(*pb.Value_NullValue)
	return ok
}

func isBSONMaxKey(v *pb.Value) bool {
	val, ok := isMapWithSingleKey(v, "__max__")
	if !ok {
		return false
	}
	_, ok = val.ValueType.(*pb.Value_NullValue)
	return ok
}

func isBSONInt32(v *pb.Value) bool {
	val, ok := isMapWithSingleKey(v, "__int__")
	if !ok {
		return false
	}
	_, ok = val.ValueType.(*pb.Value_IntegerValue)
	return ok
}

func isBSONDecimal128(v *pb.Value) bool {
	val, ok := isMapWithSingleKey(v, "__decimal128__")
	if !ok {
		return false
	}
	_, ok = val.ValueType.(*pb.Value_StringValue)
	return ok
}

func isBSONObjectID(v *pb.Value) bool {
	val, ok := isMapWithSingleKey(v, "__oid__")
	if !ok {
		return false
	}
	_, ok = val.ValueType.(*pb.Value_StringValue)
	return ok
}

func isBSONBinary(v *pb.Value) bool {
	val, ok := isMapWithSingleKey(v, "__binary__")
	if !ok {
		return false
	}
	_, ok = val.ValueType.(*pb.Value_BytesValue)
	return ok
}

func isBSONTimestamp(v *pb.Value) bool {
	val, ok := isMapWithSingleKey(v, "__request_timestamp__")
	if !ok {
		return false
	}
	tsMap, ok := val.ValueType.(*pb.Value_MapValue)
	if !ok {
		return false
	}
	tf := tsMap.MapValue.Fields
	if len(tf) != 2 {
		return false
	}
	_, ok1 := tf["seconds"].ValueType.(*pb.Value_IntegerValue)
	_, ok2 := tf["increment"].ValueType.(*pb.Value_IntegerValue)
	return ok1 && ok2
}

func isBSONRegex(v *pb.Value) bool {
	val, ok := isMapWithSingleKey(v, "__regex__")
	if !ok {
		return false
	}
	regexMap, ok := val.ValueType.(*pb.Value_MapValue)
	if !ok {
		return false
	}
	rf := regexMap.MapValue.Fields
	if len(rf) != 2 {
		return false
	}
	_, ok1 := rf["pattern"].ValueType.(*pb.Value_StringValue)
	_, ok2 := rf["options"].ValueType.(*pb.Value_StringValue)
	return ok1 && ok2
}

func isVector(v *pb.Value) bool {
	mv, ok := v.ValueType.(*pb.Value_MapValue)
	if !ok {
		return false
	}
	fields := mv.MapValue.Fields
	typeVal, ok := fields["__type__"]
	if !ok {
		return false
	}
	sv, ok := typeVal.ValueType.(*pb.Value_StringValue)
	return ok && sv.StringValue == "__vector__"
}

func extractNumber(v *pb.Value) float64 {
	switch x := v.ValueType.(type) {
	case *pb.Value_IntegerValue:
		return float64(x.IntegerValue)
	case *pb.Value_DoubleValue:
		return x.DoubleValue
	case *pb.Value_MapValue:
		if val, ok := isMapWithSingleKey(v, "__int__"); ok {
			return float64(val.GetIntegerValue())
		}
		if val, ok := isMapWithSingleKey(v, "__decimal128__"); ok {
			f, err := strconv.ParseFloat(val.GetStringValue(), 64)
			if err != nil {
				return math.NaN()
			}
			return f
		}
	}
	return 0
}

func compareBSONTimestamps(a, b *pb.Value) int {
	valA, _ := isMapWithSingleKey(a, "__request_timestamp__")
	valB, _ := isMapWithSingleKey(b, "__request_timestamp__")

	mapA := valA.GetMapValue().Fields
	mapB := valB.GetMapValue().Fields

	secA := mapA["seconds"].GetIntegerValue()
	secB := mapB["seconds"].GetIntegerValue()
	if c := compareInt64s(secA, secB); c != 0 {
		return c
	}
	incA := mapA["increment"].GetIntegerValue()
	incB := mapB["increment"].GetIntegerValue()
	return compareInt64s(incA, incB)
}

func compareBSONRegexes(a, b *pb.Value) int {
	valA, _ := isMapWithSingleKey(a, "__regex__")
	valB, _ := isMapWithSingleKey(b, "__regex__")

	mapA := valA.GetMapValue().Fields
	mapB := valB.GetMapValue().Fields

	patA := mapA["pattern"].GetStringValue()
	patB := mapB["pattern"].GetStringValue()
	if c := strings.Compare(patA, patB); c != 0 {
		return c
	}
	optA := mapA["options"].GetStringValue()
	optB := mapB["options"].GetStringValue()
	return strings.Compare(optA, optB)
}
