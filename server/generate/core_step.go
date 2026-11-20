package generate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/buildcompiler"
	"github.com/bishopfox/sliver/server/buildcore"
	"github.com/bishopfox/sliver/server/buildcore/artifactaccept"
	"github.com/bishopfox/sliver/server/buildcore/workspace"
	"github.com/bishopfox/sliver/server/buildplan"
	"github.com/bishopfox/sliver/server/codenames"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/gogo"
	"github.com/bishopfox/sliver/util"
)

const maxSliverFinalBytes = 512 << 20

// BuildCore compiles the first frozen Sliver step and leaves the accepted
// compiler output available to the ordered server-owned transform pipeline.
func (adapter SliverCoreAdapter) BuildCore(ctx context.Context, plan buildplan.Plan) (buildcore.CoreProduct, error) {
	config, err := adapter.ConfigFromPlan(plan)
	if err != nil {
		return buildcore.CoreProduct{}, err
	}
	return adapter.BuildCoreWithConfig(ctx, plan, config)
}

// BuildCoreWithConfig is the compatibility entry for Generate. The supplied
// config retains its legacy fields while the plan freezes the compiler output
// type and the later transforms. Neither intermediate bytes nor build material
// are saved to the legacy build database.
//
//nolint:gocyclo // Output acceptance and the save and close callbacks share one workspace lifecycle.
func (adapter SliverCoreAdapter) BuildCoreWithConfig(ctx context.Context, plan buildplan.Plan, config *clientpb.ImplantConfig) (product buildcore.CoreProduct, buildErr error) {
	if ctx == nil || config == nil {
		return buildcore.CoreProduct{}, errors.New("missing Sliver core context or config")
	}
	if err := ctx.Err(); err != nil {
		return buildcore.CoreProduct{}, err
	}
	if len(plan.Steps) == 0 || plan.Steps[0].Kind != "core" || plan.Steps[0].Provider != "server" ||
		plan.Steps[0].Spec.Name != "builtin/sliver" {
		return buildcore.CoreProduct{}, errors.New("plan has no local Sliver core step")
	}
	coreArtifact := plan.Steps[0].Output
	if coreArtifact.Target != plan.Request.Target || config.GOOS != coreArtifact.Target.OS || config.GOARCH != coreArtifact.Target.Arch {
		return buildcore.CoreProduct{}, errors.New("Sliver config does not match frozen core target") //nolint:staticcheck // Preserve the existing error text and proper noun.
	}
	name := plan.Request.Name
	if name == "" {
		generated, err := codenames.GetCodename()
		if err != nil {
			return buildcore.CoreProduct{}, err
		}
		name = generated
	} else if err := util.AllowedName(name); err != nil {
		return buildcore.CoreProduct{}, err
	}
	if err := adapter.Normalize(config); err != nil {
		return buildcore.CoreProduct{}, err
	}
	build, err := adapter.Prepare(name, config)
	if err != nil {
		return buildcore.CoreProduct{}, err
	}
	if err := ctx.Err(); err != nil {
		return buildcore.CoreProduct{}, err
	}
	httpConfig, err := db.LoadHTTPC2ConfigByName(config.HTTPC2ConfigName)
	if err != nil {
		return buildcore.CoreProduct{}, err
	}
	if httpConfig == nil || httpConfig.ImplantConfig == nil {
		return buildcore.CoreProduct{}, errors.New("Sliver HTTP C2 configuration is unavailable") //nolint:staticcheck // Preserve the existing error text and proper noun.
	}

	var output string
	if config.Format == clientpb.OutputFormat_SHELLCODE {
		if err := validateShellcodeCoreArtifact(coreArtifact); err != nil {
			return buildcore.CoreProduct{}, err
		}
		if len(coreArtifact.EntrySymbols) != 0 && !slices.Equal(coreArtifact.EntrySymbols, config.Exports) {
			return buildcore.CoreProduct{}, errors.New("Sliver shellcode exports differ from frozen core artifact") //nolint:staticcheck // Preserve the existing error text and proper noun.
		}
		output, err = compileShellcodeCore(ctx, name, build, config, httpConfig.ImplantConfig)
	} else {
		expected, artifactErr := sliverOutputArtifact(config)
		if artifactErr != nil {
			return buildcore.CoreProduct{}, artifactErr
		}
		if !compatibleDirectSliverArtifact(coreArtifact, expected) {
			return buildcore.CoreProduct{}, errors.New("Sliver direct compiler output differs from frozen core artifact") //nolint:staticcheck // Preserve the existing error text and proper noun.
		}
		output, err = adapter.Compile(ctx, name, build, config, httpConfig.ImplantConfig)
	}
	if err != nil {
		return buildcore.CoreProduct{}, err
	}
	defer func() {
		if buildErr == nil {
			return
		}
		if err := workspace.RemoveOutputWorkspace(assets.GetRootAppDir(), config.GOOS, config.GOARCH, name, output); err != nil {
			buildErr = errors.Join(buildErr, fmt.Errorf("remove failed Sliver core workspace: %w", err))
		}
	}()
	if err := writeSliverResult(output); err != nil {
		return buildcore.CoreProduct{}, err
	}
	coreBytes, artifact, err := artifactaccept.Accept(filepath.Dir(output), coreArtifact)
	if err != nil {
		return buildcore.CoreProduct{}, fmt.Errorf("accept Sliver core output: %w", err)
	}
	workspacePath := filepath.Dir(filepath.Dir(output))
	if adapter.KeepWorkspace {
		if err := writeRetainedBuildScript(workspacePath, runtime.GOOS); err != nil {
			return buildcore.CoreProduct{}, fmt.Errorf("prepare retained Sliver workspace: %w", err)
		}
	}

	var stateMu sync.Mutex
	closed, saveAttempted, saved := false, false, false
	var closeErr error
	closeProduct := func() error {
		stateMu.Lock()
		defer stateMu.Unlock()
		if closed {
			return closeErr
		}
		closed = true
		if adapter.KeepWorkspace && saved {
			return nil
		}
		closeErr = workspace.RemoveOutputWorkspace(assets.GetRootAppDir(), config.GOOS, config.GOARCH, name, output)
		return closeErr
	}
	saveFinal := func(saveCtx context.Context, finalArtifact buildplan.Artifact, finalName string, finalBytes []byte) (string, error) {
		stateMu.Lock()
		defer stateMu.Unlock()
		if closed || saveAttempted {
			return "", errors.New("Sliver core product is closed or already saved") //nolint:staticcheck // Preserve the existing error text and proper noun.
		}
		saveAttempted = true // A partially committed legacy save must not be retried.
		if saveCtx == nil {
			return "", errors.New("missing final save context")
		}
		if err := saveCtx.Err(); err != nil {
			return "", err
		}
		if !sameSliverArtifact(finalArtifact, plan.FinalArtifact) {
			return "", errors.New("final artifact differs from frozen plan")
		}
		if !safeSliverFinalName(finalName) || len(finalBytes) == 0 || len(finalBytes) > maxSliverFinalBytes {
			return "", errors.New("invalid Sliver final name or byte size")
		}
		finalPath := filepath.Join(filepath.Dir(output), finalName)
		if err := writeSliverFinalFile(finalPath, finalBytes); err != nil {
			return "", err
		}
		if err := writeSliverResult(finalPath); err != nil {
			return "", err
		}
		accepted, _, err := artifactaccept.Accept(filepath.Dir(output), finalArtifact)
		if err != nil {
			return "", fmt.Errorf("accept final Sliver output: %w", err)
		}
		if !bytes.Equal(accepted, finalBytes) {
			return "", errors.New("final Sliver output changed during acceptance")
		}
		if finalArtifact.Kind == "stage" {
			stageType, typeErr := sliverStageType(plan)
			if typeErr != nil {
				return "", typeErr
			}
			build.Stage = true
			err = SaveStage(build, config, accepted, stageType)
		} else {
			err = ImplantBuildSave(build, config, finalPath)
		}
		if err != nil {
			return "", err
		}
		if build.ID == "" {
			return "", errors.New("legacy Sliver storage did not assign a build ID")
		}
		saved = true
		return build.ID, nil
	}
	result := buildcore.Result{ImplantName: name, FileName: artifact.Name, Data: coreBytes, ToolchainIdentity: sliverToolchainIdentity()}
	if adapter.KeepWorkspace {
		result.WorkspacePath = workspacePath
	}
	return buildcore.CoreProduct{
		Result:    result,
		SaveFinal: saveFinal, Close: closeProduct,
	}, nil
}

// sliverStageType preserves the legacy ResourceID label for GenerateStage.
// It describes the frozen transforms without reading or recording key bytes.
func sliverStageType(plan buildplan.Plan) (string, error) {
	if plan.FinalArtifact.Kind != "stage" {
		return "", errors.New("plan does not produce a stage")
	}
	var label strings.Builder
	for _, step := range plan.Steps[1:] {
		switch step.Spec.Name {
		case "builtin/stage-compress":
			var algorithm string
			for _, option := range step.Options {
				if option.Name == "algorithm" {
					if err := json.Unmarshal(option.ValueJSON, &algorithm); err != nil {
						return "", fmt.Errorf("invalid frozen stage compression algorithm: %w", err)
					}
					break
				}
			}
			if algorithm == "" {
				algorithm = "zlib"
			}
			label.WriteString(algorithm)
			label.WriteString(" - ")
		case "builtin/stage-aes":
			label.WriteString("AES - ")
		case "builtin/stage-rc4":
			label.WriteString("RC4 - ")
		}
	}
	return label.String(), nil
}

//nolint:gocyclo // Keep each platform's exact artifact requirements together.
func validateShellcodeCoreArtifact(artifact buildplan.Artifact) error {
	target := artifact.Target
	switch target.OS {
	case WINDOWS:
		if target.Arch == "386" || target.Arch == "amd64" {
			if artifact.Kind == "executable" && artifact.Container == "pe" && artifact.GoBuildMode == "pie" && len(artifact.EntrySymbols) == 0 {
				return nil
			}
		}
	case LINUX:
		if target.Arch == "amd64" || target.Arch == "arm64" {
			if artifact.Kind == "shared-library" && artifact.Container == "elf" && artifact.GoBuildMode == "c-shared" && len(artifact.EntrySymbols) > 0 {
				return nil
			}
		}
	case DARWIN:
		if target.Arch == "arm64" {
			if artifact.Kind == "shared-library" && artifact.Container == "macho" && artifact.GoBuildMode == "c-shared" && len(artifact.EntrySymbols) > 0 {
				return nil
			}
		}
	}
	return errors.New("unsupported Sliver shellcode core artifact")
}

func sameSliverArtifact(left, right buildplan.Artifact) bool {
	return left.Kind == right.Kind && left.Container == right.Container && left.Target == right.Target &&
		left.GoBuildMode == right.GoBuildMode && slices.Equal(left.EntrySymbols, right.EntrySymbols)
}

func compatibleDirectSliverArtifact(planned, emitted buildplan.Artifact) bool {
	return planned.Kind == emitted.Kind && planned.Container == emitted.Container && planned.Target == emitted.Target &&
		planned.GoBuildMode == emitted.GoBuildMode &&
		(len(planned.EntrySymbols) == 0 || slices.Equal(planned.EntrySymbols, emitted.EntrySymbols))
}

func safeSliverFinalName(name string) bool {
	return name != "" && name != "." && name != ".." && name != "result.json" &&
		strings.TrimSpace(name) == name && filepath.Base(name) == name &&
		!strings.ContainsAny(name, "/\\:\x00")
}

func writeSliverFinalFile(path string, contents []byte) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".final-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(file.Name()) }()
	if err := file.Chmod(0600); err != nil {
		_ = file.Close()
		return err
	}
	if _, err := file.Write(contents); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}

// compileShellcodeCore retains the original Sliver Go/Mage build flags while
// stopping before Donut, Beignet, or Malasada consumes the compiler output.
//
//nolint:gocyclo // Platform validation and compiler settings share the same output contract.
func compileShellcodeCore(ctx context.Context, name string, build *clientpb.ImplantBuild, config *clientpb.ImplantConfig,
	httpConfig *clientpb.HTTPC2ImplantConfig) (output string, buildErr error) {
	appDir := assets.GetRootAppDir()
	goConfig := &gogo.GoConfig{
		GOOS: config.GOOS, GOARCH: config.GOARCH,
		GOCACHE: gogo.GetGoCache(appDir), GOMODCACHE: gogo.GetGoModCache(appDir),
		GOROOT: gogo.GetGoRootDir(appDir), GOPROXY: getGoProxy(),
		HTTPPROXY: getGoHttpProxy(), HTTPSPROXY: getGoHttpsProxy(),
		Obfuscation: config.ObfuscateSymbols, GOGARBLE: goGarble(config),
	}
	buildMode := "pie"
	ext := ".intermediate.exe"
	switch config.GOOS {
	case WINDOWS:
		if config.GOARCH != "386" && config.GOARCH != "amd64" {
			return "", errors.New("Windows shellcode core requires 386 or amd64") //nolint:staticcheck // Preserve the existing error text and platform name.
		}
		goConfig.CGO = "0"
	case LINUX, DARWIN:
		if config.GOOS == LINUX && config.GOARCH != "amd64" && config.GOARCH != "arm64" ||
			config.GOOS == DARWIN && config.GOARCH != "arm64" {
			return "", errors.New("shared-library shellcode core target is unsupported")
		}
		if len(config.Exports) == 0 {
			return "", errors.New("shared-library shellcode core needs an entry symbol")
		}
		goConfig.CGO = "1"
		if runtime.GOOS != config.GOOS || runtime.GOARCH != config.GOARCH {
			goConfig.CC, goConfig.CXX = findCrossCompilers(config.GOOS, config.GOARCH)
		}
		config.IsSharedLib = true
		buildMode = "c-shared"
		if config.GOOS == LINUX {
			ext = ".intermediate.so"
			sanitizeZigForBuild(goConfig, buildMode)
		} else {
			ext = ".intermediate.dylib"
		}
	default:
		return "", errors.New("Sliver shellcode core target OS is unsupported") //nolint:staticcheck // Preserve the existing error text and proper noun.
	}
	pkgPath, err := renderSliverGoCode(name, build, config, goConfig, httpConfig)
	if err != nil {
		return "", err
	}
	defer cleanupFailedSliverWorkspace(goConfig, config, name, &buildErr)
	output = filepath.Join(goConfig.ProjectDir, "bin", filepath.Base(name)+ext)
	tags := []string{}
	if config.NetGoEnabled {
		tags = append(tags, "netgo")
	}
	ldflags := []string{""} // Garble adds its own strip and build-id flags.
	wantStaticSO := false
	if config.GOOS == WINDOWS && !config.Debug {
		ldflags[0] += " -H=windowsgui"
	}
	if config.GOOS == LINUX {
		ldflags = applyLinuxShellcodeBuildID(ldflags)
		ldflags, wantStaticSO = applyZigStaticLinking(goConfig, buildMode, ldflags)
	}
	_, err = buildcompiler.Default().Build(ctx, buildcompiler.GoDriverID, buildcompiler.Request{
		GoConfig: *goConfig, SourceDir: pkgPath, Destination: output, BuildMode: buildMode,
		Tags: tags, LDFlags: ldflags, MaterialPath: sliverMaterialPath(goConfig),
	})
	if err != nil {
		return "", err
	}
	if config.GOOS == LINUX {
		if err := validateELFSharedObject(output, wantStaticSO); err != nil {
			return "", err
		}
	}
	return output, nil
}
