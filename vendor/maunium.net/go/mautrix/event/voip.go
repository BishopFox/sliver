// Copyright (c) 2021 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package event

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type CallHangupReason string

const (
	CallHangupICEFailed       CallHangupReason = "ice_failed"
	CallHangupInviteTimeout   CallHangupReason = "invite_timeout"
	CallHangupUserHangup      CallHangupReason = "user_hangup"
	CallHangupUserMediaFailed CallHangupReason = "user_media_failed"
	CallHangupUnknownError    CallHangupReason = "unknown_error"
)

type CallDataType string

const (
	CallDataTypeOffer  CallDataType = "offer"
	CallDataTypeAnswer CallDataType = "answer"
)

type CallData struct {
	SDP  string       `json:"sdp"`
	Type CallDataType `json:"type"`
}

type CallCandidate struct {
	Candidate     string `json:"candidate"`
	SDPMLineIndex int    `json:"sdpMLineIndex"`
	SDPMID        string `json:"sdpMid"`
}

type CallVersion string

func (cv *CallVersion) UnmarshalJSON(raw []byte) error {
	var numberVersion int
	err := json.Unmarshal(raw, &numberVersion)
	if err != nil {
		var stringVersion string
		err = json.Unmarshal(raw, &stringVersion)
		if err != nil {
			return fmt.Errorf("failed to parse CallVersion: %w", err)
		}
		*cv = CallVersion(stringVersion)
	} else {
		*cv = CallVersion(strconv.Itoa(numberVersion))
	}
	return nil
}

func (cv *CallVersion) MarshalJSON() ([]byte, error) {
	for _, char := range *cv {
		if char < '0' || char > '9' {
			// The version contains weird characters, return as string.
			return json.Marshal(string(*cv))
		}
	}
	// The version consists of only ASCII digits, return as an integer.
	return []byte(*cv), nil
}

func (cv *CallVersion) Int() (int, error) {
	return strconv.Atoi(string(*cv))
}

type BaseCallEventContent struct {
	CallID  string      `json:"call_id"`
	PartyID string      `json:"party_id"`
	Version CallVersion `json:"version,omitempty"`
}

type CallInviteEventContent struct {
	BaseCallEventContent
	Lifetime int      `json:"lifetime"`
	Offer    CallData `json:"offer"`
}

type CallCandidatesEventContent struct {
	BaseCallEventContent
	Candidates []CallCandidate `json:"candidates"`
}

type CallRejectEventContent struct {
	BaseCallEventContent
}

type CallAnswerEventContent struct {
	BaseCallEventContent
	Answer CallData `json:"answer"`
}

type CallSelectAnswerEventContent struct {
	BaseCallEventContent
	SelectedPartyID string `json:"selected_party_id"`
}

type CallNegotiateEventContent struct {
	BaseCallEventContent
	Lifetime    int      `json:"lifetime"`
	Description CallData `json:"description"`
}

type CallHangupEventContent struct {
	BaseCallEventContent
	Reason CallHangupReason `json:"reason"`
}
