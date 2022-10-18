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
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

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
func StartBuilder(externalBuilder *clientpb.Builder, rpc rpcpb.SliverRPCClient) {

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)

	builderLog.Infof("Attempting to register builder: %s", externalBuilder.Name)
	events, err := buildEvents(externalBuilder, rpc)
	if err != nil {
		os.Exit(1)
	}

	// Wait for signal or builds
	for {
		select {
		case <-sigint:
			return
		case event := <-events:
			go handleBuildEvent(externalBuilder, event, rpc)
		}
	}
}

func buildEvents(externalBuilder *clientpb.Builder, rpc rpcpb.SliverRPCClient) (<-chan *clientpb.Event, error) {
	eventStream, err := rpc.BuilderRegister(context.Background(), externalBuilder)
	if err != nil {
		builderLog.Errorf("failed to register builder: %s", err)
		return nil, err
	}
	events := make(chan *clientpb.Event)
	go func() {
		for {
			event, err := eventStream.Recv()
			if err == io.EOF || event == nil {
				builderLog.Errorf("builder event stream closed")
				os.Exit(1)
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
	return events, nil
}

// handleBuildEvent - Handle an individual build event
func handleBuildEvent(externalBuilder *clientpb.Builder, event *clientpb.Event, rpc rpcpb.SliverRPCClient) {
	parts := strings.Split(string(event.Data), ":")
	if len(parts) < 2 {
		builderLog.Errorf("Invalid build event data '%s'", event.Data)
		return
	}
	builderName := strings.Join(parts[:len(parts)-1], ":")
	if builderName != externalBuilder.Name {
		builderLog.Debugf("This build event is for someone else (%s), ignoring", builderName)
		return
	}

	implantConfigID := parts[1]
	builderLog.Infof("Build event for implant config id: %s", implantConfigID)
	extConfig, err := rpc.GenerateExternalGetImplantConfig(context.Background(), &clientpb.ImplantConfig{
		ID: implantConfigID,
	})
	if err != nil {
		builderLog.Errorf("Failed to get implant config: %s", err)
		rpc.BuilderTrigger(context.Background(), &clientpb.Event{
			EventType: consts.ExternalBuildFailedEvent,
			Data:      []byte(fmt.Sprintf("%s:%s", implantConfigID, err.Error())),
		})
		return
	}
	if extConfig == nil {
		builderLog.Errorf("nil extConfig")
		rpc.BuilderTrigger(context.Background(), &clientpb.Event{
			EventType: consts.ExternalBuildFailedEvent,
			Data:      []byte(fmt.Sprintf("%s:%s", implantConfigID, "nil external config")),
		})
		return
	}
	if !isSupportedTarget(externalBuilder.Targets, extConfig.Config) {
		builderLog.Warnf("Skipping event, unsupported target %s:%s/%s", extConfig.Config.Format, extConfig.Config.GOOS, extConfig.Config.GOARCH)
		rpc.BuilderTrigger(context.Background(), &clientpb.Event{
			EventType: consts.ExternalBuildFailedEvent,
			Data: []byte(
				fmt.Sprintf("%s:%s", implantConfigID, fmt.Sprintf("unsupported target %s:%s/%s", extConfig.Config.Format, extConfig.Config.GOOS, extConfig.Config.GOARCH)),
			),
		})
		return
	}
	if extConfig.Config.TemplateName != "sliver" {
		builderLog.Warnf("Reject event, unsupported template '%s'", extConfig.Config.TemplateName)
		rpc.BuilderTrigger(context.Background(), &clientpb.Event{
			EventType: consts.ExternalBuildFailedEvent,
			Data:      []byte(fmt.Sprintf("%s:%s", implantConfigID, "Unsupported template")),
		})
		return
	}

	if extConfig.Config.Name == "" {
		extConfig.Config.Name, _ = codenames.GetCodename()
	}
	err = util.AllowedName(extConfig.Config.Name)
	if err != nil {
		builderLog.Errorf("Invalid implant name: %s", err)
		rpc.BuilderTrigger(context.Background(), &clientpb.Event{
			EventType: consts.ExternalBuildFailedEvent,
			Data:      []byte(fmt.Sprintf("%s:%s", implantConfigID, err.Error())),
		})
		return
	}
	_, extModel := generate.ImplantConfigFromProtobuf(extConfig.Config)

	builderLog.Infof("Building %s for %s/%s (format: %s)", extConfig.Config.Name, extConfig.Config.GOOS, extConfig.Config.GOARCH, extConfig.Config.Format)
	builderLog.Infof("    [c2] mtls:%t wg:%t http/s:%t dns:%t", extModel.MTLSc2Enabled, extModel.WGc2Enabled, extModel.HTTPc2Enabled, extModel.DNSc2Enabled)
	builderLog.Infof("[pivots] tcp:%t named-pipe:%t", extModel.TCPPivotc2Enabled, extModel.NamePipec2Enabled)

	rpc.BuilderTrigger(context.Background(), &clientpb.Event{
		EventType: consts.AcknowledgeBuildEvent,
		Data:      []byte(implantConfigID),
	})

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
		rpc.BuilderTrigger(context.Background(), &clientpb.Event{
			EventType: consts.ExternalBuildFailedEvent,
			Data:      []byte(fmt.Sprintf("%s:%s", implantConfigID, err.Error())),
		})
		return
	}
	if err != nil {
		builderLog.Errorf("Failed to generate sliver: %s", err)
		rpc.BuilderTrigger(context.Background(), &clientpb.Event{
			EventType: consts.ExternalBuildFailedEvent,
			Data:      []byte(fmt.Sprintf("%s:%s", implantConfigID, err.Error())),
		})
		return
	}
	builderLog.Infof("Build completed successfully: %s", fPath)

	data, err := os.ReadFile(fPath)
	if err != nil {
		builderLog.Errorf("Failed to read generated sliver: %s", err)
		rpc.BuilderTrigger(context.Background(), &clientpb.Event{
			EventType: consts.ExternalBuildFailedEvent,
			Data:      []byte(fmt.Sprintf("%s:%s", implantConfigID, err.Error())),
		})
		return
	}

	fileName := filepath.Base(extConfig.Config.Name)
	if extConfig.Config.GOOS == "windows" {
		fileName += ".exe"
	}

	builderLog.Infof("Uploading '%s' to server ...", extConfig.Config.Name)
	_, err = rpc.GenerateExternalSaveBuild(context.Background(), &clientpb.ExternalImplantBinary{
		Name:            extConfig.Config.Name,
		ImplantConfigID: extConfig.Config.ID,
		File: &commonpb.File{
			Name: fileName,
			Data: data,
		},
	})
	if err != nil {
		builderLog.Errorf("Failed to save build: %s", err)
		rpc.BuilderTrigger(context.Background(), &clientpb.Event{
			EventType: consts.ExternalBuildFailedEvent,
			Data:      []byte(fmt.Sprintf("%s:%s", implantConfigID, err.Error())),
		})
		return
	}
	rpc.BuilderTrigger(context.Background(), &clientpb.Event{
		EventType: consts.ExternalBuildCompletedEvent,
		Data:      []byte(fmt.Sprintf("%s:%s", implantConfigID, extConfig.Config.Name)),
	})
	builderLog.Infof("All done, built and saved %s", fileName)
}

func isSupportedTarget(targets []*clientpb.CompilerTarget, config *clientpb.ImplantConfig) bool {
	for _, target := range targets {
		if target.GOOS == config.GOOS && target.GOARCH == config.GOARCH {
			return true
		}
	}
	return false
}
