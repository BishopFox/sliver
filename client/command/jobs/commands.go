package jobs

import (
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
		f.IntP("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})
	flags.Bind("jobs", false, jobsCmd, func(f *pflag.FlagSet) {
		f.Int32P("kill", "k", -1, "kill a background job")
		f.BoolP("kill-all", "K", false, "kill all jobs")
		f.IntP("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
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
		f.BoolP("persistent", "p", false, "make persistent across restarts")
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
		f.BoolP("persistent", "p", false, "make persistent across restarts")
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
		f.BoolP("disable-otp", "D", false, "disable otp authentication")
		f.BoolP("persistent", "p", false, "make persistent across restarts")
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
		f.BoolP("persistent", "p", false, "make persistent across restarts")
	})

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

		f.BoolP("persistent", "p", false, "make persistent across restarts")
	})
	flags.BindFlagCompletions(httpsCmd, func(comp *carapace.ActionMap) {
		(*comp)["cert"] = carapace.ActionFiles().Tag("certificate file")
		(*comp)["key"] = carapace.ActionFiles().Tag("key file")
	})

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

	return []*cobra.Command{jobsCmd, mtlsCmd, wgCmd, httpCmd, httpsCmd, stageCmd}
}
