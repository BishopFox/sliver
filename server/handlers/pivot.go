package handlers

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
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/log"
	"google.golang.org/protobuf/proto"
)

var (
	pivotLog = log.NamedLogger("handlers", "pivot")
)

func pivotPeerEnvelopeHandler(implantConn *core.ImplantConnection, data []byte) *sliverpb.Envelope {
	outerEnvelope := &sliverpb.Envelope{}
	err := proto.Unmarshal(data, outerEnvelope)
	if err != nil {
		return nil
	}

	// Unwrap the nested envelopes
	chain, origin, err := unwrapPivotPeerEnvelope(outerEnvelope)
	if err != nil {
		return nil
	}
	// Determine if the message is associated with an existing session

	return nil
}

func pivotPeerFailureHandler(*core.ImplantConnection, []byte) *sliverpb.Envelope {

	return nil
}

// unwrapPivotPeerEnvelope - Unwraps the nested pivot peer envelopes
func unwrapPivotPeerEnvelope(envelope *sliverpb.Envelope) ([]*sliverpb.Envelope, *sliverpb.Envelope, error) {
	chain := []*sliverpb.Envelope{envelope}
	nextEnvelope := &sliverpb.Envelope{}
	err := proto.Unmarshal(envelope.Data, nextEnvelope)
	if err != nil {
		return nil, nil, err
	}
	for nextEnvelope.Type == sliverpb.MsgPivotPeerEnvelope {
		chain = append(chain, nextEnvelope)
		nextEnvelope = &sliverpb.Envelope{}
		err = proto.Unmarshal(envelope.Data, nextEnvelope)
		if err != nil {
			return nil, nil, err
		}
	}
	return chain, nextEnvelope, nil
}
