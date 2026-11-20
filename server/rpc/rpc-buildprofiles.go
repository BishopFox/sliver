package rpc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/buildcatalog"
	"github.com/bishopfox/sliver/server/buildcore"
	"github.com/bishopfox/sliver/server/buildplan"
	"github.com/bishopfox/sliver/server/buildprofile"
	"github.com/bishopfox/sliver/server/codenames"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/generate"
	"github.com/bishopfox/sliver/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type buildProfileStoreEntry struct {
	once  sync.Once
	store *buildprofile.Store
	err   error
}

var buildProfileStores sync.Map // absolute app root -> *buildProfileStoreEntry

func (rpc *Server) buildProfileStore() (*buildprofile.Store, error) {
	root, err := filepath.Abs(assets.GetRootAppDir())
	if err != nil {
		return nil, err
	}
	value, _ := buildProfileStores.LoadOrStore(root, &buildProfileStoreEntry{})
	entry := value.(*buildProfileStoreEntry)
	entry.once.Do(func() { entry.store, entry.err = buildprofile.OpenStore(root) })
	return entry.store, entry.err
}

// ListBuildProfiles lists the profiles visible to the server.
func (rpc *Server) ListBuildProfiles(ctx context.Context, _ *commonpb.Empty) (*clientpb.BuildProfileList, error) {
	store, failures, err := rpc.profileStoreWithBackfill(ctx)
	if err != nil {
		return nil, profileInternalError(err)
	}
	list := &clientpb.BuildProfileList{}
	documents, err := store.List()
	if err != nil {
		return nil, profileInternalError(err)
	}
	for _, document := range documents {
		list.Profiles = append(list.Profiles, profileDocumentProto(document))
	}
	for name, failure := range failures {
		list.Profiles = append(list.Profiles, migrationFailureDocument(name, failure))
	}
	sort.Slice(list.Profiles, func(i, j int) bool { return list.Profiles[i].Name < list.Profiles[j].Name })
	return list, nil
}

// GetBuildProfile retrieves a profile by reference.
func (rpc *Server) GetBuildProfile(ctx context.Context, ref *clientpb.BuildProfileRef) (*clientpb.BuildProfileDocument, error) {
	if ref == nil || !validProfileLookupName(ref.Name) {
		return nil, status.Error(codes.InvalidArgument, "profile name is required")
	}
	store, failures, err := rpc.profileStoreWithBackfill(ctx)
	if err != nil {
		return nil, profileInternalError(err)
	}
	document, err := store.Get(ref.Name)
	if errors.Is(err, buildprofile.ErrNotFound) {
		if failure := failures[ref.Name]; failure != nil {
			return migrationFailureDocument(ref.Name, failure), nil
		}
		return nil, status.Error(codes.NotFound, "build profile not found")
	}
	if err != nil {
		return nil, profileInternalError(err)
	}
	return profileDocumentProto(document), nil
}

// ValidateBuildProfile validates a submitted profile without storing it.
func (rpc *Server) ValidateBuildProfile(ctx context.Context, req *clientpb.BuildProfileImportRequest) (*clientpb.BuildProfileDocument, error) {
	profile, err := rpc.validateProfileImport(ctx, req)
	if err != nil {
		return nil, err
	}
	yaml, err := buildprofile.CanonicalYAML(profile)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return profileDocumentProto(buildprofile.Document{Profile: profile, YAML: yaml}), nil
}

// ImportBuildProfile validates and imports a submitted profile.
func (rpc *Server) ImportBuildProfile(ctx context.Context, req *clientpb.BuildProfileImportRequest) (*clientpb.BuildProfileDocument, error) {
	profile, err := rpc.validateProfileImport(ctx, req)
	if err != nil {
		return nil, err
	}
	store, failures, err := rpc.profileStoreWithBackfill(ctx)
	if err != nil {
		return nil, profileInternalError(err)
	}
	if failure := failures[profile.Metadata.Name]; failure != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "legacy profile %s cannot be replaced until its migration issue is resolved: %v", profile.Metadata.Name, failure)
	}
	document, err := store.Import(profile)
	if errors.Is(err, buildprofile.ErrRevision) {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}
	if err != nil {
		return nil, profileInternalError(err)
	}
	return profileDocumentProto(document), nil
}

// DeleteBuildProfile removes the selected profile revision.
//
//nolint:gocyclo // Revision lookup, store deletion, and legacy cleanup form one operation.
func (rpc *Server) DeleteBuildProfile(ctx context.Context, ref *clientpb.BuildProfileRef) (*commonpb.Empty, error) {
	if ref == nil || !validProfileLookupName(ref.Name) {
		return nil, status.Error(codes.InvalidArgument, "profile name is required")
	}
	store, failures, err := rpc.profileStoreWithBackfill(ctx)
	if err != nil {
		return nil, profileInternalError(err)
	}
	prior, lookupErr := store.Get(ref.Name)
	if errors.Is(lookupErr, buildprofile.ErrNotFound) && failures[ref.Name] == nil {
		return nil, status.Error(codes.NotFound, "build profile not found")
	}
	if lookupErr != nil && !errors.Is(lookupErr, buildprofile.ErrNotFound) {
		return nil, profileInternalError(lookupErr)
	}
	if err := store.Delete(ref.Name); err != nil {
		return nil, profileInternalError(err)
	}
	if _, err := db.ImplantProfileByName(ref.Name); err == nil {
		if err := db.DeleteProfile(ref.Name); err != nil {
			if lookupErr == nil {
				if restoreErr := store.Restore(ref.Name, &prior); restoreErr != nil {
					return nil, profileInternalError(fmt.Errorf("delete legacy row: %w; restore profile: %v", err, restoreErr))
				}
			} else if restoreErr := store.Restore(ref.Name, nil); restoreErr != nil {
				return nil, profileInternalError(fmt.Errorf("delete legacy row: %w; restore tombstone: %v", err, restoreErr))
			}
			return nil, profileInternalError(err)
		}
	} else if !errors.Is(err, db.ErrRecordNotFound) {
		if lookupErr == nil {
			if restoreErr := store.Restore(ref.Name, &prior); restoreErr != nil {
				return nil, profileInternalError(fmt.Errorf("look up legacy row: %w; restore profile: %v", err, restoreErr))
			}
		} else if restoreErr := store.Restore(ref.Name, nil); restoreErr != nil {
			return nil, profileInternalError(fmt.Errorf("look up legacy row: %w; restore tombstone: %v", err, restoreErr))
		}
		return nil, profileInternalError(err)
	}
	return &commonpb.Empty{}, nil
}

// ExecuteProfileBuild executes a stored profile with the requested inputs.
func (rpc *Server) ExecuteProfileBuild(ctx context.Context, req *clientpb.BuildProfileExecuteRequest) (*clientpb.BuildRecord, error) {
	if req == nil || !validProfileLookupName(req.Name) {
		return nil, status.Error(codes.InvalidArgument, "profile name is required")
	}
	if req.BuildName != "" {
		if len(req.BuildName) > 128 || util.AllowedName(req.BuildName) != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid build name")
		}
	}
	store, failures, err := rpc.profileStoreWithBackfill(ctx)
	if err != nil {
		return nil, profileInternalError(err)
	}
	document, err := store.Get(req.Name)
	if errors.Is(err, buildprofile.ErrNotFound) {
		if failure := failures[req.Name]; failure != nil {
			return nil, status.Errorf(codes.FailedPrecondition, "legacy profile cannot be represented as BuildProfile YAML: %v", failure)
		}
		return nil, status.Error(codes.NotFound, "build profile not found")
	}
	if err != nil {
		return nil, profileInternalError(err)
	}
	if document.Migrated {
		return rpc.executeMigratedProfile(ctx, document, req)
	}
	return rpc.executeAuthoredProfile(ctx, document, req)
}

func validProfileLookupName(name string) bool {
	return len(name) > 0 && len(name) <= 255 && strings.TrimSpace(name) != "" &&
		name != "." && name != ".." && !strings.ContainsAny(name, "/\\\x00\r\n")
}

func (rpc *Server) executeAuthoredProfile(ctx context.Context, document buildprofile.Document, req *clientpb.BuildProfileExecuteRequest) (*clientpb.BuildRecord, error) {
	name := req.BuildName
	if name == "" {
		name = document.Profile.Metadata.Name // Placeholder until all validation succeeds.
	}
	neutral := document.Profile.Request(name)
	neutral.ProfileSnapshot = profileSnapshot(document)
	if req.Provider != nil {
		neutral.Provider = providerFromProto(req.Provider)
	}
	if err := applyEncoderOverride(&neutral, req.ShellcodeEncoder); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := validateProfileInputs(document.Profile, req.Inputs); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	resolveInput, releaseInputs, err := buildInputResolver(req.Inputs)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	defer releaseInputs()
	if req.BuildName == "" {
		neutral.Name, err = codenames.GetCodename()
		if err != nil {
			return nil, profileInternalError(err)
		}
	}
	plan, err := rpc.resolveBuildPlan(ctx, neutral)
	if err != nil {
		return nil, buildPlanRPCError(err)
	}
	var adapter buildcore.Adapter
	if plan.Steps[0].Provider == "server" {
		adapter, err = rpc.coreAdapterForPlan(plan)
		if err != nil {
			return nil, buildPlanRPCError(err)
		}
	}
	_, record, err := rpc.executeResolvedPlan(ctx, plan, adapter, nil, resolveInput)
	if err != nil {
		if record.ID != "" {
			return buildRecordProto(record), nil
		}
		return nil, buildPlanRPCError(err)
	}
	return buildRecordProto(record), nil
}

func (rpc *Server) executeMigratedProfile(ctx context.Context, document buildprofile.Document, req *clientpb.BuildProfileExecuteRequest) (*clientpb.BuildRecord, error) {
	if len(req.Inputs) != 0 {
		return nil, status.Error(codes.InvalidArgument, "migrated legacy profiles do not accept per-build inputs")
	}
	if req.Provider != nil && req.Provider.Mode != "automatic" && (req.Provider.Mode != "pinned" || req.Provider.Builder != "server") {
		return nil, status.Error(codes.FailedPrecondition, "migrated legacy profile requires the server builder; import authored YAML to change provider")
	}
	config := &clientpb.ImplantConfig{}
	if len(document.LegacyConfig) == 0 {
		return nil, status.Error(codes.FailedPrecondition, "migrated profile lacks frozen legacy configuration")
	}
	if err := proto.Unmarshal(document.LegacyConfig, config); err != nil {
		return nil, profileInternalError(err)
	}
	config.ID = "" // Never reload a concurrently edited DB config after choosing the snapshot.
	if req.ShellcodeEncoder != "" {
		if document.Profile.FinalOutput.Format != "shellcode" {
			return nil, status.Error(codes.InvalidArgument, "shellcode encoder override requires shellcode final output")
		}
		encoder, ok := map[string]clientpb.ShellcodeEncoder{"none": clientpb.ShellcodeEncoder_NONE,
			"xor": clientpb.ShellcodeEncoder_XOR, "xor-dynamic": clientpb.ShellcodeEncoder_XOR_DYNAMIC,
			"shikata-ga-nai": clientpb.ShellcodeEncoder_SHIKATA_GA_NAI}[req.ShellcodeEncoder]
		if !ok {
			return nil, status.Error(codes.InvalidArgument, "unknown shellcode encoder")
		}
		config.ShellcodeEncoder = encoder
		config.SGNEnabled = encoder == clientpb.ShellcodeEncoder_SHIKATA_GA_NAI
	}
	snapshot := profileSnapshot(document)
	snapshot.EncoderOverride = req.ShellcodeEncoder
	ctx = context.WithValue(ctx, profileSnapshotContextKey{}, snapshot)
	generated, err := rpc.Generate(ctx, &clientpb.GenerateReq{Name: req.BuildName, Config: config})
	if err != nil {
		return nil, err
	}
	return rpc.GetBuildRecord(ctx, &clientpb.BuildRecordRef{ID: generated.ImplantBuildID})
}

type profileSnapshotContextKey struct{}

func profileSnapshot(document buildprofile.Document) *buildplan.ProfileSnapshot {
	hash := sha256.Sum256(document.YAML)
	snapshot := &buildplan.ProfileSnapshot{Name: document.Profile.Metadata.Name, Revision: document.Profile.Metadata.Revision,
		YAML: append([]byte(nil), document.YAML...), SHA256: hex.EncodeToString(hash[:]), Migrated: document.Migrated}
	if document.Migrated {
		legacyHash := sha256.Sum256(document.LegacyConfig)
		snapshot.LegacyConfigSHA256 = hex.EncodeToString(legacyHash[:])
	}
	return snapshot
}

func profileDocumentProto(document buildprofile.Document) *clientpb.BuildProfileDocument {
	return &clientpb.BuildProfileDocument{Name: document.Profile.Metadata.Name, Revision: document.Profile.Metadata.Revision,
		YAML: string(document.YAML), Migrated: document.Migrated,
		FinalArtifact: buildArtifactProto(document.Profile.FinalOutput.Artifact.PlanArtifact(document.Profile.Target))}
}

func profileInternalError(err error) error {
	return status.Errorf(codes.Internal, "build profile: %v", err)
}

func (rpc *Server) profileStoreWithBackfill(ctx context.Context) (*buildprofile.Store, map[string]error, error) {
	store, err := rpc.buildProfileStore()
	if err != nil {
		return nil, nil, err
	}
	legacy, err := db.ImplantProfiles()
	if err != nil {
		return nil, nil, err
	}
	failures := map[string]error{}
	for _, item := range legacy {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		if _, err := store.Get(item.Name); err == nil {
			continue
		} else if !errors.Is(err, buildprofile.ErrNotFound) {
			return nil, nil, err
		}
		if store.IsTombstoned(item.Name) {
			continue
		}
		profile, err := rpc.legacyProfileDocument(ctx, item)
		if err != nil {
			failures[item.Name] = err
			continue
		}
		config, err := legacyConfigSnapshot(item.Config)
		if err != nil {
			failures[item.Name] = err
			continue
		}
		if _, err := store.UpsertLegacy(profile, item.Config.ID, config, false); err != nil && !errors.Is(err, buildprofile.ErrNotFound) {
			return nil, nil, err
		}
	}
	return store, failures, nil
}

func (rpc *Server) upsertLegacyProfileDocument(ctx context.Context, legacy *clientpb.ImplantProfile) error {
	profile, err := rpc.legacyProfileDocument(ctx, legacy)
	if err != nil {
		return err
	}
	store, err := rpc.buildProfileStore()
	if err != nil {
		return err
	}
	config, err := legacyConfigSnapshot(legacy.Config)
	if err != nil {
		return err
	}
	document, err := store.UpsertLegacy(profile, legacy.Config.ID, config, true)
	if err != nil {
		return err
	}
	if !document.Migrated {
		return fmt.Errorf("authored BuildProfile YAML %q remains authoritative; legacy profile was saved separately", legacy.Name)
	}
	return nil
}

func migrationFailureDocument(name string, failure error) *clientpb.BuildProfileDocument {
	return &clientpb.BuildProfileDocument{Name: name, Migrated: true, MigrationError: failure.Error()}
}

const maxLegacyConfigSnapshotBytes = 64 << 20

func legacyConfigSnapshot(config *clientpb.ImplantConfig) ([]byte, error) {
	if config == nil {
		return nil, errors.New("missing legacy implant config")
	}
	snapshot := proto.Clone(config).(*clientpb.ImplantConfig)
	snapshot.ImplantBuilds = nil // These contain generated private keys and are never profile inputs.
	encoded, err := proto.Marshal(snapshot)
	if err != nil {
		return nil, err
	}
	if len(encoded) > maxLegacyConfigSnapshotBytes {
		return nil, fmt.Errorf("legacy profile config exceeds %d bytes", maxLegacyConfigSnapshotBytes)
	}
	return encoded, nil
}

//nolint:gocyclo // Legacy fields and their pinned references are migrated together.
func (rpc *Server) legacyProfileDocument(ctx context.Context, legacy *clientpb.ImplantProfile) (buildprofile.Profile, error) {
	if legacy == nil || legacy.Config == nil {
		return buildprofile.Profile{}, errors.New("legacy profile has no implant config")
	}
	config := proto.Clone(legacy.Config).(*clientpb.ImplantConfig)
	if err := (generate.SliverCoreAdapter{}).Normalize(config); err != nil {
		return buildprofile.Profile{}, err
	}
	strategy, err := legacyConnectionStrategy(config.ConnectionStrategy)
	if err != nil {
		return buildprofile.Profile{}, err
	}
	reconnect, err := wholeSeconds(config.ReconnectInterval)
	if err != nil {
		return buildprofile.Profile{}, fmt.Errorf("reconnect interval: %w", err)
	}
	poll, err := wholeSeconds(config.PollTimeout)
	if err != nil {
		return buildprofile.Profile{}, fmt.Errorf("poll timeout: %w", err)
	}
	request, err := legacyBuildRequest(legacy.Name, config)
	if err != nil {
		return buildprofile.Profile{}, err
	}
	request.C2Settings = &buildplan.C2Settings{
		ConnectionStrategy: strategy,
		ReconnectSeconds:   reconnect, PollTimeoutSeconds: poll,
		MaxConnectionErrors: config.MaxConnectionErrors, HTTPC2Profile: config.HTTPC2ConfigName,
		WireguardKeyExchangePort: config.WGKeyExchangePort, WireguardTCPCommsPort: config.WGTcpCommsPort,
	}
	if request.C2Settings.HTTPC2Profile == "" {
		request.C2Settings.HTTPC2Profile = "default"
	}
	if config.IsBeacon {
		interval, err := wholeSeconds(config.BeaconInterval)
		if err != nil {
			return buildprofile.Profile{}, fmt.Errorf("beacon interval: %w", err)
		}
		jitter, err := wholeSeconds(config.BeaconJitter)
		if err != nil {
			return buildprofile.Profile{}, fmt.Errorf("beacon jitter: %w", err)
		}
		request.BeaconSettings = &buildplan.BeaconSettings{IntervalSeconds: interval, JitterSeconds: jitter}
	}
	catalog, err := rpc.buildCatalog()
	if err != nil {
		return buildprofile.Profile{}, err
	}
	core, err := catalog.Get(buildcatalog.Reference{Name: request.Core.Name})
	if err != nil {
		return buildprofile.Profile{}, err
	}
	request.Core = buildplan.Reference{Name: core.Reference.Name, Version: core.Reference.Version, Digest: core.Reference.Digest}
	for index := range request.Transforms {
		ref, err := catalog.Get(buildcatalog.Reference{Name: request.Transforms[index].Spec.Name})
		if err != nil {
			return buildprofile.Profile{}, err
		}
		request.Transforms[index].Spec = buildplan.Reference{Name: ref.Reference.Name, Version: ref.Reference.Version, Digest: ref.Reference.Digest}
	}
	plan, err := staticProfilePlan(ctx, catalog, request)
	if err != nil {
		return buildprofile.Profile{}, err
	}
	profile := buildprofile.Profile{
		APIVersion: "sliver.bishopfox.com/v1alpha1", Kind: "BuildProfile",
		Metadata: buildprofile.Metadata{Name: legacy.Name, Revision: 1},
		Target:   request.Target, Mode: request.Mode,
		C2: buildprofile.C2{Endpoints: request.C2, IncludeProtocols: request.IncludeProtocols,
			ConnectionStrategy: request.C2Settings.ConnectionStrategy, ReconnectSeconds: request.C2Settings.ReconnectSeconds,
			PollTimeoutSeconds: request.C2Settings.PollTimeoutSeconds, MaxConnectionErrors: request.C2Settings.MaxConnectionErrors,
			HTTPC2Profile:            request.C2Settings.HTTPC2Profile,
			WireguardKeyExchangePort: config.WGKeyExchangePort, WireguardTCPCommsPort: config.WGTcpCommsPort},
		Beacon: request.BeaconSettings, Core: request.Core,
		CoreOutput: buildprofile.ArtifactFromPlan(plan.Steps[0].Output), CoreOptions: request.CoreOptions,
		Transforms: request.Transforms, FinalOutput: buildprofile.FinalOutput{Format: request.FinalFormat,
			Artifact: buildprofile.ArtifactFromPlan(plan.FinalArtifact)},
		Provider: buildprofile.Provider{Policy: "pinned", Builder: "server"},
	}
	if profile.CoreOptions == nil {
		profile.CoreOptions = []buildplan.Option{}
	}
	if profile.Transforms == nil {
		profile.Transforms = []buildplan.TransformRequest{}
	}
	return profile, profile.Validate()
}

func wholeSeconds(nanoseconds int64) (uint64, error) {
	if nanoseconds < 0 || nanoseconds%int64(time.Second) != 0 {
		return 0, errors.New("must be a nonnegative whole number of seconds")
	}
	return uint64(nanoseconds / int64(time.Second)), nil
}

func legacyConnectionStrategy(value string) (string, error) {
	switch value {
	case "":
		return "sequential", nil
	case "r":
		return "random", nil
	case "rd":
		return "random-domain", nil
	default:
		return "", fmt.Errorf("unknown legacy C2 connection strategy %q", value)
	}
}

type staticProfileCatalog struct {
	base     *buildcatalog.Catalog
	provider string
}

func (catalog staticProfileCatalog) Get(ref buildcatalog.Reference) (buildcatalog.Snapshot, error) {
	snapshot, err := catalog.base.Get(ref)
	return catalog.available(snapshot), err
}

func (catalog staticProfileCatalog) List(query buildcatalog.Query) []buildcatalog.Snapshot {
	query.AvailableOnly = false // Validate topology independently from current provider state.
	list := catalog.base.List(query)
	for index := range list {
		list[index] = catalog.available(list[index])
	}
	return list
}

func (catalog staticProfileCatalog) available(snapshot buildcatalog.Snapshot) buildcatalog.Snapshot {
	snapshot.Available = true
	snapshot.Providers = []buildcatalog.ProviderStatus{{Name: catalog.provider, Online: true}}
	return snapshot
}

func staticProfilePlan(ctx context.Context, catalog *buildcatalog.Catalog, request buildplan.Request) (buildplan.Plan, error) {
	provider := "server"
	if request.Provider.Mode == "pinned" {
		provider = request.Provider.Builder
	}
	static := staticProfileCatalog{base: catalog, provider: provider}
	return buildplan.NewPlanner(static, func(string, buildcatalog.Snapshot, buildplan.Target, buildplan.Toolchain) bool { return true }).Plan(ctx, request)
}

func (rpc *Server) validateProfileImport(ctx context.Context, req *clientpb.BuildProfileImportRequest) (buildprofile.Profile, error) {
	if req == nil {
		return buildprofile.Profile{}, status.Error(codes.InvalidArgument, "profile YAML is required")
	}
	profile, err := buildprofile.Parse([]byte(req.YAML))
	if err != nil {
		return buildprofile.Profile{}, status.Error(codes.InvalidArgument, err.Error())
	}
	catalog, err := rpc.buildCatalog()
	if err != nil {
		return buildprofile.Profile{}, profileInternalError(err)
	}
	plan, err := staticProfilePlan(ctx, catalog, profile.Request(profile.Metadata.Name))
	if err != nil {
		return buildprofile.Profile{}, buildPlanRPCError(err)
	}
	declared := profile.FinalOutput.Artifact.PlanArtifact(profile.Target)
	if !sameArtifact(declared, plan.FinalArtifact) {
		return buildprofile.Profile{}, status.Error(codes.FailedPrecondition, "declared final artifact differs from resolved pipeline")
	}
	return profile, nil
}

func sameArtifact(a, b buildplan.Artifact) bool {
	return a.Kind == b.Kind && a.Container == b.Container && a.Target == b.Target &&
		a.GoBuildMode == b.GoBuildMode && strings.Join(a.EntrySymbols, "\x00") == strings.Join(b.EntrySymbols, "\x00")
}

// The compatibility executor resolves current catalog names. Refuse a run if
// the selected immutable definitions no longer match the migrated snapshot.

//nolint:gocyclo // All compatibility checks compare one snapshot against the same frozen plan.
func verifyMigratedProfileSnapshot(snapshot *buildplan.ProfileSnapshot, plan buildplan.Plan) error {
	profile, err := buildprofile.Parse(snapshot.YAML)
	if err != nil {
		return fmt.Errorf("invalid frozen migrated profile: %w", err)
	}
	if len(plan.Steps) == 0 || !sameReference(profile.Core, plan.Steps[0].Spec) {
		return errors.New("migrated profile core definition changed; re-save the profile")
	}
	if !sameArtifact(profile.FinalOutput.Artifact.PlanArtifact(profile.Target), plan.FinalArtifact) {
		return errors.New("migrated profile final artifact changed; re-save the profile")
	}
	expected := make([]buildplan.Reference, 0, len(profile.Transforms)+1)
	for _, transform := range profile.Transforms {
		if snapshot.EncoderOverride != "" && isEncoderTransform(transform.Spec.Name) {
			continue
		}
		expected = append(expected, transform.Spec)
	}
	if snapshot.EncoderOverride != "" && snapshot.EncoderOverride != "none" {
		encoder := map[string]string{"xor": "builtin/xor", "xor-dynamic": "builtin/xor-dynamic",
			"shikata-ga-nai": "builtin/shikata-ga-nai"}[snapshot.EncoderOverride]
		if encoder == "" {
			return fmt.Errorf("unknown frozen shellcode encoder override %q", snapshot.EncoderOverride)
		}
		expected = append(expected, buildplan.Reference{Name: encoder})
	}
	if len(plan.Steps) != len(expected)+1 {
		return errors.New("migrated profile transform sequence changed; re-save the profile")
	}
	for index, transform := range expected {
		resolved := plan.Steps[index+1]
		if resolved.Kind != "transform" || resolved.Spec.Name != transform.Name ||
			transform.Version != "" && !sameReference(transform, resolved.Spec) {
			return fmt.Errorf("migrated profile transform %s definition or order changed; re-save the profile", transform.Name)
		}
	}
	return nil
}

func sameReference(a, b buildplan.Reference) bool {
	return a.Name == b.Name && a.Version == b.Version && a.Digest == b.Digest
}

func isEncoderTransform(name string) bool {
	return name == "builtin/xor" || name == "builtin/xor-dynamic" || name == "builtin/shikata-ga-nai"
}

func applyEncoderOverride(request *buildplan.Request, encoder string) error {
	if encoder == "" {
		return nil
	}
	if request.FinalFormat != "shellcode" && request.FinalFormat != "stage" {
		return errors.New("shellcode encoder override requires shellcode or stage final output")
	}
	name, ok := map[string]string{"none": "", "xor": "builtin/xor", "xor-dynamic": "builtin/xor-dynamic",
		"shikata-ga-nai": "builtin/shikata-ga-nai"}[encoder]
	if !ok {
		return fmt.Errorf("unknown shellcode encoder %q", encoder)
	}
	transforms := make([]buildplan.TransformRequest, 0, len(request.Transforms)+1)
	inserted := false
	for _, transform := range request.Transforms {
		if transform.Spec.Name == "builtin/xor" || transform.Spec.Name == "builtin/xor-dynamic" || transform.Spec.Name == "builtin/shikata-ga-nai" {
			continue
		}
		if !inserted && name != "" && request.FinalFormat == "stage" && strings.HasPrefix(transform.Spec.Name, "builtin/stage-") {
			transforms = append(transforms, buildplan.TransformRequest{Spec: buildplan.Reference{Name: name}})
			inserted = true
		}
		transforms = append(transforms, transform)
	}
	if name != "" && !inserted {
		transforms = append(transforms, buildplan.TransformRequest{Spec: buildplan.Reference{Name: name}})
	}
	request.Transforms = transforms
	return nil
}

func validateProfileInputs(profile buildprofile.Profile, inputs []*clientpb.BuildInput) error {
	declared := make(map[string]string, len(profile.PerBuildInputs))
	for _, input := range profile.PerBuildInputs {
		declared[input.Name] = input.Kind
	}
	seen := map[string]bool{}
	for _, input := range inputs {
		if input == nil || seen[input.Name] || declared[input.Name] == "" || declared[input.Name] != input.Kind {
			return errors.New("input must match a unique declared per-build slot")
		}
		seen[input.Name] = true
	}
	for name := range declared {
		if !seen[name] {
			return fmt.Errorf("required per-build input %s is missing", name)
		}
	}
	return nil
}
