package crack

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const maxKeyboardLayoutMappingBytes int64 = 1 << 20

func readKeyboardLayoutMapping(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open keyboard layout mapping %q: %w", path, err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat keyboard layout mapping %q: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("keyboard layout mapping %q is not a regular file", path)
	}
	if info.Size() > maxKeyboardLayoutMappingBytes {
		return nil, fmt.Errorf("keyboard layout mapping %q exceeds %d bytes", path, maxKeyboardLayoutMappingBytes)
	}
	data, err := io.ReadAll(io.LimitReader(file, maxKeyboardLayoutMappingBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read keyboard layout mapping %q: %w", path, err)
	}
	if int64(len(data)) > maxKeyboardLayoutMappingBytes {
		return nil, fmt.Errorf("keyboard layout mapping %q exceeds %d bytes", path, maxKeyboardLayoutMappingBytes)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("keyboard layout mapping %q is empty", path)
	}
	return data, nil
}

func shouldRunCrack(cmd *cobra.Command, args []string) bool {
	if len(args) > 0 {
		return true
	}

	hasFlag := false
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		if flag.Changed && flag.Name != "timeout" {
			hasFlag = true
		}
	})
	return hasFlag
}

func buildCrackCommand(cmd *cobra.Command, args []string) (*clientpb.CrackCommand, error) {
	flags := cmd.Flags()
	req := &clientpb.CrackCommand{HashType: clientpb.HashType_INVALID}

	if flags.Changed("attack-mode") {
		raw, _ := flags.GetString("attack-mode")
		attackMode, err := parseCrackAttackMode(raw)
		if err != nil {
			return nil, err
		}
		req.AttackMode = attackMode
	}

	if flags.Changed("hash-type") {
		raw, _ := flags.GetString("hash-type")
		hashType, err := parseHashType(raw)
		if err != nil {
			return nil, err
		}
		req.HashType = hashType
	}
	if flags.Changed("hash-mode") {
		hashMode, _ := flags.GetUint32("hash-mode")
		req.HashMode = &hashMode
	}

	hashes, _ := flags.GetStringArray("hash")
	if len(args) > 0 {
		hashes = append(hashes, args...)
	}
	req.Hashes = hashes
	req.CredentialIDs, _ = flags.GetStringArray("credential")
	for index, credentialID := range req.CredentialIDs {
		req.CredentialIDs[index] = strings.TrimSpace(credentialID)
		if req.CredentialIDs[index] == "" {
			return nil, fmt.Errorf("--credential cannot be empty")
		}
	}
	req.CredentialCollection, _ = flags.GetString("credential-collection")
	req.CredentialCollection = strings.TrimSpace(req.CredentialCollection)
	req.IncludeCrackedCredentials, _ = flags.GetBool("include-cracked-credentials")
	if req.IncludeCrackedCredentials && len(req.CredentialIDs) == 0 && req.CredentialCollection == "" {
		return nil, fmt.Errorf("--include-cracked-credentials requires --credential or --credential-collection")
	}
	req.PositionalArguments, _ = flags.GetStringArray("input")
	if stdin, _ := flags.GetString("stdin"); stdin != "" {
		req.Stdin = []byte(stdin)
	}

	if flags.Changed("segment-size") {
		return nil, fmt.Errorf("--segment-size was removed in Hashcat v7")
	}

	req.Quiet, _ = flags.GetBool("quiet")
	req.HexCharset, _ = flags.GetBool("hex-charset")
	req.HexSalt, _ = flags.GetBool("hex-salt")
	req.HexWordlist, _ = flags.GetBool("hex-wordlist")
	req.Force, _ = flags.GetBool("force")
	req.DeprecatedCheckDisable, _ = flags.GetBool("deprecated-check-disable")
	req.AdviceDisable, _ = flags.GetBool("advice-disable")
	req.Status, _ = flags.GetBool("status")
	req.StatusJSON, _ = flags.GetBool("status-json")
	req.StatusTimer, _ = flags.GetUint32("status-timer")
	if flags.Changed("status-timer") {
		value := req.StatusTimer
		req.StatusTimerV7 = &value
	}
	req.StdinTimeoutAbort, _ = flags.GetUint32("stdin-timeout-abort")
	if flags.Changed("stdin-timeout-abort") {
		value := req.StdinTimeoutAbort
		req.StdinTimeoutAbortV7 = &value
	}
	req.MachineReadable, _ = flags.GetBool("machine-readable")
	req.KeepGuessing, _ = flags.GetBool("keep-guessing")
	req.SelfTestDisable, _ = flags.GetBool("self-test-disable")
	req.Loopback, _ = flags.GetBool("loopback")

	if raw, _ := flags.GetString("markov-hcstat2"); raw != "" {
		req.MarkovHcstat2 = []byte(raw)
	}
	req.MarkovDisable, _ = flags.GetBool("markov-disable")
	req.MarkovClassic, _ = flags.GetBool("markov-classic")
	req.MarkovInverse, _ = flags.GetBool("markov-inverse")
	req.MarkovThreshold, _ = flags.GetUint32("markov-threshold")
	req.Runtime, _ = flags.GetUint32("runtime")
	req.Session, _ = flags.GetString("session")
	req.Restore, _ = flags.GetBool("restore")
	req.RestoreDisable, _ = flags.GetBool("restore-disable")
	if raw, _ := flags.GetString("restore-file"); raw != "" {
		req.RestoreFile = []byte(raw)
	}
	req.RestorePosition, _ = flags.GetBool("restore-position")
	req.RestoreShowCommand, _ = flags.GetBool("restore-show-command")
	if req.RestoreShowCommand && (req.Restore || req.RestorePosition) {
		return nil, fmt.Errorf("--restore-show-command and restore-position modes are mutually exclusive")
	}

	req.Outfile, _ = flags.GetString("outfile")
	if formats, _ := flags.GetStringSlice("outfile-format"); len(formats) > 0 {
		for _, raw := range formats {
			format, err := parseCrackOutfileFormat(raw)
			if err != nil {
				return nil, err
			}
			req.OutfileFormat = append(req.OutfileFormat, format)
		}
	}
	req.OutfileJSON, _ = flags.GetBool("outfile-json")
	req.OutfileAutohexDisable, _ = flags.GetBool("outfile-autohex-disable")
	req.OutfileCheckTimer, _ = flags.GetUint32("outfile-check-timer")
	if flags.Changed("outfile-check-timer") {
		value := req.OutfileCheckTimer
		req.OutfileCheckTimerV7 = &value
	}
	req.WordlistAutohexDisable, _ = flags.GetBool("wordlist-autohex-disable")
	req.Separator, _ = flags.GetString("separator")
	req.Stdout, _ = flags.GetBool("stdout")
	req.Show, _ = flags.GetBool("show")
	req.Left, _ = flags.GetBool("left")
	req.Username, _ = flags.GetBool("username")
	req.Remove, _ = flags.GetBool("remove")
	req.RemoveTimer, _ = flags.GetUint32("remove-timer")
	req.PotfileDisable, _ = flags.GetBool("potfile-disable")
	if raw, _ := flags.GetString("potfile"); raw != "" {
		req.Potfile = []byte(raw)
	}

	if flags.Changed("encoding-from") {
		raw, _ := flags.GetString("encoding-from")
		if enc, err := parseCrackEncoding(raw); err == nil {
			req.EncodingFrom = enc
		}
		if _, err := strconv.ParseInt(raw, 10, 32); err != nil {
			req.EncodingFromName = raw
		}
	}
	if flags.Changed("encoding-to") {
		raw, _ := flags.GetString("encoding-to")
		if enc, err := parseCrackEncoding(raw); err == nil {
			req.EncodingTo = enc
		}
		if _, err := strconv.ParseInt(raw, 10, 32); err != nil {
			req.EncodingToName = raw
		}
	}

	req.DebugMode, _ = flags.GetUint32("debug-mode")
	req.DebugFile, _ = flags.GetString("debug-file")
	req.InductionDir, _ = flags.GetString("induction-dir")
	req.OutfileCheckDir, _ = flags.GetString("outfile-check-dir")
	req.LogfileDisable, _ = flags.GetBool("logfile-disable")
	req.HccapxMessagePair, _ = flags.GetUint32("hccapx-message-pair")
	if flags.Changed("hccapx-message-pair") {
		value := req.HccapxMessagePair
		req.HccapxMessagePairV7 = &value
	}
	req.NonceErrorCorrections, _ = flags.GetUint32("nonce-error-corrections")
	if flags.Changed("nonce-error-corrections") {
		value := req.NonceErrorCorrections
		req.NonceErrorCorrectionsV7 = &value
	}
	if raw, _ := flags.GetString("keyboard-layout-mapping"); raw != "" {
		mapping, err := readKeyboardLayoutMapping(raw)
		if err != nil {
			return nil, err
		}
		req.KeyboardLayoutMapping = mapping
	}
	req.TruecryptKeyfiles, _ = flags.GetString("truecrypt-keyfiles")
	req.VeracryptKeyfiles, _ = flags.GetString("veracrypt-keyfiles")
	if flags.Changed("veracrypt-pim-start") {
		value, _ := flags.GetUint32("veracrypt-pim-start")
		req.VeracryptPimStart = &value
	}
	if flags.Changed("veracrypt-pim-stop") {
		value, _ := flags.GetUint32("veracrypt-pim-stop")
		req.VeracryptPimStop = &value
	}

	req.Benchmark, _ = flags.GetBool("benchmark")
	req.BenchmarkAll, _ = flags.GetBool("benchmark-all")
	req.BenchmarkMin, _ = flags.GetUint32("benchmark-min")
	if flags.Changed("benchmark-max") {
		value, _ := flags.GetUint32("benchmark-max")
		req.BenchmarkMax = &value
	}
	req.SpeedOnly, _ = flags.GetBool("speed-only")
	req.ProgressOnly, _ = flags.GetBool("progress-only")
	req.PipelineStats, _ = flags.GetBool("pipeline-stats")
	req.TaskTimeBreakdown, _ = flags.GetBool("task-time-breakdown")
	req.BitmapMin, _ = flags.GetUint32("bitmap-min")
	if flags.Changed("bitmap-min") {
		value := req.BitmapMin
		req.BitmapMinV7 = &value
	}
	req.BitmapMax, _ = flags.GetUint32("bitmap-max")
	if flags.Changed("bitmap-max") {
		value := req.BitmapMax
		req.BitmapMaxV7 = &value
	}
	req.CPUAffinity = uintSliceToUint32(flags, "cpu-affinity")
	req.HookThreads, _ = flags.GetUint32("hook-threads")
	req.HashInfo, _ = flags.GetBool("hash-info")
	if req.HashInfo {
		req.HashInfoLevel = 1
	}
	if flags.Changed("hash-info-level") {
		req.HashInfoLevel, _ = flags.GetUint32("hash-info-level")
	}
	req.BackendIgnoreCUDA, _ = flags.GetBool("backend-ignore-cuda")
	req.BackendIgnoreHip, _ = flags.GetBool("backend-ignore-hip")
	req.BackendIgnoreMetal, _ = flags.GetBool("backend-ignore-metal")
	req.BackendIgnoreOpenCL, _ = flags.GetBool("backend-ignore-opencl")
	req.BackendInfo, _ = flags.GetBool("backend-info")
	if req.BackendInfo {
		req.BackendInfoLevel = 1
	}
	if flags.Changed("backend-info-level") {
		req.BackendInfoLevel, _ = flags.GetUint32("backend-info-level")
	}
	req.BackendDevices = uintSliceToUint32(flags, "backend-devices")
	req.BackendDevicesVirtMulti, _ = flags.GetUint32("backend-devices-virtmulti")
	req.BackendDevicesVirtHost, _ = flags.GetUint32("backend-devices-virthost")
	req.OpenCLDeviceTypes = uintSliceToUint32(flags, "opencl-device-types")
	req.MetalCompilerRuntime, _ = flags.GetUint32("metal-compiler-runtime")
	req.OptimizedKernelEnable, _ = flags.GetBool("optimized-kernel-enable")
	req.MultiplyAccelDisabled, _ = flags.GetBool("multiply-accel-disable")
	if !req.MultiplyAccelDisabled {
		req.MultiplyAccelDisabled, _ = flags.GetBool("multiply-accel-disabled")
	}

	if flags.Changed("workload-profile") {
		raw, _ := flags.GetString("workload-profile")
		profile, err := parseCrackWorkloadProfile(raw)
		if err != nil {
			return nil, err
		}
		req.WorkloadProfile = profile
	}

	req.KernelAccel, _ = flags.GetUint32("kernel-accel")
	req.KernelLoops, _ = flags.GetUint32("kernel-loops")
	req.KernelThreads, _ = flags.GetUint32("kernel-threads")
	req.BackendVectorWidth, _ = flags.GetUint32("backend-vector-width")
	req.SpinDamp, _ = flags.GetUint32("spin-damp")
	req.HwmonDisable, _ = flags.GetBool("hwmon-disable")
	req.HwmonTempAbort, _ = flags.GetUint32("hwmon-temp-abort")
	if flags.Changed("hwmon-temp-abort") {
		value := req.HwmonTempAbort
		req.HwmonTempAbortV7 = &value
	}
	req.ScryptTMTO, _ = flags.GetUint32("scrypt-tmto")
	if flags.Changed("scrypt-tmto") {
		value := req.ScryptTMTO
		req.ScryptTMTOV7 = &value
	}
	req.Skip, _ = flags.GetUint64("skip")
	req.Limit, _ = flags.GetUint64("limit")
	req.Keyspace, _ = flags.GetBool("keyspace")
	req.TotalCandidates, _ = flags.GetBool("total-candidates")
	req.Lookup, _ = flags.GetString("lookup")
	req.DynamicX, _ = flags.GetBool("dynamic-x")
	req.SeekDBPath, _ = flags.GetString("seekdb-path")

	req.RuleLeft, _ = flags.GetString("rule-left")
	req.RuleRight, _ = flags.GetString("rule-right")
	if values, _ := flags.GetStringArray("rules-file"); len(values) > 0 {
		req.RulesFile = []byte(values[0])
		for _, value := range values {
			req.RulesFilesV7 = append(req.RulesFilesV7, []byte(value))
		}
	}
	req.GenerateRules, _ = flags.GetUint32("generate-rules")
	req.GenerateRulesFunMin, _ = flags.GetUint32("generate-rules-func-min")
	if !flags.Changed("generate-rules-func-min") {
		req.GenerateRulesFunMin, _ = flags.GetUint32("generate-rules-fun-min")
	}
	if flags.Changed("generate-rules-func-min") || flags.Changed("generate-rules-fun-min") {
		value := req.GenerateRulesFunMin
		req.GenerateRulesFuncMinV7 = &value
	}
	req.GenerateRulesFunMax, _ = flags.GetUint32("generate-rules-func-max")
	if !flags.Changed("generate-rules-func-max") {
		req.GenerateRulesFunMax, _ = flags.GetUint32("generate-rules-fun-max")
	}
	if flags.Changed("generate-rules-func-max") || flags.Changed("generate-rules-fun-max") {
		value := req.GenerateRulesFunMax
		req.GenerateRulesFuncMaxV7 = &value
	}
	req.GenerateRulesFuncSel, _ = flags.GetString("generate-rules-func-sel")
	req.GenerateRulesSeed, _ = flags.GetInt32("generate-rules-seed")
	if flags.Changed("generate-rules-seed") && req.GenerateRulesSeed >= 0 {
		value := uint32(req.GenerateRulesSeed)
		req.GenerateRulesSeedV7 = &value
	}
	req.CustomCharset1, _ = flags.GetString("custom-charset1")
	req.CustomCharset2, _ = flags.GetString("custom-charset2")
	req.CustomCharset3, _ = flags.GetString("custom-charset3")
	req.CustomCharset4, _ = flags.GetString("custom-charset4")
	req.CustomCharset5, _ = flags.GetString("custom-charset5")
	req.CustomCharset6, _ = flags.GetString("custom-charset6")
	req.CustomCharset7, _ = flags.GetString("custom-charset7")
	req.CustomCharset8, _ = flags.GetString("custom-charset8")
	req.Identify, _ = flags.GetString("identify")
	req.IdentifyMode, _ = flags.GetBool("identify-mode")
	if req.IdentifyMode && (flags.Changed("hash-mode") || flags.Changed("hash-type")) {
		return nil, fmt.Errorf("--identify-mode and hash mode/type are mutually exclusive")
	}
	req.Increment, _ = flags.GetBool("increment")
	req.IncrementInverse, _ = flags.GetBool("increment-inverse")
	req.IncrementMin, _ = flags.GetUint32("increment-min")
	req.IncrementMax, _ = flags.GetUint32("increment-max")
	if flags.Changed("bypass-delay") {
		value, _ := flags.GetUint32("bypass-delay")
		req.BypassDelay = &value
	}
	if flags.Changed("bypass-threshold") {
		value, _ := flags.GetUint32("bypass-threshold")
		req.BypassThreshold = &value
	}
	req.SlowCandidates, _ = flags.GetBool("slow-candidates")
	req.BridgeParameter1, _ = flags.GetString("bridge-parameter1")
	req.BridgeParameter2, _ = flags.GetString("bridge-parameter2")
	req.BridgeParameter3, _ = flags.GetString("bridge-parameter3")
	req.BridgeParameter4, _ = flags.GetString("bridge-parameter4")
	req.BrainServer, _ = flags.GetBool("brain-server")
	req.BrainServerTimer, _ = flags.GetUint32("brain-server-timer")
	if flags.Changed("brain-server-timer") {
		value := req.BrainServerTimer
		req.BrainServerTimerV7 = &value
	}
	req.BrainClient, _ = flags.GetBool("brain-client")
	req.BrainClientFeatures, _ = flags.GetString("brain-client-features")
	if value, ok := parseUint32(req.BrainClientFeatures); ok {
		req.BrainClientFeaturesV7 = value
	}
	req.BrainHost, _ = flags.GetString("brain-host")
	req.BrainPort, _ = flags.GetUint32("brain-port")
	req.BrainPassword, _ = flags.GetString("brain-password")
	if flags.Changed("brain-password") {
		value := req.BrainPassword
		req.BrainPasswordV7 = &value
	}
	req.BrainSession, _ = flags.GetString("brain-session")
	if value, ok := parseHexUint32(req.BrainSession); ok {
		req.BrainSessionV7 = &value
	}
	req.BrainSessionWhitelist, _ = flags.GetString("brain-session-whitelist")
	if values, ok := parseHexUint32List(req.BrainSessionWhitelist); ok {
		req.BrainSessionWhitelistV7 = values
	}
	req.BrainFeed, _ = flags.GetBool("brain-feed")
	req.ColorCracked, _ = flags.GetBool("color-cracked")
	req.HashCopy, _ = flags.GetBool("hash-copy")
	req.EncryptWithPubkey, _ = flags.GetString("encrypt-with-pubkey")

	return req, nil
}

func uintSliceToUint32(flags *pflag.FlagSet, name string) []uint32 {
	values, _ := flags.GetUintSlice(name)
	if len(values) == 0 {
		return nil
	}
	out := make([]uint32, len(values))
	for i, value := range values {
		out[i] = uint32(value)
	}
	return out
}

func parseUint32(raw string) (uint32, bool) {
	if raw == "" {
		return 0, false
	}
	value, err := strconv.ParseUint(raw, 10, 32)
	return uint32(value), err == nil
}

func parseHexUint32(raw string) (uint32, bool) {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "0x") || strings.HasPrefix(raw, "0X") {
		raw = raw[2:]
	}
	if raw == "" {
		return 0, false
	}
	value, err := strconv.ParseUint(raw, 16, 32)
	return uint32(value), err == nil
}

func parseHexUint32List(raw string) ([]uint32, bool) {
	if raw == "" {
		return nil, false
	}
	parts := strings.Split(raw, ",")
	values := make([]uint32, 0, len(parts))
	for _, part := range parts {
		value, ok := parseHexUint32(part)
		if !ok {
			return nil, false
		}
		values = append(values, value)
	}
	return values, true
}

func parseCrackAttackMode(raw string) (clientpb.CrackAttackMode, error) {
	if raw == "" {
		return clientpb.CrackAttackMode_STRAIGHT, nil
	}
	if numeric, err := strconv.ParseInt(raw, 10, 32); err == nil {
		return clientpb.CrackAttackMode(numeric), nil
	}

	normalized := strings.ToLower(strings.TrimSpace(raw))
	normalized = strings.NewReplacer("-", "_", " ", "_").Replace(normalized)
	switch normalized {
	case "straight":
		return clientpb.CrackAttackMode_STRAIGHT, nil
	case "combination":
		return clientpb.CrackAttackMode_COMBINATION, nil
	case "bruteforce", "brute_force", "brute-force":
		return clientpb.CrackAttackMode_BRUTEFORCE, nil
	case "pcfg":
		return clientpb.CrackAttackMode_PCFG, nil
	case "hybrid_wordlist_mask":
		return clientpb.CrackAttackMode_HYBRID_WORDLIST_MASK, nil
	case "hybrid_mask_wordlist":
		return clientpb.CrackAttackMode_HYBRID_MASK_WORDLIST, nil
	case "generic":
		return clientpb.CrackAttackMode_GENERIC, nil
	case "association":
		return clientpb.CrackAttackMode_ASSOCIATION, nil
	case "no_attack", "none":
		return clientpb.CrackAttackMode_NO_ATTACK, nil
	case "hybrid":
		return clientpb.CrackAttackMode_HYBRID, nil
	default:
		return clientpb.CrackAttackMode_STRAIGHT, fmt.Errorf("unknown attack mode: %s", raw)
	}
}

func parseHashType(raw string) (clientpb.HashType, error) {
	if raw == "" {
		return clientpb.HashType_MD5, nil
	}
	if numeric, err := strconv.ParseInt(raw, 10, 32); err == nil {
		return clientpb.HashType(numeric), nil
	}
	key := normalizeEnumKey(raw)
	if value, ok := clientpb.HashType_value[key]; ok {
		return clientpb.HashType(value), nil
	}
	return clientpb.HashType_MD5, fmt.Errorf("unknown hash type: %s", raw)
}

func parseCrackEncoding(raw string) (clientpb.CrackEncoding, error) {
	if raw == "" {
		return clientpb.CrackEncoding_INVALID_ENCODING, nil
	}
	if numeric, err := strconv.ParseInt(raw, 10, 32); err == nil {
		return clientpb.CrackEncoding(numeric), nil
	}
	key := normalizeEnumKey(raw)
	if value, ok := clientpb.CrackEncoding_value[key]; ok {
		return clientpb.CrackEncoding(value), nil
	}
	return clientpb.CrackEncoding_INVALID_ENCODING, fmt.Errorf("unknown encoding: %s", raw)
}

func parseCrackOutfileFormat(raw string) (clientpb.CrackOutfileFormat, error) {
	if raw == "" {
		return clientpb.CrackOutfileFormat_INVALID_FORMAT, nil
	}
	if numeric, err := strconv.ParseInt(raw, 10, 32); err == nil {
		return clientpb.CrackOutfileFormat(numeric), nil
	}
	key := normalizeEnumKey(raw)
	if value, ok := clientpb.CrackOutfileFormat_value[key]; ok {
		return clientpb.CrackOutfileFormat(value), nil
	}
	return clientpb.CrackOutfileFormat_INVALID_FORMAT, fmt.Errorf("unknown outfile format: %s", raw)
}

func parseCrackWorkloadProfile(raw string) (clientpb.CrackWorkloadProfile, error) {
	if raw == "" {
		return clientpb.CrackWorkloadProfile_INVALID_WORKLOAD_PROFILE, nil
	}
	if numeric, err := strconv.ParseInt(raw, 10, 32); err == nil {
		return clientpb.CrackWorkloadProfile(numeric), nil
	}
	key := normalizeEnumKey(raw)
	if value, ok := clientpb.CrackWorkloadProfile_value[key]; ok {
		return clientpb.CrackWorkloadProfile(value), nil
	}
	return clientpb.CrackWorkloadProfile_INVALID_WORKLOAD_PROFILE, fmt.Errorf("unknown workload profile: %s", raw)
}

func normalizeEnumKey(value string) string {
	if value == "" {
		return ""
	}
	var builder strings.Builder
	builder.Grow(len(value))
	underscore := false
	for i := 0; i < len(value); i++ {
		ch := value[i]
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') {
			if ch >= 'a' && ch <= 'z' {
				ch -= 'a' - 'A'
			}
			builder.WriteByte(ch)
			underscore = false
			continue
		}
		if !underscore {
			builder.WriteByte('_')
			underscore = true
		}
	}
	return strings.Trim(builder.String(), "_")
}
