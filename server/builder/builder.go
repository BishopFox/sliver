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
	"sync"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/server/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	builderLog = log.NamedLogger("builder", "sliver")
)

type Builder struct {
	externalBuilder *clientpb.Builder
	mutex           *sync.Mutex
	rpc             rpcpb.SliverRPCClient
	ln              *grpc.ClientConn
	lifecycleMu     sync.Mutex
	cancel          context.CancelFunc
	stopOnce        sync.Once
	activeBuilds    sync.WaitGroup
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
	ctx, cancel := context.WithCancel(context.Background())
	b.lifecycleMu.Lock()
	b.cancel = cancel
	b.lifecycleMu.Unlock()
	defer cancel()
	for ctx.Err() == nil {
		sessionCtx, stopSession := context.WithCancel(ctx)
		events, err := b.buildEvents(sessionCtx)
		if err == nil {
			go b.reportStatus(sessionCtx)
			for event := range events {
				if event.EventType == assignedStepEvent {
					b.activeBuilds.Add(1)
					go func() {
						defer b.activeBuilds.Done()
						b.handleAssignedStep(sessionCtx, event)
					}()
				}
			}
		}
		stopSession()
		b.activeBuilds.Wait() // Reconnect only after prior work has stopped.
		if ctx.Err() != nil {
			return
		}
		builderLog.Warnf("Builder %s registration ended; retrying in two seconds", b.externalBuilder.Name)
		retry := time.NewTimer(2 * time.Second)
		select {
		case <-ctx.Done():
			retry.Stop()
			return
		case <-retry.C:
		}
	}
}

func (b *Builder) Stop() {
	builderLog.Infof("Stopping builder %s", b.externalBuilder.Name)
	b.stopOnce.Do(func() {
		b.lifecycleMu.Lock()
		cancel := b.cancel
		b.lifecycleMu.Unlock()
		if cancel != nil {
			cancel()
		}
		b.mutex.Lock()
		defer b.mutex.Unlock()
		if b.ln != nil {
			_ = b.ln.Close()
		}
	})
	builderLog.Infof("Builder %s stopped", b.externalBuilder.Name)
}

func (b *Builder) buildEvents(ctx context.Context) (<-chan *clientpb.Event, error) {
	eventStream, err := b.rpc.BuilderRegister(ctx, b.externalBuilder)
	if err != nil {
		builderLog.Errorf("failed to register builder: %s", err)
		return nil, err
	}
	events := make(chan *clientpb.Event)
	go func() {
		defer close(events)
		for {
			event, err := eventStream.Recv()
			if err != nil || event == nil {
				if err != nil && err != io.EOF && ctx.Err() == nil {
					builderLog.Errorf("builder event stream closed: %s", err)
				}
				// return instead of exit because EOF can happen when we call `Builder.Stop()`
				// during a reload from SIGHUP
				return
			}

			if event.EventType == assignedStepEvent {
				select {
				case events <- event:
				case <-ctx.Done():
					return
				}
			} else {
				builderLog.Debugf("Ignore event (%s)", event.EventType)
			}
		}
	}()
	return events, nil
}

// reportStatus sends only changing capacity and toolchain state. Immutable
// spec references and capability tuples stay bound to this connection.
func (b *Builder) reportStatus(ctx context.Context) {
	if len(b.externalBuilder.SessionToken) != 32 || b.externalBuilder.Status == nil {
		return
	}
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		statusCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		_, err := b.rpc.BuilderUpdateStatus(statusCtx, &clientpb.BuilderStatusReport{
			Name: b.externalBuilder.Name, SessionToken: b.externalBuilder.SessionToken,
			EffectiveCPUCores: effectiveCPUCores(), AvailableMemoryBytes: availableMemoryBytes(),
			Toolchains: currentToolchainStatuses(b.externalBuilder.Status.Toolchains),
		})
		cancel()
		if status.Code(err) == codes.Unimplemented {
			builderLog.Errorf("Server does not support assigned-step builder status; update the server")
			return
		}
		if err != nil && ctx.Err() == nil {
			builderLog.Warnf("Failed to report builder capacity: %s", err)
		}
	}
}
