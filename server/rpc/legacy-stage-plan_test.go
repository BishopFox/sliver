package rpc

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/buildcatalog"
	"github.com/bishopfox/sliver/server/buildplan"
	"github.com/bishopfox/sliver/server/buildtransform"
)

func TestLegacyStageRequestResolvesPinnedShellcodeCore(t *testing.T) {
	t.Parallel()
	catalog, err := buildcatalog.Open(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	planner := buildplan.NewPlanner(catalog, func(string, buildcatalog.Snapshot, buildplan.Target, buildplan.Toolchain) bool { return true })
	config := &clientpb.ImplantConfig{
		GOOS: "windows", GOARCH: "amd64", Format: clientpb.OutputFormat_SHELLCODE,
		C2: []*clientpb.ImplantC2{{URL: "mtls://stage.example:8888"}},
	}
	base, err := legacyBuildRequest("stage-example", config)
	if err != nil {
		t.Fatal(err)
	}
	basePlan, err := planner.Plan(t.Context(), base)
	if err != nil {
		t.Fatal(err)
	}
	stage, _, err := legacyStageBuildRequest(base, basePlan.Steps[0].Output, &clientpb.GenerateStageReq{
		PrependSize: true, Compress: "gzip", RC4EncryptKey: "rc4-key",
	})
	if err != nil {
		t.Fatal(err)
	}
	stagePlan, err := planner.Plan(t.Context(), stage)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"builtin/sliver", "builtin/donut", buildtransform.StageFrameSpec, buildtransform.StageCompressSpec, buildtransform.StageRC4Spec}
	if len(stagePlan.Steps) != len(want) {
		t.Fatalf("steps = %d, want %d", len(stagePlan.Steps), len(want))
	}
	for i, step := range stagePlan.Steps {
		if step.Spec.Name != want[i] {
			t.Fatalf("step %d = %s, want %s", i, step.Spec.Name, want[i])
		}
	}
	if stagePlan.Steps[0].Output.GoBuildMode != "pie" || stagePlan.FinalArtifact.Kind != "stage" {
		t.Fatalf("incorrect stage route: %+v", stagePlan)
	}
}

//nolint:gocyclo // The same prepared plan must preserve stage order and exclude secret bytes.
func TestLegacyStageBuildRequestOrdersStepsAndKeepsSecretsOutOfPlan(t *testing.T) {
	t.Parallel()
	base := buildplan.Request{
		Name: "stage-test", FinalFormat: "shellcode",
		Transforms: []buildplan.TransformRequest{{Spec: buildplan.Reference{Name: "builtin/donut"}}},
	}
	coreOutput := buildplan.Artifact{Kind: "executable", Container: "pe", Target: buildplan.Target{OS: "windows", Arch: "amd64"}}
	req := &clientpb.GenerateStageReq{
		PrependSize: true, Compress: "zlib",
		AESEncryptKey: "0123456789abcdef", AESEncryptIv: "fedcba9876543210",
	}
	stage, resolve, err := legacyStageBuildRequest(base, coreOutput, req)
	if err != nil {
		t.Fatal(err)
	}
	if stage.FinalFormat != "stage" || stage.CoreOutput == nil || !reflect.DeepEqual(*stage.CoreOutput, coreOutput) {
		t.Fatalf("stage format or pinned core output: %+v", stage)
	}
	want := []string{"builtin/donut", buildtransform.StageFrameSpec, buildtransform.StageCompressSpec, buildtransform.StageAESSpec}
	if len(stage.Transforms) != len(want) {
		t.Fatalf("transform count = %d, want %d", len(stage.Transforms), len(want))
	}
	for i, transform := range stage.Transforms {
		if transform.Spec.Name != want[i] {
			t.Fatalf("transform %d = %q, want %q", i, transform.Spec.Name, want[i])
		}
	}
	encoded, err := json.Marshal(stage)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{req.AESEncryptKey, req.AESEncryptIv} {
		if bytes.Contains(encoded, []byte(secret)) {
			t.Fatal("literal stage key material was included in build plan")
		}
	}
	key, err := resolve("secret", stageAESKeyRef)
	if err != nil || string(key) != req.AESEncryptKey {
		t.Fatalf("resolve key: %v", err)
	}
	key[0] = '!'
	again, err := resolve("secret", stageAESKeyRef)
	if err != nil || string(again) != req.AESEncryptKey {
		t.Fatalf("resolver returned mutable shared key: %v", err)
	}
	if _, err := resolve("other", stageAESKeyRef); err == nil {
		t.Fatal("resolver accepted wrong input kind")
	}
	if base.FinalFormat != "shellcode" || len(base.Transforms) != 1 {
		t.Fatal("stage request mutated base request")
	}
}

func TestLegacyStageBuildRequestRejectsInvalidOptions(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name string
		req  *clientpb.GenerateStageReq
	}{
		{"missing", nil},
		{"both encryption methods", &clientpb.GenerateStageReq{AESEncryptKey: "0123456789abcdef", RC4EncryptKey: "rc4"}},
		{"unknown compression", &clientpb.GenerateStageReq{Compress: "brotli"}},
		{"short AES key", &clientpb.GenerateStageReq{AESEncryptKey: "short"}},
		{"IV without key", &clientpb.GenerateStageReq{AESEncryptIv: "0123456789abcdef"}},
		{"short AES IV", &clientpb.GenerateStageReq{AESEncryptKey: "0123456789abcdef", AESEncryptIv: "short"}},
		{"long RC4 key", &clientpb.GenerateStageReq{RC4EncryptKey: string(bytes.Repeat([]byte{'x'}, 257))}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, _, err := legacyStageBuildRequest(buildplan.Request{}, buildplan.Artifact{}, test.req); err == nil {
				t.Fatal("expected invalid stage request to fail before compilation")
			}
		})
	}
}

func TestLegacyStageBuildRequestDefaultsAESIV(t *testing.T) {
	t.Parallel()
	request, resolve, err := legacyStageBuildRequest(buildplan.Request{}, buildplan.Artifact{}, &clientpb.GenerateStageReq{
		AESEncryptKey: "0123456789abcdef",
	})
	if err != nil {
		t.Fatal(err)
	}
	if request.Transforms[len(request.Transforms)-1].Spec.Name != buildtransform.StageAESSpec {
		t.Fatal("missing AES stage transform")
	}
	iv, err := resolve("secret", stageAESIVRef)
	if err != nil || string(iv) != "0000000000000000" {
		t.Fatalf("default IV = %q, err = %v", iv, err)
	}
}
