// Copyright (c) 2021 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package event

import (
	"encoding/base64"
	"encoding/json"
	"slices"

	"go.mau.fi/util/jsontime"

	"maunium.net/go/mautrix/id"
)

// CanonicalAliasEventContent represents the content of a m.room.canonical_alias state event.
// https://spec.matrix.org/v1.2/client-server-api/#mroomcanonical_alias
type CanonicalAliasEventContent struct {
	Alias      id.RoomAlias   `json:"alias"`
	AltAliases []id.RoomAlias `json:"alt_aliases,omitempty"`
}

// RoomNameEventContent represents the content of a m.room.name state event.
// https://spec.matrix.org/v1.2/client-server-api/#mroomname
type RoomNameEventContent struct {
	Name string `json:"name"`
}

// RoomAvatarEventContent represents the content of a m.room.avatar state event.
// https://spec.matrix.org/v1.2/client-server-api/#mroomavatar
type RoomAvatarEventContent struct {
	URL         id.ContentURIString `json:"url,omitempty"`
	Info        *FileInfo           `json:"info,omitempty"`
	MSC3414File *EncryptedFileInfo  `json:"org.matrix.msc3414.file,omitempty"`
}

// ServerACLEventContent represents the content of a m.room.server_acl state event.
// https://spec.matrix.org/v1.2/client-server-api/#server-access-control-lists-acls-for-rooms
type ServerACLEventContent struct {
	Allow           []string `json:"allow,omitempty"`
	AllowIPLiterals bool     `json:"allow_ip_literals"`
	Deny            []string `json:"deny,omitempty"`
}

// TopicEventContent represents the content of a m.room.topic state event.
// https://spec.matrix.org/v1.2/client-server-api/#mroomtopic
type TopicEventContent struct {
	Topic           string           `json:"topic"`
	ExtensibleTopic *ExtensibleTopic `json:"m.topic,omitempty"`
}

// ExtensibleTopic represents the contents of the m.topic field within the
// m.room.topic state event as described in [MSC3765].
//
// [MSC3765]: https://github.com/matrix-org/matrix-spec-proposals/pull/3765
type ExtensibleTopic = ExtensibleTextContainer

type ExtensibleTextContainer struct {
	Text []ExtensibleText `json:"m.text"`
}

func MakeExtensibleText(text string) *ExtensibleTextContainer {
	return &ExtensibleTextContainer{
		Text: []ExtensibleText{{
			Body:     text,
			MimeType: "text/plain",
		}},
	}
}

func MakeExtensibleFormattedText(plaintext, html string) *ExtensibleTextContainer {
	return &ExtensibleTextContainer{
		Text: []ExtensibleText{{
			Body:     plaintext,
			MimeType: "text/plain",
		}, {
			Body:     html,
			MimeType: "text/html",
		}},
	}
}

// ExtensibleText represents the contents of an m.text field.
type ExtensibleText struct {
	MimeType string `json:"mimetype,omitempty"`
	Body     string `json:"body"`
}

// TombstoneEventContent represents the content of a m.room.tombstone state event.
// https://spec.matrix.org/v1.2/client-server-api/#mroomtombstone
type TombstoneEventContent struct {
	Body            string    `json:"body"`
	ReplacementRoom id.RoomID `json:"replacement_room"`
}

func (tec *TombstoneEventContent) GetReplacementRoom() id.RoomID {
	if tec == nil {
		return ""
	}
	return tec.ReplacementRoom
}

type Predecessor struct {
	RoomID  id.RoomID  `json:"room_id"`
	EventID id.EventID `json:"event_id"`
}

// Deprecated: use id.RoomVersion instead
type RoomVersion = id.RoomVersion

// Deprecated: use id.RoomVX constants instead
const (
	RoomV1  = id.RoomV1
	RoomV2  = id.RoomV2
	RoomV3  = id.RoomV3
	RoomV4  = id.RoomV4
	RoomV5  = id.RoomV5
	RoomV6  = id.RoomV6
	RoomV7  = id.RoomV7
	RoomV8  = id.RoomV8
	RoomV9  = id.RoomV9
	RoomV10 = id.RoomV10
	RoomV11 = id.RoomV11
	RoomV12 = id.RoomV12
)

// CreateEventContent represents the content of a m.room.create state event.
// https://spec.matrix.org/v1.2/client-server-api/#mroomcreate
type CreateEventContent struct {
	Type        RoomType       `json:"type,omitempty"`
	Federate    *bool          `json:"m.federate,omitempty"`
	RoomVersion id.RoomVersion `json:"room_version,omitempty"`
	Predecessor *Predecessor   `json:"predecessor,omitempty"`

	// Room v12+ only
	AdditionalCreators []id.UserID `json:"additional_creators,omitempty"`

	// Deprecated: use the event sender instead
	Creator id.UserID `json:"creator,omitempty"`
}

func (cec *CreateEventContent) GetPredecessor() (p Predecessor) {
	if cec != nil && cec.Predecessor != nil {
		p = *cec.Predecessor
	}
	return
}

func (cec *CreateEventContent) SupportsCreatorPower() bool {
	if cec == nil {
		return false
	}
	return cec.RoomVersion.PrivilegedRoomCreators()
}

// JoinRule specifies how open a room is to new members.
// https://spec.matrix.org/v1.2/client-server-api/#mroomjoin_rules
type JoinRule string

const (
	JoinRulePublic          JoinRule = "public"
	JoinRuleKnock           JoinRule = "knock"
	JoinRuleInvite          JoinRule = "invite"
	JoinRuleRestricted      JoinRule = "restricted"
	JoinRuleKnockRestricted JoinRule = "knock_restricted"
	JoinRulePrivate         JoinRule = "private"
)

// JoinRulesEventContent represents the content of a m.room.join_rules state event.
// https://spec.matrix.org/v1.2/client-server-api/#mroomjoin_rules
type JoinRulesEventContent struct {
	JoinRule JoinRule        `json:"join_rule"`
	Allow    []JoinRuleAllow `json:"allow,omitempty"`
}

type JoinRuleAllowType string

const (
	JoinRuleAllowRoomMembership JoinRuleAllowType = "m.room_membership"
)

type JoinRuleAllow struct {
	RoomID id.RoomID         `json:"room_id"`
	Type   JoinRuleAllowType `json:"type"`
}

// PinnedEventsEventContent represents the content of a m.room.pinned_events state event.
// https://spec.matrix.org/v1.2/client-server-api/#mroompinned_events
type PinnedEventsEventContent struct {
	Pinned []id.EventID `json:"pinned"`
}

// HistoryVisibility specifies who can see new messages.
// https://spec.matrix.org/v1.2/client-server-api/#mroomhistory_visibility
type HistoryVisibility string

const (
	HistoryVisibilityInvited       HistoryVisibility = "invited"
	HistoryVisibilityJoined        HistoryVisibility = "joined"
	HistoryVisibilityShared        HistoryVisibility = "shared"
	HistoryVisibilityWorldReadable HistoryVisibility = "world_readable"
)

// HistoryVisibilityEventContent represents the content of a m.room.history_visibility state event.
// https://spec.matrix.org/v1.2/client-server-api/#mroomhistory_visibility
type HistoryVisibilityEventContent struct {
	HistoryVisibility HistoryVisibility `json:"history_visibility"`
}

// GuestAccess specifies whether or not guest accounts can join.
// https://spec.matrix.org/v1.2/client-server-api/#mroomguest_access
type GuestAccess string

const (
	GuestAccessCanJoin   GuestAccess = "can_join"
	GuestAccessForbidden GuestAccess = "forbidden"
)

// GuestAccessEventContent represents the content of a m.room.guest_access state event.
// https://spec.matrix.org/v1.2/client-server-api/#mroomguest_access
type GuestAccessEventContent struct {
	GuestAccess GuestAccess `json:"guest_access"`
}

type BridgeInfoSection struct {
	ID          string              `json:"id"`
	DisplayName string              `json:"displayname,omitempty"`
	AvatarURL   id.ContentURIString `json:"avatar_url,omitempty"`
	ExternalURL string              `json:"external_url,omitempty"`

	Receiver string `json:"fi.mau.receiver,omitempty"`
}

// BridgeEventContent represents the content of a m.bridge state event.
// https://github.com/matrix-org/matrix-doc/pull/2346
type BridgeEventContent struct {
	BridgeBot id.UserID          `json:"bridgebot"`
	Creator   id.UserID          `json:"creator,omitempty"`
	Protocol  BridgeInfoSection  `json:"protocol"`
	Network   *BridgeInfoSection `json:"network,omitempty"`
	Channel   BridgeInfoSection  `json:"channel"`

	BeeperRoomType   string `json:"com.beeper.room_type,omitempty"`
	BeeperRoomTypeV2 string `json:"com.beeper.room_type.v2,omitempty"`

	TempSlackRemoteIDMigratedFlag  bool `json:"com.beeper.slack_remote_id_migrated,omitempty"`
	TempSlackRemoteIDMigratedFlag2 bool `json:"com.beeper.slack_remote_id_really_migrated,omitempty"`
}

// DisappearingType represents the type of a disappearing message timer.
type DisappearingType string

const (
	DisappearingTypeNone      DisappearingType = ""
	DisappearingTypeAfterRead DisappearingType = "after_read"
	DisappearingTypeAfterSend DisappearingType = "after_send"
)

type BeeperDisappearingTimer struct {
	Type  DisappearingType      `json:"type"`
	Timer jsontime.Milliseconds `json:"timer"`
}

type marshalableBeeperDisappearingTimer BeeperDisappearingTimer

func (bdt *BeeperDisappearingTimer) MarshalJSON() ([]byte, error) {
	if bdt == nil || bdt.Type == DisappearingTypeNone {
		return []byte("{}"), nil
	}
	return json.Marshal((*marshalableBeeperDisappearingTimer)(bdt))
}

type SpaceChildEventContent struct {
	Via       []string `json:"via,omitempty"`
	Order     string   `json:"order,omitempty"`
	Suggested bool     `json:"suggested,omitempty"`
}

type SpaceParentEventContent struct {
	Via       []string `json:"via,omitempty"`
	Canonical bool     `json:"canonical,omitempty"`
}

type PolicyRecommendation string

const (
	PolicyRecommendationBan              PolicyRecommendation = "m.ban"
	PolicyRecommendationUnstableTakedown PolicyRecommendation = "org.matrix.msc4204.takedown"
	PolicyRecommendationUnstableBan      PolicyRecommendation = "org.matrix.mjolnir.ban"
	PolicyRecommendationUnban            PolicyRecommendation = "fi.mau.meowlnir.unban"
)

type PolicyHashes struct {
	SHA256 string `json:"sha256"`
}

func (ph *PolicyHashes) DecodeSHA256() *[32]byte {
	if ph == nil || ph.SHA256 == "" {
		return nil
	}
	decoded, _ := base64.StdEncoding.DecodeString(ph.SHA256)
	if len(decoded) == 32 {
		return (*[32]byte)(decoded)
	}
	return nil
}

// ModPolicyContent represents the content of a m.room.rule.user, m.room.rule.room, and m.room.rule.server state event.
// https://spec.matrix.org/v1.2/client-server-api/#moderation-policy-lists
type ModPolicyContent struct {
	Entity         string               `json:"entity,omitempty"`
	Reason         string               `json:"reason"`
	Recommendation PolicyRecommendation `json:"recommendation"`
	UnstableHashes *PolicyHashes        `json:"org.matrix.msc4205.hashes,omitempty"`
}

func (mpc *ModPolicyContent) EntityOrHash() string {
	if mpc.UnstableHashes != nil && mpc.UnstableHashes.SHA256 != "" {
		return mpc.UnstableHashes.SHA256
	}
	return mpc.Entity
}

type ElementFunctionalMembersContent struct {
	ServiceMembers []id.UserID `json:"service_members"`
}

func (efmc *ElementFunctionalMembersContent) Add(mxid id.UserID) bool {
	if slices.Contains(efmc.ServiceMembers, mxid) {
		return false
	}
	efmc.ServiceMembers = append(efmc.ServiceMembers, mxid)
	return true
}
