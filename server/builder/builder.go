package builder

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox

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
	"io"
	"os"
	"os/signal"
	"path/filepath"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/server/codenames"
	"github.com/bishopfox/sliver/server/generate"
	"github.com/bishopfox/sliver/server/log"
	"github.com/bishopfox/sliver/util"
)

var (
	builderLog = log.NamedLogger("builder", "sliver")
)

type Config struct {
	GOOSs   []string
	GOARCHs []string
	Formats []clientpb.OutputFormat
}

// StartBuilder - main entry point for the builder
func StartBuilder(rpc rpcpb.SliverRPCClient, conf Config) {

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)

	events := buildEvents(rpc)

	builderLog.Infof("Successfully started process as external builder")
	builder := sliverBuilder{
		rpc:    rpc,
		config: conf,
	}
	// Wait for signal or builds
	for {
		select {
		case <-sigint:
			return
		case event := <-events:
			go builder.HandleBuildEvent(event)
		}
	}
}

type sliverBuilder struct {
	rpc    rpcpb.SliverRPCClient
	config Config
}

func buildEvents(rpc rpcpb.SliverRPCClient) <-chan *clientpb.Event {
	eventStream, err := rpc.Events(context.Background(), &commonpb.Empty{})
	if err != nil {
		builderLog.Fatal(err)
	}
	events := make(chan *clientpb.Event)
	go func() {
		for {
			event, err := eventStream.Recv()
			if err == io.EOF || event == nil {
				return
			}

			// Trigger event based on type
			switch event.EventType {
			case consts.ExternalBuildEvent:
				events <- event
			default:
				builderLog.Debugf("Ignore event (%s)", event.EventType)
			}
		}
	}()
	return events
}

// HandleBuildEvent - Handle an individual build event
func (b *sliverBuilder) HandleBuildEvent(event *clientpb.Event) {
	implantConfigID := string(event.Data)
	builderLog.Infof("Build event for implant config id: %s", implantConfigID)
	extConfig, err := b.rpc.GenerateExternalGetImplantConfig(context.Background(), &clientpb.ImplantConfig{
		ID: implantConfigID,
	})
	if err != nil {
		builderLog.Errorf("Failed to get implant config: %s", err)
		return
	}
	if extConfig == nil {
		builderLog.Errorf("nil extConfig")
		return
	}

	// check to see if the event matches a target we're configured to build for
	if !contains(b.config.GOOSs, extConfig.Config.GOOS) {
		builderLog.Warnf("This builder is not configured to build for goos %s, ignore event", extConfig.Config.GOOS)
		return
	}
	if !contains(b.config.GOARCHs, extConfig.Config.GOARCH) {
		builderLog.Warnf("This builder is not configured to build for goarch %s, ignore event", extConfig.Config.GOARCH)
		return
	}
	if !contains(b.config.Formats, extConfig.Config.Format) {
		builderLog.Warnf("This builder is not configured to build for format %s, ignore event", extConfig.Config.Format)
		return
	}
	if extConfig.Config.Name == "" {
		extConfig.Config.Name, _ = codenames.GetCodename()
	}
	err = util.AllowedName(extConfig.Config.Name)
	if err != nil {
		builderLog.Errorf("Invalid implant name: %s", err)
		return
	}
	_, extModel := generate.ImplantConfigFromProtobuf(extConfig.Config)

	builderLog.Infof("Building %s for %s/%s (format: %s)", extConfig.Config.Name, extConfig.Config.GOOS, extConfig.Config.GOARCH, extConfig.Config.Format)
	builderLog.Infof("    [c2] mtls:%t wg:%t http/s:%t dns:%t", extModel.MTLSc2Enabled, extModel.WGc2Enabled, extModel.HTTPc2Enabled, extModel.DNSc2Enabled)
	builderLog.Infof("[pivots] tcp:%t named-pipe:%t", extModel.TCPPivotc2Enabled, extModel.NamePipec2Enabled)

	var fPath string
	switch extConfig.Config.Format {
	case clientpb.OutputFormat_SERVICE:
		fallthrough
	case clientpb.OutputFormat_EXECUTABLE:
		fPath, err = generate.SliverExecutable(extConfig.Config.Name, extConfig.OTPSecret, extModel, false)
	case clientpb.OutputFormat_SHARED_LIB:
		fPath, err = generate.SliverSharedLibrary(extConfig.Config.Name, extConfig.OTPSecret, extModel, false)
	case clientpb.OutputFormat_SHELLCODE:
		fPath, err = generate.SliverShellcode(extConfig.Config.Name, extConfig.OTPSecret, extModel, false)
	default:
		builderLog.Errorf("invalid output format: %s", extConfig.Config.Format)
		return
	}
	if err != nil {
		builderLog.Errorf("Failed to generate sliver: %s", err)
		return
	}
	builderLog.Infof("Build completed successfully: %s", fPath)

	data, err := os.ReadFile(fPath)
	if err != nil {
		builderLog.Errorf("Failed to read generated sliver: %s", err)
		return
	}

	fileName := filepath.Base(extConfig.Config.Name)
	if extConfig.Config.GOOS == "windows" {
		fileName += ".exe"
	}

	builderLog.Infof("Uploading '%s' to server ...", extConfig.Config.Name)
	_, err = b.rpc.GenerateExternalSaveBuild(context.Background(), &clientpb.ExternalImplantBinary{
		Name:            extConfig.Config.Name,
		ImplantConfigID: extConfig.Config.ID,
		File: &commonpb.File{
			Name: fileName,
			Data: data,
		},
	})
	if err != nil {
		builderLog.Errorf("Failed to save build: %s", err)
		return
	}
	builderLog.Infof("All done, built and saved %s", fileName)
}

func contains[T comparable](elems []T, v T) bool {
	for _, s := range elems {
		if v == s {
			return true
		}
	}
	return false
}
