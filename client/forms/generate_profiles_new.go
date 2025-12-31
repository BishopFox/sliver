package forms

import (
	"errors"
	"strings"

	"github.com/bishopfox/sliver/util"
	"github.com/charmbracelet/huh"
)

// GenerateProfilesNewFormResult captures the inputs needed to drive the profiles new command.
type GenerateProfilesNewFormResult struct {
	ProfileName string
	OS          string
	Arch        string
	Format      string
	Name        string
	C2Type      string
	C2Value     string
}

// GenerateProfilesNewForm prompts for profiles new flags and returns the collected values.
func GenerateProfilesNewForm() (*GenerateProfilesNewFormResult, error) {
	result := &GenerateProfilesNewFormResult{
		OS:     "windows",
		Arch:   "amd64",
		Format: "exe",
		C2Type: "mtls",
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Profile name").
				Value(&result.ProfileName).
				Validate(func(value string) error {
					trimmed := strings.TrimSpace(value)
					if trimmed == "" {
						return errors.New("profile name required")
					}
					return util.AllowedName(trimmed)
				}),
			huh.NewSelect[string]().
				Title("Target operating system").
				Options(
					huh.NewOption("Windows", "windows"),
					huh.NewOption("Linux", "linux"),
					huh.NewOption("macOS", "darwin"),
				).
				Value(&result.OS),
			huh.NewSelect[string]().
				Title("CPU architecture").
				OptionsFunc(func() []huh.Option[string] {
					switch result.OS {
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
				}, &result.OS).
				Height(3).
				Value(&result.Arch),
			huh.NewSelect[string]().
				Title("Output format").
				OptionsFunc(func() []huh.Option[string] {
					options := []huh.Option[string]{
						huh.NewOption("Executable", "exe"),
						huh.NewOption("Shared library", "shared"),
					}
					if result.OS == "windows" {
						options = append(options,
							huh.NewOption("Service", "service"),
							huh.NewOption("Shellcode", "shellcode"),
						)
					}
					return options
				}, &result.OS).
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
		),
	)

	if err := form.Run(); err != nil {
		return nil, err
	}

	return result, nil
}
