package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/version"
)

const (
	logFileName = "sliver-client.log"
)

func main() {
	displayVersion := flag.Bool("version", false, "print version number")
	config := flag.String("config", "", "config file")
	flag.Parse()

	if *displayVersion {
		fmt.Printf("v%s\n", version.ClientVersion)
		os.Exit(0)
	}

	if *config != "" {
		conf, err := assets.ReadConfig(*config)
		if err != nil {
			fmt.Printf("Error %s\n", err)
			os.Exit(3)
		}
		assets.SaveConfig(conf)
	}
	appDir := assets.GetRootAppDir()
	logFile := initLogging(appDir)
	defer logFile.Close()

	os.Args = os.Args[:1] // Stops grumble from complaining
	err := console.StartClientConsole()
	if err != nil {
		fmt.Printf("[!] %s\n", err)
	}
}

// Initialize logging
func initLogging(appDir string) *os.File {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	logFile, err := os.OpenFile(path.Join(appDir, logFileName), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(fmt.Sprintf("Error opening file: %s", err))
	}
	log.SetOutput(logFile)
	return logFile
}
