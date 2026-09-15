package e2e

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/bishopfox/sliver/client/command/armory"
	"github.com/bishopfox/sliver/client/command/extensions"
	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util/minisign"
)

const (
	armoryIndexVersion   = "v0.0.45"
	coffLoaderVersion    = "v1.0.16"
	situationalVersion   = "v0.0.28"
	armoryIndexPublicKey = "RWSBpxpRWDrD7Fe+VvRE3c2VEDC2NK80rlNCj+BX0gz44Xw07r6KQD9L"
	coffLoaderPublicKey  = "RWS76vh9dCN1PXI/+xiuXPdDFAM+lI0+AT44AmrISFSbuT/EVsoRS+a0"
	situationalPublicKey = "RWSN1vDbbpNFEi3pKpeoMPIAHIvlOM502TK4zA8zTP5Jn0//+SXJboyQ"
	armoryDownloadLimit  = 128 << 20
	armoryDownloadTries  = 3
	armoryRetryBaseDelay = 250 * time.Millisecond
)

type pinnedAsset struct {
	url    string
	sha256 string
}

var (
	armoryIndexAsset = pinnedAsset{
		url:    "https://github.com/sliverarmory/armory/releases/download/v0.0.45/armory.json",
		sha256: "9484f8c2d57d1ca33fce4234b9729b5c2953b71bee818ba442c7da9da30dd960",
	}
	armoryIndexSignatureAsset = pinnedAsset{
		url:    "https://github.com/sliverarmory/armory/releases/download/v0.0.45/armory.minisig",
		sha256: "a286aadc2336970619791e81616a57f02f2ecfc8157efb126d88889b9dada10e",
	}
	coffLoaderArchiveAsset = pinnedAsset{
		url:    "https://github.com/sliverarmory/COFFLoader/releases/download/v1.0.16/coff-loader.tar.gz",
		sha256: "ccc4e879b1f60627765745743dc8d5660e679a670890b7dc4f54c075bb301eef",
	}
	coffLoaderSignatureAsset = pinnedAsset{
		url:    "https://github.com/sliverarmory/COFFLoader/releases/download/v1.0.16/coff-loader.minisig",
		sha256: "6599f1984d692d0da6f27956d2caf669952edb7a9a48ff0417451e7f3ed0d6fc",
	}
	saEnvArchiveAsset = pinnedAsset{
		url:    "https://github.com/sliverarmory/CS-Situational-Awareness-BOF/releases/download/v0.0.28/sa-env.tar.gz",
		sha256: "acbd56a94de66252a6be35a2c1efe8a25d153074930244df9f3b1677a1b267f6",
	}
	saEnvSignatureAsset = pinnedAsset{
		url:    "https://github.com/sliverarmory/CS-Situational-Awareness-BOF/releases/download/v0.0.28/sa-env.minisig",
		sha256: "b0ffc0c5884ce0f710c8f47f7804b0dd95545cfc1f5ef51bc6e2ddc59534b36b",
	}
	saWhoamiArchiveAsset = pinnedAsset{
		url:    "https://github.com/sliverarmory/CS-Situational-Awareness-BOF/releases/download/v0.0.28/sa-whoami.tar.gz",
		sha256: "43c79a7adfa6c6b9215b25a5b2d3091a7b2ef5edaf9a39203dbe1a927453cf4a",
	}
	saWhoamiSignatureAsset = pinnedAsset{
		url:    "https://github.com/sliverarmory/CS-Situational-Awareness-BOF/releases/download/v0.0.28/sa-whoami.minisig",
		sha256: "69bd904f5f18222a18617e46d3ac397bfd7cc92e0d145dfbfd1eed52c9d3b111",
	}
)

type armoryBOF struct {
	data    []byte
	command *extensions.ExtCommand
}

type armoryAssets struct {
	loaderID      string
	loaderData    []byte
	loaderCommand *extensions.ExtCommand
	saEnv         armoryBOF
	saWhoami      armoryBOF
}

func (s *suite) exerciseArmory(target implantTarget, _ string, transport string) error {
	supported := s.opts.targetOS == "windows" && (s.opts.targetArch == "386" || s.opts.targetArch == "amd64")
	if !supported {
		s.t.Logf("SKIP Armory CS-Situational-Awareness on %s/%s: signed manifests have no exact target", s.opts.targetOS, s.opts.targetArch)
		return nil
	}

	if err := s.localStep("ArmorySignatures", "pinned index and package trust chain", func() error {
		return s.prepareArmory()
	}); err != nil {
		return err
	}

	if err := s.step(target, transport, "RegisterExtension", "signed COFFLoader exact target", func() error {
		_, err := invokeRPC(s, target, "RegisterExtension", func(ctx context.Context, request *commonpb.Request) (*sliverpb.RegisterExtension, error) {
			return s.rpc.RegisterExtension(ctx, &sliverpb.RegisterExtensionReq{
				Name: s.armory.loaderID, Data: s.armory.loaderData, OS: "windows", Init: s.armory.loaderCommand.Init, Request: request,
			})
		}, func(response *sliverpb.RegisterExtension) *commonpb.Response { return response.GetResponse() })
		return err
	}); err != nil {
		return err
	}

	if err := s.step(target, transport, "ListExtensions", "registered COFFLoader digest", func() error {
		response, err := invokeRPC(s, target, "ListExtensions", func(ctx context.Context, request *commonpb.Request) (*sliverpb.ListExtensions, error) {
			return s.rpc.ListExtensions(ctx, &sliverpb.ListExtensionsReq{Request: request})
		}, func(response *sliverpb.ListExtensions) *commonpb.Response { return response.GetResponse() })
		if err != nil {
			return err
		}
		for _, name := range response.Names {
			if name == s.armory.loaderID {
				return nil
			}
		}
		return fmt.Errorf("COFFLoader digest %s missing from %v", s.armory.loaderID, response.Names)
	}); err != nil {
		return err
	}

	expectedMarker := "SLIVER_E2E_MARKER=" + target.name()
	if err := s.exerciseArmoryBOF(target, transport, "signed sa-env BOF through COFFLoader", s.armory.saEnv, func(output []byte) error {
		return validateSAEnvOutput(output, expectedMarker)
	}); err != nil {
		return err
	}
	if err := s.exerciseArmoryBOF(target, transport, "signed sa-whoami BOF through COFFLoader", s.armory.saWhoami, validateSAWhoamiOutput); err != nil {
		return err
	}
	return nil
}

func (s *suite) exerciseArmoryBOF(target implantTarget, transport string, scenario string, bof armoryBOF, validate func([]byte) error) error {
	return s.step(target, transport, "CallExtension", scenario, func() error {
		inner, err := extensions.ParseFlagArgumentsToBuffer(nil, nil, "", bof.command)
		if err != nil {
			return fmt.Errorf("pack %s arguments: %w", bof.command.CommandName, err)
		}
		outer := core.BOFArgsBuffer{Buffer: new(bytes.Buffer)}
		if err := outer.AddString(bof.command.Entrypoint); err != nil {
			return err
		}
		if err := outer.AddData(bof.data); err != nil {
			return err
		}
		if err := outer.AddData(inner); err != nil {
			return err
		}
		args, err := outer.GetBuffer()
		if err != nil {
			return err
		}
		response, err := invokeRPC(s, target, "CallExtension", func(ctx context.Context, request *commonpb.Request) (*sliverpb.CallExtension, error) {
			return s.rpc.CallExtension(ctx, &sliverpb.CallExtensionReq{
				Name: s.armory.loaderID, Export: s.armory.loaderCommand.Entrypoint, Args: args, ServerStore: false, Request: request,
			})
		}, func(response *sliverpb.CallExtension) *commonpb.Response { return response.GetResponse() })
		if err != nil {
			return err
		}
		if err := validate(response.Output); err != nil {
			return err
		}
		return nil
	})
}

func (s *suite) prepareArmory() error {
	s.armoryOnce.Do(func() {
		s.armory, s.armoryErr = downloadArmoryAssets(s.ctx, s.opts.targetArch)
	})
	return s.armoryErr
}

func downloadArmoryAssets(ctx context.Context, targetArch string) (*armoryAssets, error) {
	httpClient := &http.Client{Timeout: 5 * time.Minute}
	indexData, err := downloadPinned(ctx, httpClient, armoryIndexAsset)
	if err != nil {
		return nil, err
	}
	indexSignature, err := downloadPinned(ctx, httpClient, armoryIndexSignatureAsset)
	if err != nil {
		return nil, err
	}
	if err := verifySignature(armoryIndexPublicKey, indexData, indexSignature); err != nil {
		return nil, fmt.Errorf("verify Armory index %s: %w", armoryIndexVersion, err)
	}
	index := &armory.ArmoryIndex{}
	if err := json.Unmarshal(indexData, index); err != nil {
		return nil, fmt.Errorf("decode signed Armory index: %w", err)
	}
	loaderPackage, err := findIndexedPackage(index, "coff-loader", coffLoaderPublicKey, "sliverarmory/COFFLoader")
	if err != nil {
		return nil, err
	}
	saEnvPackage, err := findIndexedPackage(index, "sa-env", situationalPublicKey, "sliverarmory/CS-Situational-Awareness-BOF")
	if err != nil {
		return nil, err
	}
	saWhoamiPackage, err := findIndexedPackage(index, "sa-whoami", situationalPublicKey, "sliverarmory/CS-Situational-Awareness-BOF")
	if err != nil {
		return nil, err
	}

	loaderManifest, loaderFiles, err := downloadSignedPackage(ctx, httpClient, loaderPackage, coffLoaderArchiveAsset, coffLoaderSignatureAsset, coffLoaderVersion, "sliverarmory/COFFLoader")
	if err != nil {
		return nil, fmt.Errorf("download COFFLoader: %w", err)
	}
	saEnvManifest, saEnvFiles, err := downloadSignedPackage(ctx, httpClient, saEnvPackage, saEnvArchiveAsset, saEnvSignatureAsset, situationalVersion, "sliverarmory/CS-Situational-Awareness-BOF")
	if err != nil {
		return nil, fmt.Errorf("download sa-env: %w", err)
	}
	saWhoamiManifest, saWhoamiFiles, err := downloadSignedPackage(ctx, httpClient, saWhoamiPackage, saWhoamiArchiveAsset, saWhoamiSignatureAsset, situationalVersion, "sliverarmory/CS-Situational-Awareness-BOF")
	if err != nil {
		return nil, fmt.Errorf("download sa-whoami: %w", err)
	}

	loaderCommand, err := exactCommand(loaderManifest, "coff-loader", "", "LoadAndRun")
	if err != nil {
		return nil, err
	}
	saEnvCommand, err := exactCommand(saEnvManifest, "sa-env", "coff-loader", "go")
	if err != nil {
		return nil, err
	}
	saWhoamiCommand, err := exactCommand(saWhoamiManifest, "sa-whoami", "coff-loader", "go")
	if err != nil {
		return nil, err
	}
	loaderFile, err := exactTargetFile(loaderCommand, "windows", targetArch, loaderFiles)
	if err != nil {
		return nil, fmt.Errorf("select COFFLoader target: %w", err)
	}
	bofFile, err := exactTargetFile(saEnvCommand, "windows", targetArch, saEnvFiles)
	if err != nil {
		return nil, fmt.Errorf("select sa-env target: %w", err)
	}
	saWhoamiFile, err := exactTargetFile(saWhoamiCommand, "windows", targetArch, saWhoamiFiles)
	if err != nil {
		return nil, fmt.Errorf("select sa-whoami target: %w", err)
	}
	loaderDigest := sha256.Sum256(loaderFile)
	return &armoryAssets{
		loaderID: hex.EncodeToString(loaderDigest[:]), loaderData: loaderFile, loaderCommand: loaderCommand,
		saEnv:    armoryBOF{data: bofFile, command: saEnvCommand},
		saWhoami: armoryBOF{data: saWhoamiFile, command: saWhoamiCommand},
	}, nil
}

func downloadPinned(ctx context.Context, client *http.Client, asset pinnedAsset) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt < armoryDownloadTries; attempt++ {
		data, retryable, err := downloadPinnedAttempt(ctx, client, asset)
		if err == nil {
			return data, nil
		}
		lastErr = err
		if !retryable || attempt == armoryDownloadTries-1 {
			return nil, err
		}

		timer := time.NewTimer(armoryRetryBaseDelay << attempt)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, fmt.Errorf("download %s: %w", asset.url, ctx.Err())
		case <-timer.C:
		}
	}
	return nil, lastErr
}

func downloadPinnedAttempt(ctx context.Context, client *http.Client, asset pinnedAsset) ([]byte, bool, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.url, http.NoBody)
	if err != nil {
		return nil, false, err
	}
	request.Header.Set("User-Agent", "sliver-comprehensive-e2e")
	response, err := client.Do(request)
	if err != nil {
		return nil, ctx.Err() == nil, fmt.Errorf("download %s: %w", asset.url, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4<<10))
		retryable := response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500 && response.StatusCode <= 599
		return nil, retryable, fmt.Errorf("download %s: HTTP %s", asset.url, response.Status)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, armoryDownloadLimit+1))
	if err != nil {
		return nil, ctx.Err() == nil, fmt.Errorf("download %s body: %w", asset.url, err)
	}
	if len(data) > armoryDownloadLimit {
		return nil, false, fmt.Errorf("download %s exceeded %d bytes", asset.url, armoryDownloadLimit)
	}
	digest := sha256.Sum256(data)
	actual := hex.EncodeToString(digest[:])
	if actual != asset.sha256 {
		return nil, false, fmt.Errorf("download %s SHA-256 %s, want %s", asset.url, actual, asset.sha256)
	}
	return data, false, nil
}

func verifySignature(publicKeyText string, data []byte, signature []byte) error {
	publicKey := minisign.PublicKey{}
	if err := publicKey.UnmarshalText([]byte(publicKeyText)); err != nil {
		return err
	}
	if !minisign.Verify(publicKey, data, signature) {
		return errors.New("minisign verification failed")
	}
	return nil
}

func findIndexedPackage(index *armory.ArmoryIndex, command string, publicKey string, repository string) (*armory.ArmoryPackage, error) {
	packages := append(append([]*armory.ArmoryPackage{}, index.Extensions...), index.Aliases...)
	var match *armory.ArmoryPackage
	for _, pkg := range packages {
		if pkg == nil || pkg.CommandName != command {
			continue
		}
		if match != nil {
			return nil, fmt.Errorf("signed Armory index contains duplicate %s package entries", command)
		}
		match = pkg
	}
	if match == nil {
		return nil, fmt.Errorf("signed Armory index does not contain %s", command)
	}
	if match.PublicKey != publicKey {
		return nil, fmt.Errorf("signed index %s public key mismatch", command)
	}
	if err := requireGitHubRepository(match.RepoURL, repository); err != nil {
		return nil, fmt.Errorf("signed index %s repository mismatch: %w", command, err)
	}
	return match, nil
}

func downloadSignedPackage(
	ctx context.Context,
	client *http.Client,
	pkg *armory.ArmoryPackage,
	archiveAsset pinnedAsset,
	signatureAsset pinnedAsset,
	expectedVersion string,
	expectedRepository string,
) (*extensions.ExtensionManifest, map[string][]byte, error) {
	archiveData, err := downloadPinned(ctx, client, archiveAsset)
	if err != nil {
		return nil, nil, err
	}
	signatureData, err := downloadPinned(ctx, client, signatureAsset)
	if err != nil {
		return nil, nil, err
	}
	if err := verifySignature(pkg.PublicKey, archiveData, signatureData); err != nil {
		return nil, nil, fmt.Errorf("verify %s archive: %w", pkg.CommandName, err)
	}
	signature := minisign.Signature{}
	if err := signature.UnmarshalText(signatureData); err != nil {
		return nil, nil, err
	}
	manifestData, err := base64.StdEncoding.DecodeString(signature.TrustedComment)
	if err != nil {
		return nil, nil, fmt.Errorf("decode %s signed manifest: %w", pkg.CommandName, err)
	}
	manifest, err := parseSignedExtensionManifest(manifestData)
	if err != nil {
		return nil, nil, fmt.Errorf("parse %s signed manifest: %w", pkg.CommandName, err)
	}
	if manifest.Version != expectedVersion {
		return nil, nil, fmt.Errorf("%s manifest version %q, want %q", pkg.CommandName, manifest.Version, expectedVersion)
	}
	if err := requireGitHubRepository(manifest.RepoURL, expectedRepository); err != nil {
		return nil, nil, fmt.Errorf("%s manifest repository mismatch: %w", pkg.CommandName, err)
	}
	files, err := readTarGzip(archiveData)
	if err != nil {
		return nil, nil, err
	}
	archiveManifest, err := archiveFileExact(files, extensions.ManifestFileName)
	if err != nil {
		return nil, nil, err
	}
	if !bytes.Equal(archiveManifest, manifestData) {
		return nil, nil, fmt.Errorf("%s trusted-comment manifest differs from archive manifest", pkg.CommandName)
	}
	return manifest, files, nil
}

func readTarGzip(data []byte) (map[string][]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	uncompressed, err := io.ReadAll(io.LimitReader(reader, armoryDownloadLimit+1))
	if err != nil {
		return nil, err
	}
	if len(uncompressed) > armoryDownloadLimit {
		return nil, errors.New("Armory archive exceeded decompression limit")
	}
	return readArmoryTar(uncompressed)
}

func exactCommand(manifest *extensions.ExtensionManifest, commandName string, dependency string, entrypoint string) (*extensions.ExtCommand, error) {
	var match *extensions.ExtCommand
	for _, command := range manifest.ExtCommand {
		if command == nil || command.CommandName != commandName {
			continue
		}
		if match != nil {
			return nil, fmt.Errorf("manifest contains duplicate command %s entries", commandName)
		}
		match = command
	}
	if match == nil {
		return nil, fmt.Errorf("manifest does not contain command %s", commandName)
	}
	if match.DependsOn != dependency || match.Entrypoint != entrypoint {
		return nil, fmt.Errorf("%s manifest routing mismatch: depends_on=%q entrypoint=%q", commandName, match.DependsOn, match.Entrypoint)
	}
	return match, nil
}

func exactTargetFile(command *extensions.ExtCommand, targetOS string, targetArch string, files map[string][]byte) ([]byte, error) {
	var declaredPath string
	matchCount := 0
	for _, file := range command.Files {
		if file == nil || file.OS != targetOS || file.Arch != targetArch {
			continue
		}
		matchCount++
		declaredPath = file.Path
	}
	if matchCount == 0 {
		return nil, fmt.Errorf("no exact artifact for %s/%s", targetOS, targetArch)
	}
	if matchCount > 1 {
		return nil, fmt.Errorf("manifest contains duplicate exact artifacts for %s/%s", targetOS, targetArch)
	}
	return archiveFileExact(files, declaredPath)
}

func validateSAEnvOutput(output []byte, expectedMarker string) error {
	if !bytes.Contains(output, []byte("Gathering Process Environment Variables")) || !bytes.Contains(output, []byte(expectedMarker)) {
		return errors.New("sa-env output failed stable content validation")
	}
	return nil
}

func validateSAWhoamiOutput(output []byte) error {
	for _, marker := range [][]byte{
		[]byte("UserName"),
		[]byte("SID"),
		[]byte("GROUP INFORMATION"),
		[]byte("Privilege Name"),
		[]byte("State"),
	} {
		if !bytes.Contains(output, marker) {
			return errors.New("sa-whoami output failed stable content validation")
		}
	}
	return nil
}

func requireGitHubRepository(actual string, expected string) error {
	actualRepository, err := canonicalGitHubRepository(actual)
	if err != nil {
		return fmt.Errorf("invalid GitHub repository: %w", err)
	}
	expectedRepository, err := canonicalGitHubRepository(expected)
	if err != nil {
		return fmt.Errorf("invalid expected GitHub repository: %w", err)
	}
	if actualRepository != expectedRepository {
		return fmt.Errorf("got %q, want %q", actualRepository, expectedRepository)
	}
	return nil
}

func canonicalGitHubRepository(repository string) (string, error) {
	if repository == "" || repository != strings.TrimSpace(repository) {
		return "", errors.New("repository must be non-empty and contain no surrounding whitespace")
	}

	repositoryPath := repository
	if strings.Contains(repository, "://") {
		parsed, err := url.Parse(repository)
		if err != nil {
			return "", err
		}
		if !strings.EqualFold(parsed.Scheme, "https") || !strings.EqualFold(parsed.Hostname(), "github.com") || parsed.Port() != "" {
			return "", errors.New("repository URL must use https://github.com without a port")
		}
		if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Opaque != "" {
			return "", errors.New("repository URL must not contain user information, a query, or a fragment")
		}
		if parsed.RawPath != "" || strings.Contains(parsed.EscapedPath(), "%") {
			return "", errors.New("repository URL must not contain escaped path bytes")
		}
		repositoryPath = strings.TrimPrefix(parsed.Path, "/")
	} else if strings.ContainsAny(repository, "\\:@?#") {
		return "", errors.New("repository shorthand must be owner/name")
	}

	repositoryPath = strings.TrimSuffix(repositoryPath, "/")
	parts := strings.Split(repositoryPath, "/")
	if len(parts) != 2 || !validGitHubOwner(parts[0]) {
		return "", errors.New("repository must contain exactly one valid owner and name")
	}
	name := parts[1]
	if len(name) > len(".git") && strings.EqualFold(name[len(name)-len(".git"):], ".git") {
		name = name[:len(name)-len(".git")]
	}
	if !validGitHubRepositoryName(name) {
		return "", errors.New("repository must contain exactly one valid owner and name")
	}
	return strings.ToLower(parts[0]) + "/" + strings.ToLower(name), nil
}

func validGitHubOwner(owner string) bool {
	if owner == "" || len(owner) > 39 || owner[0] == '-' || owner[len(owner)-1] == '-' {
		return false
	}
	for _, char := range owner {
		if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') && (char < '0' || char > '9') && char != '-' {
			return false
		}
	}
	return true
}

func validGitHubRepositoryName(name string) bool {
	if name == "" || name == "." || name == ".." || len(name) > 100 {
		return false
	}
	for _, char := range name {
		if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') && (char < '0' || char > '9') && !strings.ContainsRune("-._", char) {
			return false
		}
	}
	return true
}

type rawExtensionFile struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
	Path string `json:"path"`
}

type rawExtensionCommand struct {
	CommandName string             `json:"command_name"`
	Files       []rawExtensionFile `json:"files"`
}

type rawExtensionManifest struct {
	CommandName string                `json:"command_name"`
	Files       []rawExtensionFile    `json:"files"`
	Commands    []rawExtensionCommand `json:"commands"`
}

func parseSignedExtensionManifest(data []byte) (*extensions.ExtensionManifest, error) {
	manifest, err := extensions.ParseExtensionManifest(data)
	if err != nil {
		return nil, err
	}
	raw := rawExtensionManifest{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	rawCommands := raw.Commands
	if len(rawCommands) == 0 {
		rawCommands = []rawExtensionCommand{{CommandName: raw.CommandName, Files: raw.Files}}
	}
	if len(rawCommands) != len(manifest.ExtCommand) {
		return nil, errors.New("signed manifest command layout changed while parsing")
	}
	for commandIndex, rawCommand := range rawCommands {
		command := manifest.ExtCommand[commandIndex]
		if command == nil || command.CommandName != rawCommand.CommandName || len(command.Files) != len(rawCommand.Files) {
			return nil, errors.New("signed manifest command layout changed while parsing")
		}
		for fileIndex, rawFile := range rawCommand.Files {
			file := command.Files[fileIndex]
			if file == nil || !strings.EqualFold(file.OS, rawFile.OS) || !strings.EqualFold(file.Arch, rawFile.Arch) {
				return nil, errors.New("signed manifest target layout changed while parsing")
			}
			file.Path = rawFile.Path
		}
	}
	return manifest, nil
}

func readArmoryTar(data []byte) (map[string][]byte, error) {
	files := map[string][]byte{}
	reader := tar.NewReader(bytes.NewReader(data))
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return files, nil
		}
		if err != nil {
			return nil, err
		}
		if header.Typeflag == tar.TypeDir {
			continue
		}
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeRegA {
			return nil, fmt.Errorf("archive contains unsupported entry type %d at %q", header.Typeflag, header.Name)
		}
		name, err := canonicalArchivePath(header.Name)
		if err != nil {
			return nil, fmt.Errorf("archive contains invalid path %q: %w", header.Name, err)
		}
		if _, exists := files[name]; exists {
			return nil, fmt.Errorf("archive contains duplicate path %q", name)
		}
		content, err := io.ReadAll(reader)
		if err != nil {
			return nil, err
		}
		files[name] = content
	}
}

func canonicalArchivePath(name string) (string, error) {
	if strings.HasPrefix(name, "./") {
		name = strings.TrimPrefix(name, "./")
	}
	return canonicalDeclaredPath(name)
}

func canonicalDeclaredPath(name string) (string, error) {
	if name == "" || strings.ContainsRune(name, '\x00') || strings.Contains(name, "\\") {
		return "", errors.New("path must be a non-empty POSIX path")
	}
	if path.IsAbs(name) || name == ".." || strings.HasPrefix(name, "../") {
		return "", errors.New("path must be relative and remain within the archive")
	}
	clean := path.Clean(name)
	if clean != name || clean == "." {
		return "", errors.New("path must be canonical")
	}
	return clean, nil
}

func archiveFileExact(files map[string][]byte, declaredPath string) ([]byte, error) {
	name, err := canonicalDeclaredPath(declaredPath)
	if err != nil {
		return nil, fmt.Errorf("manifest artifact path %q is invalid: %w", declaredPath, err)
	}
	result, ok := files[name]
	if !ok {
		return nil, fmt.Errorf("archive does not contain exact manifest path %q", name)
	}
	return result, nil
}
