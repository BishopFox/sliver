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
	------------------------------------------------------------------------

	WARNING: These functions can be invoked by remote implants without user interaction

	------------------------------------------------------------------------

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
	pivot := pivotEntry.(*core.Pivot)
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

func handlePivotEnvelope(pivot *core.Pivot, envelope *sliverpb.Envelope) {
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
	} else if envelope.Type == sliverpb.MsgPivotServerPing {
		pivotServerPingHandler(pivot)
	} else {
		pivotLog.Errorf("no pivot handler for envelope type %v", envelope.Type)
	}
}

func pivotPeerFailureHandler(implantConn *core.ImplantConnection, data []byte) *sliverpb.Envelope {
	peerFailure := &sliverpb.PivotPeerFailure{}
	err := proto.Unmarshal(data, peerFailure)
	if err != nil {
		pivotLog.Errorf("failed to parse peer failure message: %v", err)
		return nil
	}
	pivotLog.Errorf("pivot peer failure received: %v", peerFailure)

	core.PivotSessions.Range(func(key, value interface{}) bool {
		pivot := value.(*core.Pivot)

		found := pivot.OriginID == peerFailure.PeerID

		if !found {
			pivotLog.Warnf("Filed peer not found by OriginID, searching by Peers instead")

			for _, peer := range pivot.Peers {
				if peer.PeerID == peerFailure.PeerID {
					pivotLog.Warnf("Found session with needed peer!")
					found = true
				}
			}
		}

		if found {
			session := core.Sessions.FromImplantConnection(pivot.ImplantConn)

			if session != nil {
				core.Sessions.Remove(session.ID)
			}
			defer core.PivotSessions.Delete(pivot.ID)
			return false
		}
		return true
	})
	return nil
}

func pivotServerPingHandler(pivot *core.Pivot) {
	pivot.ImplantConn.UpdateLastMessage()
	pivot.ImmediateImplantConn.UpdateLastMessage()
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
	implantBuild, err := db.ImplantBuildByPublicKeyDigest(publicKeyDigest)
	if err != nil || implantBuild == nil {
		pivotLog.Warn("Unknown public key digest")
		return nil
	}

	serverKeyPair := cryptography.AgeServerKeyPair()
	rawSessionKey, err := cryptography.AgeKeyExFromImplant(
		serverKeyPair.Private,
		implantBuild.PeerPrivateKey,
		serverKeyEx.SessionKey[32:],
	)
	if err != nil {
		pivotLog.Warn("Failed to decrypt session key from origin")
		return nil
	}
	sessionKey, err := cryptography.KeyFromBytes(rawSessionKey)
	if err != nil {
		pivotLog.Warn("Failed to create session key from bytes")
		return nil
	}
	pivotSession := core.NewPivotSession(peerEnvelope.Peers)
	pivotSession.OriginID = peerEnvelope.Peers[0].PeerID
	pivotSession.CipherCtx = cryptography.NewCipherContext(sessionKey)

	pivotLog.Infof("Pivot session %s created with origin %s and OriginID: %v", pivotSession.ID, peerEnvelope.Peers[0].Name, pivotSession.OriginID)
	pivotLog.Infof("Peers: %v", peerEnvelope.Peers)

	pivotRemoteAddr := peersToString(implantConn.RemoteAddress, peerEnvelope)

	pivotSession.ImplantConn = core.NewImplantConnection(core.PivotTransportName, pivotRemoteAddr)
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

func peersToString(remoteAddr string, peerEnvelope *sliverpb.PivotPeerEnvelope) string {
	pivotRemoteAddr := remoteAddr
	if 1 < len(peerEnvelope.Peers) {
		for index := len(peerEnvelope.Peers) - 1; 0 < index; index-- {
			pivotRemoteAddr += fmt.Sprintf("->%s", peerEnvelope.Peers[index].Name)
		}
	}
	return pivotRemoteAddr + "->"
}
