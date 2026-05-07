// Copyright 2024 Google LLC
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
	"fmt"

	pb "cloud.google.com/go/firestore/apiv1/firestorepb"
)

const (
	typeKey       = "__type__"
	typeValVector = "__vector__"
	valueKey      = "value"
)

// Vector64 is an embedding vector of float64s.
type Vector64 []float64

// Vector32 is an embedding vector of float32s.
type Vector32 []float32

// vectorToProtoValue returns a Firestore [pb.Value] representing the Vector.
func vectorToProtoValue[T float32 | float64](v []T) *pb.Value {
	if v == nil {
		return nullValue
	}
	pbVals := make([]*pb.Value, len(v))
	for i, val := range v {
		pbVals[i] = floatToProtoValue(float64(val))
	}

	return &pb.Value{
		ValueType: &pb.Value_MapValue{
			MapValue: &pb.MapValue{
				Fields: map[string]*pb.Value{
					typeKey: stringToProtoValue(typeValVector),
					valueKey: {
						ValueType: &pb.Value_ArrayValue{
							ArrayValue: &pb.ArrayValue{Values: pbVals},
						},
					},
				},
			},
		},
	}
}

func vector32FromProtoValue(v *pb.Value) (Vector32, error) {
	return vectorFromProtoValue[float32](v)
}

func vector64FromProtoValue(v *pb.Value) (Vector64, error) {
	return vectorFromProtoValue[float64](v)
}

func vectorFromProtoValue[T float32 | float64](v *pb.Value) ([]T, error) {
	pbArrVals, err := pbValToVectorVals(v)
	if err != nil {
		return nil, err
	}

	floats := make([]T, len(pbArrVals))
	for i, fval := range pbArrVals {
		dv, ok := fval.ValueType.(*pb.Value_DoubleValue)
		if !ok {
			return nil, fmt.Errorf("firestore: failed to convert %v to *pb.Value_DoubleValue", fval.ValueType)
		}
		floats[i] = T(dv.DoubleValue)
	}
	return floats, nil
}

func pbValToVectorVals(v *pb.Value) ([]*pb.Value, error) {
	/*
		Vector is stored as:
		{
			"__type__": "__vector__",
			"value": []float64{},
		}
	*/
	if v == nil {
		return nil, nil
	}
	pbMap, ok := v.ValueType.(*pb.Value_MapValue)
	if !ok {
		return nil, fmt.Errorf("firestore: cannot convert %v to *pb.Value_MapValue", v.ValueType)
	}
	m := pbMap.MapValue.Fields
	var typeVal string
	typeVal, err := stringFromProtoValue(m[typeKey])
	if err != nil {
		return nil, err
	}
	if typeVal != typeValVector {
		return nil, fmt.Errorf("firestore: value of %v : %v is not %v", typeKey, typeVal, typeValVector)
	}
	pbVal, ok := m[valueKey]
	if !ok {
		return nil, fmt.Errorf("firestore: %v not present in %v", valueKey, m)
	}

	pbArr, ok := pbVal.ValueType.(*pb.Value_ArrayValue)
	if !ok {
		return nil, fmt.Errorf("firestore: failed to convert %v to *pb.Value_ArrayValue", pbVal.ValueType)
	}

	return pbArr.ArrayValue.Values, nil
}

func stringFromProtoValue(v *pb.Value) (string, error) {
	if v == nil {
		return "", fmt.Errorf("firestore: failed to convert %v to string", v)
	}
	sv, ok := v.ValueType.(*pb.Value_StringValue)
	if !ok {
		return "", fmt.Errorf("firestore: failed to convert %v to *pb.Value_StringValue", v.ValueType)
	}
	return sv.StringValue, nil
}
