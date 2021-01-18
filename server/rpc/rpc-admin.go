package rpc

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
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"

	"golang.org/x/crypto/ssh"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/transport"
)

var (
	namePattern = regexp.MustCompile("^[a-zA-Z0-9_]*$") // Only allow alphanumeric chars
)

// CreatePlayer - This admin (server) console wants to create a new player.
func (rpc *Server) CreatePlayer(ctx context.Context, req *clientpb.NewPlayerReq) (*clientpb.NewPlayer, error) {

	res := &clientpb.NewPlayer{Response: &commonpb.Response{}}

	var save string
	if req.Save == "" {
		save, _ = os.Getwd()
	}

	rpcLog.Infof("Generating new client certificate for user %s (server at %s:%d)", req.Name, req.LHost, req.LPort)
	configJSON, err := newPlayerConfig(req.Name, req.LHost, uint16(req.LPort))
	if err != nil {
		rpcLog.Errorf("Failed to generate user config: %s", err)
		res.Response.Err = err.Error()
		return res, nil
	}

	saveTo, _ := filepath.Abs(save)
	fi, err := os.Stat(saveTo)
	if !os.IsNotExist(err) && !fi.IsDir() {
		rpcLog.Errorf("Failed to generate user config: file already exists (%s) ", saveTo)
		res.Response.Err = fmt.Sprintf("File already exists (%s)", saveTo)
		return res, nil
	}
	if !os.IsNotExist(err) && fi.IsDir() {
		filename := fmt.Sprintf("%s_%s.cfg", filepath.Base(req.Name), filepath.Base(req.LHost))
		saveTo = filepath.Join(saveTo, filename)
	}
	err = ioutil.WriteFile(saveTo, configJSON, 0600)
	if err != nil {
		rpcLog.Errorf("Failed to write config to: %s (%v) \n", saveTo, err)
		res.Response.Err = fmt.Sprintf("Failed to write config to: %s (%v) \n", saveTo, err)
		return res, nil
	}
	rpcLog.Infof("Saved new client config to: %s \n", saveTo)

	res.Success = true
	return res, nil
}

// StartMultiplayer - Make this server listen for remote console clients.
func (rpc *Server) StartMultiplayer(ctx context.Context, req *clientpb.MultiplayerReq) (*clientpb.Multiplayer, error) {
	res := &clientpb.Multiplayer{Response: &commonpb.Response{}}

	_, err := jobStartClientListener(req.LHost, uint16(req.LPort))
	if err == nil {
		rpcLog.Infof("Multiplayer mode enabled on %s:%d", req.LHost, req.LPort)
	} else {
		res.Response.Err = err.Error()
		return res, nil
	}

	res.Success = true
	return res, nil
}

// KickPlayer - Delete a player from this server.
func (rpc *Server) KickPlayer(ctx context.Context, req *clientpb.RemovePlayerReq) (*clientpb.RemovePlayer, error) {
	res := &clientpb.RemovePlayer{Response: &commonpb.Response{}}

	operator := req.Name

	if !namePattern.MatchString(operator) {
		res.Response.Err = "Invalid operator name (alphanumerics only)"
		return res, nil
	}
	if operator == "" {
		res.Response.Err = "Operator name required (--operator)"
		return res, nil
	}

	rpcLog.Infof("Removing client certificate for operator %s, please wait ... \n", operator)
	err := certs.OperatorClientRemoveCertificate(operator)
	if err != nil {
		rpcLog.Errorf("Failed to remove the operator certificate: %v \n", err)
		res.Response.Err = fmt.Sprintf("Failed to remove the operator certificate: %v \n", err)
		return res, nil
	}
	rpcLog.Infof("Operator %s kicked out. \n", operator)

	res.Success = true
	return res, nil
}

// newPlayerConfig - Generate a new player/client/operator configuration
func newPlayerConfig(operatorName, lhost string, lport uint16) ([]byte, error) {

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

	caCertPEM, serverCAKey, _ := certs.GetCertificateAuthorityPEM(certs.OperatorCA)

	// Make a fingerprint of the implant's private key, for SSH-layer authentication
	signer, _ := ssh.ParsePrivateKey(serverCAKey)
	keyBytes := sha256.Sum256(signer.PublicKey().Marshal())
	fingerprint := base64.StdEncoding.EncodeToString(keyBytes[:])

	config := assets.ClientConfig{
		Operator:          operatorName,
		LHost:             lhost,
		LPort:             int(lport),
		CACertificate:     string(caCertPEM),
		PrivateKey:        string(privateKey),
		Certificate:       string(publicKey),
		ServerFingerprint: fingerprint,
	}
	return json.Marshal(config)
}

// Start the console client listener
func jobStartClientListener(host string, port uint16) (int, error) {
	grpcServer, ln, err := transport.StartClientListener(host, port)
	if err != nil {
		return -1, err // If we fail to bind don't setup the Job
	}

	// Register the RPC server of this package. Avoids circular imports.
	rpcpb.RegisterSliverRPCServer(grpcServer, NewServer())
	go func() {
		if err := grpcServer.Serve(ln); err != nil {
			rpcLog.Warnf("gRPC server exited with error: %v", err)
		}
	}()

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
			EventType: constants.JobStoppedEvent,
		})
	}()

	core.Jobs.Add(job)
	return job.ID, nil
}
