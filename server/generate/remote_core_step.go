package generate

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/buildcore"
	"github.com/bishopfox/sliver/server/buildcore/artifactaccept"
	"github.com/bishopfox/sliver/server/buildcore/workspace"
	"github.com/bishopfox/sliver/server/buildplan"
)

// CompilePreparedSliverCore executes only the assigned core step. The server
// prepares the build keys and C2 configuration; the worker never generates a
// second identity or writes a build record. Its private source and material
// workspace is removed before the accepted artifact is returned.
//
//nolint:gocyclo // Validation, output acceptance, and cleanup are tied to the same worker assignment.
func CompilePreparedSliverCore(ctx context.Context, plan buildplan.Plan, prepared *clientpb.ExternalImplantConfig) (result buildcore.Result, buildErr error) {
	if ctx == nil || prepared == nil || prepared.GetConfig() == nil || prepared.GetBuild() == nil || prepared.GetHTTPC2() == nil ||
		prepared.GetHTTPC2().GetImplantConfig() == nil {
		return buildcore.Result{}, errors.New("missing prepared Sliver core material")
	}
	if err := ctx.Err(); err != nil {
		return buildcore.Result{}, err
	}
	if len(plan.Steps) == 0 || plan.Steps[0].Kind != "core" || plan.Steps[0].Spec.Name != "builtin/sliver" {
		return buildcore.Result{}, errors.New("assigned step is not a Sliver core")
	}
	step := plan.Steps[0]
	config, build := prepared.Config, prepared.Build
	if config.GOOS != step.Output.Target.OS || config.GOARCH != step.Output.Target.Arch ||
		step.Output.Target != plan.Request.Target || build.Name == "" || plan.Request.Name != build.Name {
		return buildcore.Result{}, errors.New("prepared Sliver target or name differs from assigned core")
	}
	adapter := SliverCoreAdapter{}
	if err := adapter.Normalize(config); err != nil {
		return buildcore.Result{}, err
	}
	var output string
	var err error
	if config.Format == clientpb.OutputFormat_SHELLCODE {
		if err := validateShellcodeCoreArtifact(step.Output); err != nil {
			return buildcore.Result{}, err
		}
		if len(step.Output.EntrySymbols) != 0 && !slices.Equal(step.Output.EntrySymbols, config.Exports) {
			return buildcore.Result{}, errors.New("prepared Sliver exports differ from assigned core")
		}
		output, err = compileShellcodeCore(ctx, build.Name, build, config, prepared.HTTPC2.ImplantConfig)
	} else {
		expected, expectedErr := sliverOutputArtifact(config)
		if expectedErr != nil {
			return buildcore.Result{}, expectedErr
		}
		if !compatibleDirectSliverArtifact(step.Output, expected) {
			return buildcore.Result{}, errors.New("prepared Sliver format differs from assigned core")
		}
		output, err = adapter.Compile(ctx, build.Name, build, config, prepared.HTTPC2.ImplantConfig)
	}
	if err != nil {
		return buildcore.Result{}, fmt.Errorf("compile assigned Sliver core: %w", err)
	}
	defer func() {
		if cleanupErr := workspace.RemoveOutputWorkspace(assets.GetRootAppDir(), config.GOOS, config.GOARCH, build.Name, output); cleanupErr != nil {
			buildErr = errors.Join(buildErr, fmt.Errorf("remove private remote core workspace: %w", cleanupErr))
			result = buildcore.Result{}
		}
	}()
	if err := writeSliverResult(output); err != nil {
		return buildcore.Result{}, err
	}
	data, info, err := artifactaccept.Accept(filepath.Dir(output), step.Output)
	if err != nil {
		return buildcore.Result{}, fmt.Errorf("accept assigned Sliver core: %w", err)
	}
	return buildcore.Result{
		BuildID: build.ID, ImplantName: build.Name, FileName: info.Name,
		Data: data, ToolchainIdentity: sliverToolchainIdentity(),
	}, nil
}

// SliverStageType derives the legacy ResourceID label from a frozen stage
// route, without exposing any of its per-build key material.
func SliverStageType(plan buildplan.Plan) (string, error) { return sliverStageType(plan) }

// SliverToolchainIdentity is the identity the worker reports for Sliver's
// installed compiler and Mage integration.
func SliverToolchainIdentity() string { return sliverToolchainIdentity() }
