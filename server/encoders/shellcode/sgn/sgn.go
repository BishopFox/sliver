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
	"bytes"
	cryptorand "crypto/rand"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/bishopfox/sliver/server/log"
	sgnpkg "github.com/moloch--/sgn/pkg"
)

var (
	sgnLog = log.NamedLogger("shellcode", "sgn")

	ErrFailedToEncode = errors.New("failed to encode shellcode")
)

// SGNConfig - Configuration for sgn
type SGNConfig struct {
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
	cfg := SGNConfig{
		Architecture:   arch,
		Iterations:     iterations,
		BadChars:       badChars,
		PlainDecoder:   false,
		Safe:           true,
		MaxObfuscation: 100,
	}
	return EncodeShellcodeWithConfig(shellcode, cfg)
}

// EncodeShellcodeWithConfig encodes a shellcode payload using the supplied SGNConfig.
func EncodeShellcodeWithConfig(shellcode []byte, cfg SGNConfig) ([]byte, error) {
	return encodeShellcodeWithConfig(shellcode, cfg, cryptorand.Reader)
}

func encodeShellcodeWithConfig(shellcode []byte, cfg SGNConfig, randomness io.Reader) ([]byte, error) {
	sgnLog.Infof("[sgn] EncodeShellcode: %d bytes", len(shellcode))

	if len(shellcode) == 0 {
		return nil, fmt.Errorf("%w: empty payload", ErrFailedToEncode)
	}

	arch, err := parseArchitecture(cfg.Architecture)
	if err != nil {
		sgnLog.Errorf("[sgn] EncodeShellcode invalid architecture %q: %s", cfg.Architecture, err)
		return nil, fmt.Errorf("%w: %s", ErrFailedToEncode, err)
	}

	if cfg.Iterations < 0 {
		cfg.Iterations = 0
	}

	needsConstraints := cfg.Asci || len(cfg.BadChars) > 0
	attempts := 1
	if needsConstraints {
		attempts = 64
	}

	for attempt := 1; attempt <= attempts; attempt++ {
		encoder, err := newEncoderWithConfig(arch, cfg)
		if err != nil {
			sgnLog.Errorf("[sgn] EncodeShellcode setup failed: %s", err)
			return nil, fmt.Errorf("%w: %s", ErrFailedToEncode, err)
		}

		var entropy [1 + sgnpkg.RandomSeedSize]byte
		if _, err := io.ReadFull(randomness, entropy[:]); err != nil {
			return nil, fmt.Errorf("%w: read encoder randomness: %w", ErrFailedToEncode, err)
		}
		encoder.Seed = entropy[0]
		var randomSeed sgnpkg.RandomSeed
		copy(randomSeed[:], entropy[1:])

		data, err := encoder.EncodeWithSeed(shellcode, randomSeed)
		if err != nil {
			sgnLog.Errorf("[sgn] EncodeShellcode attempt %d failed: %s", attempt, err)
			return nil, fmt.Errorf("%w: %s", ErrFailedToEncode, err)
		}

		if len(data) == 0 {
			return nil, fmt.Errorf("%w: attempt %d returned empty payload", ErrFailedToEncode, attempt)
		}

		if !needsConstraints || meetsConstraints(data, cfg) {
			return data, nil
		}
	}

	return nil, fmt.Errorf("%w: unable to satisfy encoding constraints", ErrFailedToEncode)
}

func newEncoderWithConfig(arch int, cfg SGNConfig) (*sgnpkg.Encoder, error) {
	encoder, err := sgnpkg.NewEncoder(arch)
	if err != nil {
		return nil, err
	}

	if cfg.MaxObfuscation > 0 {
		encoder.ObfuscationLimit = cfg.MaxObfuscation
	}

	encoder.PlainDecoder = cfg.PlainDecoder
	encoder.SaveRegisters = cfg.Safe

	if cfg.Iterations > 0 {
		encoder.EncodingCount = cfg.Iterations
	}

	return encoder, nil
}

func meetsConstraints(data []byte, cfg SGNConfig) bool {
	if cfg.Asci && !isASCIIPrintable(data) {
		return false
	}
	if len(cfg.BadChars) == 0 {
		return true
	}
	for _, bad := range cfg.BadChars {
		if bytes.IndexByte(data, bad) != -1 {
			return false
		}
	}
	return true
}

func isASCIIPrintable(data []byte) bool {
	for _, b := range data {
		if b < 0x20 || b > 0x7e {
			return false
		}
	}
	return true
}

func parseArchitecture(arch string) (int, error) {
	if arch == "" {
		return 64, nil
	}

	normalized := strings.ToLower(strings.TrimSpace(arch))
	switch normalized {
	case "amd64", "x86_64", "x64", "win64", "64":
		return 64, nil
	case "386", "x86", "i386", "ia32", "win32", "32":
		return 32, nil
	}

	if value, err := strconv.Atoi(normalized); err == nil {
		switch value {
		case 32, 386:
			return 32, nil
		case 64:
			return 64, nil
		}
	}

	return 0, fmt.Errorf("unsupported architecture %q", arch)
}
