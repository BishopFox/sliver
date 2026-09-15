package opfor

import (
	"context"
	"errors"
)

// AggressorDialogID identifies one dialog within its originating Runtime.
// Combine it with RuntimeID when a provider serves more than one Runtime.
type AggressorDialogID uint64

// AggressorDialogRowID identifies one row within an Aggressor dialog.
type AggressorDialogRowID uint64

// AggressorDialogButtonID identifies one button within an Aggressor dialog.
type AggressorDialogButtonID uint64

// AggressorDialogRowKind identifies the UI contract of one drow_* call.
type AggressorDialogRowKind string

const (
	AggressorDialogRowBeacon        AggressorDialogRowKind = "beacon"
	AggressorDialogRowCheckbox      AggressorDialogRowKind = "checkbox"
	AggressorDialogRowCombobox      AggressorDialogRowKind = "combobox"
	AggressorDialogRowExploits      AggressorDialogRowKind = "exploits"
	AggressorDialogRowFile          AggressorDialogRowKind = "file"
	AggressorDialogRowInterface     AggressorDialogRowKind = "interface"
	AggressorDialogRowKRBTGT        AggressorDialogRowKind = "krbtgt"
	AggressorDialogRowListener      AggressorDialogRowKind = "listener"
	AggressorDialogRowListenerStage AggressorDialogRowKind = "listener_stage"
	AggressorDialogRowMailServer    AggressorDialogRowKind = "mailserver"
	AggressorDialogRowProxyServer   AggressorDialogRowKind = "proxyserver"
	AggressorDialogRowSite          AggressorDialogRowKind = "site"
	AggressorDialogRowText          AggressorDialogRowKind = "text"
	AggressorDialogRowTextBig       AggressorDialogRowKind = "text_big"
)

// AggressorDialogButtonKind distinguishes callback-bearing action buttons
// from provider-owned Help buttons.
type AggressorDialogButtonKind string

const (
	AggressorDialogButtonAction AggressorDialogButtonKind = "action"
	AggressorDialogButtonHelp   AggressorDialogButtonKind = "help"
)

// AggressorDialogDefault is one top-level entry from the dictionary passed to
// dialog. Entries retain the source dictionary's observable iteration order.
type AggressorDialogDefault struct {
	Name  string
	Value Value
}

// AggressorDialogRow is one ordered row in a dialog presentation. Function is
// the exact drow_* spelling used by the script; Kind is its canonical UI kind.
// drow_listener_smb therefore has Kind AggressorDialogRowListenerStage while
// retaining Function "drow_listener_smb" and Deprecated true.
//
// Default and Options Values are transferred without scalar coercion. Options
// is a detached top-level slice, but a nested compound or object Value keeps
// its ordinary reference identity. Checkbox result encoding and every
// client-owned selector result remain provider responsibilities.
type AggressorDialogRow struct {
	ID       AggressorDialogRowID
	Kind     AggressorDialogRowKind
	Function string

	Name  string
	Label string

	Default    Value
	HasDefault bool

	CheckboxText string
	Options      []Value
	Width        int32
	HasWidth     bool

	Deprecated bool
	Equivalent AggressorDialogRowKind

	Script ScriptID
	Span   Span
}

// AggressorDialogButton is one ordered dialog button. Action buttons carry a
// Label and are submitted through AggressorDialogResponder.Activate. Help
// buttons carry a URL; opening it is a provider-owned UI effect and must not
// invoke the dialog's action callback.
type AggressorDialogButton struct {
	ID   AggressorDialogButtonID
	Kind AggressorDialogButtonKind

	Label string
	URL   string

	Script ScriptID
	Span   Span
}

// AggressorDialogPresentation is a detached top-level snapshot passed to a
// dialog provider. Slices are copied for the provider call. Values within
// Defaults, Rows, and Options retain normal OPFOR Value identity, so providers
// must snapshot or coerce capability-bearing compound/object Values before
// retaining them when that lifetime is undesirable.
//
// CreatorScript/CreationSpan identify dialog(...), while PresenterScript and
// PresentationSpan identify dialog_show(...). A valid ID is unique within its
// RuntimeID.
type AggressorDialogPresentation struct {
	ID        AggressorDialogID
	RuntimeID RuntimeID

	CreatorScript ScriptID
	CreationSpan  Span

	PresenterScript  ScriptID
	PresentationSpan Span

	Title    string
	Defaults []AggressorDialogDefault

	Description       string
	DescriptionLines  int32
	HasDescription    bool
	DescriptionScript ScriptID
	DescriptionSpan   Span

	Rows    []AggressorDialogRow
	Buttons []AggressorDialogButton
}

// AggressorDialogRowValue supplies one provider-produced value for a row.
// Activate rejects unknown or repeated RowIDs. Rows omitted from a response
// receive their captured default, or $null when no default was supplied.
type AggressorDialogRowValue struct {
	RowID AggressorDialogRowID
	Value Value
}

// AggressorDialogResponder is the one-shot capability paired with a dialog
// presentation. Activate accepts only an action button, closes the responder,
// and calls the owning script with the exact dialog object, stored button
// label, and a fresh row-name/value dictionary. Dismiss closes without calling
// script code. The first validated Activate or Dismiss consumes the one-shot;
// a callback error does not reopen it. Later operations after completion,
// dismissal, or provider failure match ErrAggressorUIClosed. Done closes when
// activation is admitted (before its callback returns), on dismissal, provider
// failure, script unload, or Runtime.Close.
//
// Activate may be called after PresentAggressorDialog returns. Callback
// admission remains tied to the creating script. If lifecycle revocation wins
// while the responder is open, a later call with a live caller context returns
// ErrScriptUnloaded; an earlier completion/dismissal remains
// ErrAggressorUIClosed. A caller context error is checked first. The returned
// Value is the callback result for importer observability; it is not used as
// the result of dialog_show. An
// activation begun while PresentAggressorDialog is still active is drained at
// that boundary and shares the presenting execution's instruction budget. A
// later retained activation is a detached UI event with a fresh top-level
// budget. Neither path exposes the presenter's raw evaluator/lifecycle state.
type AggressorDialogResponder interface {
	Activate(context.Context, AggressorDialogButtonID, ...AggressorDialogRowValue) (Value, error)
	Dismiss() error
	Done() <-chan struct{}
}

// AggressorDialogProvider presents runtime-owned dialogs. It may activate or
// dismiss synchronously, retain responder for asynchronous UI work, or return
// an error. A returned error is authoritative for dialog_show and never
// retries through Host; it cannot roll back a responder operation which the
// provider already completed, so providers should normally return nil after a
// successful Activate or Dismiss. Implementations may be called concurrently
// and should observe ctx. They may retain responder and presentation, but must
// not retain ctx after this method returns.
type AggressorDialogProvider interface {
	PresentAggressorDialog(context.Context, AggressorDialogPresentation, AggressorDialogResponder) error
}

// AggressorDialogProviderFunc adapts a function to AggressorDialogProvider.
type AggressorDialogProviderFunc func(context.Context, AggressorDialogPresentation, AggressorDialogResponder) error

// PresentAggressorDialog calls function.
func (function AggressorDialogProviderFunc) PresentAggressorDialog(
	ctx context.Context,
	presentation AggressorDialogPresentation,
	responder AggressorDialogResponder,
) error {
	if function == nil {
		return errors.New("opfor: Aggressor dialog provider is nil")
	}
	return function(ctx, presentation, responder)
}

// WithAggressorDialogProvider installs the typed importer boundary for the
// complete dialog, dbutton_*, and drow_* function family. WithFunction
// overrides retain precedence over native wrappers.
func WithAggressorDialogProvider(provider AggressorDialogProvider) Option {
	return func(config *runtimeConfig) error {
		if isNilInterface(provider) {
			return errors.New("opfor: Aggressor dialog provider is nil")
		}
		config.aggressorDialogProvider = provider
		return nil
	}
}

// AggressorPromptID identifies one prompt within its originating Runtime.
type AggressorPromptID uint64

// AggressorPromptKind identifies one documented prompt_* UI contract.
type AggressorPromptKind string

const (
	AggressorPromptConfirm       AggressorPromptKind = "confirm"
	AggressorPromptText          AggressorPromptKind = "text"
	AggressorPromptDirectoryOpen AggressorPromptKind = "directory_open"
	AggressorPromptFileOpen      AggressorPromptKind = "file_open"
	AggressorPromptFileSave      AggressorPromptKind = "file_save"
)

// AggressorPromptPresentation is one resolved prompt request. Confirm sets
// Text and Title; text sets Text and Default; directory/file-open set Title,
// Default, and Multiple; file-save sets Default. Multiple retains the exact
// third argument of directory/file-open prompts and AllowMultiple is its Sleep
// truth value. Default retains its Value identity. Fields absent for a Kind are
// $null/false/empty and their Has* flag is false.
type AggressorPromptPresentation struct {
	ID        AggressorPromptID
	Kind      AggressorPromptKind
	Name      string
	RuntimeID RuntimeID
	Script    ScriptID
	Span      Span

	Text  string
	Title string

	Default    Value
	HasDefault bool

	Multiple      Value
	HasMultiple   bool
	AllowMultiple bool
}

// AggressorPromptResponder is the one-shot capability paired with a prompt.
// OPFOR provisionally makes Confirm accept zero Values because the public
// reference does not specify its callback arity. Text, directory-open,
// file-open, and file-save accept exactly one Value, which becomes callback $1
// without coercion. A compatible multi-selection provider should encode its
// selection as the documented comma-separated scalar; OPFOR deliberately does
// not manufacture or validate that representation. Dismiss closes the prompt
// without invoking the callback. A validated Accept consumes the responder
// even when its callback returns an error. Done has the same terminal lifecycle
// contract as AggressorDialogResponder.Done. Accept uses the same synchronous
// shared-budget versus retained fresh-budget policy as Activate and checks its
// caller context before responder state.
type AggressorPromptResponder interface {
	Accept(context.Context, ...Value) (Value, error)
	Dismiss() error
	Done() <-chan struct{}
}

// AggressorPromptProvider presents prompt_* requests. It may answer
// synchronously or retain responder. A provider error is authoritative for the
// prompt_* call and does not fall back to Host, but cannot undo an Accept or
// Dismiss which already won; providers should normally return nil after a
// successful response. Implementations may be called concurrently and must
// not retain ctx after this method returns.
type AggressorPromptProvider interface {
	PresentAggressorPrompt(context.Context, AggressorPromptPresentation, AggressorPromptResponder) error
}

// AggressorPromptProviderFunc adapts a function to AggressorPromptProvider.
type AggressorPromptProviderFunc func(context.Context, AggressorPromptPresentation, AggressorPromptResponder) error

// PresentAggressorPrompt calls function.
func (function AggressorPromptProviderFunc) PresentAggressorPrompt(
	ctx context.Context,
	presentation AggressorPromptPresentation,
	responder AggressorPromptResponder,
) error {
	if function == nil {
		return errors.New("opfor: Aggressor prompt provider is nil")
	}
	return function(ctx, presentation, responder)
}

// WithAggressorPromptProvider installs the typed importer boundary for all
// five documented prompt_* functions. WithFunction overrides retain
// precedence over native wrappers.
func WithAggressorPromptProvider(provider AggressorPromptProvider) Option {
	return func(config *runtimeConfig) error {
		if isNilInterface(provider) {
			return errors.New("opfor: Aggressor prompt provider is nil")
		}
		config.aggressorPromptProvider = provider
		return nil
	}
}
