// Copyright 2018 LINE Corporation
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

// QuickReplyItems struct
type QuickReplyItems struct {
	Items []*QuickReplyButton `json:"items"`
}

// NewQuickReplyItems function
func NewQuickReplyItems(buttons ...*QuickReplyButton) *QuickReplyItems {
	return &QuickReplyItems{
		Items: buttons,
	}
}

// QuickReplyButton type
type QuickReplyButton struct {
	ImageURL string
	Action   QuickReplyAction
}

// MarshalJSON method of QuickReplyButton
func (b *QuickReplyButton) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type     string           `json:"type"`
		ImageURL string           `json:"imageUrl,omitempty"`
		Action   QuickReplyAction `json:"action"`
	}{
		Type:     "action",
		ImageURL: b.ImageURL,
		Action:   b.Action,
	})
}

// NewQuickReplyButton function
func NewQuickReplyButton(imageURL string, action QuickReplyAction) *QuickReplyButton {
	return &QuickReplyButton{
		ImageURL: imageURL,
		Action:   action,
	}
}
