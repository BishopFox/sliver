package rpc

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/bishopfox/sliver/server/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

var (
	crackCommandRPCLog     = log.NamedLogger("rpc", "crack")
	errInvalidCrackCommand = errors.New("invalid crack command")
	errActiveCrackJob      = errors.New("cannot delete an active crack job")
	errTerminalCrackJob    = errors.New("crack job is already terminal")
	errCrackJobNotPaused   = errors.New("crack job is not paused")
)

type crackJobLifecycleAction string

const (
	crackJobLifecycleCancel crackJobLifecycleAction = "cancel"
	crackJobLifecyclePause  crackJobLifecycleAction = "pause"
	crackJobLifecycleResume crackJobLifecycleAction = "resume"
)

func managedCrackFileURI(crackFile *models.CrackFile) string {
	typeName := ""
	switch clientpb.CrackFileType(crackFile.Type) {
	case clientpb.CrackFileType_WORDLIST:
		typeName = "wordlist"
	case clientpb.CrackFileType_RULES:
		typeName = "rules"
	case clientpb.CrackFileType_MARKOV_HCSTAT2:
		typeName = "hcstat2"
	default:
		return ""
	}
	return fmt.Sprintf("crackfile://%s/%s", typeName, strings.ToLower(crackFile.Sha2_256))
}

func crackFileTypeName(fileType clientpb.CrackFileType) string {
	switch fileType {
	case clientpb.CrackFileType_WORDLIST:
		return "wordlist"
	case clientpb.CrackFileType_RULES:
		return "rules"
	case clientpb.CrackFileType_MARKOV_HCSTAT2:
		return "hcstat2"
	default:
		return ""
	}
}

//nolint:gocyclo // The accepted URI, digest, and legacy-reference forms share one type-safe resolution boundary.
func resolveManagedCrackFileReference(files []models.CrackFile, value string, expectedType clientpb.CrackFileType) (string, bool, error) {
	if value == "" {
		return value, false, nil
	}
	if strings.HasPrefix(value, "crackfile://") {
		parts := strings.Split(strings.TrimPrefix(value, "crackfile://"), "/")
		if len(parts) != 2 || parts[0] != crackFileTypeName(expectedType) || len(parts[1]) != 64 {
			return "", false, fmt.Errorf("invalid managed crack file reference %q", value)
		}
		if _, err := hex.DecodeString(parts[1]); err != nil {
			return "", false, fmt.Errorf("invalid managed crack file reference %q", value)
		}
		for index := range files {
			if clientpb.CrackFileType(files[index].Type) == expectedType && strings.EqualFold(files[index].Sha2_256, parts[1]) {
				return managedCrackFileURI(&files[index]), true, nil
			}
		}
		return "", false, fmt.Errorf("unknown managed crack file reference %q", value)
	}

	matches := map[string]struct{}{}
	for index := range files {
		file := &files[index]
		if clientpb.CrackFileType(file.Type) != expectedType {
			continue
		}
		if value == file.Name || value == file.ID.String() || strings.EqualFold(value, file.Sha2_256) {
			matches[managedCrackFileURI(file)] = struct{}{}
		}
	}
	if len(matches) == 0 {
		return value, false, nil
	}
	if len(matches) > 1 {
		return "", false, fmt.Errorf("managed crack file reference %q is ambiguous", value)
	}
	for uri := range matches {
		return uri, true, nil
	}
	return value, false, nil
}

func effectiveCrackHashType(command *models.CrackCommand) int32 {
	if command.HashMode != nil {
		return int32(*command.HashMode)
	}
	if command.HashType == int32(clientpb.HashType_INVALID) {
		return int32(clientpb.HashType_MD5)
	}
	return command.HashType
}

func validateCrackHashes(hashes []string) error {
	for _, hash := range hashes {
		if strings.TrimSpace(hash) == "" {
			return errors.New("hash cannot be empty")
		}
		if strings.ContainsAny(hash, "\r\n\x00") {
			return errors.New("hash cannot contain a line break or NUL")
		}
	}
	return nil
}

//nolint:gocyclo // Mutually dependent Hashcat options are centralized so the distributed-command contract stays explicit.
func validateDistributedCrackCommand(command *models.CrackCommand) error {
	switch {
	case command.HashMode != nil && *command.HashMode > math.MaxInt32:
		return errors.New("hash mode exceeds the supported range")
	case command.HashMode == nil && command.HashType < 0:
		return errors.New("hash type cannot be negative")
	case command.Separator != "" && (len(command.Separator) != 1 || strings.ContainsAny(command.Separator, "\x00\r\n")):
		return errors.New("separator must be exactly one non-NUL byte without a line break")
	case strings.ContainsRune(command.RuleLeft, '\x00') || strings.ContainsRune(command.RuleRight, '\x00') ||
		strings.ContainsRune(command.GenerateRulesFuncSel, '\x00') || strings.ContainsRune(command.EncodingFromName, '\x00') ||
		strings.ContainsRune(command.EncodingToName, '\x00'):
		return errors.New("hashcat option cannot contain NUL")
	case command.Username:
		return errors.New("username mode cannot be distributed")
	case command.KeepGuessing:
		return errors.New("keep-guessing mode cannot be distributed")
	case command.Runtime != 0:
		return errors.New("runtime limit cannot be distributed")
	case command.Session != "":
		return errors.New("session files cannot be distributed")
	case command.Remove:
		return errors.New("hash removal cannot be distributed")
	case command.RemoveTimer != 0:
		return errors.New("hash removal timer cannot be distributed")
	case command.SegmentSize != 0:
		return errors.New("segment size is not supported by managed hashcat tasks")
	case command.GenerateRulesSeed < 0 && command.GenerateRulesSeedV7 == nil:
		return errors.New("generated-rules seed cannot be negative")
	case command.Loopback:
		return errors.New("loopback mode cannot be distributed")
	case command.InductionDir != "":
		return errors.New("induction directory cannot be distributed")
	case command.BrainFeed:
		return errors.New("brain feed mode cannot be distributed")
	case command.BrainClient:
		return errors.New("brain client mode cannot be distributed")
	case command.BrainServerTimer != 0 || command.BrainClientFeatures != "" || command.BrainHost != "" ||
		command.BrainPort != 0 || command.BrainPassword != "" || command.BrainSession != "" || command.BrainSessionWhitelist != "" ||
		command.BrainClientFeaturesV7 != 0 || command.BrainSessionV7 != nil || len(command.BrainSessionWhitelistV7) != 0 ||
		command.BrainServerTimerV7 != nil || command.BrainPasswordV7 != nil:
		return errors.New("brain configuration cannot be distributed")
	case command.EncryptWithPubkey != "":
		return errors.New("public-key encryption mode cannot be distributed")
	case command.OutfileCheckDir != "":
		return errors.New("outfile-check directory cannot be distributed")
	case (command.OutfileCheckTimerV7 != nil && *command.OutfileCheckTimerV7 != 0) ||
		(command.OutfileCheckTimerV7 == nil && command.OutfileCheckTimer != 0):
		return errors.New("outfile-check timer cannot be enabled for a distributed job")
	case command.MachineReadable:
		return errors.New("machine-readable status mode cannot be distributed")
	case command.DebugMode != 0:
		return errors.New("debug mode cannot be distributed")
	case command.DebugFile != "":
		return errors.New("debug output file cannot be distributed")
	case command.SeekDBPath != "":
		return errors.New("seek database path cannot be distributed")
	case command.BridgeParameter1 != "" || command.BridgeParameter2 != "" || command.BridgeParameter3 != "" || command.BridgeParameter4 != "":
		return errors.New("bridge parameters cannot be distributed")
	case command.HwmonDisable:
		return errors.New("hardware monitoring cannot be disabled for a distributed job")
	case command.TruecryptKeyfiles != "":
		return errors.New("TrueCrypt keyfiles cannot be distributed")
	case command.VeracryptKeyfiles != "":
		return errors.New("VeraCrypt keyfiles cannot be distributed")
	case command.Benchmark:
		return errors.New("benchmark mode cannot be queued as a crack job")
	case command.BenchmarkAll:
		return errors.New("benchmark-all mode cannot be queued as a crack job")
	case command.BenchmarkMin != 0:
		return errors.New("benchmark-min cannot be queued as a crack job")
	case command.BenchmarkMax != nil:
		return errors.New("benchmark-max cannot be queued as a crack job")
	case command.SpeedOnly:
		return errors.New("speed-only mode cannot be queued as a crack job")
	case command.ProgressOnly:
		return errors.New("progress-only mode cannot be queued as a crack job")
	case command.Stdout:
		return errors.New("stdout mode cannot be queued as a crack job")
	case command.Show:
		return errors.New("show mode cannot be queued as a crack job")
	case command.Left:
		return errors.New("left mode cannot be queued as a crack job")
	case command.HashInfo || command.HashInfoLevel != 0:
		return errors.New("hash-info mode cannot be queued as a crack job")
	case command.BackendInfo || command.BackendInfoLevel != 0:
		return errors.New("backend-info mode cannot be queued as a crack job")
	case command.Keyspace:
		return errors.New("keyspace mode is managed by the crack queue")
	case command.TotalCandidates:
		return errors.New("total-candidates mode cannot be queued as a crack job")
	case command.Lookup != "":
		return errors.New("lookup mode cannot be queued as a crack job")
	case command.IdentifyMode:
		return errors.New("identify mode cannot be queued as a crack job")
	case command.BrainServer:
		return errors.New("brain server mode cannot be queued as a crack job")
	case command.Restore:
		return errors.New("restore mode cannot be distributed")
	case command.RestorePosition:
		return errors.New("restore position cannot be distributed")
	case command.RestoreShowCommand:
		return errors.New("restore-show mode cannot be queued as a crack job")
	case len(command.RestoreFile) != 0:
		return errors.New("restore file cannot be distributed")
	case len(command.Stdin) != 0:
		return errors.New("stdin cannot be distributed")
	}
	return nil
}

func normalizeDistributedCrackCommand(command *models.CrackCommand, jobID models.UUID) {
	if len(command.PositionalArguments) != 0 {
		command.Identify = ""
	}
	if len(command.RulesFilesV7) != 0 {
		command.RulesFile = nil
	}
	if command.GenerateRulesSeedV7 != nil {
		command.GenerateRulesSeed = 0
	}
	command.RestoreDisable = true
	command.LogfileDisable = true
	command.HashCopy = true
	if len(command.MarkovHcstat2) == 0 {
		// Hashcat's built-in hcstat2 data is installation-version dependent.
		// Disable Markov ordering unless the job pins a managed hcstat2 file.
		command.MarkovDisable = true
	}
	command.OutfileCheckTimer = 0
	disabledOutfileCheckTimer := uint32(0)
	command.OutfileCheckTimerV7 = &disabledOutfileCheckTimer
	if command.GenerateRules != 0 && command.GenerateRulesSeedV7 == nil {
		seed := binary.LittleEndian.Uint32(jobID[:4])
		command.GenerateRulesSeedV7 = &seed
	}
}

//nolint:gocyclo // Legacy and v7 file fields are resolved together before the command is persisted.
func resolveManagedCrackFiles(tx *gorm.DB, command *models.CrackCommand) error {
	var files []models.CrackFile
	if err := tx.Where("is_complete = ?", true).Find(&files).Error; err != nil {
		return err
	}
	for index, argument := range command.PositionalArguments {
		if !distributedCrackOperandIsWordlist(command.AttackMode, index) && !strings.HasPrefix(argument, "crackfile://") {
			continue
		}
		uri, matched, err := resolveManagedCrackFileReference(files, argument, clientpb.CrackFileType_WORDLIST)
		if err != nil {
			return err
		}
		if matched {
			command.PositionalArguments[index] = uri
		}
	}
	if distributedCrackOperandIsWordlist(command.AttackMode, 0) || strings.HasPrefix(command.Identify, "crackfile://") {
		uri, matched, err := resolveManagedCrackFileReference(files, command.Identify, clientpb.CrackFileType_WORDLIST)
		if err != nil {
			return err
		}
		if matched {
			command.Identify = uri
		}
	}
	uri, matched, err := resolveManagedCrackFileReference(files, string(command.RulesFile), clientpb.CrackFileType_RULES)
	if err != nil {
		return err
	}
	if matched {
		command.RulesFile = []byte(uri)
	}
	for index, rules := range command.RulesFilesV7 {
		uri, matched, err := resolveManagedCrackFileReference(files, string(rules), clientpb.CrackFileType_RULES)
		if err != nil {
			return err
		}
		if matched {
			command.RulesFilesV7[index] = []byte(uri)
		}
	}
	uri, matched, err = resolveManagedCrackFileReference(files, string(command.MarkovHcstat2), clientpb.CrackFileType_MARKOV_HCSTAT2)
	if err != nil {
		return err
	}
	if matched {
		command.MarkovHcstat2 = []byte(uri)
	}
	return nil
}

func distributedCrackOperandIsWordlist(attackMode int32, index int) bool {
	switch clientpb.CrackAttackMode(attackMode) {
	case clientpb.CrackAttackMode_STRAIGHT, clientpb.CrackAttackMode_COMBINATION:
		return true
	case clientpb.CrackAttackMode_HYBRID_WORDLIST_MASK:
		return index == 0
	case clientpb.CrackAttackMode_HYBRID_MASK_WORDLIST:
		return index == 1
	default:
		return false
	}
}

func distributedCrackOperands(command *models.CrackCommand) []string {
	if len(command.PositionalArguments) != 0 {
		return command.PositionalArguments
	}
	if command.Identify != "" {
		return []string{command.Identify}
	}
	return nil
}

func isManagedWordlistReference(value string) bool {
	return strings.HasPrefix(value, "crackfile://wordlist/")
}

//nolint:gocyclo // This defensive classifier intentionally enumerates unsafe path forms from every supported platform.
func unsafeDistributedCandidatePath(value string) bool {
	if value == "" {
		return false
	}
	if strings.ContainsAny(value, "\x00\r\n") || value[0] == '/' || value[0] == '\\' {
		return true
	}
	if len(value) >= 2 && value[1] == ':' && ((value[0] >= 'a' && value[0] <= 'z') || (value[0] >= 'A' && value[0] <= 'Z')) {
		return true
	}
	for _, component := range strings.FieldsFunc(value, func(r rune) bool { return r == '/' || r == '\\' }) {
		withoutSpaces := strings.TrimRight(component, " ")
		if withoutSpaces == "." || withoutSpaces == ".." || strings.Trim(withoutSpaces, ".") == "" {
			return true
		}
		deviceName := strings.TrimRight(component, ". ")
		if deviceName == "" {
			return true
		}
		if index := strings.IndexAny(deviceName, ".:"); index >= 0 {
			deviceName = deviceName[:index]
		}
		upper := strings.ToUpper(deviceName)
		switch upper {
		case "CON", "PRN", "AUX", "NUL", "CONIN$", "CONOUT$", "CLOCK$":
			return true
		}
		if len(upper) == 4 && (strings.HasPrefix(upper, "COM") || strings.HasPrefix(upper, "LPT")) && upper[3] >= '1' && upper[3] <= '9' {
			return true
		}
	}
	return false
}

func validateInlineMask(value string) error {
	if isManagedWordlistReference(value) {
		return nil
	}
	if unsafeDistributedCandidatePath(value) || strings.HasPrefix(value, "crackfile://") {
		return fmt.Errorf("mask operand %q must be an inline mask, not a crackstation-local path", value)
	}
	return nil
}

func requireManagedWordlists(operands []string) error {
	for _, operand := range operands {
		if !isManagedWordlistReference(operand) {
			return fmt.Errorf("wordlist operand %q is not a managed crack file", operand)
		}
	}
	return nil
}

func validateDistributedCrackOperands(command *models.CrackCommand) error {
	operands := distributedCrackOperands(command)
	switch clientpb.CrackAttackMode(command.AttackMode) {
	case clientpb.CrackAttackMode_STRAIGHT:
		if len(operands) < 1 {
			return errors.New("straight attack requires at least one managed wordlist")
		}
		return requireManagedWordlists(operands)
	case clientpb.CrackAttackMode_COMBINATION:
		if len(operands) != 2 {
			return errors.New("combination attack requires exactly two managed wordlists")
		}
		return requireManagedWordlists(operands)
	case clientpb.CrackAttackMode_BRUTEFORCE:
		if len(operands) != 1 {
			return errors.New("brute-force attack requires exactly one mask")
		}
		return validateInlineMask(operands[0])
	case clientpb.CrackAttackMode_HYBRID_WORDLIST_MASK:
		if len(operands) != 2 {
			return errors.New("wordlist-mask hybrid requires exactly one managed wordlist and one mask")
		}
		if err := requireManagedWordlists(operands[:len(operands)-1]); err != nil {
			return err
		}
		return validateInlineMask(operands[len(operands)-1])
	case clientpb.CrackAttackMode_HYBRID_MASK_WORDLIST:
		if len(operands) != 2 {
			return errors.New("mask-wordlist hybrid requires exactly one mask and one managed wordlist")
		}
		if err := validateInlineMask(operands[0]); err != nil {
			return err
		}
		return requireManagedWordlists(operands[1:])
	case clientpb.CrackAttackMode_PCFG,
		clientpb.CrackAttackMode_GENERIC,
		clientpb.CrackAttackMode_ASSOCIATION,
		clientpb.CrackAttackMode_NO_ATTACK,
		clientpb.CrackAttackMode_HYBRID:
		return fmt.Errorf("attack mode %s is not supported by the distributed crack queue", clientpb.CrackAttackMode(command.AttackMode))
	default:
		return fmt.Errorf("unknown distributed attack mode %d", command.AttackMode)
	}
}

func validateDistributedCrackFileFields(command *models.CrackCommand) error {
	if len(command.RulesFile) != 0 && !strings.HasPrefix(string(command.RulesFile), "crackfile://rules/") {
		return errors.New("rules file is not a managed crack file")
	}
	for _, rulesFile := range command.RulesFilesV7 {
		if len(rulesFile) != 0 && !strings.HasPrefix(string(rulesFile), "crackfile://rules/") {
			return errors.New("rules file is not a managed crack file")
		}
	}
	if len(command.MarkovHcstat2) != 0 && !strings.HasPrefix(string(command.MarkovHcstat2), "crackfile://hcstat2/") {
		return errors.New("Markov hcstat2 file is not a managed crack file") //nolint:staticcheck // Preserve the established operator-facing error text.
	}
	customCharsets := []string{
		command.CustomCharset1, command.CustomCharset2, command.CustomCharset3, command.CustomCharset4,
		command.CustomCharset5, command.CustomCharset6, command.CustomCharset7, command.CustomCharset8,
	}
	for index, charset := range customCharsets {
		if charset == "" {
			continue
		}
		if unsafeDistributedCandidatePath(charset) || strings.HasPrefix(charset, "crackfile://") {
			return fmt.Errorf("custom charset %d must be inline, not a crackstation-local path", index+1)
		}
	}
	return nil
}

func crackCommandReferencesManagedFile(command *models.CrackCommand, uri string) bool {
	if command == nil || uri == "" {
		return false
	}
	for _, argument := range command.PositionalArguments {
		if argument == uri {
			return true
		}
	}
	if command.Identify == uri || string(command.RulesFile) == uri || string(command.MarkovHcstat2) == uri {
		return true
	}
	for _, rulesFile := range command.RulesFilesV7 {
		if string(rulesFile) == uri {
			return true
		}
	}
	return false
}

func activeCrackJobReferencesManagedFile(tx *gorm.DB, crackFile *models.CrackFile) (bool, error) {
	uri := managedCrackFileURI(crackFile)
	if uri == "" || crackFile.Sha2_256 == "" {
		return false, nil
	}
	if activeStandaloneCrackKeyspaceReferencesManagedFile(uri) {
		return true, nil
	}
	var activeJobIDs []models.UUID
	if err := tx.Model(&models.CrackJob{}).Where("completed_at = ?", time.Time{}).Pluck("id", &activeJobIDs).Error; err != nil {
		return false, err
	}
	if len(activeJobIDs) == 0 {
		return false, nil
	}
	var commands []models.CrackCommand
	if err := tx.Where("crack_job_id IN ?", activeJobIDs).Find(&commands).Error; err != nil {
		return false, err
	}
	for index := range commands {
		if crackCommandReferencesManagedFile(&commands[index], uri) {
			return true, nil
		}
	}
	return false, nil
}

//nolint:gocyclo // Explicit IDs and optional credential filters must be combined with stable deduplication semantics.
func selectCrackCredentials(tx *gorm.DB, command *models.CrackCommand) ([]models.Credential, error) {
	selected := map[models.UUID]models.Credential{}
	for _, idText := range command.CredentialIDs {
		if strings.TrimSpace(idText) == "" {
			return nil, errors.New("credential id cannot be empty")
		}
		id := models.ParseUUIDOrNil(idText)
		credential := models.Credential{}
		var lookupErr error
		if id != models.NilUUID() {
			lookupErr = tx.First(&credential, "id = ?", id).Error
		} else {
			var credentials []models.Credential
			lookupErr = tx.Find(&credentials).Error
			if lookupErr == nil {
				matches := []models.Credential{}
				for _, candidate := range credentials {
					if strings.HasPrefix(candidate.ID.String(), idText) {
						matches = append(matches, candidate)
					}
				}
				if len(matches) == 1 {
					credential = matches[0]
				} else if len(matches) == 0 {
					lookupErr = gorm.ErrRecordNotFound
				} else {
					return nil, fmt.Errorf("credential id prefix %q is ambiguous", idText)
				}
			}
		}
		if lookupErr != nil {
			if errors.Is(lookupErr, gorm.ErrRecordNotFound) {
				return nil, fmt.Errorf("credential %q was not found", idText)
			}
			return nil, lookupErr
		}
		selected[credential.ID] = credential
	}
	if command.CredentialCollection != "" {
		var credentials []models.Credential
		if err := tx.Where("collection = ?", command.CredentialCollection).Find(&credentials).Error; err != nil {
			return nil, err
		}
		for _, credential := range credentials {
			selected[credential.ID] = credential
		}
	}

	credentials := make([]models.Credential, 0, len(selected))
	for _, credential := range selected {
		if credential.Hash == "" || (credential.IsCracked && !command.IncludeCrackedCredentials) {
			continue
		}
		if credential.HashType != effectiveCrackHashType(command) {
			return nil, fmt.Errorf("credential %s has hash type %d, not %d", credential.ID, credential.HashType, effectiveCrackHashType(command))
		}
		credentials = append(credentials, credential)
	}
	sort.Slice(credentials, func(i, j int) bool { return credentials[i].ID.String() < credentials[j].ID.String() })
	seenHashes := map[string]struct{}{}
	for _, hash := range command.Hashes {
		seenHashes[hash] = struct{}{}
	}
	for _, credential := range credentials {
		if _, exists := seenHashes[credential.Hash]; exists {
			continue
		}
		command.Hashes = append(command.Hashes, credential.Hash)
		seenHashes[credential.Hash] = struct{}{}
	}
	return credentials, nil
}

// Crack validates a distributed cracking request, persists its job, and schedules work.
//
//nolint:gocyclo // Validation, resolution, job creation, and scheduling make up one transactional RPC boundary.
func (rpc *Server) Crack(ctx context.Context, req *clientpb.CrackCommand) (*clientpb.CrackResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "missing crack command")
	}
	if proto.Size(req) > 16<<20 {
		return nil, status.Error(codes.ResourceExhausted, "crack command exceeds size limit")
	}
	commandRequest := proto.Clone(req).(*clientpb.CrackCommand)
	selector := strings.TrimSpace(commandRequest.GetCrackstation())
	// Crackstation is routing metadata for synchronous queries. Never persist
	// it in a CrackCommand or send it back to a worker.
	commandRequest.Crackstation = ""
	command := models.CrackCommand{}.FromProtobuf(commandRequest)
	if isStandaloneCrackQueryCommand(command) {
		result, err := runStandaloneCrackQuery(ctx, command, selector)
		if err != nil {
			return nil, err
		}
		response := &clientpb.CrackResponse{Query: &clientpb.CrackQueryResult{
			Mode:                 result.Mode,
			CrackstationHostUUID: result.CrackstationHostUUID,
			CrackstationName:     result.CrackstationName,
			HashcatVersion:       result.HashcatVersion,
			Value:                result.Value,
			Stderr:               result.Stderr,
		}}
		if result.Mode == clientpb.CrackQueryMode_CRACK_QUERY_KEYSPACE {
			response.Keyspace = result.Value
		}
		return response, nil
	}
	if selector != "" {
		return nil, status.Error(codes.InvalidArgument, "crackstation selector is only valid for synchronous crack queries")
	}
	if err := validateDistributedCrackCommand(command); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid crack command: %s", err)
	}
	jobID := models.NewUUID()
	normalizeDistributedCrackCommand(command, jobID)
	crackFileLifecycleMu.Lock()
	defer crackFileLifecycleMu.Unlock()
	now := time.Now()
	job := &models.CrackJob{ID: jobID, UpdatedAt: now}
	var task *models.CrackTask

	err := db.Session().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := resolveManagedCrackFiles(tx, command); err != nil {
			return fmt.Errorf("%w: %v", errInvalidCrackCommand, err)
		}
		if err := validateDistributedCrackFileFields(command); err != nil {
			return fmt.Errorf("%w: %v", errInvalidCrackCommand, err)
		}
		if err := validateDistributedCrackOperands(command); err != nil {
			return fmt.Errorf("%w: %v", errInvalidCrackCommand, err)
		}
		if err := validateCrackHashes(command.Hashes); err != nil {
			return fmt.Errorf("%w: %v", errInvalidCrackCommand, err)
		}
		credentials, err := selectCrackCredentials(tx, command)
		if err != nil {
			return fmt.Errorf("%w: %v", errInvalidCrackCommand, err)
		}
		if err := validateCrackHashes(command.Hashes); err != nil {
			return fmt.Errorf("%w: %v", errInvalidCrackCommand, err)
		}
		if len(command.Hashes) == 0 {
			return fmt.Errorf("%w: no hashes selected", errInvalidCrackCommand)
		}
		if err := tx.Create(job).Error; err != nil {
			return err
		}
		command.CrackJobID = jobID
		if err := tx.Create(command).Error; err != nil {
			return err
		}
		for _, credential := range credentials {
			association := &models.CrackJobCredential{CrackJobID: jobID, CredentialID: credential.ID}
			if err := tx.Create(association).Error; err != nil {
				return err
			}
		}
		task = &models.CrackTask{
			CrackJobID: jobID,
			Kind:       int32(clientpb.CrackTaskKind_CRACK_TASK_KEYSPACE),
			State:      int32(clientpb.CrackTaskState_CRACK_TASK_QUEUED),
			UpdatedAt:  now,
		}
		if err := tx.Create(task).Error; err != nil {
			return err
		}
		taskCommand := *command
		taskCommand.ID = models.NilUUID()
		taskCommand.CreatedAt = time.Time{}
		taskCommand.CrackTaskID = task.ID
		taskCommand.CrackJobID = models.NilUUID()
		taskCommand.Keyspace = true
		taskCommand.Quiet = true
		taskCommand.Hashes = nil
		taskCommand.Skip = 0
		taskCommand.Limit = 0
		if err := tx.Create(&taskCommand).Error; err != nil {
			return err
		}
		task.Command = taskCommand
		return nil
	})
	if err != nil {
		if errors.Is(err, errInvalidCrackCommand) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		crackCommandRPCLog.Errorf("Failed to save crack job: %s", err)
		return nil, status.Error(codes.Internal, "failed to save crack job")
	}

	job.Command = *command
	job.Tasks = []models.CrackTask{*task}
	core.EventBroker.Publish(core.Event{EventType: consts.CrackJobCreated, Data: []byte(job.ID.String())})
	crackQueueReaperStarter()
	if err := scheduleCrackTasks(); err != nil {
		crackCommandRPCLog.Warnf("Failed to schedule crack job %s: %s", job.ID, err)
	}
	return &clientpb.CrackResponse{Job: job.ToProtobuf()}, nil
}

func loadCrackJob(tx *gorm.DB, id models.UUID) (*models.CrackJob, error) {
	job := &models.CrackJob{}
	err := tx.Preload("Command").Preload("Tasks.Command").Preload("Results").First(job, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	sort.Slice(job.Tasks, func(i, j int) bool { return job.Tasks[i].CreatedAt.Before(job.Tasks[j].CreatedAt) })
	sort.Slice(job.Results, func(i, j int) bool { return job.Results[i].CreatedAt.Before(job.Results[j].CreatedAt) })
	return job, nil
}

// CrackJobs lists recent distributed cracking jobs without exposing task internals.
func (rpc *Server) CrackJobs(ctx context.Context, _ *commonpb.Empty) (*clientpb.CrackJobs, error) {
	var jobs []models.CrackJob
	dbSession := db.Session().WithContext(ctx)
	if err := dbSession.Preload("Command").Preload("Tasks").Order("created_at desc").Limit(100).Find(&jobs).Error; err != nil {
		return nil, status.Error(codes.Internal, "failed to list crack jobs")
	}
	jobIDs := make([]models.UUID, 0, len(jobs))
	for index := range jobs {
		jobIDs = append(jobIDs, jobs[index].ID)
	}
	if len(jobIDs) != 0 {
		var counts []struct {
			CrackJobID  models.UUID
			ResultCount uint64
		}
		if err := dbSession.Model(&models.CrackResult{}).
			Select("crack_job_id, COUNT(*) AS result_count").
			Where("crack_job_id IN ?", jobIDs).
			Group("crack_job_id").Scan(&counts).Error; err != nil {
			return nil, status.Error(codes.Internal, "failed to count crack job results")
		}
		countsByJobID := make(map[models.UUID]uint64, len(counts))
		for _, count := range counts {
			countsByJobID[count.CrackJobID] = count.ResultCount
		}
		for index := range jobs {
			jobs[index].ResultCount = countsByJobID[jobs[index].ID]
		}
	}
	response := &clientpb.CrackJobs{Jobs: make([]*clientpb.CrackJob, 0, len(jobs))}
	for index := range jobs {
		job := jobs[index].ToProtobuf()
		job.Command = nil
		job.Results = nil
		for _, task := range job.Tasks {
			task.Command = nil
			task.Stdout = nil
			task.Stderr = nil
			task.RecoveredJSON = nil
			task.LatestStatusJSON = nil
			task.LeaseToken = ""
		}
		response.Jobs = append(response.Jobs, job)
	}
	return response, nil
}

// CrackJobByID returns one distributed cracking job with sensitive task fields redacted.
func (rpc *Server) CrackJobByID(ctx context.Context, req *clientpb.CrackJob) (*clientpb.CrackJob, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "missing crack job")
	}
	id := models.ParseUUIDOrNil(req.ID)
	if id == models.NilUUID() {
		return nil, status.Error(codes.InvalidArgument, "invalid crack job id")
	}
	job, err := loadCrackJob(db.Session().WithContext(ctx), id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, status.Error(codes.NotFound, "crack job not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to load crack job")
	}
	response := job.ToProtobuf()
	for _, task := range response.Tasks {
		task.LeaseToken = ""
	}
	return response, nil
}

func lifecycleCrackJobID(req *clientpb.CrackJob) (models.UUID, error) {
	if req == nil {
		return models.NilUUID(), status.Error(codes.InvalidArgument, "missing crack job")
	}
	jobID := models.ParseUUIDOrNil(req.ID)
	if jobID == models.NilUUID() {
		return models.NilUUID(), status.Error(codes.InvalidArgument, "invalid crack job id")
	}
	return jobID, nil
}

func loadLifecycleCrackJobSummary(tx *gorm.DB, id models.UUID) (*models.CrackJob, error) {
	job := &models.CrackJob{}
	err := tx.Select("id", "created_at", "updated_at", "completed_at", "paused_at", "cancelled_at", "err").
		First(job, "id = ?", id).Error
	return job, err
}

// lifecycleCrackJobSummary deliberately returns only bounded status metadata.
// A lifecycle mutation must not preload or serialize task output, recovered
// plaintexts, hashes, or command payloads merely to confirm its new state.
func lifecycleCrackJobSummary(job *models.CrackJob) *clientpb.CrackJob {
	response := &clientpb.CrackJob{
		ID:        job.ID.String(),
		CreatedAt: job.CreatedAt.UTC().Format(time.RFC3339),
		Status:    job.Status(),
	}
	if !job.UpdatedAt.IsZero() {
		response.UpdatedAt = job.UpdatedAt.Unix()
	}
	if !job.CompletedAt.IsZero() {
		response.CompletedAt = job.CompletedAt.UTC().Format(time.RFC3339)
	}
	return response
}

func lifecycleCrackTaskDispatch(task *models.CrackTask, action crackJobLifecycleAction) crackTaskCancelDispatch {
	hostUUID := task.CrackstationID.String()
	return crackTaskCancelDispatch{
		request: crackTaskCancelRequest{
			TaskID:     task.ID.String(),
			CrackJobID: task.CrackJobID.String(),
			HostUUID:   hostUUID,
			Attempt:    task.Attempt,
			LeaseToken: task.LeaseToken,
			Action:     string(action),
		},
		station: core.GetCrackstation(hostUUID),
	}
}

// transitionCrackJob applies one operator lifecycle mutation while holding the
// queue mutex. Remote station delivery happens only after the durable state and
// drain quarantine have committed and the mutex has been released.
func (rpc *Server) transitionCrackJob(ctx context.Context, req *clientpb.CrackJob, action crackJobLifecycleAction) (*clientpb.CrackJob, error) {
	jobID, err := lifecycleCrackJobID(req)
	if err != nil {
		return nil, err
	}

	dispatches := []crackTaskCancelDispatch{}
	changed := false
	var response *clientpb.CrackJob
	crackQueueMu.Lock()
	now := time.Now()
	loadResponse := func(tx *gorm.DB) error {
		job, err := loadLifecycleCrackJobSummary(tx, jobID)
		if err != nil {
			return err
		}
		response = lifecycleCrackJobSummary(job)
		return nil
	}
	err = db.Session().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		job := &models.CrackJob{}
		if err := tx.First(job, "id = ?", jobID).Error; err != nil {
			return err
		}

		switch action {
		case crackJobLifecyclePause:
			if !job.CompletedAt.IsZero() {
				return errTerminalCrackJob
			}
			if job.PausedAt.IsZero() {
				if err := tx.Model(job).Updates(map[string]interface{}{"paused_at": now, "updated_at": now}).Error; err != nil {
					return err
				}
				changed = true
			}
		case crackJobLifecycleResume:
			if !job.CompletedAt.IsZero() {
				return errTerminalCrackJob
			}
			if job.PausedAt.IsZero() {
				return errCrackJobNotPaused
			}
			if err := tx.Model(job).Updates(map[string]interface{}{"paused_at": time.Time{}, "updated_at": now}).Error; err != nil {
				return err
			}
			changed = true
		case crackJobLifecycleCancel:
			if !job.CancelledAt.IsZero() {
				return loadResponse(tx)
			}
			if !job.CompletedAt.IsZero() {
				return errTerminalCrackJob
			}
			if err := tx.Model(job).Updates(map[string]interface{}{
				"paused_at": time.Time{}, "cancelled_at": now, "completed_at": now, "updated_at": now,
			}).Error; err != nil {
				return err
			}
			changed = true
		default:
			return errors.New("invalid crack job lifecycle action")
		}

		if action == crackJobLifecycleResume {
			return loadResponse(tx)
		}
		var active []models.CrackTask
		if err := tx.Where("crack_job_id = ? AND state IN ?", jobID, []int32{
			int32(clientpb.CrackTaskState_CRACK_TASK_QUEUED),
			int32(clientpb.CrackTaskState_CRACK_TASK_LEASED),
			int32(clientpb.CrackTaskState_CRACK_TASK_RUNNING),
		}).Find(&active).Error; err != nil {
			return err
		}
		for index := range active {
			state := clientpb.CrackTaskState(active[index].State)
			if state == clientpb.CrackTaskState_CRACK_TASK_LEASED || state == clientpb.CrackTaskState_CRACK_TASK_RUNNING {
				dispatches = append(dispatches, lifecycleCrackTaskDispatch(&active[index], action))
			}
		}

		if action == crackJobLifecyclePause {
			if err := tx.Model(&models.CrackTask{}).
				Where("crack_job_id = ? AND state IN ?", jobID, []int32{
					int32(clientpb.CrackTaskState_CRACK_TASK_LEASED),
					int32(clientpb.CrackTaskState_CRACK_TASK_RUNNING),
				}).Updates(resetCrackTaskForRetryUpdates(now)).Error; err != nil {
				return err
			}
			return loadResponse(tx)
		}
		if err := tx.Model(&models.CrackTask{}).
			Where("crack_job_id = ? AND state IN ?", jobID, []int32{
				int32(clientpb.CrackTaskState_CRACK_TASK_QUEUED),
				int32(clientpb.CrackTaskState_CRACK_TASK_LEASED),
				int32(clientpb.CrackTaskState_CRACK_TASK_RUNNING),
			}).Updates(map[string]interface{}{
			"state":             int32(clientpb.CrackTaskState_CRACK_TASK_CANCELLED),
			"lease_token":       "",
			"lease_expires_at":  time.Time{},
			"last_heartbeat_at": time.Time{},
			"completed_at":      now,
			"updated_at":        now,
		}).Error; err != nil {
			return err
		}
		return loadResponse(tx)
	})
	if err == nil {
		registerCrackstationDrainsLocked(dispatches)
	}
	crackQueueMu.Unlock()

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, status.Error(codes.NotFound, "crack job not found")
	}
	if errors.Is(err, errTerminalCrackJob) || errors.Is(err, errCrackJobNotPaused) {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}
	if err != nil {
		crackCommandRPCLog.Errorf("Failed to %s crack job %s: %s", action, jobID, err)
		return nil, status.Errorf(codes.Internal, "failed to %s crack job", action)
	}
	if response == nil {
		return nil, status.Error(codes.Internal, "failed to summarize crack job lifecycle state")
	}

	dispatchCrackTaskCancellations(dispatches)
	if changed {
		core.EventBroker.Publish(core.Event{EventType: consts.CrackJobUpdated, Data: []byte(jobID.String())})
	}
	if err := scheduleCrackTasks(); err != nil {
		crackCommandRPCLog.Warnf("Crack job %s lifecycle action %s committed but follow-up scheduling failed: %s", jobID, action, err)
	}
	return response, nil
}

// CrackJobCancel permanently cancels all unfinished work for a distributed job.
func (rpc *Server) CrackJobCancel(ctx context.Context, req *clientpb.CrackJob) (*clientpb.CrackJob, error) {
	return rpc.transitionCrackJob(ctx, req, crackJobLifecycleCancel)
}

// CrackJobPause revokes active attempts and leaves their tasks queued behind a
// durable scheduling gate until CrackJobResume is called.
func (rpc *Server) CrackJobPause(ctx context.Context, req *clientpb.CrackJob) (*clientpb.CrackJob, error) {
	return rpc.transitionCrackJob(ctx, req, crackJobLifecyclePause)
}

// CrackJobResume clears the scheduling gate and makes queued work eligible for
// assignment again.
func (rpc *Server) CrackJobResume(ctx context.Context, req *clientpb.CrackJob) (*clientpb.CrackJob, error) {
	return rpc.transitionCrackJob(ctx, req, crackJobLifecycleResume)
}

// CrackJobDelete deletes a terminal distributed cracking job and its durable
// task, command, result, and credential-association records. The referenced
// credentials and managed crack files are intentionally preserved.
func (rpc *Server) CrackJobDelete(ctx context.Context, req *clientpb.CrackJob) (*commonpb.Empty, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "missing crack job")
	}
	jobID := models.ParseUUIDOrNil(req.ID)
	if jobID == models.NilUUID() {
		return nil, status.Error(codes.InvalidArgument, "invalid crack job id")
	}

	crackQueueMu.Lock()
	defer crackQueueMu.Unlock()

	err := db.Session().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		job := &models.CrackJob{}
		if err := tx.Select("id", "completed_at").First(job, "id = ?", jobID).Error; err != nil {
			return err
		}
		if job.CompletedAt.IsZero() {
			return errActiveCrackJob
		}

		var taskIDs []models.UUID
		if err := tx.Model(&models.CrackTask{}).Where("crack_job_id = ?", jobID).Pluck("id", &taskIDs).Error; err != nil {
			return err
		}
		if err := tx.Where("crack_job_id = ?", jobID).Delete(&models.CrackResult{}).Error; err != nil {
			return err
		}
		if err := tx.Where("crack_job_id = ?", jobID).Delete(&models.CrackJobCredential{}).Error; err != nil {
			return err
		}
		commands := tx.Where("crack_job_id = ?", jobID)
		if len(taskIDs) != 0 {
			commands = commands.Or("crack_task_id IN ?", taskIDs)
		}
		if err := commands.Delete(&models.CrackCommand{}).Error; err != nil {
			return err
		}
		if err := tx.Where("crack_job_id = ?", jobID).Delete(&models.CrackTask{}).Error; err != nil {
			return err
		}
		deleted := tx.Where("id = ?", jobID).Delete(&models.CrackJob{})
		if deleted.Error != nil {
			return deleted.Error
		}
		if deleted.RowsAffected != 1 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, status.Error(codes.NotFound, "crack job not found")
	}
	if errors.Is(err, errActiveCrackJob) {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}
	if err != nil {
		crackCommandRPCLog.Errorf("Failed to delete crack job %s: %s", jobID, err)
		return nil, status.Error(codes.Internal, "failed to delete crack job")
	}

	core.EventBroker.Publish(core.Event{EventType: consts.CrackJobUpdated, Data: []byte(jobID.String())})
	return &commonpb.Empty{}, nil
}
