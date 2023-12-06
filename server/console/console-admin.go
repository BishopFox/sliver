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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"

	"github.com/spf13/cobra"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/bishopfox/sliver/server/transport"
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

var namePattern = regexp.MustCompile("^[a-zA-Z0-9_-]*$") // Only allow alphanumeric chars

// ClientConfig - Client JSON config
type ClientConfig struct {
	Operator      string `json:"operator"`
	Token         string `json:"token"`
	LHost         string `json:"lhost"`
	LPort         int    `json:"lport"`
	CACertificate string `json:"ca_certificate"`
	PrivateKey    string `json:"private_key"`
	Certificate   string `json:"certificate"`
}

func newOperatorCmd(cmd *cobra.Command, _ []string) {
	name, _ := cmd.Flags().GetString("name")
	lhost, _ := cmd.Flags().GetString("lhost")
	lport, _ := cmd.Flags().GetUint16("lport")
	save, _ := cmd.Flags().GetString("save")
	permissions, _ := cmd.Flags().GetStringSlice("permissions")

	if save == "" {
		save, _ = os.Getwd()
	}

	fmt.Printf(Info + "Generating new client certificate, please wait ... \n")
	configJSON, err := NewOperatorConfig(name, lhost, lport, permissions)
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
		return
	}

	saveTo, _ := filepath.Abs(save)
	fi, err := os.Stat(saveTo)
	if !os.IsNotExist(err) && !fi.IsDir() {
		fmt.Printf(Warn+"File already exists %s\n", err)
		return
	}
	if !os.IsNotExist(err) && fi.IsDir() {
		filename := fmt.Sprintf("%s_%s.cfg", filepath.Base(name), filepath.Base(lhost))
		saveTo = filepath.Join(saveTo, filename)
	}
	err = os.WriteFile(saveTo, configJSON, 0o600)
	if err != nil {
		fmt.Printf(Warn+"Failed to write config to: %s (%s) \n", saveTo, err)
		return
	}
	fmt.Printf(Info+"Saved new client config to: %s \n", saveTo)
}

// NewOperatorConfig - Generate a new player/client/operator configuration
func NewOperatorConfig(operatorName string, lhost string, lport uint16, permissions []string) ([]byte, error) {
	if !namePattern.MatchString(operatorName) {
		return nil, errors.New("invalid operator name (alphanumerics only)")
	}
	if operatorName == "" {
		return nil, errors.New("operator name required")
	}
	if lhost == "" {
		return nil, errors.New("invalid lhost")
	}
	if len(permissions) == 0 {
		return nil, errors.New("must specify at least one permission")
	}

	rawToken := models.GenerateOperatorToken()
	digest := sha256.Sum256([]byte(rawToken))
	dbOperator := &models.Operator{
		Name:  operatorName,
		Token: hex.EncodeToString(digest[:]),
	}
	for _, permission := range permissions {
		switch permission {
		case "all":
			dbOperator.PermissionAll = true
			break
		case "builder":
			dbOperator.PermissionBuilder = true
		case "crackstation":
			dbOperator.PermissionCrackstation = true
		default:
			return nil, fmt.Errorf("invalid permission: %s", permission)
		}
	}
	err := db.Session().Save(dbOperator).Error
	if err != nil {
		return nil, err
	}

	publicKey, privateKey, err := certs.OperatorClientGenerateCertificate(operatorName)
	if err != nil {
		return nil, fmt.Errorf("failed to generate certificate %s", err)
	}
	caCertPEM, _, _ := certs.GetCertificateAuthorityPEM(certs.OperatorCA)
	config := ClientConfig{
		Operator:      operatorName,
		Token:         rawToken,
		LHost:         lhost,
		LPort:         int(lport),
		CACertificate: string(caCertPEM),
		PrivateKey:    string(privateKey),
		Certificate:   string(publicKey),
	}
	return json.Marshal(config)
}

func kickOperatorCmd(cmd *cobra.Command, _ []string) {
	operator, _ := cmd.Flags().GetString("name")

	fmt.Printf(Info+"Removing auth token(s) for %s, please wait ... \n", operator)
	err := db.Session().Where(&models.Operator{
		Name: operator,
	}).Delete(&models.Operator{}).Error
	if err != nil {
		return
	}
	transport.ClearTokenCache()
	fmt.Printf(Info+"Removing client certificate(s) for %s, please wait ... \n", operator)
	err = certs.OperatorClientRemoveCertificate(operator)
	if err != nil {
		fmt.Printf(Warn+"Failed to remove the operator certificate: %v \n", err)
		return
	}
	fmt.Printf(Info+"Operator %s has been kicked out.\n", operator)
}

func startMultiplayerModeCmd(cmd *cobra.Command, _ []string) {
	lhost, _ := cmd.Flags().GetString("lhost")
	lport, _ := cmd.Flags().GetUint16("lport")
	tailscale, _ := cmd.Flags().GetBool("tailscale")

	var err error
	var jobID int
	if tailscale {
		_, err = jobStartTsNetClientListener(lhost, lport)
	} else {
		jobID, err = JobStartClientListener(&clientpb.MultiplayerListenerReq{Host: lhost, Port: uint32(lport)})
	}
	if err == nil {
		fmt.Printf(Info + "Multiplayer mode enabled!\n")
		multiConfig := &clientpb.MultiplayerListenerReq{Host: lhost, Port: uint32(lport)}
		listenerJob := &clientpb.ListenerJob{
			JobID:     uint32(jobID),
			Type:      "multiplayer",
			MultiConf: multiConfig,
		}
		err = db.SaveHTTPC2Listener(listenerJob)
		if err != nil {
			fmt.Printf(Warn+"Failed to save job %v\n", err)
		}

	} else {
		fmt.Printf(Warn+"Failed to start job %v\n", err)
	}
}

func JobStartClientListener(multiplayerListener *clientpb.MultiplayerListenerReq) (int, error) {
	_, ln, err := transport.StartMtlsClientListener(multiplayerListener.Host, uint16(multiplayerListener.Port))
	if err != nil {
		return -1, err // If we fail to bind don't setup the Job
	}

	job := &core.Job{
		ID:          core.NextJobID(),
		Name:        "grpc/mtls",
		Description: "client listener",
		Protocol:    "tcp",
		Port:        uint16(multiplayerListener.Port),
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

func jobStartTsNetClientListener(host string, port uint16) (int, error) {
	_, ln, err := transport.StartTsNetClientListener(host, port)
	if err != nil {
		return -1, err // If we fail to bind don't setup the Job
	}

	job := &core.Job{
		ID:          core.NextJobID(),
		Name:        "grpc/tsnet",
		Description: "client listener",
		Protocol:    "tcp",
		Port:        uint16(port),
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
