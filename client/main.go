package main

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

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
