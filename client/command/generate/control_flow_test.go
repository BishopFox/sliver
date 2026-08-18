package generate

import (
	"testing"

	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/forms"
	"github.com/bishopfox/sliver/protobuf/clientpb"
)

func TestParseControlFlowPolicy(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    clientpb.ControlFlowPolicy
		wantErr bool
	}{
		{name: "off", value: "off", want: clientpb.ControlFlowPolicy_CONTROL_FLOW_DISABLED},
		{name: "balanced", value: "balanced-v1", want: clientpb.ControlFlowPolicy_CONTROL_FLOW_BALANCED_V1},
		{name: "case and whitespace", value: " BALANCED-V1 ", want: clientpb.ControlFlowPolicy_CONTROL_FLOW_BALANCED_V1},
		{name: "unknown", value: "aggressive", wantErr: true},
		{name: "empty", value: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseControlFlowPolicy(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseControlFlowPolicy() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("parseControlFlowPolicy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateControlFlowFlags(t *testing.T) {
	disabled := clientpb.ControlFlowPolicy_CONTROL_FLOW_DISABLED
	balanced := clientpb.ControlFlowPolicy_CONTROL_FLOW_BALANCED_V1

	if err := validateControlFlowFlags(disabled, true, true); err != nil {
		t.Fatalf("disabled control flow should not constrain other flags: %v", err)
	}
	if err := validateControlFlowFlags(balanced, false, false); err != nil {
		t.Fatalf("balanced control flow with symbol obfuscation should be valid: %v", err)
	}
	if err := validateControlFlowFlags(balanced, true, false); err == nil {
		t.Fatal("expected --debug conflict")
	}
	if err := validateControlFlowFlags(balanced, false, true); err == nil {
		t.Fatal("expected --skip-symbols conflict")
	}
}

func TestApplyGenerateBeaconFormSetsControlFlow(t *testing.T) {
	generateCmd := commandByUse(Commands(&console.SliverClient{}), consts.GenerateStr)
	if generateCmd == nil {
		t.Fatalf("missing %q command", consts.GenerateStr)
	}
	beaconCmd := commandByUse(generateCmd.Commands(), consts.BeaconStr)
	if beaconCmd == nil {
		t.Fatalf("missing %q subcommand", consts.BeaconStr)
	}

	err := applyGenerateBeaconForm(beaconCmd, &forms.GenerateBeaconFormResult{
		OS:          "darwin",
		Arch:        "arm64",
		Format:      "exe",
		C2Type:      "mtls",
		C2Value:     "127.0.0.1:31337",
		ControlFlow: "balanced-v1",
	})
	if err != nil {
		t.Fatalf("applyGenerateBeaconForm() error = %v", err)
	}
	if got, err := beaconCmd.Flags().GetString("control-flow"); err != nil || got != "balanced-v1" {
		t.Fatalf("control-flow flag = %q, %v; want balanced-v1", got, err)
	}
}
