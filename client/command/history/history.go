package history

/*
   Sliver Implant Framework
   Copyright (C) 2019  Bishop Fox

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
	"strconv"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/bishopfox/sliver/client/command/flags"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
)

// Commands returns all commands related to implant history.
func Commands(con *console.SliverClient) []*cobra.Command {
	historyCmd := &cobra.Command{
		Use:     "history",
		Short:   "Print the implant command history",
		GroupID: constants.SliverCoreHelpGroup,
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, beac := con.ActiveTarget.Get()
			if sess == nil && beac == nil {
				return nil
			}

			user, _ := cmd.Flags().GetBool("user")
			showTime, _ := cmd.Flags().GetBool("time")
			clientUser := con.Teamclient.Config().User

			req := &clientpb.HistoryRequest{
				UserOnly: user,
			}

			if sess != nil {
				req.ImplantID = sess.ID
				req.ImplantName = sess.Name
			} else if beac != nil {
				req.ImplantID = beac.ID
				req.ImplantName = beac.Name
			}

			history, err := con.Rpc.GetImplantHistory(context.Background(), req)
			if err != nil {
				return con.UnwrapServerErr(err)
			}

			commands := history.GetCommands()

			for i := len(commands) - 1; i >= 0; i-- {
				command := commands[i]

				if user && command.GetUser() != clientUser {
					continue
				}

				preLine := color.HiBlackString("%-3s", strconv.Itoa(i))

				if !user {
					preLine += console.Orange + fmt.Sprintf("%*s\t", 5, command.User) + console.Normal
				}
				if showTime {
					execAt := time.Unix(command.GetExecutedAt(), 0).Format(time.Stamp)
					preLine += console.Blue + execAt + console.Normal + "\t"
				}

				fmt.Println(preLine + command.Block)
			}

			return nil
		},
	}

	flags.Bind("history", false, historyCmd, func(f *pflag.FlagSet) {
		f.BoolP("user", "u", false, "Only print implant commands executed by user")
		f.BoolP("time", "t", false, "Print the exec time before the command line (tab separated)")
	})

	return []*cobra.Command{historyCmd}
}
