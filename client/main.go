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
	config := flag.String("config", "", "config file")
	flag.Parse()

	if *version {
		fmt.Printf("v%s\n", clientVersion)
		os.Exit(0)
	}

	if *config != "" {
		conf, err := assets.ReadConfig(*config)
		if err != nil {
			fmt.Printf("Error %s", err)
			os.Exit(1)
		}
		assets.SaveConfig(conf)
	}
	appDir := assets.GetRootAppDir()
	logFile := initLogging(appDir)
	defer logFile.Close()

	os.Args = os.Args[:1] // Stops grumble from complaining
	console.StartClientConsole()
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
