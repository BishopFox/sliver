package sgn

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
	"errors"
	"fmt"

	"github.com/bishopfox/sliver/server/log"
	sgnpkg "github.com/moloch--/sgn/pkg"
)

var (
	sgnLog = log.NamedLogger("shellcode", "sgn")

	ErrFailedToEncode = errors.New("failed to encode shellcode")
)

// SGNConfig - Configuration for sgn
type SGNConfig struct {
	AppDir string

	Architecture   string // Binary architecture (32/64) (default 32)
	Asci           bool   // Generates a full ASCI printable payload (takes very long time to bruteforce)
	BadChars       []byte // Don't use specified bad characters given in hex format (\x00\x01\x02...)
	Iterations     int    // Number of times to encode the binary (increases overall size) (default 1)
	MaxObfuscation int    // Maximum number of bytes for obfuscation (default 20)
	PlainDecoder   bool   // Do not encode the decoder stub
	Safe           bool   // Do not modify any register values

	Verbose bool

	Output string
	Input  string
}

// EncodeShellcode - Encode a shellcode
func EncodeShellcode(shellcode []byte, arch string, iterations int, badChars []byte) ([]byte, error) {
	sgnLog.Infof("[sgn] EncodeShellcode: %d bytes", len(shellcode))

	encoder, err := sgnpkg.NewEncoder(64)
	if err != nil {
		fmt.Println(err)
		return nil, fmt.Errorf("%w: %s", ErrFailedToEncode, err)
	}

	data, err := encoder.Encode(shellcode)
	if err != nil {
		sgnLog.Errorf("[sgn] EncodeShellcode: %s", err)
		return nil, fmt.Errorf("%w: %s", ErrFailedToEncode, err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("%w: no data returned from encoder", ErrFailedToEncode)
	}

	return data, nil
}
