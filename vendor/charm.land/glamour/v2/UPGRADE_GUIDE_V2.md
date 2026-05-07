# Glamour v2 Upgrade Guide

This guide will help you migrate from Glamour v1 to v2. Most upgrades are straightforward and can be completed in minutes.

## Update Import Paths

**Required:** All imports must use the new `charm.land` module path with `/v2`:

```diff
-import "github.com/charmbracelet/glamour"
-import "github.com/charmbracelet/glamour/ansi"
-import "github.com/charmbracelet/glamour/styles"
+import "charm.land/glamour/v2"
+import "charm.land/glamour/v2/ansi"
+import "charm.land/glamour/v2/styles"
```

## Update Dependencies

```bash
go get charm.land/glamour/v2@latest
```

If you need color downsampling (most apps do):

```bash
go get charm.land/lipgloss/v2@latest
```

## Remove Auto Style Detection

**Breaking:** `WithAutoStyle()` has been removed. The default style is now `"dark"`.

```diff
-r, _ := glamour.NewTermRenderer(
-    glamour.WithAutoStyle(),
-)
+r, _ := glamour.NewTermRenderer(
+    // "dark" is the default, or specify explicitly
+    glamour.WithStylePath("dark"),
+)
```

If you were relying on automatic style selection based on terminal background, you'll need to choose the style explicitly:

```go
// For light backgrounds
r, _ := glamour.NewTermRenderer(glamour.WithStylePath("light"))

// For dark backgrounds (default)
r, _ := glamour.NewTermRenderer(glamour.WithStylePath("dark"))

// Other built-in styles: "pink", "dracula", "tokyo-night", "ascii"
r, _ := glamour.NewTermRenderer(glamour.WithStylePath("dracula"))
```

Want to detect the terminal background yourself? You can use Lip Gloss:

```go
import "charm.land/lipgloss/v2"

// Detect if we're on a dark background
isDark := lipgloss.HasDarkBackground()

style := "dark"
if !isDark {
    style = "light"
}

r, _ := glamour.NewTermRenderer(glamour.WithStylePath(style))
```

## Handle Color Downsampling Explicitly

**Breaking:** `WithColorProfile()` has been removed. Use Lip Gloss for color adaptation:

```diff
-import "github.com/muesli/termenv"
-
-r, _ := glamour.NewTermRenderer(
-    glamour.WithColorProfile(termenv.TrueColor),
-)
-out, _ := r.Render(markdown)
-fmt.Print(out)
+import "charm.land/lipgloss/v2"
+
+r, _ := glamour.NewTermRenderer(
+    glamour.WithWordWrap(80),
+)
+out, _ := r.Render(markdown)
+
+// Lip Gloss handles color downsampling based on terminal capabilities
+lipgloss.Print(out)
```

Why the change? Glamour is now pure — it always produces the same output for the same input. This makes it more predictable and testable. Lip Gloss handles the terminal-specific color adaptation when you're ready to display the output.

If you don't need color adaptation (e.g., you know you're always outputting TrueColor):

```go
r, _ := glamour.NewTermRenderer(glamour.WithWordWrap(80))
out, _ := r.Render(markdown)
fmt.Print(out)  // Direct output, no downsampling
```

## Remove Overline Styles

**Breaking:** The `Overlined` field has been removed from style configurations.

If you have custom styles using `Overlined`:

```diff
 StylePrimitive: ansi.StylePrimitive{
     Bold:      &trueBool,
     Underline: &trueBool,
-    Overlined: &trueBool,
 }
```

Overline was rarely supported across terminals and not widely used. If you need similar visual separation, consider alternatives like underline, bold, inverse, or background colors.

## Update Custom Style Definitions

If you maintain custom `StyleConfig` definitions, update the import paths:

```diff
-import "github.com/charmbracelet/glamour/ansi"
+import "charm.land/glamour/v2/ansi"

 var myStyle = &ansi.StyleConfig{
     // Your custom style definition
     Document: ansi.StyleBlock{
         StylePrimitive: ansi.StylePrimitive{
             Color: stringPtr("#E6DB74"),
         },
     },
 }
```

The structure is the same; only the import path changes.

## Verify Custom Writers (Advanced)

If you implemented custom margin or padding writers using `ansi.MarginWriter`:

1. Ensure you call `.Close()` on all writer instances
2. The new `IndentWriter` and `PaddingWriter` types are available for custom use

```go
import "charm.land/glamour/v2/ansi"

mw := ansi.NewMarginWriter(ctx, w, style)
defer mw.Close()  // Important: always close writers now

// Write your content
io.WriteString(mw, content)
```

This improves memory management and prevents resource leaks.

## Example Migration

Here's a complete before/after example:

### Before (v1)

```go
package main

import (
    "fmt"
    "github.com/charmbracelet/glamour"
    "github.com/muesli/termenv"
)

func main() {
    md := `# Hello World

This is **Glamour v1**!
`
    
    r, _ := glamour.NewTermRenderer(
        glamour.WithAutoStyle(),
        glamour.WithColorProfile(termenv.TrueColor),
        glamour.WithWordWrap(80),
    )
    
    out, _ := r.Render(md)
    fmt.Print(out)
}
```

### After (v2)

```go
package main

import (
    "fmt"
    "charm.land/glamour/v2"
    "charm.land/lipgloss/v2"
)

func main() {
    md := `# Hello World

This is **Glamour v2**!
`
    
    r, _ := glamour.NewTermRenderer(
        glamour.WithStylePath("dark"),  // or omit for default
        glamour.WithWordWrap(80),
    )
    
    out, _ := r.Render(md)
    lipgloss.Print(out)  // Handles color downsampling
}
```

## Testing Your Migration

After making changes:

1. **Run your tests** — Ensure all tests pass with the new version
2. **Visual check** — Render sample markdown and verify output looks correct
3. **Check wrapping** — Pay special attention to text wrapping if you have CJK or emoji content (it should be better now!)
4. **Test hyperlinks** — If you use autolinks, verify they render correctly

## Common Issues

### "cannot find package"

Make sure you've updated your `go.mod`:

```bash
go get charm.land/glamour/v2
```

And that all imports use the new path with `/v2`.

### Colors look wrong

If colors aren't displaying correctly, make sure you're using `lipgloss.Print()` instead of `fmt.Print()`:

```go
out, _ := r.Render(markdown)
lipgloss.Print(out)  // Not fmt.Print(out)
```

### Text wrapping issues

Glamour v2 has improved text wrapping, especially for CJK characters and emojis. If you're seeing wrapping issues, it's likely a regression. Please [open an issue](https://github.com/charmbracelet/glamour/issues)!

## Need Help?

If you run into issues during migration:

- Check the [examples directory](https://github.com/charmbracelet/glamour/tree/main/examples) for working code
- Review [What's New](WHATS_NEW.md) for detailed feature changes
- Join us on [Discord](https://charm.sh/chat)
- Open an issue on [GitHub](https://github.com/charmbracelet/glamour/issues)

## Feedback

Migrated successfully? Having trouble? We'd love to hear about it!

- [Discord](https://charm.sh/chat)
- [The Fediverse](https://mastodon.social/@charmcli)
- [Twitter](https://twitter.com/charmcli)

---

Welcome to Glamour v2! 💄

Part of [Charm](https://charm.sh).

<a href="https://charm.sh/"><img alt="The Charm logo" src="https://stuff.charm.sh/charm-badge.jpg" width="400"></a>

Charm热爱开源 • Charm loves open source
