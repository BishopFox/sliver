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

	"github.com/bishopfox/sliver/implant/sliver/transports/mtls"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"

	consts "github.com/bishopfox/sliver/client/constants"
)

var (
	// Sessions - Manages implant connections
	Sessions = &sessions{
		sessions: map[uint32]*Session{},
		mutex:    &sync.RWMutex{},
	}
	rollingSessionID = uint32(0)

	// ErrUnknownMessageType - Returned if the implant did not understand the message for
	//                         example when the command is not supported on the platform
	ErrUnknownMessageType = errors.New("Unknown message type")

	// ErrImplantTimeout - The implant did not respond prior to timeout deadline
	ErrImplantTimeout = errors.New("Implant timeout")
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
	Os                string
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
}

func (s *Session) LastCheckin() time.Time {
	return s.Connection.LastMessage
}

func (s *Session) IsDead() bool {
	padding := time.Duration(10 * time.Second)
	timePassed := time.Duration(int64(math.Abs(s.LastCheckin().Sub(time.Now()).Seconds())))
	if timePassed < time.Duration(s.ReconnectInterval)+padding && timePassed < time.Duration(s.PollTimeout)+padding {
		return false
	}
	if time.Now().Sub(s.Connection.LastMessage) < mtls.PingInterval+padding {
		return false
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
		OS:                s.Os,
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
		PollInterval:      s.PollTimeout,
		Burned:            s.Burned,
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
