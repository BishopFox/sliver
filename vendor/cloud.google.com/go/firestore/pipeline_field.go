// Copyright 2026 Google LLC
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
	pb "cloud.google.com/go/firestore/apiv1/firestorepb"
)

// field represents a reference to a field in a Firestore document, or outputs of a [Pipeline] stage.
// It implements the [Expression] and [Selectable] interfaces.
//
// Field references are used to access document field values in expressions and to specify fields
// for sorting, filtering, and projecting data in Firestore pipelines.
type field struct {
	*baseExpression
	fieldPath FieldPath
}

// FieldOf creates a new field [Expression] from a dot separated field path string or [FieldPath].
func FieldOf[T string | FieldPath](path T) Expression {
	var fieldPath FieldPath
	switch p := any(path).(type) {
	case string:
		fp, err := parseDotSeparatedString(p)
		if err != nil {
			return &field{baseExpression: &baseExpression{err: err}}
		}
		fieldPath = fp
	case FieldPath:
		fieldPath = p
	}

	if err := fieldPath.validate(); err != nil {
		return &field{baseExpression: &baseExpression{err: err}}
	}
	pbVal := &pb.Value{
		ValueType: &pb.Value_FieldReferenceValue{
			FieldReferenceValue: fieldPath.toServiceFieldPath(),
		},
	}
	return &field{fieldPath: fieldPath, baseExpression: &baseExpression{pbVal: pbVal}}
}

// getSelectionDetails returns the field path string as the default alias and the field expression itself.
// This allows a field [Expression] to satisfy the [Selectable] interface, making it directly usable
// in `Select` or `AddFields` stages without explicit aliasing if the original field name is desired.
func (f *field) getSelectionDetails() (string, Expression) {
	// For Selectable, the alias is the field path itself if not otherwise aliased by `As`.
	// This makes `FieldOf("name")` selectable as "name".
	return f.fieldPath.toServiceFieldPath(), f
}

func (f *field) isSelectable() {}
