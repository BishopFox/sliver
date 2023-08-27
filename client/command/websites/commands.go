package websites

import (
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/bishopfox/sliver/client/command/completers"
	"github.com/bishopfox/sliver/client/command/flags"
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
)

// Commands returns the “ command and its subcommands.
func Commands(con *console.SliverClient) []*cobra.Command {
	websitesCmd := &cobra.Command{
		Use:   consts.WebsitesStr,
		Short: "Host static content (used with HTTP C2)",
		Long:  help.GetHelpFor([]string{consts.WebsitesStr}),
		Run: func(cmd *cobra.Command, args []string) {
			WebsitesCmd(cmd, con, args)
		},
		GroupID: consts.NetworkHelpGroup,
	}
	flags.Bind("websites", true, websitesCmd, func(f *pflag.FlagSet) {
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})
	carapace.Gen(websitesCmd).PositionalCompletion(WebsiteNameCompleter(con))

	// General websites management
	websitesRmCmd := &cobra.Command{
		Use:   consts.RmStr,
		Short: "Remove an entire website and all of its contents",
		Long:  help.GetHelpFor([]string{consts.WebsitesStr, consts.RmStr}),
		Run: func(cmd *cobra.Command, args []string) {
			WebsiteRmCmd(cmd, con, args)
		},
	}
	carapace.Gen(websitesRmCmd).PositionalCompletion(WebsiteNameCompleter(con))
	websitesCmd.AddCommand(websitesRmCmd)

	// Content management
	contentCmd := &cobra.Command{
		Use:   consts.ContentStr,
		Short: "Manage wesite contents",
	}
	websitesCmd.AddCommand(contentCmd)

	websitesRmWebContentCmd := &cobra.Command{
		Use:   consts.RmStr,
		Short: "Remove specific content from a website",
		Long:  help.GetHelpFor([]string{consts.WebsitesStr, consts.RmStr}),
		Run: func(cmd *cobra.Command, args []string) {
			WebsitesRmContent(cmd, con, args)
		},
	}
	flags.Bind("websites", false, websitesRmWebContentCmd, func(f *pflag.FlagSet) {
		f.BoolP("recursive", "r", false, "recursively add/rm content")
		f.StringP("website", "w", "", "website name")
		f.StringP("web-path", "p", "", "http path to host file at")
	})
	contentCmd.AddCommand(websitesRmWebContentCmd)
	completers.NewFlagCompsFor(websitesRmWebContentCmd, func(comp *carapace.ActionMap) {
		(*comp)["website"] = WebsiteNameCompleter(con)
	})

	websitesContentCmd := &cobra.Command{
		Use:   consts.AddStr,
		Short: "Add content to a website",
		Long:  help.GetHelpFor([]string{consts.WebsitesStr, consts.AddStr}),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			WebsitesAddContentCmd(cmd, con, args)
		},
	}
	contentCmd.AddCommand(websitesContentCmd)
	flags.Bind("websites", false, websitesContentCmd, func(f *pflag.FlagSet) {
		f.StringP("website", "w", "", "website name")
		f.StringP("content-type", "m", "", "mime content-type (if blank use file ext.)")
		f.StringP("web-path", "p", "/", "http path to host file at")
		f.BoolP("recursive", "r", false, "recursively add/rm content")
	})
	completers.NewFlagCompsFor(websitesContentCmd, func(comp *carapace.ActionMap) {
		(*comp)["website"] = WebsiteNameCompleter(con)
	})
	carapace.Gen(websitesContentCmd).PositionalCompletion(carapace.ActionFiles())

	websitesContentTypeCmd := &cobra.Command{
		Use:   consts.TypeStr,
		Short: "Update a path's content-type",
		Long:  help.GetHelpFor([]string{consts.WebsitesStr, consts.TypeStr}),
		Run: func(cmd *cobra.Command, args []string) {
			WebsitesUpdateContentCmd(cmd, con, args)
		},
	}
	flags.Bind("websites", false, websitesContentTypeCmd, func(f *pflag.FlagSet) {
		f.StringP("website", "w", "", "website name")
		f.StringP("content-type", "m", "", "mime content-type (if blank use file ext.)")
		f.StringP("web-path", "p", "/", "http path to host file at")
	})
	contentCmd.AddCommand(websitesContentTypeCmd)
	completers.NewFlagCompsFor(websitesContentTypeCmd, func(comp *carapace.ActionMap) {
		(*comp)["website"] = WebsiteNameCompleter(con)
	})

	return []*cobra.Command{websitesCmd}
}
