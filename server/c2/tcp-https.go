package c2

import (
	"crypto/tls"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

const (
	defaultHTTPTimeout = time.Second * 60
)

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

// StartHTTPSListener - Start a mutual TLS listener
func StartHTTPSListener(conf *HTTPServerConfig) *http.Server {
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
	httpsServer := &http.Server{
		Addr:         conf.Addr,
		Handler:      httpSliverRouter(),
		WriteTimeout: defaultHTTPTimeout,
		ReadTimeout:  defaultHTTPTimeout,
		IdleTimeout:  defaultHTTPTimeout,
		TLSConfig:    tlsConf,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
	}
	return httpsServer
}

// StartHTTPListener - Start a mutual TLS listener
func StartHTTPListener(conf *HTTPServerConfig) *http.Server {
	log.Printf("Starting http listener on '%s'", conf.Addr)
	httpServer := &http.Server{
		Addr:         conf.Addr,
		Handler:      httpSliverRouter(),
		WriteTimeout: defaultHTTPTimeout,
		ReadTimeout:  defaultHTTPTimeout,
		IdleTimeout:  defaultHTTPTimeout,
	}
	return httpServer
}

func httpSliverRouter() *mux.Router {
	router := mux.NewRouter()

	router.HandleFunc("/{url:.*}", indexHandler).Methods("GET")

	return router
}

func indexHandler(resp http.ResponseWriter, req *http.Request) {
	log.Printf("[http] got req")
}
