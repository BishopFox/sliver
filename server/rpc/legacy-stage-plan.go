package rpc

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/buildplan"
	"github.com/bishopfox/sliver/server/buildtransform"
)

const (
	stageAESKeyRef = "stage/aes/key"
	stageAESIVRef  = "stage/aes/iv"
	stageRC4KeyRef = "stage/rc4/key"
)

// legacyStageBuildRequest extends an already resolved legacy core route with
// explicit, ordered stage transforms. Key material is represented only by
// per-build references in the plan and is never copied into a build record.
func legacyStageBuildRequest(base buildplan.Request, coreOutput buildplan.Artifact, req *clientpb.GenerateStageReq) (buildplan.Request, buildtransform.ResolveInput, error) {
	if req == nil {
		return buildplan.Request{}, nil, errors.New("missing stage request")
	}
	if req.AESEncryptKey != "" && req.RC4EncryptKey != "" {
		return buildplan.Request{}, nil, errors.New("AES and RC4 encryption cannot be combined")
	}
	if req.Compress != "" {
		switch req.Compress {
		case "zlib", "gzip", "deflate", "deflate9":
		default:
			return buildplan.Request{}, nil, fmt.Errorf("unsupported stage compression %q", req.Compress)
		}
	}
	if req.AESEncryptKey != "" && len(req.AESEncryptKey) != 16 && len(req.AESEncryptKey) != 32 {
		return buildplan.Request{}, nil, errors.New("stage AES key must be 16 or 32 bytes")
	}
	if req.AESEncryptIv != "" && req.AESEncryptKey == "" {
		return buildplan.Request{}, nil, errors.New("stage AES IV requires an AES key")
	}
	if req.AESEncryptIv != "" && len(req.AESEncryptIv) != 16 {
		return buildplan.Request{}, nil, errors.New("stage AES IV must be 16 bytes")
	}
	if len(req.RC4EncryptKey) > 256 {
		return buildplan.Request{}, nil, errors.New("stage RC4 key must be at most 256 bytes")
	}

	request := base
	request.FinalFormat = "stage"
	request.CoreOutput = &coreOutput
	request.Transforms = append([]buildplan.TransformRequest(nil), base.Transforms...)
	request.Transforms = append(request.Transforms, buildplan.TransformRequest{
		Spec:    buildplan.Reference{Name: buildtransform.StageFrameSpec},
		Options: []buildplan.Option{stageBooleanOption("prependSize", req.PrependSize)},
	})
	if req.Compress != "" {
		request.Transforms = append(request.Transforms, buildplan.TransformRequest{
			Spec:    buildplan.Reference{Name: buildtransform.StageCompressSpec},
			Options: []buildplan.Option{stageStringOption("algorithm", req.Compress)},
		})
	}
	secrets := make(map[string][]byte, 2)
	if req.AESEncryptKey != "" {
		iv := req.AESEncryptIv
		if iv == "" {
			iv = "0000000000000000"
		}
		secrets[stageAESKeyRef] = []byte(req.AESEncryptKey)
		secrets[stageAESIVRef] = []byte(iv)
		request.Transforms = append(request.Transforms, buildplan.TransformRequest{
			Spec: buildplan.Reference{Name: buildtransform.StageAESSpec},
			Options: []buildplan.Option{
				stageSecretOption("key", stageAESKeyRef),
				stageSecretOption("iv", stageAESIVRef),
			},
		})
	} else if req.RC4EncryptKey != "" {
		secrets[stageRC4KeyRef] = []byte(req.RC4EncryptKey)
		request.Transforms = append(request.Transforms, buildplan.TransformRequest{
			Spec:    buildplan.Reference{Name: buildtransform.StageRC4Spec},
			Options: []buildplan.Option{stageSecretOption("key", stageRC4KeyRef)},
		})
	}
	resolve := func(kind, name string) ([]byte, error) {
		if kind != "secret" {
			return nil, errors.New("unsupported stage input kind")
		}
		value, ok := secrets[name]
		if !ok {
			return nil, errors.New("stage input is unavailable")
		}
		return append([]byte(nil), value...), nil
	}
	return request, resolve, nil
}

func stageBooleanOption(name string, value bool) buildplan.Option {
	encoded := json.RawMessage("false")
	if value {
		encoded = json.RawMessage("true")
	}
	return buildplan.Option{Name: name, ValueJSON: encoded}
}

func stageStringOption(name, value string) buildplan.Option {
	encoded, _ := json.Marshal(value) // A string cannot fail JSON encoding.
	return buildplan.Option{Name: name, ValueJSON: encoded}
}

func stageSecretOption(name, refName string) buildplan.Option {
	encoded, _ := json.Marshal(struct {
		Kind string `json:"kind"`
		Name string `json:"name"`
	}{Kind: "secret", Name: refName}) // Fixed string fields cannot fail JSON encoding.
	return buildplan.Option{Name: name, ValueJSON: encoded}
}
