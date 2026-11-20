package rpc

import (
	"bytes"
	"crypto/sha256"
	"debug/pe"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/buildplan"
	"github.com/bishopfox/sliver/server/buildtransform"
	"github.com/bishopfox/sliver/server/generate"
	"google.golang.org/protobuf/proto"
)

const legacyMetadataRefPrefix = "metadata-sha256:"

// legacyMetadataBuildRequest binds a generation-time PE transform to one
// per-build metadata configuration. The plan stores only its content hash and
// typed donor reference; no donor bytes or resource contents enter the record.
func legacyMetadataBuildRequest(base buildplan.Request, config *clientpb.ImplantConfig,
	metadata *clientpb.SpoofMetadataConfig) (buildplan.Request, buildtransform.ResolveInput, error) {
	if metadata == nil {
		return base, nil, nil
	}
	if config == nil || config.GOOS != "windows" {
		return buildplan.Request{}, nil, errors.New("PE metadata requires a Windows target")
	}
	switch config.Format {
	case clientpb.OutputFormat_EXECUTABLE, clientpb.OutputFormat_SERVICE, clientpb.OutputFormat_SHARED_LIB:
	default:
		return buildplan.Request{}, nil, errors.New("PE metadata requires an executable, service, or shared-library output")
	}
	converted, err := generate.SpoofMetadataConfigFromProto(metadata)
	if err != nil {
		return buildplan.Request{}, nil, err
	}
	if converted.PE == nil || converted.PE.Source == nil || len(converted.PE.Source.Data) == 0 {
		return buildplan.Request{}, nil, errors.New("PE metadata requires donor bytes")
	}
	donor := converted.PE.Source.Data
	if len(donor) > 64<<20 {
		return buildplan.Request{}, nil, errors.New("PE metadata donor exceeds 64 MiB")
	}
	donorPE, err := pe.NewFile(bytes.NewReader(donor))
	if err != nil {
		return buildplan.Request{}, nil, fmt.Errorf("invalid PE metadata donor: %w", err)
	}
	wantMachine := map[string]uint16{"386": pe.IMAGE_FILE_MACHINE_I386, "amd64": pe.IMAGE_FILE_MACHINE_AMD64,
		"arm64": pe.IMAGE_FILE_MACHINE_ARM64}[config.GOARCH]
	wantDLL := config.Format == clientpb.OutputFormat_SHARED_LIB
	gotDLL := donorPE.Characteristics&0x2000 != 0
	gotMachine := donorPE.Machine
	_ = donorPE.Close()
	if wantMachine == 0 || gotMachine != wantMachine || gotDLL != wantDLL {
		return buildplan.Request{}, nil, errors.New("PE metadata donor does not match target architecture or output type")
	}
	serialized, err := proto.MarshalOptions{Deterministic: true}.Marshal(metadata)
	if err != nil {
		return buildplan.Request{}, nil, fmt.Errorf("encode PE metadata configuration: %w", err)
	}
	if len(serialized) > 128<<20 {
		return buildplan.Request{}, nil, errors.New("PE metadata configuration exceeds 128 MiB")
	}
	hash := sha256.Sum256(serialized)
	refName := legacyMetadataRefPrefix + hex.EncodeToString(hash[:])
	reference, err := json.Marshal(map[string]string{"kind": "input-artifact", "name": refName})
	if err != nil {
		return buildplan.Request{}, nil, err
	}
	base.Transforms = append(append([]buildplan.TransformRequest(nil), base.Transforms...), buildplan.TransformRequest{
		Spec:    buildplan.Reference{Name: buildtransform.PEMetadataSpec},
		Options: []buildplan.Option{{Name: "donorArtifact", ValueJSON: reference}},
	})
	resolve := func(kind, name string) ([]byte, error) {
		if name != refName {
			return nil, errors.New("PE metadata input is unavailable")
		}
		switch kind {
		case "input-artifact":
			return bytes.Clone(donor), nil
		case "metadata-config":
			return bytes.Clone(serialized), nil
		default:
			return nil, errors.New("PE metadata input kind is unavailable")
		}
	}
	return base, resolve, nil
}
