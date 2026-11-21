// Package buildcore defines adapters and a registry for core builds.
package buildcore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/bishopfox/sliver/server/buildcore/artifactaccept"
	"github.com/bishopfox/sliver/server/buildcore/magebuild"
	"github.com/bishopfox/sliver/server/buildcore/workspace"
	"github.com/bishopfox/sliver/server/buildplan"
)

const (
	maxCoreMaterialBytes = 16 << 20
	maxCoreManifestBytes = 1 << 20
)

var mageTargetPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]*$`)

// MageBinding is the installed counterpart of CoreSpec's source and builder
// declarations. Magefile names a directory inside the installed Mage bundle;
// it is never an arbitrary path from an operator request.
type MageBinding struct {
	SourceMode string
	RendererID string
	Magefile   string
	Target     string
}

// MageBound is implemented by adapters whose CoreSpec declares a Mage builder.
// The catalog-facing executor verifies these fields before accepting a build.
type MageBound interface {
	Binding() MageBinding
}

// MageCoreAdapter is a reusable installed core integration. Prepare runs on the
// server and returns opaque, core-specific material. Render, when declared,
// changes only the private staged source. Manifest creates the core's typed
// JSON input after source and material are ready. The Mage target writes one
// primary artifact and result.json under workspace/bin.
type MageCoreAdapter struct {
	ID           string
	AppDir       string
	BindingSpec  MageBinding
	Source       fs.FS
	Magefiles    fs.FS
	BundleDigest string
	GoCmd        string
	Env          []string
	ToolchainID  string
	Available    func(buildplan.Target, buildplan.Toolchain) bool
	Prepare      func(context.Context, buildplan.Plan) ([]byte, error)
	Render       func(context.Context, *workspace.Workspace, string) error
	Manifest     func(context.Context, buildplan.Plan, *workspace.Workspace, string) ([]byte, error)
	Stdout       io.Writer
	Stderr       io.Writer
}

// ExecutorID returns the adapter ID recorded in the installed catalog.
func (adapter MageCoreAdapter) ExecutorID() string { return adapter.ID }

// Binding returns the source and Mage target binding.
func (adapter MageCoreAdapter) Binding() MageBinding { return adapter.BindingSpec }

// Supports reports the installed toolchains and targets this Mage-backed core
// can execute. An unconfigured adapter never advertises a build capability.
func (adapter MageCoreAdapter) Supports(target buildplan.Target, toolchain buildplan.Toolchain) bool {
	return adapter.Available != nil && adapter.Available(target, toolchain)
}

// Build runs one direct core step. M4's step executor will consume the same
// accepted core bytes before applying any requested transforms.
//
//nolint:gocyclo // The workspace lifecycle spans preparation, acceptance, and cleanup with one owner.
func (adapter MageCoreAdapter) Build(ctx context.Context, plan buildplan.Plan) (result Result, buildErr error) {
	if ctx == nil {
		return Result{}, errors.New("missing core build context")
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if err := adapter.validate(); err != nil {
		return Result{}, err
	}
	if len(plan.Steps) != 1 || plan.Steps[0].Kind != "core" || plan.Steps[0].Provider != "server" ||
		!sameArtifact(plan.Steps[0].Output, plan.FinalArtifact) {
		return Result{}, errors.New("the Mage core adapter requires one direct local core step")
	}
	if plan.Request.Target != plan.Steps[0].Output.Target {
		return Result{}, errors.New("core target does not match frozen plan")
	}
	if !adapter.Supports(plan.Request.Target, plan.Steps[0].Toolchain) {
		return Result{}, errors.New("installed Mage core cannot execute planned target or toolchain")
	}
	name := plan.Request.Name
	if name == "" {
		name = "core"
	}
	work, err := workspace.New(adapter.AppDir, plan.Request.Target.OS, plan.Request.Target.Arch, name)
	if err != nil {
		return Result{}, err
	}
	defer func() {
		if err := work.Remove(); err != nil {
			buildErr = errors.Join(buildErr, fmt.Errorf("remove private core workspace: %w", err))
			result = Result{}
		}
	}()
	if adapter.BindingSpec.SourceMode == "copy" {
		if err := work.CopySource(adapter.Source); err != nil {
			return Result{}, fmt.Errorf("copy core source: %w", err)
		}
	}
	if err := work.CopyMagefiles(adapter.Magefiles); err != nil {
		return Result{}, fmt.Errorf("copy core Magefile: %w", err)
	}
	materialPath := ""
	if adapter.Prepare != nil {
		material, err := adapter.Prepare(ctx, plan)
		if err != nil {
			clear(material)
			return Result{}, fmt.Errorf("prepare core material: %w", err)
		}
		defer clear(material)
		if len(material) > maxCoreMaterialBytes {
			return Result{}, errors.New("core material exceeds size limit")
		}
		if material != nil {
			materialPath, err = work.WriteInput("material.bin", material)
			if err != nil {
				return Result{}, fmt.Errorf("write private core material: %w", err)
			}
		}
	}
	if adapter.Render != nil {
		if err := adapter.Render(ctx, work, materialPath); err != nil {
			return Result{}, fmt.Errorf("render core source: %w", err)
		}
	}
	manifest, err := adapter.Manifest(ctx, plan, work, materialPath)
	if err != nil {
		return Result{}, fmt.Errorf("prepare Mage manifest: %w", err)
	}
	if len(manifest) == 0 || len(manifest) > maxCoreManifestBytes || !json.Valid(manifest) {
		return Result{}, errors.New("the Mage manifest must be bounded JSON")
	}
	manifestPath, err := work.WriteInput("build.json", manifest)
	if err != nil {
		return Result{}, fmt.Errorf("write Mage manifest: %w", err)
	}
	bundleDir := filepath.Join(work.MagefilesDir, filepath.FromSlash(adapter.BindingSpec.Magefile))
	runner := magebuild.Runner{RootDir: adapter.AppDir, GoCmd: adapter.GoCmd}
	if err := runner.Run(ctx, magebuild.Invocation{
		BundleDir: bundleDir, BundleDigest: adapter.BundleDigest, WorkDir: work.Root,
		ManifestPath: manifestPath, Target: adapter.BindingSpec.Target,
		Env: slices.Clone(adapter.Env), Stdout: adapter.Stdout, Stderr: adapter.Stderr,
	}); err != nil {
		return Result{}, fmt.Errorf("build core with Mage: %w", err)
	}
	data, info, err := artifactaccept.Accept(work.BinDir, plan.Steps[0].Output)
	if err != nil {
		return Result{}, fmt.Errorf("accept core artifact: %w", err)
	}
	identity := adapter.ToolchainID
	if identity == "" {
		identity = "mage:" + strings.TrimPrefix(adapter.BundleDigest, "sha256:")[:12]
	}
	return Result{
		BuildID: work.ID, ImplantName: name, FileName: info.Name,
		Data: data, ToolchainIdentity: identity,
	}, nil
}

//nolint:gocyclo // The installed adapter contract is checked at one boundary before any workspace is created.
func (adapter MageCoreAdapter) validate() error {
	binding := adapter.BindingSpec
	if adapter.ID == "" || adapter.AppDir == "" || adapter.Magefiles == nil || adapter.BundleDigest == "" ||
		adapter.Manifest == nil || !safeMagefileDir(binding.Magefile) || !mageTargetPattern.MatchString(binding.Target) {
		return errors.New("incomplete installed Mage core adapter")
	}
	if binding.SourceMode != "copy" && binding.SourceMode != "none" {
		return fmt.Errorf("invalid source mode %q", binding.SourceMode)
	}
	if binding.SourceMode == "copy" && adapter.Source == nil {
		return errors.New("copy mode needs installed source")
	}
	if binding.SourceMode == "none" && (adapter.Source != nil || adapter.Render != nil || binding.RendererID != "") {
		return errors.New("source-free core cannot render or copy source")
	}
	if (binding.RendererID == "") != (adapter.Render == nil) {
		return errors.New("renderer declaration and installed hook differ")
	}
	return nil
}

func safeMagefileDir(value string) bool {
	if value == "" || path.IsAbs(value) || path.Clean(value) != value || strings.ContainsAny(value, `\:`) {
		return false
	}
	for _, component := range strings.Split(value, "/") {
		if component == "" || component == "." || component == ".." {
			return false
		}
	}
	return true
}

func sameArtifact(left, right buildplan.Artifact) bool {
	return left.Kind == right.Kind && left.Container == right.Container && left.Target == right.Target &&
		left.GoBuildMode == right.GoBuildMode && slices.Equal(left.EntrySymbols, right.EntrySymbols)
}
