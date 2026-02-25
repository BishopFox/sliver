package theme

import (
	"errors"
	"image/color"
	"os"
	"path/filepath"
	"sync"

	"charm.land/lipgloss/v2"
	"github.com/bishopfox/sliver/client/assets"
	"gopkg.in/yaml.v3"
)

const (
	ThemeFileName = "theme.yaml"
)

type Palette struct {
	Default string         `yaml:"default"`
	Mods    map[int]string `yaml:"mods"`
}

type Theme struct {
	Primary   Palette `yaml:"primary"`
	Secondary Palette `yaml:"secondary"`
	Default   Palette `yaml:"default"`
	Success   Palette `yaml:"success"`
	Warning   Palette `yaml:"warning"`
	Danger    Palette `yaml:"danger"`
}

var (
	mu      sync.RWMutex
	current Theme
)

func init() {
	// Ensure there is always a usable theme without I/O.
	t, err := parseThemeYAML([]byte(DefaultThemeYAML))
	if err == nil {
		current = t
	} else {
		// Minimal fallback; should never happen unless DefaultThemeYAML is corrupted.
		current = Theme{
			Primary: Palette{Default: "#006FEE", Mods: map[int]string{}},
			Default: Palette{Default: "#ffffff", Mods: map[int]string{}},
		}
	}
}

func ThemePath() string {
	return filepath.Join(assets.GetRootAppDir(), ThemeFileName)
}

func SetCurrentTheme(t Theme) {
	mu.Lock()
	current = t
	mu.Unlock()
}

func EnsureThemeFile() error {
	p := ThemePath()
	if _, err := os.Stat(p); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.WriteFile(p, []byte(DefaultThemeYAML), 0o600)
}

func LoadThemeFromDisk() (Theme, error) {
	data, err := os.ReadFile(ThemePath())
	if err != nil {
		return Theme{}, err
	}
	return parseThemeYAML(data)
}

func LoadAndSetCurrentTheme() error {
	_ = EnsureThemeFile()
	t, err := LoadThemeFromDisk()
	if err != nil {
		return err
	}
	SetCurrentTheme(t)
	return nil
}

func parseThemeYAML(data []byte) (Theme, error) {
	var t Theme
	if err := yaml.Unmarshal(data, &t); err != nil {
		return Theme{}, err
	}
	normalizePalette(&t.Primary)
	normalizePalette(&t.Secondary)
	normalizePalette(&t.Default)
	normalizePalette(&t.Success)
	normalizePalette(&t.Warning)
	normalizePalette(&t.Danger)
	return t, nil
}

func normalizePalette(p *Palette) {
	if p.Mods == nil {
		p.Mods = map[int]string{}
	}
}

func (t Theme) paletteColor(p Palette) color.Color {
	if p.Default != "" {
		return lipgloss.Color(p.Default)
	}
	// Defensive fallback.
	return lipgloss.Color("#ffffff")
}

func (t Theme) paletteMod(p Palette, mod int) color.Color {
	if mod != 0 {
		if v, ok := p.Mods[mod]; ok && v != "" {
			return lipgloss.Color(v)
		}
	}
	return t.paletteColor(p)
}

// Primary returns the theme primary color.
func Primary() color.Color { return Current().paletteColor(Current().Primary) }
func PrimaryMod(mod int) color.Color {
	t := Current()
	return t.paletteMod(t.Primary, mod)
}

// Secondary returns the theme secondary color.
func Secondary() color.Color { return Current().paletteColor(Current().Secondary) }
func SecondaryMod(mod int) color.Color {
	t := Current()
	return t.paletteMod(t.Secondary, mod)
}

// Default returns the theme default/neutral color.
func Default() color.Color { return Current().paletteColor(Current().Default) }
func DefaultMod(mod int) color.Color {
	t := Current()
	return t.paletteMod(t.Default, mod)
}

// Success returns the theme success color.
func Success() color.Color { return Current().paletteColor(Current().Success) }
func SuccessMod(mod int) color.Color {
	t := Current()
	return t.paletteMod(t.Success, mod)
}

// Warning returns the theme warning color.
func Warning() color.Color { return Current().paletteColor(Current().Warning) }
func WarningMod(mod int) color.Color {
	t := Current()
	return t.paletteMod(t.Warning, mod)
}

// Danger returns the theme danger color.
func Danger() color.Color { return Current().paletteColor(Current().Danger) }
func DangerMod(mod int) color.Color {
	t := Current()
	return t.paletteMod(t.Danger, mod)
}

// Current returns the in-memory current theme (defaults to DefaultThemeYAML).
func Current() Theme {
	mu.RLock()
	defer mu.RUnlock()
	return current
}
