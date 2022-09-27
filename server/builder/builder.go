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

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/server/log"
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

	builderLog.Infof("Started process as external builder")
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
			case consts.BuildEvent:
				events <- event
			}
		}
	}()
	return events
}

// HandleBuildEvent - Handle an individual build event
func (b *sliverBuilder) HandleBuildEvent(event *clientpb.Event) {
	implantConfigID := string(event.Data)
	builderLog.Infof("Build event for implant config id: %s", implantConfigID)
	implantConf, err := b.rpc.GenerateExternalGetImplantConfig(context.Background(), &clientpb.ImplantConfig{
		ID: implantConfigID,
	})
	if err != nil {
		builderLog.Errorf("Failed to get implant config: %s", err)
		return
	}

	// check to see if the event matches a target we're configured to build for
	if !contains(b.config.GOOSs, implantConf.Config.GOOS) {
		builderLog.Warnf("This builder is not configured to build for goos %s, ignore event", implantConf.Config.GOOS)
		return
	}
	if !contains(b.config.GOARCHs, implantConf.Config.GOARCH) {
		builderLog.Warnf("This builder is not configured to build for goarch %s, ignore event", implantConf.Config.GOARCH)
		return
	}
	if !contains(b.config.Formats, implantConf.Config.Format) {
		builderLog.Warnf("This builder is not configured to build for format %s, ignore event", implantConf.Config.Format)
		return
	}

	builderLog.Infof("Building for %s/%s (format: %s)", implantConf.Config.GOOS, implantConf.Config.GOARCH, implantConf.Config.Format)

}

func contains[T comparable](elems []T, v T) bool {
	for _, s := range elems {
		if v == s {
			return true
		}
	}
	return false
}
