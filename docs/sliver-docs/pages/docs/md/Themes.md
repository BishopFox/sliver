### Console Themes

Sliver’s interactive consoles (both `sliver-client` and the server-only console) use a theme file to drive all lipgloss styling.

- Theme path: `~/.sliver-client/theme.yaml`
- Settings path: `~/.sliver-client/tui-settings.yaml`

If `theme.yaml` is missing, Sliver will create it with the default palette.

#### Palette Structure

`theme.yaml` defines six palettes:

- `primary`
- `secondary`
- `default`
- `success`
- `warning`
- `danger`

Each palette has a base `default` color and a set of modifier colors (`50..900`):

```yaml
primary:
  default: "#006FEE"
  mods:
    50:  "#001731"
    100: "#002e62"
    200: "#004493"
    300: "#005bc4"
    400: "#006FEE"
    500: "#338ef7"
    600: "#66aaf9"
    700: "#99c7fb"
    800: "#cce3fd"
    900: "#e6f1fe"
```

### Prompt Settings

The console prompt is configurable via `tui-settings.yaml`:

```yaml
prompt: "host"
prompt_template: |
  {{- if .IsServer -}}
  {{ .Styles.Bold.Render "[server]" }} {{ .Styles.Underline.Render "sliver" }}{{ .Target.Suffix }} >
  {{- else -}}
  {{- if .Host -}}{{ .Styles.BoldPrimary.Render (printf "[%s]" .Host) }} {{- end -}}
  {{ .Styles.Underline.Render "sliver" }}{{ .Target.Suffix }} >
  {{- end -}}
```

#### `prompt` Modes

- `operator-host`
  - `sliver-client`: `[operator@host] sliver >`
  - server console: `[server] sliver >`
- `host`
  - `sliver-client`: `[host] sliver >`
  - server console: `[server] sliver >`
- `basic`
  - both: `sliver >`
- `custom`
  - the entire prompt text is rendered from `prompt_template` (Go `text/template`)
  - if rendering fails, Sliver falls back to a minimal prompt: ` > `

### Custom Prompt Templates

When `prompt: custom`, Sliver evaluates `prompt_template` as a Go `text/template` every time the prompt is displayed.

#### Template Variables

Top-level values:

- `.IsServer` (bool)
- `.Connected` (bool)
- `.Operator` (string)
- `.Host` (string)
- `.Port` (int)
- `.HostPort` (string, `host:port` when available)
- `.Now` (time)

Active target:

- `.Target.SessionName` (string)
- `.Target.BeaconName` (string)
- `.Target.Suffix` (string)
  - pre-rendered themed suffix matching the default prompt, like ` (my-session)` when a session or beacon is selected

Theme colors:

- `.Colors.Primary.Default`
- `.Colors.Primary.Mods` (map; use `index`, e.g. `index .Colors.Primary.Mods 500`)
- same structure for `.Colors.Secondary`, `.Colors.Default`, `.Colors.Success`, `.Colors.Warning`, `.Colors.Danger`

Theme styles (lipgloss styles already wired to `theme.yaml`):

- `.Styles.Bold`, `.Styles.Underline`
- `.Styles.Primary`, `.Styles.Secondary`, `.Styles.Default`, `.Styles.Success`, `.Styles.Warning`, `.Styles.Danger`
- `.Styles.BoldPrimary`, `.Styles.BoldSecondary`, `.Styles.BoldDefault`, `.Styles.BoldSuccess`, `.Styles.BoldWarning`, `.Styles.BoldDanger`
- legacy names: `.Styles.Red`, `.Styles.Green`, `.Styles.Orange`, `.Styles.Blue`, `.Styles.Purple`, `.Styles.Cyan`, `.Styles.Gray`, `.Styles.Black`

#### Examples

Show user and host (client), show server tag (server console):

```gotemplate
{{- if .IsServer -}}
{{ .Styles.Bold.Render "[server]" }} {{ .Styles.Underline.Render "sliver" }}{{ .Target.Suffix }} >
{{- else -}}
{{- if and .Operator .Host -}}
{{ .Styles.BoldPrimary.Render (printf "[%s@%s]" .Operator .Host) }} 
{{- else if .Host -}}
{{ .Styles.BoldPrimary.Render (printf "[%s]" .Host) }} 
{{- end -}}
{{ .Styles.Underline.Render "sliver" }}{{ .Target.Suffix }} >
{{- end -}}
```

Basic prompt everywhere:

```gotemplate
{{ .Styles.Underline.Render "sliver" }}{{ .Target.Suffix }} >
```

Use a lighter primary modifier for the host tag:

```gotemplate
{{- $c := index .Colors.Primary.Mods 500 -}}
{{- if .Host -}}{{ (.Styles.Bold.Foreground $c).Render (printf "[%s]" .Host) }} {{- end -}}
{{ .Styles.Underline.Render "sliver" }}{{ .Target.Suffix }} >
```
