package statute

import (
	"fmt"
	"io"
)

// auth error defined
var (
	ErrUserAuthFailed  = fmt.Errorf("user authentication failed")
	ErrNoSupportedAuth = fmt.Errorf("no supported authentication mechanism")
)

// UserPassRequest is the negotiation user's password request packet
// The SOCKS handshake user's password request is formed as follows:
//
//	+--------------+------+----------+------+----------+
//	| USERPASS_VER | ULEN |   USER   | PLEN |   PASS   |
//	+--------------+------+----------+------+----------+
//	|      1      |   1  | Variable |   1  | Variable |
//	+--------------+------+----------+------+----------+
type UserPassRequest struct {
	Ver  byte
	Ulen byte
	Plen byte
	User []byte // 1-255 bytes
	Pass []byte // 1-255 bytes
}

// NewUserPassRequest new user's password request packet with ver,user, pass
func NewUserPassRequest(ver byte, user, pass []byte) UserPassRequest {
	return UserPassRequest{
		ver,
		byte(len(user)),
		byte(len(pass)),
		user,
		pass,
	}
}

// ParseUserPassRequest parse user's password request.
//
//nolint:nakedret
func ParseUserPassRequest(r io.Reader) (nup UserPassRequest, err error) {
	tmp := []byte{0, 0}

	// Get the version and username length
	if _, err = io.ReadAtLeast(r, tmp, 2); err != nil {
		return
	}
	nup.Ver, nup.Ulen = tmp[0], tmp[1]

	// Ensure the UserPass version
	if nup.Ver != UserPassAuthVersion {
		err = fmt.Errorf("unsupported auth version: %v", nup.Ver)
		return
	}

	// Get the user name
	nup.User = make([]byte, nup.Ulen)
	if _, err = io.ReadAtLeast(r, nup.User, int(nup.Ulen)); err != nil {
		return
	}

	// Get the password length
	if _, err = r.Read(tmp[:1]); err != nil {
		return
	}
	nup.Plen = tmp[0]

	// Get the password
	nup.Pass = make([]byte, nup.Plen)
	_, err = io.ReadAtLeast(r, nup.Pass, int(nup.Plen))
	return nup, err
}

// Bytes to bytes
func (sf UserPassRequest) Bytes() []byte {
	b := make([]byte, 0, 3+sf.Ulen+sf.Plen)
	b = append(b, sf.Ver, sf.Ulen)
	b = append(b, sf.User...)
	b = append(b, sf.Plen)
	b = append(b, sf.Pass...)
	return b
}

// UserPassReply is the negotiation user's password reply packet
// The SOCKS handshake user's password response is formed as follows:
//
//	+-----+--------+
//	| VER | status |
//	+-----+--------+
//	|  1  |     1  |
//	+-----+--------+
type UserPassReply struct {
	Ver    byte
	Status byte
}

// ParseUserPassReply parse user's password reply packet.
func ParseUserPassReply(r io.Reader) (upr UserPassReply, err error) {
	bb := []byte{0, 0}
	if _, err = io.ReadFull(r, bb); err != nil {
		return
	}
	upr.Ver, upr.Status = bb[0], bb[1]
	return
}
