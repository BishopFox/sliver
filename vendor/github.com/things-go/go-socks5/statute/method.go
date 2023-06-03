package statute

import (
	"io"
)

// MethodRequest is the negotiation method request packet
// The SOCKS handshake method request is formed as follows:
//
// +-----+----------+---------------+
// | VER | NMETHODS |    METHODS    |
// +-----+----------+---------------+
// |  1  |     1    | X'00' - X'FF' |
// +-----+----------+---------------+
type MethodRequest struct {
	Ver      byte
	NMethods byte
	Methods  []byte // 1-255 bytes
}

// NewMethodRequest new negotiation method request
func NewMethodRequest(ver byte, medthods []byte) MethodRequest {
	return MethodRequest{
		ver,
		byte(len(medthods)),
		medthods,
	}
}

// ParseMethodRequest parse method request.
func ParseMethodRequest(r io.Reader) (mr MethodRequest, err error) {
	// Read the version byte
	tmp := []byte{0}
	if _, err = r.Read(tmp); err != nil {
		return
	}
	mr.Ver = tmp[0]

	// Read number method
	if _, err = r.Read(tmp); err != nil {
		return
	}
	mr.NMethods, mr.Methods = tmp[0], make([]byte, tmp[0])
	// read methods
	_, err = io.ReadAtLeast(r, mr.Methods, int(mr.NMethods))
	return
}

// Bytes method request to bytes
func (sf MethodRequest) Bytes() []byte {
	b := make([]byte, 0, 2+sf.NMethods)
	b = append(b, sf.Ver, sf.NMethods)
	b = append(b, sf.Methods...)
	return b
}

// MethodReply is the negotiation method reply packet
// The SOCKS handshake method response is formed as follows:
//
//	+-----+--------+
//	| VER | METHOD |
//	+-----+--------+
//	|  1  |     1  |
//	+-----+--------+
type MethodReply struct {
	Ver    byte
	Method byte
}

// ParseMethodReply parse method reply.
func ParseMethodReply(r io.Reader) (n MethodReply, err error) {
	bb := []byte{0, 0}
	if _, err = io.ReadFull(r, bb); err != nil {
		return
	}
	n.Ver, n.Method = bb[0], bb[1]
	return
}
