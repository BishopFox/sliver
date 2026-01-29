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
	"strings"

	"firebase.google.com/go/v4/internal"
)

// txnRetires is the maximum number of times a transaction is retried before giving up. Transaction
// retries are triggered by concurrent conflicting updates to the same database location.
const txnRetries = 25

// Ref represents a node in the Firebase Realtime Database.
type Ref struct {
	Key  string
	Path string

	segs   []string
	client *Client
}

// TransactionNode represents the value of a node within the scope of a transaction.
type TransactionNode interface {
	Unmarshal(v interface{}) error
}

type transactionNodeImpl struct {
	Raw []byte
}

func (t *transactionNodeImpl) Unmarshal(v interface{}) error {
	return json.Unmarshal(t.Raw, v)
}

// Parent returns a reference to the parent of the current node.
//
// If the current reference points to the root of the database, Parent returns nil.
func (r *Ref) Parent() *Ref {
	l := len(r.segs)
	if l > 0 {
		path := strings.Join(r.segs[:l-1], "/")
		return r.client.NewRef(path)
	}
	return nil
}

// Child returns a reference to the specified child node.
func (r *Ref) Child(path string) *Ref {
	fp := fmt.Sprintf("%s/%s", r.Path, path)
	return r.client.NewRef(fp)
}

// Get retrieves the value at the current database location, and stores it in the value pointed to
// by v.
//
// Data deserialization is performed using https://golang.org/pkg/encoding/json/#Unmarshal, and
// therefore v has the same requirements as the json package. Specifically, it must be a pointer,
// and must not be nil.
func (r *Ref) Get(ctx context.Context, v interface{}) error {
	req := &internal.Request{
		Method: http.MethodGet,
	}
	_, err := r.sendAndUnmarshal(ctx, req, v)
	return err
}

// GetWithETag retrieves the value at the current database location, along with its ETag.
func (r *Ref) GetWithETag(ctx context.Context, v interface{}) (string, error) {
	req := &internal.Request{
		Method: http.MethodGet,
		Opts: []internal.HTTPOption{
			internal.WithHeader("X-Firebase-ETag", "true"),
		},
	}
	resp, err := r.sendAndUnmarshal(ctx, req, v)
	if err != nil {
		return "", err
	}

	return resp.Header.Get("Etag"), nil
}

// GetShallow performs a shallow read on the current database location.
//
// Shallow reads do not retrieve the child nodes of the current reference.
func (r *Ref) GetShallow(ctx context.Context, v interface{}) error {
	req := &internal.Request{
		Method: http.MethodGet,
		Opts: []internal.HTTPOption{
			internal.WithQueryParam("shallow", "true"),
		},
	}
	_, err := r.sendAndUnmarshal(ctx, req, v)
	return err
}

// GetIfChanged retrieves the value and ETag of the current database location only if the specified
// ETag does not match.
//
// If the specified ETag does not match, returns true along with the latest ETag of the database
// location. The value of the database location will be stored in v just like a regular Get() call.
// If the etag matches, returns false along with the same ETag passed into the function. No data
// will be stored in v in this case.
func (r *Ref) GetIfChanged(ctx context.Context, etag string, v interface{}) (bool, string, error) {
	req := &internal.Request{
		Method: http.MethodGet,
		Opts: []internal.HTTPOption{
			internal.WithHeader("If-None-Match", etag),
		},
		SuccessFn: successOrNotModified,
	}
	resp, err := r.sendAndUnmarshal(ctx, req, nil)
	if err != nil {
		return false, "", err
	}

	if resp.Status == http.StatusNotModified {
		return false, etag, nil
	}

	if err := json.Unmarshal(resp.Body, v); err != nil {
		return false, "", err
	}

	return true, resp.Header.Get("ETag"), nil
}

// Set stores the value v in the current database node.
//
// Set uses https://golang.org/pkg/encoding/json/#Marshal to serialize values into JSON. Therefore
// v has the same requirements as the json package. Values like functions and channels cannot be
// saved into Realtime Database.
func (r *Ref) Set(ctx context.Context, v interface{}) error {
	req := &internal.Request{
		Method: http.MethodPut,
		Body:   internal.NewJSONEntity(v),
		Opts: []internal.HTTPOption{
			internal.WithQueryParam("print", "silent"),
		},
	}
	_, err := r.sendAndUnmarshal(ctx, req, nil)
	return err
}

// SetIfUnchanged conditionally sets the data at this location to the given value.
//
// Sets the data at this location to v only if the specified ETag matches. Returns true if the
// value is written. Returns false if no changes are made to the database.
func (r *Ref) SetIfUnchanged(ctx context.Context, etag string, v interface{}) (bool, error) {
	req := &internal.Request{
		Method: http.MethodPut,
		Body:   internal.NewJSONEntity(v),
		Opts: []internal.HTTPOption{
			internal.WithHeader("If-Match", etag),
		},
		SuccessFn: successOrPreconditionFailed,
	}
	resp, err := r.sendAndUnmarshal(ctx, req, nil)
	if err != nil {
		return false, err
	}

	if resp.Status == http.StatusPreconditionFailed {
		return false, nil
	}

	return true, nil
}

// Push creates a new child node at the current location, and returns a reference to it.
//
// If v is not nil, it will be set as the initial value of the new child node. If v is nil, the
// new child node will be created with empty string as the value.
func (r *Ref) Push(ctx context.Context, v interface{}) (*Ref, error) {
	if v == nil {
		v = ""
	}

	req := &internal.Request{
		Method: http.MethodPost,
		Body:   internal.NewJSONEntity(v),
	}
	var d struct {
		Name string `json:"name"`
	}
	if _, err := r.sendAndUnmarshal(ctx, req, &d); err != nil {
		return nil, err
	}

	return r.Child(d.Name), nil
}

// Update modifies the specified child keys of the current location to the provided values.
func (r *Ref) Update(ctx context.Context, v map[string]interface{}) error {
	if len(v) == 0 {
		return fmt.Errorf("value argument must be a non-empty map")
	}

	req := &internal.Request{
		Method: http.MethodPatch,
		Body:   internal.NewJSONEntity(v),
		Opts: []internal.HTTPOption{
			internal.WithQueryParam("print", "silent"),
		},
	}
	_, err := r.sendAndUnmarshal(ctx, req, nil)
	return err
}

// UpdateFn represents a function type that can be passed into Transaction().
type UpdateFn func(TransactionNode) (interface{}, error)

// Transaction atomically modifies the data at this location.
//
// Unlike a normal Set(), which just overwrites the data regardless of its previous state,
// Transaction() is used to modify the existing value to a new value, ensuring there are no
// conflicts with other clients simultaneously writing to the same location.
//
// This is accomplished by passing an update function which is used to transform the current value
// of this reference into a new value. If another client writes to this location before the new
// value is successfully saved, the update function is called again with the new current value, and
// the write will be retried. In case of repeated failures, this method will retry the transaction up
// to 25 times before giving up and returning an error.
//
// The update function may also force an early abort by returning an error instead of returning a
// value.
func (r *Ref) Transaction(ctx context.Context, fn UpdateFn) error {
	req := &internal.Request{
		Method: http.MethodGet,
		Opts: []internal.HTTPOption{
			internal.WithHeader("X-Firebase-ETag", "true"),
		},
	}
	resp, err := r.sendAndUnmarshal(ctx, req, nil)
	if err != nil {
		return err
	}

	etag := resp.Header.Get("Etag")
	for i := 0; i < txnRetries; i++ {
		new, err := fn(&transactionNodeImpl{resp.Body})
		if err != nil {
			return err
		}

		req := &internal.Request{
			Method: http.MethodPut,
			Body:   internal.NewJSONEntity(new),
			Opts: []internal.HTTPOption{
				internal.WithHeader("If-Match", etag),
			},
			SuccessFn: successOrPreconditionFailed,
		}
		resp, err = r.sendAndUnmarshal(ctx, req, nil)
		if err != nil {
			return err
		}

		if resp.Status == http.StatusOK {
			return nil
		}

		etag = resp.Header.Get("ETag")
	}
	return fmt.Errorf("transaction aborted after failed retries")
}

// Delete removes this node from the database.
func (r *Ref) Delete(ctx context.Context) error {
	req := &internal.Request{
		Method: http.MethodDelete,
	}
	_, err := r.sendAndUnmarshal(ctx, req, nil)
	return err
}

func (r *Ref) sendAndUnmarshal(
	ctx context.Context, req *internal.Request, v interface{}) (*internal.Response, error) {
	req.URL = r.Path
	return r.client.sendAndUnmarshal(ctx, req, v)
}

func successOrNotModified(resp *internal.Response) bool {
	return internal.HasSuccessStatus(resp) || resp.Status == http.StatusNotModified
}

func successOrPreconditionFailed(resp *internal.Response) bool {
	return internal.HasSuccessStatus(resp) || resp.Status == http.StatusPreconditionFailed
}
