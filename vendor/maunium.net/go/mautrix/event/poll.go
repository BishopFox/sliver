// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package event

type PollResponseEventContent struct {
	RelatesTo RelatesTo `json:"m.relates_to"`
	Response  struct {
		Answers []string `json:"answers"`
	} `json:"org.matrix.msc3381.poll.response"`
}

func (content *PollResponseEventContent) GetRelatesTo() *RelatesTo {
	return &content.RelatesTo
}

func (content *PollResponseEventContent) OptionalGetRelatesTo() *RelatesTo {
	if content.RelatesTo.Type == "" {
		return nil
	}
	return &content.RelatesTo
}

func (content *PollResponseEventContent) SetRelatesTo(rel *RelatesTo) {
	content.RelatesTo = *rel
}

type MSC1767Message struct {
	Text    string           `json:"org.matrix.msc1767.text,omitempty"`
	HTML    string           `json:"org.matrix.msc1767.html,omitempty"`
	Message []ExtensibleText `json:"org.matrix.msc1767.message,omitempty"`
}

type PollStartEventContent struct {
	RelatesTo *RelatesTo `json:"m.relates_to,omitempty"`
	Mentions  *Mentions  `json:"m.mentions,omitempty"`
	PollStart struct {
		Kind          string         `json:"kind"`
		MaxSelections int            `json:"max_selections"`
		Question      MSC1767Message `json:"question"`
		Answers       []struct {
			ID string `json:"id"`
			MSC1767Message
		} `json:"answers"`
	} `json:"org.matrix.msc3381.poll.start"`
}

func (content *PollStartEventContent) GetRelatesTo() *RelatesTo {
	if content.RelatesTo == nil {
		content.RelatesTo = &RelatesTo{}
	}
	return content.RelatesTo
}

func (content *PollStartEventContent) OptionalGetRelatesTo() *RelatesTo {
	return content.RelatesTo
}

func (content *PollStartEventContent) SetRelatesTo(rel *RelatesTo) {
	content.RelatesTo = rel
}
