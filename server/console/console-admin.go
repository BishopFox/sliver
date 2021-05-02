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
	"os"
	"path/filepath"
	"regexp"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/constants"
	clientLog "github.com/bishopfox/sliver/client/log"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/log"
	"github.com/bishopfox/sliver/server/transport"
)

var (
	adminLog = log.NamedLogger("console", "admin")

	namePattern = regexp.MustCompile("^[a-zA-Z0-9_]*$") // Only allow alphanumeric chars
)

// NewOperator - Command for creating a new operator user.
type NewOperator struct {
	Options OperatorOptions `group:"operator options"`
}

// OperatorOptions - Options for operator creation.
type OperatorOptions struct {
	Operator string `long:"operator" short:"o" description:"name of operator" required:"true"`
	LHost    string `long:"lhost" short:"l" description:"server listening host" default:"localhost"`
	LPort    uint16 `long:"lport" short:"p" description:"server listening port" default:"31337"`
	Save     string `long:"save" short:"s" description:"directory/file to save configuration file"`
}

// Execute - Create new operator user.
func (n NewOperator) Execute(args []string) (err error) {
	cliLog := clientLog.ClientLogger

	var save string
	if n.Options.Save == "" {
		save, _ = os.Getwd()
	}

	cliLog.Debugf("Generating new client certificate for user %s (server at %s:%d)",
		n.Options.Operator, n.Options.LHost, n.Options.LPort)
	configJSON, err := NewPlayerConfig(n.Options.Operator, n.Options.LHost, n.Options.LPort)
	if err != nil {
		fmt.Printf(Error+"Failed to generate user config: %v \n", err)
		return
	}

	saveTo, _ := filepath.Abs(save)
	fi, err := os.Stat(saveTo)
	if !os.IsNotExist(err) && !fi.IsDir() {
		fmt.Printf(Error+"Failed to generate user config: file already exists (%s) ", saveTo)
		return
	}
	if !os.IsNotExist(err) && fi.IsDir() {
		filename := fmt.Sprintf("%s_%s.cfg", filepath.Base(n.Options.Operator), filepath.Base(n.Options.LHost))
		saveTo = filepath.Join(saveTo, filename)
	}
	err = ioutil.WriteFile(saveTo, configJSON, 0600)
	if err != nil {
		fmt.Printf(Error+"Failed to write config to: %s (%v) \n", saveTo, err)
		return
	}
	cliLog.Debugf("Saved new client config to: %s \n", saveTo)
	fmt.Printf(Info+"Created player config for operator %s at %s:%d \n",
		n.Options.Operator, n.Options.LHost, n.Options.LPort)

	return
}

// KickOperator - Kick an operator out of server and remove certificates.
type KickOperator struct {
	Positional struct {
		Operator string `description:"name of operator to kick off"`
	} `positional-args:"yes"`
}

// Execute - Kick operator from server.
func (k *KickOperator) Execute(args []string) (err error) {

	operator := k.Positional.Operator

	if !namePattern.MatchString(operator) {
		fmt.Printf(Error + "Invalid operator name (alphanumerics only)")
		return
	}
	if operator == "" {
		fmt.Printf(Error + "Operator name required (--operator)")
		return
	}

	adminLog.Infof("Removing client certificate for operator %s, please wait ... \n", operator)
	err = certs.OperatorClientRemoveCertificate(operator)
	if err != nil {
		err = fmt.Errorf("failed to remove the operator certificate: %v", err)
		adminLog.Errorf(err.Error())
		fmt.Printf(Error+"Failed to kick player: %v \n", err)
		return
	}
	adminLog.Infof("Operator %s kicked out. \n", operator)
	fmt.Printf(Info+"Kicked player %s from the server \n", k.Positional.Operator)

	return
}

// MultiplayerMode - Enable team playing on server
type MultiplayerMode struct {
	Options MultiplayerOptions `group:"multiplayer options"`
}

// MultiplayerOptions - Available to server multiplayer mode.
type MultiplayerOptions struct {
	LHost string `long:"lhost" short:"l" description:"server listening host" default:"localhost"`
	LPort uint16 `long:"lport" short:"p" description:"server listening port" default:"31337"`
}

// Execute - Start multiplayer mode.
func (m *MultiplayerMode) Execute(args []string) (err error) {

	_, err = jobStartClientListener(m.Options.LHost, m.Options.LPort)
	if err == nil {
		// Temporary: increase jobs counter
		adminLog.Infof("Multiplayer mode enabled on %s:%d", m.Options.LHost, m.Options.LPort)
		fmt.Printf(Info+"Started Multiplayer gRPC listener at %s:%d \n", m.Options.LHost, m.Options.LPort)
		return
	}

	fmt.Printf(Error+"Failed to start gRPC client listener: %v \n", err)
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
		return nil, fmt.Errorf("Failed to generate certificate %s", err)
	}

	caCertPEM, _, _ := certs.GetCertificateAuthorityPEM(certs.OperatorCA)
	// caCertPEM, serverCAKey, _ := certs.GetCertificateAuthorityPEM(certs.OperatorCA)

	// Make a fingerprint of the implant's private key, for SSH-layer authentication
	// signer, _ := ssh.ParsePrivateKey(serverCAKey)
	// keyBytes := sha256.Sum256(signer.PublicKey().Marshal())
	// fingerprint := base64.StdEncoding.EncodeToString(keyBytes[:])

	config := assets.ClientConfig{
		Operator:      operatorName,
		LHost:         lhost,
		LPort:         int(lport),
		CACertificate: string(caCertPEM),
		PrivateKey:    string(privateKey),
		Certificate:   string(publicKey),
		// ServerFingerprint: fingerprint,
	}
	return json.Marshal(config)
}

// Start the console client listener
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
		adminLog.Printf("Stopping client listener (%d) ...\n", job.ID)
		ln.Close() // Kills listener GoRoutines in startMutualTLSListener() but NOT connections

		core.Jobs.Remove(job)

		core.EventBroker.Publish(core.Event{
			Job:       job,
			EventType: constants.JobStoppedEvent,
		})
	}()

	core.Jobs.Add(job)
	return job.ID, nil
}
