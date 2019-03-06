package c2

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
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
	httpsServer := &http.Server{
		Addr:         conf.Addr,
		Handler:      httpSliverRouter(server),
		WriteTimeout: defaultHTTPTimeout,
		ReadTimeout:  defaultHTTPTimeout,
		IdleTimeout:  defaultHTTPTimeout,
		TLSConfig:    tlsConf,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
	}
	server.HTTPServer = httpsServer
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
	httpServer := &http.Server{
		Addr:         conf.Addr,
		Handler:      httpSliverRouter(server),
		WriteTimeout: defaultHTTPTimeout,
		ReadTimeout:  defaultHTTPTimeout,
		IdleTimeout:  defaultHTTPTimeout,
	}
	server.HTTPServer = httpServer
	return server
}

func httpSliverRouter(server *SliverHTTPC2) *mux.Router {
	router := mux.NewRouter()

	// This is a *functional* implementation, we'll obfuscate later
	router.HandleFunc("/rsakey", server.rsaKeyHandler).Methods("GET")
	router.HandleFunc("/start", server.startSessionHandler).Methods("POST")
	router.HandleFunc("/session", server.sessionHandler).Methods("POST")
	router.HandleFunc("/poll", server.pollHandler).Methods("GET")
	router.HandleFunc("/stop", server.stopHandler).Methods("POST")

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

}

func (s *SliverHTTPC2) sessionHandler(resp http.ResponseWriter, req *http.Request) {

}

func (s *SliverHTTPC2) pollHandler(resp http.ResponseWriter, req *http.Request) {

}

func (s *SliverHTTPC2) stopHandler(resp http.ResponseWriter, req *http.Request) {

}
