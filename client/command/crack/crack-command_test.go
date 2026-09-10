package crack

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

func TestBuildCrackCommandReadsKeyboardLayoutMapping(t *testing.T) {
	mapping := []byte("a A\nb B\n")
	path := filepath.Join(t.TempDir(), "keyboard.hckmap")
	if err := os.WriteFile(path, mapping, 0600); err != nil {
		t.Fatal(err)
	}
	cmd := &cobra.Command{Use: "crack-test"}
	bindCrackFlags(cmd.Flags())
	if err := cmd.Flags().Parse([]string{"--keyboard-layout-mapping=" + path}); err != nil {
		t.Fatal(err)
	}
	got, err := buildCrackCommand(cmd, nil)
	if err != nil {
		t.Fatalf("buildCrackCommand() error = %v", err)
	}
	if !bytes.Equal(got.KeyboardLayoutMapping, mapping) {
		t.Fatalf("KeyboardLayoutMapping = %q, want file contents %q", got.KeyboardLayoutMapping, mapping)
	}
}

func TestBuildCrackCommandRejectsInvalidKeyboardLayoutMapping(t *testing.T) {
	tests := []struct {
		name string
		path func(*testing.T) string
		want string
	}{
		{
			name: "missing",
			path: func(t *testing.T) string { return filepath.Join(t.TempDir(), "missing.hckmap") },
			want: "open keyboard layout mapping",
		},
		{
			name: "directory",
			path: func(t *testing.T) string { return t.TempDir() },
			want: "is not a regular file",
		},
		{
			name: "empty",
			path: func(t *testing.T) string {
				path := filepath.Join(t.TempDir(), "empty.hckmap")
				if err := os.WriteFile(path, nil, 0600); err != nil {
					t.Fatal(err)
				}
				return path
			},
			want: "is empty",
		},
		{
			name: "too large",
			path: func(t *testing.T) string {
				path := filepath.Join(t.TempDir(), "large.hckmap")
				if err := os.WriteFile(path, make([]byte, maxKeyboardLayoutMappingBytes+1), 0600); err != nil {
					t.Fatal(err)
				}
				return path
			},
			want: "exceeds",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: "crack-test"}
			bindCrackFlags(cmd.Flags())
			if err := cmd.Flags().Parse([]string{"--keyboard-layout-mapping=" + test.path(t)}); err != nil {
				t.Fatal(err)
			}
			if _, err := buildCrackCommand(cmd, nil); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("buildCrackCommand() error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func TestParseCrackAttackMode(t *testing.T) {
	tests := []struct {
		name string
		want clientpb.CrackAttackMode
	}{
		{name: "straight", want: clientpb.CrackAttackMode_STRAIGHT},
		{name: "combination", want: clientpb.CrackAttackMode_COMBINATION},
		{name: "bruteforce", want: clientpb.CrackAttackMode_BRUTEFORCE},
		{name: "pcfg", want: clientpb.CrackAttackMode_PCFG},
		{name: "hybrid-wordlist-mask", want: clientpb.CrackAttackMode_HYBRID_WORDLIST_MASK},
		{name: "hybrid-mask-wordlist", want: clientpb.CrackAttackMode_HYBRID_MASK_WORDLIST},
		{name: "generic", want: clientpb.CrackAttackMode_GENERIC},
		{name: "association", want: clientpb.CrackAttackMode_ASSOCIATION},
		{name: "no-attack", want: clientpb.CrackAttackMode_NO_ATTACK},
		{name: "hybrid", want: clientpb.CrackAttackMode_HYBRID},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseCrackAttackMode(test.name)
			if err != nil {
				t.Fatalf("parseCrackAttackMode(%q) returned error: %v", test.name, err)
			}
			if got != test.want {
				t.Fatalf("parseCrackAttackMode(%q) = %v, want %v", test.name, got, test.want)
			}
		})
	}
}

func TestBuildCrackCommandHashcatShortFlags(t *testing.T) {
	cmd := &cobra.Command{Use: "crack-test"}
	bindCrackFlags(cmd.Flags())
	if err := cmd.Flags().Parse([]string{"-a", "3", "-m", "1000"}); err != nil {
		t.Fatal(err)
	}
	got, err := buildCrackCommand(cmd, []string{"hash"})
	if err != nil {
		t.Fatalf("buildCrackCommand() error = %v", err)
	}
	if got.AttackMode != clientpb.CrackAttackMode_BRUTEFORCE {
		t.Fatalf("AttackMode = %s, want BRUTEFORCE", got.AttackMode)
	}
	if got.HashType != clientpb.HashType(1000) {
		t.Fatalf("HashType = %d, want 1000", got.HashType)
	}
}

func TestBuildCrackCommandHashcatV7Options(t *testing.T) {
	cmd := &cobra.Command{Use: "crack-test"}
	bindCrackFlags(cmd.Flags())
	args := []string{
		"--attack-mode=hybrid",
		"--hash-mode=0",
		"--input=first-input",
		"--input=second-input",
		"--stdin=stdin-data",
		"--advice-disable",
		"--status-timer=0",
		"--stdin-timeout-abort=0",
		"--restore-position",
		"--outfile=outfile.txt",
		"--outfile-json",
		"--debug-file=debug.txt",
		"--induction-dir=induction",
		"--outfile-check-dir=outfile-check",
		"--outfile-check-timer=0",
		"--truecrypt-keyfiles=truecrypt.keys",
		"--veracrypt-keyfiles=veracrypt.keys",
		"--veracrypt-pim-start=11",
		"--veracrypt-pim-stop=12",
		"--encoding-from=windows-1252",
		"--encoding-to=utf-8",
		"--hccapx-message-pair=0",
		"--nonce-error-corrections=0",
		"--benchmark-min=13",
		"--benchmark-max=14",
		"--pipeline-stats",
		"--task-time-breakdown",
		"--bitmap-min=0",
		"--bitmap-max=0",
		"--hash-info",
		"--hash-info-level=2",
		"--backend-info",
		"--backend-info-level=2",
		"--backend-devices-virtmulti=15",
		"--backend-devices-virthost=16",
		"--metal-compiler-runtime=17",
		"--multiply-accel-disable",
		"--hwmon-temp-abort=0",
		"--total-candidates",
		"--lookup=lookup-value",
		"--dynamic-x",
		"--seekdb-path=seekdb",
		"--rule-left=l",
		"--rule-right=r",
		"--rules-file=:",
		"--rules-file=$1",
		"--custom-charset5=cs5",
		"--custom-charset6=cs6",
		"--custom-charset7=cs7",
		"--custom-charset8=cs8",
		"--identify=legacy-input",
		"--increment-inverse",
		"--bypass-delay=18",
		"--bypass-threshold=19",
		"--bridge-parameter1=bridge1",
		"--bridge-parameter2=bridge2",
		"--bridge-parameter3=bridge3",
		"--bridge-parameter4=bridge4",
		"--scrypt-tmto=0",
		"--generate-rules-func-min=4",
		"--generate-rules-func-max=8",
		"--generate-rules-seed=0",
		"--brain-client-features=3",
		"--brain-password=",
		"--brain-server-timer=0",
		"--brain-session=deadbeef",
		"--brain-session-whitelist=1,2,0x3",
		"--brain-feed",
		"--color-cracked",
		"--hash-copy",
		"--encrypt-with-pubkey=key.pub",
	}
	if err := cmd.Flags().Parse(args); err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	got, err := buildCrackCommand(cmd, nil)
	if err != nil {
		t.Fatalf("buildCrackCommand returned error: %v", err)
	}
	zero := uint32(0)
	want := &clientpb.CrackCommand{
		AttackMode:              clientpb.CrackAttackMode_HYBRID,
		HashType:                clientpb.HashType_INVALID,
		HashMode:                &zero,
		Stdin:                   []byte("stdin-data"),
		AdviceDisable:           true,
		StatusTimerV7:           &zero,
		StdinTimeoutAbortV7:     &zero,
		Outfile:                 "outfile.txt",
		DebugFile:               "debug.txt",
		InductionDir:            "induction",
		OutfileCheckDir:         "outfile-check",
		OutfileCheckTimerV7:     &zero,
		TruecryptKeyfiles:       "truecrypt.keys",
		VeracryptKeyfiles:       "veracrypt.keys",
		VeracryptPimStart:       uint32Pointer(11),
		VeracryptPimStop:        uint32Pointer(12),
		EncodingFromName:        "windows-1252",
		EncodingToName:          "utf-8",
		HccapxMessagePairV7:     &zero,
		NonceErrorCorrectionsV7: &zero,
		RuleLeft:                "l",
		RuleRight:               "r",
		RulesFile:               []byte(":"),
		RulesFilesV7:            [][]byte{[]byte(":"), []byte("$1")},
		PipelineStats:           true,
		TaskTimeBreakdown:       true,
		BitmapMinV7:             &zero,
		BitmapMaxV7:             &zero,
		HashInfo:                true,
		HashInfoLevel:           2,
		BackendInfo:             true,
		BackendInfoLevel:        2,
		MetalCompilerRuntime:    17,
		RestorePosition:         true,
		OutfileJSON:             true,
		DynamicX:                true,
		SeekDBPath:              "seekdb",
		BenchmarkMin:            13,
		BenchmarkMax:            uint32Pointer(14),
		BridgeParameter1:        "bridge1",
		BridgeParameter2:        "bridge2",
		BridgeParameter3:        "bridge3",
		BridgeParameter4:        "bridge4",
		BackendDevicesVirtMulti: 15,
		BackendDevicesVirtHost:  16,
		MultiplyAccelDisabled:   true,
		HwmonTempAbortV7:        &zero,
		TotalCandidates:         true,
		Lookup:                  "lookup-value",
		CustomCharset5:          "cs5",
		CustomCharset6:          "cs6",
		CustomCharset7:          "cs7",
		CustomCharset8:          "cs8",
		IncrementInverse:        true,
		BypassDelay:             uint32Pointer(18),
		BypassThreshold:         uint32Pointer(19),
		ScryptTMTOV7:            &zero,
		GenerateRulesFunMin:     4,
		GenerateRulesFunMax:     8,
		GenerateRulesFuncMinV7:  uint32Pointer(4),
		GenerateRulesFuncMaxV7:  uint32Pointer(8),
		GenerateRulesSeedV7:     &zero,
		BrainClientFeatures:     "3",
		BrainClientFeaturesV7:   3,
		BrainPasswordV7:         stringPointer(""),
		BrainSession:            "deadbeef",
		BrainSessionV7:          uint32Pointer(0xdeadbeef),
		BrainServerTimerV7:      &zero,
		BrainSessionWhitelist:   "1,2,0x3",
		BrainSessionWhitelistV7: []uint32{1, 2, 3},
		BrainFeed:               true,
		ColorCracked:            true,
		HashCopy:                true,
		EncryptWithPubkey:       "key.pub",
		Identify:                "legacy-input",
		PositionalArguments:     []string{"first-input", "second-input"},
	}
	if !proto.Equal(got, want) {
		t.Fatal("buildCrackCommand() did not map the Hashcat v7 options as expected")
	}
}

func uint32Pointer(value uint32) *uint32 {
	return &value
}

func stringPointer(value string) *string {
	return &value
}

func TestBuildCrackCommandBrainSessionsAreHexadecimal(t *testing.T) {
	cmd := &cobra.Command{Use: "crack-test"}
	bindCrackFlags(cmd.Flags())
	if err := cmd.Flags().Parse([]string{
		"--brain-session=10",
		"--brain-session-whitelist=10,20",
	}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	got, err := buildCrackCommand(cmd, nil)
	if err != nil {
		t.Fatalf("buildCrackCommand returned error: %v", err)
	}
	if got.BrainSessionV7 == nil || *got.BrainSessionV7 != 0x10 {
		t.Fatalf("BrainSessionV7 = %v, want 0x10", got.BrainSessionV7)
	}
	wantWhitelist := []uint32{0x10, 0x20}
	if !slices.Equal(got.BrainSessionWhitelistV7, wantWhitelist) {
		t.Fatalf("BrainSessionWhitelistV7 = %#v, want %#v", got.BrainSessionWhitelistV7, wantWhitelist)
	}
}

func TestBuildCrackCommandPreservesCommasInPositionalInputs(t *testing.T) {
	cmd := &cobra.Command{Use: "crack-test"}
	bindCrackFlags(cmd.Flags())
	if err := cmd.Flags().Parse([]string{
		"--input=hashes,2026.txt",
		"--input=?d?d,?l?l",
	}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	got, err := buildCrackCommand(cmd, nil)
	if err != nil {
		t.Fatalf("buildCrackCommand returned error: %v", err)
	}
	want := []string{"hashes,2026.txt", "?d?d,?l?l"}
	if !slices.Equal(got.PositionalArguments, want) {
		t.Fatalf("PositionalArguments = %#v, want %#v", got.PositionalArguments, want)
	}
}

func TestBuildCrackCommandPreservesCommasInHashes(t *testing.T) {
	cmd := &cobra.Command{Use: "crack-test"}
	bindCrackFlags(cmd.Flags())
	if err := cmd.Flags().Parse([]string{
		"--hash=digest:salt,with,commas",
		"--hash=second:digest,salt",
	}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	got, err := buildCrackCommand(cmd, nil)
	if err != nil {
		t.Fatalf("buildCrackCommand returned error: %v", err)
	}
	want := []string{"digest:salt,with,commas", "second:digest,salt"}
	if !slices.Equal(got.Hashes, want) {
		t.Fatalf("Hashes = %#v, want %#v", got.Hashes, want)
	}
}

func TestBuildCrackCommandRejectsConflictingRestoreModes(t *testing.T) {
	cmd := &cobra.Command{Use: "crack-test"}
	bindCrackFlags(cmd.Flags())
	if err := cmd.Flags().Parse([]string{"--restore-position", "--restore-show-command"}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	_, err := buildCrackCommand(cmd, nil)
	if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("buildCrackCommand() error = %v, want restore conflict error", err)
	}
}

func TestBuildCrackCommandIdentifyHasNoImplicitHashType(t *testing.T) {
	cmd := &cobra.Command{Use: "crack-test"}
	bindCrackFlags(cmd.Flags())
	if err := cmd.Flags().Parse([]string{"--identify-mode", "--input=hashes.txt"}); err != nil {
		t.Fatal(err)
	}
	got, err := buildCrackCommand(cmd, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.HashType != clientpb.HashType_INVALID || got.HashMode != nil {
		t.Fatalf("identify command inherited hash mode/type: %#v", got)
	}
}

func TestBuildCrackCommandLegacyFlagAliases(t *testing.T) {
	cmd := &cobra.Command{Use: "crack-test"}
	bindCrackFlags(cmd.Flags())
	if err := cmd.Flags().Parse([]string{
		"--multiply-accel-disabled",
		"--generate-rules-fun-min=2",
		"--generate-rules-fun-max=6",
	}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	got, err := buildCrackCommand(cmd, nil)
	if err != nil {
		t.Fatalf("buildCrackCommand returned error: %v", err)
	}
	if !got.MultiplyAccelDisabled || got.GenerateRulesFunMin != 2 || got.GenerateRulesFunMax != 6 {
		t.Fatalf("legacy aliases were not preserved: %#v", got)
	}
}

func TestBuildCrackCommandRejectsSegmentSize(t *testing.T) {
	cmd := &cobra.Command{Use: "crack-test"}
	bindCrackFlags(cmd.Flags())
	if err := cmd.Flags().Set("segment-size", "32"); err != nil {
		t.Fatalf("set segment-size: %v", err)
	}

	_, err := buildCrackCommand(cmd, nil)
	if err == nil || !strings.Contains(err.Error(), "removed in Hashcat v7") {
		t.Fatalf("buildCrackCommand() error = %v, want Hashcat v7 removal error", err)
	}
}

func TestBuildCrackCommandCredentialSelectors(t *testing.T) {
	cmd := &cobra.Command{Use: "crack-test"}
	bindCrackFlags(cmd.Flags())
	if err := cmd.Flags().Parse([]string{
		"--credential= first-id ",
		"--credential=second-id",
		"--credential-collection= domain-admins ",
		"--include-cracked-credentials",
	}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	got, err := buildCrackCommand(cmd, nil)
	if err != nil {
		t.Fatalf("buildCrackCommand returned error: %v", err)
	}
	if !slices.Equal(got.CredentialIDs, []string{"first-id", "second-id"}) {
		t.Fatalf("CredentialIDs = %#v", got.CredentialIDs)
	}
	if got.CredentialCollection != "domain-admins" {
		t.Fatalf("CredentialCollection = %q", got.CredentialCollection)
	}
	if !got.IncludeCrackedCredentials {
		t.Fatal("IncludeCrackedCredentials is false")
	}
}

func TestBuildCrackCommandRejectsEmptyCredential(t *testing.T) {
	cmd := &cobra.Command{Use: "crack-test"}
	bindCrackFlags(cmd.Flags())
	if err := cmd.Flags().Parse([]string{"--credential= "}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}
	if _, err := buildCrackCommand(cmd, nil); err == nil || !strings.Contains(err.Error(), "cannot be empty") {
		t.Fatalf("buildCrackCommand error = %v", err)
	}
}

func TestBuildCrackCommandIncludeCrackedRequiresSelector(t *testing.T) {
	cmd := &cobra.Command{Use: "crack-test"}
	bindCrackFlags(cmd.Flags())
	if err := cmd.Flags().Parse([]string{"--include-cracked-credentials"}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}
	if _, err := buildCrackCommand(cmd, nil); err == nil || !strings.Contains(err.Error(), "requires --credential") {
		t.Fatalf("buildCrackCommand error = %v", err)
	}
}
