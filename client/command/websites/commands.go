package websites

import (
	"github.com/bishopfox/sliver/client/command/completers"
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

	websitesListCmd := &cobra.Command{
		Use:   consts.ListStr,
		Short: "List configured websites",
		Long:  help.GetHelpFor([]string{consts.WebsitesStr, consts.ListStr}),
		Run: func(cmd *cobra.Command, args []string) {
			ListWebsites(cmd, con, args)
		},
	}
	websitesCmd.AddCommand(websitesListCmd)

	websitesShowCmd := &cobra.Command{
		Use:   "show [name]",
		Short: "Show contents of a website",
		Long:  help.GetHelpFor([]string{consts.WebsitesStr}),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ListWebsiteContent(args[0], con)
		},
	}
	carapace.Gen(websitesShowCmd).PositionalCompletion(WebsiteNameCompleter(con))
	websitesShowCmd.ValidArgsFunction = websiteNameValidArgs(con)
	websitesCmd.AddCommand(websitesShowCmd)

	// websites upload - everything about content the website receives via PUT/POST
	websitesUploadCmd := &cobra.Command{
		Use:   consts.UploadStr,
		Short: "Manage content uploaded to a website (PUT/POST requests)",
		Long:  help.GetHelpFor([]string{consts.WebsitesStr, consts.UploadStr}),
	}
	websitesCmd.AddCommand(websitesUploadCmd)

	websitesUploadEnableCmd := &cobra.Command{
		Use:   consts.EnableStr + " [name]",
		Short: "Allow the website to accept uploads, creating the website if needed",
		Long:  help.GetHelpFor([]string{consts.WebsitesStr, consts.UploadStr, consts.EnableStr}),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			WebsitesUploadEnableCmd(cmd, con, args)
		},
	}
	carapace.Gen(websitesUploadEnableCmd).PositionalCompletion(WebsiteNameCompleter(con))
	websitesUploadEnableCmd.ValidArgsFunction = websiteNameValidArgs(con)
	websitesUploadCmd.AddCommand(websitesUploadEnableCmd)

	websitesUploadDisableCmd := &cobra.Command{
		Use:   consts.DisableStr + " [name]",
		Short: "Stop the website from accepting uploads",
		Long:  help.GetHelpFor([]string{consts.WebsitesStr, consts.UploadStr, consts.DisableStr}),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			WebsitesUploadDisableCmd(cmd, con, args)
		},
	}
	carapace.Gen(websitesUploadDisableCmd).PositionalCompletion(WebsiteNameCompleter(con))
	websitesUploadDisableCmd.ValidArgsFunction = websiteNameValidArgs(con)
	websitesUploadCmd.AddCommand(websitesUploadDisableCmd)

	websitesUploadListCmd := &cobra.Command{
		Use:   consts.ListStr + " [name]",
		Short: "List content uploaded to a website",
		Long:  help.GetHelpFor([]string{consts.WebsitesStr, consts.UploadStr, consts.ListStr}),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			WebsitesUploadListCmd(cmd, con, args)
		},
	}
	carapace.Gen(websitesUploadListCmd).PositionalCompletion(WebsiteNameCompleter(con))
	websitesUploadListCmd.ValidArgsFunction = websiteNameValidArgs(con)
	websitesUploadCmd.AddCommand(websitesUploadListCmd)

	websitesUploadDownloadCmd := &cobra.Command{
		Use:   consts.DownloadStr + " [id]",
		Short: "Download uploaded content, writes the raw file and a .json with its metadata",
		Long:  help.GetHelpFor([]string{consts.WebsitesStr, consts.UploadStr, consts.DownloadStr}),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			WebsitesUploadDownloadCmd(cmd, con, args)
		},
	}
	flags.Bind("websites", false, websitesUploadDownloadCmd, func(f *pflag.FlagSet) {
		f.StringP("save", "s", "", "directory to save the files to (default: current directory)")
	})
	flags.BindFlagCompletions(websitesUploadDownloadCmd, func(comp *carapace.ActionMap) {
		(*comp)["save"] = carapace.ActionDirectories().Tag("save directory")
	})
	completers.RegisterLocalFilePathFlagCompletion(websitesUploadDownloadCmd, "save")
	carapace.Gen(websitesUploadDownloadCmd).PositionalCompletion(UploadedContentCompleter(con))
	websitesUploadCmd.AddCommand(websitesUploadDownloadCmd)

	websitesUploadRmCmd := &cobra.Command{
		Use:   consts.RmStr + " [name]",
		Short: "Remove all content uploaded to a website",
		Long:  help.GetHelpFor([]string{consts.WebsitesStr, consts.UploadStr, consts.RmStr}),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			WebsitesUploadRmCmd(cmd, con, args)
		},
	}
	carapace.Gen(websitesUploadRmCmd).PositionalCompletion(WebsiteNameCompleter(con))
	websitesUploadRmCmd.ValidArgsFunction = websiteNameValidArgs(con)
	websitesUploadCmd.AddCommand(websitesUploadRmCmd)

	websitesUploadRmContentCmd := &cobra.Command{
		Use:   consts.RmWebContentStr,
		Short: "Remove specific content uploaded to a website",
		Long:  help.GetHelpFor([]string{consts.WebsitesStr, consts.UploadStr, consts.RmWebContentStr}),
		Run: func(cmd *cobra.Command, args []string) {
			WebsitesUploadRmContentCmd(cmd, con, args)
		},
	}
	flags.Bind("websites", false, websitesUploadRmContentCmd, func(f *pflag.FlagSet) {
		f.BoolP("recursive", "r", false, "recursively add/rm content")
		f.StringP("website", "w", "", "website name")
		f.StringP("web-path", "p", "", "http path the content was uploaded to")
		f.StringP("id", "i", "", "id of a single uploaded content item")
	})
	flags.BindFlagCompletions(websitesUploadRmContentCmd, func(comp *carapace.ActionMap) {
		(*comp)["website"] = WebsiteNameCompleter(con)
		(*comp)["id"] = UploadedContentCompleter(con)
	})
	websitesUploadCmd.AddCommand(websitesUploadRmContentCmd)

	websitesRmCmd := &cobra.Command{
		Use:   consts.RmStr + " [name]",
		Short: "Remove an entire website and all of its contents",
		Long:  help.GetHelpFor([]string{consts.WebsitesStr, consts.RmStr}),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			WebsiteRmCmd(cmd, con, args)
		},
	}
	carapace.Gen(websitesRmCmd).PositionalCompletion(WebsiteNameCompleter(con))
	websitesRmCmd.ValidArgsFunction = websiteNameValidArgs(con)
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
	completers.RegisterLocalFilePathFlagCompletion(websitesContentCmd, "content")
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
