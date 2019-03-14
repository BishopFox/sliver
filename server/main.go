package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"

	"sliver/server/assets"
	"sliver/server/console"
)

const (
	logFileName = "sliver.log"
)

var (
	sliverServerVersion = fmt.Sprintf("0.0.4 - %s", assets.GitVersion)
)

func main() {
	unpack := flag.Bool("unpack", false, "force unpack assets")
	version := flag.Bool("version", false, "print version number")
	flag.Parse()

	if *version {
		fmt.Printf("v%s\n", sliverServerVersion)
		os.Exit(0)
	}

	appDir := assets.GetRootAppDir()
	logFile := initLogging(appDir)
	defer logFile.Close()

	assets.Setup()

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
