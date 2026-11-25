// Package minimal installs a small, independent sliver-v1 core. Its Magefile
// builds a session implant with mTLS registration and Ping support.
package minimal

import (
	"bytes"
	"context"
	"crypto/sha256"
	"embed"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/bishopfox/sliver/implant"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/buildcore"
	"github.com/bishopfox/sliver/server/buildcore/workspace"
	"github.com/bishopfox/sliver/server/buildplan"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/cryptography"
	"github.com/bishopfox/sliver/server/generate"
	"github.com/bishopfox/sliver/server/gogo"
)

const (
	// SpecName names the installed minimal core specification.
	SpecName = "fixture/minimal-sliver-v1"
	// ExecutorID identifies the installed minimal core executor.
	ExecutorID              = "minimal-sliver-v1-go"
	maxMaterialBytes        = 16 << 20
	defaultReconnectSeconds = uint64(2)
	maxReconnectSeconds     = uint64(9223372036)
)

//go:embed source/main.go magefiles/minimal/magefile.go
var files embed.FS

// Adapter implements the installed minimal core specification.
type Adapter struct{}

type material struct {
	Name                    string `json:"name"`
	Endpoint                string `json:"endpoint"`
	ReconnectSeconds        uint64 `json:"reconnectSeconds"`
	TargetOS                string `json:"targetOS"`
	TargetArch              string `json:"targetArch"`
	PeerPrivateKey          string `json:"peerPrivateKey"`
	PeerPublicKey           string `json:"peerPublicKey"`
	PeerPublicKeyDigest     string `json:"peerPublicKeyDigest"`
	PeerPublicKeySignature  string `json:"peerPublicKeySignature"`
	AgeServerPublicKey      string `json:"ageServerPublicKey"`
	MinisignServerPublicKey string `json:"minisignServerPublicKey"`
	MtlsCACert              string `json:"mtlsCACert"`
	MtlsCert                string `json:"mtlsCert"`
	MtlsKey                 string `json:"mtlsKey"`
}

type manifest struct {
	SourceDir    string `json:"sourceDir"`
	BinDir       string `json:"binDir"`
	MaterialPath string `json:"materialPath"`
	GoBinary     string `json:"goBinary"`
	GoRoot       string `json:"goRoot"`
	GoCache      string `json:"goCache"`
	GoTempDir    string `json:"goTempDir"`
	GoModCache   string `json:"goModCache"`
	GOOS         string `json:"goos"`
	GOARCH       string `json:"goarch"`
	OutputName   string `json:"outputName"`
}

func init() {
	if err := buildcore.RegisterDefault(Adapter{}); err != nil {
		panic(err)
	}
}

// ExecutorID returns the ID of the minimal core executor.
func (Adapter) ExecutorID() string { return ExecutorID }

// Binding returns the build source and target binding.
func (Adapter) Binding() buildcore.MageBinding {
	return buildcore.MageBinding{SourceMode: "copy", Magefile: "magefiles/minimal", Target: "Build"}
}

// Supports reports whether the adapter accepts the target and toolchain.
func (Adapter) Supports(target buildplan.Target, toolchain buildplan.Toolchain) bool {
	if toolchain.Driver != "go" || len(toolchain.Requirements) != 1 || toolchain.Requirements[0] != "go" {
		return false
	}
	switch {
	case target.OS == "darwin" && target.Arch == "arm64":
	case target.OS == "linux" && (target.Arch == "amd64" || target.Arch == "arm64"):
	default:
		return false
	}
	path := filepath.Join(gogo.GetGoRootDir(assets.GetRootAppDir()), "bin", goExecutable())
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0100 != 0
}

func goExecutable() string {
	if runtime.GOOS == "windows" {
		return "go.exe"
	}
	return "go"
}

// PrepareRemote prepares the material needed for an assigned remote build.
func (Adapter) PrepareRemote(ctx context.Context, plan buildplan.Plan) ([]byte, error) {
	if ctx == nil {
		return nil, errors.New("missing context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := validatePlan(plan); err != nil {
		return nil, err
	}
	endpoint := plan.Request.C2[0].Address
	serverCA, _, err := certs.GetCertificateAuthorityPEM(certs.MtlsServerCA)
	if err != nil {
		return nil, fmt.Errorf("get mTLS server CA: %w", err)
	}
	clientCert, clientKey, err := certs.MtlsC2ImplantGenerateECCCertificate(plan.Request.Name)
	if err != nil {
		return nil, fmt.Errorf("generate mTLS client certificate: %w", err)
	}
	peer, err := cryptography.RandomAgeKeyPair()
	if err != nil {
		return nil, fmt.Errorf("generate peer key: %w", err)
	}
	peerDigest := sha256.Sum256([]byte(peer.Public))
	m := material{
		Name: plan.Request.Name, Endpoint: endpoint, ReconnectSeconds: reconnectSeconds(plan),
		TargetOS: plan.Request.Target.OS, TargetArch: plan.Request.Target.Arch,
		PeerPrivateKey: peer.Private, PeerPublicKey: peer.Public,
		PeerPublicKeyDigest:     hex.EncodeToString(peerDigest[:]),
		PeerPublicKeySignature:  cryptography.MinisignServerSign([]byte(peer.Public)),
		AgeServerPublicKey:      cryptography.AgeServerKeyPair().Public,
		MinisignServerPublicKey: cryptography.MinisignServerPublicKey(),
		MtlsCACert:              string(serverCA), MtlsCert: string(clientCert), MtlsKey: string(clientKey),
	}
	encoded, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	if len(encoded) > maxMaterialBytes {
		return nil, errors.New("core material exceeds size limit")
	}
	return encoded, nil
}

//nolint:gocyclo // All target, artifact, and material requirements are checked at the same validation boundary.
func validatePlan(plan buildplan.Plan) error {
	if len(plan.Steps) == 0 || plan.Steps[0].Kind != "core" || plan.Steps[0].Spec.Name != SpecName ||
		plan.Request.Core.Name != SpecName || plan.Request.Mode != "session" || plan.Request.Name == "" {
		return errors.New("plan is not a minimal sliver-v1 session core")
	}
	core := plan.Steps[0]
	if core.Output.Target != plan.Request.Target || core.Output.Kind != "executable" ||
		core.Output.GoBuildMode != "default" || len(core.Output.EntrySymbols) != 0 ||
		core.Toolchain.Driver != "go" || len(core.Toolchain.Requirements) != 1 || core.Toolchain.Requirements[0] != "go" {
		return errors.New("minimal core plan has an unsupported artifact or toolchain")
	}
	switch {
	case plan.Request.Target.OS == "darwin" && plan.Request.Target.Arch == "arm64" && core.Output.Container == "macho":
	case plan.Request.Target.OS == "linux" && (plan.Request.Target.Arch == "amd64" || plan.Request.Target.Arch == "arm64") && core.Output.Container == "elf":
	default:
		return errors.New("minimal core plan has an unsupported target")
	}
	if len(plan.Request.C2) != 1 || plan.Request.C2[0].Protocol != "mtls" {
		return errors.New("minimal core requires exactly one mTLS C2 endpoint")
	}
	if settings := plan.Request.C2Settings; settings != nil {
		if settings.ConnectionStrategy != "sequential" || settings.ReconnectSeconds == 0 ||
			settings.ReconnectSeconds > maxReconnectSeconds || settings.PollTimeoutSeconds != 0 ||
			settings.MaxConnectionErrors != 0 || settings.HTTPC2Profile != "default" ||
			settings.WireguardKeyExchangePort != 0 || settings.WireguardTCPCommsPort != 0 {
			return errors.New("minimal core supports sequential mTLS reconnect only; reconnectSeconds must be positive, pollTimeoutSeconds and maxConnectionErrors must be zero, httpC2Profile must be default, and WireGuard ports must be unset")
		}
	}
	u, err := url.Parse(plan.Request.C2[0].Address)
	if err != nil || u.Scheme != "mtls" || u.Hostname() == "" || u.Port() == "" ||
		u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" {
		return errors.New("minimal core requires a concrete mtls://host:port endpoint")
	}
	return nil
}

func reconnectSeconds(plan buildplan.Plan) uint64 {
	if plan.Request.C2Settings == nil {
		return defaultReconnectSeconds
	}
	return plan.Request.C2Settings.ReconnectSeconds
}

//nolint:gocyclo // Frozen-plan matching and cryptographic material validation use the same decoded record.
func parseMaterial(plan buildplan.Plan, raw []byte) (material, error) {
	if err := validatePlan(plan); err != nil {
		return material{}, err
	}
	if len(raw) == 0 || len(raw) > maxMaterialBytes {
		return material{}, errors.New("invalid core material size")
	}
	var m material
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&m); err != nil {
		return material{}, errors.New("invalid core material")
	}
	if err := dec.Decode(new(json.RawMessage)); !errors.Is(err, io.EOF) {
		return material{}, errors.New("trailing core material data")
	}
	if m.Name != plan.Request.Name || m.Endpoint != plan.Request.C2[0].Address ||
		m.ReconnectSeconds != reconnectSeconds(plan) ||
		m.TargetOS != plan.Request.Target.OS || m.TargetArch != plan.Request.Target.Arch ||
		m.PeerPrivateKey == "" || m.MtlsCert == "" || m.MtlsKey == "" || m.MtlsCACert == "" ||
		m.MinisignServerPublicKey == "" {
		return material{}, errors.New("core material does not match frozen plan")
	}
	return m, nil
}

// BuildRemote builds a core with the material supplied for a remote assignment.
func (Adapter) BuildRemote(ctx context.Context, plan buildplan.Plan, raw []byte) (buildcore.Result, error) {
	if _, err := parseMaterial(plan, raw); err != nil {
		return buildcore.Result{}, err
	}
	copyMaterial := append([]byte(nil), raw...)
	defer clear(copyMaterial)
	return mageAdapter(func(context.Context, buildplan.Plan) ([]byte, error) {
		return append([]byte(nil), copyMaterial...), nil
	}).Build(ctx, coreOnlyPlan(plan))
}

// BuildCore builds and validates the core artifact.
func (adapter Adapter) BuildCore(ctx context.Context, plan buildplan.Plan) (buildcore.CoreProduct, error) {
	raw, err := adapter.PrepareRemote(ctx, plan)
	if err != nil {
		return buildcore.CoreProduct{}, err
	}
	result, err := adapter.BuildRemote(ctx, plan, raw)
	if err != nil {
		clear(raw)
		return buildcore.CoreProduct{}, err
	}
	return buildcore.CoreProduct{
		Result: result,
		SaveFinal: func(ctx context.Context, final buildplan.Artifact, name string, data []byte) (string, error) {
			return adapter.CompleteRemote(ctx, plan, raw, final, name, data)
		},
		Close: func() error { clear(raw); return nil },
	}, nil
}

// Build runs a direct local build and persists the artifact.
func (adapter Adapter) Build(ctx context.Context, plan buildplan.Plan) (buildcore.Result, error) {
	if len(plan.Steps) != 1 || plan.Steps[0].Provider != "server" {
		return buildcore.Result{}, errors.New("direct minimal core build requires one step")
	}
	product, err := adapter.BuildCore(ctx, plan)
	if err != nil {
		return buildcore.Result{}, err
	}
	defer func() { _ = product.Close() }()
	id, err := product.SaveFinal(ctx, plan.FinalArtifact, product.Result.FileName, product.Result.Data)
	if err != nil {
		return buildcore.Result{}, err
	}
	product.Result.BuildID = id
	return product.Result, nil
}

func coreOnlyPlan(plan buildplan.Plan) buildplan.Plan {
	only := plan
	only.Steps = []buildplan.Step{plan.Steps[0]}
	only.Steps[0].Provider = "server"
	only.FinalArtifact = only.Steps[0].Output
	return only
}

// CompleteRemote accepts the result of an assigned remote build.
func (Adapter) CompleteRemote(ctx context.Context, plan buildplan.Plan, raw []byte, final buildplan.Artifact, name string, data []byte) (string, error) {
	if ctx == nil {
		return "", errors.New("missing context")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	m, err := parseMaterial(plan, raw)
	if err != nil {
		return "", err
	}
	if name == "" || filepath.Base(name) != name || len(data) == 0 || final.Target != plan.Request.Target {
		return "", errors.New("invalid final core artifact")
	}
	appDir := assets.GetRootAppDir()
	staging, err := os.MkdirTemp(appDir, "minimal-save-*")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.RemoveAll(staging) }()
	path := filepath.Join(staging, "artifact")
	if err := os.WriteFile(path, data, 0600); err != nil {
		return "", err
	}
	build := &clientpb.ImplantBuild{
		Name: m.Name, AgeServerPublicKey: m.AgeServerPublicKey,
		PeerPublicKey: m.PeerPublicKey, PeerPrivateKey: m.PeerPrivateKey,
		PeerPublicKeyDigest: m.PeerPublicKeyDigest, PeerPublicKeySignature: m.PeerPublicKeySignature,
		MinisignServerPublicKey: m.MinisignServerPublicKey,
		MtlsCACert:              m.MtlsCACert, MtlsCert: m.MtlsCert, MtlsKey: m.MtlsKey,
	}
	config := &clientpb.ImplantConfig{
		GOOS: m.TargetOS, GOARCH: m.TargetArch, Format: clientpb.OutputFormat_EXECUTABLE,
		IncludeMTLS: true, HTTPC2ConfigName: "default",
		ReconnectInterval: int64(time.Duration(m.ReconnectSeconds) * time.Second),
		C2:                []*clientpb.ImplantC2{{URL: m.Endpoint, Priority: 1}},
	}
	if err := generate.ImplantBuildSave(build, config, path); err != nil {
		return "", fmt.Errorf("save minimal core build: %w", err)
	}
	return build.ID, nil
}

func mageAdapter(prepare func(context.Context, buildplan.Plan) ([]byte, error)) buildcore.MageCoreAdapter {
	source, _ := fs.Sub(files, "source")
	digest := mageDigest()
	appDir := assets.GetRootAppDir()
	goRoot := gogo.GetGoRootDir(appDir)
	return buildcore.MageCoreAdapter{
		ID: ExecutorID, AppDir: appDir, BindingSpec: Adapter{}.Binding(),
		Source: source, Magefiles: files, BundleDigest: digest,
		GoCmd:       filepath.Join(goRoot, "bin", goExecutable()),
		Env:         []string{"PATH=" + filepath.Join(goRoot, "bin") + string(os.PathListSeparator) + os.Getenv("PATH")},
		ToolchainID: generate.SliverToolchainIdentity(),
		Available:   Adapter{}.Supports, Prepare: prepare, Manifest: writeManifest,
	}
}

func mageDigest() string {
	data, err := files.ReadFile("magefiles/minimal/magefile.go")
	if err != nil {
		panic(err)
	}
	h := sha256.New()
	_, _ = io.WriteString(h, "magefile.go")
	_, _ = h.Write([]byte{0})
	_ = binary.Write(h, binary.LittleEndian, int64(len(data)))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write(data)
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

func writeManifest(ctx context.Context, plan buildplan.Plan, work *workspace.Workspace, materialPath string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if materialPath == "" {
		return nil, errors.New("missing private core material")
	}
	if err := assets.SetupGoPath(work.SourceDir, false); err != nil {
		return nil, fmt.Errorf("stage protobuf source: %w", err)
	}
	if err := os.WriteFile(filepath.Join(work.SourceDir, "go.mod"), []byte(implant.GoMod), 0600); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(work.SourceDir, "go.sum"), []byte(implant.GoSum), 0600); err != nil {
		return nil, err
	}
	if err := fs.WalkDir(implant.Vendor, "vendor", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		destination := filepath.Join(work.SourceDir, filepath.FromSlash(path))
		if entry.IsDir() {
			return os.MkdirAll(destination, 0700)
		}
		if !entry.Type().IsRegular() {
			return errors.New("embedded vendor contains non-regular file")
		}
		data, err := implant.Vendor.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(destination, data, 0600)
	}); err != nil {
		return nil, fmt.Errorf("stage vendored dependencies: %w", err)
	}
	appDir := assets.GetRootAppDir()
	goRoot := gogo.GetGoRootDir(appDir)
	// Go caches object files containing the compiled credential constants.
	// Keep those objects in this workspace so Remove clears them with source.
	goCache := filepath.Join(work.Root, "go-cache")
	if err := os.Mkdir(goCache, 0700); err != nil {
		return nil, fmt.Errorf("create private compiler cache: %w", err)
	}
	goTempDir := filepath.Join(work.Root, "go-tmp")
	if err := os.Mkdir(goTempDir, 0700); err != nil {
		return nil, fmt.Errorf("create private compiler temp directory: %w", err)
	}
	m := manifest{
		SourceDir: work.SourceDir, BinDir: work.BinDir, MaterialPath: materialPath,
		GoBinary: filepath.Join(goRoot, "bin", goExecutable()), GoRoot: goRoot,
		GoCache: goCache, GoTempDir: goTempDir, GoModCache: gogo.GetGoModCache(appDir),
		GOOS: plan.Request.Target.OS, GOARCH: plan.Request.Target.Arch,
		OutputName: plan.Request.Name,
	}
	return json.Marshal(m)
}
