package clientpb

import (
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestCrackCommandHashcatV7FieldNumbers(t *testing.T) {
	fields := (&CrackCommand{}).ProtoReflect().Descriptor().Fields()
	want := map[protoreflect.Name]protoreflect.FieldNumber{
		"Outfile": 28, "DebugFile": 45, "InductionDir": 46, "OutfileCheckDir": 47,
		"TruecryptKeyfiles": 52, "VeracryptKeyfiles": 53, "VeracryptPimStart": 54,
		"VeracryptPimStop": 55, "RuleLeft": 88, "RuleRight": 89,
		"PipelineStats": 114, "TaskTimeBreakdown": 115, "MetalCompilerRuntime": 116,
		"RestorePosition": 117, "OutfileJSON": 118, "DynamicX": 119,
		"SeekDBPath": 120, "BenchmarkMin": 121, "BenchmarkMax": 122,
		"BridgeParameter1": 123, "BridgeParameter2": 124, "BridgeParameter3": 125,
		"BridgeParameter4": 126, "BackendDevicesVirtMulti": 127,
		"BackendDevicesVirtHost": 128, "TotalCandidates": 129, "Lookup": 130,
		"CustomCharset5": 131, "CustomCharset6": 132, "CustomCharset7": 133,
		"CustomCharset8": 134, "IncrementInverse": 135, "BypassDelay": 136,
		"BypassThreshold": 137, "BrainFeed": 138, "ColorCracked": 139,
		"HashCopy": 140, "EncryptWithPubkey": 141, "IdentifyMode": 142,
		"PositionalArguments": 143, "EncodingFromName": 144, "EncodingToName": 145,
		"HashInfoLevel": 146, "BackendInfoLevel": 147, "HccapxMessagePairV7": 148,
		"NonceErrorCorrectionsV7": 149, "ScryptTMTOV7": 150,
		"GenerateRulesSeedV7": 151, "BrainClientFeaturesV7": 152,
		"BrainSessionV7": 153, "BrainSessionWhitelistV7": 154, "Stdin": 155,
		"AdviceDisable": 156, "HashMode": 157, "RestoreShowCommand": 158,
		"BrainServerTimerV7": 159, "StatusTimerV7": 160,
		"StdinTimeoutAbortV7": 161, "OutfileCheckTimerV7": 162,
		"BitmapMinV7": 163, "BitmapMaxV7": 164, "HwmonTempAbortV7": 165,
		"RulesFilesV7": 166, "BrainPasswordV7": 167,
		"GenerateRulesFuncMinV7": 168, "GenerateRulesFuncMaxV7": 169,
	}
	for name, number := range want {
		field := fields.ByName(name)
		if field == nil {
			t.Fatalf("CrackCommand is missing field %s", name)
		}
		if field.Number() != number {
			t.Fatalf("CrackCommand.%s tag = %d, want %d", name, field.Number(), number)
		}
	}
}

func TestCrackCommandHashcatV7PresenceRoundTrip(t *testing.T) {
	zero := uint32(0)
	empty := ""
	want := &CrackCommand{
		VeracryptPimStart:      &zero,
		VeracryptPimStop:       &zero,
		BenchmarkMax:           &zero,
		BypassDelay:            &zero,
		BypassThreshold:        &zero,
		BrainSessionV7:         &zero,
		HashMode:               &zero,
		BrainServerTimerV7:     &zero,
		StatusTimerV7:          &zero,
		StdinTimeoutAbortV7:    &zero,
		OutfileCheckTimerV7:    &zero,
		BitmapMinV7:            &zero,
		BitmapMaxV7:            &zero,
		HwmonTempAbortV7:       &zero,
		RulesFilesV7:           [][]byte{[]byte(":"), []byte("$1")},
		BrainPasswordV7:        &empty,
		GenerateRulesFuncMinV7: &zero,
		GenerateRulesFuncMaxV7: &zero,
	}
	raw, err := proto.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	got := &CrackCommand{}
	if err := proto.Unmarshal(raw, got); err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(got, want) {
		t.Fatalf("explicit-zero/empty presence was not preserved: got %v, want %v", got, want)
	}
}
