// Copyright 2019 Google Inc. All Rights Reserved.
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

package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"firebase.google.com/go/v4/internal"
	"google.golang.org/api/iterator"
)

const maxReturnedResults = 1000

// Users returns an iterator over Users.
//
// If nextPageToken is empty, the iterator will start at the beginning.
// If the nextPageToken is not empty, the iterator starts after the token.
func (c *baseClient) Users(ctx context.Context, nextPageToken string) *UserIterator {
	it := &UserIterator{
		ctx:    ctx,
		client: c,
	}
	it.pageInfo, it.nextFunc = iterator.NewPageInfo(
		it.fetch,
		func() int { return len(it.users) },
		func() interface{} { b := it.users; it.users = nil; return b })
	it.pageInfo.MaxSize = maxReturnedResults
	it.pageInfo.Token = nextPageToken
	return it
}

// UserIterator is an iterator over Users.
//
// Also see: https://github.com/GoogleCloudPlatform/google-cloud-go/wiki/Iterator-Guidelines
type UserIterator struct {
	client   *baseClient
	ctx      context.Context
	nextFunc func() error
	pageInfo *iterator.PageInfo
	users    []*ExportedUserRecord
}

// PageInfo supports pagination. See the google.golang.org/api/iterator package for details.
// Page size can be determined by the NewPager(...) function described there.
func (it *UserIterator) PageInfo() *iterator.PageInfo { return it.pageInfo }

// Next returns the next result. Its second return value is [iterator.Done] if
// there are no more results. Once Next returns [iterator.Done], all subsequent
// calls will return [iterator.Done].
func (it *UserIterator) Next() (*ExportedUserRecord, error) {
	if err := it.nextFunc(); err != nil {
		return nil, err
	}
	user := it.users[0]
	it.users = it.users[1:]
	return user, nil
}

func (it *UserIterator) fetch(pageSize int, pageToken string) (string, error) {
	query := make(url.Values)
	query.Set("maxResults", strconv.Itoa(pageSize))
	if pageToken != "" {
		query.Set("nextPageToken", pageToken)
	}

	url, err := it.client.makeUserMgtURL(fmt.Sprintf("/accounts:batchGet?%s", query.Encode()))
	if err != nil {
		return "", err
	}

	req := &internal.Request{
		Method: http.MethodGet,
		URL:    url,
	}
	var parsed struct {
		Users         []userQueryResponse `json:"users"`
		NextPageToken string              `json:"nextPageToken"`
	}
	_, err = it.client.httpClient.DoAndUnmarshal(it.ctx, req, &parsed)
	if err != nil {
		return "", err
	}

	for _, u := range parsed.Users {
		eu, err := u.makeExportedUserRecord()
		if err != nil {
			return "", err
		}
		it.users = append(it.users, eu)
	}
	it.pageInfo.Token = parsed.NextPageToken
	return parsed.NextPageToken, nil
}

// ExportedUserRecord is the returned user value used when listing all the users.
type ExportedUserRecord struct {
	*UserRecord
	PasswordHash string
	PasswordSalt string
}
