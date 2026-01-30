package shellcode

import (
	"github.com/bishopfox/sliver/server/encoders/shellcode/amd64"
)

type ShellcodeEncoderArgs struct {
	Iterations int
	Key        []byte
}

type ShellcodeEncoder interface {
	Description() string
	Encode(data []byte, args ShellcodeEncoderArgs) ([]byte, error)
}

type XorEncoder struct{}

func (e *XorEncoder) Encode(data []byte, args ShellcodeEncoderArgs) ([]byte, error) {
	for i := 0; i < args.Iterations; i++ {
		encoded, err := amd64.Xor(data, args.Key)
		if err != nil {
			return nil, err
		}
		data = encoded
	}
	return data, nil
}

func (e *XorEncoder) Description() string {
	return "Basic XOR encoder for AMD64"
}

type XorDynamicEncoder struct{}

func (e *XorDynamicEncoder) Encode(data []byte, args ShellcodeEncoderArgs) ([]byte, error) {
	for i := 0; i < args.Iterations; i++ {
		encoded, err := amd64.XorDynamic(data, args.Key)
		if err != nil {
			return nil, err
		}
		data = encoded
	}
	return data, nil
}

func (e *XorDynamicEncoder) Description() string {
	return "An x64 XOR encoder with dynamic key size for AMD64"
}

var (
	ShellcodeEncoders = map[string]map[string]ShellcodeEncoder{
		"amd64": {
			"xor":         &XorEncoder{},
			"xor_dynamic": &XorDynamicEncoder{},
		},
	}
)
