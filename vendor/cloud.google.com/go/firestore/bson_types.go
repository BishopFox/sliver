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

// BSONObjectID represents a BSON ObjectID as a 24-character lowercase hex string.
type BSONObjectID string

// String returns the string representation of the BSONObjectID.
func (id BSONObjectID) String() string {
	return string(id)
}

// BSONRegex represents a BSON Regular Expression.
type BSONRegex struct {
	Pattern string
	Options string
}

// BSONTimestamp represents a BSON Timestamp.
type BSONTimestamp struct {
	Seconds   uint32
	Increment uint32
}

// BSONDecimal128 represents a BSON Decimal128.
type BSONDecimal128 string

// BSONMinKey represents BSON MinKey.
type BSONMinKey struct{}

// BSONMaxKey represents BSON MaxKey.
type BSONMaxKey struct{}

// BSONBinary represents BSON Binary data with subtype != 0.
type BSONBinary struct {
	Subtype byte
	Data    []byte
}

// BSONInt32 represents a BSON 32-bit integer.
type BSONInt32 int32
