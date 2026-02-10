package core

import "testing"

func TestTrackAndRemoveExternalBuildAssignment(t *testing.T) {
	buildID := "build-test-id"
	TrackExternalBuildAssignment(buildID, "builder-a", "operator-a")
	t.Cleanup(func() {
		RemoveExternalBuildAssignment(buildID)
	})

	assignment := GetExternalBuildAssignment(buildID)
	if assignment == nil {
		t.Fatalf("expected assignment for build %s", buildID)
	}
	if assignment.BuilderName != "builder-a" {
		t.Fatalf("expected builder-a, got %s", assignment.BuilderName)
	}
	if assignment.OperatorName != "operator-a" {
		t.Fatalf("expected operator-a, got %s", assignment.OperatorName)
	}

	RemoveExternalBuildAssignment(buildID)
	if GetExternalBuildAssignment(buildID) != nil {
		t.Fatalf("expected assignment removal for build %s", buildID)
	}
}
