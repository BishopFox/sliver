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
	"net"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"gorm.io/gorm"

	clientassets "github.com/bishopfox/sliver/client/assets"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/console/forms"
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

func newOperatorCmd(cmd *cobra.Command, _ []string) {
	name, _ := cmd.Flags().GetString("name")
	lhost, _ := cmd.Flags().GetString("lhost")
	lport, _ := cmd.Flags().GetUint16("lport")
	save, _ := cmd.Flags().GetString("save")
	permissions, _ := cmd.Flags().GetStringSlice("permissions")
	includeWG, _ := cmd.Flags().GetBool("enable-wg")

	if save == "" {
		save, _ = os.Getwd()
	}

	fmt.Printf(Info + "Generating new client certificate, please wait ... \n")
	configJSON, err := NewOperatorConfig(name, lhost, lport, permissions, includeWG)
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
func NewOperatorConfig(operatorName string, lhost string, lport uint16, permissions []string, includeWG bool) ([]byte, error) {
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

	var wgConfig *clientassets.ClientWGConfig
	operatorSaved := false
	if includeWG {
		certs.SetupMultiplayerWGKeys()
		clientPrivKey, clientPubKey, err := certs.GenerateWGKeyPair()
		if err != nil {
			return nil, fmt.Errorf("failed to generate wireguard keys: %w", err)
		}
		_, serverPubKey, err := certs.GetMultiplayerWGServerKeys()
		if err != nil {
			return nil, fmt.Errorf("failed to get server wireguard keys: %w", err)
		}

		for attempt := 0; attempt < 32; attempt++ {
			clientIP, err := db.NextAvailableMultiplayerWGIP()
			if err != nil {
				return nil, fmt.Errorf("failed to allocate wireguard address: %w", err)
			}

			operatorRecord := *dbOperator
			operatorRecord.WGPubKey = clientPubKey
			operatorRecord.WGTunIP = clientIP
			err = db.Session().Transaction(func(tx *gorm.DB) error {
				if err := db.ReserveWGIPTx(tx, clientIP, models.WGIPOwnerTypeOperator, operatorName); err != nil {
					return err
				}
				return tx.Create(&operatorRecord).Error
			})
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				continue
			}
			if err != nil {
				return nil, err
			}

			dbOperator = &operatorRecord
			wgConfig = &clientassets.ClientWGConfig{
				ServerPubKey:     serverPubKey,
				ClientPrivateKey: clientPrivKey,
				ClientPubKey:     clientPubKey,
				ClientIP:         clientIP,
				ServerIP:         certs.MultiplayerWireGuardServerIP,
			}
			operatorSaved = true
			break
		}
		if !operatorSaved {
			return nil, fmt.Errorf("failed to allocate a unique wireguard address after %d attempts", 32)
		}
	} else {
		err := db.Session().Create(dbOperator).Error
		if err != nil {
			return nil, err
		}
		operatorSaved = true
	}
	if !operatorSaved {
		return nil, errors.New("failed to persist operator")
	}

	publicKey, privateKey, err := certs.OperatorClientGenerateCertificate(operatorName)
	if err != nil {
		return nil, fmt.Errorf("failed to generate certificate %s", err)
	}
	caCertPEM, _, _ := certs.GetCertificateAuthorityPEM(certs.OperatorCA)
	config := clientassets.ClientConfig{
		Operator:      operatorName,
		Token:         rawToken,
		LHost:         lhost,
		LPort:         int(lport),
		CACertificate: string(caCertPEM),
		PrivateKey:    string(privateKey),
		Certificate:   string(publicKey),
		WG:            wgConfig,
	}
	if includeWG {
		publishOperatorWGPeerAddition(dbOperator)
	}
	return json.Marshal(config)
}

func kickOperatorCmd(cmd *cobra.Command, args []string) {
	operator, _ := cmd.Flags().GetString("name")

	if shouldPromptKickOperator(cmd, args) {
		names, err := operatorNames()
		if err != nil {
			fmt.Printf(Warn+"Failed to list operators: %v\n", err)
			return
		}
		if len(names) == 0 {
			fmt.Printf(Warn + "No operators found.\n")
			return
		}
		if err := forms.SelectOperator("Select operator to kick", names, &operator); err != nil {
			if errors.Is(err, forms.ErrUserAborted) {
				return
			}
			fmt.Printf(Warn+"Operator selection failed: %v\n", err)
			return
		}
		if err := cmd.Flags().Set("name", operator); err != nil {
			fmt.Printf(Warn+"Failed to set operator name: %v\n", err)
			return
		}
	}

	operator = strings.TrimSpace(operator)
	if operator == "" {
		fmt.Printf(Warn + "Operator name required (use --name).\n")
		return
	}

	exists, err := operatorExists(operator)
	if err != nil {
		fmt.Printf(Warn+"Failed to lookup operator %s: %v\n", operator, err)
		return
	}
	if !exists {
		fmt.Printf(Warn+"Operator %s does not exist.\n", operator)
		return
	}

	fmt.Printf(Info+"Removing auth token(s) for %s, please wait ... \n", operator)
	err = kickOperator(operator)
	if err != nil {
		fmt.Printf(Warn+"Failed to kick operator %s: %v\n", operator, err)
		return
	}
	fmt.Printf(Info+"Operator %s has been kicked out.\n", operator)
}

func shouldPromptKickOperator(cmd *cobra.Command, args []string) bool {
	if len(args) != 0 {
		return false
	}
	return cmd.Flags().NFlag() == 0
}

func removeOperator(operator string) error {
	operators, err := operatorRecordsByName(operator)
	if err != nil {
		return err
	}
	err = db.Session().Where(&models.Operator{
		Name: operator,
	}).Delete(&models.Operator{}).Error
	if err != nil {
		return err
	}
	for _, dbOperator := range operators {
		if dbOperator == nil {
			continue
		}
		if err := db.ReleaseWGIP(dbOperator.WGTunIP); err != nil {
			return err
		}
	}
	transport.ClearTokenCache()
	return nil
}

func revokeOperatorClientCertificate(operator string) error {
	return certs.OperatorClientRemoveCertificate(operator)
}

func closeOperatorStreams(operator string) {
	transport.CloseOperatorStreams(operator)
}

func kickOperator(operator string) error {
	dbOperators, err := operatorRecordsByName(operator)
	if err != nil {
		return err
	}

	if err := removeOperator(operator); err != nil {
		return err
	}
	defer closeOperatorStreams(operator)
	for _, dbOperator := range dbOperators {
		publishOperatorWGPeerRemoval(dbOperator)
	}
	return revokeOperatorClientCertificate(operator)
}

func operatorNames() ([]string, error) {
	operators, err := db.OperatorAll()
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(operators))
	seen := make(map[string]struct{}, len(operators))
	for _, operator := range operators {
		if operator == nil {
			continue
		}
		name := strings.TrimSpace(operator.Name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

func operatorExists(name string) (bool, error) {
	if strings.TrimSpace(name) == "" {
		return false, nil
	}
	err := db.Session().Where(&models.Operator{
		Name: name,
	}).First(&models.Operator{}).Error
	if err == nil {
		return true, nil
	}
	if errors.Is(err, db.ErrRecordNotFound) {
		return false, nil
	}
	return false, err
}

func operatorRecordsByName(name string) ([]*models.Operator, error) {
	operators := []*models.Operator{}
	err := db.Session().Where(&models.Operator{Name: name}).Find(&operators).Error
	return operators, err
}

func publishOperatorWGPeerRemoval(operator *models.Operator) {
	if operator == nil || strings.TrimSpace(operator.WGPubKey) == "" {
		return
	}
	core.EventBroker.Publish(core.Event{
		EventType: consts.MultiplayerWireGuardRemoved,
		Data:      []byte(fmt.Sprintf("public_key=%s\nremove=true\n", operator.WGPubKey)),
	})
}

func publishOperatorWGPeerAddition(operator *models.Operator) {
	if operator == nil || strings.TrimSpace(operator.WGPubKey) == "" || strings.TrimSpace(operator.WGTunIP) == "" {
		return
	}
	core.EventBroker.Publish(core.Event{
		EventType: consts.MultiplayerWireGuardNewPeer,
		Data:      []byte(fmt.Sprintf("public_key=%s\nallowed_ip=%s/32\n", operator.WGPubKey, operator.WGTunIP)),
	})
}

func startMultiplayerModeCmd(cmd *cobra.Command, _ []string) {
	lhost, _ := cmd.Flags().GetString("lhost")
	lport, _ := cmd.Flags().GetUint16("lport")
	tailscale, _ := cmd.Flags().GetBool("tailscale")
	enableWG, _ := cmd.Flags().GetBool("enable-wg")
	useWireGuard := enableWG && !tailscale

	var err error
	var jobID int
	if tailscale {
		_, err = jobStartTsNetClientListener(lhost, lport)
	} else {
		jobID, err = JobStartClientListener(&clientpb.MultiplayerListenerReq{
			Host:      lhost,
			Port:      uint32(lport),
			WireGuard: useWireGuard,
		})
	}
	if err == nil {
		fmt.Printf(Info + "Multiplayer mode enabled!\n")
		multiConfig := &clientpb.MultiplayerListenerReq{
			Host:      lhost,
			Port:      uint32(lport),
			WireGuard: useWireGuard,
		}
		listenerJob := &clientpb.ListenerJob{
			JobID:     uint32(jobID),
			Type:      "multiplayer",
			MultiConf: multiConfig,
		}
		err = db.SaveC2Listener(listenerJob)
		if err != nil {
			fmt.Printf(Warn+"Failed to save job %v\n", err)
		}

	} else {
		fmt.Printf(Warn+"Failed to start job %v\n", err)
	}
}

func JobStartClientListener(multiplayerListener *clientpb.MultiplayerListenerReq) (int, error) {
	var (
		ln  net.Listener
		err error
	)
	if multiplayerListener.GetWireGuard() {
		_, ln, err = transport.StartWGWrappedMtlsClientListener(multiplayerListener.Host, uint16(multiplayerListener.Port))
	} else {
		_, ln, err = transport.StartMtlsClientListener(multiplayerListener.Host, uint16(multiplayerListener.Port))
	}
	if err != nil {
		return -1, err // If we fail to bind don't setup the Job
	}

	name := "grpc/mtls"
	description := "client listener"
	protocol := "tcp"
	if multiplayerListener.GetWireGuard() {
		name = "grpc/mtls+wg"
		description = "wireguard-wrapped client listener"
		protocol = "udp"
	}

	job := &core.Job{
		ID:          core.NextJobID(),
		Name:        name,
		Description: description,
		Protocol:    protocol,
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
