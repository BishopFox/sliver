// Copyright © 2019 The vt-go authors. All Rights Reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package vt

import "encoding/json"

type relationshipData struct {
	Data  json.RawMessage `json:"data,omitempty"`
	Links Links           `json:"links,omitempty"`
	// IsOneToOne is true if this is a one-to-one relationship and False if
	// otherwise. If true Objects contains one object at most.
	IsOneToOne bool
	Objects    []*Object
}

// Relationship contains information about a relationship between objects.
type Relationship struct {
	data relationshipData
}

// IsOneToOne returns true if this is a one-to-one relationship.
func (r *Relationship) IsOneToOne() bool {
	return r.data.IsOneToOne
}

// Objects return the objects in this relationship.
func (r *Relationship) Objects() []*Object {
	return r.data.Objects
}
