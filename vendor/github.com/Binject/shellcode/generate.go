package shellcode

import (
	"errors"

	"github.com/Binject/shellcode/api"
)

// Generate - makes a shellcode from a registered template module
func Generate(os api.Os, arch api.Arch, name string, params api.Parameters) ([]byte, error) {

	gs := api.LookupShellCode(os, arch)
	for _, g := range gs {
		if g.Name == name {
			return g.Function(params)
		}
	}
	return nil, errors.New("No Matching Shellcode Found")
}
