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
	"github.com/bishopfox/sliver/client/command/completers"
	"github.com/bishopfox/sliver/client/command/flags"
	"github.com/bishopfox/sliver/client/command/generate"
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Commands returns the “ command and its subcommands.
func Commands(con *console.SliverClient) []*cobra.Command {
	// Job control
	jobsCmd := &cobra.Command{
		Use:   consts.JobsStr,
		Short: "Job control",
		Long:  help.GetHelpFor([]string{consts.JobsStr}),
		Run: func(cmd *cobra.Command, args []string) {
			JobsCmd(cmd, con, args)
		},
		GroupID: consts.NetworkHelpGroup,
	}
	flags.Bind("jobs", true, jobsCmd, func(f *pflag.FlagSet) {
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})
	flags.Bind("jobs", false, jobsCmd, func(f *pflag.FlagSet) {
		f.Int32P("kill", "k", -1, "kill a background job")
		f.BoolP("kill-all", "K", false, "kill all jobs")
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})
	flags.BindFlagCompletions(jobsCmd, func(comp *carapace.ActionMap) {
		(*comp)["kill"] = JobsIDCompleter(con)
	})

	// Mutual TLS
	mtlsCmd := &cobra.Command{
		Use:   consts.MtlsStr,
		Short: "Start an mTLS listener",
		Long:  help.GetHelpFor([]string{consts.MtlsStr}),
		Run: func(cmd *cobra.Command, args []string) {
			MTLSListenerCmd(cmd, con, args)
		},
		GroupID: consts.NetworkHelpGroup,
	}
	flags.Bind("mTLS listener", false, mtlsCmd, func(f *pflag.FlagSet) {
		f.StringP("lhost", "L", "", "interface to bind server to")
		f.Uint32P("lport", "l", generate.DefaultMTLSLPort, "tcp listen port")
	})

	// Wireguard
	wgCmd := &cobra.Command{
		Use:   consts.WGStr,
		Short: "Start a WireGuard listener",
		Long:  help.GetHelpFor([]string{consts.WGStr}),
		Run: func(cmd *cobra.Command, args []string) {
			WGListenerCmd(cmd, con, args)
		},
		GroupID: consts.NetworkHelpGroup,
	}
	flags.Bind("WireGuard listener", false, wgCmd, func(f *pflag.FlagSet) {
		f.StringP("lhost", "L", "", "interface to bind server to")
		f.Uint32P("lport", "l", generate.DefaultWGLPort, "udp listen port")
		f.Uint32P("nport", "n", generate.DefaultWGNPort, "virtual tun interface listen port")
		f.Uint32P("key-port", "x", generate.DefaultWGKeyExPort, "virtual tun interface key exchange port")
	})

	// DNS
	dnsCmd := &cobra.Command{
		Use:   consts.DnsStr,
		Short: "Start a DNS listener",
		Long:  help.GetHelpFor([]string{consts.DnsStr}),
		Run: func(cmd *cobra.Command, args []string) {
			DNSListenerCmd(cmd, con, args)
		},
		GroupID: consts.NetworkHelpGroup,
	}
	flags.Bind("DNS listener", false, dnsCmd, func(f *pflag.FlagSet) {
		f.StringP("domains", "d", "", "parent domain(s) to use for DNS c2")
		f.BoolP("no-canaries", "c", false, "disable dns canary detection")
		f.StringP("lhost", "L", "", "interface to bind server to")
		f.Uint32P("lport", "l", generate.DefaultDNSLPort, "udp listen port")
	})

	// HTTP
	httpCmd := &cobra.Command{
		Use:   consts.HttpStr,
		Short: "Start an HTTP listener",
		Long:  help.GetHelpFor([]string{consts.HttpStr}),
		Run: func(cmd *cobra.Command, args []string) {
			HTTPListenerCmd(cmd, con, args)
		},
		GroupID: consts.NetworkHelpGroup,
	}
	flags.Bind("HTTP listener", false, httpCmd, func(f *pflag.FlagSet) {
		f.StringP("domain", "d", "", "limit responses to specific domain")
		f.StringP("website", "w", "", "website name (see websites cmd)")
		f.StringP("lhost", "L", "", "interface to bind server to")
		f.Uint32P("lport", "l", generate.DefaultHTTPLPort, "tcp listen port")
		f.BoolP("disable-otp", "D", false, "disable otp authentication")
		f.StringP("long-poll-timeout", "T", "1s", "server-side long poll timeout")
		f.StringP("long-poll-jitter", "J", "2s", "server-side long poll jitter")
	})
	flags.BindFlagCompletions(httpCmd, func(comp *carapace.ActionMap) {
		(*comp)["website"] = WebsiteNameCompleter(con)
	})
	registerWebsiteFlagCompletion(httpCmd, "website", con)

	// HTTPS
	httpsCmd := &cobra.Command{
		Use:   consts.HttpsStr,
		Short: "Start an HTTPS listener",
		Long:  help.GetHelpFor([]string{consts.HttpsStr}),
		Run: func(cmd *cobra.Command, args []string) {
			HTTPSListenerCmd(cmd, con, args)
		},
		GroupID: consts.NetworkHelpGroup,
	}
	flags.Bind("HTTPS listener", false, httpsCmd, func(f *pflag.FlagSet) {
		f.StringP("domain", "d", "", "limit responses to specific domain")
		f.StringP("website", "w", "", "website name (see websites cmd)")
		f.StringP("lhost", "L", "", "interface to bind server to")
		f.Uint32P("lport", "l", generate.DefaultHTTPSLPort, "tcp listen port")
		f.BoolP("disable-otp", "D", false, "disable otp authentication")
		f.StringP("long-poll-timeout", "T", "1s", "server-side long poll timeout")
		f.StringP("long-poll-jitter", "J", "2s", "server-side long poll jitter")

		f.StringP("cert", "c", "", "PEM encoded certificate file")
		f.StringP("key", "k", "", "PEM encoded private key file")
		f.BoolP("lets-encrypt", "e", false, "attempt to provision a let's encrypt certificate")
		f.BoolP("disable-randomized-jarm", "E", false, "disable randomized jarm fingerprints")
	})
	flags.BindFlagCompletions(httpsCmd, func(comp *carapace.ActionMap) {
		(*comp)["website"] = WebsiteNameCompleter(con)
		(*comp)["cert"] = carapace.ActionFiles().Tag("certificate file")
		(*comp)["key"] = carapace.ActionFiles().Tag("key file")
	})
	completers.RegisterLocalFilePathFlagCompletions(httpsCmd, "cert", "key")
	registerWebsiteFlagCompletion(httpsCmd, "website", con)

	// Staging listeners
	stageCmd := &cobra.Command{
		Use:   consts.StageListenerStr,
		Short: "Start a stager listener",
		Long:  help.GetHelpFor([]string{consts.StageListenerStr}),
		Run: func(cmd *cobra.Command, args []string) {
			StageListenerCmd(cmd, con, args)
		},
		GroupID: consts.NetworkHelpGroup,
	}
	flags.Bind("stage listener", false, stageCmd, func(f *pflag.FlagSet) {
		f.StringP("profile", "p", "", "implant profile name to link with the listener")
		f.StringP("url", "u", "", "URL to which the stager will call back to")
		f.StringP("cert", "c", "", "path to PEM encoded certificate file (HTTPS only)")
		f.StringP("key", "k", "", "path to PEM encoded private key file (HTTPS only)")
		f.BoolP("lets-encrypt", "e", false, "attempt to provision a let's encrypt certificate (HTTPS only)")
		f.String("aes-encrypt-key", "", "encrypt stage with AES encryption key")
		f.String("aes-encrypt-iv", "", "encrypt stage with AES encryption iv")
		f.String("rc4-encrypt-key", "", "encrypt stage with RC4 encryption key")
		f.StringP("compress", "C", "none", "compress the stage before encrypting (zlib, gzip, deflate9, none)")
		f.BoolP("prepend-size", "P", false, "prepend the size of the stage to the payload (to use with MSF stagers)")
	})
	flags.BindFlagCompletions(stageCmd, func(comp *carapace.ActionMap) {
		(*comp)["profile"] = generate.ProfileNameCompleter(con)
		(*comp)["cert"] = carapace.ActionFiles().Tag("certificate file")
		(*comp)["key"] = carapace.ActionFiles().Tag("key file")
		(*comp)["compress"] = carapace.ActionValues([]string{"zlib", "gzip", "deflate9", "none"}...).Tag("compression formats")
	})
	completers.RegisterLocalFilePathFlagCompletions(stageCmd, "cert", "key")

	// Trigger
	triggerCmd := &cobra.Command{
		Use:   consts.TriggerStr,
		Short: "Start an authenticated UDP trigger listener (task dispatcher)",
		Long:  help.GetHelpFor([]string{consts.TriggerStr}),
		Run: func(cmd *cobra.Command, args []string) {
			TriggerListenerCmd(cmd, con, args)
		},
		GroupID: consts.NetworkHelpGroup,
	}
	flags.Bind("Trigger listener", false, triggerCmd, func(f *pflag.FlagSet) {
		f.StringP("lhost", "L", "0.0.0.0", "interface to bind server to")
		f.Uint32P("lport", "l", 46290, "udp listen port")
		f.StringP("secret-env", "S", "", "env var name holding the HMAC shared secret (preferred; no secret in argv)")
		f.String("secret", "", "HMAC shared secret (direct value; visible in ps — prefer --secret-env). Omit both for stdin prompt")
		f.String("server-id", "sliver-trigger", "audit identifier embedded in events")
		f.StringArrayP("task", "i", nil, "task binding NAME:KIND:ARGS (repeatable; KIND in wake-beacon, stop-job, exec, reverse-shell)")
		f.StringArray("allowed-source", nil, "allow only this IP or CIDR (repeatable; empty=any)")
		f.StringArray("allowed-client", nil, "allow only this client_id (repeatable; empty=any)")
	})

	triggerTasksCmd := &cobra.Command{
		Use:   "tasks <job-id>",
		Short: "List task bindings for a running trigger listener",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			TriggerTasksCmd(cmd, con, args)
		},
	}
	triggerCmd.AddCommand(triggerTasksCmd)

	// Trigger dispatch: ad-hoc tasking by job ID (server-side task dispatch).
	triggerDispatchCmd := &cobra.Command{
		Use:   "dispatch <job-id> <task-name>",
		Short: "Dispatch an ad-hoc task to a running trigger listener",
		Long: `Dispatch an ad-hoc task to a running trigger listener by job ID.

The task-name must match a task binding already registered on the listener.
This enables interactive, on-the-fly tasking of active trigger jobs,
analogous to beacon interaction.

Examples:
  trigger dispatch 7 wake-beacon-alpha
  trigger dispatch 7 kill-mtls`,
		Args: cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			TriggerDispatchCmd(cmd, con, args)
		},
	}
	triggerCmd.AddCommand(triggerDispatchCmd)

	// Trigger send: send a signed UDP trigger packet to an implant.
	triggerSendCmd := &cobra.Command{
		Use:   "send <target-ip|trigger-index> <intent>",
		Short: "Send a signed trigger packet to an implant's triggerwake port",
		Long: `Construct a signed trigger packet (HMAC-SHA256, JSON-over-UDP)
and send it to an implant's triggerwake listener. Everything is handled
natively within sliver -- no external tools required.

The first argument can be either:
  - A target IP/hostname (backward compatible, e.g. 192.168.1.42)
  - A trigger index from the "triggers" command (integer, e.g. 1)

When a trigger index is used, the port, secret, and client-id are
auto-populated from the implant's build config and stored target mapping.

Intents:
  "wake"           Wake a dormant implant (fire-and-forget)
  "self-destruct"  Wipe the implant and exit (fire-and-forget, DESTRUCTIVE)
  "exec"           Execute a command and return output (bidirectional)

The secret must match the one baked into the implant at generation time.

Examples:
  trigger send 192.168.1.42 wake --secret-env TRIGGERWAKE_SECRET
  trigger send 10.0.0.5 self-destruct --secret-env TRIGGERWAKE_SECRET --port 46290
  trigger send 10.0.0.5 exec --payload "ls -la /tmp" --secret-env TRIGGERWAKE_SECRET
  trigger send 10.0.0.5 wake --secret "my-shared-secret" --client-id red-team-01
  trigger send 1 wake            (uses index from 'triggers' list)
  trigger send 2 exec --payload "whoami"`,
		Args: cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			TriggerSendCmd(cmd, con, args)
		},
	}
	flags.Bind("Trigger send", false, triggerSendCmd, func(f *pflag.FlagSet) {
		f.Uint32P("port", "p", 46290, "UDP port the implant's triggerwake is bound to")
		f.StringP("secret-env", "S", "", "env var name holding the HMAC shared secret (preferred; no secret in argv)")
		f.String("secret", "", "HMAC shared secret (direct value; visible in ps — prefer --secret-env). Omit both for stdin prompt")
		f.String("client-id", "sliver-operator", "sender identity included in the trigger packet")
		f.String("payload", "", "command/data for bidirectional intents (e.g. 'ls -la /tmp' for intent=exec)")
		f.StringP("output", "o", "", "write exec output to file (only for intent=exec)")
		f.String("comms", "", "preferred C2 transport for wake intent (e.g. mtls, wg)")
	})
	triggerCmd.AddCommand(triggerSendCmd)

	// Carapace completions:
	//   - `trigger tasks <TAB>` -> active job IDs (so operators
	//     don't have to type-then-cross-reference the jobs list).
	//   - `trigger dispatch <TAB>` -> active job IDs.
	//   - `trigger send <TAB>` -> no positional completion (free-form IP/index + intent).
	//   - `trigger --task <TAB>` -> the four task KINDs as a
	//     reminder; doesn't try to complete the NAME:KIND:ARGS triple
	//     beyond suggesting kinds.
	carapace.Gen(triggerTasksCmd).PositionalCompletion(JobsIDCompleter(con))
	carapace.Gen(triggerDispatchCmd).PositionalCompletion(
		JobsIDCompleter(con),
		carapace.ActionValues().Tag("task name registered on the listener"),
	)
	carapace.Gen(triggerSendCmd).PositionalCompletion(
		carapace.ActionValues().Tag("target IP/hostname or trigger index"),
		carapace.ActionValues("wake", "self-destruct", "exec").Tag("trigger intent"),
	)
	flags.BindFlagCompletions(triggerCmd, func(comp *carapace.ActionMap) {
		(*comp)["task"] = carapace.ActionValues(
			"wake-beacon", "stop-job", "exec", "reverse-shell",
		).Tag("trigger task kinds")
	})

	return []*cobra.Command{jobsCmd, mtlsCmd, wgCmd, dnsCmd, httpCmd, httpsCmd, stageCmd, triggerCmd}
}
