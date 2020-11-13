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
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"

	"github.com/bishopfox/sliver/client/assets"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/transport"
)

var (
	namePattern = regexp.MustCompile("^[a-zA-Z0-9_]*$") // Only allow alphanumeric chars
)

// NewOperator - Command for creating a new operator user.
type NewOperator struct {
	Options OperatorOptions `group:"Operator options"`
}

// OperatorOptions - Options for operator creation.
type OperatorOptions struct {
	Operator string `long:"operator" description:"Name of operator" required:"true"`
	LHost    string `long:"lhost" description:"Server listening host (default: localhost)" default:"localhost"`
	LPort    int    `long:"lport" description:"Server listening port (default: 31337)" default:"31337"`
	Save     string `long:"save" description:"Directory/file to save configuration file"`
}

// Execute - Create new operator user.
func (n NewOperator) Execute(args []string) (err error) {

	if n.Options.Save == "" {
		n.Options.Save, _ = os.Getwd()
	}

	fmt.Printf(util.Info + "Generating new client certificate, please wait ... \n")
	configJSON, err := NewPlayerConfig(n.Options.Operator, n.Options.LHost, uint16(n.Options.LPort))
	if err != nil {
		fmt.Printf(util.Warn+"%s", err)
		return
	}

	saveTo, _ := filepath.Abs(n.Options.Save)
	fi, err := os.Stat(saveTo)
	if !os.IsNotExist(err) && !fi.IsDir() {
		fmt.Printf(util.Warn+"File already exists %v\n", err)
		return
	}
	if !os.IsNotExist(err) && fi.IsDir() {
		filename := fmt.Sprintf("%s_%s.cfg", filepath.Base(n.Options.Operator), filepath.Base(n.Options.LHost))
		saveTo = filepath.Join(saveTo, filename)
	}
	err = ioutil.WriteFile(saveTo, configJSON, 0600)
	if err != nil {
		fmt.Printf(util.Warn+"Failed to write config to: %s (%v) \n", saveTo, err)
		return
	}
	fmt.Printf(util.Info+"Saved new client config to: %s \n", saveTo)

	return
}

// NewPlayerConfig - Generate a new player/client/operator configuration
func NewPlayerConfig(operatorName, lhost string, lport uint16) ([]byte, error) {

	if !namePattern.MatchString(operatorName) {
		return nil, errors.New("Invalid operator name (alphanumerics only)")
	}

	if operatorName == "" {
		return nil, errors.New("Operator name required")
	}

	if lhost == "" {
		return nil, errors.New("Invalid lhost")
	}

	publicKey, privateKey, err := certs.OperatorClientGenerateCertificate(operatorName)
	if err != nil {
		return nil, fmt.Errorf(util.Warn+"Failed to generate certificate %s", err)
	}
	caCertPEM, _, _ := certs.GetCertificateAuthorityPEM(certs.OperatorCA)
	config := assets.ClientConfig{
		Operator:      operatorName,
		LHost:         lhost,
		LPort:         int(lport),
		CACertificate: string(caCertPEM),
		PrivateKey:    string(privateKey),
		Certificate:   string(publicKey),
	}
	return json.Marshal(config)
}

// KickOperator - Kick an operator out of server and remove certificates.
type KickOperator struct {
	Positional struct {
		Operator string `description:"Name of operator to kick off"`
	} `positional-args:"yes"`
}

// Execute - Kick operator from server.
func (k *KickOperator) Execute(args []string) (err error) {

	operator := k.Positional.Operator

	if !namePattern.MatchString(operator) {
		fmt.Println(util.Warn + "Invalid operator name (alphanumerics only)")
		return
	}

	if operator == "" {
		fmt.Printf(util.Warn + "Operator name required (--operator) \n")
		return
	}
	fmt.Printf(util.Info+"Removing client certificate for operator %s, please wait ... \n", operator)
	err = certs.OperatorClientRemoveCertificate(operator)
	if err != nil {
		fmt.Printf(util.Warn+"Failed to remove the operator certificate: %v \n", err)
		return
	}
	fmt.Printf(util.Info+"Operator %s kicked out. \n", operator)

	return
}

// MultiplayerMode - Enable team playing on server
type MultiplayerMode struct {
	Options MultiplayerOptions `group:"Multiplayer options"`
}

// MultiplayerOptions - Available to server multiplayer mode.
type MultiplayerOptions struct {
	LHost string `long:"lhost" description:"Server listening host (default: localhost)" default:"localhost"`
	LPort int    `long:"lport" description:"Server listening port (default: 31337)" default:"31337"`
}

// Execute - Start multiplayer mode.
func (m *MultiplayerMode) Execute(args []string) (err error) {

	_, err = jobStartClientListener(m.Options.LHost, uint16(m.Options.LPort))
	if err == nil {
		fmt.Printf(util.Info + "Multiplayer mode enabled!\n")
	} else {
		fmt.Printf(util.Warn+"Failed to start job %v\n", err)
	}
	return
}

func jobStartClientListener(host string, port uint16) (int, error) {
	_, ln, err := transport.StartClientListener(host, port)
	if err != nil {
		return -1, err // If we fail to bind don't setup the Job
	}

	job := &core.Job{
		ID:          core.NextJobID(),
		Name:        "grpc",
		Description: "client listener",
		Protocol:    "tcp",
		Port:        port,
		JobCtrl:     make(chan bool),
	}

	go func() {
		<-job.JobCtrl
		log.Printf("Stopping client listener (%d) ...\n", job.ID)
		ln.Close() // Kills listener GoRoutines in startMutualTLSListener() but NOT connections

		core.Jobs.Remove(job)

		core.EventBroker.Publish(core.Event{
			Job:       job,
			EventType: consts.JobStoppedEvent,
		})
	}()

	core.Jobs.Add(job)
	return job.ID, nil
}
