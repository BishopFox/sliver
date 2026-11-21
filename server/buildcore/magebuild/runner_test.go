package magebuild

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/magefile/mage/mage"
)

//nolint:gocyclo // Both the compiled runner and CLI must consume the same fixture and manifest.
func TestBuiltinSliverMageBuild(t *testing.T) {
	goCmd, err := exec.LookPath("go")
	if err != nil {
		t.Skip("host Go compiler is unavailable")
	}
	root := t.TempDir()
	work := filepath.Join(root, "work")
	source := filepath.Join(work, "src")
	bin := filepath.Join(work, "bin")
	input := filepath.Join(work, "inputs")
	writeMageFixture(t, source, bin, input)
	goRoot, err := exec.CommandContext(t.Context(), goCmd, "env", "GOROOT").Output()
	if err != nil {
		t.Fatalf("resolve fixture GOROOT: %v", err)
	}
	bundle := filepath.Join(work, "magefiles")
	if err := os.Mkdir(bundle, 0700); err != nil {
		t.Fatal(err)
	}
	digest, err := StageBuiltinSliverBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}
	if expected, err := BuiltinSliverDigest(); err != nil || digest != expected {
		t.Fatalf("staged Magefile digest: %s, want %s: %v", digest, expected, err)
	}
	destination := filepath.Join(bin, executableName("fixture"))
	resultPath := filepath.Join(bin, "result.json")
	manifest := GoManifest{
		Version: 1,
		GoConfig: GoConfig{
			ProjectDir: work, GOOS: runtime.GOOS, GOARCH: runtime.GOARCH,
			GOROOT: strings.TrimSpace(string(goRoot)), GOCACHE: filepath.Join(root, "go-cache"),
			GOMODCACHE: filepath.Join(root, "mod-cache"), GOPROXY: "off", CGO: "0",
		},
		SourceDir: source, Destination: destination, ResultPath: resultPath,
	}
	encoded, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(input, "build.json")
	if err := os.WriteFile(manifestPath, encoded, 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	var stderr bytes.Buffer
	runner := Runner{RootDir: root, GoCmd: goCmd}
	if err := runner.Run(ctx, Invocation{
		BundleDir: bundle, BundleDigest: digest, WorkDir: work,
		ManifestPath: manifestPath, Target: "build",
		Env:    append(os.Environ(), "GOTOOLCHAIN=local", "GOCACHE="+filepath.Join(root, "go-cache")),
		Stderr: &stderr,
	}); err != nil {
		t.Fatalf("Mage build failed: %v\n%s", err, stderr.String())
	}
	data, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatal(err)
	}
	var result Result
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	if result.Primary != filepath.Base(destination) || result.DisplayName != result.Primary {
		t.Fatalf("unexpected Mage result: %+v", result)
	}
	output, err := exec.Command(destination).CombinedOutput()
	if err != nil || strings.TrimSpace(string(output)) != "mage-build-ok" {
		t.Fatalf("compiled artifact: %q, %v", output, err)
	}
	checkMageCLI(t, bundle, work, manifestPath, resultPath, destination)
}

func writeMageFixture(t *testing.T, source, bin, input string) {
	t.Helper()
	for _, dir := range []string{source, bin, input} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(source, "go.mod"), []byte("module example.com/magefixture\n\ngo 1.22\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "main.go"), []byte("package main\nimport \"fmt\"\nfunc main() { fmt.Println(\"mage-build-ok\") }\n"), 0600); err != nil {
		t.Fatal(err)
	}
}

func checkMageCLI(t *testing.T, bundle, work, manifestPath, resultPath, destination string) {
	t.Helper()
	// The same installed Magefile must remain usable through the Mage CLI
	// argument path, independent of the server's cached compiled runner.
	if err := os.Remove(destination); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(resultPath); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	if code := mage.ParseAndRun(&bytes.Buffer{}, &stderr, nil,
		[]string{"-d", bundle, "-w", work, "build", manifestPath}); code != 0 {
		t.Fatalf("Mage CLI path exited %d: %s", code, stderr.String())
	}
	if _, err := os.Stat(resultPath); err != nil {
		t.Fatalf("Mage CLI path did not publish result: %v", err)
	}
	output, err := exec.Command(destination).CombinedOutput()
	if err != nil || strings.TrimSpace(string(output)) != "mage-build-ok" {
		t.Fatalf("Mage CLI artifact: %q, %v", output, err)
	}
}

func TestDigestBundleRejectsSymlinkAndMutation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "magefile.go")
	if err := os.WriteFile(path, []byte("package main"), 0600); err != nil {
		t.Fatal(err)
	}
	first, err := DigestBundle(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("package main\n"), 0600); err != nil {
		t.Fatal(err)
	}
	second, err := DigestBundle(dir)
	if err != nil || second == first {
		t.Fatalf("changed source did not change digest: %s %s %v", first, second, err)
	}
	if err := os.Symlink(path, filepath.Join(dir, "link.go")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := DigestBundle(dir); err == nil {
		t.Fatal("accepted symlink in Magefile bundle")
	}
}

func TestRunnerRejectsChangedBundleBeforeExecution(t *testing.T) {
	root := t.TempDir()
	work := filepath.Join(root, "work")
	input := filepath.Join(work, "inputs")
	if err := os.MkdirAll(input, 0700); err != nil {
		t.Fatal(err)
	}
	bundle := filepath.Join(work, "magefiles")
	digest, err := StageBuiltinSliverBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bundle, "extra.go"), []byte("// changed"), 0600); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(input, "build.json")
	if err := os.WriteFile(manifest, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	err = (Runner{RootDir: root}).Run(context.Background(), Invocation{
		BundleDir: bundle, BundleDigest: digest, WorkDir: work,
		ManifestPath: manifest, Target: "Build",
	})
	if err == nil || !strings.Contains(err.Error(), "digest") {
		t.Fatalf("changed Magefile bundle was accepted: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "mage-runners")); !os.IsNotExist(err) {
		t.Fatal("untrusted Magefile bundle reached compilation")
	}
}
