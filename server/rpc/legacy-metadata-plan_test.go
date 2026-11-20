package rpc

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/buildplan"
	"github.com/bishopfox/sliver/server/buildtransform"
)

//nolint:gocyclo // The digest and absence of donor bytes are asserted against the same request.
func TestLegacyMetadataBuildRequestFreezesDigestWithoutDonorBytes(t *testing.T) {
	donor := testCorePE()
	base := buildplan.Request{Target: buildplan.Target{OS: "windows", Arch: "amd64"}, FinalFormat: "exe"}
	config := &clientpb.ImplantConfig{GOOS: "windows", GOARCH: "amd64", Format: clientpb.OutputFormat_EXECUTABLE}
	metadata := &clientpb.SpoofMetadataConfig{PE: &clientpb.PESpoofMetadataConfig{
		Source:            &clientpb.SpoofMetadataFile{Name: "donor.exe", Data: donor},
		ResourceDirectory: &clientpb.IMAGE_RESOURCE_DIRECTORY{TimeDateStamp: 1234},
	}}
	request, resolve, err := legacyMetadataBuildRequest(base, config, metadata)
	if err != nil {
		t.Fatal(err)
	}
	if len(request.Transforms) != 1 || request.Transforms[0].Spec.Name != buildtransform.PEMetadataSpec {
		t.Fatalf("metadata transform was not appended: %+v", request.Transforms)
	}
	rawPlan, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(rawPlan, donor) || bytes.Contains(rawPlan, []byte("donor.exe")) ||
		bytes.Contains(rawPlan, []byte("1234")) {
		t.Fatal("donor bytes or override fields leaked into frozen plan")
	}
	var ref struct{ Kind, Name string }
	if err := json.Unmarshal(request.Transforms[0].Options[0].ValueJSON, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.Kind != "input-artifact" || !strings.HasPrefix(ref.Name, legacyMetadataRefPrefix) {
		t.Fatalf("metadata plan has no typed content reference: %+v", ref)
	}
	resolvedDonor, err := resolve("input-artifact", ref.Name)
	if err != nil || !bytes.Equal(resolvedDonor, donor) {
		t.Fatalf("donor did not resolve: %v", err)
	}
	resolvedConfig, err := resolve("metadata-config", ref.Name)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(resolvedConfig)
	if ref.Name != legacyMetadataRefPrefix+hex.EncodeToString(hash[:]) {
		t.Fatal("metadata reference does not bind full config bytes")
	}
	resolvedDonor[0] ^= 0xff
	if donor[0] != 'M' {
		t.Fatal("resolver exposed mutable donor input")
	}
}

func TestLegacyMetadataBuildRequestRejectsUnsupportedTargetAndDonor(t *testing.T) {
	donor := testCorePE()
	metadata := &clientpb.SpoofMetadataConfig{PE: &clientpb.PESpoofMetadataConfig{
		Source: &clientpb.SpoofMetadataFile{Data: donor},
	}}
	for _, config := range []*clientpb.ImplantConfig{
		{GOOS: "linux", GOARCH: "amd64", Format: clientpb.OutputFormat_EXECUTABLE},
		{GOOS: "windows", GOARCH: "amd64", Format: clientpb.OutputFormat_SHELLCODE},
		{GOOS: "windows", GOARCH: "386", Format: clientpb.OutputFormat_EXECUTABLE},
		{GOOS: "windows", GOARCH: "amd64", Format: clientpb.OutputFormat_SHARED_LIB},
	} {
		if _, _, err := legacyMetadataBuildRequest(buildplan.Request{}, config, metadata); err == nil {
			t.Fatalf("accepted incompatible target %+v", config)
		}
	}
}
