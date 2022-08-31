package implant

import (
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

const defaultTimeout = 60

// ActiveImplant exposes common methods between
// Sliver clientpb.Session and clientpb.Beacon
// that are required by Operator implants
type ActiveImplant interface {
	GetID() string
	GetHostname() string
	GetPID() int32
	GetOS() string
	GetArch() string
	GetFilename() string
	GetReconnectInterval() int64
}

func MakeRequest(a ActiveImplant) *commonpb.Request {
	timeout := int64(defaultTimeout)
	req := &commonpb.Request{
		Timeout: timeout,
	}
	if a == nil {
		return nil
	}

	beacon, ok := a.(*clientpb.Beacon)
	if ok {
		req.BeaconID = beacon.ID
		req.Async = true
		return req
	}
	session, ok := a.(*clientpb.Session)
	if ok {
		req.SessionID = session.ID
		req.Async = false
		return req
	}

	return nil
}
