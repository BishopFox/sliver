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
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	serverAssets "github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/gogo"
	utilAssets "github.com/bishopfox/sliver/util/assets"
)

const (
	// ControlFlowCapability is the protocol/policy contract understood by
	// coordinators and capable external builders.
	ControlFlowCapability = "garble.control-flow/balanced-v1"
	// LocalControlFlowCapability is advertised by a coordinator only when its
	// pinned local Garble runtime is present and verified.
	LocalControlFlowCapability = "garble.control-flow.local/balanced-v1"

	controlFlowDirectivePrefix = "//garble:controlflow"
	balancedV1Directive        = controlFlowDirectivePrefix + " block_splits=2 junk_jumps=2 flatten_passes=1 flatten_hardening=xor trash_blocks=0"
)

type controlFlowPolicy struct {
	directive string
	functions map[string]controlFlowFunction
}

type controlFlowFunction struct {
	sourcePath string
	name       string
}

var (
	// ErrControlFlowBuildBusy is returned instead of allowing an unbounded
	// queue of memory-intensive SSA builds to accumulate on the server.
	ErrControlFlowBuildBusy = errors.New("another control-flow build is already running")
	// ErrControlFlowUnavailable indicates that the extracted compiler asset is
	// absent or does not match Sliver's pinned control-flow-capable artifact.
	ErrControlFlowUnavailable = errors.New("control-flow obfuscation is unavailable")
	controlFlowBuildSlot      = make(chan struct{}, 1)
)

var balancedV1ControlFlowPolicy = &controlFlowPolicy{
	directive: balancedV1Directive,
	functions: map[string]controlFlowFunction{
		"handlers.determineDirPathFilter": {
			sourcePath: "sliver/handlers/handlers.go",
			name:       "determineDirPathFilter",
		},
		"runner.registerSliver": {
			sourcePath: "sliver/runner/runner.go",
			name:       "registerSliver",
		},
		"transports.randomCCDomain": {
			sourcePath: "sliver/transports/transports.go",
			name:       "randomCCDomain",
		},
	},
}

// ValidateControlFlowConfig validates the public control-flow policy before
// any source is rendered or cryptographic build material is created.
func ValidateControlFlowConfig(config *clientpb.ImplantConfig) error {
	_, err := controlFlowPolicyForConfig(config)
	return err
}

func controlFlowPolicyForConfig(config *clientpb.ImplantConfig) (*controlFlowPolicy, error) {
	if config == nil {
		return nil, fmt.Errorf("implant config cannot be nil")
	}

	switch config.ControlFlow {
	case clientpb.ControlFlowPolicy_CONTROL_FLOW_DISABLED:
		return nil, nil
	case clientpb.ControlFlowPolicy_CONTROL_FLOW_BALANCED_V1:
		if !config.ObfuscateSymbols {
			return nil, fmt.Errorf("control-flow obfuscation requires symbol obfuscation")
		}
		if config.Debug {
			return nil, fmt.Errorf("control-flow obfuscation is not supported for debug builds")
		}
		if templateName := strings.TrimSpace(config.TemplateName); templateName != "" && templateName != SliverTemplateName {
			return nil, fmt.Errorf("control-flow obfuscation is not supported by implant template %q", templateName)
		}
		return balancedV1ControlFlowPolicy, nil
	default:
		return nil, fmt.Errorf("unsupported control-flow obfuscation policy: %d", config.ControlFlow)
	}
}

func controlFlowDirective(config *clientpb.ImplantConfig, functionID string) (string, bool, error) {
	policy, err := controlFlowPolicyForConfig(config)
	if err != nil {
		return "", false, err
	}
	if policy == nil {
		return "", false, nil
	}
	if _, ok := policy.functions[functionID]; !ok {
		return "", false, fmt.Errorf("function %q is not part of control-flow policy %s", functionID, config.ControlFlow)
	}
	return policy.directive, true, nil
}

func controlFlowObfuscationEnabled(config *clientpb.ImplantConfig) bool {
	return config != nil && config.ControlFlow == clientpb.ControlFlowPolicy_CONTROL_FLOW_BALANCED_V1
}

// ControlFlowEnabled reports whether an implant config requests a supported
// non-disabled control-flow policy.
func ControlFlowEnabled(config *clientpb.ImplantConfig) bool {
	return controlFlowObfuscationEnabled(config)
}

// HasControlFlowCapability reports whether a server or external builder
// advertises support for Sliver's current control-flow policy contract.
func HasControlFlowCapability(capabilities []string) bool {
	for _, capability := range capabilities {
		if capability == ControlFlowCapability {
			return true
		}
	}
	return false
}

// HasLocalControlFlowCapability reports whether a coordinator can currently
// execute the balanced-v1 policy with its local Garble runtime.
func HasLocalControlFlowCapability(capabilities []string) bool {
	for _, capability := range capabilities {
		if capability == LocalControlFlowCapability {
			return true
		}
	}
	return false
}

// ControlFlowRuntimeError verifies that the host Garble executable is exactly
// the artifact pinned by Sliver's asset manifest. Builder capability
// advertisement and local build admission fail closed when extraction is
// stale or incomplete; the coordinator's protocol capability remains
// independent so healthy external builders can still be used.
func ControlFlowRuntimeError() error {
	garbleName := "garble"
	if runtime.GOOS == "windows" {
		garbleName += ".exe"
	}
	garblePath := filepath.Join(gogo.GetGoRootDir(serverAssets.GetRootAppDir()), "bin", garbleName)
	if err := utilAssets.VerifyGarbleBinary(garblePath, runtime.GOOS, runtime.GOARCH); err != nil {
		return fmt.Errorf("%w: %v", ErrControlFlowUnavailable, err)
	}
	return nil
}

// AcquireControlFlowBuildSlot admits at most one local control-flow build. It
// fails immediately under contention so queued RPCs cannot retain generated
// key material or outlive a canceled client request while waiting for Garble.
func AcquireControlFlowBuildSlot(config *clientpb.ImplantConfig) (func(), error) {
	policy, err := controlFlowPolicyForConfig(config)
	if err != nil {
		return nil, err
	}
	if policy == nil {
		return func() {}, nil
	}
	if err := ControlFlowRuntimeError(); err != nil {
		return nil, err
	}

	select {
	case controlFlowBuildSlot <- struct{}{}:
		var releaseOnce sync.Once
		return func() {
			releaseOnce.Do(func() { <-controlFlowBuildSlot })
		}, nil
	default:
		return nil, ErrControlFlowBuildBusy
	}
}

func controlFlowFunctionsInSource(policy *controlFlowPolicy, sourcePath string, source []byte) ([]string, error) {
	file, err := parser.ParseFile(token.NewFileSet(), sourcePath, source, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse rendered control-flow source %q: %w", sourcePath, err)
	}

	functionIDs := []string{}
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Doc == nil {
			continue
		}
		directiveCount := 0
		for _, comment := range function.Doc.List {
			if !strings.HasPrefix(comment.Text, controlFlowDirectivePrefix) {
				continue
			}
			if comment.Text != policy.directive {
				return nil, fmt.Errorf("control-flow directive on %s in %s does not match the selected policy", function.Name.Name, sourcePath)
			}
			directiveCount++
		}
		if directiveCount == 0 {
			continue
		}
		if directiveCount != 1 {
			return nil, fmt.Errorf("function %s in %s has %d control-flow directives, expected exactly one", function.Name.Name, sourcePath, directiveCount)
		}
		if function.Recv != nil {
			return nil, fmt.Errorf("unexpected control-flow directive on method %s in %s", function.Name.Name, sourcePath)
		}

		functionID := ""
		for candidateID, candidate := range policy.functions {
			if candidate.sourcePath == sourcePath && candidate.name == function.Name.Name {
				functionID = candidateID
				break
			}
		}
		if functionID == "" {
			return nil, fmt.Errorf("unexpected control-flow directive on %s in %s", function.Name.Name, sourcePath)
		}
		functionIDs = append(functionIDs, functionID)
	}
	return functionIDs, nil
}
