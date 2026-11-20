package buildplan

import (
	"context"
	"errors"
	"testing"

	"github.com/bishopfox/sliver/server/buildcatalog"
)

func builtinPlanner(t *testing.T, toolchain ToolchainCheck) *Planner {
	t.Helper()
	catalog, err := buildcatalog.Open(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	return NewPlanner(catalog, toolchain)
}

func baseRequest(os, arch, format string) Request {
	return Request{
		Target: Target{OS: os, Arch: arch}, Mode: "session", FinalFormat: format,
		C2: []Endpoint{{Protocol: "mtls", Address: "mtls://127.0.0.1:8888"}},
	}
}

func acceptToolchain(string, buildcatalog.Snapshot, Target, Toolchain) bool { return true }

func TestBuiltinRoutesAndExplicitEncoder(t *testing.T) {
	for _, test := range []struct {
		name, os, arch, format, coreMode, converter string
		steps                                       int
	}{
		{"windows shellcode", "windows", "amd64", "shellcode", "pie", "builtin/donut", 2},
		{"linux shellcode", "linux", "amd64", "shellcode", "c-shared", "builtin/malasada", 2},
		{"darwin shellcode", "darwin", "arm64", "shellcode", "c-shared", "builtin/beignet", 2},
		{"windows shared", "windows", "amd64", "shared", "c-shared", "", 1},
		{"windows executable", "windows", "amd64", "exe", "default", "", 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			planner := builtinPlanner(t, acceptToolchain)
			plan, err := planner.Plan(context.Background(), baseRequest(test.os, test.arch, test.format))
			if err != nil {
				t.Fatal(err)
			}
			if len(plan.Steps) != test.steps || plan.Steps[0].Output.GoBuildMode != test.coreMode ||
				!artifactEqual(plan.Steps[len(plan.Steps)-1].Output, plan.FinalArtifact) {
				t.Fatalf("wrong route: %+v", plan.Steps)
			}
			if test.converter != "" && plan.Steps[1].Spec.Name != test.converter {
				t.Fatalf("wrong converter: %s", plan.Steps[1].Spec.Name)
			}
			if plan.SHA256 == "" || plan.Steps[0].Spec.Digest == "" || plan.Steps[0].Provider != "server" {
				t.Fatalf("plan did not freeze its identity: %+v", plan)
			}
		})
	}

	planner := builtinPlanner(t, acceptToolchain)
	request := baseRequest("windows", "amd64", "shellcode")
	request.Transforms = []TransformRequest{{Spec: Reference{Name: "builtin/shikata-ga-nai"}}}
	plan, err := planner.Plan(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Steps) != 3 || plan.Steps[1].Spec.Name != "builtin/donut" ||
		plan.Steps[2].Spec.Name != "builtin/shikata-ga-nai" {
		t.Fatalf("explicit encoder did not follow an implicit converter: %+v", plan.Steps)
	}
}

func TestPlannerRejectsUnsupportedBeforeBuild(t *testing.T) {
	planner := builtinPlanner(t, acceptToolchain)
	request := baseRequest("windows", "arm64", "shellcode")
	assertPlanCode(t, planner, request, CodeIncompatible)

	request = baseRequest("linux", "amd64", "exe")
	request.C2[0].Address = "https://127.0.0.1:8888"
	assertPlanCode(t, planner, request, CodeInvalidRequest)

	request = baseRequest("windows", "amd64", "exe")
	request.CoreOutput = &Artifact{Kind: "executable", Container: "pe", GoBuildMode: "pie"}
	assertPlanCode(t, planner, request, CodeIncompatible) // PIE is only an intermediate artifact.

	request = baseRequest("windows", "amd64", "shellcode")
	request.Transforms = []TransformRequest{{Spec: Reference{Name: "builtin/donut"},
		Options: []Option{{Name: "entropy", ValueJSON: []byte("9007199254740992")}}}}
	assertPlanCode(t, planner, request, CodeInvalidRequest)

	request = baseRequest("windows", "amd64", "exe")
	request.CoreOptions = []Option{{Name: "debug", ValueJSON: []byte("null")}}
	assertPlanCode(t, planner, request, CodeInvalidRequest)
}

func TestToolchainUnavailable(t *testing.T) {
	planner := builtinPlanner(t, func(_ string, _ buildcatalog.Snapshot, _ Target, toolchain Toolchain) bool {
		for _, requirement := range toolchain.Requirements {
			if requirement == "cgo" {
				return false
			}
		}
		return true
	})
	assertPlanCode(t, planner, baseRequest("linux", "amd64", "shared"), CodeUnavailableProvider)
	if _, err := planner.Plan(context.Background(), baseRequest("linux", "amd64", "exe")); err != nil {
		t.Fatal(err)
	}
}

type testCatalog struct {
	core       buildcatalog.Snapshot
	transforms []buildcatalog.Snapshot
}

func (catalog testCatalog) Get(ref buildcatalog.Reference) (buildcatalog.Snapshot, error) {
	if ref.Name == catalog.core.Reference.Name {
		return catalog.core, nil
	}
	for _, snapshot := range catalog.transforms {
		if ref.Name == snapshot.Reference.Name {
			return snapshot, nil
		}
	}
	return buildcatalog.Snapshot{}, buildcatalog.ErrNotFound
}

func (catalog testCatalog) List(query buildcatalog.Query) []buildcatalog.Snapshot {
	if query.Kind == "TransformSpec" {
		return catalog.transforms
	}
	return nil
}

func TestAutomaticRouteAmbiguity(t *testing.T) {
	target := buildcatalog.Target{OS: "windows", Arch: "amd64"}
	provider := []buildcatalog.ProviderStatus{{Name: "server", Online: true}}
	core := buildcatalog.Snapshot{Kind: "CoreSpec", Reference: buildcatalog.Reference{Name: "builtin/sliver", Version: "1", Digest: "core"},
		Providers: provider, Available: true, Core: &buildcatalog.CoreSpec{Capabilities: []buildcatalog.CoreCapability{{
			Targets: []buildcatalog.Target{target}, Mode: "session", Protocols: []string{"mtls"},
			Output:    buildcatalog.CoreOutput{Artifact: buildcatalog.Artifact{Kind: "executable", Container: "pe", GoBuildMode: "pie"}},
			Toolchain: buildcatalog.Toolchain{Driver: "go"},
		}}}}
	transforms := make([]buildcatalog.Snapshot, 2)
	for i, name := range []string{"one", "two"} {
		transforms[i] = buildcatalog.Snapshot{Kind: "TransformSpec", Reference: buildcatalog.Reference{Name: name, Version: "1", Digest: name},
			Providers: provider, Available: true, Transform: &buildcatalog.TransformSpec{Capabilities: []buildcatalog.TransformCapability{{
				Targets: []buildcatalog.Target{target}, Input: buildcatalog.Artifact{Kind: "executable", Container: "pe", GoBuildMode: "pie"},
				Output: buildcatalog.Artifact{Kind: "shellcode", Container: "raw"}, Toolchain: buildcatalog.Toolchain{Driver: "builtin"},
			}}}}
	}
	planner := NewPlanner(testCatalog{core: core, transforms: transforms}, acceptToolchain)
	assertPlanCode(t, planner, baseRequest("windows", "amd64", "shellcode"), CodeAmbiguousRoute)
}

func assertPlanCode(t *testing.T, planner *Planner, request Request, code ErrorCode) {
	t.Helper()
	_, err := planner.Plan(context.Background(), request)
	var planError *PlanError
	if !errors.As(err, &planError) || planError.Code != code {
		t.Fatalf("wanted %s, got %v", code, err)
	}
}
