package buildtransform

import (
	"strings"
	"testing"

	"github.com/bishopfox/sliver/server/buildplan"
)

func TestValidateInstalledStepAcceptsAllLocalTransformFamilies(t *testing.T) {
	windowTarget := buildplan.Target{OS: "windows", Arch: "amd64"}
	pe := buildplan.Artifact{Kind: "executable", Container: "pe", Target: windowTarget, GoBuildMode: "default"}
	stage := buildplan.Artifact{Kind: "stage", Container: "raw", Target: windowTarget}
	steps := []buildplan.Step{
		converterTestStep("builtin/donut", "windows", "amd64", "executable", "pe", "pie"),
		converterTestStep("builtin/beignet", "darwin", "arm64", "shared-library", "macho", "c-shared"),
		converterTestStep("builtin/malasada", "linux", "amd64", "shared-library", "elf", "c-shared"),
		encoderStep("builtin/shikata-ga-nai", windowTarget),
		encoderStep("builtin/xor", windowTarget),
		encoderStep("builtin/xor-dynamic", windowTarget),
		testStageStep(StageFrameSpec, pe, stage, testStageOption("prependSize", "true")),
		testStageStep(StageCompressSpec, stage, stage, testStageOption("algorithm", `"zlib"`)),
		testStageStep(StageAESSpec, stage, stage,
			testStageOption("key", `{"kind":"secret","name":"key"}`),
			testStageOption("iv", `{"kind":"secret","name":"iv"}`)),
		testStageStep(StageRC4Spec, stage, stage,
			testStageOption("key", `{"kind":"secret","name":"key"}`)),
		testStageStep(PEMetadataSpec, pe, pe,
			testStageOption("donorArtifact", `{"kind":"input-artifact","name":"donor"}`)),
	}
	for _, step := range steps {
		t.Run(step.Spec.Name, func(t *testing.T) {
			if err := ValidateInstalledStep(step); err != nil {
				t.Fatalf("valid installed step rejected: %v", err)
			}
		})
	}
}

func TestValidateInstalledStepRejectsUninstalledAndInvalidSteps(t *testing.T) {
	target := buildplan.Target{OS: "windows", Arch: "amd64"}
	pe := buildplan.Artifact{Kind: "executable", Container: "pe", Target: target, GoBuildMode: "default"}
	stage := buildplan.Artifact{Kind: "stage", Container: "raw", Target: target}
	base := testStageStep(StageFrameSpec, pe, stage)
	for _, tc := range []struct {
		name string
		step buildplan.Step
	}{
		{"unknown spec", testStageStep("other/transform", pe, stage)},
		{"remote provider", func() buildplan.Step { s := base; s.Provider = "remote"; return s }()},
		{"wrong toolchain", func() buildplan.Step { s := base; s.Toolchain.Driver = "go"; return s }()},
		{"requirements", func() buildplan.Step {
			s := base
			s.Toolchain.Requirements = []string{"command:go"}
			return s
		}()},
		{"wrong digest", func() buildplan.Step {
			s := base
			s.Spec.Digest = "sha256:" + strings.Repeat("0", 64)
			return s
		}()},
		{"invalid frame option", testStageStep(StageFrameSpec, pe, stage, testStageOption("prependSize", "null"))},
		{"unsupported shellcode frame target", func() buildplan.Step {
			t := buildplan.Target{OS: "windows", Arch: "arm64"}
			input := buildplan.Artifact{Kind: "shellcode", Container: "raw", Target: t}
			output := buildplan.Artifact{Kind: "stage", Container: "raw", Target: t}
			return testStageStep(StageFrameSpec, input, output)
		}()},
		{"unsupported stage target", func() buildplan.Step {
			t := buildplan.Target{OS: "freebsd", Arch: "amd64"}
			artifact := buildplan.Artifact{Kind: "stage", Container: "raw", Target: t}
			return testStageStep(StageCompressSpec, artifact, artifact)
		}()},
		{"invalid compression", testStageStep(StageCompressSpec, stage, stage, testStageOption("algorithm", `"brotli"`))},
		{"missing AES IV", testStageStep(StageAESSpec, stage, stage,
			testStageOption("key", `{"kind":"secret","name":"key"}`))},
		{"invalid RC4 reference", testStageStep(StageRC4Spec, stage, stage,
			testStageOption("key", `{"kind":"input-artifact","name":"key"}`))},
		{"invalid donor reference", testStageStep(PEMetadataSpec, pe, pe,
			testStageOption("donorArtifact", `{"kind":"secret","name":"donor"}`))},
		{"unsupported PE build mode", func() buildplan.Step {
			artifact := pe
			artifact.GoBuildMode = "pie"
			return testStageStep(PEMetadataSpec, artifact, artifact,
				testStageOption("donorArtifact", `{"kind":"input-artifact","name":"donor"}`))
		}()},
		{"invalid converter option", func() buildplan.Step {
			s := converterTestStep("builtin/donut", "windows", "amd64", "executable", "pe", "pie")
			s.Options = []buildplan.Option{testStageOption("entropy", "4")}
			return s
		}()},
		{"invalid encoder option", func() buildplan.Step {
			s := encoderStep("builtin/xor", target)
			s.Options = []buildplan.Option{testStageOption("iterations", "2")}
			return s
		}()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateInstalledStep(tc.step); err == nil {
				t.Fatal("invalid step was accepted before compile")
			}
		})
	}
}
