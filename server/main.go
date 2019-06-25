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

	"github.com/bishopfox/sliver/client/version"
	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/console"
)

var (
	sliverServerVersion = fmt.Sprintf("Client v%s\nServer v0.0.7", version.FullVersion())
)

const (
	logFileName = "console.log"
)

func main() {
	unpack := flag.Bool("unpack", false, "force unpack assets")
	version := flag.Bool("version", false, "print version number")
	flag.Parse()

	if *version {
		fmt.Printf("%s\n", sliverServerVersion)
		os.Exit(0)
	}

	assets.Setup(*unpack)
	if *unpack {
		os.Exit(0)
	}

	appDir := assets.GetRootAppDir()
	logFile := initLogging(appDir)
	defer logFile.Close()

	certs.SetupCAs()
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
