package generate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/forms"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/util"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"
)

const maxProfileYAMLBytes = 1 << 20

// GetProfileDefinition returns the server's canonical, validated profile.
// Consumers use FinalArtifact rather than inferring it from legacy flags.
func GetProfileDefinition(ctx context.Context, name string, con *console.SliverClient) (*clientpb.BuildProfileDocument, error) {
	if ctx == nil || con == nil || con.Rpc == nil || strings.TrimSpace(name) == "" {
		return nil, errors.New("missing profile name or connection")
	}
	return con.Rpc.GetBuildProfile(ctx, &clientpb.BuildProfileRef{Name: name})
}

// ExecuteProfile returns only bytes that match the completed build record.
// This is shared by the profile command and profile-consuming commands.
func ExecuteProfile(ctx context.Context, request *clientpb.BuildProfileExecuteRequest,
	con *console.SliverClient) (*clientpb.BuildRecord, *commonpb.File, error) {
	if ctx == nil || con == nil || con.Rpc == nil || request == nil || strings.TrimSpace(request.Name) == "" {
		return nil, nil, errors.New("missing profile request or connection")
	}
	record, err := con.Rpc.ExecuteProfileBuild(ctx, request)
	if err != nil {
		return nil, nil, err
	}
	if record == nil || record.State != clientpb.BuildRecordState_BUILD_RECORD_SUCCEEDED || record.FinalArtifact == nil {
		if failure := record.GetFailure(); failure != nil && failure.Message != "" {
			return record, nil, fmt.Errorf("profile build failed: %s", failure.Message)
		}
		return record, nil, errors.New("profile build did not produce a final artifact")
	}
	response, err := con.Rpc.GetBuildArtifact(ctx, &clientpb.BuildRecordRef{ID: record.ID})
	if err != nil {
		return record, nil, err
	}
	file := response.GetFile()
	metadata := response.GetMetadata()
	if file == nil || !safeProfileFileName(file.GetName()) || len(file.Data) == 0 || metadata == nil ||
		record.FinalArtifact.Type == nil || metadata.Type == nil {
		return record, nil, errors.New("profile build returned an empty or untyped artifact")
	}
	hash := sha256.Sum256(file.Data)
	checksum := hex.EncodeToString(hash[:])
	if checksum != record.FinalArtifact.SHA256 || checksum != metadata.SHA256 ||
		uint64(len(file.Data)) != record.FinalArtifact.SizeBytes || uint64(len(file.Data)) != metadata.SizeBytes ||
		!proto.Equal(record.FinalArtifact.Type, metadata.Type) ||
		record.FinalArtifact.FileName != file.Name || metadata.FileName != file.Name {
		return record, nil, errors.New("profile artifact differs from its frozen record")
	}
	return record, file, nil
}

func safeProfileFileName(name string) bool {
	return name != "" && name != "." && name != ".." && !strings.ContainsAny(name, "/\\")
}

func readProfileYAML(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxProfileYAMLBytes+1))
	if err != nil {
		return "", err
	}
	if len(data) == 0 || len(data) > maxProfileYAMLBytes || !utf8.Valid(data) {
		return "", errors.New("profile YAML is empty, too large, or invalid UTF-8")
	}
	return string(data), nil
}

func ProfilesImportCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	raw, err := readProfileYAML(args[0])
	if err != nil {
		con.PrintErrorf("Read profile YAML: %s\n", err)
		return
	}
	ctx, cancel := con.GrpcContext(cmd)
	defer cancel()
	doc, err := con.Rpc.ImportBuildProfile(ctx, &clientpb.BuildProfileImportRequest{YAML: raw})
	if err != nil {
		con.PrintErrorf("Import profile: %s\n", err)
		return
	}
	con.PrintInfof("Imported profile %s revision %d (%s/%s)\n", doc.Name, doc.Revision,
		doc.GetFinalArtifact().GetKind(), doc.GetFinalArtifact().GetContainer())
}

func ProfilesValidateCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	raw, err := readProfileYAML(args[0])
	if err != nil {
		con.PrintErrorf("Read profile YAML: %s\n", err)
		return
	}
	ctx, cancel := con.GrpcContext(cmd)
	defer cancel()
	doc, err := con.Rpc.ValidateBuildProfile(ctx, &clientpb.BuildProfileImportRequest{YAML: raw})
	if err != nil {
		con.PrintErrorf("Invalid profile: %s\n", err)
		return
	}
	con.PrintInfof("Valid profile %s revision %d; final artifact %s/%s\n", doc.Name, doc.Revision,
		doc.GetFinalArtifact().GetKind(), doc.GetFinalArtifact().GetContainer())
}

func ProfilesExportCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	ctx, cancel := con.GrpcContext(cmd)
	defer cancel()
	doc, err := GetProfileDefinition(ctx, args[0], con)
	if err != nil {
		con.PrintErrorf("Get profile: %s\n", err)
		return
	}
	if doc.YAML == "" || len(doc.YAML) > maxProfileYAMLBytes {
		con.PrintErrorf("Profile YAML is empty or too large\n")
		return
	}
	save, _ := cmd.Flags().GetString("save")
	path, err := profileExportPath(save, doc.Name+".yaml")
	if err != nil {
		con.PrintErrorf("Choose profile export path: %s\n", err)
		return
	}
	if err := writeProfileYAML(path, []byte(doc.YAML)); err != nil {
		con.PrintErrorf("Export profile: %s\n", err)
		return
	}
	con.PrintInfof("Exported profile %s revision %d to %s\n", doc.Name, doc.Revision, path)
}

func profileExportPath(save, fileName string) (string, error) {
	if !safeProfileFileName(fileName) {
		return "", errors.New("invalid profile export file name")
	}
	if save == "" {
		save = fileName
	}
	save = expandPath(save)
	info, err := os.Stat(save)
	if err == nil && info.IsDir() {
		save = filepath.Join(save, fileName)
	} else if os.IsNotExist(err) && (strings.HasSuffix(save, "/") || strings.HasSuffix(save, string(os.PathSeparator))) {
		if err := os.MkdirAll(save, 0700); err != nil {
			return "", err
		}
		save = filepath.Join(save, fileName)
	} else if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	path, err := filepath.Abs(save)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return "", err
	}
	if _, err := os.Stat(path); err == nil {
		var confirm bool
		_ = forms.Confirm("Overwrite existing profile YAML file?", &confirm)
		if !confirm {
			return "", errors.New("file already exists")
		}
	} else if !os.IsNotExist(err) {
		return "", err
	}
	return path, nil
}

func writeProfileYAML(path string, data []byte) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".profile-export-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if err := file.Chmod(0o600); err != nil {
		file.Close()
		return err
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}

type profileSummary struct {
	Target struct {
		OS   string `yaml:"os"`
		Arch string `yaml:"arch"`
	} `yaml:"target"`
	Mode string `yaml:"mode"`
	C2   struct {
		Endpoints []struct {
			Protocol string `yaml:"protocol"`
			Address  string `yaml:"address"`
		} `yaml:"endpoints"`
		ConnectionStrategy  string `yaml:"connectionStrategy"`
		ReconnectSeconds    uint32 `yaml:"reconnectSeconds"`
		PollTimeoutSeconds  uint32 `yaml:"pollTimeoutSeconds"`
		MaxConnectionErrors uint32 `yaml:"maxConnectionErrors"`
		HTTPC2Profile       string `yaml:"httpC2Profile"`
	} `yaml:"c2"`
	Beacon struct {
		IntervalSeconds uint32 `yaml:"intervalSeconds"`
		JitterSeconds   uint32 `yaml:"jitterSeconds"`
	} `yaml:"beacon"`
	Core struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version"`
	} `yaml:"core"`
	Transforms []struct {
		Spec struct {
			Name    string `yaml:"name"`
			Version string `yaml:"version"`
		} `yaml:"spec"`
	} `yaml:"transforms"`
	FinalOutput struct {
		Format string `yaml:"format"`
	} `yaml:"finalOutput"`
	Provider struct {
		Policy  string `yaml:"policy"`
		Builder string `yaml:"builder"`
	} `yaml:"provider"`
	PerBuildInputs []profileInputDeclaration `yaml:"perBuildInputs"`
}

type profileInputDeclaration struct {
	Name string `yaml:"name"`
	Kind string `yaml:"kind"`
}

func validateDeclaredProfileInputs(raw string, supplied []*clientpb.BuildInput) error {
	var profile struct {
		PerBuildInputs []profileInputDeclaration `yaml:"perBuildInputs"`
	}
	if err := yaml.Unmarshal([]byte(raw), &profile); err != nil {
		return fmt.Errorf("decode profile inputs: %w", err)
	}
	declared := make(map[string]string, len(profile.PerBuildInputs))
	for _, input := range profile.PerBuildInputs {
		declared[input.Name] = input.Kind
	}
	seen := make(map[string]bool, len(supplied))
	for _, input := range supplied {
		if input == nil || declared[input.Name] != input.Kind || seen[input.Name] {
			return fmt.Errorf("per-build input %q is undeclared, duplicated, or has the wrong kind", input.GetName())
		}
		seen[input.Name] = true
	}
	for _, input := range profile.PerBuildInputs {
		if seen[input.Name] {
			continue
		}
		flag := "--input-artifact"
		if input.Kind == "secret" {
			flag = "--secret-file"
		}
		return fmt.Errorf("profile requires per-build input %q (%s): use %s %s=path", input.Name, input.Kind, flag, input.Name)
	}
	return nil
}

func printBuildProfileInfo(doc *clientpb.BuildProfileDocument, con *console.SliverClient) error {
	if diagnostic, ok := profileMigrationInfo(doc); ok {
		con.Printf("%s", diagnostic)
		return nil
	}
	if doc == nil || doc.YAML == "" || doc.FinalArtifact == nil {
		return errors.New("profile has no canonical definition")
	}
	var details profileSummary
	if err := yaml.Unmarshal([]byte(doc.YAML), &details); err != nil {
		return err
	}
	con.Printf("Profile: %s (revision %d)\n", doc.Name, doc.Revision)
	con.Printf("Target: %s/%s  Mode: %s\n", details.Target.OS, details.Target.Arch, details.Mode)
	con.Printf("Core: %s@%s\n", details.Core.Name, details.Core.Version)
	for index, transform := range details.Transforms {
		con.Printf("Transform %d: %s@%s\n", index+1, transform.Spec.Name, transform.Spec.Version)
	}
	con.Printf("Final: %s (%s/%s)\n", details.FinalOutput.Format, doc.FinalArtifact.Kind, doc.FinalArtifact.Container)
	if details.Provider.Builder != "" {
		con.Printf("Provider: %s %s\n", details.Provider.Policy, details.Provider.Builder)
	} else {
		con.Printf("Provider: %s\n", details.Provider.Policy)
	}
	for _, endpoint := range details.C2.Endpoints {
		con.Printf("C2: %s %s\n", endpoint.Protocol, endpoint.Address)
	}
	con.Printf("C2 strategy: %s; reconnect %ds; poll timeout %ds; max errors %d\n",
		details.C2.ConnectionStrategy, details.C2.ReconnectSeconds, details.C2.PollTimeoutSeconds,
		details.C2.MaxConnectionErrors)
	if details.C2.HTTPC2Profile != "" {
		con.Printf("HTTP C2 profile: %s\n", details.C2.HTTPC2Profile)
	}
	if details.Mode == "beacon" {
		con.Printf("Beacon: interval %ds; jitter %ds\n", details.Beacon.IntervalSeconds, details.Beacon.JitterSeconds)
	}
	for _, input := range details.PerBuildInputs {
		con.Printf("Per-build input: %s (%s)\n", input.Name, input.Kind)
	}
	if doc.Migrated {
		con.Printf("Origin: migrated legacy profile\n")
		if doc.MigrationError != "" {
			con.Printf("Migration diagnostic: %s\n", doc.MigrationError)
		}
	}
	return nil
}

func profileMigrationInfo(doc *clientpb.BuildProfileDocument) (string, bool) {
	if doc == nil || !doc.Migrated || doc.MigrationError == "" || (doc.YAML != "" && doc.FinalArtifact != nil) {
		return "", false
	}
	return fmt.Sprintf("Profile: %s (revision %d)\nOrigin: migrated legacy profile\nMigration diagnostic: %s\n",
		doc.Name, doc.Revision, doc.MigrationError), true
}

func canonicalProfileGenerate(cmd *cobra.Command, con *console.SliverClient, profileName string) (bool, error) {
	request, err := profileExecuteRequestFromFlags(cmd, profileName)
	if err != nil {
		return true, err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	doc, err := GetProfileDefinition(ctx, profileName, con)
	if status.Code(err) == codes.Unimplemented {
		if len(request.Inputs) != 0 {
			return true, errors.New("the server does not support YAML per-build inputs")
		}
		return false, nil // Older servers retain the legacy profile RPCs.
	}
	if err != nil {
		return true, err
	}
	if doc.Migrated {
		if len(request.Inputs) != 0 {
			return true, errors.New("migrated legacy profiles do not declare YAML per-build inputs")
		}
		if doc.MigrationError != "" {
			con.PrintWarnf("Profile %s requires legacy generation: %s\n", profileName, doc.MigrationError)
			return false, nil
		}
		if request.Provider != nil {
			return false, nil // Legacy external builder selection remains available.
		}
	}
	if err := validateDeclaredProfileInputs(doc.YAML, request.Inputs); err != nil {
		return true, err
	}
	record, file, err := ExecuteProfile(ctx, request, con)
	if status.Code(err) == codes.Unimplemented {
		if doc.Migrated {
			return false, nil // Older servers retain the legacy profile RPCs.
		}
		return true, errors.New("the server cannot execute authored YAML profile pipelines")
	}
	if err != nil {
		return true, err
	}
	if !proto.Equal(doc.FinalArtifact, record.GetFinalArtifact().GetType()) {
		return true, errors.New("completed artifact type differs from the profile definition")
	}
	save, _ := cmd.Flags().GetString("save")
	if save == "" {
		save, _ = os.Getwd()
	}
	config := &clientpb.ImplantConfig{}
	artifact := record.FinalArtifact.Type
	config.GOOS, config.GOARCH = artifact.GOOS, artifact.GOARCH
	switch artifact.Kind {
	case "executable":
		config.Format = clientpb.OutputFormat_EXECUTABLE
	case "service-executable":
		config.Format = clientpb.OutputFormat_SERVICE
	}
	path, err := outputSavePath(save, file.Name, config, con)
	if err != nil {
		return true, err
	}
	if err := os.WriteFile(path, file.Data, 0700); err != nil {
		return true, err
	}
	con.PrintInfof("Profile %s build %s saved to %s\n", profileName, record.ID, path)
	return true, nil
}

func profileExecuteRequestFromFlags(cmd *cobra.Command, profileName string) (*clientpb.BuildProfileExecuteRequest, error) {
	if cmd == nil || strings.TrimSpace(profileName) == "" {
		return nil, errors.New("profile name required")
	}
	request := &clientpb.BuildProfileExecuteRequest{Name: profileName}
	name, _ := cmd.Flags().GetString("name")
	if name != "" {
		name = strings.ToLower(name)
		if err := util.AllowedName(name); err != nil {
			return nil, err
		}
		request.BuildName = name
	}
	pin, _ := cmd.Flags().GetString("builder")
	if cmd.Flags().Changed("builder") && strings.TrimSpace(pin) == "" {
		return nil, errors.New("builder pin cannot be empty")
	}
	if pin != "" {
		request.Provider = &clientpb.BuildProviderPolicy{Mode: "pinned", Builder: pin}
	} else if remote, _ := cmd.Flags().GetBool("external-builder"); remote {
		request.Provider = &clientpb.BuildProviderPolicy{Mode: "remote"}
	}
	if cmd.Flags().Changed("shellcode-encoder") {
		raw, _ := cmd.Flags().GetString("shellcode-encoder")
		encoder, err := canonicalEncoderOverride(raw)
		if err != nil {
			return nil, err
		}
		request.ShellcodeEncoder = encoder
	}
	artifacts, _ := cmd.Flags().GetStringArray("input-artifact")
	secrets, _ := cmd.Flags().GetStringArray("secret-file")
	for _, source := range []struct {
		kind   string
		values []string
	}{{"input-artifact", artifacts}, {"secret", secrets}} {
		for _, value := range source.values {
			input, err := profileInputFromFlag(source.kind, value)
			if err != nil {
				return nil, err
			}
			request.Inputs = append(request.Inputs, input)
		}
	}
	return request, nil
}

func profileInputFromFlag(kind, assignment string) (*clientpb.BuildInput, error) {
	name, path, ok := strings.Cut(assignment, "=")
	if !ok || path == "" || !validProfileInputName(name) {
		return nil, fmt.Errorf("invalid %s input %q; use name=path", kind, assignment)
	}
	limit := int64(64 << 20)
	if kind == "secret" {
		limit = 1 << 20
	}
	file, err := os.Open(expandPath(path))
	if err != nil {
		return nil, fmt.Errorf("read %s input %q: %w", kind, name, err)
	}
	defer file.Close()
	if info, err := file.Stat(); err != nil {
		return nil, fmt.Errorf("stat %s input %q: %w", kind, name, err)
	} else if info.Mode().IsRegular() && info.Size() > limit {
		return nil, fmt.Errorf("%s input %q exceeds %d bytes", kind, name, limit)
	}
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, fmt.Errorf("read %s input %q: %w", kind, name, err)
	}
	if len(data) == 0 || int64(len(data)) > limit {
		return nil, fmt.Errorf("%s input %q is empty or exceeds %d bytes", kind, name, limit)
	}
	return &clientpb.BuildInput{Kind: kind, Name: name, Data: data}, nil
}

func validProfileInputName(name string) bool {
	if len(name) == 0 || name[0] < 'a' || name[0] > 'z' {
		return false
	}
	for _, ch := range name[1:] {
		if !((ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-') {
			return false
		}
	}
	return true
}

func canonicalEncoderOverride(raw string) (string, error) {
	switch normalizeShellcodeEncoderName(raw) {
	case "none", "xor":
		return normalizeShellcodeEncoderName(raw), nil
	case "xor_dynamic":
		return "xor-dynamic", nil
	case "shikata_ga_nai":
		return "shikata-ga-nai", nil
	default:
		return "", fmt.Errorf("unsupported shellcode encoder override %q", raw)
	}
}
