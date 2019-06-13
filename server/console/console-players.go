package console

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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/transport"

	"github.com/desertbit/grumble"
)

const (
	// ANSI Colors
	normal    = "\033[0m"
	black     = "\033[30m"
	red       = "\033[31m"
	green     = "\033[32m"
	orange    = "\033[33m"
	blue      = "\033[34m"
	purple    = "\033[35m"
	cyan      = "\033[36m"
	gray      = "\033[37m"
	bold      = "\033[1m"
	clearln   = "\r\x1b[2K"
	upN       = "\033[%dA"
	downN     = "\033[%dB"
	underline = "\033[4m"

	// Info - Display colorful information
	Info = bold + cyan + "[*] " + normal
	// Warn - Warn a user
	Warn = bold + red + "[!] " + normal
	// Debug - Display debug information
	Debug = bold + purple + "[-] " + normal
	// Woot - Display success
	Woot = bold + green + "[$] " + normal
)

// ClientConfig - Client JSON config
type ClientConfig struct {
	Operator      string `json:"operator"`
	LHost         string `json:"lhost"`
	LPort         int    `json:"lport"`
	CACertificate string `json:"ca_certificate"`
	PrivateKey    string `json:"private_key"`
	Certificate   string `json:"certificate"`
}

func newPlayerCmd(ctx *grumble.Context) {
	operator := ctx.Flags.String("operator")
	lhost := ctx.Flags.String("lhost")
	lport := ctx.Flags.Int("lport")
	save := ctx.Flags.String("save")

	regex, _ := regexp.Compile("[^A-Za-z0-9]+") // Only allow alphanumeric chars
	operator = regex.ReplaceAllString(operator, "")

	if operator == "" {
		fmt.Printf(Warn + "Operator name required (--operator) \n")
		return
	}

	if lhost == "" {
		fmt.Printf(Warn + "Missing lhost (--lhost) \n")
		return
	}

	if save == "" {
		save, _ = os.Getwd()
	}

	fmt.Printf(Info + "Generating new client certificate, please wait ... \n")
	publicKey, privateKey, err := certs.OperatorClientGenerateCertificate(operator)
	if err != nil {
		fmt.Printf(Warn+"Failed to generate certificate %s", err)
		return
	}
	caCertPEM, _, _ := certs.GetCertificateAuthorityPEM(certs.OperatorCA)
	config := ClientConfig{
		Operator:      operator,
		LHost:         lhost,
		LPort:         lport,
		CACertificate: string(caCertPEM),
		PrivateKey:    string(privateKey),
		Certificate:   string(publicKey),
	}
	configJSON, _ := json.Marshal(config)
	saveTo, _ := filepath.Abs(save)
	fi, err := os.Stat(saveTo)
	if err != nil {
		fmt.Printf(Warn+"Failed to generate sliver %v\n", err)
		return
	}
	if fi.IsDir() {
		filename := fmt.Sprintf("%s_%s.cfg", filepath.Base(operator), filepath.Base(lhost))
		saveTo = filepath.Join(saveTo, filename)
	}
	err = ioutil.WriteFile(saveTo, configJSON, 0600)
	if err != nil {
		fmt.Printf(Warn+"Failed to write config to: %s (%v) \n", saveTo, err)
		return
	}
	fmt.Printf(Info+"Saved new client config to: %s \n", saveTo)
}

func kickPlayerCmd(ctx *grumble.Context) {

}

func startMultiplayerModeCmd(ctx *grumble.Context) {
	server := ctx.Flags.String("server")
	lport := uint16(ctx.Flags.Int("lport"))

	_, err := jobStartClientListener(server, lport)
	if err == nil {
		fmt.Printf(Info + "Multiplayer mode enabled!\n")
	} else {
		fmt.Printf(Warn+"Failed to start job %v\n", err)
	}
}

func jobStartClientListener(bindIface string, port uint16) (int, error) {
	ln, err := transport.StartClientListener(bindIface, port)
	if err != nil {
		return -1, err // If we fail to bind don't setup the Job
	}

	job := &core.Job{
		ID:          core.GetJobID(),
		Name:        "rpc",
		Description: "client listener",
		Protocol:    "tcp",
		Port:        port,
		JobCtrl:     make(chan bool),
	}

	go func() {
		<-job.JobCtrl
		log.Printf("Stopping client listener (%d) ...\n", job.ID)
		ln.Close() // Kills listener GoRoutines in startMutualTLSListener() but NOT connections

		core.Jobs.RemoveJob(job)

		core.EventBroker.Publish(core.Event{
			Job:       job,
			EventType: consts.StoppedEvent,
		})
	}()

	core.Jobs.AddJob(job)

	return job.ID, nil
}
