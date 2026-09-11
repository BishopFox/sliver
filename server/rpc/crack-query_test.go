package rpc

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/db/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type standaloneQueryTestResult struct {
	response *clientpb.CrackResponse
	err      error
}

//nolint:gocyclo // The table compares every transient query mode across assignment and result delivery.
func TestStandaloneCrackQueryModesUseTransientWorkerProtocol(t *testing.T) {
	database := setupCrackstationRPCTestDB(t)
	resetStandaloneCrackKeyspaceTasksForTest(t)
	disableCrackQueueReaperForTest(t)
	station := addQueueTestStation(t, database, "11111111-1111-4111-8111-111111111111", nil)
	station.Station.Name = "alpha"
	station.Station.Capabilities = []string{clientpb.CrackstationCapabilityCrackQueryV1}
	setStandaloneKeyspaceTestStatus(station, station.HostUUID, clientpb.States_IDLE, false, "")
	rpcServer := &Server{}
	stationContext := crackstationTestContext("queue-test")

	tests := []struct {
		name      string
		mode      clientpb.CrackQueryMode
		command   *clientpb.CrackCommand
		stdout    string
		stderr    string
		keyspace  string
		wantValue string
	}{
		{
			name: "keyspace", mode: clientpb.CrackQueryMode_CRACK_QUERY_KEYSPACE,
			command: &clientpb.CrackCommand{AttackMode: clientpb.CrackAttackMode_BRUTEFORCE, HashType: clientpb.HashType_INVALID, Keyspace: true, PositionalArguments: []string{"?d"}},
			stdout:  "10\n", keyspace: "10", wantValue: "10",
		},
		{
			name: "total candidates", mode: clientpb.CrackQueryMode_CRACK_QUERY_TOTAL_CANDIDATES,
			command: &clientpb.CrackCommand{AttackMode: clientpb.CrackAttackMode_BRUTEFORCE, HashType: clientpb.HashType_INVALID, TotalCandidates: true, PositionalArguments: []string{"?d"}},
			stdout:  "10\n", wantValue: "10",
		},
		{
			name: "lookup", mode: clientpb.CrackQueryMode_CRACK_QUERY_LOOKUP,
			command: &clientpb.CrackCommand{AttackMode: clientpb.CrackAttackMode_BRUTEFORCE, HashType: clientpb.HashType_INVALID, Lookup: "7", Skip: 2, Limit: 5, PositionalArguments: []string{"?d"}},
			stdout:  "candidate-7\n", wantValue: "candidate-7",
		},
		{
			name: "identify", mode: clientpb.CrackQueryMode_CRACK_QUERY_IDENTIFY,
			command: &clientpb.CrackCommand{HashType: clientpb.HashType_INVALID, IdentifyMode: true, Hashes: []string{"5f4dcc3b5aa765d61d8327deb882cf99"}},
			stdout:  "0\n100\n", wantValue: "0\n100",
		},
		{
			name: "hash info level activates mode", mode: clientpb.CrackQueryMode_CRACK_QUERY_HASH_INFO,
			command: &clientpb.CrackCommand{HashType: clientpb.HashType_MD5, HashInfoLevel: 2},
			stdout:  `{"mode":0}` + "\n", stderr: "hashcat warning\n", wantValue: `{"mode":0}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := proto.Clone(test.command).(*clientpb.CrackCommand)
			request.Crackstation = station.Station.Name
			result := make(chan standaloneQueryTestResult, 1)
			go func() {
				response, err := rpcServer.Crack(t.Context(), request)
				result <- standaloneQueryTestResult{response: response, err: err}
			}()

			eventType := consts.CrackQuery
			wantKind := clientpb.CrackTaskKind_CRACK_TASK_QUERY
			if test.mode == clientpb.CrackQueryMode_CRACK_QUERY_KEYSPACE {
				eventType = consts.CrackKeyspace
				wantKind = clientpb.CrackTaskKind_CRACK_TASK_KEYSPACE
			}
			assignment := receiveStandaloneAssignment(t, station, eventType)
			fetched, err := rpcServer.CrackTaskByID(stationContext, &clientpb.CrackTask{
				ID: assignment.TaskID, HostUUID: assignment.HostUUID, Attempt: assignment.Attempt, LeaseToken: assignment.LeaseToken,
			})
			if err != nil {
				t.Fatalf("fetch query task: %v", err)
			}
			if fetched.Kind != wantKind || fetched.CrackJobID != "" || fetched.Command == nil {
				t.Fatalf("query task lifecycle = %#v", fetched)
			}
			if fetched.Command.Crackstation != "" {
				t.Fatalf("worker received server-only selector %q", fetched.Command.Crackstation)
			}

			completed := proto.Clone(fetched).(*clientpb.CrackTask)
			completed.Command = nil
			completed.State = clientpb.CrackTaskState_CRACK_TASK_COMPLETED
			completed.CompletedAt = time.Now().Unix()
			completed.Stdout = []byte(test.stdout)
			completed.Stderr = []byte(test.stderr)
			completed.Keyspace = test.keyspace
			if _, err := rpcServer.CrackTaskUpdate(stationContext, completed); err != nil {
				t.Fatalf("complete query task: %v", err)
			}

			got := <-result
			if got.err != nil {
				t.Fatalf("Crack query: %v", got.err)
			}
			if got.response == nil || got.response.Job != nil || got.response.Query == nil {
				t.Fatalf("query response = %#v", got.response)
			}
			query := got.response.Query
			if query.Mode != test.mode || query.Value != test.wantValue || query.CrackstationHostUUID != station.HostUUID ||
				query.CrackstationName != station.Station.Name || query.HashcatVersion != "hashcat-test-v1" {
				t.Fatalf("query result = %#v", query)
			}
			if query.Stderr != test.stderr {
				t.Fatalf("query stderr = %q, want %q", query.Stderr, test.stderr)
			}
			if test.mode == clientpb.CrackQueryMode_CRACK_QUERY_KEYSPACE {
				if got.response.Keyspace != test.wantValue {
					t.Fatalf("legacy keyspace = %q, want %q", got.response.Keyspace, test.wantValue)
				}
			} else if got.response.Keyspace != "" {
				t.Fatalf("non-keyspace response populated legacy keyspace %q", got.response.Keyspace)
			}
		})
	}

	for name, model := range map[string]interface{}{
		"jobs": &models.CrackJob{}, "tasks": &models.CrackTask{}, "commands": &models.CrackCommand{},
	} {
		var count int64
		if err := database.Model(model).Count(&count).Error; err != nil {
			t.Fatalf("count persisted %s: %v", name, err)
		}
		if count != 0 {
			t.Fatalf("standalone queries persisted %d %s", count, name)
		}
	}
}

func TestStandaloneCrackQuerySelectorErrors(t *testing.T) {
	database := setupCrackstationRPCTestDB(t)
	resetStandaloneCrackKeyspaceTasksForTest(t)
	first := addQueueTestStation(t, database, "11111111-1111-4111-8111-111111111111", nil)
	first.Station.Name = "duplicate"
	first.Station.Capabilities = []string{clientpb.CrackstationCapabilityCrackQueryV1}
	setStandaloneKeyspaceTestStatus(first, first.HostUUID, clientpb.States_IDLE, false, "")
	second := addQueueTestStation(t, database, "22222222-2222-4222-8222-222222222222", nil)
	second.Station.Name = "duplicate"
	second.Station.Capabilities = []string{clientpb.CrackstationCapabilityCrackQueryV1}
	setStandaloneKeyspaceTestStatus(second, second.HostUUID, clientpb.States_IDLE, false, "")
	rpcServer := &Server{}

	query := func(selector string) error {
		_, err := rpcServer.Crack(t.Context(), &clientpb.CrackCommand{
			HashType: clientpb.HashType_MD5, HashInfo: true, Crackstation: selector,
		})
		return err
	}
	if err := query("missing"); status.Code(err) != codes.NotFound {
		t.Fatalf("missing selector error = %v, want NotFound", err)
	}
	if err := query("duplicate"); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("duplicate selector error = %v, want InvalidArgument", err)
	}

	setStandaloneKeyspaceTestStatus(second, second.HostUUID, clientpb.States_CRACKING, false, "")
	if err := query(second.HostUUID); status.Code(err) != codes.ResourceExhausted {
		t.Fatalf("busy UUID selector error = %v, want ResourceExhausted", err)
	}
	if err := query(strings.Repeat("x", maxCrackstationSelectorBytes+1)); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("oversized selector error = %v, want InvalidArgument", err)
	}
	if err := query(string([]byte{0xff})); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("invalid UTF-8 selector error = %v, want InvalidArgument", err)
	}
}

func TestStandaloneCrackQueryValidation(t *testing.T) {
	tests := []struct {
		name    string
		command *models.CrackCommand
	}{
		{name: "multiple modes", command: &models.CrackCommand{Keyspace: true, TotalCandidates: true}},
		{name: "keyspace skip", command: &models.CrackCommand{Keyspace: true, Skip: 1}},
		{name: "total limit", command: &models.CrackCommand{TotalCandidates: true, Limit: 1}},
		{name: "lookup newline", command: &models.CrackCommand{Lookup: "x\ny"}},
		{name: "lookup oversized", command: &models.CrackCommand{Lookup: strings.Repeat("x", maxStandaloneLookupBytes+1)}},
		{name: "identify hash mode", command: &models.CrackCommand{HashType: int32(clientpb.HashType_INVALID), HashMode: uint32Pointer(0), IdentifyMode: true, Hashes: []string{"hash"}}},
		{name: "identify credential", command: &models.CrackCommand{HashType: int32(clientpb.HashType_INVALID), IdentifyMode: true, Hashes: []string{"hash"}, CredentialIDs: []string{models.NewUUID().String()}}},
		{name: "identify attack input", command: &models.CrackCommand{HashType: int32(clientpb.HashType_INVALID), IdentifyMode: true, Hashes: []string{"hash"}, PositionalArguments: []string{"?d"}}},
		{name: "identify unrelated option", command: &models.CrackCommand{HashType: int32(clientpb.HashType_INVALID), IdentifyMode: true, Hashes: []string{"hash"}, Force: true}},
		{name: "hash info level", command: &models.CrackCommand{HashInfoLevel: 3}},
		{name: "hash info attack input", command: &models.CrackCommand{HashInfo: true, PositionalArguments: []string{"?d"}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mode, err := standaloneCrackQueryMode(test.command)
			if err == nil {
				err = prepareStandaloneCrackQuery(test.command, mode)
			}
			if err == nil {
				t.Fatal("invalid query command was accepted")
			}
		})
	}

	tooMany := make([]string, maxStandaloneIdentifyHashes+1)
	for index := range tooMany {
		tooMany[index] = "hash"
	}
	if err := validateStandaloneIdentifyHashes(tooMany); err == nil {
		t.Fatal("identify hash count limit was not enforced")
	}
	invalidUTF8 := string([]byte{utf8.RuneSelf, 0xff})
	if err := validateStandaloneIdentifyHashes([]string{invalidUTF8}); err == nil {
		t.Fatal("invalid UTF-8 identify hash was accepted")
	}
}

func TestStandaloneCrackLookupRejectsMultipleStraightWordlists(t *testing.T) {
	database := setupCrackstationRPCTestDB(t)
	wordlistNames := []string{"lookup-first.txt", "lookup-second.txt"}
	for index, name := range wordlistNames {
		if err := database.Create(&models.CrackFile{
			Name: name, Type: int32(clientpb.CrackFileType_WORDLIST),
			Sha2_256: strings.Repeat(string(rune('a'+index)), 64), IsComplete: true,
		}).Error; err != nil {
			t.Fatalf("create managed wordlist %q: %v", name, err)
		}
	}

	_, err := (&Server{}).Crack(t.Context(), &clientpb.CrackCommand{
		AttackMode: clientpb.CrackAttackMode_STRAIGHT,
		HashType:   clientpb.HashType_INVALID,
		Lookup:     "0",
		PositionalArguments: []string{
			wordlistNames[0],
			wordlistNames[1],
		},
	})
	if status.Code(err) != codes.InvalidArgument || !strings.Contains(err.Error(), "exactly one managed wordlist") {
		t.Fatalf("multi-wordlist straight lookup error = %v, want InvalidArgument", err)
	}

	multipleManagedWordlists := &models.CrackCommand{
		AttackMode: int32(clientpb.CrackAttackMode_STRAIGHT),
		PositionalArguments: []string{
			"crackfile://wordlist/first",
			"crackfile://wordlist/second",
		},
	}
	for _, mode := range []clientpb.CrackQueryMode{
		clientpb.CrackQueryMode_CRACK_QUERY_KEYSPACE,
		clientpb.CrackQueryMode_CRACK_QUERY_TOTAL_CANDIDATES,
	} {
		if err := validateStandaloneCrackQueryOperands(multipleManagedWordlists, mode); err != nil {
			t.Fatalf("%s rejected multiple straight wordlists: %v", mode, err)
		}
	}
}

func TestStandaloneCrackQueryCapabilityRouting(t *testing.T) {
	database := setupCrackstationRPCTestDB(t)
	resetStandaloneCrackKeyspaceTasksForTest(t)
	disableCrackQueueReaperForTest(t)

	legacy := addQueueTestStation(t, database, "11111111-1111-4111-8111-111111111111", nil)
	legacy.Station.Name = "legacy"
	setStandaloneKeyspaceTestStatus(legacy, legacy.HostUUID, clientpb.States_IDLE, false, "")
	capable := addQueueTestStation(t, database, "22222222-2222-4222-8222-222222222222", nil)
	capable.Station.Name = "capable"
	capable.Station.Capabilities = []string{"future-capability", clientpb.CrackstationCapabilityCrackQueryV1}
	setStandaloneKeyspaceTestStatus(capable, capable.HostUUID, clientpb.States_IDLE, false, "")

	if _, err := (&Server{}).Crack(t.Context(), &clientpb.CrackCommand{
		HashType: clientpb.HashType_MD5, HashInfo: true, Crackstation: legacy.HostUUID,
	}); status.Code(err) != codes.FailedPrecondition || !strings.Contains(err.Error(), clientpb.CrackstationCapabilityCrackQueryV1) {
		t.Fatalf("explicit unsupported station error = %v, want FailedPrecondition naming capability", err)
	}

	queryEntry, err := reserveStandaloneCrackQueryTask(t.Context(), models.NewUUID(), &models.CrackCommand{
		HashType: int32(clientpb.HashType_MD5), HashInfo: true,
	}, clientpb.CrackQueryMode_CRACK_QUERY_HASH_INFO, "")
	if err != nil {
		t.Fatalf("reserve generic query: %v", err)
	}
	queryAssignment := receiveStandaloneAssignment(t, capable, consts.CrackQuery)
	if queryEntry.task.HostUUID != capable.HostUUID || queryAssignment.HostUUID != capable.HostUUID {
		t.Fatalf("generic query selected %q/%q, want capable station %q", queryEntry.task.HostUUID, queryAssignment.HostUUID, capable.HostUUID)
	}
	select {
	case event := <-legacy.Events:
		t.Fatalf("generic query was sent to legacy station: %#v", event)
	default:
	}

	keyspaceEntry, err := reserveStandaloneCrackQueryTask(t.Context(), models.NewUUID(), &models.CrackCommand{
		AttackMode: int32(clientpb.CrackAttackMode_BRUTEFORCE), Keyspace: true, PositionalArguments: []string{"?d"},
	}, clientpb.CrackQueryMode_CRACK_QUERY_KEYSPACE, legacy.HostUUID)
	if err != nil {
		t.Fatalf("reserve legacy keyspace query: %v", err)
	}
	keyspaceAssignment := receiveStandaloneAssignment(t, legacy, consts.CrackKeyspace)
	if keyspaceEntry.task.HostUUID != legacy.HostUUID || keyspaceAssignment.HostUUID != legacy.HostUUID {
		t.Fatalf("keyspace selected %q/%q, want legacy station %q", keyspaceEntry.task.HostUUID, keyspaceAssignment.HostUUID, legacy.HostUUID)
	}
}

func TestStandaloneCrackQueryResultValidation(t *testing.T) {
	for _, mode := range []clientpb.CrackQueryMode{
		clientpb.CrackQueryMode_CRACK_QUERY_LOOKUP,
		clientpb.CrackQueryMode_CRACK_QUERY_IDENTIFY,
		clientpb.CrackQueryMode_CRACK_QUERY_HASH_INFO,
	} {
		if _, err := standaloneCrackQueryValue(mode, &clientpb.CrackTask{}); err == nil {
			t.Fatalf("%s accepted empty stdout", mode)
		}
	}
	if _, err := standaloneCrackQueryValue(clientpb.CrackQueryMode_CRACK_QUERY_TOTAL_CANDIDATES, &clientpb.CrackTask{Stdout: []byte("01")}); err == nil {
		t.Fatal("total-candidates accepted a noncanonical count")
	}
	if _, err := standaloneCrackQueryValue(clientpb.CrackQueryMode_CRACK_QUERY_LOOKUP, &clientpb.CrackTask{Stdout: []byte("value"), Stderr: []byte{0xff}}); err == nil {
		t.Fatal("query accepted invalid UTF-8 stderr")
	}
}

func TestStandaloneCrackQueryMalformedTerminalResultIsDataLossAndReleasesStation(t *testing.T) {
	database := setupCrackstationRPCTestDB(t)
	resetStandaloneCrackKeyspaceTasksForTest(t)
	disableCrackQueueReaperForTest(t)
	station := addQueueTestStation(t, database, "11111111-1111-4111-8111-111111111111", nil)
	station.Station.Capabilities = []string{clientpb.CrackstationCapabilityCrackQueryV1}
	setStandaloneKeyspaceTestStatus(station, station.HostUUID, clientpb.States_IDLE, false, "")
	rpcServer := &Server{}
	result := make(chan error, 1)
	go func() {
		_, err := rpcServer.Crack(t.Context(), &clientpb.CrackCommand{
			AttackMode: clientpb.CrackAttackMode_BRUTEFORCE, HashType: clientpb.HashType_INVALID,
			TotalCandidates: true, PositionalArguments: []string{"?d"},
		})
		result <- err
	}()

	assignment := receiveStandaloneAssignment(t, station, consts.CrackQuery)
	fetched, err := rpcServer.CrackTaskByID(crackstationTestContext("queue-test"), &clientpb.CrackTask{
		ID: assignment.TaskID, HostUUID: assignment.HostUUID, Attempt: assignment.Attempt, LeaseToken: assignment.LeaseToken,
	})
	if err != nil {
		t.Fatalf("fetch query task: %v", err)
	}
	completed := proto.Clone(fetched).(*clientpb.CrackTask)
	completed.Command = nil
	completed.State = clientpb.CrackTaskState_CRACK_TASK_COMPLETED
	completed.CompletedAt = time.Now().Unix()
	completed.Stdout = []byte("01")
	if _, err := rpcServer.CrackTaskUpdate(crackstationTestContext("queue-test"), completed); err != nil {
		t.Fatalf("malformed worker terminal update was not acknowledged: %v", err)
	}
	if err := <-result; status.Code(err) != codes.DataLoss {
		t.Fatalf("waiting query error = %v, want DataLoss", err)
	}

	crackQueueMu.Lock()
	_, stillReserved := standaloneCrackKeyspaceTasks[assignment.TaskID]
	available, selectErr := selectIdleCrackstationLocked(t.Context())
	crackQueueMu.Unlock()
	if stillReserved || selectErr != nil || available != station {
		t.Fatalf("malformed result reservation=%v available=%p want=%p err=%v", stillReserved, available, station, selectErr)
	}
}

func uint32Pointer(value uint32) *uint32 {
	return &value
}
