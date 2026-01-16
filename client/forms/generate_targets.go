package forms

import (
	"sort"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/charmbracelet/huh"
)

type formatOption struct {
	format clientpb.OutputFormat
	label  string
	value  string
}

type targetOptionBindings struct {
	Common *bool
	OS     *string
	Arch   *string
}

var formatOptionOrder = []formatOption{
	{format: clientpb.OutputFormat_EXECUTABLE, label: "Executable", value: "exe"},
	{format: clientpb.OutputFormat_SHARED_LIB, label: "Shared library", value: "shared"},
	{format: clientpb.OutputFormat_SERVICE, label: "Service", value: "service"},
	{format: clientpb.OutputFormat_SHELLCODE, label: "Shellcode", value: "shellcode"},
}

func commonOSOptions() []huh.Option[string] {
	return []huh.Option[string]{
		huh.NewOption("Windows", "windows"),
		huh.NewOption("Linux", "linux"),
		huh.NewOption("macOS", "darwin"),
	}
}

func commonArchOptions(goos string) []huh.Option[string] {
	switch goos {
	case "darwin":
		return []huh.Option[string]{
			huh.NewOption("amd64", "amd64"),
			huh.NewOption("arm64", "arm64"),
		}
	case "linux":
		return []huh.Option[string]{
			huh.NewOption("amd64", "amd64"),
			huh.NewOption("386", "386"),
		}
	default:
		return []huh.Option[string]{
			huh.NewOption("amd64", "amd64"),
			huh.NewOption("386", "386"),
		}
	}
}

func commonFormatOptions(goos string) []huh.Option[string] {
	options := []huh.Option[string]{
		huh.NewOption("Executable", "exe"),
		huh.NewOption("Shared library", "shared"),
	}
	if goos == "windows" {
		options = append(options,
			huh.NewOption("Service", "service"),
			huh.NewOption("Shellcode", "shellcode"),
		)
	}
	return options
}

func firstOptionValue(options []huh.Option[string]) (string, bool) {
	if len(options) == 0 {
		return "", false
	}
	return options[0].Value, true
}

func maybeResetTargets(commonPlatformsOnly bool, lastCommonPlatformsOnly *bool, compilerTargets []*clientpb.CompilerTarget, goos, goarch, format *string) {
	if lastCommonPlatformsOnly == nil || goos == nil || goarch == nil || format == nil {
		return
	}
	if commonPlatformsOnly == *lastCommonPlatformsOnly {
		return
	}
	*lastCommonPlatformsOnly = commonPlatformsOnly
	if commonPlatformsOnly {
		return
	}

	osOptions := commonOSOptions()
	if len(compilerTargets) > 0 {
		osOptions = osOptionsFromTargets(compilerTargets)
		if len(osOptions) == 0 {
			osOptions = commonOSOptions()
		}
	}
	if value, ok := firstOptionValue(osOptions); ok {
		*goos = value
	}

	archOptions := commonArchOptions(*goos)
	if len(compilerTargets) > 0 {
		archOptions = archOptionsFromTargets(compilerTargets, *goos)
		if len(archOptions) == 0 {
			archOptions = commonArchOptions(*goos)
		}
	}
	if value, ok := firstOptionValue(archOptions); ok {
		*goarch = value
	}

	formatOptions := commonFormatOptions(*goos)
	if len(compilerTargets) > 0 {
		formatOptions = formatOptionsFromTargets(compilerTargets, *goos, *goarch)
		if len(formatOptions) == 0 {
			formatOptions = commonFormatOptions(*goos)
		}
	}
	if value, ok := firstOptionValue(formatOptions); ok {
		*format = value
	}
}

func compilerTargetList(compiler *clientpb.Compiler) []*clientpb.CompilerTarget {
	if compiler == nil {
		return nil
	}
	targets := make([]*clientpb.CompilerTarget, 0, len(compiler.Targets)+len(compiler.UnsupportedTargets))
	targets = append(targets, compiler.Targets...)
	targets = append(targets, compiler.UnsupportedTargets...)
	return targets
}

func osOptionsFromTargets(targets []*clientpb.CompilerTarget) []huh.Option[string] {
	if len(targets) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	for _, target := range targets {
		if target.GOOS == "" {
			continue
		}
		seen[target.GOOS] = struct{}{}
	}
	values := make([]string, 0, len(seen))
	for value := range seen {
		values = append(values, value)
	}
	sort.Strings(values)

	options := make([]huh.Option[string], 0, len(values))
	for _, value := range values {
		options = append(options, huh.NewOption(osLabel(value), value))
	}
	return options
}

func archOptionsFromTargets(targets []*clientpb.CompilerTarget, goos string) []huh.Option[string] {
	if len(targets) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	for _, target := range targets {
		if goos != "" && target.GOOS != goos {
			continue
		}
		if target.GOARCH == "" {
			continue
		}
		seen[target.GOARCH] = struct{}{}
	}
	values := make([]string, 0, len(seen))
	for value := range seen {
		values = append(values, value)
	}
	sort.Strings(values)

	options := make([]huh.Option[string], 0, len(values))
	for _, value := range values {
		options = append(options, huh.NewOption(value, value))
	}
	return options
}

func formatOptionsFromTargets(targets []*clientpb.CompilerTarget, goos, goarch string) []huh.Option[string] {
	if len(targets) == 0 {
		return nil
	}
	available := map[clientpb.OutputFormat]struct{}{}
	for _, target := range targets {
		if goos != "" && target.GOOS != goos {
			continue
		}
		if goarch != "" && target.GOARCH != goarch {
			continue
		}
		available[target.Format] = struct{}{}
	}

	options := make([]huh.Option[string], 0, len(available))
	for _, option := range formatOptionOrder {
		if _, ok := available[option.format]; ok {
			options = append(options, huh.NewOption(option.label, option.value))
		}
	}
	return options
}

func osLabel(value string) string {
	switch value {
	case "windows":
		return "Windows"
	case "linux":
		return "Linux"
	case "darwin":
		return "macOS"
	default:
		return value
	}
}
