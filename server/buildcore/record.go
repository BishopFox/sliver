package buildcore

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"github.com/bishopfox/sliver/server/buildplan"
)

// CompleteRecord is the compatibility recorder for callers that already have
// one final artifact. The ordered RPC executor records observed metadata for
// every step and stores its final bytes directly.
func CompleteRecord(store *buildplan.Store, plan buildplan.Plan, result Result) (buildplan.Record, error) {
	if store == nil || result.BuildID == "" || result.FileName == "" || len(plan.Steps) == 0 {
		return buildplan.Record{}, errors.New("incomplete core build result or record store")
	}
	_, err := store.CreateWithID(result.BuildID, plan)
	if err != nil {
		return buildplan.Record{}, err
	}
	hash := sha256.Sum256(result.Data)
	final := buildplan.ArtifactInfo{
		Type:   plan.FinalArtifact,
		Name:   result.FileName,
		Size:   int64(len(result.Data)),
		SHA256: hex.EncodeToString(hash[:]),
	}
	for index, step := range plan.Steps {
		if _, err = store.UpdateStep(result.BuildID, index, buildplan.StepUpdate{State: buildplan.StepRunning}); err != nil {
			return buildplan.Record{}, err
		}
		update := buildplan.StepUpdate{State: buildplan.StepComplete, Progress: 100}
		if step.Kind == "core" {
			update.ToolchainIdentity = result.ToolchainIdentity
		} else {
			update.ToolchainIdentity = step.Spec.Name + "@" + step.Spec.Version
		}
		if index == len(plan.Steps)-1 {
			update.Artifact = &final
		}
		if _, err = store.UpdateStep(result.BuildID, index, update); err != nil {
			return buildplan.Record{}, err
		}
	}
	return store.Complete(result.BuildID, final, result.Data)
}
