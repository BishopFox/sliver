package generate

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

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
	"errors"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/implant"
	"github.com/bishopfox/sliver/protobuf/clientpb"
)

func TestValidateControlFlowConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *clientpb.ImplantConfig
		wantErr string
	}{
		{
			name:   "disabled is backward compatible",
			config: &clientpb.ImplantConfig{},
		},
		{
			name: "balanced v1",
			config: &clientpb.ImplantConfig{
				ObfuscateSymbols: true,
				ControlFlow:      clientpb.ControlFlowPolicy_CONTROL_FLOW_BALANCED_V1,
				TemplateName:     SliverTemplateName,
			},
		},
		{
			name: "requires symbol obfuscation",
			config: &clientpb.ImplantConfig{
				ControlFlow: clientpb.ControlFlowPolicy_CONTROL_FLOW_BALANCED_V1,
			},
			wantErr: "requires symbol obfuscation",
		},
		{
			name: "rejects debug builds",
			config: &clientpb.ImplantConfig{
				Debug:            true,
				ObfuscateSymbols: true,
				ControlFlow:      clientpb.ControlFlowPolicy_CONTROL_FLOW_BALANCED_V1,
			},
			wantErr: "not supported for debug builds",
		},
		{
			name: "rejects unsupported template",
			config: &clientpb.ImplantConfig{
				ObfuscateSymbols: true,
				ControlFlow:      clientpb.ControlFlowPolicy_CONTROL_FLOW_BALANCED_V1,
				TemplateName:     "custom",
			},
			wantErr: "not supported by implant template",
		},
		{
			name: "rejects unknown policy",
			config: &clientpb.ImplantConfig{
				ObfuscateSymbols: true,
				ControlFlow:      clientpb.ControlFlowPolicy(99),
			},
			wantErr: "unsupported control-flow obfuscation policy",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateControlFlowConfig(test.config)
			if test.wantErr == "" {
				if err != nil {
					t.Fatalf("ValidateControlFlowConfig() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("ValidateControlFlowConfig() error = %v, want substring %q", err, test.wantErr)
			}
		})
	}
}

func TestBalancedV1ControlFlowPolicy(t *testing.T) {
	config := &clientpb.ImplantConfig{
		ObfuscateSymbols: true,
		ControlFlow:      clientpb.ControlFlowPolicy_CONTROL_FLOW_BALANCED_V1,
	}

	if len(balancedV1ControlFlowPolicy.functions) != 3 {
		t.Fatalf("balanced-v1 function count = %d, want 3", len(balancedV1ControlFlowPolicy.functions))
	}
	for functionID := range balancedV1ControlFlowPolicy.functions {
		directive, enabled, err := controlFlowDirective(config, functionID)
		if err != nil {
			t.Fatalf("controlFlowDirective(%q) error = %v", functionID, err)
		}
		if !enabled {
			t.Fatalf("controlFlowDirective(%q) unexpectedly disabled", functionID)
		}
		if directive != balancedV1Directive {
			t.Fatalf("controlFlowDirective(%q) = %q, want %q", functionID, directive, balancedV1Directive)
		}
	}

	if _, _, err := controlFlowDirective(config, "runner.notAllowlisted"); err == nil {
		t.Fatal("controlFlowDirective accepted a non-allowlisted function")
	}
}

func TestBalancedV1SourceAnnotations(t *testing.T) {
	found := map[string]struct{}{}
	visitedPaths := map[string]struct{}{}
	for _, expected := range balancedV1ControlFlowPolicy.functions {
		if _, visited := visitedPaths[expected.sourcePath]; visited {
			continue
		}
		visitedPaths[expected.sourcePath] = struct{}{}

		source, err := implant.FS.ReadFile(expected.sourcePath)
		if err != nil {
			t.Fatalf("read %s: %v", expected.sourcePath, err)
		}
		functionIDs, err := controlFlowFunctionsInSource(balancedV1ControlFlowPolicy, expected.sourcePath, source)
		if err != nil {
			t.Fatalf("validate %s: %v", expected.sourcePath, err)
		}
		for _, functionID := range functionIDs {
			if _, duplicate := found[functionID]; duplicate {
				t.Fatalf("control-flow directive for %s appears more than once", functionID)
			}
			found[functionID] = struct{}{}
		}
	}

	if len(found) != len(balancedV1ControlFlowPolicy.functions) {
		t.Fatalf("found %d balanced-v1 source annotations, want %d", len(found), len(balancedV1ControlFlowPolicy.functions))
	}
}

func TestControlFlowSourceRejectsUnapprovedDirective(t *testing.T) {
	source := []byte(`package handlers

//garble:controlflow flatten_passes=4
func determineDirPathFilter() {}
`)
	_, err := controlFlowFunctionsInSource(
		balancedV1ControlFlowPolicy,
		"sliver/handlers/handlers.go",
		source,
	)
	if err == nil || !strings.Contains(err.Error(), "does not match the selected policy") {
		t.Fatalf("controlFlowFunctionsInSource() error = %v, want policy mismatch", err)
	}
}

func TestControlFlowSourceRejectsAllowlistedNameOnMethod(t *testing.T) {
	source := []byte(`package handlers

type receiver struct{}

//garble:controlflow block_splits=2 junk_jumps=2 flatten_passes=1 flatten_hardening=xor trash_blocks=0
func (receiver) determineDirPathFilter() {}
`)
	_, err := controlFlowFunctionsInSource(
		balancedV1ControlFlowPolicy,
		"sliver/handlers/handlers.go",
		source,
	)
	if err == nil || !strings.Contains(err.Error(), "on method") {
		t.Fatalf("controlFlowFunctionsInSource() error = %v, want method rejection", err)
	}
}

func TestControlFlowSourceScansEveryDirectiveComment(t *testing.T) {
	source := []byte(`package handlers

//garble:controlflow block_splits=2 junk_jumps=2 flatten_passes=1 flatten_hardening=xor trash_blocks=0
//garble:controlflow max
func determineDirPathFilter() {}
`)
	_, err := controlFlowFunctionsInSource(
		balancedV1ControlFlowPolicy,
		"sliver/handlers/handlers.go",
		source,
	)
	if err == nil || !strings.Contains(err.Error(), "does not match the selected policy") {
		t.Fatalf("controlFlowFunctionsInSource() error = %v, want second directive rejection", err)
	}
}

func TestControlFlowCapabilities(t *testing.T) {
	if !HasControlFlowCapability([]string{"unrelated", ControlFlowCapability}) {
		t.Fatal("HasControlFlowCapability did not find advertised capability")
	}
	if HasControlFlowCapability([]string{"unrelated"}) {
		t.Fatal("HasControlFlowCapability accepted an unrelated capability")
	}
}

func TestControlFlowBuildAdmission(t *testing.T) {
	disabledRelease, err := AcquireControlFlowBuildSlot(&clientpb.ImplantConfig{})
	if err != nil {
		t.Fatalf("AcquireControlFlowBuildSlot(disabled) error = %v", err)
	}
	disabledRelease()

	config := &clientpb.ImplantConfig{
		ObfuscateSymbols: true,
		ControlFlow:      clientpb.ControlFlowPolicy_CONTROL_FLOW_BALANCED_V1,
		TemplateName:     SliverTemplateName,
	}
	firstRelease, err := AcquireControlFlowBuildSlot(config)
	if err != nil {
		t.Fatalf("first AcquireControlFlowBuildSlot() error = %v", err)
	}
	defer firstRelease()

	if _, err := AcquireControlFlowBuildSlot(config); !errors.Is(err, ErrControlFlowBuildBusy) {
		t.Fatalf("contended AcquireControlFlowBuildSlot() error = %v, want ErrControlFlowBuildBusy", err)
	}
	firstRelease()

	secondRelease, err := AcquireControlFlowBuildSlot(config)
	if err != nil {
		t.Fatalf("AcquireControlFlowBuildSlot() after release error = %v", err)
	}
	secondRelease()
}
