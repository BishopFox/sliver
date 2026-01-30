package shellcode

import (
	"github.com/bishopfox/sliver/server/encoders/shellcode/amd64"
	"github.com/bishopfox/sliver/server/encoders/shellcode/sgn"
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

type ShikataGaNaiEncoderAmd64 struct{}

func (e *ShikataGaNaiEncoderAmd64) Encode(data []byte, args ShellcodeEncoderArgs) ([]byte, error) {
	return sgn.EncodeShellcode(data, "amd64", args.Iterations, []byte{})
}

func (e *ShikataGaNaiEncoderAmd64) Description() string {
	return "Shikata Ga Nai encoder for AMD64"
}

type ShikataGaNaiEncoder386 struct{}

func (e *ShikataGaNaiEncoder386) Encode(data []byte, args ShellcodeEncoderArgs) ([]byte, error) {
	return sgn.EncodeShellcode(data, "386", args.Iterations, []byte{})
}

func (e *ShikataGaNaiEncoder386) Description() string {
	return "Shikata Ga Nai encoder for 386"
}

var (
	ShellcodeEncoders = map[string]map[string]ShellcodeEncoder{
		"amd64": {
			"xor":            &XorEncoder{},
			"xor_dynamic":    &XorDynamicEncoder{},
			"shikata_ga_nai": &ShikataGaNaiEncoderAmd64{},
		},
		"386": {
			"shikata_ga_nai": &ShikataGaNaiEncoder386{},
		},
	}
)
