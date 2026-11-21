//go:build sliver_generate

package rpc

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/generate"
	"google.golang.org/protobuf/proto"
)

//nolint:gocyclo // The artifact and stored record are checked against the same on-demand request.
func TestOnDemandShellcodeBuildRecord(t *testing.T) {
	setupGenerateSpoofTest(t)
	name := fmt.Sprintf("on-demand-shellcode-%d", time.Now().UnixNano())
	generatedName := ""
	t.Cleanup(func() {
		if generatedName != "" {
			_ = RemoveBuildByName(generatedName)
			_ = os.RemoveAll(filepath.Join(generate.GetSliversDir(), "windows", "amd64", generatedName))
		}
	})
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Minute)
	defer cancel()
	config := &clientpb.ImplantConfig{
		ID: "old-executable-config", GOOS: "windows", GOARCH: "amd64",
		Format: clientpb.OutputFormat_EXECUTABLE, Debug: true,
		ObfuscateSymbols: true, ShellcodeEncoder: clientpb.ShellcodeEncoder_XOR, SGNEnabled: true,
		C2: []*clientpb.ImplantC2{{URL: "mtls://127.0.0.1:8888"}}, HTTPC2ConfigName: "default",
	}
	before := proto.Clone(config)
	rpc := &Server{}
	generated, err := rpc.generateOnDemandShellcode(ctx, name, config, false)
	if err != nil {
		t.Fatalf("on-demand shellcode generation: %v", err)
	}
	generatedName = generated.GetImplantName()
	if !proto.Equal(config, before) || generatedName == name || !strings.HasPrefix(generatedName, name+"_") ||
		generated.GetImplantBuildID() == "" || len(generated.GetFile().GetData()) == 0 {
		t.Fatalf("generation mutated the source or returned no saved build: %+v", generated)
	}
	secondName, err := uniqueTaskBuildName(name, taskBuildNameExists)
	if err != nil || secondName == generatedName || !strings.HasPrefix(secondName, name+"_") {
		t.Fatalf("repeated allocation reused an on-demand name: first=%q second=%q err=%v", generatedName, secondName, err)
	}
	build, err := db.ImplantBuildByID(generated.ImplantBuildID)
	if err != nil || build == nil || build.Name != generatedName {
		t.Fatalf("saved on-demand build missing: %+v, %v", build, err)
	}
	savedConfig, err := db.ImplantConfigByID(build.ImplantConfigID)
	if err != nil || savedConfig == nil || savedConfig.Format != clientpb.OutputFormat_SHELLCODE ||
		savedConfig.ShellcodeEncoder != clientpb.ShellcodeEncoder_NONE || savedConfig.ObfuscateSymbols {
		t.Fatalf("saved config does not describe raw shellcode: %+v, %v", savedConfig, err)
	}
	record, err := rpc.GetBuildRecord(ctx, &clientpb.BuildRecordRef{ID: generated.ImplantBuildID})
	if err != nil || record == nil || record.Plan == nil || record.State != clientpb.BuildRecordState_BUILD_RECORD_SUCCEEDED ||
		len(record.Plan.Steps) != 2 || record.Plan.Steps[0].Spec.Name != "builtin/sliver" ||
		record.Plan.Steps[1].Spec.Name != "builtin/donut" {
		t.Fatalf("on-demand build did not use the resolved core and converter: %+v, %v", record, err)
	}
	hash := sha256.Sum256(generated.File.Data)
	if record.FinalArtifact == nil || record.FinalArtifact.SHA256 != hex.EncodeToString(hash[:]) {
		t.Fatalf("build record final bytes differ from shellcode: %+v", record.FinalArtifact)
	}
	stored, err := rpc.GetBuildArtifact(ctx, &clientpb.BuildRecordRef{ID: generated.ImplantBuildID})
	if err != nil || !bytes.Equal(stored.GetFile().GetData(), generated.File.Data) {
		t.Fatalf("recorded shellcode differs from returned bytes: %v", err)
	}
	reused, arch, err := getSliverShellcode(generatedName)
	if err != nil || arch != "amd64" || !bytes.Equal(reused, generated.File.Data) {
		t.Fatalf("saved shellcode cannot be reused: arch=%q err=%v", arch, err)
	}
}
