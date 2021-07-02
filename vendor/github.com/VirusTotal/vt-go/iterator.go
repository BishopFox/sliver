// Copyright © 2017 The vt-go authors. All Rights Reserved.
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

import (
	"bytes"
	"compress/flate"
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strconv"
	"time"
)

const (
	ok = iota
	retry
	stop
)

type cursor struct {
	Link   string
	Offset int
}

func (c *cursor) encode() string {
	if c.Link == "" {
		return ""
	}
	var b bytes.Buffer
	b64 := base64.NewEncoder(base64.RawURLEncoding, &b)
	fw, _ := flate.NewWriter(b64, flate.BestCompression)
	json.NewEncoder(fw).Encode(c)
	fw.Close()
	return b.String()
}

func (c *cursor) decode(s string) error {
	if s == "" {
		c.Link = ""
		c.Offset = 0
		return nil
	}
	b := bytes.NewBufferString(s)
	fr := flate.NewReader(base64.NewDecoder(base64.RawURLEncoding, b))
	err := json.NewDecoder(fr).Decode(&c)
	if err != nil {
		return err
	}
	return nil
}

type collectionObject struct {
	object *Object
	cursor cursor
}

// IteratorOption represents an option passed to an iterator.
type IteratorOption func(*Iterator) error

// IteratorCursor specifies a cursor for the iterator. The iterator will start
// at the point indicated by the cursor.
func IteratorCursor(cursor string) IteratorOption {
	return func(it *Iterator) error {
		it.cursor = cursor
		return nil
	}
}

// IteratorFilter specifies a filtering query that is sent to the backend. The
// backend will return items that comply with the condition imposed by the
// filter. The filter syntax varies depending on the collection being iterated.
func IteratorFilter(filter string) IteratorOption {
	return func(it *Iterator) error {
		it.filter = filter
		return nil
	}
}

// IteratorBatchSize specifies the number of items that are retrieved in a
// single call to the backend.
func IteratorBatchSize(n int) IteratorOption {
	return func(it *Iterator) error {
		it.batchSize = n
		return nil
	}
}

// IteratorLimit specifies a maximum number of items that will be returned by
// the iterator.
func IteratorLimit(n int) IteratorOption {
	return func(it *Iterator) error {
		it.limit = n
		return nil
	}
}

// IteratorDescriptorsOnly receives a boolean that indicate whether or not we want
// the backend to respond with object descriptors instead of the full objects.
func IteratorDescriptorsOnly(b bool) IteratorOption {
	return func(it *Iterator) error {
		it.descriptorsOnly = b
		return nil
	}
}

// Iterator represents a iterator over a collection of VirusTotal objects.
type Iterator struct {
	client          *Client
	ch              chan interface{}
	done            chan bool
	next            *Object
	err             error
	closed          bool
	limit           int
	count           int
	batchSize       int
	filter          string
	cursor          string
	descriptorsOnly bool
	links           Links
	meta            map[string]interface{}
}

func newIterator(cli *Client, u *url.URL, options ...IteratorOption) (*Iterator, error) {

	skip := 0
	it := &Iterator{
		client: cli,
		ch:     make(chan interface{}, 50),
		done:   make(chan bool)}

	for _, opt := range options {
		if err := opt(it); err != nil {
			return nil, err
		}
	}

	if it.cursor != "" {
		c := cursor{}
		err := c.decode(it.cursor)
		if err != nil {
			return nil, err
		}
		it.links.Next = c.Link
		skip = c.Offset
	} else {
		q := u.Query()
		if it.batchSize > 0 {
			q.Add("limit", strconv.Itoa(it.batchSize))
		}
		if it.filter != "" {
			q.Add("filter", it.filter)
		}
		if it.descriptorsOnly {
			q.Add("descriptors_only", "true")
		}
		u.RawQuery = q.Encode()
		it.links.Next = u.String()
	}

	go it.iterate(skip)

	return it, nil
}

// Next advances the iterator to the next object and returns true if there are
// more objects or false if the end of the collection has been reached.
func (it *Iterator) Next() bool {
	if it.limit > 0 && it.count == it.limit {
		return false
	}
	item, ok := <-it.ch
	if ok {
		switch v := item.(type) {
		case collectionObject:
			it.next = v.object
			it.cursor = v.cursor.encode()
			it.count++
		case error:
			it.next = nil
			it.err = v
		}
	}
	return ok && it.next != nil
}

// Get returns the current object in the collection iterator.
func (it *Iterator) Get() *Object {
	return it.next
}

// Cursor returns a token indicating the current iterator's position.
func (it *Iterator) Cursor() string {
	return it.cursor
}

// Close closes a collection iterator.
func (it *Iterator) Close() {
	if !it.closed {
		it.closed = true
		it.done <- true
	}
}

// Meta returns the metadata returned by the server during the last call to
// the collection's endpoint.
func (it *Iterator) Meta() map[string]interface{} {
	return it.meta
}

// Error returns any error occurred during the iteration of a collection.
func (it *Iterator) Error() error {
	return it.err
}

func (it *Iterator) trySendToChannel(item interface{}) int {
	select {
	case <-it.done:
		return stop
	case it.ch <- item:
		return ok
	default:
		return retry
	}
}

func (it *Iterator) sendToChannel(item interface{}) int {
	sent := false
	for !sent {
		switch it.trySendToChannel(item) {
		case ok:
			sent = true
		case retry:
			time.Sleep(10 * time.Millisecond)
		case stop:
			return stop
		}
	}
	return ok
}

func (it *Iterator) getMoreObjects() (objs []*Object, err error) {
	nextURL, err := url.Parse(it.links.Next)
	if err != nil {
		return nil, err
	}
	var resp *Response
	var data json.RawMessage
	if resp, err = it.client.GetData(nextURL, &data); err != nil {
		return nil, err
	}
	// Try to unmarshall the data into an object, if it succeeds is because the
	// user passed and endpoint that returns a single object to the iterator.
	// This case is handled as a collection that returns a single object.
	obj := &Object{}
	if err = json.Unmarshal(data, obj); err == nil {
		objs = append(objs, obj)
	} else if err = json.Unmarshal(data, &objs); err != nil {
		return nil, err
	}
	it.links = resp.Links
	it.meta = resp.Meta
	return objs, nil
}

func (it *Iterator) iterate(skip int) {
	sent := 0
loop:
	for it.limit == 0 || sent < it.limit {
		// Send request to the API to get more objects.
		objects, err := it.getMoreObjects()
		if err != nil {
			// If an error occurred send it through the channel
			if it.sendToChannel(err) == stop {
				break loop
			}
		}

		objects = objects[skip:]
		for i, object := range objects {
			co := collectionObject{object: object}
			if i == len(objects)-1 {
				co.cursor.Link = it.links.Next
				co.cursor.Offset = 0
			} else {
				co.cursor.Link = it.links.Self
				co.cursor.Offset = skip + i + 1
			}
			if it.sendToChannel(co) == stop {
				break loop
			}
			sent++
		}

		if len(objects) == 0 || it.links.Next == "" {
			break loop
		}

		skip = 0
	}
	it.closed = true
	close(it.ch)
	close(it.done)
}
