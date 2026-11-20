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
	"github.com/bishopfox/sliver/server/buildplan"
	"github.com/bishopfox/sliver/server/buildtransform"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/encoders"
	"github.com/bishopfox/sliver/server/generate"
)

func hasRemoteStep(plan buildplan.Plan) bool {
	for _, step := range plan.Steps {
		if step.Provider != "server" {
			return true
		}
	}
	return false
}

// runMixedPlan executes the same frozen step sequence as the local pipeline.
// Each remote result is accepted before the following step sees its bytes.
func (rpc *Server) runMixedPlan(ctx context.Context, plan *buildplan.Plan, adapter buildcore.Adapter,
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
	if plan == nil {
		run.err = errors.New("missing build plan")
		return
	}
	if err := preflightLocalSteps(*plan, resolve); err != nil {
		run.err = err
		return
	}
	if config != nil {
		run.err = errors.New("legacy configuration cannot execute a remote step")
		return
	}
	var product buildcore.CoreProduct
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
	var data []byte
	var name string
	if plan.Steps[0].Provider == "server" {
		if adapter == nil {
			run.err = errors.New("missing local core adapter")
			return
		}
		var err error
		if coreAdapter, ok := adapter.(buildcore.CoreStepAdapter); ok {
			product, err = coreAdapter.BuildCore(ctx, *plan)
		} else {
			corePlan := *plan
			corePlan.Steps = slices.Clone(plan.Steps[:1])
			corePlan.FinalArtifact = plan.Steps[0].Output
			if format := directFormatForCore(plan.Steps[0].Output); format != "" {
				corePlan.Request.FinalFormat = format
			}
			product.Result, err = adapter.Build(ctx, corePlan)
		}
		if err != nil {
			run.err = fmt.Errorf("core build: %w", err)
			return
		}
		run.result = product.Result
		name, data = run.result.FileName, bytes.Clone(run.result.Data)
		info, err := artifactaccept.InspectBytes(data, plan.Steps[0].Output, name)
		if err != nil {
			run.err = fmt.Errorf("core artifact: %w", err)
			return
		}
		run.steps = append(run.steps, info)
		run.selections = append(run.selections, nil)
		run.identities = append(run.identities, run.result.ToolchainIdentity)
	} else {
		result, selection, err := rpc.dispatchRemoteStep(ctx, plan, 0, "", nil, nil, nil)
		run.selections = append(run.selections, selection)
		prepared := result.prepared
		if prepared != nil && prepared.Build != nil {
			run.result.BuildID = prepared.Build.ID
			run.result.ImplantName = prepared.Build.Name
		}
		if err != nil {
			run.err = fmt.Errorf("remote core build: %w", err)
			return
		}
		if prepared == nil || prepared.Build == nil {
			run.err = errors.New("remote core returned no prepared build identity")
			return
		}
		data, name = result.data, result.info.Name
		run.result.FileName, run.result.ToolchainIdentity = name, result.identity
		run.steps = append(run.steps, result.info)
		run.identities = append(run.identities, result.identity)
		product.SaveFinal = func(saveCtx context.Context, final buildplan.Artifact, finalName string, finalBytes []byte) (string, error) {
			return saveRemoteSliverFinal(saveCtx, *plan, prepared, final, finalName, finalBytes)
		}
	}
	if run.result.ImplantName == "" {
		run.result.ImplantName = plan.Request.Name
	}
	assignmentBuildID := run.result.BuildID
	if assignmentBuildID == "" {
		var err error
		assignmentBuildID, err = newStepAssignmentID()
		if err != nil {
			run.err = fmt.Errorf("allocate remote assignment build ID: %w", err)
			return
		}
	}
	workDir := ""
	for index := 1; index < len(plan.Steps); index++ {
		run.failed = index
		step := plan.Steps[index]
		if err := ctx.Err(); err != nil {
			run.err = err
			return
		}
		if step.Input == nil || !samePipelineArtifact(*step.Input, plan.Steps[index-1].Output) {
			run.err = fmt.Errorf("step %s input differs from prior output", step.ID)
			return
		}
		var info buildplan.ArtifactInfo
		var identity string
		if step.Provider == "server" {
			if step.Spec.Name == "builtin/malasada" && workDir == "" {
				appDir, err := filepath.Abs(assets.GetRootAppDir())
				if err != nil {
					run.err = err
					return
				}
				workDir, err = os.MkdirTemp(appDir, "build-transform-*")
				if err != nil {
					run.err = err
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
			var err error
			data, err = applyLocalStep(ctx, step, data, workDir, resolve)
			if err != nil {
				run.err = fmt.Errorf("step %s: %w", step.ID, err)
				return
			}
			if step.Output.Kind == "shellcode" || step.Output.Kind == "stage" {
				name = run.result.ImplantName + ".bin"
			}
			info, err = artifactaccept.InspectBytes(data, step.Output, name)
			if err != nil {
				run.err = fmt.Errorf("step %s artifact: %w", step.ID, err)
				return
			}
			identity = step.Spec.Name + "@" + step.Spec.Version
			run.selections = append(run.selections, nil)
		} else {
			inputs, err := remoteInputsForStep(step, resolve)
			if err != nil {
				run.err = fmt.Errorf("step %s inputs: %w", step.ID, err)
				return
			}
			inputPayload := payloadForArtifact(data, run.steps[index-1])
			result, selection, dispatchErr := rpc.dispatchRemoteStep(ctx, plan, index, assignmentBuildID, inputPayload, nil, inputs)
			clear(inputPayload.Data)
			for _, material := range inputs {
				clear(material.Data)
			}
			run.selections = append(run.selections, selection)
			if dispatchErr != nil {
				run.err = fmt.Errorf("step %s: %w", step.ID, dispatchErr)
				return
			}
			data, name, identity = result.data, result.info.Name, result.identity
			info = result.info
			if step.Output.Kind == "shellcode" || step.Output.Kind == "stage" {
				name = run.result.ImplantName + ".bin"
				info, err = artifactaccept.InspectBytes(data, step.Output, name)
				if err != nil {
					run.err = fmt.Errorf("step %s artifact: %w", step.ID, err)
					return
				}
			}
		}
		run.steps = append(run.steps, info)
		run.identities = append(run.identities, identity)
	}
	run.failed = len(plan.Steps) - 1
	if product.SaveFinal != nil {
		buildID, err := product.SaveFinal(ctx, plan.FinalArtifact, name, data)
		if err != nil {
			run.err = fmt.Errorf("save final build artifact: %w", err)
			run.steps = run.steps[:len(run.steps)-1]
			run.failed = len(run.steps)
			return
		}
		run.result.BuildID = buildID
	}
	run.result.FileName, run.result.Data = name, data
	return
}

func (rpc *Server) prepareRemoteCore(plan buildplan.Plan) (*clientpb.ExternalImplantConfig, error) {
	if len(plan.Steps) == 0 || plan.Steps[0].Spec.Name != "builtin/sliver" {
		return nil, errors.New("no installed remote core preparation for this spec or name")
	}
	name := plan.Request.Name
	if name == "" {
		identifier, err := newStepAssignmentID()
		if err != nil {
			return nil, fmt.Errorf("allocate remote build name: %w", err)
		}
		name = "build" + identifier[:16]
	}
	adapter := generate.SliverCoreAdapter{}
	config, err := adapter.ConfigFromPlan(plan)
	if err != nil {
		return nil, err
	}
	httpC2, err := db.LoadHTTPC2ConfigByName(config.HTTPC2ConfigName)
	if err != nil || httpC2 == nil || httpC2.ImplantConfig == nil {
		return nil, errors.New("Sliver HTTP C2 configuration is unavailable")
	}
	prepared, err := generate.SliverExternal(name, config)
	if err != nil {
		return nil, err
	}
	prepared.HTTPC2 = httpC2
	prepared.Encoders = map[string]uint64{
		"base32": encoders.Base32EncoderID, "base58": encoders.Base58EncoderID,
		"base64": encoders.Base64EncoderID, "english": encoders.EnglishEncoderID,
		"gzip": encoders.GzipEncoderID, "hex": encoders.HexEncoderID, "png": encoders.PNGEncoderID,
	}
	return prepared, nil
}

func saveRemoteSliverFinal(ctx context.Context, plan buildplan.Plan, prepared *clientpb.ExternalImplantConfig,
	final buildplan.Artifact, name string, data []byte) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if !samePipelineArtifact(final, plan.FinalArtifact) || prepared == nil || prepared.Build == nil || prepared.Config == nil {
		return "", errors.New("remote final differs from frozen Sliver plan")
	}
	if _, err := artifactaccept.InspectBytes(data, final, name); err != nil {
		return "", err
	}
	if final.Kind == "stage" {
		stageType, err := generate.SliverStageType(plan)
		if err != nil {
			return "", err
		}
		prepared.Build.Stage = true
		if err := generate.SaveStage(prepared.Build, prepared.Config, data, stageType); err != nil {
			return "", err
		}
	} else {
		file, err := os.CreateTemp(assets.GetRootAppDir(), "remote-final-*")
		if err != nil {
			return "", err
		}
		defer os.Remove(file.Name())
		if err := file.Chmod(0600); err != nil {
			file.Close()
			return "", err
		}
		if _, err := file.Write(data); err != nil {
			file.Close()
			return "", err
		}
		if err := file.Close(); err != nil {
			return "", err
		}
		if err := generate.ImplantBuildSave(prepared.Build, prepared.Config, file.Name()); err != nil {
			return "", err
		}
	}
	return prepared.Build.ID, nil
}

func remoteInputsForStep(step buildplan.Step, resolve buildtransform.ResolveInput) ([]*clientpb.BuildInput, error) {
	var result []*clientpb.BuildInput
	seen := make(map[string]bool)
	add := func(kind, name string, required bool) error {
		if seen[kind+"\x00"+name] {
			return nil
		}
		if resolve == nil {
			if required {
				return errors.New("per-build input resolver is unavailable")
			}
			return nil
		}
		data, err := resolve(kind, name)
		if err != nil {
			if required {
				return err
			}
			return nil
		}
		seen[kind+"\x00"+name] = true
		result = append(result, &clientpb.BuildInput{Kind: kind, Name: name, Data: data})
		return nil
	}
	for _, option := range step.Options {
		var ref struct {
			Kind string `json:"kind"`
			Name string `json:"name"`
		}
		if json.Unmarshal(option.ValueJSON, &ref) != nil || ref.Name == "" {
			continue
		}
		switch ref.Kind {
		case "secret", "input-artifact", "metadata-config":
			if err := add(ref.Kind, ref.Name, true); err != nil {
				return nil, err
			}
			if ref.Kind == "input-artifact" && strings.HasPrefix(ref.Name, "metadata-sha256:") {
				if err := add("metadata-config", ref.Name, false); err != nil {
					return nil, err
				}
			}
		}
	}
	return result, nil
}
