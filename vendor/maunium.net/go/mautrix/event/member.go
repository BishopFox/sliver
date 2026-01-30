// Copyright (c) 2020 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package event

import (
	"maunium.net/go/mautrix/id"
)

// Membership is an enum specifying the membership state of a room member.
type Membership string

func (ms Membership) IsInviteOrJoin() bool {
	return ms == MembershipJoin || ms == MembershipInvite
}

func (ms Membership) IsLeaveOrBan() bool {
	return ms == MembershipLeave || ms == MembershipBan
}

// The allowed membership states as specified in spec section 10.5.5.
const (
	MembershipJoin   Membership = "join"
	MembershipLeave  Membership = "leave"
	MembershipInvite Membership = "invite"
	MembershipBan    Membership = "ban"
	MembershipKnock  Membership = "knock"
)

// MemberEventContent represents the content of a m.room.member state event.
// https://spec.matrix.org/v1.2/client-server-api/#mroommember
type MemberEventContent struct {
	Membership                   Membership          `json:"membership"`
	AvatarURL                    id.ContentURIString `json:"avatar_url,omitempty"`
	Displayname                  string              `json:"displayname,omitempty"`
	IsDirect                     bool                `json:"is_direct,omitempty"`
	ThirdPartyInvite             *ThirdPartyInvite   `json:"third_party_invite,omitempty"`
	Reason                       string              `json:"reason,omitempty"`
	JoinAuthorisedViaUsersServer id.UserID           `json:"join_authorised_via_users_server,omitempty"`
	MSC3414File                  *EncryptedFileInfo  `json:"org.matrix.msc3414.file,omitempty"`

	MSC4293RedactEvents bool `json:"org.matrix.msc4293.redact_events,omitempty"`
}

type SignedThirdPartyInvite struct {
	Token      string                         `json:"token"`
	Signatures map[string]map[id.KeyID]string `json:"signatures,omitempty"`
	MXID       string                         `json:"mxid"`
}

type ThirdPartyInvite struct {
	DisplayName string                 `json:"display_name"`
	Signed      SignedThirdPartyInvite `json:"signed"`
}

type ThirdPartyInviteEventContent struct {
	DisplayName    string                `json:"display_name"`
	KeyValidityURL string                `json:"key_validity_url"`
	PublicKey      id.Ed25519            `json:"public_key"`
	PublicKeys     []ThirdPartyInviteKey `json:"public_keys,omitempty"`
}

type ThirdPartyInviteKey struct {
	KeyValidityURL string     `json:"key_validity_url,omitempty"`
	PublicKey      id.Ed25519 `json:"public_key"`
}
