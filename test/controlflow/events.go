//go:build sliver_controlflow_e2e

package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"google.golang.org/protobuf/proto"
)

type eventPump struct {
	events chan *clientpb.Event
	errors chan error
}

func startEventPump(stream rpcpb.SliverRPC_EventsClient) *eventPump {
	pump := &eventPump{
		events: make(chan *clientpb.Event, 256),
		errors: make(chan error, 1),
	}
	go func() {
		defer close(pump.events)
		for {
			event, err := stream.Recv()
			if err != nil {
				pump.errors <- err
				return
			}
			select {
			case pump.events <- event:
			case <-stream.Context().Done():
				pump.errors <- stream.Context().Err()
				return
			}
		}
	}()
	return pump
}

func waitForListenerReady(ctx context.Context, pump *eventPump, jobID uint32, address string) error {
	ticker := time.NewTicker(listenerPollInterval)
	defer ticker.Stop()
	started := false
	for {
		if started {
			connection, err := net.DialTimeout("tcp", address, listenerDialTimeout)
			if err == nil {
				_ = connection.Close()
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-pump.events:
			if !ok {
				return eventStreamError(pump, "event stream ended while starting implant mTLS listener")
			}
			if event.Job == nil || event.Job.ID != jobID {
				continue
			}
			switch event.EventType {
			case consts.JobStartedEvent:
				started = true
			case consts.JobStoppedEvent:
				if event.Err != "" {
					return fmt.Errorf("implant mTLS listener job %d stopped: %s", jobID, event.Err)
				}
				return fmt.Errorf("implant mTLS listener job %d stopped before accepting connections", jobID)
			}
		case <-ticker.C:
		}
	}
}

func waitForMatchingSession(
	ctx context.Context,
	pump *eventPump,
	process *managedProcess,
	listenerJobID uint32,
	implantName string,
) (*clientpb.Event, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-process.done:
			for {
				select {
				case event, ok := <-pump.events:
					if !ok {
						return nil, eventStreamError(pump, "event stream ended while waiting for SessionOpened")
					}
					matched, err := matchingSession(event, listenerJobID, implantName)
					if matched || err != nil {
						return event, err
					}
				default:
					return nil, process.failure("generated session exited before SessionOpened")
				}
			}
		case event, ok := <-pump.events:
			if !ok {
				return nil, eventStreamError(pump, "event stream ended while waiting for SessionOpened")
			}
			matched, err := matchingSession(event, listenerJobID, implantName)
			if matched || err != nil {
				return event, err
			}
		}
	}
}

func matchingSession(event *clientpb.Event, listenerJobID uint32, implantName string) (bool, error) {
	if err := listenerStoppedError(event, listenerJobID); err != nil {
		return false, err
	}
	if event.EventType != consts.SessionOpenedEvent || event.Session == nil || event.Session.Name != implantName {
		return false, nil
	}
	if event.Session.OS != targetOS || event.Session.Arch != targetArch {
		return true, fmt.Errorf("session target = %s/%s, want %s/%s", event.Session.OS, event.Session.Arch, targetOS, targetArch)
	}
	if event.Session.ID == "" {
		return true, errors.New("matching SessionOpened event has an empty ID")
	}
	return true, nil
}

func waitForMatchingBeacon(
	ctx context.Context,
	pump *eventPump,
	process *managedProcess,
	listenerJobID uint32,
	implantName string,
) (*clientpb.Beacon, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-process.done:
			for {
				select {
				case event, ok := <-pump.events:
					if !ok {
						return nil, eventStreamError(pump, "event stream ended while waiting for BeaconRegistered")
					}
					beacon, matched, err := matchingBeacon(event, listenerJobID, implantName)
					if matched || err != nil {
						return beacon, err
					}
				default:
					return nil, process.failure("generated beacon exited before BeaconRegistered")
				}
			}
		case event, ok := <-pump.events:
			if !ok {
				return nil, eventStreamError(pump, "event stream ended while waiting for BeaconRegistered")
			}
			beacon, matched, err := matchingBeacon(event, listenerJobID, implantName)
			if matched || err != nil {
				return beacon, err
			}
		}
	}
}

func matchingBeacon(event *clientpb.Event, listenerJobID uint32, implantName string) (*clientpb.Beacon, bool, error) {
	if err := listenerStoppedError(event, listenerJobID); err != nil {
		return nil, false, err
	}
	if event.EventType != consts.BeaconRegisteredEvent {
		return nil, false, nil
	}
	beacon := &clientpb.Beacon{}
	if err := proto.Unmarshal(event.Data, beacon); err != nil {
		return nil, false, fmt.Errorf("decode BeaconRegistered event: %w", err)
	}
	if beacon.Name != implantName {
		return nil, false, nil
	}
	if beacon.OS != targetOS || beacon.Arch != targetArch {
		return beacon, true, fmt.Errorf("beacon target = %s/%s, want %s/%s", beacon.OS, beacon.Arch, targetOS, targetArch)
	}
	if beacon.ID == "" {
		return beacon, true, errors.New("matching BeaconRegistered event has an empty ID")
	}
	return beacon, true, nil
}

func listenerStoppedError(event *clientpb.Event, jobID uint32) error {
	if event.EventType != consts.JobStoppedEvent || event.Job == nil || event.Job.ID != jobID {
		return nil
	}
	if event.Err != "" {
		return fmt.Errorf("implant mTLS listener job %d stopped: %s", jobID, event.Err)
	}
	return fmt.Errorf("implant mTLS listener job %d stopped before implant verification completed", jobID)
}

func eventStreamError(pump *eventPump, message string) error {
	select {
	case err := <-pump.errors:
		if err != nil {
			return fmt.Errorf("%s: %w", message, err)
		}
	default:
	}
	return errors.New(message)
}
