package opfor

import (
	"context"
	"errors"
	"fmt"
)

type aggressorBeaconActionSpec struct {
	kind             AggressorBeaconActionKind
	minimum          int
	maximum          int
	hasCallback      bool
	callbackIndex    int
	callbackRequired bool
}

// aggressorBeaconActionSpecs is intentionally limited to a coherent documented
// operational surface. It is not a heuristic for every function whose name
// starts with b: value-returning queries, transcript records, technique
// registries, and beacon_inline_execute have distinct native boundaries and
// remain outside this dispatcher.
var aggressorBeaconActionSpecs = map[string]aggressorBeaconActionSpec{
	"bargue_add":                     {kind: AggressorBeaconActionArgumentSpoofAdd, minimum: 3, maximum: 3},
	"bargue_list":                    {kind: AggressorBeaconActionArgumentSpoofList, minimum: 1, maximum: 1},
	"bargue_remove":                  {kind: AggressorBeaconActionArgumentSpoofRemove, minimum: 2, maximum: 2},
	"bbeacon_config":                 {kind: AggressorBeaconActionConfigure, minimum: 2, maximum: 6},
	"bbeacon_gate":                   {kind: AggressorBeaconActionGate, minimum: 2, maximum: 2},
	"bbeacon_interpreter":            {kind: AggressorBeaconActionInterpreter, minimum: 2, maximum: 4, hasCallback: true, callbackIndex: 3},
	"bbeacon_interpreter_lint":       {kind: AggressorBeaconActionInterpreterLint, minimum: 2, maximum: 3, hasCallback: true, callbackIndex: 2},
	"bblockdlls":                     {kind: AggressorBeaconActionBlockDLLs, minimum: 2, maximum: 2},
	"bbrowserpivot":                  {kind: AggressorBeaconActionBrowserPivot, minimum: 3, maximum: 3},
	"bbrowserpivot_stop":             {kind: AggressorBeaconActionBrowserPivotStop, minimum: 1, maximum: 1},
	"bcancel":                        {kind: AggressorBeaconActionCancelDownload, minimum: 2, maximum: 2},
	"bcd":                            {kind: AggressorBeaconActionChangeDirectory, minimum: 2, maximum: 2},
	"bcheckin":                       {kind: AggressorBeaconActionCheckin, minimum: 1, maximum: 1},
	"bclear":                         {kind: AggressorBeaconActionClearTasks, minimum: 1, maximum: 1},
	"bclipboard":                     {kind: AggressorBeaconActionClipboard, minimum: 1, maximum: 1},
	"bconnect":                       {kind: AggressorBeaconActionConnect, minimum: 2, maximum: 3},
	"bcovertvpn":                     {kind: AggressorBeaconActionCovertVPN, minimum: 3, maximum: 4},
	"bcp":                            {kind: AggressorBeaconActionCopy, minimum: 3, maximum: 3},
	"bdata_store_list":               {kind: AggressorBeaconActionDataStoreList, minimum: 1, maximum: 1},
	"bdata_store_load":               {kind: AggressorBeaconActionDataStoreLoad, minimum: 3, maximum: 4},
	"bdata_store_unload":             {kind: AggressorBeaconActionDataStoreUnload, minimum: 2, maximum: 2},
	"bdcsync":                        {kind: AggressorBeaconActionDCSync, minimum: 2, maximum: 5},
	"bdesktop":                       {kind: AggressorBeaconActionDesktop, minimum: 1, maximum: 1},
	"bdllinject":                     {kind: AggressorBeaconActionDLLInject, minimum: 3, maximum: 3},
	"bdllload":                       {kind: AggressorBeaconActionDLLLoad, minimum: 3, maximum: 3},
	"bdllspawn":                      {kind: AggressorBeaconActionDLLSpawn, minimum: 6, maximum: 7, hasCallback: true, callbackIndex: 6},
	"bdownload":                      {kind: AggressorBeaconActionDownload, minimum: 2, maximum: 2},
	"bdrives":                        {kind: AggressorBeaconActionListDrives, minimum: 1, maximum: 1},
	"bexecute":                       {kind: AggressorBeaconActionExecute, minimum: 2, maximum: 2},
	"bexecute_assembly":              {kind: AggressorBeaconActionExecuteAssembly, minimum: 3, maximum: 5, hasCallback: true, callbackIndex: 4},
	"bexit":                          {kind: AggressorBeaconActionExit, minimum: 1, maximum: 1},
	"bgetprivs":                      {kind: AggressorBeaconActionEnablePrivileges, minimum: 2, maximum: 2},
	"bgetsystem":                     {kind: AggressorBeaconActionGetSystem, minimum: 1, maximum: 1},
	"bgetuid":                        {kind: AggressorBeaconActionGetUID, minimum: 1, maximum: 1},
	"bhashdump":                      {kind: AggressorBeaconActionHashdump, minimum: 1, maximum: 4, hasCallback: true, callbackIndex: 3},
	"binject":                        {kind: AggressorBeaconActionInject, minimum: 3, maximum: 4},
	"binline_execute":                {kind: AggressorBeaconActionInlineExecute, minimum: 3, maximum: 4, hasCallback: true, callbackIndex: 3},
	"binline_execute_pe":             {kind: AggressorBeaconActionInlineExecutePE, minimum: 3, maximum: 4, hasCallback: true, callbackIndex: 3},
	"bipconfig":                      {kind: AggressorBeaconActionIPConfig, minimum: 2, maximum: 2, hasCallback: true, callbackIndex: 1, callbackRequired: true},
	"bjob_send_data":                 {kind: AggressorBeaconActionJobSendData, minimum: 3, maximum: 3},
	"bjobkill":                       {kind: AggressorBeaconActionJobKill, minimum: 2, maximum: 2},
	"bjobs":                          {kind: AggressorBeaconActionJobs, minimum: 1, maximum: 1},
	"bkerberos_ccache_use":           {kind: AggressorBeaconActionKerberosCCacheUse, minimum: 2, maximum: 2},
	"bkerberos_ticket_purge":         {kind: AggressorBeaconActionKerberosTicketPurge, minimum: 1, maximum: 1},
	"bkerberos_ticket_use":           {kind: AggressorBeaconActionKerberosTicketUse, minimum: 2, maximum: 2},
	"bkeylogger":                     {kind: AggressorBeaconActionKeylogger, minimum: 1, maximum: 3},
	"bkill":                          {kind: AggressorBeaconActionKill, minimum: 2, maximum: 2},
	"blink":                          {kind: AggressorBeaconActionLink, minimum: 2, maximum: 3},
	"bloginuser":                     {kind: AggressorBeaconActionLoginUser, minimum: 4, maximum: 4},
	"blogonpasswords":                {kind: AggressorBeaconActionLogonPasswords, minimum: 1, maximum: 3},
	"bls":                            {kind: AggressorBeaconActionListFiles, minimum: 1, maximum: 3, hasCallback: true, callbackIndex: 2},
	"bmimikatz":                      {kind: AggressorBeaconActionMimikatz, minimum: 2, maximum: 5, hasCallback: true, callbackIndex: 4},
	"bmimikatz_small":                {kind: AggressorBeaconActionMimikatzSmall, minimum: 2, maximum: 5, hasCallback: true, callbackIndex: 4},
	"bmkdir":                         {kind: AggressorBeaconActionMakeDirectory, minimum: 2, maximum: 2},
	"bmode":                          {kind: AggressorBeaconActionMode, minimum: 2, maximum: 2},
	"bmv":                            {kind: AggressorBeaconActionMove, minimum: 3, maximum: 3},
	"bnet":                           {kind: AggressorBeaconActionNet, minimum: 4, maximum: 7, hasCallback: true, callbackIndex: 6},
	"bnote":                          {kind: AggressorBeaconActionNote, minimum: 2, maximum: 2},
	"bpassthehash":                   {kind: AggressorBeaconActionPassTheHash, minimum: 4, maximum: 6},
	"bpause":                         {kind: AggressorBeaconActionPause, minimum: 2, maximum: 2},
	"bportscan":                      {kind: AggressorBeaconActionPortscan, minimum: 5, maximum: 8, hasCallback: true, callbackIndex: 7},
	"bpowerpick":                     {kind: AggressorBeaconActionPowerPick, minimum: 2, maximum: 5, hasCallback: true, callbackIndex: 4},
	"bpowershell":                    {kind: AggressorBeaconActionPowerShell, minimum: 2, maximum: 4, hasCallback: true, callbackIndex: 3},
	"bpowershell_import":             {kind: AggressorBeaconActionPowerShellImport, minimum: 2, maximum: 2},
	"bpowershell_import_clear":       {kind: AggressorBeaconActionPowerShellImportClear, minimum: 1, maximum: 1},
	"bppid":                          {kind: AggressorBeaconActionParentPID, minimum: 2, maximum: 2},
	"bprintscreen":                   {kind: AggressorBeaconActionPrintScreen, minimum: 1, maximum: 3},
	"bps":                            {kind: AggressorBeaconActionListProcesses, minimum: 1, maximum: 2, hasCallback: true, callbackIndex: 1},
	"bpsexec":                        {kind: AggressorBeaconActionPSExec, minimum: 4, maximum: 5},
	"bpsexec_command":                {kind: AggressorBeaconActionPSExecCommand, minimum: 4, maximum: 4},
	"bpsinject":                      {kind: AggressorBeaconActionPowerShellInject, minimum: 4, maximum: 5, hasCallback: true, callbackIndex: 4},
	"bpwd":                           {kind: AggressorBeaconActionPrintWorkingDirectory, minimum: 1, maximum: 1},
	"bread_pipe":                     {kind: AggressorBeaconActionReadPipe, minimum: 7, maximum: 8, hasCallback: true, callbackIndex: 7},
	"breg_queryv":                    {kind: AggressorBeaconActionRegistryQueryValue, minimum: 4, maximum: 4},
	"brev2self":                      {kind: AggressorBeaconActionRevertToSelf, minimum: 1, maximum: 1},
	"brm":                            {kind: AggressorBeaconActionRemove, minimum: 2, maximum: 2},
	"brportfwd":                      {kind: AggressorBeaconActionReversePortForward, minimum: 4, maximum: 4},
	"brportfwd_local":                {kind: AggressorBeaconActionReversePortForwardLocal, minimum: 4, maximum: 4},
	"brportfwd_stop":                 {kind: AggressorBeaconActionReversePortForwardStop, minimum: 2, maximum: 2},
	"brun":                           {kind: AggressorBeaconActionRun, minimum: 2, maximum: 2},
	"brunas":                         {kind: AggressorBeaconActionRunAs, minimum: 5, maximum: 5},
	"brunu":                          {kind: AggressorBeaconActionRunUnder, minimum: 3, maximum: 3},
	"bscreenwatch":                   {kind: AggressorBeaconActionScreenwatch, minimum: 1, maximum: 3},
	"bsetenv":                        {kind: AggressorBeaconActionSetEnvironment, minimum: 3, maximum: 3},
	"bshell":                         {kind: AggressorBeaconActionShell, minimum: 2, maximum: 2},
	"bshinject":                      {kind: AggressorBeaconActionShellcodeInject, minimum: 4, maximum: 4},
	"bshspawn":                       {kind: AggressorBeaconActionShellcodeSpawn, minimum: 3, maximum: 3},
	"bscreenshot":                    {kind: AggressorBeaconActionScreenshot, minimum: 1, maximum: 3},
	"bsleep":                         {kind: AggressorBeaconActionSleep, minimum: 3, maximum: 3},
	"bsleepu":                        {kind: AggressorBeaconActionSleepUnified, minimum: 2, maximum: 2},
	"bsocks":                         {kind: AggressorBeaconActionSOCKS, minimum: 2, maximum: 7},
	"bsocks_stop":                    {kind: AggressorBeaconActionSOCKSStop, minimum: 1, maximum: 1},
	"bspawn":                         {kind: AggressorBeaconActionSpawn, minimum: 2, maximum: 3},
	"bspawnas":                       {kind: AggressorBeaconActionSpawnAs, minimum: 5, maximum: 5},
	"bspawnto":                       {kind: AggressorBeaconActionSpawnTo, minimum: 3, maximum: 3},
	"bspawnu":                        {kind: AggressorBeaconActionSpawnUnder, minimum: 3, maximum: 3},
	"bspunnel":                       {kind: AggressorBeaconActionSpawnTunnel, minimum: 5, maximum: 5},
	"bspunnel_local":                 {kind: AggressorBeaconActionSpawnTunnelLocal, minimum: 5, maximum: 5},
	"bssh":                           {kind: AggressorBeaconActionSSH, minimum: 5, maximum: 7},
	"bssh_key":                       {kind: AggressorBeaconActionSSHKey, minimum: 5, maximum: 7},
	"bsteal_token":                   {kind: AggressorBeaconActionStealToken, minimum: 2, maximum: 3},
	"bsudo":                          {kind: AggressorBeaconActionSudo, minimum: 3, maximum: 3},
	"bsyscall_method":                {kind: AggressorBeaconActionSyscallMethod, minimum: 2, maximum: 2},
	"btimestomp":                     {kind: AggressorBeaconActionTimestomp, minimum: 3, maximum: 3},
	"btoken_store_remove":            {kind: AggressorBeaconActionTokenStoreRemove, minimum: 2, maximum: 2},
	"btoken_store_remove_all":        {kind: AggressorBeaconActionTokenStoreRemoveAll, minimum: 1, maximum: 1},
	"btoken_store_show":              {kind: AggressorBeaconActionTokenStoreShow, minimum: 1, maximum: 1},
	"btoken_store_steal":             {kind: AggressorBeaconActionTokenStoreSteal, minimum: 3, maximum: 3},
	"btoken_store_steal_and_use":     {kind: AggressorBeaconActionTokenStoreStealAndUse, minimum: 3, maximum: 3},
	"btoken_store_use":               {kind: AggressorBeaconActionTokenStoreUse, minimum: 2, maximum: 2},
	"bunlink":                        {kind: AggressorBeaconActionUnlink, minimum: 2, maximum: 3},
	"bupload":                        {kind: AggressorBeaconActionUpload, minimum: 2, maximum: 2},
	"bupload_raw":                    {kind: AggressorBeaconActionUploadRaw, minimum: 3, maximum: 4},
	"beacon_console_watermark":       {kind: AggressorBeaconActionConsoleWatermark, minimum: 2, maximum: 2},
	"beacon_console_watermark_reset": {kind: AggressorBeaconActionConsoleWatermarkReset, minimum: 1, maximum: 1},
	"beacon_job_hide_output":         {kind: AggressorBeaconActionJobHideOutput, minimum: 3, maximum: 3},
	"beacon_job_name":                {kind: AggressorBeaconActionJobName, minimum: 3, maximum: 3},
	"beacon_link":                    {kind: AggressorBeaconActionSmartLink, minimum: 3, maximum: 3},
	"beacon_remove":                  {kind: AggressorBeaconActionRemoveFromDisplay, minimum: 1, maximum: 1},
	"beacon_stage_pipe":              {kind: AggressorBeaconActionStagePipe, minimum: 4, maximum: 4},
	"beacon_stage_tcp":               {kind: AggressorBeaconActionStageTCP, minimum: 5, maximum: 5},
}

// aggressorBeaconActionFunctions returns native wrappers around the
// importer-owned Beacon operational boundary. With no provider, a valid call
// preserves the original reference-bearing Host invocation exactly once.
func (r *Runtime) aggressorBeaconActionFunctions() map[string]NativeFunc {
	functions := make(map[string]NativeFunc, len(aggressorBeaconActionSpecs))
	for name := range aggressorBeaconActionSpecs {
		functions[name] = r.aggressorBeaconAction
	}
	return functions
}

func (r *Runtime) aggressorBeaconAction(ctx context.Context, invocation Invocation) (Value, error) {
	spec, exists := aggressorBeaconActionSpecs[invocation.Name]
	if !exists {
		return Null(), &UnsupportedError{
			Operation: "Aggressor Beacon action",
			Name:      invocation.Name,
			Span:      invocation.Span,
		}
	}
	if err := requireAggressorBeaconActionArguments(invocation, spec.minimum, spec.maximum); err != nil {
		return Null(), err
	}
	if r == nil {
		return Null(), errors.New("opfor: runtime is nil")
	}

	provider := r.aggressorBeaconActionProvider
	if isNilInterface(provider) {
		// This compatibility path must not resolve, copy, or replace any
		// Argument. In particular, a Host continues to receive pass-by-name and
		// ordinary bare-variable Cell capabilities exactly as before.
		result, err := r.host.Call(ctx, invocation)
		return result, preserveNativeBoundaryError(ctx, err)
	}

	if ctx == nil {
		ctx = context.Background()
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}

	// Snapshot every reference exactly once. Value copies intentionally retain
	// compound identity; only the top-level Arguments slice is detached.
	values := invocation.Values()
	action := AggressorBeaconAction{
		Kind:          spec.kind,
		Name:          invocation.Name,
		RuntimeID:     r.ID(),
		Script:        invocation.Script,
		Span:          invocation.Span,
		Bindings:      invocation.Bindings(),
		Target:        values[0],
		CallbackState: AggressorCallbackOmitted,
	}
	action.Arguments = make([]Value, 0, len(values)-1)
	for index := 1; index < len(values); index++ {
		if spec.hasCallback && index == spec.callbackIndex {
			if values[index].IsNull() {
				if spec.callbackRequired {
					return Null(), fmt.Errorf("&%s: argument %d is not callable: %w",
						builtinName(invocation.Name), index+1, ErrInvalidCallable)
				}
				action.CallbackState = AggressorCallbackNull
				continue
			}
			callback, err := invocation.RetainCallback(values[index])
			if err != nil {
				if errors.Is(err, ErrInvalidCallable) {
					description := "is not callable or $null"
					if spec.callbackRequired {
						description = "is not callable"
					}
					return Null(), fmt.Errorf("&%s: argument %d %s: %w",
						builtinName(invocation.Name), index+1, description, err)
				}
				return Null(), err
			}
			action.Callback = callback
			action.CallbackState = AggressorCallbackCallable
			continue
		}
		action.Arguments = append(action.Arguments, values[index])
	}

	if err := provider.DispatchAggressorBeaconAction(ctx, action); err != nil {
		return Null(), preserveNativeBoundaryError(ctx, err)
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}
	return Null(), nil
}

func requireAggressorBeaconActionArguments(invocation Invocation, minimum, maximum int) error {
	count := len(invocation.Arguments)
	if count >= minimum && count <= maximum {
		return nil
	}
	if minimum == maximum {
		return requireExactAggressorClientArguments(invocation, minimum)
	}
	return fmt.Errorf("&%s: expected %d to %d argument(s), received %d",
		builtinName(invocation.Name), minimum, maximum, count)
}
