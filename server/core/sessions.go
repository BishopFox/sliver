package core

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"errors"
	"math"
	"sync"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"

	consts "github.com/bishopfox/sliver/client/constants"
)

var (
	// Sessions - Manages implant connections
	Sessions = &sessions{
		sessions: &map[uint32]*Session{},
		mutex:    &sync.RWMutex{},
	}
	hiveID = new(uint32)

	// ErrUnknownMessateType - Returned if the implant did not understand the message for
	//                         example when the command is not supported on the platform
	ErrUnknownMessateType = errors.New("Unknown message type")

	// ErrImplantTimeout - The implant did not respond prior to timeout deadline
	ErrImplantTimeout = errors.New("Implant timeout")
)

// Session - Represents a connection to an implant
type Session struct {
	ID                uint32
	Name              string
	Hostname          string
	Username          string
	UID               string
	GID               string
	Os                string
	Version           string
	Arch              string
	Transport         string
	RemoteAddress     string
	PID               int32
	Filename          string
	LastCheckin       *time.Time
	Send              chan *sliverpb.Envelope
	Resp              map[uint64]chan *sliverpb.Envelope
	RespMutex         *sync.RWMutex
	ActiveC2          string
	IsDead            bool
	ReconnectInterval uint32
}

// ToProtobuf - Get the protobuf version of the object
func (s *Session) ToProtobuf() *clientpb.Session {
	var (
		lastCheckin string
		isDead      bool
	)
	if s.LastCheckin != nil {
		lastCheckin = s.LastCheckin.Format(time.RFC1123)

		// Calculates how much time has passed in seconds and compares that to the ReconnectInterval+10 of the Implant.
		// (ReconnectInterval+10 seconds is just abitrary padding to account for potential delays)
		// If it hasn't checked in, flag it as DEAD.
		var timePassed = uint32(math.Abs(s.LastCheckin.Sub(time.Now()).Seconds()))

		if timePassed > (s.ReconnectInterval + 10) {
			isDead = true
		} else {
			isDead = false
		}
	}

	return &clientpb.Session{
		ID:                uint32(s.ID),
		Name:              s.Name,
		Hostname:          s.Hostname,
		Username:          s.Username,
		UID:               s.UID,
		GID:               s.GID,
		OS:                s.Os,
		Version:           s.Version,
		Arch:              s.Arch,
		Transport:         s.Transport,
		RemoteAddress:     s.RemoteAddress,
		PID:               int32(s.PID),
		Filename:          s.Filename,
		LastCheckin:       lastCheckin,
		ActiveC2:          s.ActiveC2,
		IsDead:            isDead,
		ReconnectInterval: s.ReconnectInterval,
	}
}

// Request - Sends a protobuf request to the active sliver and returns the response
func (s *Session) Request(msgType uint32, timeout time.Duration, data []byte) ([]byte, error) {

	resp := make(chan *sliverpb.Envelope)
	reqID := EnvelopeID()
	s.RespMutex.Lock()
	s.Resp[reqID] = resp
	s.RespMutex.Unlock()
	defer func() {
		s.RespMutex.Lock()
		defer s.RespMutex.Unlock()
		// close(resp)
		delete(s.Resp, reqID)
	}()
	s.Send <- &sliverpb.Envelope{
		ID:   reqID,
		Type: msgType,
		Data: data,
	}

	var respEnvelope *sliverpb.Envelope
	select {
	case respEnvelope = <-resp:
	case <-time.After(timeout):
		return nil, ErrImplantTimeout
	}
	if respEnvelope.UnknownMessageType {
		return nil, ErrUnknownMessateType
	}
	s.UpdateCheckin()
	return respEnvelope.Data, nil
}

func (s *Session) UpdateCheckin() {
	now := time.Now()
	s.LastCheckin = &now
}

// sessions - Manages the slivers, provides atomic access
type sessions struct {
	mutex    *sync.RWMutex
	sessions *map[uint32]*Session
}

// All - Return a list of all sessions
func (s *sessions) All() []*Session {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	all := []*Session{}
	for _, session := range *s.sessions {
		all = append(all, session)
	}
	return all
}

// Get - Get a session by ID
func (s *sessions) Get(sessionID uint32) *Session {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return (*s.sessions)[sessionID]
}

// Add - Add a sliver to the hive (atomically)
func (s *sessions) Add(session *Session) *Session {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	(*s.sessions)[session.ID] = session
	EventBroker.Publish(Event{
		EventType: consts.SessionOpenedEvent,
		Session:   session,
	})
	return session
}

// Remove - Remove a sliver from the hive (atomically)
func (s *sessions) Remove(sessionID uint32) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	session := (*s.sessions)[sessionID]
	if session != nil {
		delete((*s.sessions), sessionID)
		EventBroker.Publish(Event{
			EventType: consts.SessionClosedEvent,
			Session:   session,
		})
	}
}

// NextSessionID - Returns an incremental nonce as an id
func NextSessionID() uint32 {
	newID := (*hiveID) + 1
	(*hiveID)++
	return newID
}

func (s *sessions) UpdateSession(session *Session) *Session {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	(*s.sessions)[session.ID] = session
	EventBroker.Publish(Event{
		EventType: consts.SessionUpdateEvent,
		Session:   session,
	})
	return session
}
