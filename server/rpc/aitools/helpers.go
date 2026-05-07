package aitools

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"google.golang.org/protobuf/proto"
)

type responseMessage interface {
	proto.Message
	GetResponse() *commonpb.Response
}

func targetSchemaProperties() map[string]any {
	return map[string]any{
		"session_id": map[string]any{"type": "string", "description": "Interactive session ID to run the command against."},
		"beacon_id":  map[string]any{"type": "string", "description": "Beacon ID to run the command against."},
	}
}

func mergeSchemaProperties(extra map[string]any) map[string]any {
	properties := targetSchemaProperties()
	for key, value := range extra {
		properties[key] = value
	}
	return properties
}

func callTargetRPC[T responseMessage](
	ctx context.Context,
	target toolTarget,
	invoke func(context.Context, *commonpb.Request) (T, error),
	newResponse func() T,
) (T, error) {
	var zero T

	req, isBeacon, callCtx, cancel, err := buildRequestContext(ctx, target)
	if err != nil {
		return zero, err
	}
	if cancel != nil {
		defer cancel()
	}

	resp, err := invoke(callCtx, req)
	if err != nil {
		return zero, err
	}
	if err := genericResponseError(resp.GetResponse()); err != nil {
		return zero, err
	}
	if isBeacon && resp.GetResponse() != nil && resp.GetResponse().Async {
		resolved := newResponse()
		if err := waitForBeaconTaskResponse(callCtx, resp.GetResponse().TaskID, resolved); err != nil {
			return zero, err
		}
		resp = resolved
		if err := genericResponseError(resp.GetResponse()); err != nil {
			return zero, err
		}
	}
	return resp, nil
}

func (e *executor) lookupTargetMetadata(ctx context.Context, sessionID, beaconID string) (*clientpb.Session, *clientpb.Beacon, error) {
	target, err := e.resolveTarget(sessionID, beaconID)
	if err != nil {
		return nil, nil, err
	}
	if e == nil || e.backend == nil {
		return nil, nil, fmt.Errorf("AI tool executor is unavailable")
	}

	if target.SessionID != "" {
		sessionsResp, err := e.backend.GetSessions(ctx, &commonpb.Empty{})
		if err != nil {
			return nil, nil, err
		}
		for _, session := range sessionsResp.GetSessions() {
			if session != nil && session.ID == target.SessionID {
				return session, nil, nil
			}
		}
		for _, session := range sessionsResp.GetSessions() {
			if session != nil && strings.HasPrefix(session.ID, target.SessionID) {
				return session, nil, nil
			}
		}
		return nil, nil, fmt.Errorf("session %q not found", target.SessionID)
	}

	beaconsResp, err := e.backend.GetBeacons(ctx, &commonpb.Empty{})
	if err != nil {
		return nil, nil, err
	}
	for _, beacon := range beaconsResp.GetBeacons() {
		if beacon != nil && beacon.ID == target.BeaconID {
			return nil, beacon, nil
		}
	}
	for _, beacon := range beaconsResp.GetBeacons() {
		if beacon != nil && strings.HasPrefix(beacon.ID, target.BeaconID) {
			return nil, beacon, nil
		}
	}
	return nil, nil, fmt.Errorf("beacon %q not found", target.BeaconID)
}

func bytesToTextAndBase64(data []byte) (string, string) {
	if len(data) == 0 {
		return "", ""
	}
	if utf8.Valid(data) {
		return string(data), ""
	}
	return "", base64.StdEncoding.EncodeToString(data)
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum)
}
