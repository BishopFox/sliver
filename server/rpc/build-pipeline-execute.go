package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/buildcore"
	"github.com/bishopfox/sliver/server/buildcore/artifactaccept"
	"github.com/bishopfox/sliver/server/buildcore/workspace"
	"github.com/bishopfox/sliver/server/buildplan"
	"github.com/bishopfox/sliver/server/buildtransform"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/generate"
)

// pipelineRun retains intermediate bytes only while this call is active. The
// record holds their verified metadata; only the final byte stream is saved.
type pipelineRun struct {
	result     buildcore.Result
	steps      []buildplan.ArtifactInfo
	selections []*buildplan.StepSelection
	identities []string
	failed     int
	savedFinal bool
	err        error
}

// executeResolvedPlan is shared by neutral and legacy generation. It records
// the observed bytes at every completed step and freezes final bytes before a
// successful caller can return. A failed run still returns its failed record.

//nolint:gocyclo // The build record and artifact share one execution and persistence lifecycle.
func (rpc *Server) executeResolvedPlan(ctx context.Context, plan buildplan.Plan, adapter buildcore.Adapter,
	config *clientpb.ImplantConfig, resolve buildtransform.ResolveInput) (output buildcore.Result, record buildplan.Record, finalErr error) {
	var run pipelineRun
	if hasRemoteStep(plan) {
		run = rpc.runMixedPlan(ctx, &plan, adapter, config, resolve)
	} else {
		run = runResolvedPlan(ctx, plan, adapter, config, resolve)
	}
	if run.savedFinal {
		defer func() {
			if finalErr == nil {
				return
			}
			if run.result.WorkspacePath != "" {
				placeholder := filepath.Join(run.result.WorkspacePath, "bin", "retained")
				if err := workspace.RemoveOutputWorkspace(assets.GetRootAppDir(), plan.Request.Target.OS,
					plan.Request.Target.Arch, run.result.ImplantName, placeholder); err != nil {
					finalErr = errors.Join(finalErr, fmt.Errorf("remove retained workspace after record failure: %w", err))
				}
			}
			if err := rollbackSavedRuntimeBuild(run.result.BuildID, run.result.ImplantName); err != nil {
				finalErr = errors.Join(finalErr, fmt.Errorf("roll back saved runtime build: %w", err))
			}
		}()
	}
	if run.err != nil {
		var planErr *buildplan.PlanError
		if errors.As(run.err, &planErr) && len(run.steps) == 0 {
			return buildcore.Result{}, buildplan.Record{}, run.err
		}
	}
	store, err := rpc.buildRecordStore()
	if err != nil {
		return buildcore.Result{}, buildplan.Record{}, fmt.Errorf("open build record store: %w", err)
	}
	if run.result.BuildID != "" {
		record, err = store.CreateWithID(run.result.BuildID, plan)
	} else {
		record, err = store.Create(plan)
		if err == nil {
			run.result.BuildID = record.ID
		}
	}
	if err != nil {
		return buildcore.Result{}, buildplan.Record{}, fmt.Errorf("create build record: %w", err)
	}
	for index, info := range run.steps {
		var selection *buildplan.StepSelection
		if index < len(run.selections) {
			selection = run.selections[index]
		}
		if _, err := store.UpdateStep(record.ID, index, buildplan.StepUpdate{State: buildplan.StepRunning, Selection: selection}); err != nil {
			return buildcore.Result{}, buildplan.Record{}, err
		}
		identity := plan.Steps[index].Spec.Name + "@" + plan.Steps[index].Spec.Version
		if index == 0 {
			identity = run.result.ToolchainIdentity
		}
		if index < len(run.identities) && run.identities[index] != "" {
			identity = run.identities[index]
		}
		record, err = store.UpdateStep(record.ID, index, buildplan.StepUpdate{
			State: buildplan.StepComplete, Progress: 100, Artifact: &info, ToolchainIdentity: identity, Selection: selection,
		})
		if err != nil {
			return buildcore.Result{}, buildplan.Record{}, err
		}
	}
	if run.err != nil {
		failed := run.failed
		if failed < 0 || failed >= len(plan.Steps) {
			failed = len(run.steps)
		}
		if failed >= len(plan.Steps) {
			failed = len(plan.Steps) - 1
		}
		var selection *buildplan.StepSelection
		if failed < len(run.selections) {
			selection = run.selections[failed]
		}
		record, err = store.UpdateStep(record.ID, failed, buildplan.StepUpdate{
			State: buildplan.StepFailed, Failure: run.err.Error(), Selection: selection,
		})
		if err != nil {
			return buildcore.Result{}, buildplan.Record{}, errors.Join(run.err, err)
		}
		return buildcore.Result{}, record, run.err
	}
	if len(run.steps) != len(plan.Steps) {
		return buildcore.Result{}, record, errors.New("completed pipeline has missing step artifacts")
	}
	record, err = store.Complete(record.ID, run.steps[len(run.steps)-1], run.result.Data)
	if err != nil {
		return buildcore.Result{}, record, fmt.Errorf("store final build bytes: %w", err)
	}
	return run.result, record, nil
}

// A runtime build is usable for authentication as soon as SaveFinal commits
// it. If the typed record cannot be persisted afterward, remove precisely the
// build saved by this run so failed builds cannot remain connectable.
func rollbackSavedRuntimeBuild(id, name string) error {
	if id == "" || name == "" {
		return errors.New("missing saved runtime build identity")
	}
	build, err := db.ImplantBuildByID(id)
	if err != nil {
		return err
	}
	if build == nil || build.ID != id || build.Name != name {
		return fmt.Errorf("saved runtime build %s changed identity", id)
	}
	return discardRuntimeBuildExact(build)
}

//nolint:gocyclo // The ordered core and transform steps share artifact and cleanup state.
func runResolvedPlan(ctx context.Context, plan buildplan.Plan, adapter buildcore.Adapter,
	config *clientpb.ImplantConfig, resolve buildtransform.ResolveInput) (run pipelineRun) {
	run.failed = 0
	if ctx == nil {
		run.err = errors.New("missing build context")
		return
	}
	if err := ctx.Err(); err != nil {
		run.err = err
		return
	}
	if err := preflightLocalSteps(plan, resolve); err != nil {
		run.err = err
		return
	}
	if adapter == nil {
		run.err = errors.New("missing core adapter")
		return
	}
	var product buildcore.CoreProduct
	var err error
	if config != nil {
		sliver, ok := adapter.(generate.SliverCoreAdapter)
		if !ok {
			run.err = errors.New("legacy configuration requires the Sliver core adapter")
			return
		}
		product, err = sliver.BuildCoreWithConfig(ctx, plan, config)
	} else if stepAdapter, ok := adapter.(buildcore.CoreStepAdapter); ok {
		product, err = stepAdapter.BuildCore(ctx, plan)
	} else {
		corePlan := plan
		corePlan.Steps = slices.Clone(plan.Steps[:1])
		corePlan.FinalArtifact = plan.Steps[0].Output
		if format := directFormatForCore(plan.Steps[0].Output); format != "" {
			corePlan.Request.FinalFormat = format
		}
		var result buildcore.Result
		result, err = adapter.Build(ctx, corePlan)
		product.Result = result
	}
	if err != nil {
		run.err = fmt.Errorf("core build: %w", err)
		return
	}
	run.result = product.Result
	defer func() {
		if product.Close == nil {
			return
		}
		if closeErr := product.Close(); closeErr != nil {
			run.err = errors.Join(run.err, fmt.Errorf("remove private core workspace: %w", closeErr))
			if len(run.steps) == len(plan.Steps) {
				run.steps = run.steps[:len(run.steps)-1]
			}
			run.failed = len(run.steps)
		}
	}()
	if run.result.ImplantName == "" {
		run.result.ImplantName = plan.Request.Name
	}
	name := run.result.FileName
	data := bytes.Clone(run.result.Data)
	info, err := artifactaccept.InspectBytes(data, plan.Steps[0].Output, name)
	if err != nil {
		run.err = fmt.Errorf("core artifact: %w", err)
		return
	}
	run.steps = append(run.steps, info)
	workDir := ""
	for index := 1; index < len(plan.Steps); index++ {
		run.failed = index
		step := plan.Steps[index]
		if err := ctx.Err(); err != nil {
			run.err = err
			return
		}
		if !samePipelineArtifact(*step.Input, plan.Steps[index-1].Output) {
			run.err = fmt.Errorf("step %s input differs from prior output", step.ID)
			return
		}
		if step.Spec.Name == "builtin/malasada" && workDir == "" {
			appDir, pathErr := filepath.Abs(assets.GetRootAppDir())
			if pathErr != nil {
				run.err = fmt.Errorf("resolve private transform root: %w", pathErr)
				return
			}
			workDir, err = os.MkdirTemp(appDir, "build-transform-*")
			if err != nil {
				run.err = fmt.Errorf("create private transform workspace: %w", err)
				return
			}
			defer func() {
				if removeErr := os.RemoveAll(workDir); removeErr != nil {
					run.err = errors.Join(run.err, fmt.Errorf("remove private transform workspace: %w", removeErr))
					if len(run.steps) == len(plan.Steps) {
						run.steps = run.steps[:len(run.steps)-1]
					}
					run.failed = len(run.steps)
				}
			}()
		}
		data, err = applyLocalStep(ctx, step, data, workDir, resolve)
		if err != nil {
			run.err = fmt.Errorf("step %s: %w", step.ID, err)
			return
		}
		switch step.Output.Kind {
		case "shellcode", "stage":
			name = run.result.ImplantName + ".bin"
		}
		info, err = artifactaccept.InspectBytes(data, step.Output, name)
		if err != nil {
			run.err = fmt.Errorf("step %s artifact: %w", step.ID, err)
			return
		}
		run.steps = append(run.steps, info)
	}
	run.failed = len(plan.Steps) - 1
	if product.SaveFinal != nil {
		buildID, saveErr := product.SaveFinal(ctx, plan.FinalArtifact, name, data)
		if saveErr != nil {
			run.err = fmt.Errorf("save final build artifact: %w", saveErr)
			run.steps = run.steps[:len(run.steps)-1]
			run.failed = len(run.steps)
			return
		}
		run.result.BuildID = buildID
		run.savedFinal = true
	}
	run.result.FileName = name
	run.result.Data = data
	return
}

func directFormatForCore(artifact buildplan.Artifact) string {
	switch artifact.Kind {
	case "executable":
		return "exe"
	case "service-executable":
		return "service"
	case "shared-library", "go-plugin", "go-shared-library":
		return "shared"
	case "c-archive-bundle", "c-archive", "go-archive":
		return "archive"
	case "shellcode":
		return "shellcode"
	case "stage":
		return "stage"
	default:
		return ""
	}
}

//nolint:gocyclo // All installed-step and input checks complete before work starts.
func preflightLocalSteps(plan buildplan.Plan, resolve buildtransform.ResolveInput) error {
	unsupported := func(step, detail string) error {
		return &buildplan.PlanError{Code: buildplan.CodeUnsupportedExecution, Step: step, Detail: detail}
	}
	if len(plan.Steps) == 0 || plan.Steps[0].Kind != "core" || plan.Steps[0].Provider == "" {
		return unsupported("core", "core step has no local execution adapter")
	}
	stage := 0
	converter := false
	encoded := false
	metadata := false
	for index := 1; index < len(plan.Steps); index++ {
		step := plan.Steps[index]
		localStep := step
		localStep.Provider = "server"
		if err := buildtransform.ValidateInstalledStep(localStep); err != nil {
			return unsupported(step.ID, err.Error())
		}
		if !samePipelineArtifact(*step.Input, plan.Steps[index-1].Output) {
			return unsupported(step.ID, "transform input differs from prior output")
		}
		switch step.Spec.Name {
		case "builtin/donut", "builtin/beignet", "builtin/malasada":
			if index != 1 || converter || stage != 0 || metadata {
				return unsupported(step.ID, "converter must immediately follow the core")
			}
			converter = true
		case "builtin/shikata-ga-nai", "builtin/xor", "builtin/xor-dynamic":
			if !converter || stage != 0 || metadata {
				return unsupported(step.ID, "shellcode encoder must follow a converter before stage packaging")
			}
			encoded = true
		case buildtransform.PEMetadataSpec:
			if metadata || converter || encoded || stage != 0 || resolve == nil {
				return unsupported(step.ID, "PE metadata requires one pre-stage donor input")
			}
			metadata = true
		case buildtransform.StageFrameSpec:
			if stage != 0 {
				return unsupported(step.ID, "stage framing must be the first stage step")
			}
			stage = 1
		case buildtransform.StageCompressSpec:
			if stage != 1 {
				return unsupported(step.ID, "stage compression must follow framing")
			}
			stage = 2
		case buildtransform.StageAESSpec, buildtransform.StageRC4Spec:
			if stage != 1 && stage != 2 || resolve == nil {
				return unsupported(step.ID, "stage encryption requires framed bytes and per-build keys")
			}
			stage = 3
		default:
			return unsupported(step.ID, "transform has no local execution adapter")
		}
	}
	if !samePipelineArtifact(plan.Steps[len(plan.Steps)-1].Output, plan.FinalArtifact) {
		return unsupported("final", "final artifact differs from last step output")
	}
	return nil
}

func applyLocalStep(ctx context.Context, step buildplan.Step, input []byte, workDir string,
	resolve buildtransform.ResolveInput) ([]byte, error) {
	switch step.Spec.Name {
	case "builtin/donut", "builtin/beignet", "builtin/malasada":
		return buildtransform.Convert(ctx, step, input, workDir)
	case "builtin/shikata-ga-nai", "builtin/xor", "builtin/xor-dynamic":
		return buildtransform.EncodeShellcode(ctx, step, input)
	case buildtransform.StageFrameSpec, buildtransform.StageCompressSpec,
		buildtransform.StageAESSpec, buildtransform.StageRC4Spec:
		return buildtransform.ApplyStageStep(ctx, step, input, resolve)
	case buildtransform.PEMetadataSpec:
		var donorRef struct {
			Kind string `json:"kind"`
			Name string `json:"name"`
		}
		for _, option := range step.Options {
			if option.Name == "donorArtifact" {
				if err := json.Unmarshal(option.ValueJSON, &donorRef); err != nil {
					return nil, errors.New("invalid PE donor reference")
				}
				break
			}
		}
		if resolve == nil || donorRef.Kind != "input-artifact" || donorRef.Name == "" {
			return nil, errors.New("PE donor is unavailable")
		}
		donor, err := resolve(donorRef.Kind, donorRef.Name)
		if err != nil {
			return nil, errors.New("PE donor is unavailable")
		}
		if strings.HasPrefix(donorRef.Name, "metadata-sha256:") {
			configJSON, err := resolve("metadata-config", donorRef.Name)
			if err != nil {
				return nil, errors.New("PE metadata configuration is unavailable")
			}
			return buildtransform.ApplyPEMetadataConfig(ctx, step, input, donor, configJSON)
		}
		return buildtransform.ApplyPEMetadata(ctx, step, input, donor)
	default:
		return nil, errors.New("transform has no local execution adapter")
	}
}

func samePipelineArtifact(left, right buildplan.Artifact) bool {
	return left.Kind == right.Kind && left.Container == right.Container && left.Target == right.Target &&
		left.GoBuildMode == right.GoBuildMode && slices.Equal(left.EntrySymbols, right.EntrySymbols)
}
