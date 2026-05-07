package aitools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bishopfox/sliver/client/packages"
	serverassets "github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/util"
)

const (
	aiAliasManifestFileName     = "alias.json"
	aiExtensionManifestFileName = "extension.json"
)

type aiAliasFile struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
	Path string `json:"path"`
}

type aiAliasArgument struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Desc     string   `json:"desc"`
	Optional bool     `json:"optional"`
	Default  any      `json:"default,omitempty"`
	Choices  []string `json:"choices,omitempty"`
}

type aiAliasManifest struct {
	Name           string `json:"name"`
	Version        string `json:"version"`
	CommandName    string `json:"command_name"`
	OriginalAuthor string `json:"original_author"`
	RepoURL        string `json:"repo_url"`
	Help           string `json:"help"`
	LongHelp       string `json:"long_help"`
	Entrypoint     string `json:"entrypoint"`
	AllowArgs      bool   `json:"allow_args"`
	DefaultArgs    string `json:"default_args"`
	IsReflective   bool   `json:"is_reflective"`
	IsAssembly     bool   `json:"is_assembly"`

	Arguments []*aiAliasArgument     `json:"arguments"`
	Files     []*aiAliasFile         `json:"files"`
	Schema    *packages.OutputSchema `json:"schema"`

	RootPath string `json:"-"`
}

type aiExtensionFile struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
	Path string `json:"path"`
}

type aiExtensionArgument struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Desc     string   `json:"desc"`
	Optional bool     `json:"optional"`
	Default  any      `json:"default,omitempty"`
	Choices  []string `json:"choices,omitempty"`
}

type aiExtensionCommand struct {
	CommandName string                 `json:"command_name"`
	Help        string                 `json:"help"`
	LongHelp    string                 `json:"long_help"`
	Entrypoint  string                 `json:"entrypoint"`
	DependsOn   string                 `json:"depends_on"`
	Init        string                 `json:"init"`
	Files       []*aiExtensionFile     `json:"files"`
	Arguments   []*aiExtensionArgument `json:"arguments"`
	Schema      *packages.OutputSchema `json:"schema"`

	Manifest *aiExtensionManifest `json:"-"`
}

type aiExtensionManifest struct {
	Name            string `json:"name"`
	PackageName     string `json:"package_name"`
	Version         string `json:"version"`
	ExtensionAuthor string `json:"extension_author"`
	OriginalAuthor  string `json:"original_author"`
	RepoURL         string `json:"repo_url"`

	ExtCommand []*aiExtensionCommand `json:"commands"`

	RootPath string `json:"-"`
}

type aiLegacyExtensionManifest struct {
	Name            string                 `json:"name"`
	CommandName     string                 `json:"command_name"`
	Version         string                 `json:"version"`
	ExtensionAuthor string                 `json:"extension_author"`
	OriginalAuthor  string                 `json:"original_author"`
	RepoURL         string                 `json:"repo_url"`
	Help            string                 `json:"help"`
	LongHelp        string                 `json:"long_help"`
	Entrypoint      string                 `json:"entrypoint"`
	DependsOn       string                 `json:"depends_on"`
	Init            string                 `json:"init"`
	Files           []*aiExtensionFile     `json:"files"`
	Arguments       []*aiExtensionArgument `json:"arguments"`
	Schema          *packages.OutputSchema `json:"schema"`

	RootPath string `json:"-"`
}

type extensionCandidate struct {
	Command  *aiExtensionCommand
	RootPath string
}

func loadAIAliases() ([]*aiAliasManifest, []string) {
	storeDir := serverassets.GetAIAliasesDir()
	dirEntries, err := os.ReadDir(storeDir)
	if err != nil {
		return nil, []string{fmt.Sprintf("failed to read alias store %s: %s", storeDir, err)}
	}

	aliases := []*aiAliasManifest{}
	warnings := []string{}
	for _, entry := range dirEntries {
		if !entry.IsDir() {
			continue
		}
		manifestPath := filepath.Join(storeDir, entry.Name(), aiAliasManifestFileName)
		manifest, err := loadAIAliasManifest(manifestPath)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("skipping alias manifest %s: %s", manifestPath, err))
			continue
		}
		aliases = append(aliases, manifest)
	}

	sort.Slice(aliases, func(i, j int) bool {
		if aliases[i].CommandName == aliases[j].CommandName {
			return aliases[i].RootPath < aliases[j].RootPath
		}
		return aliases[i].CommandName < aliases[j].CommandName
	})
	sort.Strings(warnings)
	return aliases, warnings
}

func loadAIExtensions() ([]*aiExtensionManifest, []string) {
	storeDir := serverassets.GetAIExtensionsDir()
	dirEntries, err := os.ReadDir(storeDir)
	if err != nil {
		return nil, []string{fmt.Sprintf("failed to read extension store %s: %s", storeDir, err)}
	}

	extensions := []*aiExtensionManifest{}
	warnings := []string{}
	for _, entry := range dirEntries {
		if !entry.IsDir() {
			continue
		}
		manifestPath := filepath.Join(storeDir, entry.Name(), aiExtensionManifestFileName)
		manifest, err := loadAIExtensionManifest(manifestPath)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("skipping extension manifest %s: %s", manifestPath, err))
			continue
		}
		extensions = append(extensions, manifest)
	}

	sort.Slice(extensions, func(i, j int) bool {
		if extensionDisplayName(extensions[i]) == extensionDisplayName(extensions[j]) {
			return extensions[i].RootPath < extensions[j].RootPath
		}
		return extensionDisplayName(extensions[i]) < extensionDisplayName(extensions[j])
	})
	sort.Strings(warnings)
	return extensions, warnings
}

func loadAIAliasManifest(manifestPath string) (*aiAliasManifest, error) {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}

	manifest, err := parseAIAliasManifest(data)
	if err != nil {
		return nil, err
	}
	manifest.RootPath = filepath.Dir(manifestPath)
	return manifest, nil
}

func loadAIExtensionManifest(manifestPath string) (*aiExtensionManifest, error) {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}

	manifest, err := parseAIExtensionManifest(data)
	if err != nil {
		return nil, err
	}
	manifest.RootPath = filepath.Dir(manifestPath)
	for _, command := range manifest.ExtCommand {
		command.Manifest = manifest
	}
	return manifest, nil
}

func parseAIAliasManifest(data []byte) (*aiAliasManifest, error) {
	manifest := &aiAliasManifest{}
	if err := json.Unmarshal(data, manifest); err != nil {
		return nil, err
	}
	if strings.TrimSpace(manifest.Name) == "" {
		return nil, fmt.Errorf("missing alias name in manifest")
	}
	if strings.TrimSpace(manifest.CommandName) == "" {
		return nil, fmt.Errorf("missing command_name in alias manifest")
	}
	if strings.TrimSpace(manifest.Help) == "" {
		return nil, fmt.Errorf("missing help in alias manifest")
	}

	for _, aliasFile := range manifest.Files {
		if aliasFile == nil {
			continue
		}
		if strings.TrimSpace(aliasFile.OS) == "" {
			return nil, fmt.Errorf("missing files.os in alias manifest")
		}
		if strings.TrimSpace(aliasFile.Arch) == "" {
			return nil, fmt.Errorf("missing files.arch in alias manifest")
		}
		aliasFile.OS = strings.ToLower(strings.TrimSpace(aliasFile.OS))
		aliasFile.Arch = strings.ToLower(strings.TrimSpace(aliasFile.Arch))
		aliasFile.Path = util.ResolvePath(aliasFile.Path)
		if aliasFile.Path == "" || aliasFile.Path == string(os.PathSeparator) {
			return nil, fmt.Errorf("missing files.path in alias manifest")
		}
	}

	if manifest.Schema != nil {
		if !packages.IsValidSchemaType(manifest.Schema.Name) {
			return nil, fmt.Errorf("%s is not a valid schema type", manifest.Schema.Name)
		}
		manifest.Schema.IngestColumns()
	}

	return manifest, nil
}

func parseAIExtensionManifest(data []byte) (*aiExtensionManifest, error) {
	manifest := &aiExtensionManifest{}
	if err := json.Unmarshal(data, manifest); err != nil || len(manifest.ExtCommand) == 0 {
		legacy := &aiLegacyExtensionManifest{}
		if legacyErr := json.Unmarshal(data, legacy); legacyErr != nil {
			if err != nil {
				return nil, err
			}
			return nil, legacyErr
		}
		manifest = convertLegacyAIExtensionManifest(legacy)
	}

	if strings.TrimSpace(manifest.Name) == "" {
		return nil, fmt.Errorf("missing name field in extension manifest")
	}

	for _, command := range manifest.ExtCommand {
		if command == nil {
			continue
		}
		command.Manifest = manifest
		if strings.TrimSpace(command.CommandName) == "" {
			return nil, fmt.Errorf("missing command_name field in extension manifest")
		}
		if strings.TrimSpace(command.Help) == "" {
			return nil, fmt.Errorf("missing help field in extension manifest")
		}
		if len(command.Files) == 0 {
			return nil, fmt.Errorf("missing files field in extension manifest")
		}
		for _, extFile := range command.Files {
			if extFile == nil {
				continue
			}
			if strings.TrimSpace(extFile.OS) == "" {
				return nil, fmt.Errorf("missing files.os field in extension manifest")
			}
			if strings.TrimSpace(extFile.Arch) == "" {
				return nil, fmt.Errorf("missing files.arch field in extension manifest")
			}
			extFile.OS = strings.ToLower(strings.TrimSpace(extFile.OS))
			extFile.Arch = strings.ToLower(strings.TrimSpace(extFile.Arch))
			extFile.Path = util.ResolvePath(extFile.Path)
			if extFile.Path == "" || extFile.Path == string(os.PathSeparator) {
				return nil, fmt.Errorf("missing files.path field in extension manifest")
			}
		}
		if command.Schema != nil {
			if !packages.IsValidSchemaType(command.Schema.Name) {
				return nil, fmt.Errorf("%s is not a valid schema type", command.Schema.Name)
			}
			command.Schema.IngestColumns()
		}
	}

	return manifest, nil
}

func convertLegacyAIExtensionManifest(legacy *aiLegacyExtensionManifest) *aiExtensionManifest {
	return &aiExtensionManifest{
		Name:            legacy.CommandName,
		Version:         legacy.Version,
		ExtensionAuthor: legacy.ExtensionAuthor,
		OriginalAuthor:  legacy.OriginalAuthor,
		RepoURL:         legacy.RepoURL,
		RootPath:        legacy.RootPath,
		ExtCommand: []*aiExtensionCommand{
			{
				CommandName: legacy.CommandName,
				Help:        legacy.Help,
				LongHelp:    legacy.LongHelp,
				Entrypoint:  legacy.Entrypoint,
				DependsOn:   legacy.DependsOn,
				Init:        legacy.Init,
				Files:       legacy.Files,
				Arguments:   legacy.Arguments,
				Schema:      legacy.Schema,
			},
		},
	}
}

func selectAIAlias(manifests []*aiAliasManifest, commandName string, rootPath string, targetOS string, targetArch string) (*aiAliasManifest, error) {
	commandName = normalizePackageCommandName(commandName)
	if commandName == "" {
		return nil, fmt.Errorf("command_name is required")
	}

	candidates := []*aiAliasManifest{}
	for _, manifest := range manifests {
		if manifest == nil {
			continue
		}
		if normalizePackageCommandName(manifest.CommandName) != commandName {
			continue
		}
		if rootPath != "" && !samePackageRootPath(manifest.RootPath, rootPath) {
			continue
		}
		candidates = append(candidates, manifest)
	}

	switch len(candidates) {
	case 0:
		if rootPath != "" {
			return nil, fmt.Errorf("alias command %q not found at root_path %q", commandName, rootPath)
		}
		return nil, fmt.Errorf("alias command %q not found in %s", commandName, serverassets.GetAIAliasesDir())
	case 1:
		return candidates[0], nil
	}

	if targetOS != "" && targetArch != "" {
		compatible := []*aiAliasManifest{}
		for _, manifest := range candidates {
			if _, _, err := manifest.artifactForTarget(targetOS, targetArch); err == nil {
				compatible = append(compatible, manifest)
			}
		}
		if len(compatible) == 1 {
			return compatible[0], nil
		}
	}

	paths := make([]string, 0, len(candidates))
	for _, manifest := range candidates {
		paths = append(paths, manifest.RootPath)
	}
	sort.Strings(paths)
	return nil, fmt.Errorf("alias command %q is ambiguous; specify root_path from search_aliases: %s", commandName, strings.Join(paths, ", "))
}

func selectAIExtensionCommand(manifests []*aiExtensionManifest, commandName string, rootPath string, targetOS string, targetArch string) (*aiExtensionCommand, error) {
	commandName = normalizePackageCommandName(commandName)
	if commandName == "" {
		return nil, fmt.Errorf("command_name is required")
	}

	candidates := []extensionCandidate{}
	for _, manifest := range manifests {
		if manifest == nil {
			continue
		}
		if rootPath != "" && !samePackageRootPath(manifest.RootPath, rootPath) {
			continue
		}
		for _, command := range manifest.ExtCommand {
			if command == nil {
				continue
			}
			if normalizePackageCommandName(command.CommandName) == commandName {
				candidates = append(candidates, extensionCandidate{Command: command, RootPath: manifest.RootPath})
			}
		}
	}

	switch len(candidates) {
	case 0:
		if rootPath != "" {
			return nil, fmt.Errorf("extension command %q not found at root_path %q", commandName, rootPath)
		}
		return nil, fmt.Errorf("extension command %q not found in %s", commandName, serverassets.GetAIExtensionsDir())
	case 1:
		return candidates[0].Command, nil
	}

	if targetOS != "" && targetArch != "" {
		compatible := []*aiExtensionCommand{}
		for _, candidate := range candidates {
			if _, _, err := candidate.Command.artifactForTarget(targetOS, targetArch); err == nil {
				compatible = append(compatible, candidate.Command)
			}
		}
		if len(compatible) == 1 {
			return compatible[0], nil
		}
	}

	paths := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		paths = append(paths, candidate.RootPath)
	}
	sort.Strings(paths)
	return nil, fmt.Errorf("extension command %q is ambiguous; specify root_path from search_extensions: %s", commandName, strings.Join(paths, ", "))
}

func (manifest *aiAliasManifest) artifactForTarget(targetOS string, targetArch string) (string, *aiAliasFile, error) {
	for _, aliasFile := range manifest.Files {
		if aliasFile == nil {
			continue
		}
		if aliasFile.OS != strings.ToLower(strings.TrimSpace(targetOS)) || aliasFile.Arch != strings.ToLower(strings.TrimSpace(targetArch)) {
			continue
		}
		artifactPath := joinPackageArtifactPath(manifest.RootPath, aliasFile.Path)
		if _, err := os.Stat(artifactPath); err != nil {
			return "", aliasFile, fmt.Errorf("alias artifact not found: %s", artifactPath)
		}
		return artifactPath, aliasFile, nil
	}
	return "", nil, fmt.Errorf("no alias artifact found for %s/%s", targetOS, targetArch)
}

func (command *aiExtensionCommand) artifactForTarget(targetOS string, targetArch string) (string, *aiExtensionFile, error) {
	for _, extFile := range command.Files {
		if extFile == nil {
			continue
		}
		if extFile.OS != strings.ToLower(strings.TrimSpace(targetOS)) || extFile.Arch != strings.ToLower(strings.TrimSpace(targetArch)) {
			continue
		}
		artifactPath := joinPackageArtifactPath(command.Manifest.RootPath, extFile.Path)
		if _, err := os.Stat(artifactPath); err != nil {
			return "", extFile, fmt.Errorf("extension artifact not found: %s", artifactPath)
		}
		return artifactPath, extFile, nil
	}
	return "", nil, fmt.Errorf("no extension artifact found for %s/%s", targetOS, targetArch)
}

func aliasPlatforms(manifest *aiAliasManifest) []string {
	platforms := map[string]struct{}{}
	for _, aliasFile := range manifest.Files {
		if aliasFile == nil {
			continue
		}
		platforms[fmt.Sprintf("%s/%s", aliasFile.OS, aliasFile.Arch)] = struct{}{}
	}
	return sortedMapKeys(platforms)
}

func extensionPlatforms(command *aiExtensionCommand) []string {
	platforms := map[string]struct{}{}
	for _, extFile := range command.Files {
		if extFile == nil {
			continue
		}
		platforms[fmt.Sprintf("%s/%s", extFile.OS, extFile.Arch)] = struct{}{}
	}
	return sortedMapKeys(platforms)
}

func extensionDisplayName(manifest *aiExtensionManifest) string {
	if manifest == nil {
		return ""
	}
	if strings.TrimSpace(manifest.PackageName) != "" {
		return strings.TrimSpace(manifest.PackageName)
	}
	return strings.TrimSpace(manifest.Name)
}

func normalizePackageCommandName(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func samePackageRootPath(left string, right string) bool {
	if left == "" || right == "" {
		return false
	}
	return filepath.Clean(left) == filepath.Clean(right)
}

func joinPackageArtifactPath(rootPath string, manifestPath string) string {
	trimmed := strings.TrimLeft(strings.ReplaceAll(manifestPath, "\\", "/"), "/")
	return filepath.Join(rootPath, filepath.FromSlash(trimmed))
}

func sortedMapKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
