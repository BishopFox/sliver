package certs

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
	"os"
	"path"

	"github.com/bishopfox/sliver/server/log"

	"golang.org/x/crypto/acme/autocert"
)

const (
	// ACMEDirName - Name of dir to store ACME certs
	ACMEDirName = "acme"
)

var (
	acmeLog = log.NamedLogger("certs", "acme")
)

// GetACMEDir - Dir to store ACME certs
func GetACMEDir() string {
	acmePath := path.Join(getCertDir(), ACMEDirName)
	if _, err := os.Stat(acmePath); os.IsNotExist(err) {
		acmeLog.Infof("[mkdir] %s", acmePath)
		os.MkdirAll(acmePath, os.ModePerm)
	}
	return acmePath
}

// GetACMEManager - Get an ACME cert/tls config with the certs
func GetACMEManager(domain string) *autocert.Manager {
	acmeDir := GetACMEDir()
	return &autocert.Manager{
		Cache:  autocert.DirCache(acmeDir),
		Prompt: autocert.AcceptTOS,
	}
}
