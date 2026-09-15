package crack

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	maxCrackBackendInfoLevel uint32 = 2
	maxCrackHashInfoLevel    uint32 = 2
)

type crackInvocationMode uint8

const (
	crackInvocationDashboard crackInvocationMode = iota
	crackInvocationJob
	crackInvocationBackendInfo
	crackInvocationKeyspace
	crackInvocationTotalCandidates
	crackInvocationLookup
	crackInvocationIdentify
	crackInvocationHashInfo
)

type crackInvocation struct {
	mode             crackInvocationMode
	backendInfoLevel uint32
	hashInfoLevel    uint32
	crackstation     string
}

type crackQueryModeSelection struct {
	mode     crackInvocationMode
	flagName string
}

var incompatibleCandidateQueryFlags = map[string]struct{}{
	"benchmark":               {},
	"benchmark-all":           {},
	"benchmark-min":           {},
	"benchmark-max":           {},
	"brain-client":            {},
	"brain-client-features":   {},
	"brain-feed":              {},
	"brain-host":              {},
	"brain-password":          {},
	"brain-port":              {},
	"brain-server":            {},
	"brain-server-timer":      {},
	"brain-session":           {},
	"brain-session-whitelist": {},
	"left":                    {},
	"progress-only":           {},
	"restore":                 {},
	"restore-file":            {},
	"restore-position":        {},
	"restore-show-command":    {},
	"show":                    {},
	"speed-only":              {},
}

// classifyCrackInvocation separates terminal command modes from work that
// belongs in the distributed crack queue. Backend information comes from the
// registration snapshots; the remaining query modes execute synchronously on
// one idle crackstation and never create a CrackJob.
//
//nolint:gocyclo // The ordered flag matrix keeps mutually exclusive query modes and their validation in one decision point.
func classifyCrackInvocation(cmd *cobra.Command, args []string) (crackInvocation, error) {
	flags := cmd.Flags()
	backendInfo, _ := flags.GetBool("backend-info")
	backendInfoLevel, _ := flags.GetUint32("backend-info-level")
	backendInfoLevelChanged := flags.Changed("backend-info-level")
	hashInfo, _ := flags.GetBool("hash-info")
	hashInfoLevel, _ := flags.GetUint32("hash-info-level")
	hashInfoLevelChanged := flags.Changed("hash-info-level")
	keyspace, _ := flags.GetBool("keyspace")
	totalCandidates, _ := flags.GetBool("total-candidates")
	lookup, _ := flags.GetString("lookup")
	lookupChanged := flags.Changed("lookup")
	identify, _ := flags.GetBool("identify-mode")
	crackstation, _ := flags.GetString("crackstation")
	crackstation = strings.TrimSpace(crackstation)

	if flags.Changed("crackstation") && crackstation == "" {
		return crackInvocation{}, fmt.Errorf("--crackstation cannot be empty")
	}
	if lookupChanged && lookup == "" {
		return crackInvocation{}, fmt.Errorf("--lookup cannot be empty")
	}

	selections := make([]crackQueryModeSelection, 0, 6)
	if backendInfo || backendInfoLevelChanged {
		selections = append(selections, crackQueryModeSelection{mode: crackInvocationBackendInfo, flagName: "--backend-info"})
	}
	if keyspace {
		selections = append(selections, crackQueryModeSelection{mode: crackInvocationKeyspace, flagName: "--keyspace"})
	}
	if totalCandidates {
		selections = append(selections, crackQueryModeSelection{mode: crackInvocationTotalCandidates, flagName: "--total-candidates"})
	}
	if lookupChanged {
		selections = append(selections, crackQueryModeSelection{mode: crackInvocationLookup, flagName: "--lookup"})
	}
	if identify {
		selections = append(selections, crackQueryModeSelection{mode: crackInvocationIdentify, flagName: "--identify-mode"})
	}
	if hashInfo || hashInfoLevelChanged {
		selections = append(selections, crackQueryModeSelection{mode: crackInvocationHashInfo, flagName: "--hash-info"})
	}
	if len(selections) > 1 {
		names := make([]string, 0, len(selections))
		for _, selection := range selections {
			names = append(names, selection.flagName)
		}
		sort.Strings(names)
		return crackInvocation{}, fmt.Errorf("crack query modes cannot be combined: %s", strings.Join(names, ", "))
	}
	if len(selections) == 0 {
		if flags.Changed("crackstation") {
			return crackInvocation{}, fmt.Errorf("--crackstation requires --keyspace, --total-candidates, --lookup, --identify-mode, or --hash-info")
		}
		if shouldRunCrack(cmd, args) {
			return crackInvocation{mode: crackInvocationJob}, nil
		}
		return crackInvocation{mode: crackInvocationDashboard}, nil
	}

	invocation := crackInvocation{mode: selections[0].mode, crackstation: crackstation}
	switch invocation.mode {
	case crackInvocationBackendInfo:
		if !backendInfoLevelChanged {
			backendInfoLevel = 1
		}
		if backendInfoLevel == 0 || backendInfoLevel > maxCrackBackendInfoLevel {
			return crackInvocation{}, fmt.Errorf("--backend-info-level must be 1 or %d", maxCrackBackendInfoLevel)
		}
		conflicts := changedCrackFlagConflicts(flags, map[string]struct{}{
			"backend-info": {}, "backend-info-level": {}, "timeout": {},
		})
		if len(args) != 0 {
			conflicts = append(conflicts, "positional hash arguments")
		}
		if len(conflicts) != 0 {
			sort.Strings(conflicts)
			return crackInvocation{}, fmt.Errorf("backend-info query cannot be combined with %s", strings.Join(conflicts, ", "))
		}
		invocation.backendInfoLevel = backendInfoLevel
	case crackInvocationHashInfo:
		if !hashInfoLevelChanged {
			hashInfoLevel = 1
		}
		if hashInfoLevel == 0 || hashInfoLevel > maxCrackHashInfoLevel {
			return crackInvocation{}, fmt.Errorf("--hash-info-level must be 1 or %d", maxCrackHashInfoLevel)
		}
		conflicts := changedCrackFlagConflicts(flags, map[string]struct{}{
			"crackstation": {}, "hash-info": {}, "hash-info-level": {}, "hash-mode": {}, "hash-type": {}, "timeout": {},
		})
		if len(args) != 0 {
			conflicts = append(conflicts, "positional hash arguments")
		}
		if len(conflicts) != 0 {
			sort.Strings(conflicts)
			return crackInvocation{}, fmt.Errorf("hash-info query cannot be combined with %s", strings.Join(conflicts, ", "))
		}
		invocation.hashInfoLevel = hashInfoLevel
	case crackInvocationIdentify:
		conflicts := changedCrackFlagConflicts(flags, map[string]struct{}{
			"crackstation": {}, "hash": {}, "identify-mode": {}, "timeout": {},
		})
		if len(conflicts) != 0 {
			return crackInvocation{}, fmt.Errorf("identify query cannot be combined with %s", strings.Join(conflicts, ", "))
		}
		if !hasCrackIdentifyInput(flags, args) {
			return crackInvocation{}, fmt.Errorf("--identify-mode requires a positional hash or --hash")
		}
	case crackInvocationKeyspace, crackInvocationTotalCandidates, crackInvocationLookup:
		if conflicts := candidateQueryConflicts(flags, args, invocation.mode); len(conflicts) != 0 {
			return crackInvocation{}, fmt.Errorf("%s query cannot be combined with %s", crackInvocationName(invocation.mode), strings.Join(conflicts, ", "))
		}
	}
	return invocation, nil
}

func crackInvocationName(mode crackInvocationMode) string {
	switch mode {
	case crackInvocationBackendInfo:
		return "backend-info"
	case crackInvocationKeyspace:
		return "keyspace"
	case crackInvocationTotalCandidates:
		return "total-candidates"
	case crackInvocationLookup:
		return "lookup"
	case crackInvocationIdentify:
		return "identify"
	case crackInvocationHashInfo:
		return "hash-info"
	default:
		return "crack"
	}
}

func changedCrackFlagConflicts(flags *pflag.FlagSet, allowed map[string]struct{}) []string {
	conflicts := make([]string, 0)
	flags.Visit(func(flag *pflag.Flag) {
		if _, ok := allowed[flag.Name]; ok || changedBoolFlagIsFalse(flags, flag) {
			return
		}
		conflicts = append(conflicts, "--"+flag.Name)
	})
	sort.Strings(conflicts)
	return conflicts
}

func changedBoolFlagIsFalse(flags *pflag.FlagSet, flag *pflag.Flag) bool {
	if flag.Value.Type() != "bool" {
		return false
	}
	value, err := flags.GetBool(flag.Name)
	return err == nil && !value
}

func hasCrackIdentifyInput(flags *pflag.FlagSet, args []string) bool {
	if len(args) != 0 {
		return true
	}
	hashes, _ := flags.GetStringArray("hash")
	return len(hashes) != 0
}

func candidateQueryConflicts(flags *pflag.FlagSet, args []string, mode crackInvocationMode) []string {
	conflicts := make([]string, 0)
	if len(args) != 0 {
		conflicts = append(conflicts, "positional hash arguments")
	}
	flags.Visit(func(flag *pflag.Flag) {
		if changedBoolFlagIsFalse(flags, flag) {
			return
		}
		switch flag.Name {
		case "hash", "credential", "credential-collection":
			conflicts = append(conflicts, "--"+flag.Name)
			return
		case "include-cracked-credentials":
			includeCracked, _ := flags.GetBool(flag.Name)
			if includeCracked {
				conflicts = append(conflicts, "--"+flag.Name)
			}
			return
		case "skip", "limit":
			if mode != crackInvocationLookup {
				value, _ := flags.GetUint64(flag.Name)
				if value != 0 {
					conflicts = append(conflicts, "--"+flag.Name)
				}
			}
			return
		}
		if _, incompatible := incompatibleCandidateQueryFlags[flag.Name]; incompatible {
			conflicts = append(conflicts, "--"+flag.Name)
		}
	})
	sort.Strings(conflicts)
	return conflicts
}
