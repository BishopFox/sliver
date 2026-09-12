package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

const (
	maxCrackstationSelectorBytes   = 256
	maxStandaloneLookupBytes       = 4 << 10
	maxStandaloneIdentifyHashes    = 10_000
	maxStandaloneIdentifyHashBytes = 8 << 20
)

var (
	errNoIdleCrackstation              = errors.New("no idle crackstations are available")
	errCrackstationEventQueueFull      = errors.New("selected crackstation event queue is full")
	errCrackstationSelectorNotFound    = errors.New("requested crackstation is not connected")
	errCrackstationSelectorAmbiguous   = errors.New("requested crackstation name is ambiguous")
	errSelectedCrackstationUnavailable = errors.New("requested crackstation is busy")
	errSelectedCrackstationUnsupported = fmt.Errorf("requested crackstation does not advertise capability %q", clientpb.CrackstationCapabilityCrackQueryV1)

	// standaloneCrackKeyspaceTasks contains every transient synchronous query.
	// The historical name is retained because queue and station lifecycle code
	// already calls these helpers. These tasks deliberately do not live in
	// crack_tasks: that table belongs to a CrackJob through crack_job_id.
	// crackQueueMu guards this map so reservations are atomic with scheduling.
	standaloneCrackKeyspaceTasks = map[string]*standaloneCrackKeyspaceTask{}
)

// standaloneCrackKeyspaceTask is the lifecycle record for any synchronous
// crack query. Its historical name is retained for compatibility with the
// existing queue integration.
type standaloneCrackKeyspaceTask struct {
	task            *clientpb.CrackTask
	station         *core.Crackstation
	mode            clientpb.CrackQueryMode
	stationName     string
	hashcatVersion  string
	done            chan struct{}
	result          *clientpb.CrackTask
	resultValue     string
	resultStderr    string
	resultErr       error
	cancelRequested bool
	fetched         bool
}

type standaloneCrackQueryResult struct {
	Mode                 clientpb.CrackQueryMode
	CrackstationHostUUID string
	CrackstationName     string
	HashcatVersion       string
	Value                string
	Stderr               string
}

func isStandaloneCrackQueryCommand(command *models.CrackCommand) bool {
	if command == nil {
		return false
	}
	return command.Keyspace || command.TotalCandidates || command.Lookup != "" || command.IdentifyMode || command.HashInfo || command.HashInfoLevel != 0
}

func standaloneCrackQueryMode(command *models.CrackCommand) (clientpb.CrackQueryMode, error) {
	if command == nil {
		return clientpb.CrackQueryMode_CRACK_QUERY_UNSPECIFIED, errors.New("missing crack command")
	}
	modes := make([]clientpb.CrackQueryMode, 0, 2)
	if command.Keyspace {
		modes = append(modes, clientpb.CrackQueryMode_CRACK_QUERY_KEYSPACE)
	}
	if command.TotalCandidates {
		modes = append(modes, clientpb.CrackQueryMode_CRACK_QUERY_TOTAL_CANDIDATES)
	}
	if command.Lookup != "" {
		modes = append(modes, clientpb.CrackQueryMode_CRACK_QUERY_LOOKUP)
	}
	if command.IdentifyMode {
		modes = append(modes, clientpb.CrackQueryMode_CRACK_QUERY_IDENTIFY)
	}
	if command.HashInfo || command.HashInfoLevel != 0 {
		modes = append(modes, clientpb.CrackQueryMode_CRACK_QUERY_HASH_INFO)
	}
	if len(modes) == 0 {
		return clientpb.CrackQueryMode_CRACK_QUERY_UNSPECIFIED, errors.New("no synchronous query mode is enabled")
	}
	if len(modes) != 1 {
		return clientpb.CrackQueryMode_CRACK_QUERY_UNSPECIFIED, errors.New("synchronous query modes are mutually exclusive")
	}
	return modes[0], nil
}

// runStandaloneCrackQuery validates and resolves a query command, reserves an
// explicitly idle crackstation, and waits for the existing worker task
// protocol. It never creates a CrackJob or durable CrackTask.
func runStandaloneCrackQuery(ctx context.Context, command *models.CrackCommand, selector string) (*standaloneCrackQueryResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}
	selector = strings.TrimSpace(selector)
	if err := validateStandaloneCrackstationSelector(selector); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid crackstation selector: %s", err)
	}
	mode, err := standaloneCrackQueryMode(command)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid crack query: %s", err)
	}
	if err := prepareStandaloneCrackQuery(command, mode); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid crack query: %s", err)
	}

	taskID := models.NewUUID()
	normalizeDistributedCrackCommand(command, taskID)
	command.Quiet = true
	command.CredentialIDs = nil
	command.CredentialCollection = ""
	command.IncludeCrackedCredentials = false
	if mode != clientpb.CrackQueryMode_CRACK_QUERY_IDENTIFY {
		command.Hashes = nil
	}
	if mode != clientpb.CrackQueryMode_CRACK_QUERY_LOOKUP {
		command.Skip = 0
		command.Limit = 0
	}

	// Resolve managed names while deletion is excluded, then register the
	// in-memory reference before releasing the lifecycle lock. CrackFileDelete
	// consults activeStandaloneCrackKeyspaceReferencesManagedFile under this
	// same lock, so a resolved file cannot disappear during worker sync.
	crackFileLifecycleMu.Lock()
	if err := resolveManagedCrackFiles(db.Session().WithContext(ctx), command); err != nil {
		crackFileLifecycleMu.Unlock()
		return nil, status.Errorf(codes.InvalidArgument, "invalid crack query: %s", err)
	}
	if err := validateDistributedCrackFileFields(command); err != nil {
		crackFileLifecycleMu.Unlock()
		return nil, status.Errorf(codes.InvalidArgument, "invalid crack query: %s", err)
	}
	if err := validateStandaloneCrackQueryOperands(command, mode); err != nil {
		crackFileLifecycleMu.Unlock()
		return nil, status.Errorf(codes.InvalidArgument, "invalid crack query: %s", err)
	}

	entry, err := reserveStandaloneCrackQueryTask(ctx, taskID, command, mode, selector)
	crackFileLifecycleMu.Unlock()
	if err != nil {
		return nil, standaloneCrackQueryDispatchError(err)
	}
	crackQueueReaperStarter()

	select {
	case <-entry.done:
		if entry.resultErr != nil {
			return nil, entry.resultErr
		}
		return &standaloneCrackQueryResult{
			Mode:                 entry.mode,
			CrackstationHostUUID: entry.task.HostUUID,
			CrackstationName:     entry.stationName,
			HashcatVersion:       entry.hashcatVersion,
			Value:                entry.resultValue,
			Stderr:               entry.resultStderr,
		}, nil
	case <-ctx.Done():
		requestStandaloneCrackKeyspaceCancellation(taskID.String(), entry)
		return nil, status.FromContextError(ctx.Err()).Err()
	}
}

// runStandaloneCrackKeyspace preserves the original internal API for focused
// lifecycle tests and older call sites.
func runStandaloneCrackKeyspace(ctx context.Context, command *models.CrackCommand) (string, error) {
	result, err := runStandaloneCrackQuery(ctx, command, "")
	if err != nil {
		return "", err
	}
	if result.Mode != clientpb.CrackQueryMode_CRACK_QUERY_KEYSPACE {
		return "", status.Error(codes.Internal, "standalone keyspace query returned the wrong mode")
	}
	return result.Value, nil
}

func validateStandaloneCrackstationSelector(selector string) error {
	if len(selector) > maxCrackstationSelectorBytes {
		return fmt.Errorf("selector exceeds %d bytes", maxCrackstationSelectorBytes)
	}
	if strings.ContainsAny(selector, "\x00\r\n") {
		return errors.New("selector cannot contain a line break or NUL")
	}
	if !utf8.ValidString(selector) {
		return errors.New("selector is not valid UTF-8")
	}
	return nil
}

func standaloneCrackQueryDispatchError(err error) error {
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return status.FromContextError(err).Err()
	case errors.Is(err, errCrackstationSelectorNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, errCrackstationSelectorAmbiguous):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, errSelectedCrackstationUnsupported):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, errSelectedCrackstationUnavailable), errors.Is(err, errNoIdleCrackstation), errors.Is(err, errCrackstationEventQueueFull):
		return status.Error(codes.ResourceExhausted, err.Error())
	case status.Code(err) != codes.Unknown:
		return err
	default:
		return status.Errorf(codes.Internal, "failed to dispatch crack query: %s", err)
	}
}

//nolint:gocyclo // Keep the mode-specific Hashcat compatibility checks in one auditable validation boundary.
func prepareStandaloneCrackQuery(command *models.CrackCommand, mode clientpb.CrackQueryMode) error {
	if command == nil {
		return errors.New("missing crack command")
	}

	switch mode {
	case clientpb.CrackQueryMode_CRACK_QUERY_KEYSPACE, clientpb.CrackQueryMode_CRACK_QUERY_TOTAL_CANDIDATES, clientpb.CrackQueryMode_CRACK_QUERY_LOOKUP:
		if len(command.Hashes) != 0 || len(command.CredentialIDs) != 0 || command.CredentialCollection != "" || command.IncludeCrackedCredentials {
			return errors.New("candidate queries do not accept hashes or credential selectors")
		}
		if mode != clientpb.CrackQueryMode_CRACK_QUERY_LOOKUP && (command.Skip != 0 || command.Limit != 0) {
			return errors.New("keyspace and total-candidates queries cannot be combined with skip or limit")
		}
		if mode == clientpb.CrackQueryMode_CRACK_QUERY_LOOKUP {
			if len(command.Lookup) > maxStandaloneLookupBytes {
				return fmt.Errorf("lookup value exceeds %d bytes", maxStandaloneLookupBytes)
			}
			if strings.ContainsAny(command.Lookup, "\x00\r\n") {
				return errors.New("lookup value cannot contain a line break or NUL")
			}
			if !utf8.ValidString(command.Lookup) {
				return errors.New("lookup value is not valid UTF-8")
			}
		}
	case clientpb.CrackQueryMode_CRACK_QUERY_IDENTIFY:
		if command.HashMode != nil || command.HashType != int32(clientpb.HashType_INVALID) {
			return errors.New("identify queries cannot specify a hash mode or hash type")
		}
		if len(command.CredentialIDs) != 0 || command.CredentialCollection != "" || command.IncludeCrackedCredentials {
			return errors.New("identify queries accept inline hashes, not credential selectors")
		}
		if len(command.PositionalArguments) != 0 || command.Identify != "" {
			return errors.New("identify queries accept hashes, not attack inputs")
		}
		if command.Skip != 0 || command.Limit != 0 {
			return errors.New("identify queries cannot be combined with skip or limit")
		}
		if err := validateIdentifyQueryFields(command); err != nil {
			return err
		}
		if err := validateStandaloneIdentifyHashes(command.Hashes); err != nil {
			return err
		}
	case clientpb.CrackQueryMode_CRACK_QUERY_HASH_INFO:
		if len(command.Hashes) != 0 || len(command.CredentialIDs) != 0 || command.CredentialCollection != "" || command.IncludeCrackedCredentials {
			return errors.New("hash-info queries do not accept hashes or credential selectors")
		}
		if len(command.PositionalArguments) != 0 || command.Identify != "" {
			return errors.New("hash-info queries do not accept attack inputs")
		}
		if command.Skip != 0 || command.Limit != 0 {
			return errors.New("hash-info queries cannot be combined with skip or limit")
		}
		if command.HashInfoLevel == 0 {
			command.HashInfoLevel = 1
		}
		if command.HashInfoLevel > 2 {
			return errors.New("hash-info level must be 1 or 2")
		}
		command.HashInfo = true
		if err := validateHashInfoQueryFields(command); err != nil {
			return err
		}
	default:
		return errors.New("unsupported synchronous query mode")
	}

	return validateStandaloneCrackQueryWithDistributedRules(command)
}

func validateStandaloneIdentifyHashes(hashes []string) error {
	if len(hashes) == 0 {
		return errors.New("identify query requires at least one hash")
	}
	if len(hashes) > maxStandaloneIdentifyHashes {
		return fmt.Errorf("identify query exceeds %d hashes", maxStandaloneIdentifyHashes)
	}
	totalBytes := 0
	for _, hash := range hashes {
		if !utf8.ValidString(hash) {
			return errors.New("identify query hash is not valid UTF-8")
		}
		totalBytes += len(hash)
		if totalBytes > maxStandaloneIdentifyHashBytes {
			return fmt.Errorf("identify query hashes exceed %d bytes", maxStandaloneIdentifyHashBytes)
		}
	}
	return validateCrackHashes(hashes)
}

func validateIdentifyQueryFields(command *models.CrackCommand) error {
	actual := command.ToProtobuf()
	allowed := &clientpb.CrackCommand{
		HashType:     clientpb.HashType_INVALID,
		Hashes:       append([]string(nil), actual.Hashes...),
		IdentifyMode: true,
	}
	if !proto.Equal(actual, allowed) {
		return errors.New("identify query includes options unrelated to hash identification")
	}
	return nil
}

func validateHashInfoQueryFields(command *models.CrackCommand) error {
	actual := command.ToProtobuf()
	allowed := &clientpb.CrackCommand{
		HashType:      actual.HashType,
		HashMode:      actual.HashMode,
		HashInfo:      true,
		HashInfoLevel: actual.HashInfoLevel,
	}
	if !proto.Equal(actual, allowed) {
		return errors.New("hash-info query includes options unrelated to hash information")
	}
	return nil
}

func validateStandaloneCrackQueryWithDistributedRules(command *models.CrackCommand) error {
	keyspace := command.Keyspace
	totalCandidates := command.TotalCandidates
	lookup := command.Lookup
	identifyMode := command.IdentifyMode
	hashInfo := command.HashInfo
	hashInfoLevel := command.HashInfoLevel
	hashType := command.HashType

	command.Keyspace = false
	command.TotalCandidates = false
	command.Lookup = ""
	command.IdentifyMode = false
	command.HashInfo = false
	command.HashInfoLevel = 0
	if command.HashMode == nil && command.HashType == int32(clientpb.HashType_INVALID) {
		command.HashType = int32(clientpb.HashType_MD5)
	}
	err := validateDistributedCrackCommand(command)
	command.HashType = hashType
	command.Keyspace = keyspace
	command.TotalCandidates = totalCandidates
	command.Lookup = lookup
	command.IdentifyMode = identifyMode
	command.HashInfo = hashInfo
	command.HashInfoLevel = hashInfoLevel
	return err
}

func validateStandaloneCrackQueryOperands(command *models.CrackCommand, mode clientpb.CrackQueryMode) error {
	switch mode {
	case clientpb.CrackQueryMode_CRACK_QUERY_KEYSPACE, clientpb.CrackQueryMode_CRACK_QUERY_TOTAL_CANDIDATES, clientpb.CrackQueryMode_CRACK_QUERY_LOOKUP:
		if mode == clientpb.CrackQueryMode_CRACK_QUERY_LOOKUP &&
			clientpb.CrackAttackMode(command.AttackMode) == clientpb.CrackAttackMode_STRAIGHT &&
			len(distributedCrackOperands(command)) != 1 {
			return errors.New("straight attack lookup requires exactly one managed wordlist")
		}
		return validateDistributedCrackOperands(command)
	case clientpb.CrackQueryMode_CRACK_QUERY_IDENTIFY:
		return validateStandaloneIdentifyHashes(command.Hashes)
	case clientpb.CrackQueryMode_CRACK_QUERY_HASH_INFO:
		return nil
	default:
		return errors.New("unsupported synchronous query mode")
	}
}

//nolint:gocyclo // Reservation, capability selection, enqueue, and rollback must remain atomic under the queue lock.
func reserveStandaloneCrackQueryTask(ctx context.Context, taskID models.UUID, command *models.CrackCommand, mode clientpb.CrackQueryMode, selector string) (*standaloneCrackKeyspaceTask, error) {
	crackQueueMu.Lock()
	defer crackQueueMu.Unlock()

	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}
	now := time.Now()
	reapStandaloneCrackKeyspaceTasksLocked(now)

	requiredCapability := ""
	if mode != clientpb.CrackQueryMode_CRACK_QUERY_KEYSPACE {
		requiredCapability = clientpb.CrackstationCapabilityCrackQueryV1
	}
	stations, err := selectableCrackstationsWithCapabilityLocked(ctx, selector, requiredCapability)
	if err != nil {
		return nil, err
	}
	if len(stations) == 0 {
		if selector != "" {
			return nil, errSelectedCrackstationUnavailable
		}
		return nil, errNoIdleCrackstation
	}

	for _, station := range stations {
		snapshot := station.Snapshot()
		if !crackstationSupportsCapability(snapshot, requiredCapability) {
			if selector != "" {
				return nil, errSelectedCrackstationUnsupported
			}
			continue
		}
		if core.GetCrackstation(snapshot.HostUUID) != station || !standaloneCrackstationIsIdle(snapshot) || !standaloneCrackstationMatchesSelector(snapshot, selector) {
			if selector != "" {
				return nil, errSelectedCrackstationUnavailable
			}
			continue
		}
		leaseToken := models.NewUUID().String()
		kind := clientpb.CrackTaskKind_CRACK_TASK_QUERY
		eventType := consts.CrackQuery
		if mode == clientpb.CrackQueryMode_CRACK_QUERY_KEYSPACE {
			// Preserve compatibility with keyspace-capable workers deployed before
			// the generic synchronous query protocol was introduced.
			kind = clientpb.CrackTaskKind_CRACK_TASK_KEYSPACE
			eventType = consts.CrackKeyspace
		}
		task := &clientpb.CrackTask{
			ID:             taskID.String(),
			HostUUID:       snapshot.HostUUID,
			CreatedAt:      now.Unix(),
			Command:        command.ToProtobuf(),
			Kind:           kind,
			State:          clientpb.CrackTaskState_CRACK_TASK_LEASED,
			Attempt:        1,
			LeaseToken:     leaseToken,
			LeaseExpiresAt: now.Add(crackKeyspaceLeaseDuration).Unix(),
			UpdatedAt:      now.Unix(),
		}
		entry := &standaloneCrackKeyspaceTask{
			task: task, station: station, mode: mode,
			stationName: snapshot.Name, hashcatVersion: snapshot.HashcatVersion,
			done: make(chan struct{}),
		}
		standaloneCrackKeyspaceTasks[task.ID] = entry

		assignmentData, marshalErr := json.Marshal(crackTaskAssignment{
			TaskID: task.ID, HostUUID: task.HostUUID, Attempt: task.Attempt, LeaseToken: task.LeaseToken,
		})
		if marshalErr != nil {
			delete(standaloneCrackKeyspaceTasks, task.ID)
			return nil, marshalErr
		}
		event := &clientpb.Event{EventType: eventType, Data: assignmentData}
		select {
		case station.Events <- event:
			return entry, nil
		default:
			delete(standaloneCrackKeyspaceTasks, task.ID)
			if selector != "" {
				return nil, errCrackstationEventQueueFull
			}
		}
	}
	if selector != "" {
		return nil, errSelectedCrackstationUnavailable
	}
	return nil, errCrackstationEventQueueFull
}

// reserveStandaloneCrackKeyspaceTask preserves the original internal helper.
func reserveStandaloneCrackKeyspaceTask(ctx context.Context, taskID models.UUID, command *models.CrackCommand) (*standaloneCrackKeyspaceTask, error) {
	return reserveStandaloneCrackQueryTask(ctx, taskID, command, clientpb.CrackQueryMode_CRACK_QUERY_KEYSPACE, "")
}

func selectIdleCrackstationLocked(ctx context.Context) (*core.Crackstation, error) {
	stations, err := selectableCrackstationsLocked(ctx, "")
	if err != nil || len(stations) == 0 {
		return nil, err
	}
	return stations[0], nil
}

func selectableCrackstationsLocked(ctx context.Context, selector string) ([]*core.Crackstation, error) {
	return selectableCrackstationsWithCapabilityLocked(ctx, selector, "")
}

//nolint:gocyclo // Ownership, lease, selector, and capability filters intentionally share one deterministic pass.
func selectableCrackstationsWithCapabilityLocked(ctx context.Context, selector string, requiredCapability string) ([]*core.Crackstation, error) {
	busyHosts := map[string]struct{}{}
	var active []models.CrackTask
	if err := db.Session().WithContext(ctx).
		Select("crackstation_id").
		Where("state IN ?", []int32{
			int32(clientpb.CrackTaskState_CRACK_TASK_LEASED),
			int32(clientpb.CrackTaskState_CRACK_TASK_RUNNING),
		}).Find(&active).Error; err != nil {
		return nil, err
	}
	for index := range active {
		if active[index].CrackstationID != models.NilUUID() {
			busyHosts[active[index].CrackstationID.String()] = struct{}{}
		}
	}
	for _, entry := range standaloneCrackKeyspaceTasks {
		if entry != nil && entry.task != nil && entry.task.HostUUID != "" {
			busyHosts[entry.task.HostUUID] = struct{}{}
		}
	}

	snapshots := core.AllCrackstations()
	sort.Slice(snapshots, func(i, j int) bool { return snapshots[i].HostUUID < snapshots[j].HostUUID })
	if selector != "" {
		matches := make([]*clientpb.Crackstation, 0, 2)
		selectorUUID := models.ParseUUIDOrNil(selector)
		if selectorUUID != models.NilUUID() {
			for _, snapshot := range snapshots {
				if snapshot.HostUUID == selectorUUID.String() {
					matches = append(matches, snapshot)
				}
			}
		} else {
			for _, snapshot := range snapshots {
				if snapshot.Name == selector {
					matches = append(matches, snapshot)
				}
			}
			if len(matches) > 1 {
				return nil, errCrackstationSelectorAmbiguous
			}
		}
		if len(matches) == 0 {
			return nil, errCrackstationSelectorNotFound
		}
		if len(matches) > 1 {
			return nil, errCrackstationSelectorAmbiguous
		}
		if !crackstationSupportsCapability(matches[0], requiredCapability) {
			return nil, errSelectedCrackstationUnsupported
		}
		return availableRuntimeCrackstation(matches[0], busyHosts)
	}

	stations := make([]*core.Crackstation, 0, len(snapshots))
	for _, snapshot := range snapshots {
		if !crackstationSupportsCapability(snapshot, requiredCapability) {
			continue
		}
		available, err := availableRuntimeCrackstation(snapshot, busyHosts)
		if err != nil || len(available) == 0 {
			continue
		}
		stations = append(stations, available[0])
	}
	return stations, nil
}

func crackstationSupportsCapability(station *clientpb.Crackstation, requiredCapability string) bool {
	if requiredCapability == "" {
		return true
	}
	if station == nil {
		return false
	}
	for _, capability := range station.Capabilities {
		if capability == requiredCapability {
			return true
		}
	}
	return false
}

func availableRuntimeCrackstation(snapshot *clientpb.Crackstation, busyHosts map[string]struct{}) ([]*core.Crackstation, error) {
	if !standaloneCrackstationIsIdle(snapshot) {
		return nil, errSelectedCrackstationUnavailable
	}
	if _, busy := busyHosts[snapshot.HostUUID]; busy {
		return nil, errSelectedCrackstationUnavailable
	}
	station := core.GetCrackstation(snapshot.HostUUID)
	if station == nil {
		return nil, errCrackstationSelectorNotFound
	}
	current := station.Snapshot()
	if current.HostUUID != snapshot.HostUUID || current.Name != snapshot.Name || !standaloneCrackstationIsIdle(current) {
		return nil, errSelectedCrackstationUnavailable
	}
	if _, busy := busyHosts[current.HostUUID]; busy {
		return nil, errSelectedCrackstationUnavailable
	}
	return []*core.Crackstation{station}, nil
}

func standaloneCrackstationMatchesSelector(station *clientpb.Crackstation, selector string) bool {
	if selector == "" {
		return true
	}
	selectorUUID := models.ParseUUIDOrNil(selector)
	if selectorUUID != models.NilUUID() {
		return station != nil && station.HostUUID == selectorUUID.String()
	}
	return station != nil && station.Name == selector
}

func standaloneCrackstationIsIdle(station *clientpb.Crackstation) bool {
	if station == nil || models.ParseUUIDOrNil(station.HostUUID) == models.NilUUID() || station.Status == nil {
		return false
	}
	status := station.Status
	return status.HostUUID == station.HostUUID && status.State == clientpb.States_IDLE && !status.IsSyncing && status.CurrentCrackJobID == ""
}

func requestStandaloneCrackKeyspaceCancellation(taskID string, expected *standaloneCrackKeyspaceTask) {
	crackQueueMu.Lock()
	defer crackQueueMu.Unlock()
	entry := standaloneCrackKeyspaceTasks[taskID]
	if entry == nil || entry != expected {
		return
	}
	// There is no worker cancellation event. Keep this tombstone and reserve
	// its station until the worker acknowledges the stale lease, disconnects,
	// or the lease expires.
	entry.cancelRequested = true
}

// standaloneCrackTaskByID is the in-memory fallback for CrackTaskByID. The
// boolean reports whether the task ID belonged to the standalone registry.
//
//nolint:gocyclo // Registry validation and response redaction must use one locked snapshot of the transient task.
func (rpc *Server) standaloneCrackTaskByID(ctx context.Context, req *clientpb.CrackTask) (*clientpb.CrackTask, bool, error) {
	if req == nil {
		return nil, false, nil
	}
	crackQueueMu.Lock()
	entry := standaloneCrackKeyspaceTasks[req.ID]
	if entry == nil {
		crackQueueMu.Unlock()
		return nil, false, nil
	}
	task := entry.task
	if req.HostUUID == "" || req.HostUUID != task.HostUUID || req.Attempt != task.Attempt || req.LeaseToken == "" || req.LeaseToken != task.LeaseToken {
		crackQueueMu.Unlock()
		return nil, true, status.Error(codes.PermissionDenied, "crack task assignment does not match lease")
	}
	expectedHost := task.HostUUID
	crackQueueMu.Unlock()

	if err := rpc.authorizeCrackstation(ctx, expectedHost); err != nil {
		return nil, true, err
	}

	// Authorization is intentionally outside crackQueueMu. Recheck the exact
	// entry and lease before acknowledging cancellation or expiry.
	crackQueueMu.Lock()
	entry = standaloneCrackKeyspaceTasks[req.ID]
	if entry == nil || entry.task == nil || entry.task.HostUUID != req.HostUUID || entry.task.Attempt != req.Attempt || entry.task.LeaseToken != req.LeaseToken {
		crackQueueMu.Unlock()
		return nil, true, status.Error(codes.Aborted, errStaleCrackTaskAttempt.Error())
	}
	if entry.cancelRequested {
		if !entry.fetched {
			completeStandaloneCrackKeyspaceTaskLocked(req.ID, entry, nil,
				status.Error(codes.Canceled, "standalone crack query was canceled"))
		}
		crackQueueMu.Unlock()
		return nil, true, status.Error(codes.Aborted, errStaleCrackTaskAttempt.Error())
	}
	if standaloneCrackKeyspaceTaskStaleLocked(req.ID, entry, time.Now()) {
		crackQueueMu.Unlock()
		return nil, true, status.Error(codes.Aborted, errStaleCrackTaskAttempt.Error())
	}
	entry.fetched = true
	response := proto.Clone(entry.task).(*clientpb.CrackTask)
	crackQueueMu.Unlock()
	return response, true, nil
}

// standaloneCrackTaskUpdate is the in-memory fallback for CrackTaskUpdate.
//
//nolint:gocyclo // Lease validation, status transitions, and completion are one atomic in-memory lifecycle operation.
func (rpc *Server) standaloneCrackTaskUpdate(ctx context.Context, req *clientpb.CrackTask) (bool, error) {
	if req == nil {
		return false, nil
	}
	if err := validateStandaloneCrackTaskUpdate(req); err != nil {
		crackQueueMu.Lock()
		_, handled := standaloneCrackKeyspaceTasks[req.ID]
		crackQueueMu.Unlock()
		return handled, err
	}

	crackQueueMu.Lock()
	entry := standaloneCrackKeyspaceTasks[req.ID]
	if entry == nil {
		crackQueueMu.Unlock()
		return false, nil
	}
	expectedHost := entry.task.HostUUID
	crackQueueMu.Unlock()
	if req.HostUUID == "" || req.HostUUID != expectedHost {
		return true, status.Error(codes.PermissionDenied, "crack task is not assigned to host")
	}
	if err := rpc.authorizeCrackstation(ctx, req.HostUUID); err != nil {
		return true, err
	}

	crackQueueMu.Lock()
	defer crackQueueMu.Unlock()
	entry = standaloneCrackKeyspaceTasks[req.ID]
	if entry == nil {
		return true, status.Error(codes.Aborted, errStaleCrackTaskAttempt.Error())
	}
	now := time.Now()
	task := entry.task
	if task.HostUUID != req.HostUUID || task.Attempt != req.Attempt || task.LeaseToken == "" || task.LeaseToken != req.LeaseToken {
		return true, status.Error(codes.Aborted, errStaleCrackTaskAttempt.Error())
	}
	if entry.cancelRequested {
		terminal := req.CompletedAt != 0 || req.State == clientpb.CrackTaskState_CRACK_TASK_COMPLETED || req.State == clientpb.CrackTaskState_CRACK_TASK_FAILED
		if task.State == clientpb.CrackTaskState_CRACK_TASK_LEASED || terminal {
			completeStandaloneCrackKeyspaceTaskLocked(req.ID, entry, nil,
				status.Error(codes.Canceled, "standalone crack query was canceled"))
		}
		return true, status.Error(codes.Aborted, errStaleCrackTaskAttempt.Error())
	}
	if standaloneCrackKeyspaceTaskStaleLocked(req.ID, entry, now) {
		return true, status.Error(codes.Aborted, errStaleCrackTaskAttempt.Error())
	}
	if task.State != clientpb.CrackTaskState_CRACK_TASK_LEASED && task.State != clientpb.CrackTaskState_CRACK_TASK_RUNNING {
		return true, status.Error(codes.Aborted, errStaleCrackTaskAttempt.Error())
	}

	state := req.State
	if req.CompletedAt != 0 {
		if req.Err != "" {
			state = clientpb.CrackTaskState_CRACK_TASK_FAILED
		} else {
			state = clientpb.CrackTaskState_CRACK_TASK_COMPLETED
		}
	} else if req.StartedAt != 0 {
		state = clientpb.CrackTaskState_CRACK_TASK_RUNNING
	}
	if state == clientpb.CrackTaskState_CRACK_TASK_FAILED && req.Err == "" {
		req.Err = "crack query failed"
	}
	if state != clientpb.CrackTaskState_CRACK_TASK_LEASED && state != clientpb.CrackTaskState_CRACK_TASK_RUNNING && state != clientpb.CrackTaskState_CRACK_TASK_COMPLETED && state != clientpb.CrackTaskState_CRACK_TASK_FAILED {
		return true, fmt.Errorf("invalid crack task transition to %s", state.String())
	}

	value := ""
	var resultErr error
	if state == clientpb.CrackTaskState_CRACK_TASK_COMPLETED {
		value, resultErr = standaloneCrackQueryValue(entry.mode, req)
	} else if req.Keyspace != "" {
		return true, errors.New("crack task keyspace is only valid on a successfully completed keyspace query")
	}

	task.State = state
	task.Err = req.Err
	task.Stdout = append(task.Stdout[:0], req.Stdout...)
	task.Stderr = append(task.Stderr[:0], req.Stderr...)
	task.ExitCode = req.ExitCode
	task.StdoutTruncated = req.StdoutTruncated
	task.StderrTruncated = req.StderrTruncated
	task.StdoutTotalBytes = req.StdoutTotalBytes
	task.StderrTotalBytes = req.StderrTotalBytes
	task.LatestStatusJSON = append(task.LatestStatusJSON[:0], req.LatestStatusJSON...)
	task.UpdatedAt = now.Unix()
	if state == clientpb.CrackTaskState_CRACK_TASK_RUNNING && task.StartedAt == 0 {
		task.StartedAt = now.Unix()
	}

	terminal := state == clientpb.CrackTaskState_CRACK_TASK_COMPLETED || state == clientpb.CrackTaskState_CRACK_TASK_FAILED
	if !terminal {
		task.LastHeartbeatAt = now.Unix()
		task.LeaseExpiresAt = now.Add(crackKeyspaceLeaseDuration).Unix()
		return true, nil
	}
	task.CompletedAt = now.Unix()
	task.LeaseExpiresAt = 0
	if entry.mode == clientpb.CrackQueryMode_CRACK_QUERY_KEYSPACE {
		task.Keyspace = value
	}
	result := proto.Clone(task).(*clientpb.CrackTask)
	if state == clientpb.CrackTaskState_CRACK_TASK_FAILED {
		resultErr = status.Errorf(codes.Unknown, "crack query failed: %s", task.Err)
	} else if resultErr != nil {
		resultErr = status.Errorf(codes.DataLoss, "invalid crack query result: %s", resultErr)
	}
	entry.resultValue = value
	entry.resultStderr = string(req.Stderr)
	completeStandaloneCrackKeyspaceTaskLocked(req.ID, entry, result, resultErr)
	if err := scheduleCrackTasksLocked(now); err != nil {
		crackCommandRPCLog.Warnf("Completed standalone crack query but could not schedule queued work: %s", err)
	}
	// The terminal update has been consumed and the operator-facing result has
	// already been delivered through entry.done. Acknowledge the worker even
	// when that result is invalid or failed so it does not retry a completed
	// transient task; runStandaloneCrackQuery returns resultErr to the operator.
	return true, nil
}

func validateStandaloneCrackTaskUpdate(req *clientpb.CrackTask) error {
	if proto.Size(req) > maxCrackTaskUpdateBytes {
		return errors.New("crack task update exceeds size limit")
	}
	if len(req.Err) > maxCrackTaskErrorBytes {
		return errors.New("crack task error exceeds size limit")
	}
	if len(req.Stdout) > maxCrackOutputBytes || len(req.Stderr) > maxCrackOutputBytes {
		return errors.New("crack task output exceeds size limit")
	}
	if len(req.LatestStatusJSON) > maxCrackStatusBytes {
		return errors.New("crack task status exceeds size limit")
	}
	if len(req.LatestStatusJSON) != 0 && !json.Valid(req.LatestStatusJSON) {
		return errors.New("invalid crack task status")
	}
	if len(req.RecoveredJSON) != 0 {
		return errors.New("query tasks cannot report recovered results")
	}
	return nil
}

//nolint:gocyclo // Mode-specific response validation stays centralized so every synchronous query has the same trust boundary.
func standaloneCrackQueryValue(mode clientpb.CrackQueryMode, req *clientpb.CrackTask) (string, error) {
	if req == nil {
		return "", errors.New("missing crack query result")
	}
	if req.ExitCode != 0 {
		return "", fmt.Errorf("hashcat exited with status %d", req.ExitCode)
	}
	if req.StdoutTruncated || req.StderrTruncated {
		return "", errors.New("hashcat query output was truncated")
	}
	if !utf8.Valid(req.Stdout) || !utf8.Valid(req.Stderr) {
		return "", errors.New("hashcat query output is not valid UTF-8")
	}
	stdout := strings.TrimSpace(string(req.Stdout))
	switch mode {
	case clientpb.CrackQueryMode_CRACK_QUERY_KEYSPACE:
		value := strings.TrimSpace(req.Keyspace)
		if req.Keyspace != "" && value != req.Keyspace {
			return "", errors.New("invalid keyspace result")
		}
		if value == "" {
			value = stdout
		} else if stdout != "" && stdout != value {
			return "", errors.New("keyspace field and stdout disagree")
		}
		return canonicalCrackQueryCount("keyspace", value)
	case clientpb.CrackQueryMode_CRACK_QUERY_TOTAL_CANDIDATES:
		if req.Keyspace != "" {
			return "", errors.New("total-candidates result cannot contain a keyspace field")
		}
		return canonicalCrackQueryCount("total-candidates", stdout)
	case clientpb.CrackQueryMode_CRACK_QUERY_LOOKUP, clientpb.CrackQueryMode_CRACK_QUERY_IDENTIFY, clientpb.CrackQueryMode_CRACK_QUERY_HASH_INFO:
		if req.Keyspace != "" {
			return "", errors.New("non-keyspace result cannot contain a keyspace field")
		}
		if stdout == "" {
			return "", errors.New("hashcat query returned empty stdout")
		}
		return stdout, nil
	default:
		return "", errors.New("unknown crack query mode")
	}
}

func canonicalCrackQueryCount(label, value string) (string, error) {
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil || strconv.FormatUint(parsed, 10) != value {
		return "", fmt.Errorf("invalid %s result", label)
	}
	return value, nil
}

// standaloneCrackKeyspaceResult preserves the original keyspace parser API.
func standaloneCrackKeyspaceResult(req *clientpb.CrackTask) (string, error) {
	return standaloneCrackQueryValue(clientpb.CrackQueryMode_CRACK_QUERY_KEYSPACE, req)
}

// updateStandaloneCrackTaskStatusLocked is the in-memory fallback for
// updateCrackTaskStatus. The caller must hold crackQueueMu.
func updateStandaloneCrackTaskStatusLocked(event crackTaskStatusEvent, now time.Time) (bool, error) {
	entry := standaloneCrackKeyspaceTasks[event.TaskID]
	if entry == nil {
		return false, nil
	}
	if err := validateCrackTaskStatusEvent(event); err != nil {
		return true, err
	}
	task := entry.task
	if task.HostUUID != event.HostUUID || task.Attempt != event.Attempt || task.LeaseToken == "" || task.LeaseToken != event.LeaseToken {
		return true, errStaleCrackTaskAttempt
	}
	if entry.cancelRequested {
		return true, errStaleCrackTaskAttempt
	}
	if standaloneCrackKeyspaceTaskStaleLocked(event.TaskID, entry, now) {
		return true, errStaleCrackTaskAttempt
	}
	if task.State != clientpb.CrackTaskState_CRACK_TASK_LEASED && task.State != clientpb.CrackTaskState_CRACK_TASK_RUNNING {
		return true, errStaleCrackTaskAttempt
	}
	task.State = clientpb.CrackTaskState_CRACK_TASK_RUNNING
	task.LatestStatusJSON = append(task.LatestStatusJSON[:0], event.Status...)
	task.LastHeartbeatAt = now.Unix()
	task.LeaseExpiresAt = now.Add(crackKeyspaceLeaseDuration).Unix()
	task.UpdatedAt = now.Unix()
	if task.StartedAt == 0 {
		task.StartedAt = now.Unix()
	}
	return true, nil
}

func standaloneCrackKeyspaceTaskStaleLocked(taskID string, entry *standaloneCrackKeyspaceTask, now time.Time) bool {
	if entry == nil || entry.task == nil {
		return true
	}
	if entry.task.LeaseExpiresAt == 0 || !time.Unix(entry.task.LeaseExpiresAt, 0).After(now) {
		completeStandaloneCrackKeyspaceTaskLocked(taskID, entry, nil,
			status.Error(codes.DeadlineExceeded, "standalone crack query lease expired"))
		return true
	}
	return false
}

func completeStandaloneCrackKeyspaceTaskLocked(taskID string, entry *standaloneCrackKeyspaceTask, result *clientpb.CrackTask, resultErr error) {
	current := standaloneCrackKeyspaceTasks[taskID]
	if current == nil || current != entry {
		return
	}
	entry.result = result
	entry.resultErr = resultErr
	delete(standaloneCrackKeyspaceTasks, taskID)
	close(entry.done)
}

func reapStandaloneCrackKeyspaceTasksLocked(now time.Time) {
	for taskID, entry := range standaloneCrackKeyspaceTasks {
		if entry == nil || entry.task == nil || entry.task.LeaseExpiresAt == 0 || !time.Unix(entry.task.LeaseExpiresAt, 0).After(now) {
			completeStandaloneCrackKeyspaceTaskLocked(taskID, entry, nil,
				status.Error(codes.DeadlineExceeded, "standalone crack query lease expired"))
		}
	}
}

// failStandaloneCrackKeyspaceTasksForStationLocked is connection-specific:
// an older stream disconnecting must not fail work assigned to a replacement
// connection registered with the same stable host UUID.
func failStandaloneCrackKeyspaceTasksForStationLocked(station *core.Crackstation) {
	if station == nil {
		return
	}
	for taskID, entry := range standaloneCrackKeyspaceTasks {
		if entry == nil || entry.station != station {
			continue
		}
		completeStandaloneCrackKeyspaceTaskLocked(taskID, entry, nil,
			status.Error(codes.Unavailable, "crackstation disconnected during synchronous query"))
	}
}

func excludeStandaloneCrackKeyspaceHostsLocked(available map[string]*core.Crackstation) {
	for _, entry := range standaloneCrackKeyspaceTasks {
		if entry != nil && entry.task != nil {
			delete(available, entry.task.HostUUID)
		}
	}
}

// activeStandaloneCrackKeyspaceReferencesManagedFile is called while
// crackFileLifecycleMu is held. Reservation follows lifecycle -> queue locking.
func activeStandaloneCrackKeyspaceReferencesManagedFile(uri string) bool {
	if uri == "" {
		return false
	}
	crackQueueMu.Lock()
	defer crackQueueMu.Unlock()
	for _, entry := range standaloneCrackKeyspaceTasks {
		if entry == nil || entry.task == nil || entry.task.Command == nil {
			continue
		}
		command := models.CrackCommand{}.FromProtobuf(entry.task.Command)
		if crackCommandReferencesManagedFile(command, uri) {
			return true
		}
	}
	return false
}
