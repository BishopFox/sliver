package crack

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

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
	req := &clientpb.CrackCommand{}

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

	hashes, _ := flags.GetStringSlice("hash")
	if len(args) > 0 {
		hashes = append(hashes, args...)
	}
	req.Hashes = hashes

	req.Quiet, _ = flags.GetBool("quiet")
	req.HexCharset, _ = flags.GetBool("hex-charset")
	req.HexSalt, _ = flags.GetBool("hex-salt")
	req.HexWordlist, _ = flags.GetBool("hex-wordlist")
	req.Force, _ = flags.GetBool("force")
	req.DeprecatedCheckDisable, _ = flags.GetBool("deprecated-check-disable")
	req.Status, _ = flags.GetBool("status")
	req.StatusJSON, _ = flags.GetBool("status-json")
	req.StatusTimer, _ = flags.GetUint32("status-timer")
	req.StdinTimeoutAbort, _ = flags.GetUint32("stdin-timeout-abort")
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

	if formats, _ := flags.GetStringSlice("outfile-format"); len(formats) > 0 {
		for _, raw := range formats {
			format, err := parseCrackOutfileFormat(raw)
			if err != nil {
				return nil, err
			}
			req.OutfileFormat = append(req.OutfileFormat, format)
		}
	}
	req.OutfileAutohexDisable, _ = flags.GetBool("outfile-autohex-disable")
	req.OutfileCheckTimer, _ = flags.GetUint32("outfile-check-timer")
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
		enc, err := parseCrackEncoding(raw)
		if err != nil {
			return nil, err
		}
		req.EncodingFrom = enc
	}
	if flags.Changed("encoding-to") {
		raw, _ := flags.GetString("encoding-to")
		enc, err := parseCrackEncoding(raw)
		if err != nil {
			return nil, err
		}
		req.EncodingTo = enc
	}

	req.DebugMode, _ = flags.GetUint32("debug-mode")
	req.LogfileDisable, _ = flags.GetBool("logfile-disable")
	req.HccapxMessagePair, _ = flags.GetUint32("hccapx-message-pair")
	req.NonceErrorCorrections, _ = flags.GetUint32("nonce-error-corrections")
	if raw, _ := flags.GetString("keyboard-layout-mapping"); raw != "" {
		req.KeyboardLayoutMapping = []byte(raw)
	}

	req.Benchmark, _ = flags.GetBool("benchmark")
	req.BenchmarkAll, _ = flags.GetBool("benchmark-all")
	req.SpeedOnly, _ = flags.GetBool("speed-only")
	req.ProgressOnly, _ = flags.GetBool("progress-only")
	req.SegmentSize, _ = flags.GetUint32("segment-size")
	req.BitmapMin, _ = flags.GetUint32("bitmap-min")
	req.BitmapMax, _ = flags.GetUint32("bitmap-max")
	req.CPUAffinity = uintSliceToUint32(flags, "cpu-affinity")
	req.HookThreads, _ = flags.GetUint32("hook-threads")
	req.HashInfo, _ = flags.GetBool("hash-info")
	req.BackendIgnoreCUDA, _ = flags.GetBool("backend-ignore-cuda")
	req.BackendIgnoreHip, _ = flags.GetBool("backend-ignore-hip")
	req.BackendIgnoreMetal, _ = flags.GetBool("backend-ignore-metal")
	req.BackendIgnoreOpenCL, _ = flags.GetBool("backend-ignore-opencl")
	req.BackendInfo, _ = flags.GetBool("backend-info")
	req.BackendDevices = uintSliceToUint32(flags, "backend-devices")
	req.OpenCLDeviceTypes = uintSliceToUint32(flags, "opencl-device-types")
	req.OptimizedKernelEnable, _ = flags.GetBool("optimized-kernel-enable")
	req.MultiplyAccelDisabled, _ = flags.GetBool("multiply-accel-disabled")

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
	req.ScryptTMTO, _ = flags.GetUint32("scrypt-tmto")
	req.Skip, _ = flags.GetUint64("skip")
	req.Limit, _ = flags.GetUint64("limit")
	req.Keyspace, _ = flags.GetBool("keyspace")

	if raw, _ := flags.GetString("rules-file"); raw != "" {
		req.RulesFile = []byte(raw)
	}
	req.GenerateRules, _ = flags.GetUint32("generate-rules")
	req.GenerateRulesFunMin, _ = flags.GetUint32("generate-rules-fun-min")
	req.GenerateRulesFunMax, _ = flags.GetUint32("generate-rules-fun-max")
	req.GenerateRulesFuncSel, _ = flags.GetString("generate-rules-func-sel")
	req.GenerateRulesSeed, _ = flags.GetInt32("generate-rules-seed")
	req.CustomCharset1, _ = flags.GetString("custom-charset1")
	req.CustomCharset2, _ = flags.GetString("custom-charset2")
	req.CustomCharset3, _ = flags.GetString("custom-charset3")
	req.CustomCharset4, _ = flags.GetString("custom-charset4")
	req.Identify, _ = flags.GetString("identify")
	req.Increment, _ = flags.GetBool("increment")
	req.IncrementMin, _ = flags.GetUint32("increment-min")
	req.IncrementMax, _ = flags.GetUint32("increment-max")
	req.SlowCandidates, _ = flags.GetBool("slow-candidates")
	req.BrainServer, _ = flags.GetBool("brain-server")
	req.BrainServerTimer, _ = flags.GetUint32("brain-server-timer")
	req.BrainClient, _ = flags.GetBool("brain-client")
	req.BrainClientFeatures, _ = flags.GetString("brain-client-features")
	req.BrainHost, _ = flags.GetString("brain-host")
	req.BrainPort, _ = flags.GetUint32("brain-port")
	req.BrainPassword, _ = flags.GetString("brain-password")
	req.BrainSession, _ = flags.GetString("brain-session")
	req.BrainSessionWhitelist, _ = flags.GetString("brain-session-whitelist")

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
	case "hybrid_wordlist_mask":
		return clientpb.CrackAttackMode_HYBRID_WORDLIST_MASK, nil
	case "hybrid_mask_wordlist":
		return clientpb.CrackAttackMode_HYBRID_MASK_WORDLIST, nil
	case "association":
		return clientpb.CrackAttackMode_ASSOCIATION, nil
	case "no_attack", "none":
		return clientpb.CrackAttackMode_NO_ATTACK, nil
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
