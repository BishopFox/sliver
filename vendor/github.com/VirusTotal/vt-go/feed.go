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

import (
	"bufio"
	"compress/bzip2"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// FeedType ...
type FeedType string

const (
	// FileFeed is the feed type passed to NewFeed() for getting a feed with
	// all the files being scanned by VirusTotal.
	FileFeed FeedType = "files"
)

// A Feed represents a stream of objects received from VirusTotal via the
// feed API v3. This API allows you to get information about objects as they are
// processed by VirusTotal in real-time. Objects are sent on channel C.
type Feed struct {
	C        chan *Object
	client   *Client
	feedType FeedType
	// t is the time of the current package and n is index of the current item
	// within the package, the feed cursor is determined by the t and n.
	t                        time.Time
	n                        int64
	stop                     chan bool
	stopped                  bool
	err                      error
	missingPackagesTolerance int
}

// FeedOption represents an option passed to a NewFeed.
type FeedOption func(*Feed) error

// FeedBufferSize specifies the size of the Feed's buffer.
func FeedBufferSize(size int) FeedOption {
	return func(f *Feed) error {
		f.C = make(chan *Object, size)
		return nil
	}
}

// FeedCursor specifies the point in time where the feed starts. Files processed
// by VirusTotal after that time will be retrieved. The cursor is a string with
// the format YYYYMMDDhhmm, indicating the date and time with minute precision.
// If a empty string is passed as cursor the current time will be used.
func FeedCursor(cursor string) FeedOption {
	return func(f *Feed) error {
		var err error
		// An empty cursor is acceptable, it's equivalent to passing no cursor
		// at all.
		if cursor == "" {
			return nil
		}
		// Cursor can be either YYYYMMDDhhmm or YYYYMMDDhhmm-N where N
		// indicates a line number within package YYYYMMDDhhmm.
		s := strings.Split(cursor, "-")
		if len(s) > 1 {
			f.n, err = strconv.ParseInt(s[1], 10, 32)
		}
		if err == nil {
			f.t, err = time.Parse("200601021504", s[0])
		}
		return err
	}
}

// NewFeed creates a Feed that receives objects from the specified type. Objects
// are send on channel C. The feed can be stopped at any moment by calling Stop.
// This example illustrates how a Feed is typically used:
//
//  feed, err := vt.Client(<api key>).NewFeed(vt.FileFeed)
//  if err != nil {
//     ... handle error
//  }
//  for fileObj := range feed.C {
//     ... do something with file object
//  }
//  if feed.Error() != nil {
//     ... feed as been stopped by some error.
//  }
//
func (cli *Client) NewFeed(t FeedType, options ...FeedOption) (*Feed, error) {
	feed := &Feed{
		client:                   cli,
		feedType:                 t,
		t:                        time.Now().UTC().Add(-1 * time.Hour),
		stop:                     make(chan bool, 1),
		missingPackagesTolerance: 1,
	}

	for _, opt := range options {
		if err := opt(feed); err != nil {
			return nil, err
		}
	}

	// If the channel hasn't been created yet with a custom buffer size by
	// WithBufferSize, let's create it with a default size.
	if feed.C == nil {
		feed.C = make(chan *Object, 1000)
	}

	go feed.retrieve()

	return feed, nil
}

// Cursor returns a string that can be passed to FeedCursor for creating a
// feed that resumes where a previous one left.
func (f *Feed) Cursor() string {
	return fmt.Sprintf("%s-%d", f.t.Format("200601021504"), f.n)
}

// Error returns any error occurred so far.
func (f *Feed) Error() error {
	return f.err
}

// Stop causes the feed to stop sending objects to the channel C. After Stop is
// called the feed still sends all the objects that it has buffered.
func (f *Feed) Stop() error {
	if !f.stopped {
		f.stopped = true
		f.stop <- true
	}
	return nil
}

// Send the object to the feed's channel, except if it was stopped.
func (f *Feed) sendToChannel(object *Object) int {
	select {
	case <-f.stop:
		return stop
	case f.C <- object:
		return ok
	}
}

// Wait for the given amount of time, but exits earlier if the feed is stopped
// during the waiting period.
func (f *Feed) wait(d time.Duration) int {
	select {
	case <-f.stop:
		return stop
	case <-time.After(d):
		return ok
	}
}

var errNoAvailableYet = errors.New("not available yet")
var errNotFound = errors.New("not found")

func (f *Feed) getObjects(packageTime string) ([]*Object, error) {

	u := URL("feeds/%s/%s", f.feedType, packageTime)

	httpResp, err := f.client.sendRequest("GET", u, nil, nil)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	switch httpResp.StatusCode {
	case http.StatusBadRequest:
		if resp, err := f.client.parseResponse(httpResp); err != nil {
			if resp.Error.Code == "NotAvailableYet" {
				return nil, errNoAvailableYet
			}
		}
	case http.StatusNotFound:
		return nil, errNotFound
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, errors.New(httpResp.Status)
	}

	sc := bufio.NewScanner(bzip2.NewReader(httpResp.Body))
	// By default bufio.Scanner uses a buffer that is limited to a maximum size
	// defined by bufio.MaxScanBufferSize (64KB). This is too small for
	// accommodating the large JSONs stored in the feed files. So we create an
	// initial 1MB buffer and let it grow up to 10MB.
	buffer := make([]byte, 1*1024*1024)
	sc.Buffer(buffer, 10*1024*1024)

	objects := make([]*Object, 0)
	for sc.Scan() {
		obj := &Object{}
		if err := json.Unmarshal(sc.Bytes(), obj); err != nil {
			return objects, err
		}
		objects = append(objects, obj)
	}

	return objects, sc.Err()
}

func (f *Feed) retrieve() {
	waitDuration := 20 * time.Second
	missingPackages := 0
loop:
	for {
		packageTime := f.t.Format("200601021504") // YYYYMMDDhhmm
		objects, err := f.getObjects(packageTime)
		objects = objects[f.n:]
		switch err {
		case nil:
			for _, object := range objects {
				if f.sendToChannel(object) == stop {
					break loop
				}
				f.n++
			}
			f.t = f.t.Add(60 * time.Second)
			f.n = 0
			waitDuration = 20 * time.Second
			missingPackages = 0
		case errNoAvailableYet:
			// Feed package is not available yet, let's wait for 1 minute and
			// try again. If Close() is called during the waiting period it
			// exits early and breaks the loop.
			if f.wait(waitDuration) == stop {
				break loop
			}
			waitDuration *= 2
		case errNotFound:
			// The feed tolerates some missing packages, if the number of missing
			// packages is greater than missingPackagesTolerance an error is
			// returned, if not, it tries to get the next package.
			missingPackages++
			if missingPackages > f.missingPackagesTolerance {
				f.err = err
				break loop
			}
			f.t = f.t.Add(60 * time.Second)
		default:
			f.err = err
			break loop
		}
	}
	f.stopped = true
	close(f.C)
	close(f.stop)
}
