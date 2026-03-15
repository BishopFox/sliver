# Huh v2 Upgrade Guide

This guide will help you migrate from Huh v1 to v2. Most changes are straightforward, and many are handled automatically by your IDE or `gofmt`.

> [!TIP]
> For a high-level overview of what's new, check out [What's New in Huh v2](WHATS_NEW_V2.md).

## Quick Start

Update your imports and dependencies, and you're 90% done:

```bash
# Update your go.mod
go get charm.land/huh/v2@latest
go get charm.land/bubbletea/v2@latest
go get charm.land/lipgloss/v2@latest
go get charm.land/bubbles/v2@latest
```

Then update your import paths:

```go
// Before
import (
    "github.com/charmbracelet/huh"
    "github.com/charmbracelet/huh/spinner"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    "github.com/charmbracelet/bubbles/key"
)

// After
import (
    "charm.land/huh/v2"
    "charm.land/huh/v2/spinner"
    tea "charm.land/bubbletea/v2"
    "charm.land/lipgloss/v2"
    "charm.land/bubbles/v2/key"
)
```

## Breaking Changes

### Import Paths

All Charm imports now use the `charm.land` vanity domain with a `/v2` version suffix.

| v1 | v2 |
|----|-----|
| `github.com/charmbracelet/huh` | `charm.land/huh/v2` |
| `github.com/charmbracelet/huh/spinner` | `charm.land/huh/v2/spinner` |
| `github.com/charmbracelet/bubbletea` | `charm.land/bubbletea/v2` |
| `github.com/charmbracelet/lipgloss` | `charm.land/lipgloss/v2` |
| `github.com/charmbracelet/bubbles` | `charm.land/bubbles/v2` |

### Theme Changes

Themes are now passed by value and take a `bool` parameter for dark mode detection.

**Before:**
```go
form := huh.NewForm(
    // ...
).WithTheme(huh.ThemeCharm())
```

**After:**
```go
isDark := lipgloss.HasDarkBackground() // or detect however you prefer
form := huh.NewForm(
    // ...
).WithTheme(huh.ThemeCharm(isDark))
```

All built-in themes now follow this pattern:

```go
huh.ThemeCharm(isDark bool) *Styles
huh.ThemeDracula(isDark bool) *Styles
huh.ThemeCatppuccin(isDark bool) *Styles
huh.ThemeBase(isDark bool) *Styles
huh.ThemeBase16(isDark bool) *Styles
```

### Theme Type Changes

The `Theme` type has changed from a struct to an interface:

**Before:**
```go
type Theme struct {
    Form FormStyles
    Group GroupStyles
    FieldSeparator lipgloss.Style
    Blurred FieldStyles
    Focused FieldStyles
    Help help.Styles
}
```

**After:**
```go
type ThemeFunc func(isDark bool) *Styles
```

If you created custom themes, you'll need to update them to this new function signature:

```go
func MyCustomTheme(isDark bool) *Styles {
    styles := &huh.Styles{
        // Your custom styles...
    }
    return styles
}
```

### Field-Level WithAccessible Removed

Individual fields no longer have `WithAccessible()` methods. Accessible mode is now controlled exclusively at the form level, making it simpler and more consistent.

**Before:**
```go
// v1 - each field could have its own accessible setting
input := huh.NewInput().
    Title("Name").
    WithAccessible(true)  // ❌ No longer exists

select := huh.NewSelect[string]().
    Title("Country").
    Options(huh.NewOptions("US", "CA", "MX")...).
    WithAccessible(true)  // ❌ Removed

confirm := huh.NewConfirm().
    Title("Continue?").
    WithAccessible(true)  // ❌ Gone from all field types

form := huh.NewForm(
    huh.NewGroup(input, select, confirm),
).WithAccessible(true)
```

**After:**
```go
// v2 - only the form controls accessible mode
input := huh.NewInput().
    Title("Name")

select := huh.NewSelect[string]().
    Title("Country").
    Options(huh.NewOptions("US", "CA", "MX")...)

confirm := huh.NewConfirm().
    Title("Continue?")

form := huh.NewForm(
    huh.NewGroup(input, select, confirm),
).WithAccessible(true)  // ✅ One setting for all fields
```

**Fields affected:**
- `Input.WithAccessible()` - removed
- `Text.WithAccessible()` - removed
- `Select.WithAccessible()` - removed
- `MultiSelect.WithAccessible()` - removed
- `Confirm.WithAccessible()` - removed
- `Note.WithAccessible()` - removed
- `FilePicker.WithAccessible()` - removed

The separate `github.com/charmbracelet/huh/accessibility` package is also gone. Just use `Form.WithAccessible()` directly.

### Bubble Tea v2 Integration

All methods that returned or accepted Bubble Tea types have been updated to v2:

**Field Methods:**
- `Blur() tea.Cmd` (now returns `charm.land/bubbletea/v2.Cmd`)
- `Focus() tea.Cmd` (now returns `charm.land/bubbletea/v2.Cmd`)
- `Init() tea.Cmd` (now returns `charm.land/bubbletea/v2.Cmd`)
- `Update(tea.Msg) (tea.Model, tea.Cmd)` (now uses v2 types)

**Form Methods:**
- `Init() tea.Cmd` (now returns `charm.land/bubbletea/v2.Cmd`)
- `Update(tea.Msg) (tea.Model, tea.Cmd)` (now uses v2 types)
- `WithProgramOptions(...tea.ProgramOption)` (now uses v2 types)

**Key Bindings:**
- `KeyBinds() []key.Binding` (now returns `charm.land/bubbles/v2/key.Binding`)

These changes are mostly mechanical. Your IDE should help you update these automatically.

### Lip Gloss v2 Types

All Lip Gloss types have been updated to v2. This affects style definitions in custom themes:

**Before:**
```go
import "github.com/charmbracelet/lipgloss"

style := lipgloss.NewStyle().
    Foreground(lipgloss.Color("205"))
```

**After:**
```go
import "charm.land/lipgloss/v2"

style := lipgloss.NewStyle().
    Foreground(lipgloss.Color("205"))
```

The API is largely the same, but the import path and internal types have changed.

### Position Type

Button alignment now uses Lip Gloss v2's `Position` type:

**Before:**
```go
import "github.com/charmbracelet/lipgloss"

field.WithButtonAlignment(lipgloss.Left)
```

**After:**
```go
import "charm.land/lipgloss/v2"

field.WithButtonAlignment(lipgloss.Left)
```

## New Features

### View Hooks

You can now modify the view before it's rendered:

```go
form.WithViewHook(func(v tea.View) tea.View {
    // Modify view properties like alt screen, mouse mode, etc.
    v.AltScreen = true
    return v
})
```

### Width Method

Select and MultiSelect fields now expose a `Width()` method for getting the field's width:

```go
width := multiSelect.Width()
```

### Model Type

The `Model` type is now exported, improving type safety when working with forms in Bubble Tea applications:

```go
var _ tea.Model = (*huh.Model)(nil)
```

## Migration Checklist

- [ ] Update `go.mod` dependencies to v2
- [ ] Update all import paths from `github.com/charmbracelet/` to `charm.land/` with `/v2` suffix
- [ ] Update theme calls to pass `isDark bool` parameter
- [ ] Remove field-level `WithAccessible()` calls (e.g., from `Input`, `Select`, etc.)
- [ ] Keep form-level `WithAccessible()` calls (those still work)
- [ ] Remove imports from `github.com/charmbracelet/huh/accessibility` package
- [ ] Update custom themes to `ThemeFunc` signature if applicable
- [ ] Run `go mod tidy`
- [ ] Run tests
- [ ] Update any documentation or examples

## Common Issues

### Import Cycles

If you encounter import cycle issues, make sure all Charm dependencies are on v2:

```bash
go list -m all | grep charmbracelet
go list -m all | grep charm.land
```

Ensure nothing is still referencing v1 versions.

### Type Mismatches

If you see type errors with `tea.Model`, `tea.Msg`, or `tea.Cmd`, double-check your Bubble Tea import:

```go
import tea "charm.land/bubbletea/v2"  // Make sure it's v2!
```

### Theme Signature Errors

If you get errors about theme functions, remember all built-in themes now require a `bool` parameter:

```go
// ✅ Correct
form.WithTheme(huh.ThemeCharm(true))

// ❌ Wrong
form.WithTheme(huh.ThemeCharm())
```

## Getting Help

If you run into issues:

- Check the [examples](./examples) directory for reference implementations
- Read the [Bubble Tea v2 Upgrade Guide](https://github.com/charmbracelet/bubbletea/blob/main/UPGRADE_GUIDE_V2.md)
- Ask in [Discord](https://charm.land/chat) or [Matrix](https://charm.land/matrix)
- Open an issue on [GitHub](https://github.com/charmbracelet/huh/issues)

---

Part of [Charm](https://charm.land).

<a href="https://charm.land/"><img alt="The Charm logo" src="https://stuff.charm.sh/charm-badge.jpg" width="400"></a>

Charm热爱开源 • Charm loves open source • نحنُ نحب المصادر المفتوحة
