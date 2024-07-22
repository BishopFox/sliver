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
	"path/filepath"
	"strings"
	"sync"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/bishopfox/sliver/server/encoders"
	"github.com/bishopfox/sliver/server/generate"
	"github.com/bishopfox/sliver/server/log"
	"google.golang.org/grpc"
)

var (
	builderLog = log.NamedLogger("builder", "sliver")
)

type Config struct {
	GOOSs   []string
	GOARCHs []string
	Formats []clientpb.OutputFormat
}

type Builder struct {
	externalBuilder *clientpb.Builder
	mutex           *sync.Mutex
	rpc             rpcpb.SliverRPCClient
	ln              *grpc.ClientConn
}

func NewBuilder(config *clientpb.Builder, m *sync.Mutex, rpc rpcpb.SliverRPCClient, ln *grpc.ClientConn) *Builder {
	return &Builder{
		externalBuilder: config,
		mutex:           m,
		rpc:             rpc,
		ln:              ln,
	}
}

// StartBuilder - main entry point for the builder
func (b *Builder) Start() {
	builderLog.Infof("Attempting to register builder: %s", b.externalBuilder.Name)
	events, err := b.buildEvents()
	if err != nil {
		os.Exit(1)
	}

	for event := range events {
		go b.handleBuildEvent(event)
	}
}

func (b *Builder) Stop() {
	builderLog.Infof("Stopping builder %s", b.externalBuilder.Name)
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.ln.Close()
	builderLog.Infof("Builder %s stopped", b.externalBuilder.Name)
}

func (b *Builder) buildEvents() (<-chan *clientpb.Event, error) {
	eventStream, err := b.rpc.BuilderRegister(context.Background(), b.externalBuilder)
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
				// return instead of exit because EOF can happen when we call `Builder.Stop()`
				// during a reload from SIGHUP
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
	return events, nil
}

// handleBuildEvent - Handle an individual build event
func (b *Builder) handleBuildEvent(event *clientpb.Event) {

	parts := strings.Split(string(event.Data), ":")
	if len(parts) < 2 {
		builderLog.Errorf("Invalid build event data '%s'", event.Data)
		return
	}
	builderName := strings.Join(parts[:len(parts)-1], ":")
	if builderName != b.externalBuilder.Name {
		builderLog.Debugf("This build event is for someone else (%s), ignoring", builderName)
		return
	}

	implantBuildID := parts[1]
	builderLog.Infof("Build event for implant build id: %s", implantBuildID)
	extConfig, err := b.rpc.GenerateExternalGetBuildConfig(context.Background(), &clientpb.ImplantBuild{
		ID: implantBuildID,
	})
	if err != nil {
		builderLog.Errorf("Failed to get build config: %s", err)
		b.rpc.BuilderTrigger(context.Background(), &clientpb.Event{
			EventType: consts.ExternalBuildFailedEvent,
			Data:      []byte(fmt.Sprintf("%s:%s", implantBuildID, err.Error())),
		})
		return
	}
	if extConfig == nil {
		builderLog.Errorf("nil extConfig")
		b.rpc.BuilderTrigger(context.Background(), &clientpb.Event{
			EventType: consts.ExternalBuildFailedEvent,
			Data:      []byte(fmt.Sprintf("%s:%s", implantBuildID, "nil external config")),
		})
		return
	}
	if !isSupportedTarget(b.externalBuilder.Targets, extConfig.Config) {
		builderLog.Warnf("Skipping event, unsupported target %s:%s/%s", extConfig.Config.Format, extConfig.Config.GOOS, extConfig.Config.GOARCH)
		b.rpc.BuilderTrigger(context.Background(), &clientpb.Event{
			EventType: consts.ExternalBuildFailedEvent,
			Data: []byte(
				fmt.Sprintf("%s:%s", implantBuildID, fmt.Sprintf("unsupported target %s:%s/%s", extConfig.Config.Format, extConfig.Config.GOOS, extConfig.Config.GOARCH)),
			),
		})
		return
	}
	if extConfig.Config.TemplateName != "sliver" {
		builderLog.Warnf("Reject event, unsupported template '%s'", extConfig.Config.TemplateName)
		b.rpc.BuilderTrigger(context.Background(), &clientpb.Event{
			EventType: consts.ExternalBuildFailedEvent,
			Data:      []byte(fmt.Sprintf("%s:%s", implantBuildID, "Unsupported template")),
		})
		return
	}

	extModel := models.ImplantConfigFromProtobuf(extConfig.Config)

	builderLog.Infof("Building %s for %s/%s (format: %s)", extConfig.Build.Name, extConfig.Config.GOOS, extConfig.Config.GOARCH, extConfig.Config.Format)
	builderLog.Infof("    [c2] mtls:%t wg:%t http/s:%t dns:%t", extModel.IncludeMTLS, extModel.IncludeWG, extModel.IncludeHTTP, extModel.IncludeDNS)
	builderLog.Infof("[pivots] tcp:%t named-pipe:%t", extModel.IncludeTCP, extModel.IncludeNamePipe)

	b.rpc.BuilderTrigger(context.Background(), &clientpb.Event{
		EventType: consts.AcknowledgeBuildEvent,
		Data:      []byte(implantBuildID),
	})

	httpC2Config, err := db.LoadHTTPC2ConfigByName(extConfig.Config.HTTPC2ConfigName)
	if err != nil {
		builderLog.Errorf("Failed to load c2 config: %s", err)
		b.rpc.BuilderTrigger(context.Background(), &clientpb.Event{
			EventType: consts.ExternalBuildFailedEvent,
			Data:      []byte(fmt.Sprintf("%s:%s", implantBuildID, err.Error())),
		})
		return
	}
	encoders.Base32EncoderID = extConfig.Encoders["base32"]
	encoders.Base58EncoderID = extConfig.Encoders["base58"]
	encoders.Base64EncoderID = extConfig.Encoders["base64"]
	encoders.EnglishEncoderID = extConfig.Encoders["english"]
	encoders.GzipEncoderID = extConfig.Encoders["gzip"]
	encoders.HexEncoderID = extConfig.Encoders["hex"]
	encoders.PNGEncoderID = extConfig.Encoders["png"]

	var fPath string
	switch extConfig.Config.Format {
	case clientpb.OutputFormat_SERVICE:
		fallthrough
	case clientpb.OutputFormat_EXECUTABLE:
		b.mutex.Lock()
		fPath, err = generate.SliverExecutable(extConfig.Build.Name, extConfig.Build, extConfig.Config, httpC2Config.ImplantConfig)
		b.mutex.Unlock()
	case clientpb.OutputFormat_SHARED_LIB:
		b.mutex.Lock()
		fPath, err = generate.SliverSharedLibrary(extConfig.Build.Name, extConfig.Build, extConfig.Config, httpC2Config.ImplantConfig)
		b.mutex.Unlock()
	case clientpb.OutputFormat_SHELLCODE:
		b.mutex.Lock()
		fPath, err = generate.SliverShellcode(extConfig.Build.Name, extConfig.Build, extConfig.Config, httpC2Config.ImplantConfig)
		b.mutex.Unlock()
	default:
		builderLog.Errorf("invalid output format: %s", extConfig.Config.Format)
		b.rpc.BuilderTrigger(context.Background(), &clientpb.Event{
			EventType: consts.ExternalBuildFailedEvent,
			Data:      []byte(fmt.Sprintf("%s:%s", implantBuildID, err)),
		})
		return
	}
	if err != nil {
		builderLog.Errorf("Failed to generate sliver: %s", err)
		b.rpc.BuilderTrigger(context.Background(), &clientpb.Event{
			EventType: consts.ExternalBuildFailedEvent,
			Data:      []byte(fmt.Sprintf("%s:%s", implantBuildID, err.Error())),
		})
		return
	}
	builderLog.Infof("Build completed successfully: %s", fPath)

	data, err := os.ReadFile(fPath)
	if err != nil {
		builderLog.Errorf("Failed to read generated sliver: %s", err)
		b.rpc.BuilderTrigger(context.Background(), &clientpb.Event{
			EventType: consts.ExternalBuildFailedEvent,
			Data:      []byte(fmt.Sprintf("%s:%s", implantBuildID, err.Error())),
		})
		return
	}

	fileName := filepath.Base(extConfig.Build.Name)
	if extConfig.Config.GOOS == "windows" {
		fileName += ".exe"
	}

	builderLog.Infof("Uploading '%s' to server ...", extConfig.Build.Name)
	_, err = b.rpc.GenerateExternalSaveBuild(context.Background(), &clientpb.ExternalImplantBinary{
		Name:           extConfig.Build.Name,
		ImplantBuildID: extConfig.Build.ID,
		File: &commonpb.File{
			Name: fileName,
			Data: data,
		},
	})
	if err != nil {
		builderLog.Errorf("Failed to save build: %s", err)
		b.rpc.BuilderTrigger(context.Background(), &clientpb.Event{
			EventType: consts.ExternalBuildFailedEvent,
			Data:      []byte(fmt.Sprintf("%s:%s", implantBuildID, err.Error())),
		})
		return
	}
	b.rpc.BuilderTrigger(context.Background(), &clientpb.Event{
		EventType: consts.ExternalBuildCompletedEvent,
		Data:      []byte(fmt.Sprintf("%s:%s", implantBuildID, extConfig.Build.Name)),
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
