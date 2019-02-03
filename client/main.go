package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/user"
	"path"
)

const (
	operator  = `{{.Operator}}`
	keyPEM    = `{{.Key}}`
	certPEM   = `{{.Cert}}`
	caCertPEM = `{{.CACert}}`

	defaultServerIP = `{{.DefaultServer}}`

	logFileName = ".sliver-client.log"
)

var (
	clientVersion = "0.0.1"
)

func main() {
	version := flag.Bool("version", false, "print version number")
	flag.Parse()

	if *version {
		fmt.Printf("v%s\n", clientVersion)
		os.Exit(0)
	}

	logFile := initLogging()
	defer logFile.Close()

	startConsole()
}

// Initialize logging
func initLogging() *os.File {
	user, err := user.Current()
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	logFile, err := os.OpenFile(path.Join(user.HomeDir, logFileName), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
	}
	log.SetOutput(logFile)
	return logFile
}
