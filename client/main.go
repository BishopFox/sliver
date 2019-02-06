package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"sliver/client/assets"
	"sliver/client/console"
)

const (
	logFileName = "sliver-client.log"
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

	appDir := assets.GetRootAppDir()
	logFile := initLogging(appDir)
	defer logFile.Close()

	console.Start()
}

// Initialize logging
func initLogging(appDir string) *os.File {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	logFile, err := os.OpenFile(path.Join(appDir, logFileName), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
	}
	log.SetOutput(logFile)
	return logFile
}
