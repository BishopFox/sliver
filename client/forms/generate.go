package forms

import (
	"errors"
	"strings"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/util"
	"github.com/charmbracelet/huh"
)

// ErrUserAborted exposes the huh user-abort sentinel for callers.
var ErrUserAborted = huh.ErrUserAborted

// GenerateFormResult captures the inputs needed to drive the generate command.
type GenerateFormResult struct {
	OS      string
	Arch    string
	Format  string
	Name    string
	C2Type  string
	C2Value string
	Save    string
}

// GenerateForm prompts for core generate flags and returns the collected values.
func GenerateForm(compiler *clientpb.Compiler) (*GenerateFormResult, error) {
	result := &GenerateFormResult{
		OS:     "windows",
		Arch:   "amd64",
		Format: "exe",
		C2Type: "mtls",
	}
	commonPlatformsOnly := true
	lastCommonPlatformsOnly := commonPlatformsOnly
	compilerTargets := compilerTargetList(compiler)
	targetBindings := &targetOptionBindings{
		Common: &commonPlatformsOnly,
		OS:     &result.OS,
		Arch:   &result.Arch,
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Common Platforms Only").
				Value(&commonPlatformsOnly),
			huh.NewSelect[string]().
				Title("Target operating system").
				OptionsFunc(func() []huh.Option[string] {
					maybeResetTargets(commonPlatformsOnly, &lastCommonPlatformsOnly, compilerTargets, &result.OS, &result.Arch, &result.Format)
					if commonPlatformsOnly || len(compilerTargets) == 0 {
						return commonOSOptions()
					}
					return osOptionsFromTargets(compilerTargets)
				}, targetBindings).
				Value(&result.OS),
			huh.NewSelect[string]().
				Title("CPU architecture").
				OptionsFunc(func() []huh.Option[string] {
					maybeResetTargets(commonPlatformsOnly, &lastCommonPlatformsOnly, compilerTargets, &result.OS, &result.Arch, &result.Format)
					if commonPlatformsOnly || len(compilerTargets) == 0 {
						return commonArchOptions(result.OS)
					}
					return archOptionsFromTargets(compilerTargets, result.OS)
				}, targetBindings).
				Height(3).
				Value(&result.Arch),
			huh.NewSelect[string]().
				Title("Output format").
				OptionsFunc(func() []huh.Option[string] {
					maybeResetTargets(commonPlatformsOnly, &lastCommonPlatformsOnly, compilerTargets, &result.OS, &result.Arch, &result.Format)
					if commonPlatformsOnly || len(compilerTargets) == 0 {
						return commonFormatOptions(result.OS)
					}
					return formatOptionsFromTargets(compilerTargets, result.OS, result.Arch)
				}, targetBindings).
				Height(3).
				Value(&result.Format),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Implant name (optional)").
				Value(&result.Name).
				Validate(func(value string) error {
					if strings.TrimSpace(value) == "" {
						return nil
					}
					return util.AllowedName(strings.ToLower(value))
				}),
			huh.NewSelect[string]().
				Title("C2 transport").
				OptionsFunc(func() []huh.Option[string] {
					options := []huh.Option[string]{
						huh.NewOption("mTLS", "mtls"),
						huh.NewOption("WireGuard", "wg"),
						huh.NewOption("HTTP(S)", "http"),
						huh.NewOption("DNS", "dns"),
					}
					if result.OS == "windows" {
						options = append(options,
							huh.NewOption("Named pipe", "named-pipe"),
							huh.NewOption("TCP pivot", "tcp-pivot"),
						)
					}
					return options
				}, &result.OS).
				Height(6).
				Value(&result.C2Type),
			huh.NewInput().
				Title("C2 connection string(s)").
				Description("Comma separated; example: 1.2.3.4:8888 or https://c2.example.com").
				Value(&result.C2Value).
				Validate(func(value string) error {
					if strings.TrimSpace(value) == "" {
						return errors.New("connection string required")
					}
					return nil
				}),
			huh.NewInput().
				Title("Save path (optional)").
				Description("File or directory; defaults to the current directory").
				Value(&result.Save),
		),
	)

	if err := form.Run(); err != nil {
		return nil, err
	}

	return result, nil
}
