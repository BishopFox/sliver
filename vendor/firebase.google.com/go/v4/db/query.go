// Copyright 2018 Google Inc. All Rights Reserved.
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

package db

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"firebase.google.com/go/v4/internal"
)

// QueryNode represents a data node retrieved from an ordered query.
type QueryNode interface {
	Key() string
	Unmarshal(v interface{}) error
}

// Query represents a complex query that can be executed on a Ref.
//
// Complex queries can consist of up to 2 components: a required ordering constraint, and an
// optional filtering constraint. At the server, data is first sorted according to the given
// ordering constraint (e.g. order by child). Then the filtering constraint (e.g. limit, range) is
// applied on the sorted data to produce the final result. Despite the ordering constraint, the
// final result is returned by the server as an unordered collection. Therefore the values read
// from a Query instance are not ordered.
type Query struct {
	client              *Client
	path                string
	order               orderBy
	limFirst, limLast   int
	start, end, equalTo interface{}
}

// StartAt returns a shallow copy of the Query with v set as a lower bound of a range query.
//
// The resulting Query will only return child nodes with a value greater than or equal to v.
func (q *Query) StartAt(v interface{}) *Query {
	q2 := &Query{}
	*q2 = *q
	q2.start = v
	return q2
}

// EndAt returns a shallow copy of the Query with v set as a upper bound of a range query.
//
// The resulting Query will only return child nodes with a value less than or equal to v.
func (q *Query) EndAt(v interface{}) *Query {
	q2 := &Query{}
	*q2 = *q
	q2.end = v
	return q2
}

// EqualTo returns a shallow copy of the Query with v set as an equals constraint.
//
// The resulting Query will only return child nodes whose values equal to v.
func (q *Query) EqualTo(v interface{}) *Query {
	q2 := &Query{}
	*q2 = *q
	q2.equalTo = v
	return q2
}

// LimitToFirst returns a shallow copy of the Query, which is anchored to the first n
// elements of the window.
func (q *Query) LimitToFirst(n int) *Query {
	q2 := &Query{}
	*q2 = *q
	q2.limFirst = n
	return q2
}

// LimitToLast returns a shallow copy of the Query, which is anchored to the last n
// elements of the window.
func (q *Query) LimitToLast(n int) *Query {
	q2 := &Query{}
	*q2 = *q
	q2.limLast = n
	return q2
}

// Get executes the Query and populates v with the results.
//
// Data deserialization is performed using https://golang.org/pkg/encoding/json/#Unmarshal, and
// therefore v has the same requirements as the json package. Specifically, it must be a pointer,
// and must not be nil.
//
// Despite the ordering constraint of the Query, results are not stored in any particular order
// in v. Use GetOrdered() to obtain ordered results.
func (q *Query) Get(ctx context.Context, v interface{}) error {
	qp := make(map[string]string)
	if err := initQueryParams(q, qp); err != nil {
		return err
	}

	req := &internal.Request{
		Method: http.MethodGet,
		URL:    q.path,
		Opts:   []internal.HTTPOption{internal.WithQueryParams(qp)},
	}
	_, err := q.client.sendAndUnmarshal(ctx, req, v)
	return err
}

// GetOrdered executes the Query and returns the results as an ordered slice.
func (q *Query) GetOrdered(ctx context.Context) ([]QueryNode, error) {
	var temp interface{}
	if err := q.Get(ctx, &temp); err != nil {
		return nil, err
	}
	if temp == nil {
		return nil, nil
	}

	sn := newSortableNodes(temp, q.order)
	sort.Sort(sn)
	result := make([]QueryNode, len(sn))
	for i, v := range sn {
		result[i] = v
	}
	return result, nil
}

// OrderByChild returns a Query that orders data by child values before applying filters.
//
// Returned Query can be used to set additional parameters, and execute complex database queries
// (e.g. limit queries, range queries). If r has a context associated with it, the resulting Query
// will inherit it.
func (r *Ref) OrderByChild(child string) *Query {
	return newQuery(r, orderByChild(child))
}

// OrderByKey returns a Query that orders data by key before applying filters.
//
// Returned Query can be used to set additional parameters, and execute complex database queries
// (e.g. limit queries, range queries). If r has a context associated with it, the resulting Query
// will inherit it.
func (r *Ref) OrderByKey() *Query {
	return newQuery(r, orderByProperty("$key"))
}

// OrderByValue returns a Query that orders data by value before applying filters.
//
// Returned Query can be used to set additional parameters, and execute complex database queries
// (e.g. limit queries, range queries). If r has a context associated with it, the resulting Query
// will inherit it.
func (r *Ref) OrderByValue() *Query {
	return newQuery(r, orderByProperty("$value"))
}

func newQuery(r *Ref, ob orderBy) *Query {
	return &Query{
		client: r.client,
		path:   r.Path,
		order:  ob,
	}
}

func initQueryParams(q *Query, qp map[string]string) error {
	ob, err := q.order.encode()
	if err != nil {
		return err
	}
	qp["orderBy"] = ob

	if q.limFirst > 0 && q.limLast > 0 {
		return fmt.Errorf("cannot set both limit parameter: first = %d, last = %d", q.limFirst, q.limLast)
	} else if q.limFirst < 0 {
		return fmt.Errorf("limit first cannot be negative: %d", q.limFirst)
	} else if q.limLast < 0 {
		return fmt.Errorf("limit last cannot be negative: %d", q.limLast)
	}

	if q.limFirst > 0 {
		qp["limitToFirst"] = strconv.Itoa(q.limFirst)
	} else if q.limLast > 0 {
		qp["limitToLast"] = strconv.Itoa(q.limLast)
	}

	if err := encodeFilter("startAt", q.start, qp); err != nil {
		return err
	}
	if err := encodeFilter("endAt", q.end, qp); err != nil {
		return err
	}
	return encodeFilter("equalTo", q.equalTo, qp)
}

func encodeFilter(key string, val interface{}, m map[string]string) error {
	if val == nil {
		return nil
	}
	b, err := json.Marshal(val)
	if err != nil {
		return err
	}
	m[key] = string(b)
	return nil
}

type orderBy interface {
	encode() (string, error)
}

type orderByChild string

func (p orderByChild) encode() (string, error) {
	if p == "" {
		return "", fmt.Errorf("empty child path")
	} else if strings.ContainsAny(string(p), invalidChars) {
		return "", fmt.Errorf("invalid child path with illegal characters: %q", p)
	}
	segs := parsePath(string(p))
	if len(segs) == 0 {
		return "", fmt.Errorf("invalid child path: %q", p)
	}
	b, err := json.Marshal(strings.Join(segs, "/"))
	if err != nil {
		return "", nil
	}
	return string(b), nil
}

type orderByProperty string

func (p orderByProperty) encode() (string, error) {
	b, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Firebase type ordering: https://firebase.google.com/docs/database/rest/retrieve-data#section-rest-ordered-data
const (
	typeNull      = 0
	typeBoolFalse = 1
	typeBoolTrue  = 2
	typeNumeric   = 3
	typeString    = 4
	typeObject    = 5
)

// comparableKey is a union type of numeric values and strings.
type comparableKey struct {
	Num *float64
	Str *string
}

func (k *comparableKey) Compare(o *comparableKey) int {
	if k.Str != nil && o.Str != nil {
		return strings.Compare(*k.Str, *o.Str)
	} else if k.Num != nil && o.Num != nil {
		if *k.Num < *o.Num {
			return -1
		} else if *k.Num == *o.Num {
			return 0
		}
		return 1
	} else if k.Num != nil {
		// numeric keys appear before string keys
		return -1
	}
	return 1
}

func newComparableKey(v interface{}) *comparableKey {
	if s, ok := v.(string); ok {
		return &comparableKey{Str: &s}
	}

	// Numeric values could be int (in the case of array indices and type constants), or float64 (if
	// the value was received as json).
	if i, ok := v.(int); ok {
		f := float64(i)
		return &comparableKey{Num: &f}
	}

	f := v.(float64)
	return &comparableKey{Num: &f}
}

type queryNodeImpl struct {
	CompKey   *comparableKey
	Value     interface{}
	Index     interface{}
	IndexType int
}

func (q *queryNodeImpl) Key() string {
	if q.CompKey.Str != nil {
		return *q.CompKey.Str
	}
	// Numeric keys in queryNodeImpl are always array indices, and can be safely converted into int.
	return strconv.Itoa(int(*q.CompKey.Num))
}

func (q *queryNodeImpl) Unmarshal(v interface{}) error {
	b, err := json.Marshal(q.Value)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

func newQueryNode(key, val interface{}, order orderBy) *queryNodeImpl {
	var index interface{}
	if prop, ok := order.(orderByProperty); ok {
		if prop == "$value" {
			index = val
		} else {
			index = key
		}
	} else {
		path := order.(orderByChild)
		index = extractChildValue(val, string(path))
	}
	return &queryNodeImpl{
		CompKey:   newComparableKey(key),
		Value:     val,
		Index:     index,
		IndexType: getIndexType(index),
	}
}

type sortableNodes []*queryNodeImpl

func (s sortableNodes) Len() int {
	return len(s)
}

func (s sortableNodes) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s sortableNodes) Less(i, j int) bool {
	a, b := s[i], s[j]
	var aKey, bKey *comparableKey
	if a.IndexType == b.IndexType {
		// If the indices have the same type and are comparable (i.e. numeric or string), compare
		// them directly. Otherwise, compare the keys.
		if (a.IndexType == typeNumeric || a.IndexType == typeString) && a.Index != b.Index {
			aKey, bKey = newComparableKey(a.Index), newComparableKey(b.Index)
		} else {
			aKey, bKey = a.CompKey, b.CompKey
		}
	} else {
		// If the indices are of different types, use the type ordering of Firebase.
		aKey, bKey = newComparableKey(a.IndexType), newComparableKey(b.IndexType)
	}

	return aKey.Compare(bKey) < 0
}

func newSortableNodes(values interface{}, order orderBy) sortableNodes {
	var entries sortableNodes
	if m, ok := values.(map[string]interface{}); ok {
		for key, val := range m {
			entries = append(entries, newQueryNode(key, val, order))
		}
	} else if l, ok := values.([]interface{}); ok {
		for key, val := range l {
			entries = append(entries, newQueryNode(key, val, order))
		}
	} else {
		entries = append(entries, newQueryNode(0, values, order))
	}
	return entries
}

// extractChildValue retrieves the value at path from val.
//
// If the given path does not exist in val, or val does not support child path traversal,
// extractChildValue returns nil.
func extractChildValue(val interface{}, path string) interface{} {
	segments := parsePath(path)
	curr := val
	for _, s := range segments {
		if curr == nil {
			return nil
		}

		currMap, ok := curr.(map[string]interface{})
		if !ok {
			return nil
		}
		if curr, ok = currMap[s]; !ok {
			return nil
		}
	}
	return curr
}

func getIndexType(index interface{}) int {
	if index == nil {
		return typeNull
	} else if b, ok := index.(bool); ok {
		if b {
			return typeBoolTrue
		}
		return typeBoolFalse
	} else if _, ok := index.(float64); ok {
		return typeNumeric
	} else if _, ok := index.(string); ok {
		return typeString
	}
	return typeObject
}
