package mautrix

import (
	"context"
	"errors"
	"fmt"

	"maunium.net/go/mautrix/id"
)

var _ SyncStore = (*MemorySyncStore)(nil)
var _ SyncStore = (*AccountDataStore)(nil)

// SyncStore is an interface which must be satisfied to store client data.
//
// You can either write a struct which persists this data to disk, or you can use the
// provided "MemorySyncStore" which just keeps data around in-memory which is lost on
// restarts.
type SyncStore interface {
	SaveFilterID(ctx context.Context, userID id.UserID, filterID string) error
	LoadFilterID(ctx context.Context, userID id.UserID) (string, error)
	SaveNextBatch(ctx context.Context, userID id.UserID, nextBatchToken string) error
	LoadNextBatch(ctx context.Context, userID id.UserID) (string, error)
}

// Deprecated: renamed to SyncStore
type Storer = SyncStore

// MemorySyncStore implements the Storer interface.
//
// Everything is persisted in-memory as maps. It is not safe to load/save filter IDs
// or next batch tokens on any goroutine other than the syncing goroutine: the one
// which called Client.Sync().
type MemorySyncStore struct {
	Filters   map[id.UserID]string
	NextBatch map[id.UserID]string
}

// SaveFilterID to memory.
func (s *MemorySyncStore) SaveFilterID(ctx context.Context, userID id.UserID, filterID string) error {
	s.Filters[userID] = filterID
	return nil
}

// LoadFilterID from memory.
func (s *MemorySyncStore) LoadFilterID(ctx context.Context, userID id.UserID) (string, error) {
	return s.Filters[userID], nil
}

// SaveNextBatch to memory.
func (s *MemorySyncStore) SaveNextBatch(ctx context.Context, userID id.UserID, nextBatchToken string) error {
	s.NextBatch[userID] = nextBatchToken
	return nil
}

// LoadNextBatch from memory.
func (s *MemorySyncStore) LoadNextBatch(ctx context.Context, userID id.UserID) (string, error) {
	return s.NextBatch[userID], nil
}

// NewMemorySyncStore constructs a new MemorySyncStore.
func NewMemorySyncStore() *MemorySyncStore {
	return &MemorySyncStore{
		Filters:   make(map[id.UserID]string),
		NextBatch: make(map[id.UserID]string),
	}
}

// AccountDataStore uses account data to store the next batch token, and stores the filter ID in memory
// (as filters can be safely recreated every startup).
type AccountDataStore struct {
	FilterID  string
	EventType string
	client    *Client
	nextBatch string
}

type accountData struct {
	NextBatch string `json:"next_batch"`
}

func (s *AccountDataStore) SaveFilterID(ctx context.Context, userID id.UserID, filterID string) error {
	if userID.String() != s.client.UserID.String() {
		panic("AccountDataStore must only be used with a single account")
	}
	s.FilterID = filterID
	return nil
}

func (s *AccountDataStore) LoadFilterID(ctx context.Context, userID id.UserID) (string, error) {
	if userID.String() != s.client.UserID.String() {
		panic("AccountDataStore must only be used with a single account")
	}
	return s.FilterID, nil
}

func (s *AccountDataStore) SaveNextBatch(ctx context.Context, userID id.UserID, nextBatchToken string) error {
	if userID.String() != s.client.UserID.String() {
		panic("AccountDataStore must only be used with a single account")
	} else if nextBatchToken == s.nextBatch {
		return nil
	}

	data := accountData{
		NextBatch: nextBatchToken,
	}

	err := s.client.SetAccountData(ctx, s.EventType, data)
	if err != nil {
		return fmt.Errorf("failed to save next batch token to account data: %w", err)
	} else {
		s.client.Log.Debug().
			Str("old_token", s.nextBatch).
			Str("new_token", nextBatchToken).
			Msg("Saved next batch token")
		s.nextBatch = nextBatchToken
	}
	return nil
}

func (s *AccountDataStore) LoadNextBatch(ctx context.Context, userID id.UserID) (string, error) {
	if userID.String() != s.client.UserID.String() {
		panic("AccountDataStore must only be used with a single account")
	}

	data := &accountData{}

	err := s.client.GetAccountData(ctx, s.EventType, data)
	if err != nil {
		if errors.Is(err, MNotFound) {
			s.client.Log.Debug().Msg("No next batch token found in account data")
			return "", nil
		} else {
			return "", fmt.Errorf("failed to load next batch token from account data: %w", err)
		}
	}
	s.nextBatch = data.NextBatch
	s.client.Log.Debug().Str("next_batch", data.NextBatch).Msg("Loaded next batch token from account data")

	return s.nextBatch, nil
}

// NewAccountDataStore returns a new AccountDataStore, which stores
// the next_batch token as a custom event in account data in the
// homeserver.
//
// AccountDataStore is only appropriate for bots, not appservices.
//
// The event type should be a reversed DNS name like tld.domain.sub.internal and
// must be unique for a client. The data stored in it is considered internal
// and must not be modified through outside means. You should also add a filter
// for account data changes of this event type, to avoid ending up in a sync
// loop:
//
//	filter := mautrix.Filter{
//		AccountData: mautrix.FilterPart{
//			Limit: 20,
//			NotTypes: []event.Type{
//				event.NewEventType(eventType),
//			},
//		},
//	}
//	// If you use a custom Syncer, set the filter there, not like this
//	client.Syncer.(*mautrix.DefaultSyncer).FilterJSON = &filter
//	client.Store = mautrix.NewAccountDataStore("com.example.mybot.store", client)
//	go func() {
//		err := client.Sync()
//		// don't forget to check err
//	}()
func NewAccountDataStore(eventType string, client *Client) *AccountDataStore {
	return &AccountDataStore{
		EventType: eventType,
		client:    client,
	}
}
