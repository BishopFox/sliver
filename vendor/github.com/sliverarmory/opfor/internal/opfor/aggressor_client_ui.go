package opfor

import (
	"context"
	"errors"
)

// ErrAggressorPopupStale reports that a retained popup composer no longer
// names the complete binding generation captured by show_popup. popup_clear,
// binding-owner unload, or creator unload makes a composer stale; it never
// retargets to newer registrations with the same name.
var ErrAggressorPopupStale = errors.New("opfor: Aggressor popup composition is stale")

// AggressorClientUIOperation identifies one client-owned UI operation. String
// values are the exact Aggressor function names so they remain stable in
// importer logs and adapters.
type AggressorClientUIOperation string

const (
	AggressorClientUIAddTab            AggressorClientUIOperation = "addTab"
	AggressorClientUIAddVisualization  AggressorClientUIOperation = "addVisualization"
	AggressorClientUIShowVisualization AggressorClientUIOperation = "showVisualization"
	AggressorClientUIShowPopup         AggressorClientUIOperation = "show_popup"
	AggressorClientUIMenubar           AggressorClientUIOperation = "menubar"
	AggressorClientUIPopupClear        AggressorClientUIOperation = "popup_clear"
	AggressorClientUISeparator         AggressorClientUIOperation = "separator"
	AggressorClientUIRemoveTab         AggressorClientUIOperation = "removeTab"
	AggressorClientUINextTab           AggressorClientUIOperation = "nextTab"
	AggressorClientUIPreviousTab       AggressorClientUIOperation = "previousTab"
	AggressorClientUIAddToClipboard    AggressorClientUIOperation = "add_to_clipboard"
	AggressorClientUIOpenURL           AggressorClientUIOperation = "url_open"
	AggressorClientUIShowError         AggressorClientUIOperation = "show_error"
	AggressorClientUIShowMessage       AggressorClientUIOperation = "show_message"

	// The open* operations are client-owned window, dialog, tab, or browser
	// commands. Their values intentionally retain the exact documented
	// Aggressor function spelling so one provider can route them without a
	// second name translation table.
	AggressorClientUIOpenAboutDialog                     AggressorClientUIOperation = "openAboutDialog"
	AggressorClientUIOpenApplicationManager              AggressorClientUIOperation = "openApplicationManager"
	AggressorClientUIOpenAutoRunDialog                   AggressorClientUIOperation = "openAutoRunDialog"
	AggressorClientUIOpenBeaconBrowser                   AggressorClientUIOperation = "openBeaconBrowser"
	AggressorClientUIOpenBeaconConsole                   AggressorClientUIOperation = "openBeaconConsole"
	AggressorClientUIOpenBrowserPivotSetup               AggressorClientUIOperation = "openBrowserPivotSetup"
	AggressorClientUIOpenCloneSiteDialog                 AggressorClientUIOperation = "openCloneSiteDialog"
	AggressorClientUIOpenConnectDialog                   AggressorClientUIOperation = "openConnectDialog"
	AggressorClientUIOpenCovertVPNSetup                  AggressorClientUIOperation = "openCovertVPNSetup"
	AggressorClientUIOpenCredentialManager               AggressorClientUIOperation = "openCredentialManager"
	AggressorClientUIOpenDefaultShortcutsDialog          AggressorClientUIOperation = "openDefaultShortcutsDialog"
	AggressorClientUIOpenDownloadBrowser                 AggressorClientUIOperation = "openDownloadBrowser"
	AggressorClientUIOpenElevateDialog                   AggressorClientUIOperation = "openElevateDialog"
	AggressorClientUIOpenEventLog                        AggressorClientUIOperation = "openEventLog"
	AggressorClientUIOpenFileBrowser                     AggressorClientUIOperation = "openFileBrowser"
	AggressorClientUIOpenGoldenTicketDialog              AggressorClientUIOperation = "openGoldenTicketDialog"
	AggressorClientUIOpenHTMLApplicationDialog           AggressorClientUIOperation = "openHTMLApplicationDialog"
	AggressorClientUIOpenHostFileDialog                  AggressorClientUIOperation = "openHostFileDialog"
	AggressorClientUIOpenInterfaceManager                AggressorClientUIOperation = "openInterfaceManager"
	AggressorClientUIOpenJavaSignedAppletDialog          AggressorClientUIOperation = "openJavaSignedAppletDialog"
	AggressorClientUIOpenJavaSmartAppletDialog           AggressorClientUIOperation = "openJavaSmartAppletDialog"
	AggressorClientUIOpenJobBrowser                      AggressorClientUIOperation = "openJobBrowser"
	AggressorClientUIOpenJobConsole                      AggressorClientUIOperation = "openJobConsole"
	AggressorClientUIOpenJumpDialog                      AggressorClientUIOperation = "openJumpDialog"
	AggressorClientUIOpenKeystrokeBrowser                AggressorClientUIOperation = "openKeystrokeBrowser"
	AggressorClientUIOpenListenerManager                 AggressorClientUIOperation = "openListenerManager"
	AggressorClientUIOpenMakeTokenDialog                 AggressorClientUIOperation = "openMakeTokenDialog"
	AggressorClientUIOpenMalleableProfileDialog          AggressorClientUIOperation = "openMalleableProfileDialog"
	AggressorClientUIOpenNewCredentialDialog             AggressorClientUIOperation = "openNewCredentialDialog"
	AggressorClientUIOpenOfficeMacroDialog               AggressorClientUIOperation = "openOfficeMacroDialog"
	AggressorClientUIOpenOneLinerDialog                  AggressorClientUIOperation = "openOneLinerDialog"
	AggressorClientUIOpenOrActivate                      AggressorClientUIOperation = "openOrActivate"
	AggressorClientUIOpenPayloadGeneratorDialog          AggressorClientUIOperation = "openPayloadGeneratorDialog"
	AggressorClientUIOpenPayloadGeneratorStageDialog     AggressorClientUIOperation = "openPayloadGeneratorStageDialog"
	AggressorClientUIOpenPayloadHelper                   AggressorClientUIOperation = "openPayloadHelper"
	AggressorClientUIOpenPayloadStoreManager             AggressorClientUIOperation = "openPayloadStoreManager"
	AggressorClientUIOpenPivotListenerSetup              AggressorClientUIOperation = "openPivotListenerSetup"
	AggressorClientUIOpenPortScanner                     AggressorClientUIOperation = "openPortScanner"
	AggressorClientUIOpenPortScannerLocal                AggressorClientUIOperation = "openPortScannerLocal"
	AggressorClientUIOpenPowerShellWebDialog             AggressorClientUIOperation = "openPowerShellWebDialog"
	AggressorClientUIOpenPreferencesDialog               AggressorClientUIOperation = "openPreferencesDialog"
	AggressorClientUIOpenProcessBrowser                  AggressorClientUIOperation = "openProcessBrowser"
	AggressorClientUIOpenSOCKSBrowser                    AggressorClientUIOperation = "openSOCKSBrowser"
	AggressorClientUIOpenSOCKSSetup                      AggressorClientUIOperation = "openSOCKSSetup"
	AggressorClientUIOpenScreenshotBrowser               AggressorClientUIOperation = "openScreenshotBrowser"
	AggressorClientUIOpenScriptConsole                   AggressorClientUIOperation = "openScriptConsole"
	AggressorClientUIOpenScriptManager                   AggressorClientUIOperation = "openScriptManager"
	AggressorClientUIOpenScriptedWebDialog               AggressorClientUIOperation = "openScriptedWebDialog"
	AggressorClientUIOpenServiceBrowser                  AggressorClientUIOperation = "openServiceBrowser"
	AggressorClientUIOpenSiteManager                     AggressorClientUIOperation = "openSiteManager"
	AggressorClientUIOpenSpawnAsDialog                   AggressorClientUIOperation = "openSpawnAsDialog"
	AggressorClientUIOpenSpawnDialog                     AggressorClientUIOperation = "openSpawnDialog"
	AggressorClientUIOpenSpearPhishDialog                AggressorClientUIOperation = "openSpearPhishDialog"
	AggressorClientUIOpenSystemInformationDialog         AggressorClientUIOperation = "openSystemInformationDialog"
	AggressorClientUIOpenSystemProfilerDialog            AggressorClientUIOperation = "openSystemProfilerDialog"
	AggressorClientUIOpenTargetBrowser                   AggressorClientUIOperation = "openTargetBrowser"
	AggressorClientUIOpenUserDefinedBrowser              AggressorClientUIOperation = "openUserDefinedBrowser"
	AggressorClientUIOpenWebLog                          AggressorClientUIOperation = "openWebLog"
	AggressorClientUIOpenWindowsExecutableDialog         AggressorClientUIOperation = "openWindowsExecutableDialog"
	AggressorClientUIOpenWindowsExecutableStageAllDialog AggressorClientUIOperation = "openWindowsExecutableStageAllDialog"
	AggressorClientUIOpenWindowsExecutableStageDialog    AggressorClientUIOperation = "openWindowsExecutableStageDialog"

	// These operations generate or insert documented client-side browser and
	// menu components. OPFOR transports opaque Values; the importer owns every
	// concrete GUI object and menu-tree mutation.
	AggressorClientUIGenerateBeaconBrowser  AggressorClientUIOperation = "bbrowser"
	AggressorClientUIColorMenu              AggressorClientUIOperation = "colorMenu"
	AggressorClientUIFileBrowser            AggressorClientUIOperation = "file_browser"
	AggressorClientUIInsertColorMenu        AggressorClientUIOperation = "insert_color_menu"
	AggressorClientUIInsertComponent        AggressorClientUIOperation = "insert_component"
	AggressorClientUIGeneratePivotGraph     AggressorClientUIOperation = "pgraph"
	AggressorClientUIProcessBrowser         AggressorClientUIOperation = "process_browser"
	AggressorClientUIGenerateSessionBrowser AggressorClientUIOperation = "sbrowser"
	AggressorClientUIGenerateTargetBrowser  AggressorClientUIOperation = "tbrowser"
)

// AggressorPopupComposer is the restricted script capability attached to a
// show_popup or menubar request. Compose invokes, in registration order, the
// exact set of popup registrations selected when the request ran. A show_popup
// composer supplies that request's component as every popup binding's $1; a
// menubar composer invokes each popup without positional arguments. It exposes
// neither the Runtime nor arbitrary binding lookup, arguments, or retargeting.
//
// A composer may be retained and invoked after HandleAggressorClientUI
// returns. It remains jointly owned by the show_popup caller and every selected
// popup binding. Compose returns ErrAggressorPopupStale without intentionally
// invoking a partial set when that captured generation has been cleared or
// unloaded. Compose may synchronously reenter HandleAggressorClientUI for a
// separator declared by a selected popup, so providers must not hold a
// non-reentrant lock while calling it. The supplied context controls the
// callback invocation and must not be retained after Compose returns.
type AggressorPopupComposer interface {
	Compose(context.Context) error
}

// AggressorClientUIRequest is one resolved client UI request. Name is the
// exact normalized function spelling used by the script. RuntimeID is the
// nonzero process-local identity of the originating Runtime; Script and Span
// identify the call site without exposing a *Runtime.
// Bindings is the opaque callback capability for that exact Runtime; retaining
// it retains the Runtime but does not expose its evaluator or binding registry.
//
// Arguments is a detached top-level positional snapshot. Every argument is
// resolved exactly once before the provider call. Scalar Values are immutable
// while compound, object, and binary Values retain their ordinary identity and
// provenance. The sole openPayloadHelper function argument is not duplicated
// in Arguments: Callback contains its retained, lifecycle-bound capability.
// Providers that retain a request also retain any capabilities reachable
// through its Values and should snapshot or detach them when that lifetime is
// undesirable.
//
// Popup is non-nil for show_popup or menubar when a matching popup registration
// was active at request time. For show_popup, the original event, popup name,
// and component remain available in Arguments at positions zero through two.
// For menubar, the description and popup name remain at positions zero and one.
// Composition is a detached snapshot of the active popup/menu composition for
// separator, insert_color_menu, and insert_component; it lets an importer
// place the supplied component in the correct menu tree without embedding
// interpreter state in that snapshot. It is nil when one of those functions is
// invoked outside a registered popup/menu callback.
//
// Callback is non-nil only for openPayloadHelper. It is a retained,
// script-owned, multi-shot capability whose Invoke arguments become the
// chooser callback's positional arguments. In particular, a provider passes
// the selected listener as its first argument. Invocation honors the supplied
// context and is rejected after the creating Script generation retires, its
// Script unloads, or Runtime closes.
type AggressorClientUIRequest struct {
	Operation AggressorClientUIOperation
	Name      string

	RuntimeID RuntimeID
	Script    ScriptID
	Span      Span
	Bindings  AggressorBindings

	Arguments []Value
	Callback  Callable
	Popup     AggressorPopupComposer

	Composition *BindingInvocation
}

// AggressorClientUIProvider supplies embedding-client UI effects. It is called
// synchronously exactly once for every valid invocation when configured.
// OPFOR transfers a successful returned Value directly to the script without
// coercion, validation, or cloning for value-producing operations. This is
// also the open* result policy: an adapter may return its client object where
// Cobalt exposes one (for example a dockable log tab), and returns Null for an
// effect-only dialog. The documented effect-only popup_clear, file_browser,
// process_browser, insert_color_menu, and insert_component operations always
// return $null on the typed route; any provider result is discarded.
// openPayloadHelper likewise returns $null after provider success because its
// documented output is delivered through Callback, not through a synchronous
// return value. Provider errors are authoritative and are never
// retried through Host because a UI effect may already have occurred.
//
// Implementations may be called concurrently and should observe ctx. The
// provider may retain request Values, Callback, and Popup subject to their
// documented capability lifetimes, but must not retain ctx after this method
// returns.
type AggressorClientUIProvider interface {
	HandleAggressorClientUI(context.Context, AggressorClientUIRequest) (Value, error)
}

// AggressorClientUIProviderFunc adapts a function to
// AggressorClientUIProvider.
type AggressorClientUIProviderFunc func(context.Context, AggressorClientUIRequest) (Value, error)

// HandleAggressorClientUI calls function.
func (function AggressorClientUIProviderFunc) HandleAggressorClientUI(
	ctx context.Context,
	request AggressorClientUIRequest,
) (Value, error) {
	if function == nil {
		return Null(), errors.New("opfor: Aggressor client UI provider is nil")
	}
	return function(ctx, request)
}

// WithAggressorClientUIProvider installs the typed importer boundary for
// client-owned tabs, visualizations, popups, menubar registration, navigation,
// clipboard, URLs, message presentation, documented open* windows, and
// client-side browser/menu component helpers. OPFOR never constructs Swing
// widgets. WithFunction overrides retain precedence over the native wrappers.
func WithAggressorClientUIProvider(provider AggressorClientUIProvider) Option {
	return func(config *runtimeConfig) error {
		if isNilInterface(provider) {
			return errors.New("opfor: Aggressor client UI provider is nil")
		}
		config.aggressorClientUIProvider = provider
		return nil
	}
}
