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
	"errors"
	"sync"

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

	pivotSessions = &sync.Map{} // ID -> Pivot
)

// Pivot - Wraps an ImplantConnection
type Pivot struct {
	ID               string
	ImplantConn      *core.ImplantConnection
	EntryImplantConn *core.ImplantConnection
	CipherCtx        *cryptography.CipherContext
	Chain            []*sliverpb.PivotPeerEnvelope
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
// connection from which we received the pivot peer envelope we need to unwrap and forward it
// note the data passed as an argument to this handler is already extracted from the most recent
// envelope so we can just straight to parsing PivotPeerEnvelope's
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
	}

	return resp
}

func pivotPeerFailureHandler(*core.ImplantConnection, []byte) *sliverpb.Envelope {

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
	pivotSession.CipherCtx = cryptography.NewCipherContext(sessionKey)
	pivotSession.ImplantConn = core.NewImplantConnection("pivot", "pivot-remote-address")
	pivotSession.EntryImplantConn = implantConn
	pivotSessions.Store(pivotSession.ID, pivotSession)

	innerEnvelope, _ := proto.Marshal(&sliverpb.Envelope{
		Type: sliverpb.MsgPivotServerKeyExchange,
		Data: uuid.FromStringOrNil(pivotSession.ID).Bytes(),
	})
	ciphertext, err := pivotSession.CipherCtx.Encrypt(innerEnvelope)
	if err != nil {
		pivotLog.Warn("Failed to encrypt pivot server key exchange response")
		return nil
	}

	responseEnvelope, err := wrapPivotPeerEnvelope(pivotSession.Chain, &sliverpb.Envelope{
		Type: sliverpb.MsgPivotOriginEnvelope,
		Data: ciphertext,
	})
	if err != nil {
		pivotLog.Warnf("Failed to wrap pivot peer envelope: %v", err)
		return nil
	}
	return responseEnvelope
}

// wrapPivotPeerEnvelope - To wrap a response envelope we iterate backwards through the chain and wrap the response
func wrapPivotPeerEnvelope(chain []*sliverpb.PivotPeerEnvelope, envelope *sliverpb.Envelope) (*sliverpb.Envelope, error) {
	pivotLog.Debugf("Wrapping pivot peer envelope ...")
	data, err := proto.Marshal(envelope)
	if err != nil {
		pivotLog.Errorf("Failed to marshal pivot peer envelope: %v", err)
		return nil, err
	}
	if len(chain) < 1 {
		return nil, errors.New("peer chain is empty")
	}

	// Get the last peer in the chain and wrap the actual message
	peer := chain[len(chain)-1]
	peerEnvelopes := &sliverpb.PivotPeerEnvelope{
		PeerID:  peer.PeerID,
		PivotID: peer.PivotID, // This only goes in the inner most envelope
		Data:    data,
	}
	if 1 < len(chain) {
		// Iterate in reverse order wrapping the envelopes
		for index := len(chain) - 2; index >= 0; index-- {
			peerData, err := proto.Marshal(peerEnvelopes)
			if err != nil {
				pivotLog.Errorf("Failed to marshal pivot peer envelope: %v", err)
				return nil, err
			}
			peerEnvelopes = &sliverpb.PivotPeerEnvelope{
				PeerID: chain[index].PeerID,
				Data:   peerData,
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

// unwrapPivotPeerEnvelope - Unwraps the nested pivot peer envelopes
func unwrapPivotPeerEnvelopes(data []byte) ([]*sliverpb.PivotPeerEnvelope, *sliverpb.Envelope, error) {
	peerEnvelope := &sliverpb.PivotPeerEnvelope{}
	err := proto.Unmarshal(data, peerEnvelope)
	if err != nil {
		pivotLog.Errorf("failed to parse outermost peer envelope")
		return nil, nil, err
	}
	pivotLog.Debugf("Outer peer: %s", peerEnvelope.Name)

	chain := []*sliverpb.PivotPeerEnvelope{peerEnvelope}
	innerEnvelope := &sliverpb.Envelope{}
	err = proto.Unmarshal(peerEnvelope.Data, innerEnvelope)
	if err != nil {
		pivotLog.Errorf("failed to parse inner peer envelope")
		return nil, nil, err
	}
	pivotLog.Debugf("Inner envelope type: %d", innerEnvelope.Type)

	// Unwrap all inner pivot peer envelopes
	depth := 0
	for innerEnvelope.Type == sliverpb.MsgPivotPeerEnvelope {
		peerEnvelope := &sliverpb.PivotPeerEnvelope{}
		err := proto.Unmarshal(innerEnvelope.Data, peerEnvelope)
		if err != nil {
			pivotLog.Errorf("failed to parse inner peer envelope at depth %d", depth)
			return nil, nil, err
		}
		chain = append(chain, peerEnvelope)
		innerEnvelope = &sliverpb.Envelope{}
		err = proto.Unmarshal(peerEnvelope.Data, innerEnvelope)
		if err != nil {
			return nil, nil, err
		}
		depth++
	}
	return chain, innerEnvelope, nil
}
