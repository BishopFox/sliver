package readline

import (
	"os"
	"regexp"
	"strings"
)

// Instance is used to encapsulate the parameter group and run time of any given
// readline instance so that you can reuse the readline API for multiple entry
// captures without having to repeatedly unload configuration.
type Instance struct {

	// Public Prompt
	Multiline       bool   // If set to true, the shell will have a two-line prompt.
	prompt          string // If Multiline is true, this is the first line of the prompt
	MultilinePrompt string // The second line of the prompt, where input follows.
	HideNextPrompt  bool   // When true, the next occurence of the shell will
	// not show the first line of the prompt (if multiline mode)

	// InputMode - The shell can be used in Vim editing mode, or Emacs (classic).
	InputMode InputMode

	// Vim parameters/functions
	// ShowVimMode - If set to true, a string '[i] >' or '[N] >' indicating the
	// current Vim mode will be appended to the prompt variable, therefore added to
	// the user's custom prompt is set. Applies for both single and multiline prompts
	ShowVimMode     bool
	VimModeColorize bool // If set to true, varies colors of the VimModePrompt

	// RefreshMultiline allows the user's program to refresh the input prompt.
	// In this version, the prompt is treated like the input line following it:
	// we can refresh it at any time, like we do with SyntaxHighlighter below.
	// RefreshMultiline func([]rune) string

	// PasswordMask is what character to hide password entry behind.
	// Once enabled, set to 0 (zero) to disable the mask again.
	PasswordMask rune

	// SyntaxHighlight is a helper function to provide syntax highlighting.
	// Once enabled, set to nil to disable again.
	SyntaxHighlighter func([]rune) string

	// History is an interface for querying the readline history.
	// This is exposed as an interface to allow you the flexibility to define how
	// you want your history managed (eg file on disk, database, cloud, or even
	// no history at all). By default it uses a dummy interface that only stores
	// historic items in memory.
	History History
	// AltHistory is an alternative history input, in case a console user would
	// like to have two different history flows.
	AltHistory History

	// HistoryAutoWrite defines whether items automatically get written to
	// history.
	// Enabled by default. Set to false to disable.
	HistoryAutoWrite bool // = true

	// TabCompleter is a simple function that offers completion suggestions.
	// It takes the readline line ([]rune) and cursor pos.
	// Returns a prefix string, and several completion groups with their items and description
	TabCompleter func([]rune, int) (string, []*CompletionGroup)

	// MaxTabCompletionRows is the maximum number of rows to display in the tab
	// completion grid.
	MaxTabCompleterRows int // = 4

	// SyntaxCompletion is used to autocomplete code syntax (like braces and
	// quotation marks). If you want to complete words or phrases then you might
	// be better off using the TabCompletion function.
	// SyntaxCompletion takes the line ([]rune) and cursor position, and returns
	// the new line and cursor position.
	SyntaxCompleter func([]rune, int) ([]rune, int)

	// HintText is a helper function which displays hint text the prompt.
	// HintText takes the line input from the promt and the cursor position.
	// It returns the hint text to display.
	HintText func([]rune, int) []rune

	// HintColor any ANSI escape codes you wish to use for hint formatting. By
	// default this will just be blue.
	HintFormatting string

	// TempDirectory is the path to write temporary files when editing a line in
	// $EDITOR. This will default to os.TempDir()
	TempDirectory string

	// GetMultiLine is a callback to your host program. Since multiline support
	// is handled by the application rather than readline itself, this callback
	// is required when calling $EDITOR. However if this function is not set
	// then readline will just use the current line.
	GetMultiLine func([]rune) []rune

	// readline operating parameters
	mlnPrompt      []rune // Our multiline prompt, different from multiline below
	mlnArrow       []rune
	promptLen      int    //= 4
	line           []rune // This is the input line, with entered text: full line = mlnPrompt + line
	pos            int
	multiline      []byte
	multisplit     []string
	skipStdinRead  bool
	stillOnRefresh bool // True if some logs have printed asynchronously since last loop.

	// history
	lineBuf    string
	histPos    int
	histNavIdx int // Used for quick history navigation.

	// hint text
	hintY    int //= 0
	hintText []rune

	// tab completion
	tcGroups             []*CompletionGroup // All of our suggestions tree is in here
	modeTabCompletion    bool
	tabCompletionSelect  bool // We may have completions, printed, but do we want to select a candidate ?
	tcPrefix             string
	tcOffset             int
	tcPosX               int
	tcPosY               int
	tcMaxX               int
	tcMaxY               int
	tcUsedY              int
	tcMaxLength          int
	tabCompletionReverse bool // Groups sometimes use this indicator to know how they should handle their index

	// When too many completions, we ask the user to confirm with another Tab keypress.
	compConfirmWait bool

	// Virtual completion
	currentComp  []rune // The currently selected item, not yet a real part of the input line.
	lineComp     []rune // Same as rl.line, but with the currentComp inserted.
	lineRemain   []rune // When we complete in the middle of a line, we cut and keep the remain.
	compAddSpace bool   // When this is true, any insertion of a candidate into the real line is done with an added space.

	// Tab Find
	modeTabFind  bool           // This does not change, because we will search in all options, no matter the group
	tfLine       []rune         // The current search pattern entered
	modeAutoFind bool           // for when invoked via ^R or ^F outside of [tab]
	searchMode   FindMode       // Used for varying hints, and underlying functions called
	regexSearch  *regexp.Regexp // Holds the current search regex match
	mainHist     bool           // Which history stdin do we want
	histHint     []rune         // We store a hist hint, for dual history sources

	// vim
	modeViMode       viMode //= vimInsert
	viIteration      string
	viUndoHistory    []undoItem
	viUndoSkipAppend bool
	viYankBuffer     string

	// event
	evtKeyPress map[string]func(string, []rune, int) *EventReturn
}

// NewInstance is used to create a readline instance and initialise it with sane defaults.
func NewInstance() *Instance {
	rl := new(Instance)

	// Prompt
	rl.Multiline = false
	rl.prompt = ">>> "
	rl.promptLen = len(rl.computePrompt())
	rl.mlnArrow = []rune{' ', '>', ' '}

	// Input Editing
	rl.InputMode = Emacs
	rl.ShowVimMode = true // In case the user sets input mode to Vim, everything is ready.

	// Completion
	rl.MaxTabCompleterRows = 100

	// History
	rl.History = new(ExampleHistory) // In-memory history by default.
	rl.HistoryAutoWrite = true

	// Others
	rl.HintFormatting = seqFgBlue
	rl.evtKeyPress = make(map[string]func(string, []rune, int) *EventReturn)
	rl.TempDirectory = os.TempDir()

	return rl
}

// WrapText - Wraps a text given a specified width, and returns the formatted
// string as well the number of lines it will occupy
func WrapText(text string, lineWidth int) (wrapped string, lines int) {
	words := strings.Fields(text)
	if len(words) == 0 {
		return
	}
	wrapped = words[0]
	spaceLeft := lineWidth - len(wrapped)
	// There must be at least a line
	if text != "" {
		lines++
	}
	for _, word := range words[1:] {
		if len(word)+1 > spaceLeft {
			lines++
			wrapped += "\n" + word
			spaceLeft = lineWidth - len(word)
		} else {
			wrapped += " " + word
			spaceLeft -= 1 + len(word)
		}
	}
	return
}
