# Text
[![Go Reference](https://pkg.go.dev/badge/github.com/jedib0t/go-pretty/v6/text.svg)](https://pkg.go.dev/github.com/jedib0t/go-pretty/v6/text)

Package with utility functions to manipulate strings/text with full support for
ANSI escape sequences (colors, formatting, etc.).

Used heavily in the other packages in this repo ([list](../list),
[progress](../progress), and [table](../table)).

## Features

### Colors & Formatting

  - **ANSI Color Support** - Full support for terminal colors and formatting
    - Foreground colors (Black, Red, Green, Yellow, Blue, Magenta, Cyan, White)
    - Background colors (matching foreground set)
    - Hi-intensity variants for both foreground and background
    - Text attributes (Bold, Faint, Italic, Underline, Blink, Reverse, Concealed, CrossedOut)
    - Automatic color detection based on environment variables (`NO_COLOR`, `FORCE_COLOR`, `TERM`)
    - Global enable/disable functions for colors
    - Cached escape sequences for performance
  - **Text Formatting** - Transform text while preserving escape sequences
    - `FormatDefault` - No transformation
    - `FormatLower` - Convert to lowercase
    - `FormatTitle` - Convert to title case
    - `FormatUpper` - Convert to uppercase
  - **HTML Support** - Generate HTML class attributes for colors
  - **Color Combinations** - Combine multiple colors and attributes

### Alignment

  - **Horizontal Alignment**
    - `AlignDefault` / `AlignLeft` - Left-align text
    - `AlignCenter` - Center-align text
    - `AlignRight` - Right-align text
    - `AlignJustify` - Justify text (distribute spaces between words)
    - `AlignAuto` - Auto-detect: right-align numbers, left-align text
    - HTML and Markdown property generation for alignment
  - **Vertical Alignment**
    - `VAlignTop` - Align to top
    - `VAlignMiddle` - Align to middle
    - `VAlignBottom` - Align to bottom
    - Works with both string arrays and multi-line strings
    - HTML property generation for vertical alignment

### Text Wrapping

  - **WrapHard** - Hard wrap at specified length, breaks words if needed
    - Handles ANSI escape sequences without breaking formatting
    - Preserves paragraph breaks
  - **WrapSoft** - Soft wrap at specified length, tries to keep words intact
    - Handles ANSI escape sequences without breaking formatting
    - Preserves paragraph breaks
  - **WrapText** - Similar to WrapHard but also respects line breaks
    - Handles ANSI escape sequences without breaking formatting

### String Utilities

  - **Width Calculation**
    - `StringWidth` - Calculate display width of string (including escape sequences)
    - `StringWidthWithoutEscSequences` - Calculate display width ignoring escape sequences
    - `RuneWidth` - Calculate display width of a single rune (handles East Asian characters)
    - `LongestLineLen` - Find the longest line in a multi-line string
  - **String Manipulation**
    - `Trim` - Trim string to specified length while preserving escape sequences
    - `Pad` - Pad string to specified length with a character
    - `Snip` - Snip string to specified length with an indicator (e.g., "~")
    - `RepeatAndTrim` - Repeat string until it reaches specified length
    - `InsertEveryN` - Insert a character every N characters
    - `ProcessCRLF` - Process carriage returns and line feeds correctly
    - `Widen` - Convert half-width characters to full-width
  - **Escape Sequence Handling**
    - All functions properly handle ANSI escape sequences
    - Escape sequences are preserved during transformations
    - Width calculations ignore escape sequences

### Cursor Control

  - Move cursor in all directions
    - `CursorUp` - Move cursor up N lines
    - `CursorDown` - Move cursor down N lines
    - `CursorLeft` - Move cursor left N characters
    - `CursorRight` - Move cursor right N characters
    - `EraseLine` - Erase all characters to the right of cursor
  - Generate ANSI escape sequences for terminal cursor manipulation

### Hyperlinks

  - **Terminal Hyperlinks** - Create clickable hyperlinks in supported terminals
    - Uses OSC 8 escape sequences
    - Format: `Hyperlink(url, text)`
    - Falls back to plain text in unsupported terminals

### Transformers

  - **Number Transformer** - Format numbers with colors
    - Positive numbers colored green
    - Negative numbers colored red
    - Custom format string support (e.g., `%.2f`)
    - Supports all numeric types (int, uint, float)
  - **JSON Transformer** - Pretty-print JSON strings or objects
    - Customizable indentation (prefix and indent string)
    - Validates JSON before formatting
  - **Time Transformer** - Format time.Time objects
    - Custom layout support (e.g., `time.RFC3339`)
    - Timezone localization support
    - Auto-detects common time formats from strings
  - **Unix Time Transformer** - Format Unix timestamps
    - Handles seconds, milliseconds, microseconds, and nanoseconds
    - Auto-detects timestamp unit based on value
    - Timezone localization support
  - **URL Transformer** - Format URLs with styling
    - Underlined and colored blue by default
    - Custom color support

### Text Direction

  - **Bidirectional Text Support**
    - `LeftToRight` - Force left-to-right text direction
    - `RightToLeft` - Force right-to-left text direction
    - Uses Unicode directional markers

### Filtering

  - **String Filtering** - Filter string slices with custom functions
    - `Filter(slice, predicate)` - Returns filtered slice

### East Asian Character Support

  - Proper width calculation for East Asian characters (full-width, half-width)
  - Configurable East Asian width handling via `OverrideRuneWidthEastAsianWidth()`
  - Handles mixed character sets correctly