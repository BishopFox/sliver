// Copyright 2020 LINE Corporation
//
// LINE Corporation licenses this file to you under the Apache License,
// version 2.0 (the "License"); you may not use this file except in compliance
// with the License. You may obtain a copy of the License at:
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package linebot

import (
	"encoding/json"
)

// Sender type
type Sender struct {
	Name    string
	IconURL string
}

// MarshalJSON method of QuickReplyButton
func (s *Sender) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Name    string `json:"name,omitempty"`
		IconURL string `json:"iconUrl,omitempty"`
	}{
		Name:    s.Name,
		IconURL: s.IconURL,
	})
}

// NewSender function
func NewSender(name, iconURL string) *Sender {
	return &Sender{
		Name:    name,
		IconURL: iconURL,
	}
}
