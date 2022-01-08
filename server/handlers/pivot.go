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
*/

import (
	"encoding/base64"
	"fmt"
	"strings"

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
	Chain                []*sliverpb.PivotPeerEnvelope
}

// Start - Starts the pivot send loop which forwards envelopes from the pivot ImplantConnection
// to the ImmediateImplantConnection (the closest peer in the chain)
func (p *Pivot) Start() {
	go func() {
		defer func() {
			pivotLog.Debugf("pivot session %s send loop closing", p.ID)
		}()
		for envelope := range p.ImplantConn.Send {
			envelope, err := proto.Marshal(envelope)
			if err != nil {
				pivotLog.Errorf("failed to marshal envelope: %v", err)
				continue
			}
			ciphertext, err := p.CipherCtx.Encrypt(envelope)
			if err != nil {
				pivotLog.Errorf("failed to encrypt envelope: %v", err)
				continue
			}
			peerEnvelope, err := p.WrapPivotPeerEnvelope(ciphertext)
			if err != nil {
				pivotLog.Errorf("failed to wrap pivot peer envelope: %v", err)
				continue
			}
			p.ImmediateImplantConn.Send <- peerEnvelope
		}
	}()
}

// WrapPivotPeerEnvelope - To wrap a response envelope we iterate backwards through the chain and wrap the response
func (p *Pivot) WrapPivotPeerEnvelope(ciphertext []byte) (*sliverpb.Envelope, error) {

	pivotLog.Debugf("[wrap] response pivot peer envelope ...")

	// Get the last peer in the chain and wrap the actual message
	originEnvelope := &sliverpb.PivotPeerEnvelope{
		PeerID:      p.OriginID,
		NextMsgType: sliverpb.MsgPivotOriginEnvelope,
		Data:        ciphertext,
	}
	pivotLog.Debugf("[wrap] origin: %d", originEnvelope.PeerID)
	originData, err := proto.Marshal(originEnvelope)
	if err != nil {
		pivotLog.Errorf("Failed to marshal pivot peer envelope: %v", err)
		return nil, err
	}
	lastPeer := p.Chain[len(p.Chain)-1]
	peerEnvelopes := &sliverpb.PivotPeerEnvelope{
		PeerID:      lastPeer.PeerID,
		NextMsgType: sliverpb.MsgPivotOriginEnvelope,
		Data:        originData,
	}
	pivotLog.Debugf("[wrap] via: %d", lastPeer.PeerID)
	if 1 < len(p.Chain) {
		// Iterate in reverse order wrapping the envelopes
		for index := len(p.Chain) - 2; index >= 0; index-- {
			peerData, err := proto.Marshal(peerEnvelopes)
			if err != nil {
				pivotLog.Errorf("Failed to marshal pivot peer envelope: %v", err)
				return nil, err
			}
			pivotLog.Debugf("[wrap] via: %d", p.Chain[index].PeerID)
			peerEnvelopes = &sliverpb.PivotPeerEnvelope{
				PeerID:      p.Chain[index].PeerID,
				NextMsgType: sliverpb.MsgPivotPeerEnvelope,
				Data:        peerData,
			}
		}
	}

	peerEnvelopesData, err := proto.Marshal(peerEnvelopes)
	if err != nil {
		pivotLog.Errorf("Failed to marshal pivot peer envelope: %v", err)
		return nil, err
	}
	return &sliverpb.Envelope{
		Type: sliverpb.MsgPivotPeerEnvelope,
		Data: peerEnvelopesData,
	}, nil
}

// NewPivotSession - Creates a new pivot session
func NewPivotSession(chain []*sliverpb.PivotPeerEnvelope) *Pivot {
	id, _ := uuid.NewV4()
	return &Pivot{
		ID:    id.String(),
		Chain: chain,
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

	// Unwrap the nested envelopes
	chain, originEnvelope, err := unwrapPivotPeerEnvelopes(data)
	if err != nil {
		return nil
	}
	// Determine if the message is associated with an existing session

	for index, peer := range chain {
		pivotLog.Debugf("Chain[%d] = %v", index, peer.Name)
	}
	pivotLog.Debugf("Origin MsgType = %v", originEnvelope.Type)

	var resp *sliverpb.Envelope
	switch originEnvelope.Type {
	case sliverpb.MsgPivotServerKeyExchange:
		resp = serverKeyExchange(implantConn, chain, originEnvelope)
	case sliverpb.MsgPivotOriginEnvelope:
		resp = originEnvelopeHandler(implantConn, chain, originEnvelope)
	case sliverpb.MsgPivotPeerFailure:
		resp = pivotPeerFailureHandler(implantConn, chain, originEnvelope)
	case sliverpb.MsgPivotServerPing:
		resp = pivotServerPingHandler(implantConn, chain, originEnvelope)
	}

	return resp
}

func originEnvelopeHandler(implantConn *core.ImplantConnection, chain []*sliverpb.PivotPeerEnvelope, origin *sliverpb.Envelope) *sliverpb.Envelope {
	// There should be a uuid prepended to the data buffer, so check to make sure its long enough to parse
	if len(origin.Data) < 16 {
		pivotLog.Errorf("origin envelope data too small")
		return nil
	}
	pivotSessionID := uuid.FromBytesOrNil(origin.Data[:16]).String()
	pivotLog.Debugf("origin envelope pivot session ID = %s", pivotSessionID)
	pivotEntry, ok := core.PivotSessions.Load(pivotSessionID)
	if !ok {
		pivotLog.Errorf("pivot session id '%s' not found", pivotSessionID)
		return nil
	}
	pivot := pivotEntry.(*Pivot)
	plaintext, err := pivot.CipherCtx.Decrypt(origin.Data[16:])
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

func pivotPeerFailureHandler(implantConn *core.ImplantConnection, chain []*sliverpb.PivotPeerEnvelope, origin *sliverpb.Envelope) *sliverpb.Envelope {
	pivotLog.Errorf("pivot peer failure received")
	return nil
}

func pivotServerPingHandler(implantConn *core.ImplantConnection, chain []*sliverpb.PivotPeerEnvelope, origin *sliverpb.Envelope) *sliverpb.Envelope {
	pivotLog.Debugf("pivot server ping received")

	// Find the pivot session for the server ping
	pivotSessionID := uuid.FromBytesOrNil(origin.Data[:16]).String()
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
func serverKeyExchange(implantConn *core.ImplantConnection, chain []*sliverpb.PivotPeerEnvelope, originEnvelope *sliverpb.Envelope) *sliverpb.Envelope {
	keyExchange := &sliverpb.PivotServerKeyExchange{}
	err := proto.Unmarshal(originEnvelope.Data, keyExchange)
	if err != nil {
		pivotLog.Errorf("Un-marshaling pivot server key exchange error: %v", err)
		return nil
	}

	peerIDs := []string{}
	for _, peer := range chain {
		peerIDs = append(peerIDs, fmt.Sprintf("%d", peer.PeerID))
	}
	pivotLog.Debugf("[pivot server key exchange] Peers: %s, Origin: %d", strings.Join(peerIDs, " => "), keyExchange.OriginID)

	var publicKeyDigest [32]byte
	copy(publicKeyDigest[:], keyExchange.SessionKey[:32])
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
	rawSessionKey, err := cryptography.ECCDecrypt(&senderPublicKey, serverKeyPair.Private, keyExchange.SessionKey[32:])
	if err != nil {
		pivotLog.Warn("Failed to decrypt session key from origin")
		return nil
	}
	sessionKey, err := cryptography.KeyFromBytes(rawSessionKey)
	if err != nil {
		pivotLog.Warn("Failed to create session key from bytes")
		return nil
	}
	pivotSession := NewPivotSession(chain)
	pivotSession.OriginID = keyExchange.OriginID
	pivotSession.CipherCtx = cryptography.NewCipherContext(sessionKey)
	pivotRemoteAddr := fmt.Sprintf("%s->", chain[0].Name)
	for _, peer := range chain[1:] {
		pivotRemoteAddr += fmt.Sprintf("%s->", peer.Name)
	}
	pivotSession.ImplantConn = core.NewImplantConnection("pivot", pivotRemoteAddr)
	pivotSession.ImmediateImplantConn = implantConn
	core.PivotSessions.Store(pivotSession.ID, pivotSession)
	innerEnvelope, _ := proto.Marshal(&sliverpb.Envelope{
		Type: sliverpb.MsgPivotServerKeyExchange,
		Data: uuid.FromStringOrNil(pivotSession.ID).Bytes(),
	})
	ciphertext, err := pivotSession.CipherCtx.Encrypt(innerEnvelope)
	if err != nil {
		pivotLog.Warn("Failed to encrypt pivot server key exchange response")
		return nil
	}

	responseEnvelope, err := pivotSession.WrapPivotPeerEnvelope(ciphertext)
	if err != nil {
		pivotLog.Warnf("Failed to wrap pivot peer envelope: %v", err)
		return nil
	}
	pivotSession.Start()
	return responseEnvelope
}

// unwrapPivotPeerEnvelope - Unwraps the nested pivot peer envelopes
func unwrapPivotPeerEnvelopes(data []byte) ([]*sliverpb.PivotPeerEnvelope, *sliverpb.Envelope, error) {
	peerEnvelope := &sliverpb.PivotPeerEnvelope{}
	err := proto.Unmarshal(data, peerEnvelope)
	if err != nil {
		pivotLog.Errorf("failed to parse outermost peer envelope")
		return nil, nil, err
	}
	pivotLog.Debugf("Entry peer: %s", peerEnvelope.Name)
	pivotLog.Debugf("Next envelope type: %d", peerEnvelope.NextMsgType)

	chain := []*sliverpb.PivotPeerEnvelope{peerEnvelope}
	depth := 0
	for peerEnvelope.NextMsgType == sliverpb.MsgPivotPeerEnvelope {
		nextEnvelope := &sliverpb.PivotPeerEnvelope{}
		err := proto.Unmarshal(peerEnvelope.Data, nextEnvelope)
		if err != nil {
			pivotLog.Errorf("failed to parse inner peer envelope at depth %d", depth)
			return nil, nil, err
		}
		pivotLog.Debugf("Next peer: %s", nextEnvelope.Name)
		pivotLog.Debugf("Next envelope type: %d", nextEnvelope.NextMsgType)
		chain = append(chain, nextEnvelope)
		depth++
		peerEnvelope = nextEnvelope
	}

	return chain, &sliverpb.Envelope{
		Type: peerEnvelope.NextMsgType,
		Data: peerEnvelope.Data,
	}, nil
}
