package statute

import (
	"errors"
)

// VersionSocks5 socks protocol version
const VersionSocks5 = byte(0x05)

// request command defined
const (
	CommandConnect   = byte(0x01)
	CommandBind      = byte(0x02)
	CommandAssociate = byte(0x03)
)

// method defined
const (
	MethodNoAuth       = byte(0x00)
	MethodGSSAPI       = byte(0x01) // TODO: not support now
	MethodUserPassAuth = byte(0x02)
	MethodNoAcceptable = byte(0xff)
)

// address type defined
const (
	ATYPIPv4   = byte(0x01)
	ATYPDomain = byte(0x03)
	ATYPIPv6   = byte(0x04)
)

// reply status
const (
	RepSuccess uint8 = iota
	RepServerFailure
	RepRuleFailure
	RepNetworkUnreachable
	RepHostUnreachable
	RepConnectionRefused
	RepTTLExpired
	RepCommandNotSupported
	RepAddrTypeNotSupported
	// 0x09 - 0xff unassigned
)

// auth defined
const (
	// user password version
	UserPassAuthVersion = byte(0x01)
	// auth status
	AuthSuccess = byte(0x00)
	AuthFailure = byte(0x01)
)

// error defined
var (
	ErrUnrecognizedAddrType = errors.New("unrecognized address type")
	ErrNotSupportVersion    = errors.New("not support version")
	ErrNotSupportMethod     = errors.New("not support method")
)
