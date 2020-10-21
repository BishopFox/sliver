package api

import "errors"

type ParamType byte

const (
	IP        ParamType = iota
	Port      ParamType = iota
	Entry     ParamType = iota
	Entry64   ParamType = iota
	ShellCode ParamType = iota
)

// Parameters - config arguments for shellcode generating modules
type Parameters struct {
	IP        string
	Port      uint16
	Entry     uint32
	Entry64   uint64
	ShellCode []byte
}

func (p Parameters) Require(types []ParamType) error {

	for _, t := range types {
		switch t {
		case IP:
			if len(p.IP) < 1 {
				return errors.New("Ip is a required parameter")
			}
		case Port:
			if p.Port == 0 {
				return errors.New("Port is a required parameter")
			}
		case Entry:
			if p.Entry == 0 {
				return errors.New("Entry is a required parameter")
			}
		case Entry64:
			if p.Entry64 == 0 {
				return errors.New("Entry64 is a required parameter")
			}
		case ShellCode:
			if p.Port == 0 {
				return errors.New("ShellCode is a required parameter")
			}
		}
	}
	return nil
}
