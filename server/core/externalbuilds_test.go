package core

import (
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
)

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

func TestExternalBuildAssignmentsFollowExactBuilderRegistration(t *testing.T) {
	first := &clientpb.Builder{Name: "external-disconnect-test", OperatorName: "operator-a"}
	if err := AddBuilder(first); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { RemoveBuilder(first.Name) })
	if !TrackExternalBuildAssignmentForBuilder("first-build", first) {
		t.Fatal("connected builder could not track its build")
	}
	ids := RemoveBuilderAndExternalBuildAssignments(first)
	if len(ids) != 1 || ids[0] != "first-build" || GetExternalBuildAssignment("first-build") != nil {
		t.Fatalf("disconnect did not take first builder assignment: %v", ids)
	}
	if TrackExternalBuildAssignmentForBuilder("late-build", first) {
		t.Fatal("disconnected builder assigned a late build")
	}
	second := &clientpb.Builder{Name: first.Name, OperatorName: first.OperatorName}
	if err := AddBuilder(second); err != nil {
		t.Fatal(err)
	}
	if TrackExternalBuildAssignmentForBuilder("stale-build", first) {
		t.Fatal("old builder assigned a build to the replacement connection")
	}
	if !TrackExternalBuildAssignmentForBuilder("second-build", second) {
		t.Fatal("replacement builder could not track its build")
	}
	if ids := RemoveBuilderAndExternalBuildAssignments(first); len(ids) != 0 || GetExternalBuildAssignment("second-build") == nil {
		t.Fatalf("old disconnect removed replacement assignment: %v", ids)
	}
	if ids := RemoveBuilderAndExternalBuildAssignments(second); len(ids) != 1 || ids[0] != "second-build" {
		t.Fatalf("replacement disconnect did not take its build: %v", ids)
	}
}
