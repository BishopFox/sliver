// Copyright (c) 2023 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package mautrix

import (
	"context"
	"maps"
	"sync"

	"github.com/rs/zerolog"
	"go.mau.fi/util/exerrors"

	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

// StateStore is an interface for storing basic room state information.
type StateStore interface {
	IsInRoom(ctx context.Context, roomID id.RoomID, userID id.UserID) bool
	IsInvited(ctx context.Context, roomID id.RoomID, userID id.UserID) bool
	IsMembership(ctx context.Context, roomID id.RoomID, userID id.UserID, allowedMemberships ...event.Membership) bool
	GetMember(ctx context.Context, roomID id.RoomID, userID id.UserID) (*event.MemberEventContent, error)
	TryGetMember(ctx context.Context, roomID id.RoomID, userID id.UserID) (*event.MemberEventContent, error)
	SetMembership(ctx context.Context, roomID id.RoomID, userID id.UserID, membership event.Membership) error
	SetMember(ctx context.Context, roomID id.RoomID, userID id.UserID, member *event.MemberEventContent) error
	IsConfusableName(ctx context.Context, roomID id.RoomID, currentUser id.UserID, name string) ([]id.UserID, error)
	ClearCachedMembers(ctx context.Context, roomID id.RoomID, memberships ...event.Membership) error
	ReplaceCachedMembers(ctx context.Context, roomID id.RoomID, evts []*event.Event, onlyMemberships ...event.Membership) error

	SetPowerLevels(ctx context.Context, roomID id.RoomID, levels *event.PowerLevelsEventContent) error
	GetPowerLevels(ctx context.Context, roomID id.RoomID) (*event.PowerLevelsEventContent, error)

	SetCreate(ctx context.Context, evt *event.Event) error
	GetCreate(ctx context.Context, roomID id.RoomID) (*event.Event, error)

	GetJoinRules(ctx context.Context, roomID id.RoomID) (*event.JoinRulesEventContent, error)
	SetJoinRules(ctx context.Context, roomID id.RoomID, content *event.JoinRulesEventContent) error

	HasFetchedMembers(ctx context.Context, roomID id.RoomID) (bool, error)
	MarkMembersFetched(ctx context.Context, roomID id.RoomID) error
	GetAllMembers(ctx context.Context, roomID id.RoomID) (map[id.UserID]*event.MemberEventContent, error)

	SetEncryptionEvent(ctx context.Context, roomID id.RoomID, content *event.EncryptionEventContent) error
	IsEncrypted(ctx context.Context, roomID id.RoomID) (bool, error)

	GetRoomJoinedOrInvitedMembers(ctx context.Context, roomID id.RoomID) ([]id.UserID, error)
}

type StateStoreUpdater interface {
	UpdateState(ctx context.Context, evt *event.Event)
}

func UpdateStateStore(ctx context.Context, store StateStore, evt *event.Event) {
	if store == nil || evt == nil || evt.StateKey == nil {
		return
	}
	if directUpdater, ok := store.(StateStoreUpdater); ok {
		directUpdater.UpdateState(ctx, evt)
		return
	}
	// We only care about events without a state key (power levels, encryption) or member events with state key
	if evt.Type != event.StateMember && evt.GetStateKey() != "" {
		return
	}
	var err error
	switch content := evt.Content.Parsed.(type) {
	case *event.MemberEventContent:
		err = store.SetMember(ctx, evt.RoomID, id.UserID(evt.GetStateKey()), content)
	case *event.PowerLevelsEventContent:
		err = store.SetPowerLevels(ctx, evt.RoomID, content)
	case *event.EncryptionEventContent:
		err = store.SetEncryptionEvent(ctx, evt.RoomID, content)
	case *event.CreateEventContent:
		err = store.SetCreate(ctx, evt)
	case *event.JoinRulesEventContent:
		err = store.SetJoinRules(ctx, evt.RoomID, content)
	default:
		switch evt.Type {
		case event.StateMember, event.StatePowerLevels, event.StateEncryption, event.StateCreate:
			zerolog.Ctx(ctx).Warn().
				Stringer("event_id", evt.ID).
				Str("event_type", evt.Type.Type).
				Type("content_type", evt.Content.Parsed).
				Msg("Got known event type with unknown content type in UpdateStateStore")
		}
	}
	if err != nil {
		zerolog.Ctx(ctx).Warn().Err(err).
			Stringer("event_id", evt.ID).
			Str("event_type", evt.Type.Type).
			Msg("Failed to update state store")
	}
}

// StateStoreSyncHandler can be added as an event handler in the syncer to update the state store automatically.
//
//	client.Syncer.(mautrix.ExtensibleSyncer).OnEvent(client.StateStoreSyncHandler)
//
// DefaultSyncer.ParseEventContent must also be true for this to work (which it is by default).
func (cli *Client) StateStoreSyncHandler(ctx context.Context, evt *event.Event) {
	UpdateStateStore(ctx, cli.StateStore, evt)
}

type MemoryStateStore struct {
	Registrations  map[id.UserID]bool                                    `json:"registrations"`
	Members        map[id.RoomID]map[id.UserID]*event.MemberEventContent `json:"memberships"`
	MembersFetched map[id.RoomID]bool                                    `json:"members_fetched"`
	PowerLevels    map[id.RoomID]*event.PowerLevelsEventContent          `json:"power_levels"`
	Encryption     map[id.RoomID]*event.EncryptionEventContent           `json:"encryption"`
	Create         map[id.RoomID]*event.Event                            `json:"create"`
	JoinRules      map[id.RoomID]*event.JoinRulesEventContent            `json:"join_rules"`

	registrationsLock sync.RWMutex
	membersLock       sync.RWMutex
	powerLevelsLock   sync.RWMutex
	encryptionLock    sync.RWMutex
	joinRulesLock     sync.RWMutex
}

func NewMemoryStateStore() StateStore {
	return &MemoryStateStore{
		Registrations:  make(map[id.UserID]bool),
		Members:        make(map[id.RoomID]map[id.UserID]*event.MemberEventContent),
		MembersFetched: make(map[id.RoomID]bool),
		PowerLevels:    make(map[id.RoomID]*event.PowerLevelsEventContent),
		Encryption:     make(map[id.RoomID]*event.EncryptionEventContent),
		Create:         make(map[id.RoomID]*event.Event),
	}
}

func (store *MemoryStateStore) IsRegistered(_ context.Context, userID id.UserID) (bool, error) {
	store.registrationsLock.RLock()
	defer store.registrationsLock.RUnlock()
	registered, ok := store.Registrations[userID]
	return ok && registered, nil
}

func (store *MemoryStateStore) MarkRegistered(_ context.Context, userID id.UserID) error {
	store.registrationsLock.Lock()
	defer store.registrationsLock.Unlock()
	store.Registrations[userID] = true
	return nil
}

func (store *MemoryStateStore) GetRoomMembers(_ context.Context, roomID id.RoomID) (map[id.UserID]*event.MemberEventContent, error) {
	store.membersLock.RLock()
	members, ok := store.Members[roomID]
	store.membersLock.RUnlock()
	if !ok {
		members = make(map[id.UserID]*event.MemberEventContent)
		store.membersLock.Lock()
		store.Members[roomID] = members
		store.membersLock.Unlock()
	}
	return members, nil
}

func (store *MemoryStateStore) GetRoomJoinedOrInvitedMembers(ctx context.Context, roomID id.RoomID) ([]id.UserID, error) {
	members, err := store.GetRoomMembers(ctx, roomID)
	if err != nil {
		return nil, err
	}
	ids := make([]id.UserID, 0, len(members))
	for id := range members {
		ids = append(ids, id)
	}
	return ids, nil
}

func (store *MemoryStateStore) GetMembership(ctx context.Context, roomID id.RoomID, userID id.UserID) (event.Membership, error) {
	return exerrors.Must(store.GetMember(ctx, roomID, userID)).Membership, nil
}

func (store *MemoryStateStore) GetMember(ctx context.Context, roomID id.RoomID, userID id.UserID) (*event.MemberEventContent, error) {
	member, err := store.TryGetMember(ctx, roomID, userID)
	if member == nil && err == nil {
		member = &event.MemberEventContent{Membership: event.MembershipLeave}
	}
	return member, err
}

func (store *MemoryStateStore) IsConfusableName(ctx context.Context, roomID id.RoomID, currentUser id.UserID, name string) ([]id.UserID, error) {
	// TODO implement?
	return nil, nil
}

func (store *MemoryStateStore) TryGetMember(_ context.Context, roomID id.RoomID, userID id.UserID) (member *event.MemberEventContent, err error) {
	store.membersLock.RLock()
	defer store.membersLock.RUnlock()
	members, membersOk := store.Members[roomID]
	if !membersOk {
		return
	}
	member = members[userID]
	return
}

func (store *MemoryStateStore) IsInRoom(ctx context.Context, roomID id.RoomID, userID id.UserID) bool {
	return store.IsMembership(ctx, roomID, userID, "join")
}

func (store *MemoryStateStore) IsInvited(ctx context.Context, roomID id.RoomID, userID id.UserID) bool {
	return store.IsMembership(ctx, roomID, userID, "join", "invite")
}

func (store *MemoryStateStore) IsMembership(ctx context.Context, roomID id.RoomID, userID id.UserID, allowedMemberships ...event.Membership) bool {
	membership := exerrors.Must(store.GetMembership(ctx, roomID, userID))
	for _, allowedMembership := range allowedMemberships {
		if allowedMembership == membership {
			return true
		}
	}
	return false
}

func (store *MemoryStateStore) SetMembership(_ context.Context, roomID id.RoomID, userID id.UserID, membership event.Membership) error {
	store.membersLock.Lock()
	members, ok := store.Members[roomID]
	if !ok {
		members = map[id.UserID]*event.MemberEventContent{
			userID: {Membership: membership},
		}
	} else {
		member, ok := members[userID]
		if !ok {
			members[userID] = &event.MemberEventContent{Membership: membership}
		} else {
			member.Membership = membership
			members[userID] = member
		}
	}
	store.Members[roomID] = members
	store.membersLock.Unlock()
	return nil
}

func (store *MemoryStateStore) SetMember(_ context.Context, roomID id.RoomID, userID id.UserID, member *event.MemberEventContent) error {
	store.membersLock.Lock()
	members, ok := store.Members[roomID]
	if !ok {
		members = map[id.UserID]*event.MemberEventContent{
			userID: member,
		}
	} else {
		members[userID] = member
	}
	store.Members[roomID] = members
	store.membersLock.Unlock()
	return nil
}

func (store *MemoryStateStore) ClearCachedMembers(_ context.Context, roomID id.RoomID, memberships ...event.Membership) error {
	store.membersLock.Lock()
	defer store.membersLock.Unlock()
	members, ok := store.Members[roomID]
	if !ok {
		return nil
	}
	for userID, member := range members {
		for _, membership := range memberships {
			if membership == member.Membership {
				delete(members, userID)
				break
			}
		}
	}
	store.MembersFetched[roomID] = false
	return nil
}

func (store *MemoryStateStore) HasFetchedMembers(ctx context.Context, roomID id.RoomID) (bool, error) {
	store.membersLock.RLock()
	defer store.membersLock.RUnlock()
	return store.MembersFetched[roomID], nil
}

func (store *MemoryStateStore) MarkMembersFetched(ctx context.Context, roomID id.RoomID) error {
	store.membersLock.Lock()
	defer store.membersLock.Unlock()
	store.MembersFetched[roomID] = true
	return nil
}

func (store *MemoryStateStore) ReplaceCachedMembers(ctx context.Context, roomID id.RoomID, evts []*event.Event, onlyMemberships ...event.Membership) error {
	_ = store.ClearCachedMembers(ctx, roomID, onlyMemberships...)
	for _, evt := range evts {
		UpdateStateStore(ctx, store, evt)
	}
	if len(onlyMemberships) == 0 {
		_ = store.MarkMembersFetched(ctx, roomID)
	}
	return nil
}

func (store *MemoryStateStore) GetAllMembers(ctx context.Context, roomID id.RoomID) (map[id.UserID]*event.MemberEventContent, error) {
	store.membersLock.RLock()
	defer store.membersLock.RUnlock()
	return maps.Clone(store.Members[roomID]), nil
}

func (store *MemoryStateStore) SetPowerLevels(_ context.Context, roomID id.RoomID, levels *event.PowerLevelsEventContent) error {
	store.powerLevelsLock.Lock()
	store.PowerLevels[roomID] = levels
	store.powerLevelsLock.Unlock()
	return nil
}

func (store *MemoryStateStore) GetPowerLevels(_ context.Context, roomID id.RoomID) (levels *event.PowerLevelsEventContent, err error) {
	store.powerLevelsLock.RLock()
	levels = store.PowerLevels[roomID]
	if levels != nil && levels.CreateEvent == nil {
		levels.CreateEvent = store.Create[roomID]
	}
	store.powerLevelsLock.RUnlock()
	return
}

func (store *MemoryStateStore) GetPowerLevel(ctx context.Context, roomID id.RoomID, userID id.UserID) (int, error) {
	return exerrors.Must(store.GetPowerLevels(ctx, roomID)).GetUserLevel(userID), nil
}

func (store *MemoryStateStore) GetPowerLevelRequirement(ctx context.Context, roomID id.RoomID, eventType event.Type) (int, error) {
	return exerrors.Must(store.GetPowerLevels(ctx, roomID)).GetEventLevel(eventType), nil
}

func (store *MemoryStateStore) HasPowerLevel(ctx context.Context, roomID id.RoomID, userID id.UserID, eventType event.Type) (bool, error) {
	return exerrors.Must(store.GetPowerLevel(ctx, roomID, userID)) >= exerrors.Must(store.GetPowerLevelRequirement(ctx, roomID, eventType)), nil
}

func (store *MemoryStateStore) SetCreate(ctx context.Context, evt *event.Event) error {
	store.powerLevelsLock.Lock()
	store.Create[evt.RoomID] = evt
	if pls, ok := store.PowerLevels[evt.RoomID]; ok && pls.CreateEvent == nil {
		pls.CreateEvent = evt
	}
	store.powerLevelsLock.Unlock()
	return nil
}

func (store *MemoryStateStore) GetCreate(ctx context.Context, roomID id.RoomID) (*event.Event, error) {
	store.powerLevelsLock.RLock()
	evt := store.Create[roomID]
	store.powerLevelsLock.RUnlock()
	return evt, nil
}

func (store *MemoryStateStore) SetEncryptionEvent(_ context.Context, roomID id.RoomID, content *event.EncryptionEventContent) error {
	store.encryptionLock.Lock()
	store.Encryption[roomID] = content
	store.encryptionLock.Unlock()
	return nil
}

func (store *MemoryStateStore) GetEncryptionEvent(_ context.Context, roomID id.RoomID) (*event.EncryptionEventContent, error) {
	store.encryptionLock.RLock()
	defer store.encryptionLock.RUnlock()
	return store.Encryption[roomID], nil
}

func (store *MemoryStateStore) SetJoinRules(ctx context.Context, roomID id.RoomID, content *event.JoinRulesEventContent) error {
	store.joinRulesLock.Lock()
	store.JoinRules[roomID] = content
	store.joinRulesLock.Unlock()
	return nil
}

func (store *MemoryStateStore) GetJoinRules(ctx context.Context, roomID id.RoomID) (*event.JoinRulesEventContent, error) {
	store.joinRulesLock.RLock()
	defer store.joinRulesLock.RUnlock()
	return store.JoinRules[roomID], nil
}

func (store *MemoryStateStore) IsEncrypted(ctx context.Context, roomID id.RoomID) (bool, error) {
	cfg, err := store.GetEncryptionEvent(ctx, roomID)
	return cfg != nil && cfg.Algorithm == id.AlgorithmMegolmV1, err
}

func (store *MemoryStateStore) FindSharedRooms(ctx context.Context, userID id.UserID) (rooms []id.RoomID, err error) {
	store.membersLock.RLock()
	defer store.membersLock.RUnlock()
	for roomID, members := range store.Members {
		if _, ok := members[userID]; ok {
			rooms = append(rooms, roomID)
		}
	}
	return rooms, nil
}
