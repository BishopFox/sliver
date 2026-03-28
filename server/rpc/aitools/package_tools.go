package aitools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bishopfox/sliver/client/packages"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	serverai "github.com/bishopfox/sliver/server/ai"
	serverassets "github.com/bishopfox/sliver/server/assets"
)

const (
	aiAliasDefaultAssemblyArch = "x84"
	aiAliasDefaultRuntime      = "v4.0.30319"
)

var aiAliasDefaultHostProcess = map[string]string{
	"windows": `c:\windows\system32\notepad.exe`,
	"linux":   "/bin/bash",
	"darwin":  "/Applications/Safari.app/Contents/MacOS/SafariForWebKitDevelopment",
}

type searchPackagesArgs struct {
	targetArgs
	Query          string `json:"query,omitempty"`
	OnlyCompatible bool   `json:"only_compatible,omitempty"`
	MaxResults     int    `json:"max_results,omitempty"`
}

type executeAliasArgs struct {
	targetArgs
	CommandName string   `json:"command_name,omitempty"`
	RootPath    string   `json:"root_path,omitempty"`
	Args        []string `json:"args,omitempty"`
	Process     string   `json:"process,omitempty"`
	ProcessArgs []string `json:"process_args,omitempty"`
	PPID        uint32   `json:"ppid,omitempty"`

	Arch       string `json:"arch,omitempty"`
	Method     string `json:"method,omitempty"`
	ClassName  string `json:"class_name,omitempty"`
	AppDomain  string `json:"app_domain,omitempty"`
	InProcess  bool   `json:"in_process,omitempty"`
	Runtime    string `json:"runtime,omitempty"`
	AmsiBypass bool   `json:"amsi_bypass,omitempty"`
	EtwBypass  bool   `json:"etw_bypass,omitempty"`
}

type executeExtensionArgs struct {
	targetArgs
	CommandName string         `json:"command_name,omitempty"`
	RootPath    string         `json:"root_path,omitempty"`
	Args        []string       `json:"args,omitempty"`
	NamedArgs   map[string]any `json:"named_args,omitempty"`
}

type packageTargetResult struct {
	TargetType string `json:"target_type,omitempty"`
	SessionID  string `json:"session_id,omitempty"`
	BeaconID   string `json:"beacon_id,omitempty"`
	Hostname   string `json:"hostname,omitempty"`
	OS         string `json:"os,omitempty"`
	Arch       string `json:"arch,omitempty"`
}

type packageArgumentResult struct {
	Name     string   `json:"name"`
	Type     string   `json:"type,omitempty"`
	Desc     string   `json:"desc,omitempty"`
	Optional bool     `json:"optional"`
	Default  any      `json:"default,omitempty"`
	Choices  []string `json:"choices,omitempty"`
}

type packageOutputSchemaResult struct {
	Name    string   `json:"name,omitempty"`
	Columns []string `json:"columns,omitempty"`
	GroupBy string   `json:"group_by,omitempty"`
}

type aliasSearchResult struct {
	Name                 string                     `json:"name"`
	CommandName          string                     `json:"command_name"`
	Version              string                     `json:"version,omitempty"`
	Help                 string                     `json:"help,omitempty"`
	LongHelp             string                     `json:"long_help,omitempty"`
	OriginalAuthor       string                     `json:"original_author,omitempty"`
	RepoURL              string                     `json:"repo_url,omitempty"`
	RootPath             string                     `json:"root_path"`
	Entrypoint           string                     `json:"entrypoint,omitempty"`
	DefaultArgs          string                     `json:"default_args,omitempty"`
	AllowArgs            bool                       `json:"allow_args"`
	IsReflective         bool                       `json:"is_reflective"`
	IsAssembly           bool                       `json:"is_assembly"`
	ExecutionMode        string                     `json:"execution_mode"`
	SupportedPlatforms   []string                   `json:"supported_platforms,omitempty"`
	Arguments            []packageArgumentResult    `json:"arguments,omitempty"`
	OutputSchema         *packageOutputSchemaResult `json:"output_schema,omitempty"`
	Compatible           bool                       `json:"compatible"`
	CompatibilityChecked bool                       `json:"compatibility_checked"`
	CompatibilityReason  string                     `json:"compatibility_reason,omitempty"`
	ArtifactPath         string                     `json:"artifact_path,omitempty"`
}

type extensionSearchResult struct {
	Name                   string                     `json:"name"`
	PackageName            string                     `json:"package_name,omitempty"`
	CommandName            string                     `json:"command_name"`
	Version                string                     `json:"version,omitempty"`
	Help                   string                     `json:"help,omitempty"`
	LongHelp               string                     `json:"long_help,omitempty"`
	ExtensionAuthor        string                     `json:"extension_author,omitempty"`
	OriginalAuthor         string                     `json:"original_author,omitempty"`
	RepoURL                string                     `json:"repo_url,omitempty"`
	RootPath               string                     `json:"root_path"`
	Entrypoint             string                     `json:"entrypoint,omitempty"`
	DependsOn              string                     `json:"depends_on,omitempty"`
	ExecutionMode          string                     `json:"execution_mode"`
	SupportedPlatforms     []string                   `json:"supported_platforms,omitempty"`
	Arguments              []packageArgumentResult    `json:"arguments,omitempty"`
	OutputSchema           *packageOutputSchemaResult `json:"output_schema,omitempty"`
	Compatible             bool                       `json:"compatible"`
	CompatibilityChecked   bool                       `json:"compatibility_checked"`
	CompatibilityReason    string                     `json:"compatibility_reason,omitempty"`
	ArtifactPath           string                     `json:"artifact_path,omitempty"`
	DependencyAvailable    bool                       `json:"dependency_available,omitempty"`
	DependencyRootPath     string                     `json:"dependency_root_path,omitempty"`
	DependencyArtifactPath string                     `json:"dependency_artifact_path,omitempty"`
}

type aliasSearchResponse struct {
	Query         string               `json:"query,omitempty"`
	StoreDir      string               `json:"store_dir"`
	Target        *packageTargetResult `json:"target,omitempty"`
	TotalMatches  int                  `json:"total_matches"`
	ReturnedCount int                  `json:"returned_count"`
	Results       []aliasSearchResult  `json:"results"`
	Warnings      []string             `json:"warnings,omitempty"`
}

type extensionSearchResponse struct {
	Query         string                  `json:"query,omitempty"`
	StoreDir      string                  `json:"store_dir"`
	Target        *packageTargetResult    `json:"target,omitempty"`
	TotalMatches  int                     `json:"total_matches"`
	ReturnedCount int                     `json:"returned_count"`
	Results       []extensionSearchResult `json:"results"`
	Warnings      []string                `json:"warnings,omitempty"`
}

type aliasExecutionResult struct {
	CommandName     string               `json:"command_name"`
	Name            string               `json:"name"`
	RootPath        string               `json:"root_path"`
	Target          *packageTargetResult `json:"target,omitempty"`
	ExecutionMode   string               `json:"execution_mode"`
	ArtifactPath    string               `json:"artifact_path"`
	Entrypoint      string               `json:"entrypoint,omitempty"`
	Process         string               `json:"process,omitempty"`
	ProcessArgs     []string             `json:"process_args,omitempty"`
	Args            []string             `json:"args,omitempty"`
	UsedDefaultArgs bool                 `json:"used_default_args,omitempty"`
	Arch            string               `json:"arch,omitempty"`
	ClassName       string               `json:"class_name,omitempty"`
	Method          string               `json:"method,omitempty"`
	AppDomain       string               `json:"app_domain,omitempty"`
	InProcess       bool                 `json:"in_process,omitempty"`
	Runtime         string               `json:"runtime,omitempty"`
	IsDLL           bool                 `json:"is_dll,omitempty"`
	OutputText      string               `json:"output_text,omitempty"`
	OutputBase64    string               `json:"output_base64,omitempty"`
	Warnings        []string             `json:"warnings,omitempty"`
}

type extensionExecutionResult struct {
	CommandName            string               `json:"command_name"`
	Name                   string               `json:"name"`
	PackageName            string               `json:"package_name,omitempty"`
	RootPath               string               `json:"root_path"`
	Target                 *packageTargetResult `json:"target,omitempty"`
	ExecutionMode          string               `json:"execution_mode"`
	ArtifactPath           string               `json:"artifact_path"`
	DependsOn              string               `json:"depends_on,omitempty"`
	DependencyRootPath     string               `json:"dependency_root_path,omitempty"`
	DependencyArtifactPath string               `json:"dependency_artifact_path,omitempty"`
	RegisteredName         string               `json:"registered_name"`
	Export                 string               `json:"export"`
	Args                   []string             `json:"args,omitempty"`
	OutputText             string               `json:"output_text,omitempty"`
	OutputBase64           string               `json:"output_base64,omitempty"`
	Warnings               []string             `json:"warnings,omitempty"`
}

func packageToolDefinitions() []serverai.AgenticToolDefinition {
	return []serverai.AgenticToolDefinition{
		{
			Name:        "search_aliases",
			Description: "Search server-side AI aliases copied into ~/.sliver/ai/aliases. Results include manifest details and, when a target is selected, compatibility for that session or beacon.",
			Parameters: map[string]any{
				"type": "object",
				"properties": mergeSchemaProperties(map[string]any{
					"query":           map[string]any{"type": "string", "description": "Optional search terms matched against alias names, command names, help text, authors, and argument descriptions."},
					"only_compatible": map[string]any{"type": "boolean", "description": "Only return aliases compatible with the selected target. Requires a target."},
					"max_results":     map[string]any{"type": "integer", "description": "Maximum number of results to return. Defaults to 25 and is capped at 100."},
				}),
				"additionalProperties": false,
			},
		},
		{
			Name:        "search_extensions",
			Description: "Search server-side AI extensions copied into ~/.sliver/ai/extensions. Results include manifest details, dependency availability, and target compatibility when a target is selected.",
			Parameters: map[string]any{
				"type": "object",
				"properties": mergeSchemaProperties(map[string]any{
					"query":           map[string]any{"type": "string", "description": "Optional search terms matched against extension names, command names, help text, authors, dependencies, and argument descriptions."},
					"only_compatible": map[string]any{"type": "boolean", "description": "Only return extensions compatible with the selected target. Requires a target."},
					"max_results":     map[string]any{"type": "integer", "description": "Maximum number of results to return. Defaults to 25 and is capped at 100."},
				}),
				"additionalProperties": false,
			},
		},
		{
			Name:        "execute_alias",
			Description: "Execute a server-side AI alias from ~/.sliver/ai/aliases on the selected session or beacon. Use search_aliases first when you need command metadata or a root_path to disambiguate duplicates.",
			Parameters: map[string]any{
				"type": "object",
				"properties": mergeSchemaProperties(map[string]any{
					"command_name": map[string]any{"type": "string", "description": "Alias command_name to execute."},
					"root_path":    map[string]any{"type": "string", "description": "Optional root_path from search_aliases when more than one package exposes the same command."},
					"args":         map[string]any{"type": "array", "description": "Positional arguments passed to the alias.", "items": map[string]any{"type": "string"}},
					"process":      map[string]any{"type": "string", "description": "Optional process path used for sideload or execute-assembly hosting."},
					"process_args": map[string]any{"type": "array", "description": "Optional arguments for the hosting process.", "items": map[string]any{"type": "string"}},
					"ppid":         map[string]any{"type": "integer", "description": "Optional parent process ID to spoof on Windows."},
					"arch":         map[string]any{"type": "string", "description": "Assembly architecture hint. Defaults to x84 for .NET assemblies."},
					"method":       map[string]any{"type": "string", "description": "Optional .NET method name."},
					"class_name":   map[string]any{"type": "string", "description": "Optional .NET class name."},
					"app_domain":   map[string]any{"type": "string", "description": "Optional .NET AppDomain name."},
					"in_process":   map[string]any{"type": "boolean", "description": "Run .NET assemblies in-process when supported."},
					"runtime":      map[string]any{"type": "string", "description": "Optional .NET runtime version."},
					"amsi_bypass":  map[string]any{"type": "boolean", "description": "Attempt AMSI bypass for in-process .NET assemblies."},
					"etw_bypass":   map[string]any{"type": "boolean", "description": "Attempt ETW bypass for in-process .NET assemblies."},
				}),
				"required":             []string{"command_name"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "execute_extension",
			Description: "Execute a server-side AI extension from ~/.sliver/ai/extensions on the selected session or beacon. BOFs are supported and dependencies such as coff-loader are registered automatically when present in the server-side store.",
			Parameters: map[string]any{
				"type": "object",
				"properties": mergeSchemaProperties(map[string]any{
					"command_name": map[string]any{"type": "string", "description": "Extension command_name to execute."},
					"root_path":    map[string]any{"type": "string", "description": "Optional root_path from search_extensions when more than one package exposes the same command."},
					"args":         map[string]any{"type": "array", "description": "Raw command tokens passed to the extension. For BOFs use flag-style tokens such as [\"--pid\", \"1234\"].", "items": map[string]any{"type": "string"}},
					"named_args": map[string]any{
						"type":                 "object",
						"description":          "Optional manifest-keyed arguments for BOFs. Values are converted to flag-style tokens in manifest order.",
						"additionalProperties": map[string]any{"type": []any{"string", "integer", "number", "boolean"}},
					},
				}),
				"required":             []string{"command_name"},
				"additionalProperties": false,
			},
		},
	}
}

func (e *executor) callPackageTool(ctx context.Context, name string, arguments string) (string, bool, error) {
	switch strings.TrimSpace(name) {
	case "search_aliases":
		var args searchPackagesArgs
		if err := decodeToolArgs(arguments, &args); err != nil {
			return "", true, err
		}
		result, err := e.callSearchAliases(ctx, args)
		return result, true, err
	case "search_extensions":
		var args searchPackagesArgs
		if err := decodeToolArgs(arguments, &args); err != nil {
			return "", true, err
		}
		result, err := e.callSearchExtensions(ctx, args)
		return result, true, err
	case "execute_alias":
		var args executeAliasArgs
		if err := decodeToolArgs(arguments, &args); err != nil {
			return "", true, err
		}
		result, err := e.callExecuteAlias(ctx, args)
		return result, true, err
	case "execute_extension":
		var args executeExtensionArgs
		if err := decodeToolArgs(arguments, &args); err != nil {
			return "", true, err
		}
		result, err := e.callExecuteExtension(ctx, args)
		return result, true, err
	default:
		return "", false, nil
	}
}

func (e *executor) callSearchAliases(ctx context.Context, args searchPackagesArgs) (string, error) {
	target, err := e.optionalPackageTarget(ctx, args.SessionID, args.BeaconID)
	if err != nil {
		return "", err
	}
	if args.OnlyCompatible && target == nil {
		return "", fmt.Errorf("only_compatible requires session_id, beacon_id, or a conversation target")
	}

	aliases, warnings := loadAIAliases()
	query := strings.TrimSpace(args.Query)
	results := make([]aliasSearchResult, 0, len(aliases))
	type rankedResult struct {
		score  int
		result aliasSearchResult
	}
	ranked := make([]rankedResult, 0, len(aliases))
	for _, manifest := range aliases {
		if manifest == nil {
			continue
		}
		score := scorePackageQuery(
			query,
			[]string{manifest.CommandName, manifest.Name, manifest.Help},
			append([]string{manifest.LongHelp, manifest.OriginalAuthor, manifest.RepoURL}, aliasArgumentSearchText(manifest.Arguments)...),
		)
		if score == 0 {
			continue
		}

		result := aliasSearchResult{
			Name:               manifest.Name,
			CommandName:        manifest.CommandName,
			Version:            manifest.Version,
			Help:               manifest.Help,
			LongHelp:           manifest.LongHelp,
			OriginalAuthor:     manifest.OriginalAuthor,
			RepoURL:            manifest.RepoURL,
			RootPath:           manifest.RootPath,
			Entrypoint:         manifest.Entrypoint,
			DefaultArgs:        manifest.DefaultArgs,
			AllowArgs:          manifest.AllowArgs,
			IsReflective:       manifest.IsReflective,
			IsAssembly:         manifest.IsAssembly,
			ExecutionMode:      aliasExecutionMode(manifest),
			SupportedPlatforms: aliasPlatforms(manifest),
			Arguments:          aliasArgumentsResult(manifest.Arguments),
			OutputSchema:       packageOutputSchema(manifest.Schema),
		}

		if target != nil {
			result.CompatibilityChecked = true
			artifactPath, _, artifactErr := manifest.artifactForTarget(target.OS, target.Arch)
			if artifactErr == nil {
				result.Compatible = true
				result.ArtifactPath = artifactPath
				result.CompatibilityReason = fmt.Sprintf("compatible with %s/%s", target.OS, target.Arch)
			} else {
				result.CompatibilityReason = artifactErr.Error()
			}
		}
		if args.OnlyCompatible && !result.Compatible {
			continue
		}

		ranked = append(ranked, rankedResult{score: score, result: result})
	}

	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].result.Compatible != ranked[j].result.Compatible {
			return ranked[i].result.Compatible
		}
		if ranked[i].score != ranked[j].score {
			return ranked[i].score > ranked[j].score
		}
		if ranked[i].result.CommandName == ranked[j].result.CommandName {
			return ranked[i].result.RootPath < ranked[j].result.RootPath
		}
		return ranked[i].result.CommandName < ranked[j].result.CommandName
	})

	limit := sanitizePackageSearchLimit(args.MaxResults)
	for _, entry := range ranked {
		results = append(results, entry.result)
		if len(results) == limit {
			break
		}
	}

	return marshalToolResult(aliasSearchResponse{
		Query:         query,
		StoreDir:      serverassets.GetAIAliasesDir(),
		Target:        target,
		TotalMatches:  len(ranked),
		ReturnedCount: len(results),
		Results:       results,
		Warnings:      warnings,
	})
}

func (e *executor) callSearchExtensions(ctx context.Context, args searchPackagesArgs) (string, error) {
	target, err := e.optionalPackageTarget(ctx, args.SessionID, args.BeaconID)
	if err != nil {
		return "", err
	}
	if args.OnlyCompatible && target == nil {
		return "", fmt.Errorf("only_compatible requires session_id, beacon_id, or a conversation target")
	}

	extensions, warnings := loadAIExtensions()
	query := strings.TrimSpace(args.Query)
	type rankedResult struct {
		score  int
		result extensionSearchResult
	}
	ranked := make([]rankedResult, 0, len(extensions))
	for _, manifest := range extensions {
		if manifest == nil {
			continue
		}
		for _, command := range manifest.ExtCommand {
			if command == nil {
				continue
			}
			score := scorePackageQuery(
				query,
				[]string{command.CommandName, manifest.Name, manifest.PackageName, command.Help},
				append([]string{command.LongHelp, manifest.ExtensionAuthor, manifest.OriginalAuthor, manifest.RepoURL, command.DependsOn}, extensionArgumentSearchText(command.Arguments)...),
			)
			if score == 0 {
				continue
			}

			result := extensionSearchResult{
				Name:               manifest.Name,
				PackageName:        manifest.PackageName,
				CommandName:        command.CommandName,
				Version:            manifest.Version,
				Help:               command.Help,
				LongHelp:           command.LongHelp,
				ExtensionAuthor:    manifest.ExtensionAuthor,
				OriginalAuthor:     manifest.OriginalAuthor,
				RepoURL:            manifest.RepoURL,
				RootPath:           manifest.RootPath,
				Entrypoint:         command.Entrypoint,
				DependsOn:          command.DependsOn,
				ExecutionMode:      extensionExecutionMode(command),
				SupportedPlatforms: extensionPlatforms(command),
				Arguments:          extensionArgumentsResult(command.Arguments),
				OutputSchema:       packageOutputSchema(command.Schema),
			}

			if target != nil {
				result.CompatibilityChecked = true
				artifactPath, _, artifactErr := command.artifactForTarget(target.OS, target.Arch)
				if artifactErr == nil {
					result.ArtifactPath = artifactPath
					result.Compatible = true
					result.CompatibilityReason = fmt.Sprintf("compatible with %s/%s", target.OS, target.Arch)
				} else {
					result.CompatibilityReason = artifactErr.Error()
				}

				if result.Compatible && strings.TrimSpace(command.DependsOn) != "" {
					dependency, depErr := selectAIExtensionCommand(extensions, command.DependsOn, "", target.OS, target.Arch)
					if depErr != nil {
						result.Compatible = false
						result.CompatibilityReason = depErr.Error()
					} else {
						dependencyArtifactPath, _, depArtifactErr := dependency.artifactForTarget(target.OS, target.Arch)
						if depArtifactErr != nil {
							result.Compatible = false
							result.CompatibilityReason = depArtifactErr.Error()
						} else {
							result.DependencyAvailable = true
							result.DependencyRootPath = dependency.Manifest.RootPath
							result.DependencyArtifactPath = dependencyArtifactPath
						}
					}
				}
			}
			if args.OnlyCompatible && !result.Compatible {
				continue
			}

			ranked = append(ranked, rankedResult{score: score, result: result})
		}
	}

	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].result.Compatible != ranked[j].result.Compatible {
			return ranked[i].result.Compatible
		}
		if ranked[i].score != ranked[j].score {
			return ranked[i].score > ranked[j].score
		}
		if ranked[i].result.CommandName == ranked[j].result.CommandName {
			return ranked[i].result.RootPath < ranked[j].result.RootPath
		}
		return ranked[i].result.CommandName < ranked[j].result.CommandName
	})

	limit := sanitizePackageSearchLimit(args.MaxResults)
	results := make([]extensionSearchResult, 0, minInt(limit, len(ranked)))
	for _, entry := range ranked {
		results = append(results, entry.result)
		if len(results) == limit {
			break
		}
	}

	return marshalToolResult(extensionSearchResponse{
		Query:         query,
		StoreDir:      serverassets.GetAIExtensionsDir(),
		Target:        target,
		TotalMatches:  len(ranked),
		ReturnedCount: len(results),
		Results:       results,
		Warnings:      warnings,
	})
}

func (e *executor) callExecuteAlias(ctx context.Context, args executeAliasArgs) (string, error) {
	session, beacon, err := e.lookupTargetMetadata(ctx, args.SessionID, args.BeaconID)
	if err != nil {
		return "", err
	}
	targetInfo := packageTargetFromMetadata(session, beacon)
	target := toolTargetFromMetadata(session, beacon)

	aliases, _ := loadAIAliases()
	manifest, err := selectAIAlias(aliases, args.CommandName, args.RootPath, targetInfo.OS, targetInfo.Arch)
	if err != nil {
		return "", err
	}

	artifactPath, _, err := manifest.artifactForTarget(targetInfo.OS, targetInfo.Arch)
	if err != nil {
		return "", err
	}
	binData, err := os.ReadFile(artifactPath)
	if err != nil {
		return "", err
	}

	execArgs := copyStringSlice(args.Args)
	usedDefaultArgs := false
	if len(execArgs) == 0 && strings.TrimSpace(manifest.DefaultArgs) != "" {
		execArgs = strings.Fields(manifest.DefaultArgs)
		usedDefaultArgs = true
	}

	warnings := []string{}
	if joinedArgs := strings.TrimSpace(strings.Join(execArgs, " ")); len(joinedArgs) > 256 && (manifest.IsAssembly || !manifest.IsReflective) {
		warnings = append(warnings, "arguments exceed 256 characters and may be truncated by the Donut loader")
	}

	processName := strings.TrimSpace(args.Process)
	if processName == "" {
		processName, err = defaultAliasHostProcess(targetInfo.OS)
		if err != nil {
			return "", err
		}
	}
	processArgs := copyStringSlice(args.ProcessArgs)
	isDLL := strings.EqualFold(filepath.Ext(artifactPath), ".dll")

	result := aliasExecutionResult{
		CommandName:     manifest.CommandName,
		Name:            manifest.Name,
		RootPath:        manifest.RootPath,
		Target:          targetInfo,
		ExecutionMode:   aliasExecutionMode(manifest),
		ArtifactPath:    artifactPath,
		Entrypoint:      manifest.Entrypoint,
		Process:         processName,
		ProcessArgs:     processArgs,
		Args:            execArgs,
		UsedDefaultArgs: usedDefaultArgs,
		IsDLL:           isDLL,
		Warnings:        warnings,
	}

	switch {
	case manifest.IsAssembly:
		arch := strings.TrimSpace(args.Arch)
		if arch == "" {
			arch = aiAliasDefaultAssemblyArch
		}
		runtime := strings.TrimSpace(args.Runtime)
		if runtime == "" && args.InProcess {
			runtime = aiAliasDefaultRuntime
		}
		resp, err := callTargetRPC(
			ctx,
			target,
			func(callCtx context.Context, req *commonpb.Request) (*sliverpb.ExecuteAssembly, error) {
				return e.backend.ExecuteAssembly(callCtx, &sliverpb.ExecuteAssemblyReq{
					Request:     req,
					IsDLL:       isDLL,
					Process:     processName,
					Arguments:   execArgs,
					Assembly:    binData,
					Arch:        arch,
					Method:      strings.TrimSpace(args.Method),
					ClassName:   strings.TrimSpace(args.ClassName),
					AppDomain:   strings.TrimSpace(args.AppDomain),
					ProcessArgs: processArgs,
					PPid:        args.PPID,
					InProcess:   args.InProcess,
					Runtime:     runtime,
					AmsiBypass:  args.AmsiBypass,
					EtwBypass:   args.EtwBypass,
				})
			},
			func() *sliverpb.ExecuteAssembly { return &sliverpb.ExecuteAssembly{} },
		)
		if err != nil {
			return "", err
		}
		result.Arch = arch
		result.Method = strings.TrimSpace(args.Method)
		result.ClassName = strings.TrimSpace(args.ClassName)
		result.AppDomain = strings.TrimSpace(args.AppDomain)
		result.InProcess = args.InProcess
		result.Runtime = runtime
		result.OutputText, result.OutputBase64 = bytesToTextAndBase64(resp.Output)

	case manifest.IsReflective:
		resp, err := callTargetRPC(
			ctx,
			target,
			func(callCtx context.Context, req *commonpb.Request) (*sliverpb.SpawnDll, error) {
				return e.backend.SpawnDll(callCtx, &sliverpb.InvokeSpawnDllReq{
					Request:     req,
					Args:        execArgs,
					Data:        binData,
					ProcessName: processName,
					EntryPoint:  manifest.Entrypoint,
					Kill:        true,
					ProcessArgs: processArgs,
					PPid:        args.PPID,
				})
			},
			func() *sliverpb.SpawnDll { return &sliverpb.SpawnDll{} },
		)
		if err != nil {
			return "", err
		}
		result.OutputText = strings.TrimSpace(resp.Result)

	default:
		resp, err := callTargetRPC(
			ctx,
			target,
			func(callCtx context.Context, req *commonpb.Request) (*sliverpb.Sideload, error) {
				return e.backend.Sideload(callCtx, &sliverpb.SideloadReq{
					Request:     req,
					Data:        binData,
					ProcessName: processName,
					Args:        execArgs,
					EntryPoint:  manifest.Entrypoint,
					Kill:        true,
					IsDLL:       isDLL,
					PPid:        args.PPID,
					ProcessArgs: processArgs,
				})
			},
			func() *sliverpb.Sideload { return &sliverpb.Sideload{} },
		)
		if err != nil {
			return "", err
		}
		result.OutputText = strings.TrimSpace(resp.Result)
	}

	return marshalToolResult(result)
}

func (e *executor) callExecuteExtension(ctx context.Context, args executeExtensionArgs) (string, error) {
	session, beacon, err := e.lookupTargetMetadata(ctx, args.SessionID, args.BeaconID)
	if err != nil {
		return "", err
	}
	targetInfo := packageTargetFromMetadata(session, beacon)
	target := toolTargetFromMetadata(session, beacon)

	extensions, _ := loadAIExtensions()
	command, err := selectAIExtensionCommand(extensions, args.CommandName, args.RootPath, targetInfo.OS, targetInfo.Arch)
	if err != nil {
		return "", err
	}

	artifactPath, _, err := command.artifactForTarget(targetInfo.OS, targetInfo.Arch)
	if err != nil {
		return "", err
	}
	artifactData, err := os.ReadFile(artifactPath)
	if err != nil {
		return "", err
	}

	extensionArgs, err := resolveExtensionArgumentTokens(command, args.Args, args.NamedArgs)
	if err != nil {
		return "", err
	}

	loaded, err := callTargetRPC(
		ctx,
		target,
		func(callCtx context.Context, req *commonpb.Request) (*sliverpb.ListExtensions, error) {
			return e.backend.ListExtensions(callCtx, &sliverpb.ListExtensionsReq{Request: req})
		},
		func() *sliverpb.ListExtensions { return &sliverpb.ListExtensions{} },
	)
	if err != nil {
		return "", err
	}

	isBOF := strings.EqualFold(filepath.Ext(artifactPath), ".o")
	warnings := []string{}
	result := extensionExecutionResult{
		CommandName:   command.CommandName,
		Name:          command.Manifest.Name,
		PackageName:   command.Manifest.PackageName,
		RootPath:      command.Manifest.RootPath,
		Target:        targetInfo,
		ExecutionMode: extensionExecutionMode(command),
		ArtifactPath:  artifactPath,
		DependsOn:     command.DependsOn,
		Args:          extensionArgs,
		Warnings:      warnings,
	}

	callName := ""
	callExport := strings.TrimSpace(command.Entrypoint)
	callArgs := []byte{}

	if isBOF {
		if strings.TrimSpace(command.DependsOn) == "" {
			return "", fmt.Errorf("BOF extension %q is missing depends_on", command.CommandName)
		}
		dependency, err := selectAIExtensionCommand(extensions, command.DependsOn, "", targetInfo.OS, targetInfo.Arch)
		if err != nil {
			return "", err
		}
		dependencyArtifactPath, _, err := dependency.artifactForTarget(targetInfo.OS, targetInfo.Arch)
		if err != nil {
			return "", err
		}
		dependencyData, err := os.ReadFile(dependencyArtifactPath)
		if err != nil {
			return "", err
		}
		dependencyHash, err := e.ensureExtensionRegistered(ctx, target, loaded.Names, dependency, dependencyArtifactPath, dependencyData, targetInfo.OS)
		if err != nil {
			return "", err
		}
		callArgs, err = buildBOFExtensionArgs(command, artifactData, extensionArgs)
		if err != nil {
			return "", err
		}
		callName = dependencyHash
		callExport = strings.TrimSpace(dependency.Entrypoint)
		result.DependencyRootPath = dependency.Manifest.RootPath
		result.DependencyArtifactPath = dependencyArtifactPath
		result.RegisteredName = dependencyHash
		result.Export = callExport
	} else {
		registeredName, err := e.ensureExtensionRegistered(ctx, target, loaded.Names, command, artifactPath, artifactData, targetInfo.OS)
		if err != nil {
			return "", err
		}
		callName = registeredName
		callExport = strings.TrimSpace(command.Entrypoint)
		callArgs = []byte(strings.Join(extensionArgs, " "))
		result.RegisteredName = registeredName
		result.Export = callExport
	}

	resp, err := callTargetRPC(
		ctx,
		target,
		func(callCtx context.Context, req *commonpb.Request) (*sliverpb.CallExtension, error) {
			return e.backend.CallExtension(callCtx, &sliverpb.CallExtensionReq{
				Request: req,
				Name:    callName,
				Export:  callExport,
				Args:    callArgs,
			})
		},
		func() *sliverpb.CallExtension { return &sliverpb.CallExtension{} },
	)
	if err != nil {
		return "", err
	}

	result.OutputText, result.OutputBase64 = bytesToTextAndBase64(resp.Output)
	return marshalToolResult(result)
}

func (e *executor) ensureExtensionRegistered(ctx context.Context, target toolTarget, loaded []string, command *aiExtensionCommand, artifactPath string, artifactData []byte, targetOS string) (string, error) {
	registeredName := sha256Hex(artifactData)
	if containsString(loaded, registeredName) {
		return registeredName, nil
	}

	_, err := callTargetRPC(
		ctx,
		target,
		func(callCtx context.Context, req *commonpb.Request) (*sliverpb.RegisterExtension, error) {
			return e.backend.RegisterExtension(callCtx, &sliverpb.RegisterExtensionReq{
				Request: req,
				Name:    registeredName,
				Data:    artifactData,
				OS:      targetOS,
				Init:    strings.TrimSpace(command.Init),
			})
		},
		func() *sliverpb.RegisterExtension { return &sliverpb.RegisterExtension{} },
	)
	if err != nil {
		return "", fmt.Errorf("failed to register extension %s from %s: %w", command.CommandName, artifactPath, err)
	}
	return registeredName, nil
}

func (e *executor) optionalPackageTarget(ctx context.Context, sessionID string, beaconID string) (*packageTargetResult, error) {
	sessionID = strings.TrimSpace(sessionID)
	beaconID = strings.TrimSpace(beaconID)
	if sessionID == "" && beaconID == "" && e != nil && e.conversation != nil {
		sessionID = strings.TrimSpace(e.conversation.GetTargetSessionID())
		beaconID = strings.TrimSpace(e.conversation.GetTargetBeaconID())
	}
	if sessionID == "" && beaconID == "" {
		return nil, nil
	}
	session, beacon, err := e.lookupTargetMetadata(ctx, sessionID, beaconID)
	if err != nil {
		return nil, err
	}
	return packageTargetFromMetadata(session, beacon), nil
}

func packageTargetFromMetadata(session *clientpb.Session, beacon *clientpb.Beacon) *packageTargetResult {
	if session != nil {
		return &packageTargetResult{
			TargetType: "session",
			SessionID:  session.ID,
			Hostname:   session.Hostname,
			OS:         strings.ToLower(strings.TrimSpace(session.OS)),
			Arch:       strings.ToLower(strings.TrimSpace(session.Arch)),
		}
	}
	if beacon != nil {
		return &packageTargetResult{
			TargetType: "beacon",
			BeaconID:   beacon.ID,
			Hostname:   beacon.Hostname,
			OS:         strings.ToLower(strings.TrimSpace(beacon.OS)),
			Arch:       strings.ToLower(strings.TrimSpace(beacon.Arch)),
		}
	}
	return nil
}

func toolTargetFromMetadata(session *clientpb.Session, beacon *clientpb.Beacon) toolTarget {
	if session != nil {
		return toolTarget{SessionID: session.ID}
	}
	if beacon != nil {
		return toolTarget{BeaconID: beacon.ID}
	}
	return toolTarget{}
}

func aliasArgumentsResult(arguments []*aiAliasArgument) []packageArgumentResult {
	results := make([]packageArgumentResult, 0, len(arguments))
	for _, arg := range arguments {
		if arg == nil {
			continue
		}
		results = append(results, packageArgumentResult{
			Name:     arg.Name,
			Type:     arg.Type,
			Desc:     arg.Desc,
			Optional: arg.Optional,
			Default:  arg.Default,
			Choices:  copyStringSlice(arg.Choices),
		})
	}
	return results
}

func extensionArgumentsResult(arguments []*aiExtensionArgument) []packageArgumentResult {
	results := make([]packageArgumentResult, 0, len(arguments))
	for _, arg := range arguments {
		if arg == nil {
			continue
		}
		results = append(results, packageArgumentResult{
			Name:     arg.Name,
			Type:     arg.Type,
			Desc:     arg.Desc,
			Optional: arg.Optional,
			Default:  arg.Default,
			Choices:  copyStringSlice(arg.Choices),
		})
	}
	return results
}

func packageOutputSchema(schema *packages.OutputSchema) *packageOutputSchemaResult {
	if schema == nil {
		return nil
	}
	return &packageOutputSchemaResult{
		Name:    schema.Name,
		Columns: copyStringSlice(schema.Columns()),
		GroupBy: schema.GroupBy,
	}
}

func aliasArgumentSearchText(arguments []*aiAliasArgument) []string {
	results := make([]string, 0, len(arguments)*2)
	for _, arg := range arguments {
		if arg == nil {
			continue
		}
		results = append(results, arg.Name, arg.Desc)
	}
	return results
}

func extensionArgumentSearchText(arguments []*aiExtensionArgument) []string {
	results := make([]string, 0, len(arguments)*2)
	for _, arg := range arguments {
		if arg == nil {
			continue
		}
		results = append(results, arg.Name, arg.Desc)
	}
	return results
}

func aliasExecutionMode(manifest *aiAliasManifest) string {
	switch {
	case manifest == nil:
		return ""
	case manifest.IsAssembly:
		return "assembly"
	case manifest.IsReflective:
		return "reflective_dll"
	default:
		return "sideload"
	}
}

func extensionExecutionMode(command *aiExtensionCommand) string {
	if command == nil {
		return ""
	}
	for _, extFile := range command.Files {
		if extFile == nil {
			continue
		}
		if strings.EqualFold(filepath.Ext(extFile.Path), ".o") {
			return "bof"
		}
		break
	}
	return "extension"
}

func scorePackageQuery(query string, primaryFields []string, secondaryFields []string) int {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return 1
	}

	score := 0
	joinedPrimary := strings.ToLower(strings.Join(primaryFields, "\n"))
	joinedSecondary := strings.ToLower(strings.Join(secondaryFields, "\n"))
	if strings.Contains(joinedPrimary, query) {
		score += 25
	} else if strings.Contains(joinedSecondary, query) {
		score += 8
	}

	for _, token := range packageSearchTokens(query) {
		if strings.Contains(joinedPrimary, token) {
			score += 10
			continue
		}
		if strings.Contains(joinedSecondary, token) {
			score += 3
		}
	}
	return score
}

func packageSearchTokens(query string) []string {
	fields := strings.FieldsFunc(query, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	})
	results := make([]string, 0, len(fields))
	seen := map[string]struct{}{}
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if len(field) < 2 {
			continue
		}
		if _, ok := seen[field]; ok {
			continue
		}
		seen[field] = struct{}{}
		results = append(results, field)
	}
	return results
}

func sanitizePackageSearchLimit(limit int) int {
	switch {
	case limit <= 0:
		return 25
	case limit > 100:
		return 100
	default:
		return limit
	}
}

func defaultAliasHostProcess(targetOS string) (string, error) {
	process, ok := aiAliasDefaultHostProcess[strings.ToLower(strings.TrimSpace(targetOS))]
	if !ok {
		return "", fmt.Errorf("no default host process for %s target; provide process explicitly", targetOS)
	}
	return process, nil
}

func resolveExtensionArgumentTokens(command *aiExtensionCommand, args []string, namedArgs map[string]any) ([]string, error) {
	if len(args) > 0 && len(namedArgs) > 0 {
		return nil, fmt.Errorf("provide either args or named_args, not both")
	}
	if len(namedArgs) == 0 {
		return copyStringSlice(args), nil
	}
	if len(command.Arguments) == 0 {
		return nil, fmt.Errorf("extension %q does not define named arguments; use args instead", command.CommandName)
	}

	remaining := map[string]any{}
	for key, value := range namedArgs {
		remaining[key] = value
	}

	tokens := []string{}
	for _, arg := range command.Arguments {
		if arg == nil {
			continue
		}
		raw, ok := remaining[arg.Name]
		if !ok {
			continue
		}
		if raw == nil {
			return nil, fmt.Errorf("named argument %q cannot be null", arg.Name)
		}
		tokens = append(tokens, "--"+arg.Name, fmt.Sprint(raw))
		delete(remaining, arg.Name)
	}
	if len(remaining) > 0 {
		keys := make([]string, 0, len(remaining))
		for key := range remaining {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		return nil, fmt.Errorf("unknown named_args for extension %q: %s", command.CommandName, strings.Join(keys, ", "))
	}
	return tokens, nil
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == strings.TrimSpace(needle) {
			return true
		}
	}
	return false
}

func copyStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	return out
}

func minInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}
