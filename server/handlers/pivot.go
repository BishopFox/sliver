package handlers

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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
	---------------------------------------------------------------------

	General structure of a pivot messages:


	[Envelope].Data -> [Peer Envelope].Data -> Envelope -> C2

	The outer envelope is always a type of "Peer Envelope" and like all
	other message types is used to send a message. The Peer Envelope contains
	the routing information (a list of pivot peers) and the data to be sent.
	The Peer Envelope's data is an encrypted Envelope (the actual message).

	The Peer Envelope's data is encrypted with the server session key, peers
	can see the Peer Envelope metadata, but this is encrypted on the wire
	between peers using the peer to peer keys.

*/

import (
	"encoding/base64"
	"fmt"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/cryptography"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/log"
	"github.com/gofrs/uuid"
	"google.golang.org/protobuf/proto"
)

var (
	pivotLog = log.NamedLogger("handlers", "pivot")
)

// Pivot - Wraps an ImplantConnection
type Pivot struct {
	ID                   string
	OriginID             int64
	ImplantConn          *core.ImplantConnection
	ImmediateImplantConn *core.ImplantConnection
	CipherCtx            *cryptography.CipherContext
	Peers                []*sliverpb.PivotPeer
}

// Start - Starts the pivot send loop which forwards envelopes from the pivot ImplantConnection
// to the ImmediateImplantConnection (the closest peer in the chain)
func (p *Pivot) Start() {
	go func() {
		defer func() {
			pivotLog.Debugf("pivot session %s send loop closing (origin id: %d)", p.ID, p.OriginID)
		}()
		for envelope := range p.ImplantConn.Send {
			envelopeData, err := proto.Marshal(envelope)
			if err != nil {
				pivotLog.Errorf("failed to marshal envelope: %v", err)
				continue
			}
			ciphertext, err := p.CipherCtx.Encrypt(envelopeData)
			if err != nil {
				pivotLog.Errorf("failed to encrypt envelope: %v", err)
				continue
			}
			peerEnvelopeData, _ := proto.Marshal(&sliverpb.PivotPeerEnvelope{
				Type:  envelope.Type,
				Peers: p.Peers,
				Data:  ciphertext,
			})
			if err != nil {
				pivotLog.Errorf("failed to wrap pivot peer envelope: %v", err)
				continue
			}
			p.ImmediateImplantConn.Send <- &sliverpb.Envelope{
				Type: sliverpb.MsgPivotPeerEnvelope,
				Data: peerEnvelopeData,
			}
		}
	}()
}

// NewPivotSession - Creates a new pivot session
func NewPivotSession(chain []*sliverpb.PivotPeer) *Pivot {
	id, _ := uuid.NewV4()
	return &Pivot{
		ID:    id.String(),
		Peers: chain,
	}
}

// ------------------------
// Handler functions
// ------------------------
// pivotPeerEnvelopeHandler - Ingress point for any pivot traffic, the `implantConn` here is the
// connection from which we received the pivot peer envelope we need to unwrap and forward it.
// NOTE: the data passed as an argument to this handler is already extracted from the most recent
// envelope so we can just start parsing PivotPeerEnvelope's
func pivotPeerEnvelopeHandler(implantConn *core.ImplantConnection, data []byte) *sliverpb.Envelope {
	pivotLog.Debugf("received pivot peer envelope ...")
	peerEnvelope := &sliverpb.PivotPeerEnvelope{}
	err := proto.Unmarshal(data, peerEnvelope)
	if err != nil {
		pivotLog.Errorf("failed to parse outermost peer envelope")
		return nil
	}

	var resp *sliverpb.Envelope
	switch peerEnvelope.Type {
	case sliverpb.MsgPivotServerKeyExchange:
		resp = serverKeyExchange(implantConn, peerEnvelope)
	case sliverpb.MsgPivotSessionEnvelope:
		resp = sessionEnvelopeHandler(implantConn, peerEnvelope)
	case sliverpb.MsgPivotPeerFailure:
		resp = peerFailureHandler(implantConn, peerEnvelope)
	case sliverpb.MsgPivotServerPing:
		resp = serverPingHandler(implantConn, peerEnvelope)
	}

	return resp
}

func sessionEnvelopeHandler(implantConn *core.ImplantConnection, peerEnvelope *sliverpb.PivotPeerEnvelope) *sliverpb.Envelope {
	pivotSessionID := uuid.FromBytesOrNil(peerEnvelope.PivotSessionID).String()
	if pivotSessionID == "" {
		pivotLog.Errorf("failed to parse pivot session id from peer envelope")
		return nil
	}
	pivotLog.Debugf("session envelope pivot session ID = %s", pivotSessionID)
	pivotEntry, ok := core.PivotSessions.Load(pivotSessionID)
	if !ok {
		pivotLog.Errorf("pivot session id '%s' not found", pivotSessionID)
		return nil
	}
	pivot := pivotEntry.(*Pivot)
	plaintext, err := pivot.CipherCtx.Decrypt(peerEnvelope.Data)
	if err != nil {
		pivotLog.Errorf("failed to decrypt pivot session data: %v", err)
		return nil
	}
	envelope := &sliverpb.Envelope{}
	err = proto.Unmarshal(plaintext, envelope)
	if err != nil {
		pivotLog.Errorf("failed to unmarshal pivot session data: %v", err)
		return nil
	}

	go handlePivotEnvelope(pivot, envelope)

	return nil
}

func handlePivotEnvelope(pivot *Pivot, envelope *sliverpb.Envelope) {
	pivotLog.Debugf("pivot session %s received envelope: %v", pivot.ID, envelope.Type)
	handlers := GetNonPivotHandlers()
	pivot.ImplantConn.UpdateLastMessage()
	if envelope.ID != 0 {
		pivot.ImplantConn.RespMutex.RLock()
		defer pivot.ImplantConn.RespMutex.RUnlock()
		if resp, ok := pivot.ImplantConn.Resp[envelope.ID]; ok {
			resp <- envelope
		}
	} else if handler, ok := handlers[envelope.Type]; ok {
		respEnvelope := handler(pivot.ImplantConn, envelope.Data)
		if respEnvelope != nil {
			go func() {
				pivot.ImplantConn.Send <- respEnvelope
			}()
		}
	} else {
		pivotLog.Errorf("no pivot handler for envelope type %v", envelope.Type)
	}
}

func peerFailureHandler(implantConn *core.ImplantConnection, peerEnvelope *sliverpb.PivotPeerEnvelope) *sliverpb.Envelope {
	pivotLog.Errorf("pivot peer failure received")
	return nil
}

func serverPingHandler(implantConn *core.ImplantConnection, peerEnvelope *sliverpb.PivotPeerEnvelope) *sliverpb.Envelope {
	pivotSessionID := uuid.FromBytesOrNil(peerEnvelope.PivotSessionID).String()
	if pivotSessionID == "" {
		pivotLog.Errorf("failed to parse pivot session id from peer envelope")
		return nil
	}

	// Find the pivot session for the server ping
	pivotLog.Debugf("origin envelope pivot session ID = %s", pivotSessionID)
	pivotEntry, ok := core.PivotSessions.Load(pivotSessionID)
	if !ok {
		pivotLog.Errorf("pivot session id '%s' not found", pivotSessionID)
		return nil
	}
	pivot := pivotEntry.(*Pivot)

	// Update last message time
	pivot.ImplantConn.UpdateLastMessage()

	return nil
}

// ------------------------
// Non-handler functions
// ------------------------
func serverKeyExchange(implantConn *core.ImplantConnection, peerEnvelope *sliverpb.PivotPeerEnvelope) *sliverpb.Envelope {
	// Normally the peer envelope data would be encrypted, but the server key exchange messages are special
	// only the session key field is encrypted (with the server's public key)
	if len(peerEnvelope.Peers) < 2 {
		pivotLog.Errorf("pivot peer list too small (%d)", len(peerEnvelope.Peers))
		return nil
	}

	serverKeyEx := &sliverpb.PivotServerKeyExchange{}
	err := proto.Unmarshal(peerEnvelope.Data, serverKeyEx)
	if err != nil {
		pivotLog.Errorf("failed to parse pivot server key exchange: %v", err)
		return nil
	}
	if len(serverKeyEx.SessionKey) < 32 {
		pivotLog.Errorf("pivot server key exchange field too small (%d)", len(serverKeyEx.SessionKey))
		return nil
	}

	// The first 32 bytes are the sha256 hash of the implant's public key
	// everything after that is the encrypted session key
	var publicKeyDigest [32]byte
	copy(publicKeyDigest[:], serverKeyEx.SessionKey[:32])
	implantConfig, err := db.ImplantConfigByECCPublicKeyDigest(publicKeyDigest)
	if err != nil || implantConfig == nil {
		pivotLog.Warn("Unknown public key digest")
		return nil
	}
	publicKey, err := base64.RawStdEncoding.DecodeString(implantConfig.ECCPublicKey)
	if err != nil || len(publicKey) != 32 {
		pivotLog.Warn("Failed to decode public key")
		return nil
	}
	var senderPublicKey [32]byte
	copy(senderPublicKey[:], publicKey)
	serverKeyPair := cryptography.ECCServerKeyPair()
	rawSessionKey, err := cryptography.ECCDecrypt(&senderPublicKey, serverKeyPair.Private, serverKeyEx.SessionKey[32:])
	if err != nil {
		pivotLog.Warn("Failed to decrypt session key from origin")
		return nil
	}
	sessionKey, err := cryptography.KeyFromBytes(rawSessionKey)
	if err != nil {
		pivotLog.Warn("Failed to create session key from bytes")
		return nil
	}
	pivotSession := NewPivotSession(peerEnvelope.Peers)
	pivotLog.Infof("Pivot session %s created with origin %s", pivotSession.ID, peerEnvelope.Peers[0].Name)
	pivotSession.OriginID = peerEnvelope.Peers[0].PeerID
	pivotSession.CipherCtx = cryptography.NewCipherContext(sessionKey)

	pivotRemoteAddr := peersToString(peerEnvelope) + implantConn.RemoteAddress

	pivotSession.ImplantConn = core.NewImplantConnection("pivot", pivotRemoteAddr)
	pivotSession.ImmediateImplantConn = implantConn
	core.PivotSessions.Store(pivotSession.ID, pivotSession)
	keyExRespEnvelope := MustMarshal(&sliverpb.Envelope{
		Type: sliverpb.MsgPivotServerKeyExchange,
		Data: MustMarshal(&sliverpb.PivotServerKeyExchange{
			SessionKey: uuid.FromStringOrNil(pivotSession.ID).Bytes(), // Re-use the bytes field
		}),
	})
	ciphertext, err := pivotSession.CipherCtx.Encrypt(keyExRespEnvelope)
	if err != nil {
		pivotLog.Warn("Failed to encrypt pivot server key exchange response")
		return nil
	}
	peerEnvelopeData, _ := proto.Marshal(&sliverpb.PivotPeerEnvelope{
		Peers: pivotSession.Peers,
		Data:  ciphertext,
	})
	if err != nil {
		pivotLog.Warn("Failed to encrypt pivot server key exchange response")
		return nil
	}

	pivotSession.Start()
	return &sliverpb.Envelope{
		Type: sliverpb.MsgPivotPeerEnvelope,
		Data: peerEnvelopeData,
	}
}

// MustMarshal - Marshals or returns an empty byte array
func MustMarshal(msg proto.Message) []byte {
	data, err := proto.Marshal(msg)
	if err != nil {
		pivotLog.Errorf("failed to marshal message: %v", err)
		return nil
	}
	return data
}

func peersToString(peerEnvelope *sliverpb.PivotPeerEnvelope) string {
	if len(peerEnvelope.Peers) < 1 {
		return ""
	}
	pivotRemoteAddr := ""
	for _, peer := range peerEnvelope.Peers[1:] {
		pivotRemoteAddr += fmt.Sprintf("%s->", peer.Name)
	}
	return pivotRemoteAddr
}
