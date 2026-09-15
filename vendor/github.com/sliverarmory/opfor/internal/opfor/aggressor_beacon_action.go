package opfor

import (
	"context"
	"errors"
)

// AggressorBeaconActionKind identifies one canonical Beacon tasking action.
// The string values are the exact Aggressor function names so they are stable
// across adapters and remain useful in logs without a second name mapping.
type AggressorBeaconActionKind string

const (
	// AggressorBeaconActionArgumentSpoofAdd identifies bargue_add.
	AggressorBeaconActionArgumentSpoofAdd AggressorBeaconActionKind = "bargue_add"
	// AggressorBeaconActionArgumentSpoofList identifies bargue_list.
	AggressorBeaconActionArgumentSpoofList AggressorBeaconActionKind = "bargue_list"
	// AggressorBeaconActionArgumentSpoofRemove identifies bargue_remove.
	AggressorBeaconActionArgumentSpoofRemove AggressorBeaconActionKind = "bargue_remove"
	// AggressorBeaconActionConfigure identifies bbeacon_config.
	AggressorBeaconActionConfigure AggressorBeaconActionKind = "bbeacon_config"
	// AggressorBeaconActionGate identifies bbeacon_gate.
	AggressorBeaconActionGate AggressorBeaconActionKind = "bbeacon_gate"
	// AggressorBeaconActionInterpreter identifies bbeacon_interpreter.
	AggressorBeaconActionInterpreter AggressorBeaconActionKind = "bbeacon_interpreter"
	// AggressorBeaconActionInterpreterLint identifies bbeacon_interpreter_lint.
	AggressorBeaconActionInterpreterLint AggressorBeaconActionKind = "bbeacon_interpreter_lint"
	// AggressorBeaconActionBlockDLLs identifies bblockdlls.
	AggressorBeaconActionBlockDLLs AggressorBeaconActionKind = "bblockdlls"
	// AggressorBeaconActionBrowserPivot identifies bbrowserpivot.
	AggressorBeaconActionBrowserPivot AggressorBeaconActionKind = "bbrowserpivot"
	// AggressorBeaconActionBrowserPivotStop identifies bbrowserpivot_stop.
	AggressorBeaconActionBrowserPivotStop AggressorBeaconActionKind = "bbrowserpivot_stop"
	// AggressorBeaconActionCancelDownload identifies bcancel.
	AggressorBeaconActionCancelDownload AggressorBeaconActionKind = "bcancel"
	// AggressorBeaconActionChangeDirectory identifies bcd.
	AggressorBeaconActionChangeDirectory AggressorBeaconActionKind = "bcd"
	// AggressorBeaconActionCheckin identifies bcheckin.
	AggressorBeaconActionCheckin AggressorBeaconActionKind = "bcheckin"
	// AggressorBeaconActionClearTasks identifies bclear.
	AggressorBeaconActionClearTasks AggressorBeaconActionKind = "bclear"
	// AggressorBeaconActionClipboard identifies bclipboard.
	AggressorBeaconActionClipboard AggressorBeaconActionKind = "bclipboard"
	// AggressorBeaconActionConnect identifies bconnect.
	AggressorBeaconActionConnect AggressorBeaconActionKind = "bconnect"
	// AggressorBeaconActionCovertVPN identifies bcovertvpn.
	AggressorBeaconActionCovertVPN AggressorBeaconActionKind = "bcovertvpn"
	// AggressorBeaconActionDownload identifies bdownload.
	AggressorBeaconActionDownload AggressorBeaconActionKind = "bdownload"
	// AggressorBeaconActionDataStoreList identifies bdata_store_list.
	AggressorBeaconActionDataStoreList AggressorBeaconActionKind = "bdata_store_list"
	// AggressorBeaconActionDataStoreLoad identifies bdata_store_load.
	AggressorBeaconActionDataStoreLoad AggressorBeaconActionKind = "bdata_store_load"
	// AggressorBeaconActionDataStoreUnload identifies bdata_store_unload.
	AggressorBeaconActionDataStoreUnload AggressorBeaconActionKind = "bdata_store_unload"
	// AggressorBeaconActionDCSync identifies bdcsync.
	AggressorBeaconActionDCSync AggressorBeaconActionKind = "bdcsync"
	// AggressorBeaconActionDesktop identifies bdesktop.
	AggressorBeaconActionDesktop AggressorBeaconActionKind = "bdesktop"
	// AggressorBeaconActionDLLLoad identifies bdllload.
	AggressorBeaconActionDLLLoad AggressorBeaconActionKind = "bdllload"
	// AggressorBeaconActionExecute identifies bexecute.
	AggressorBeaconActionExecute AggressorBeaconActionKind = "bexecute"
	// AggressorBeaconActionExit identifies bexit.
	AggressorBeaconActionExit AggressorBeaconActionKind = "bexit"
	// AggressorBeaconActionEnablePrivileges identifies bgetprivs.
	AggressorBeaconActionEnablePrivileges AggressorBeaconActionKind = "bgetprivs"
	// AggressorBeaconActionGetSystem identifies bgetsystem.
	AggressorBeaconActionGetSystem AggressorBeaconActionKind = "bgetsystem"
	// AggressorBeaconActionGetUID identifies bgetuid.
	AggressorBeaconActionGetUID AggressorBeaconActionKind = "bgetuid"
	// AggressorBeaconActionListFiles identifies bls.
	AggressorBeaconActionListFiles AggressorBeaconActionKind = "bls"
	// AggressorBeaconActionInject identifies binject.
	AggressorBeaconActionInject AggressorBeaconActionKind = "binject"
	// AggressorBeaconActionInlineExecutePE identifies binline_execute_pe.
	AggressorBeaconActionInlineExecutePE AggressorBeaconActionKind = "binline_execute_pe"
	// AggressorBeaconActionIPConfig identifies bipconfig.
	AggressorBeaconActionIPConfig AggressorBeaconActionKind = "bipconfig"
	// AggressorBeaconActionJobSendData identifies bjob_send_data.
	AggressorBeaconActionJobSendData AggressorBeaconActionKind = "bjob_send_data"
	// AggressorBeaconActionJobKill identifies bjobkill.
	AggressorBeaconActionJobKill AggressorBeaconActionKind = "bjobkill"
	// AggressorBeaconActionJobs identifies bjobs.
	AggressorBeaconActionJobs AggressorBeaconActionKind = "bjobs"
	// AggressorBeaconActionKerberosCCacheUse identifies bkerberos_ccache_use.
	AggressorBeaconActionKerberosCCacheUse AggressorBeaconActionKind = "bkerberos_ccache_use"
	// AggressorBeaconActionKerberosTicketPurge identifies bkerberos_ticket_purge.
	AggressorBeaconActionKerberosTicketPurge AggressorBeaconActionKind = "bkerberos_ticket_purge"
	// AggressorBeaconActionKerberosTicketUse identifies bkerberos_ticket_use.
	AggressorBeaconActionKerberosTicketUse AggressorBeaconActionKind = "bkerberos_ticket_use"
	// AggressorBeaconActionKeylogger identifies bkeylogger.
	AggressorBeaconActionKeylogger AggressorBeaconActionKind = "bkeylogger"
	// AggressorBeaconActionKill identifies bkill.
	AggressorBeaconActionKill AggressorBeaconActionKind = "bkill"
	// AggressorBeaconActionLink identifies blink.
	AggressorBeaconActionLink AggressorBeaconActionKind = "blink"
	// AggressorBeaconActionLoginUser identifies bloginuser.
	AggressorBeaconActionLoginUser AggressorBeaconActionKind = "bloginuser"
	// AggressorBeaconActionLogonPasswords identifies blogonpasswords.
	AggressorBeaconActionLogonPasswords AggressorBeaconActionKind = "blogonpasswords"
	// AggressorBeaconActionMode identifies bmode.
	AggressorBeaconActionMode AggressorBeaconActionKind = "bmode"
	// AggressorBeaconActionNote identifies bnote.
	AggressorBeaconActionNote AggressorBeaconActionKind = "bnote"
	// AggressorBeaconActionPassTheHash identifies bpassthehash.
	AggressorBeaconActionPassTheHash AggressorBeaconActionKind = "bpassthehash"
	// AggressorBeaconActionPause identifies bpause.
	AggressorBeaconActionPause AggressorBeaconActionKind = "bpause"
	// AggressorBeaconActionPowerShell identifies bpowershell.
	AggressorBeaconActionPowerShell AggressorBeaconActionKind = "bpowershell"
	// AggressorBeaconActionPowerShellImport identifies bpowershell_import.
	AggressorBeaconActionPowerShellImport AggressorBeaconActionKind = "bpowershell_import"
	// AggressorBeaconActionParentPID identifies bppid.
	AggressorBeaconActionParentPID AggressorBeaconActionKind = "bppid"
	// AggressorBeaconActionPrintScreen identifies bprintscreen.
	AggressorBeaconActionPrintScreen AggressorBeaconActionKind = "bprintscreen"
	// AggressorBeaconActionListProcesses identifies bps.
	AggressorBeaconActionListProcesses AggressorBeaconActionKind = "bps"
	// AggressorBeaconActionPSExec identifies bpsexec.
	AggressorBeaconActionPSExec AggressorBeaconActionKind = "bpsexec"
	// AggressorBeaconActionPSExecCommand identifies bpsexec_command.
	AggressorBeaconActionPSExecCommand AggressorBeaconActionKind = "bpsexec_command"
	// AggressorBeaconActionPrintWorkingDirectory identifies bpwd.
	AggressorBeaconActionPrintWorkingDirectory AggressorBeaconActionKind = "bpwd"
	// AggressorBeaconActionRegistryQueryValue identifies breg_queryv.
	AggressorBeaconActionRegistryQueryValue AggressorBeaconActionKind = "breg_queryv"
	// AggressorBeaconActionRevertToSelf identifies brev2self.
	AggressorBeaconActionRevertToSelf AggressorBeaconActionKind = "brev2self"
	// AggressorBeaconActionRemove identifies brm.
	AggressorBeaconActionRemove AggressorBeaconActionKind = "brm"
	// AggressorBeaconActionReversePortForward identifies brportfwd.
	AggressorBeaconActionReversePortForward AggressorBeaconActionKind = "brportfwd"
	// AggressorBeaconActionReversePortForwardLocal identifies brportfwd_local.
	AggressorBeaconActionReversePortForwardLocal AggressorBeaconActionKind = "brportfwd_local"
	// AggressorBeaconActionReversePortForwardStop identifies brportfwd_stop.
	AggressorBeaconActionReversePortForwardStop AggressorBeaconActionKind = "brportfwd_stop"
	// AggressorBeaconActionRun identifies brun.
	AggressorBeaconActionRun AggressorBeaconActionKind = "brun"
	// AggressorBeaconActionRunAs identifies brunas.
	AggressorBeaconActionRunAs AggressorBeaconActionKind = "brunas"
	// AggressorBeaconActionRunUnder identifies brunu.
	AggressorBeaconActionRunUnder AggressorBeaconActionKind = "brunu"
	// AggressorBeaconActionScreenshot identifies bscreenshot.
	AggressorBeaconActionScreenshot AggressorBeaconActionKind = "bscreenshot"
	// AggressorBeaconActionScreenwatch identifies bscreenwatch.
	AggressorBeaconActionScreenwatch AggressorBeaconActionKind = "bscreenwatch"
	// AggressorBeaconActionSetEnvironment identifies bsetenv.
	AggressorBeaconActionSetEnvironment AggressorBeaconActionKind = "bsetenv"
	// AggressorBeaconActionShell identifies bshell.
	AggressorBeaconActionShell AggressorBeaconActionKind = "bshell"
	// AggressorBeaconActionShellcodeInject identifies bshinject.
	AggressorBeaconActionShellcodeInject AggressorBeaconActionKind = "bshinject"
	// AggressorBeaconActionShellcodeSpawn identifies bshspawn.
	AggressorBeaconActionShellcodeSpawn AggressorBeaconActionKind = "bshspawn"
	// AggressorBeaconActionSleep identifies bsleep.
	AggressorBeaconActionSleep AggressorBeaconActionKind = "bsleep"
	// AggressorBeaconActionSleepUnified identifies bsleepu.
	AggressorBeaconActionSleepUnified AggressorBeaconActionKind = "bsleepu"
	// AggressorBeaconActionSOCKS identifies bsocks.
	AggressorBeaconActionSOCKS AggressorBeaconActionKind = "bsocks"
	// AggressorBeaconActionSOCKSStop identifies bsocks_stop.
	AggressorBeaconActionSOCKSStop AggressorBeaconActionKind = "bsocks_stop"
	// AggressorBeaconActionSpawn identifies bspawn.
	AggressorBeaconActionSpawn AggressorBeaconActionKind = "bspawn"
	// AggressorBeaconActionSpawnAs identifies bspawnas.
	AggressorBeaconActionSpawnAs AggressorBeaconActionKind = "bspawnas"
	// AggressorBeaconActionSpawnTo identifies bspawnto.
	AggressorBeaconActionSpawnTo AggressorBeaconActionKind = "bspawnto"
	// AggressorBeaconActionSpawnUnder identifies bspawnu.
	AggressorBeaconActionSpawnUnder AggressorBeaconActionKind = "bspawnu"
	// AggressorBeaconActionSpawnTunnel identifies bspunnel.
	AggressorBeaconActionSpawnTunnel AggressorBeaconActionKind = "bspunnel"
	// AggressorBeaconActionSpawnTunnelLocal identifies bspunnel_local.
	AggressorBeaconActionSpawnTunnelLocal AggressorBeaconActionKind = "bspunnel_local"
	// AggressorBeaconActionSSH identifies bssh.
	AggressorBeaconActionSSH AggressorBeaconActionKind = "bssh"
	// AggressorBeaconActionSSHKey identifies bssh_key.
	AggressorBeaconActionSSHKey AggressorBeaconActionKind = "bssh_key"
	// AggressorBeaconActionStealToken identifies bsteal_token.
	AggressorBeaconActionStealToken AggressorBeaconActionKind = "bsteal_token"
	// AggressorBeaconActionSudo identifies bsudo.
	AggressorBeaconActionSudo AggressorBeaconActionKind = "bsudo"
	// AggressorBeaconActionSyscallMethod identifies bsyscall_method.
	AggressorBeaconActionSyscallMethod AggressorBeaconActionKind = "bsyscall_method"
	// AggressorBeaconActionTokenStoreRemove identifies btoken_store_remove.
	AggressorBeaconActionTokenStoreRemove AggressorBeaconActionKind = "btoken_store_remove"
	// AggressorBeaconActionTokenStoreRemoveAll identifies btoken_store_remove_all.
	AggressorBeaconActionTokenStoreRemoveAll AggressorBeaconActionKind = "btoken_store_remove_all"
	// AggressorBeaconActionTokenStoreShow identifies btoken_store_show.
	AggressorBeaconActionTokenStoreShow AggressorBeaconActionKind = "btoken_store_show"
	// AggressorBeaconActionTokenStoreSteal identifies btoken_store_steal.
	AggressorBeaconActionTokenStoreSteal AggressorBeaconActionKind = "btoken_store_steal"
	// AggressorBeaconActionTokenStoreStealAndUse identifies btoken_store_steal_and_use.
	AggressorBeaconActionTokenStoreStealAndUse AggressorBeaconActionKind = "btoken_store_steal_and_use"
	// AggressorBeaconActionTokenStoreUse identifies btoken_store_use.
	AggressorBeaconActionTokenStoreUse AggressorBeaconActionKind = "btoken_store_use"
	// AggressorBeaconActionUnlink identifies bunlink.
	AggressorBeaconActionUnlink AggressorBeaconActionKind = "bunlink"
	// AggressorBeaconActionConsoleWatermark identifies beacon_console_watermark.
	AggressorBeaconActionConsoleWatermark AggressorBeaconActionKind = "beacon_console_watermark"
	// AggressorBeaconActionConsoleWatermarkReset identifies beacon_console_watermark_reset.
	AggressorBeaconActionConsoleWatermarkReset AggressorBeaconActionKind = "beacon_console_watermark_reset"
	// AggressorBeaconActionJobHideOutput identifies beacon_job_hide_output.
	AggressorBeaconActionJobHideOutput AggressorBeaconActionKind = "beacon_job_hide_output"
	// AggressorBeaconActionJobName identifies beacon_job_name.
	AggressorBeaconActionJobName AggressorBeaconActionKind = "beacon_job_name"
	// AggressorBeaconActionSmartLink identifies beacon_link.
	AggressorBeaconActionSmartLink AggressorBeaconActionKind = "beacon_link"
	// AggressorBeaconActionRemoveFromDisplay identifies beacon_remove.
	AggressorBeaconActionRemoveFromDisplay AggressorBeaconActionKind = "beacon_remove"
	// AggressorBeaconActionStagePipe identifies beacon_stage_pipe.
	AggressorBeaconActionStagePipe AggressorBeaconActionKind = "beacon_stage_pipe"
	// AggressorBeaconActionStageTCP identifies beacon_stage_tcp.
	AggressorBeaconActionStageTCP AggressorBeaconActionKind = "beacon_stage_tcp"
	// AggressorBeaconActionHashdump identifies bhashdump.
	AggressorBeaconActionHashdump AggressorBeaconActionKind = "bhashdump"
	// AggressorBeaconActionNet identifies bnet.
	AggressorBeaconActionNet AggressorBeaconActionKind = "bnet"
	// AggressorBeaconActionPowerShellImportClear identifies bpowershell_import_clear.
	AggressorBeaconActionPowerShellImportClear AggressorBeaconActionKind = "bpowershell_import_clear"
	// AggressorBeaconActionPowerPick identifies bpowerpick.
	AggressorBeaconActionPowerPick AggressorBeaconActionKind = "bpowerpick"
	// AggressorBeaconActionPowerShellInject identifies bpsinject.
	AggressorBeaconActionPowerShellInject AggressorBeaconActionKind = "bpsinject"
	// AggressorBeaconActionMimikatz identifies bmimikatz.
	AggressorBeaconActionMimikatz AggressorBeaconActionKind = "bmimikatz"
	// AggressorBeaconActionMimikatzSmall identifies bmimikatz_small.
	AggressorBeaconActionMimikatzSmall AggressorBeaconActionKind = "bmimikatz_small"
	// AggressorBeaconActionPortscan identifies bportscan.
	AggressorBeaconActionPortscan AggressorBeaconActionKind = "bportscan"
	// AggressorBeaconActionDLLSpawn identifies bdllspawn.
	AggressorBeaconActionDLLSpawn AggressorBeaconActionKind = "bdllspawn"
	// AggressorBeaconActionExecuteAssembly identifies bexecute_assembly.
	AggressorBeaconActionExecuteAssembly AggressorBeaconActionKind = "bexecute_assembly"
	// AggressorBeaconActionInlineExecute identifies binline_execute.
	AggressorBeaconActionInlineExecute AggressorBeaconActionKind = "binline_execute"
	// AggressorBeaconActionDLLInject identifies bdllinject.
	AggressorBeaconActionDLLInject AggressorBeaconActionKind = "bdllinject"
	// AggressorBeaconActionReadPipe identifies bread_pipe.
	AggressorBeaconActionReadPipe AggressorBeaconActionKind = "bread_pipe"
	// AggressorBeaconActionCopy identifies bcp(idOrArray, source, destination).
	AggressorBeaconActionCopy AggressorBeaconActionKind = "bcp"
	// AggressorBeaconActionListDrives identifies bdrives(idOrArray).
	AggressorBeaconActionListDrives AggressorBeaconActionKind = "bdrives"
	// AggressorBeaconActionMakeDirectory identifies bmkdir(idOrArray, folder).
	AggressorBeaconActionMakeDirectory AggressorBeaconActionKind = "bmkdir"
	// AggressorBeaconActionMove identifies bmv(idOrArray, source, destination).
	AggressorBeaconActionMove AggressorBeaconActionKind = "bmv"
	// AggressorBeaconActionTimestomp identifies
	// btimestomp(idOrArray, targetFile, sourceFile).
	AggressorBeaconActionTimestomp AggressorBeaconActionKind = "btimestomp"
	// AggressorBeaconActionUpload identifies bupload(idOrArray, localPath).
	AggressorBeaconActionUpload AggressorBeaconActionKind = "bupload"
	// AggressorBeaconActionUploadRaw identifies
	// bupload_raw(idOrArray, remoteFile, rawContent[, localPath]).
	AggressorBeaconActionUploadRaw AggressorBeaconActionKind = "bupload_raw"
)

// AggressorCallbackState preserves the three observable shapes of an optional
// callback argument. Callback alone cannot distinguish an omitted argument
// from an explicitly supplied $null value because both have no Callable.
type AggressorCallbackState uint8

const (
	// AggressorCallbackOmitted means the function form had no callback argument.
	// It is the zero value so actions without callback support naturally use it.
	AggressorCallbackOmitted AggressorCallbackState = iota
	// AggressorCallbackNull means the script explicitly supplied $null in the
	// optional callback position.
	AggressorCallbackNull
	// AggressorCallbackCallable means Callback contains a retained, script-owned
	// callable capability.
	AggressorCallbackCallable
)

// AggressorBeaconAction is one resolved Beacon operational request. Most
// actions describe remote tasking; bbeacon_interpreter_lint and the
// beacon_console_*, beacon_job_*, and beacon_remove forms describe
// Cobalt-client effects associated with a Beacon. Kind is the canonical action
// and Name is the exact normalized function spelling invoked by the script.
// RuntimeID is the nonzero process-local identity of the originating runtime;
// Script and Span identify the call site without exposing a *Runtime.
// Bindings is the opaque callback capability for that exact Runtime; retaining
// it retains the Runtime but does not expose its evaluator or binding registry.
//
// Target is the first, action-target argument. Arguments contains the
// remaining ordinary arguments in source order, excluding both Target and an
// optional callback. Wrappers resolve every Value exactly once before the
// provider call and copy the Arguments slice, so no pass-by-name Cell or
// Invocation crosses this boundary. OPFOR deliberately does not coerce,
// flatten, or otherwise interpret these spec-neutral fields. Scalar Values
// are immutable, while arrays, hashes, functions, objects, and nested compound
// graphs retain their ordinary reference identity. A provider which retains
// the action therefore also retains any capabilities reachable through those
// Values; snapshot scalar coercions or detached graphs when that lifetime is
// undesirable.
//
// Filesystem action arguments are task descriptions, not instructions for
// OPFOR's local filesystem. In particular, bupload's local path and
// bupload_raw's raw content and optional local path are transferred unchanged;
// reading local files, fan-out across an array Target, and issuing remote
// filesystem tasks remain importer-owned effects.
//
// CallbackState distinguishes omission, explicit $null, and a callable value.
// Callback is nil for Omitted and Null. For Callable it is a retained
// capability tied to Script: it may be invoked after Dispatch returns, honors
// the invocation context supplied by the provider, and rejects calls after
// the owning script unloads or its Runtime closes.
type AggressorBeaconAction struct {
	Kind AggressorBeaconActionKind
	Name string

	RuntimeID RuntimeID
	Script    ScriptID
	Span      Span
	Bindings  AggressorBindings

	Target        Value
	Arguments     []Value
	Callback      Callable
	CallbackState AggressorCallbackState
}

// AggressorBeaconActionProvider performs Beacon tasking and associated
// Cobalt-client Beacon operations for the typed action wrappers.
// DispatchAggressorBeaconAction is called synchronously exactly once for each
// valid invocation when a provider is configured. A nil error means the action
// was accepted. Returning an error rejects the invocation; OPFOR does not retry
// it through Host, because doing so could duplicate a side effect.
//
// Implementations may be called concurrently and should observe ctx. A
// provider may retain Action Values and Callback subject to the capability
// lifetimes documented on AggressorBeaconAction, but must not retain ctx after
// DispatchAggressorBeaconAction returns.
type AggressorBeaconActionProvider interface {
	DispatchAggressorBeaconAction(context.Context, AggressorBeaconAction) error
}

// AggressorBeaconActionProviderFunc adapts a function to
// AggressorBeaconActionProvider.
type AggressorBeaconActionProviderFunc func(context.Context, AggressorBeaconAction) error

// DispatchAggressorBeaconAction calls function.
func (function AggressorBeaconActionProviderFunc) DispatchAggressorBeaconAction(
	ctx context.Context,
	action AggressorBeaconAction,
) error {
	if function == nil {
		return errors.New("opfor: Aggressor Beacon action provider is nil")
	}
	return function(ctx, action)
}

// WithAggressorBeaconActionProvider installs the typed importer boundary for
// the supported Beacon operational actions. Provider success is side-effect only;
// native wrappers return $null. Provider errors are authoritative and never
// fall back to Host. Importer-defined WithFunction callbacks retain precedence
// over the native wrappers.
func WithAggressorBeaconActionProvider(provider AggressorBeaconActionProvider) Option {
	return func(config *runtimeConfig) error {
		if isNilInterface(provider) {
			return errors.New("opfor: Aggressor Beacon action provider is nil")
		}
		config.aggressorBeaconActionProvider = provider
		return nil
	}
}
