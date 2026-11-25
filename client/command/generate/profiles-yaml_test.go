package generate

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type profileNameCheckRPC struct {
	rpcpb.SliverRPCClient
	doc  *clientpb.BuildProfileDocument
	err  error
	name string
}

func (rpc *profileNameCheckRPC) GetBuildProfile(_ context.Context, ref *clientpb.BuildProfileRef, _ ...grpc.CallOption) (*clientpb.BuildProfileDocument, error) {
	rpc.name = ref.Name
	return rpc.doc, rpc.err
}

func TestNewProfileAuthoredYAMLGuard(t *testing.T) {
	for _, tc := range []struct {
		name    string
		doc     *clientpb.BuildProfileDocument
		rpcErr  error
		wantErr string
	}{
		{"authored", &clientpb.BuildProfileDocument{Name: "taken", Migrated: false}, nil, "export and edit its YAML"},
		{"migrated", &clientpb.BuildProfileDocument{Name: "taken", Migrated: true}, nil, ""},
		{"missing", nil, status.Error(codes.NotFound, "missing"), ""},
		{"older server", nil, status.Error(codes.Unimplemented, "missing RPC"), ""},
		{"lookup failed", nil, status.Error(codes.PermissionDenied, "denied"), "check existing profile"},
		{"empty response", nil, nil, "no profile status"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rpc := &profileNameCheckRPC{doc: tc.doc, err: tc.rpcErr}
			err := checkNewProfileName(context.Background(), "taken", &console.SliverClient{Rpc: rpc})
			if rpc.name != "taken" {
				t.Fatalf("looked up %q, want taken", rpc.name)
			}
			if tc.wantErr == "" && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)) {
				t.Fatalf("got %v, want %q", err, tc.wantErr)
			}
		})
	}
}

func TestProfileExportFileNamePortableAtNameLimit(t *testing.T) {
	first := strings.Repeat("a", 254) + "b"
	second := strings.Repeat("a", 254) + "c"
	firstFile := profileExportFileName(first)
	secondFile := profileExportFileName(second)
	if !safeProfileFileName(firstFile) || !safeProfileFileName(secondFile) || firstFile == secondFile {
		t.Fatalf("long profile export names collide or are unsafe: %q, %q", firstFile, secondFile)
	}
	if got := profileExportFileName("short"); got != "short.yaml" {
		t.Fatalf("ordinary profile export name = %q", got)
	}
	if got := profileExportFileName("CON"); !safeProfileFileName(got) {
		t.Fatalf("reserved profile export name = %q", got)
	}
}

func TestProfileCommandsRegistered(t *testing.T) {
	profiles := commandByUse(Commands(&console.SliverClient{}), consts.ProfilesStr)
	if profiles == nil {
		t.Fatal("missing profiles command")
	}
	for _, use := range []string{"import <file.yaml>", "validate <file.yaml>", "export <name>"} {
		if commandByUse(profiles.Commands(), use) == nil {
			t.Fatalf("missing profiles %s", use)
		}
	}
	for _, use := range []string{consts.GenerateStr, consts.StageStr} {
		cmd := commandByUse(profiles.Commands(), use)
		if cmd == nil {
			t.Fatalf("missing profiles %s", use)
		}
		for _, flag := range []string{"input-artifact", "secret-file"} {
			if cmd.Flags().Lookup(flag) == nil {
				t.Fatalf("profiles %s is missing --%s", use, flag)
			}
		}
	}
}

func TestProfileInfoMigrationDiagnostic(t *testing.T) {
	doc := &clientpb.BuildProfileDocument{
		Name: "old-profile", Revision: 3, Migrated: true,
		MigrationError: "unsupported legacy setting",
	}
	info, ok := profileMigrationInfo(doc)
	if !ok || !strings.Contains(info, "Profile: old-profile (revision 3)") ||
		!strings.Contains(info, "Migration diagnostic: unsupported legacy setting") {
		t.Fatalf("missing migration diagnostic: ok=%t info=%q", ok, info)
	}
	for _, other := range []*clientpb.BuildProfileDocument{
		{Migrated: true, MigrationError: ""},
		{Migrated: false, MigrationError: "bad", YAML: ""},
		{Migrated: true, MigrationError: "bad", YAML: "kind: BuildProfile", FinalArtifact: &clientpb.BuildArtifactType{}},
	} {
		if _, ok := profileMigrationInfo(other); ok {
			t.Fatalf("unexpected migration diagnostic for %+v", other)
		}
	}
}

func TestProfileInputsExactSlots(t *testing.T) {
	raw := `perBuildInputs:
  - name: donor-pe
    kind: input-artifact
  - name: aes-key
    kind: secret
`
	inputs := []*clientpb.BuildInput{{Name: "donor-pe", Kind: "input-artifact"}, {Name: "aes-key", Kind: "secret"}}
	if err := validateDeclaredProfileInputs(raw, inputs); err != nil {
		t.Fatalf("valid inputs rejected: %v", err)
	}
	for _, tc := range []struct {
		name   string
		inputs []*clientpb.BuildInput
		want   string
	}{
		{"missing", inputs[:1], "--secret-file aes-key=path"},
		{"wrong kind", []*clientpb.BuildInput{{Name: "donor-pe", Kind: "secret"}}, "wrong kind"},
		{"undeclared", append(append([]*clientpb.BuildInput{}, inputs...), &clientpb.BuildInput{Name: "other", Kind: "secret"}), "undeclared"},
		{"duplicate", append(append([]*clientpb.BuildInput{}, inputs...), inputs[0]), "duplicated"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateDeclaredProfileInputs(raw, tc.inputs)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want error containing %q", err, tc.want)
			}
		})
	}
}

//nolint:gocyclo // The test exercises the same bounded input handling for accepted and rejected files.
func TestProfileInputBounds(t *testing.T) {
	dir := t.TempDir()
	secret := filepath.Join(dir, "secret")
	if err := os.WriteFile(secret, []byte("test-key"), 0o600); err != nil {
		t.Fatal(err)
	}
	input, err := profileInputFromFlag("secret", "aes-key="+secret)
	if err != nil || input.Kind != "secret" || input.Name != "aes-key" || string(input.Data) != "test-key" {
		t.Fatalf("secret input: input=%v err=%v", input, err)
	}
	oversizeSecret := filepath.Join(dir, "oversize-secret")
	file, err := os.Create(oversizeSecret)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate((1 << 20) + 1); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := profileInputFromFlag("secret", "aes-key="+oversizeSecret); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("oversize secret: %v", err)
	}
	oversizeArtifact := filepath.Join(dir, "oversize-artifact")
	file, err = os.Create(oversizeArtifact)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate((64 << 20) + 1); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := profileInputFromFlag("input-artifact", "donor-pe="+oversizeArtifact); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("oversize artifact: %v", err)
	}
	if _, err := profileInputFromFlag("secret", "bad_name="+secret); err == nil {
		t.Fatal("invalid input name accepted")
	}
	if _, err := profileInputFromFlag("secret", "aes-key="); err == nil {
		t.Fatal("missing path accepted")
	}
}

func TestProfileOverrides(t *testing.T) {
	dir := t.TempDir()
	artifactPath := filepath.Join(dir, "donor")
	secretPath := filepath.Join(dir, "key")
	for path, data := range map[string]string{artifactPath: "PE-data", secretPath: "secret"} {
		if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	cmd := &cobra.Command{}
	cmd.Flags().String("name", "", "")
	cmd.Flags().String("builder", "", "")
	cmd.Flags().Bool("external-builder", false, "")
	cmd.Flags().String("shellcode-encoder", "", "")
	cmd.Flags().StringArray("input-artifact", nil, "")
	cmd.Flags().StringArray("secret-file", nil, "")
	for name, value := range map[string]string{
		"name": "PayloadOne", "builder": "builder-a", "shellcode-encoder": "xor_dynamic",
		"input-artifact": "donor-pe=" + artifactPath, "secret-file": "aes-key=" + secretPath,
	} {
		if err := cmd.Flags().Set(name, value); err != nil {
			t.Fatal(err)
		}
	}
	request, err := profileExecuteRequestFromFlags(cmd, "my-profile")
	if err != nil {
		t.Fatal(err)
	}
	if request.Name != "my-profile" || request.BuildName != "payloadone" || request.Provider.GetMode() != "pinned" ||
		request.Provider.GetBuilder() != "builder-a" || request.ShellcodeEncoder != "xor-dynamic" || len(request.Inputs) != 2 {
		t.Fatalf("unexpected overrides: %+v", request)
	}
	if request.Inputs[0].Name != "donor-pe" || request.Inputs[1].Name != "aes-key" {
		t.Fatalf("unexpected input order: %+v", request.Inputs)
	}
	if _, err := canonicalEncoderOverride("bogus"); err == nil {
		t.Fatal("unsupported encoder accepted")
	}
}

func TestAuthoredProfileStageFlags(t *testing.T) {
	stage := &clientpb.BuildArtifactType{Kind: "stage", Container: "raw"}
	for _, flag := range []string{"aes-encrypt-key", "aes-encrypt-iv", "rc4-encrypt-key", "prepend-size", "compress"} {
		t.Run(flag, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().String("aes-encrypt-key", "", "")
			cmd.Flags().String("aes-encrypt-iv", "", "")
			cmd.Flags().String("rc4-encrypt-key", "", "")
			cmd.Flags().Bool("prepend-size", false, "")
			cmd.Flags().String("compress", "", "")
			value := "value"
			if flag == "prepend-size" {
				value = "false"
			}
			if err := cmd.Flags().Set(flag, value); err != nil {
				t.Fatal(err)
			}
			if err := validateAuthoredStageCommand(cmd, stage); err == nil || !strings.Contains(err.Error(), "profile YAML") {
				t.Fatalf("authored stage flag accepted: %v", err)
			}
		})
	}
	if err := validateAuthoredStageCommand(&cobra.Command{}, stage); err != nil {
		t.Fatal(err)
	}
	if err := validateAuthoredStageCommand(&cobra.Command{}, &clientpb.BuildArtifactType{Kind: "executable", Container: "pe"}); err == nil {
		t.Fatal("non-stage profile accepted")
	}
}

func TestProfileYAMLReadBound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "profile.yaml")
	if err := os.WriteFile(path, []byte("kind: BuildProfile\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readProfileYAML(path); err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(maxProfileYAMLBytes + 1); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := readProfileYAML(path); err == nil {
		t.Fatal("oversize YAML accepted")
	}
}
