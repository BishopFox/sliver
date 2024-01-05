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
	"sync"
	"time"

	"github.com/bishopfox/sliver/implant/sliver/transports/mtls"
	"github.com/bishopfox/sliver/implant/sliver/transports/wireguard"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/log"
	"github.com/gofrs/uuid"

	consts "github.com/bishopfox/sliver/client/constants"
)

var (
	sessionsLog = log.NamedLogger("core", "sessions")

	// Sessions - Manages implant connections
	Sessions = &sessions{
		sessions: &sync.Map{},
	}

	// ErrUnknownMessageType - Returned if the implant did not understand the message for
	//                         example when the command is not supported on the platform
	ErrUnknownMessageType = errors.New("unknown message type")

	// ErrImplantTimeout - The implant did not respond prior to timeout deadline
	ErrImplantTimeout = errors.New("implant timeout")
)

// Session - Represents a connection to an implant
type Session struct {
	ID                string
	Name              string
	Hostname          string
	Username          string
	UUID              string
	UID               string
	GID               string
	OS                string
	Version           string
	Arch              string
	PID               int32
	Filename          string
	Connection        *ImplantConnection
	ActiveC2          string
	ReconnectInterval int64
	ProxyURL          string
	PollTimeout       int64
	Burned            bool
	Extensions        []string
	ConfigID          string
	PeerID            int64
	Locale            string
	FirstContact      int64
	Integrity         string
}

// LastCheckin - Get the last time a session message was received
func (s *Session) LastCheckin() time.Time {
	return s.Connection.GetLastMessage()
}

// IsDead - See if last check-in is within expected variance
func (s *Session) IsDead() bool {
	sessionsLog.Debugf("Checking health of %s", s.ID)
	sessionsLog.Debugf("Last checkin was %v", s.LastCheckin())
	padding := time.Duration(10 * time.Second) // Arbitrary margin of error
	timePassed := time.Since(s.LastCheckin())
	reconnect := time.Duration(s.ReconnectInterval)
	pollTimeout := time.Duration(s.PollTimeout)
	if timePassed < reconnect+padding && timePassed < pollTimeout+padding {
		sessionsLog.Debugf("Last message within reconnect interval / poll timeout + padding")
		return false
	}
	if s.Connection.Transport == consts.MtlsStr {
		if timePassed < mtls.PingInterval+padding {
			sessionsLog.Debugf("Last message within ping interval with padding")
			return false
		}
	}
	if s.Connection.Transport == consts.WGStr {
		if timePassed < wireguard.PingInterval+padding {
			sessionsLog.Debugf("Last message with ping interval with padding")
			return false
		}
	}
	if s.Connection.Transport == "pivot" {
		if time.Since(s.Connection.GetLastMessage()) < time.Duration(time.Minute)+padding {
			sessionsLog.Debugf("Last message within pivot/server ping interval with padding")
			return false
		}
	}
	return true
}

// ToProtobuf - Get the protobuf version of the object
func (s *Session) ToProtobuf() *clientpb.Session {
	return &clientpb.Session{
		ID:                s.ID,
		Name:              s.Name,
		Hostname:          s.Hostname,
		Username:          s.Username,
		UUID:              s.UUID,
		UID:               s.UID,
		GID:               s.GID,
		OS:                s.OS,
		Version:           s.Version,
		Arch:              s.Arch,
		Transport:         s.Connection.Transport,
		RemoteAddress:     s.Connection.RemoteAddress,
		PID:               int32(s.PID),
		Filename:          s.Filename,
		LastCheckin:       s.LastCheckin().Unix(),
		ActiveC2:          s.ActiveC2,
		IsDead:            s.IsDead(),
		ReconnectInterval: s.ReconnectInterval,
		ProxyURL:          s.ProxyURL,
		Burned:            s.Burned,
		PeerID:            s.PeerID,
		Locale:            s.Locale,
		FirstContact:      s.FirstContact,
		Integrity:         s.Integrity,
	}
}

// Request - Sends a protobuf request to the active sliver and returns the response
func (s *Session) Request(msgType uint32, timeout time.Duration, data []byte) ([]byte, error) {
	resp := make(chan *sliverpb.Envelope)
	reqID := EnvelopeID()
	s.Connection.RespMutex.Lock()
	s.Connection.Resp[reqID] = resp
	s.Connection.RespMutex.Unlock()
	defer func() {
		s.Connection.RespMutex.Lock()
		defer s.Connection.RespMutex.Unlock()
		// close(resp)
		delete(s.Connection.Resp, reqID)
	}()
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	select {
	case s.Connection.Send <- &sliverpb.Envelope{
		ID:   reqID,
		Type: msgType,
		Data: data,
	}:
	case <-time.After(timeout):
		return nil, ErrImplantTimeout
	}

	var respEnvelope *sliverpb.Envelope
	select {
	case respEnvelope = <-resp:
	case <-time.After(timeout):
		return nil, ErrImplantTimeout
	}
	if respEnvelope.UnknownMessageType {
		return nil, ErrUnknownMessageType
	}
	return respEnvelope.Data, nil
}

// sessions - Manages the slivers, provides atomic access
type sessions struct {
	sessions *sync.Map // map[uint32]*Session
}

// All - Return a list of all sessions
func (s *sessions) All() []*Session {
	all := []*Session{}
	s.sessions.Range(func(key, value interface{}) bool {
		all = append(all, value.(*Session))
		return true
	})
	return all
}

// Get - Get a session by ID
func (s *sessions) Get(sessionID string) *Session {
	if val, ok := s.sessions.Load(sessionID); ok {
		return val.(*Session)
	}
	return nil
}

// Add - Add a sliver to the hive (atomically)
func (s *sessions) Add(session *Session) *Session {
	s.sessions.Store(session.ID, session)
	EventBroker.Publish(Event{
		EventType: consts.SessionOpenedEvent,
		Session:   session,
	})
	return session
}

// Remove - Remove a sliver from the hive (atomically)
func (s *sessions) Remove(sessionID string) {
	val, ok := s.sessions.Load(sessionID)
	if !ok {
		return
	}
	parentSession := val.(*Session)
	children := findAllChildrenByPeerID(parentSession.PeerID)
	s.sessions.Delete(parentSession.ID)
	coreLog.Debugf("Removing %d children of session %s (%v)", len(children), parentSession.ID, children)
	for _, child := range children {
		childSession, ok := s.sessions.LoadAndDelete(child.SessionID)
		if ok {
			PivotSessions.Delete(childSession.(*Session).Connection.ID)
			EventBroker.Publish(Event{
				EventType: consts.SessionClosedEvent,
				Session:   childSession.(*Session),
			})
		}
	}

	// Remove the parent session
	EventBroker.Publish(Event{
		EventType: consts.SessionClosedEvent,
		Session:   parentSession,
	})
}

// NewSession - Create a new session
func NewSession(implantConn *ImplantConnection) *Session {
	implantConn.UpdateLastMessage()
	return &Session{
		ID:           nextSessionID(),
		Connection:   implantConn,
		FirstContact: time.Now().Unix(),
		Integrity:    "-",
	}
}

// FromImplantConnection - Find the session associated with an implant connection
func (s *sessions) FromImplantConnection(conn *ImplantConnection) *Session {
	var found *Session
	s.sessions.Range(func(key, value interface{}) bool {
		if value.(*Session).Connection.ID == conn.ID {
			found = value.(*Session)
			return false
		}
		return true
	})
	return found
}

// nextSessionID - Returns an incremental nonce as an id
func nextSessionID() string {
	id, _ := uuid.NewV4()
	return id.String()
}

// UpdateSession - In place update of a session pointer
func (s *sessions) UpdateSession(session *Session) *Session {
	s.sessions.Store(session.ID, session)
	EventBroker.Publish(Event{
		EventType: consts.SessionUpdateEvent,
		Session:   session,
	})
	return session
}
