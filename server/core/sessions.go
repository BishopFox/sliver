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
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/log"

	consts "github.com/bishopfox/sliver/client/constants"
)

var (
	sessionsLog = log.NamedLogger("core", "sessions")

	// Sessions - Manages implant connections
	Sessions = &sessions{
		sessions: map[uint32]*Session{},
		mutex:    &sync.RWMutex{},
	}
	rollingSessionID = uint32(0)

	// ErrUnknownMessageType - Returned if the implant did not understand the message for
	//                         example when the command is not supported on the platform
	ErrUnknownMessageType = errors.New("unknown message type")

	// ErrImplantTimeout - The implant did not respond prior to timeout deadline
	ErrImplantTimeout = errors.New("implant timeout")
)

// Session - Represents a connection to an implant
type Session struct {
	ID                uint32
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
}

func (s *Session) LastCheckin() time.Time {
	return s.Connection.LastMessage
}

func (s *Session) IsDead() bool {
	sessionsLog.Debugf("Last checkin was %v", s.Connection.LastMessage)
	padding := time.Duration(10 * time.Second) // Arbitrary margin of error
	timePassed := time.Since(s.LastCheckin())
	reconnect := time.Duration(s.ReconnectInterval)
	pollTimeout := time.Duration(s.PollTimeout)
	if timePassed < reconnect+padding && timePassed < pollTimeout+padding {
		sessionsLog.Debugf("Last message within reconnect interval / poll timeout with padding")
		return false
	}
	if s.Connection.Transport == consts.MtlsStr {
		if time.Since(s.Connection.LastMessage) < mtls.PingInterval+padding {
			sessionsLog.Debugf("Last message within ping interval with padding")
			return false
		}
	}
	return true
}

// ToProtobuf - Get the protobuf version of the object
func (s *Session) ToProtobuf() *clientpb.Session {
	return &clientpb.Session{
		ID:                uint32(s.ID),
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
		// ConfigID:          s.ConfigID,
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
	s.Connection.Send <- &sliverpb.Envelope{
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
		return nil, ErrUnknownMessageType
	}
	return respEnvelope.Data, nil
}

// sessions - Manages the slivers, provides atomic access
type sessions struct {
	mutex    *sync.RWMutex
	sessions map[uint32]*Session
}

// All - Return a list of all sessions
func (s *sessions) All() []*Session {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	all := []*Session{}
	for _, session := range s.sessions {
		all = append(all, session)
	}
	return all
}

// Get - Get a session by ID
func (s *sessions) Get(sessionID uint32) *Session {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.sessions[sessionID]
}

// Add - Add a sliver to the hive (atomically)
func (s *sessions) Add(session *Session) *Session {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.sessions[session.ID] = session
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
	session := s.sessions[sessionID]
	if session != nil {

		// Remove any pivots associated with this session
		PivotSessions.Range(func(key, value interface{}) bool {

			return true
		})

		delete(s.sessions, sessionID)
		EventBroker.Publish(Event{
			EventType: consts.SessionClosedEvent,
			Session:   session,
		})
	}
}

func NewSession(implantConn *ImplantConnection) *Session {
	implantConn.UpdateLastMessage()
	return &Session{
		ID:         nextSessionID(),
		Connection: implantConn,
	}
}

func SessionFromImplantConnection(conn *ImplantConnection) *Session {
	Sessions.mutex.RLock()
	defer Sessions.mutex.RUnlock()
	for _, session := range Sessions.sessions {
		if session.Connection.ID == conn.ID {
			return session
		}
	}
	return nil
}

// nextSessionID - Returns an incremental nonce as an id
func nextSessionID() uint32 {
	newID := rollingSessionID + 1
	rollingSessionID++
	return newID
}

func (s *sessions) UpdateSession(session *Session) *Session {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.sessions[session.ID] = session
	EventBroker.Publish(Event{
		EventType: consts.SessionUpdateEvent,
		Session:   session,
	})
	return session
}
