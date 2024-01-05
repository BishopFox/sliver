package websites

import (
	"github.com/bishopfox/sliver/client/command/flags"
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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

	websitesRmWebContentCmd := &cobra.Command{
		Use:   consts.RmWebContentStr,
		Short: "Remove specific content from a website",
		Long:  help.GetHelpFor([]string{consts.WebsitesStr, consts.RmWebContentStr}),
		Run: func(cmd *cobra.Command, args []string) {
			WebsitesRmContent(cmd, con, args)
		},
	}
	flags.Bind("websites", false, websitesRmWebContentCmd, func(f *pflag.FlagSet) {
		f.BoolP("recursive", "r", false, "recursively add/rm content")
		f.StringP("website", "w", "", "website name")
		f.StringP("web-path", "p", "", "http path to host file at")
	})
	websitesCmd.AddCommand(websitesRmWebContentCmd)
	flags.BindFlagCompletions(websitesRmWebContentCmd, func(comp *carapace.ActionMap) {
		(*comp)["website"] = WebsiteNameCompleter(con)
	})

	websitesContentCmd := &cobra.Command{
		Use:   consts.AddWebContentStr,
		Short: "Add content to a website",
		Long:  help.GetHelpFor([]string{consts.WebsitesStr, consts.RmWebContentStr}),
		Run: func(cmd *cobra.Command, args []string) {
			WebsitesAddContentCmd(cmd, con, args)
		},
	}
	flags.Bind("websites", false, websitesContentCmd, func(f *pflag.FlagSet) {
		f.StringP("website", "w", "", "website name")
		f.StringP("content-type", "m", "", "mime content-type (if blank use file ext.)")
		f.StringP("web-path", "p", "/", "http path to host file at")
		f.StringP("content", "c", "", "local file path/dir (must use --recursive for dir)")
		f.BoolP("recursive", "r", false, "recursively add/rm content")
	})
	flags.BindFlagCompletions(websitesContentCmd, func(comp *carapace.ActionMap) {
		(*comp)["content"] = carapace.ActionFiles().Tag("content directory/files")
		(*comp)["website"] = WebsiteNameCompleter(con)
	})
	websitesCmd.AddCommand(websitesContentCmd)

	websitesContentTypeCmd := &cobra.Command{
		Use:   consts.WebContentTypeStr,
		Short: "Update a path's content-type",
		Long:  help.GetHelpFor([]string{consts.WebsitesStr, consts.WebContentTypeStr}),
		Run: func(cmd *cobra.Command, args []string) {
			WebsitesUpdateContentCmd(cmd, con, args)
		},
	}
	flags.Bind("websites", false, websitesContentTypeCmd, func(f *pflag.FlagSet) {
		f.StringP("website", "w", "", "website name")
		f.StringP("content-type", "m", "", "mime content-type (if blank use file ext.)")
		f.StringP("web-path", "p", "/", "http path to host file at")
	})
	websitesCmd.AddCommand(websitesContentTypeCmd)
	flags.BindFlagCompletions(websitesContentTypeCmd, func(comp *carapace.ActionMap) {
		(*comp)["website"] = WebsiteNameCompleter(con)
	})

	return []*cobra.Command{websitesCmd}
}
