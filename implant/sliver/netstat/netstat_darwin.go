package netstat

var skStates = [...]string{
	"UNKNOWN",
}

// Socket states
const (
	Established SkState = 0x01
	SynSent             = 0x02
	SynRecv             = 0x03
	FinWait1            = 0x04
	FinWait2            = 0x05
	TimeWait            = 0x06
	Close               = 0x07
	CloseWait           = 0x08
	LastAck             = 0x09
	Listen              = 0x0a
	Closing             = 0x0b
)

func osTCPSocks(accept AcceptFn) ([]SockTabEntry, error) {
	return nil, nil
}

func osTCP6Socks(accept AcceptFn) ([]SockTabEntry, error) {
	return nil, nil
}

func osUDPSocks(accept AcceptFn) ([]SockTabEntry, error) {
	return nil, nil
}

func osUDP6Socks(accept AcceptFn) ([]SockTabEntry, error) {
	return nil, nil
}
