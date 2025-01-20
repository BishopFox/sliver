package filesystem

import (
	"github.com/bishopfox/sliver/client/command/flags"
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/command/loot"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Commands returns the “ command and its subcommands.
func Commands(con *console.SliverClient) []*cobra.Command {
	mvCmd := &cobra.Command{
		Use:   consts.MvStr,
		Short: "Move or rename a file",
		Long:  help.GetHelpFor([]string{consts.MvStr}),
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			MvCmd(cmd, con, args)
		},
		GroupID: consts.FilesystemHelpGroup,
	}
	flags.Bind("", false, mvCmd, func(f *pflag.FlagSet) {
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})
	carapace.Gen(mvCmd).PositionalCompletion(
		carapace.ActionValues().Usage("path to source file (required)"),
		carapace.ActionValues().Usage("path to dest file (required)"),
	)

	cpCmd := &cobra.Command{
		Use:   consts.CpStr,
		Short: "Copy a file",
		Long:  help.GetHelpFor([]string{consts.CpStr}),
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			CpCmd(cmd, con, args)
		},
		GroupID: consts.FilesystemHelpGroup,
	}
	flags.Bind("", false, cpCmd, func(f *pflag.FlagSet) {
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})
	carapace.Gen(cpCmd).PositionalCompletion(
		carapace.ActionValues().Usage("path to source file (required)"),
		carapace.ActionValues().Usage("path to dest file (required)"),
	)

	lsCmd := &cobra.Command{
		Use:   consts.LsStr,
		Short: "List current directory",
		Long:  help.GetHelpFor([]string{consts.LsStr}),
		Args:  cobra.RangeArgs(0, 1),
		Run: func(cmd *cobra.Command, args []string) {
			LsCmd(cmd, con, args)
		},
		GroupID: consts.FilesystemHelpGroup,
	}
	flags.Bind("", false, lsCmd, func(f *pflag.FlagSet) {
		f.BoolP("reverse", "r", false, "reverse sort order")
		f.BoolP("modified", "m", false, "sort by modified time")
		f.BoolP("size", "s", false, "sort by size")
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})
	carapace.Gen(lsCmd).PositionalCompletion(carapace.ActionValues().Usage("path to enumerate (optional)"))

	rmCmd := &cobra.Command{
		Use:   consts.RmStr,
		Short: "Remove a file or directory",
		Long:  help.GetHelpFor([]string{consts.RmStr}),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			RmCmd(cmd, con, args)
		},
		GroupID: consts.FilesystemHelpGroup,
	}
	flags.Bind("", false, rmCmd, func(f *pflag.FlagSet) {
		f.BoolP("recursive", "r", false, "recursively remove files")
		f.BoolP("force", "F", false, "ignore safety and forcefully remove files")
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})
	carapace.Gen(rmCmd).PositionalCompletion(carapace.ActionValues().Usage("path to the file to remove"))

	mkdirCmd := &cobra.Command{
		Use:   consts.MkdirStr,
		Short: "Make a directory",
		Long:  help.GetHelpFor([]string{consts.MkdirStr}),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			MkdirCmd(cmd, con, args)
		},
		GroupID: consts.FilesystemHelpGroup,
	}
	flags.Bind("", false, mkdirCmd, func(f *pflag.FlagSet) {
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})
	carapace.Gen(mkdirCmd).PositionalCompletion(carapace.ActionValues().Usage("path to the directory to create"))

	cdCmd := &cobra.Command{
		Use:   consts.CdStr,
		Short: "Change directory",
		Long:  help.GetHelpFor([]string{consts.CdStr}),
		Args:  cobra.RangeArgs(0, 1),
		Run: func(cmd *cobra.Command, args []string) {
			CdCmd(cmd, con, args)
		},
		GroupID: consts.FilesystemHelpGroup,
	}
	flags.Bind("", false, cdCmd, func(f *pflag.FlagSet) {
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})
	carapace.Gen(cdCmd).PositionalCompletion(carapace.ActionValues().Usage("path to the directory"))

	pwdCmd := &cobra.Command{
		Use:   consts.PwdStr,
		Short: "Print working directory",
		Long:  help.GetHelpFor([]string{consts.PwdStr}),
		Run: func(cmd *cobra.Command, args []string) {
			PwdCmd(cmd, con, args)
		},
		GroupID: consts.FilesystemHelpGroup,
	}
	flags.Bind("", false, pwdCmd, func(f *pflag.FlagSet) {
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})

	catCmd := &cobra.Command{
		Use:   consts.CatStr,
		Short: "Dump file to stdout",
		Long:  help.GetHelpFor([]string{consts.CatStr}),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			CatCmd(cmd, con, args)
		},
		GroupID: consts.FilesystemHelpGroup,
	}
	flags.Bind("", false, catCmd, func(f *pflag.FlagSet) {
		f.BoolP("colorize-output", "c", false, "colorize output")
		f.BoolP("hex", "x", false, "display as a hex dump")
		f.BoolP("loot", "X", false, "save output as loot")
		f.StringP("name", "n", "", "name to assign loot (optional)")
		f.StringP("type", "T", "", "force a specific loot type (file/cred) if looting (optional)")
		f.StringP("file-type", "F", "", "force a specific file type (binary/text) if looting (optional)")
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})
	carapace.Gen(catCmd).PositionalCompletion(carapace.ActionValues().Usage("path to the file to print"))

	downloadCmd := &cobra.Command{
		Use:   consts.DownloadStr,
		Short: "Download a file",
		Long:  help.GetHelpFor([]string{consts.DownloadStr}),
		Args:  cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			DownloadCmd(cmd, con, args)
		},
		GroupID: consts.FilesystemHelpGroup,
	}
	flags.Bind("", false, downloadCmd, func(f *pflag.FlagSet) {
		f.BoolP("loot", "X", false, "save output as loot")
		f.StringP("type", "T", "", "force a specific loot type (file/cred) if looting")
		f.StringP("file-type", "F", "", "force a specific file type (binary/text) if looting")
		f.StringP("name", "n", "", "name to assign the download if looting")
		f.BoolP("recurse", "r", false, "recursively download all files in a directory")
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})
	flags.BindFlagCompletions(downloadCmd, func(comp *carapace.ActionMap) {
		(*comp)["type"] = loot.LootTypeCompleter(con)
		(*comp)["file-type"] = loot.FileTypeCompleter(con)
	})
	carapace.Gen(downloadCmd).PositionalCompletion(
		carapace.ActionValues().Usage("path to the file or directory to download"),
		carapace.ActionFiles().Usage("local path where the downloaded file will be saved (optional)"),
	)

	uploadCmd := &cobra.Command{
		Use:   consts.UploadStr,
		Short: "Upload a file",
		Long:  help.GetHelpFor([]string{consts.UploadStr}),
		Args:  cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			UploadCmd(cmd, con, args)
		},
		GroupID: consts.FilesystemHelpGroup,
	}
	flags.Bind("", false, uploadCmd, func(f *pflag.FlagSet) {
		f.BoolP("ioc", "i", false, "track uploaded file as an ioc")
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
		f.BoolP("overwrite", "o", false, "overwrite file if it exists")
	})
	carapace.Gen(uploadCmd).PositionalCompletion(
		carapace.ActionFiles().Usage("local path to the file to upload"),
		carapace.ActionValues().Usage("path to the file or directory to upload to (optional)"),
	)

	memfilesCmd := &cobra.Command{
		Use:     consts.MemfilesStr,
		Short:   "List current memfiles",
		Long:    help.GetHelpFor([]string{consts.MemfilesStr}),
		GroupID: consts.FilesystemHelpGroup,
		Run: func(cmd *cobra.Command, args []string) {
			MemfilesListCmd(cmd, con, args)
		},
	}
	flags.Bind("", true, memfilesCmd, func(f *pflag.FlagSet) {
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})

	memfilesAddCmd := &cobra.Command{
		Use:   consts.AddStr,
		Short: "Add a memfile",
		Long:  help.GetHelpFor([]string{consts.MemfilesStr, consts.AddStr}),
		Run: func(cmd *cobra.Command, args []string) {
			MemfilesAddCmd(cmd, con, args)
		},
	}
	memfilesCmd.AddCommand(memfilesAddCmd)

	memfilesRmCmd := &cobra.Command{
		Use:   consts.RmStr,
		Short: "Remove a memfile",
		Long:  help.GetHelpFor([]string{consts.MemfilesStr, consts.RmStr}),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			MemfilesRmCmd(cmd, con, args)
		},
	}
	memfilesCmd.AddCommand(memfilesRmCmd)

	carapace.Gen(memfilesRmCmd).PositionalCompletion(carapace.ActionValues().Usage("memfile file descriptor"))

	mountCmd := &cobra.Command{
		Use:   consts.MountStr,
		Short: "Get information on mounted filesystems",
		Long:  help.GetHelpFor([]string{consts.MountStr}),
		Run: func(cmd *cobra.Command, args []string) {
			MountCmd(cmd, con, args)
		},
		GroupID: consts.FilesystemHelpGroup,
	}
	flags.Bind("", false, mountCmd, func(f *pflag.FlagSet) {
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})

	grepCmd := &cobra.Command{
		Use:   consts.GrepStr,
		Short: "Search for strings that match a regex within a file or directory",
		Long:  help.GetHelpFor([]string{consts.GrepStr}),
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			GrepCmd(cmd, con, args)
		},
		GroupID: consts.FilesystemHelpGroup,
	}
	flags.Bind("", false, grepCmd, func(f *pflag.FlagSet) {
		f.BoolP("colorize-output", "c", false, "colorize output")
		f.BoolP("loot", "X", false, "save output as loot (loot is saved without formatting)")
		f.StringP("name", "n", "", "name to assign loot (optional)")
		f.StringP("type", "T", "", "force a specific loot type (file/cred) if looting (optional)")
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
		f.BoolP("recursive", "r", false, "search recursively")
		f.BoolP("insensitive", "i", false, "case-insensitive search")
		f.Int32P("after", "A", 0, "number of lines to print after a match (ignored if the file is binary)")
		f.Int32P("before", "B", 0, "number of lines to print before a match (ignored if the file is binary)")
		f.Int32P("context", "C", 0, "number of lines to print before and after a match (ignored if the file is binary), equivalent to -A x -B x")
		f.BoolP("exact", "e", false, "match the search term exactly")
	})
	carapace.Gen(grepCmd).PositionalCompletion(
		carapace.ActionValues().Usage("regex to search the file for"),
		carapace.ActionValues().Usage("remote path / file to search in"),
	)

	headCmd := &cobra.Command{
		Use:   consts.HeadStr,
		Short: "Grab the first number of bytes or lines from a file",
		Long:  help.GetHelpFor([]string{consts.HeadStr}),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			/*
				The last argument tells head if the user requested the head or tail of the file
				True means head, false means tail
			*/
			HeadCmd(cmd, con, args, true)
		},
		GroupID: consts.FilesystemHelpGroup,
	}
	flags.Bind("", false, headCmd, func(f *pflag.FlagSet) {
		f.BoolP("colorize-output", "c", false, "colorize output")
		f.BoolP("hex", "x", false, "display as a hex dump")
		f.BoolP("loot", "X", false, "save output as loot")
		f.StringP("name", "n", "", "name to assign loot (optional)")
		f.StringP("type", "T", "", "force a specific loot type (file/cred) if looting (optional)")
		f.StringP("file-type", "F", "", "force a specific file type (binary/text) if looting (optional)")
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
		f.Int64P("bytes", "b", 0, "Grab the first number of bytes from the file")
		f.Int64P("lines", "l", 10, "Grab the first number of lines from the file")
	})
	carapace.Gen(headCmd).PositionalCompletion(carapace.ActionValues().Usage("path to the file to print"))

	tailCmd := &cobra.Command{
		Use:   consts.TailStr,
		Short: "Grab the last number of bytes or lines from a file",
		Long:  help.GetHelpFor([]string{consts.TailStr}),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			/*
				The last argument tells head if the user requested the head or tail of the file
				True means head, false means tail
			*/
			HeadCmd(cmd, con, args, false)
		},
		GroupID: consts.FilesystemHelpGroup,
	}
	flags.Bind("", false, tailCmd, func(f *pflag.FlagSet) {
		f.BoolP("colorize-output", "c", false, "colorize output")
		f.BoolP("hex", "x", false, "display as a hex dump")
		f.BoolP("loot", "X", false, "save output as loot")
		f.StringP("name", "n", "", "name to assign loot (optional)")
		f.StringP("type", "T", "", "force a specific loot type (file/cred) if looting (optional)")
		f.StringP("file-type", "F", "", "force a specific file type (binary/text) if looting (optional)")
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
		f.Int64P("bytes", "b", 0, "Grab the last number of bytes from the file")
		f.Int64P("lines", "l", 10, "Grab the last number of lines from the file")
	})
	carapace.Gen(tailCmd).PositionalCompletion(carapace.ActionValues().Usage("path to the file to print"))

	return []*cobra.Command{
		mvCmd,
		cpCmd,
		lsCmd,
		rmCmd,
		mkdirCmd,
		pwdCmd,
		catCmd,
		cdCmd,
		downloadCmd,
		uploadCmd,
		memfilesCmd,
		mountCmd,
		grepCmd,
		headCmd,
		tailCmd,
	}
}
