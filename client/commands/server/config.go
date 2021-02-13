package server

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
	"strings"

	"github.com/evilsocket/islazy/tui"
	"google.golang.org/grpc"

	cctx "github.com/bishopfox/sliver/client/context"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/clientpb"
)

// Config - Manage console configuration. Prints current by default
type Config struct{}

// Execute - Manage console configuration. Prints current by default
func (c *Config) Execute(args []string) (err error) {
	conf := cctx.Config

	fmt.Println(tui.Bold(tui.Blue(" Console configuration\n")))

	fmt.Println(tui.Yellow("Prompts"))
	pad := fmt.Sprintf("%-15s", "server (right)")
	fmt.Printf(" "+pad+" :  %s\n", conf.ServerPrompt.Right)
	pad = fmt.Sprintf("%-15s", "server (left)")
	fmt.Printf(" "+pad+" :  %s\n", conf.ServerPrompt.Left)
	pad = fmt.Sprintf("%-15s", "sliver (right)")
	fmt.Printf(" "+pad+" :  %s\n", conf.SliverPrompt.Right)
	pad = fmt.Sprintf("%-15s", "sliver (left)")
	fmt.Printf(" "+pad+" :  %s\n", conf.SliverPrompt.Left)

	fmt.Println(tui.Yellow("\nOthers"))
	pad = fmt.Sprintf("%-15s", "console hints")
	fmt.Printf(" "+pad+" :  %t\n", conf.Hints)

	var input string
	if conf.Vim {
		input = tui.Bold("Vim")
	} else {
		input = tui.Bold("Emacs")
	}
	pad = fmt.Sprintf("%-15s", "input mode")
	fmt.Printf(" "+pad+" :  %s\n", input)

	fmt.Println()

	// Check if this config has been saved (they should be identical)
	req := &clientpb.GetConsoleConfigReq{}
	res, err := transport.RPC.LoadConsoleConfig(context.Background(), req, grpc.EmptyCallOption{})
	if err != nil {
		fmt.Printf(util.Warn + "Could not check if current config is saved\n")
		return
	}
	// An error thrown in the request means we did not find the configuration.
	if res.Response.Err != "" {
		fmt.Printf(util.Warn + "Current configuration is not saved, type 'config save' to do so.\n")
		return
	}

	cf := res.Config
	if (cf.ServerPromptRight == conf.ServerPrompt.Right) && (cf.ServerPromptLeft == conf.ServerPrompt.Left) &&
		(cf.SliverPromptRight == conf.SliverPrompt.Right) && (cf.SliverPromptLeft == conf.SliverPrompt.Left) &&
		(cf.Vim == conf.Vim) && (cf.Hints == conf.Hints) {
		fmt.Printf(util.Info + "Current configuration is saved\n")
	} else {
		fmt.Printf(util.Warn + "Current configuration is not saved, type 'config save' to do so.\n")
	}

	return
}

// SaveConfig - Save the current console configuration.
type SaveConfig struct{}

// Execute - Save the current console configuration.
func (c *SaveConfig) Execute(args []string) (err error) {

	currentConf := cctx.Config.ToProtobuf()
	req := &clientpb.SaveConsoleConfigReq{
		Config: currentConf,
	}
	res, err := transport.RPC.SaveUserConsoleConfig(context.Background(), req, grpc.EmptyCallOption{})
	if err != nil {
		fmt.Printf(util.RPCError+"%v\n", err)
		return
	}

	if res.Response.Err != "" {
		fmt.Printf(util.Error+"Error saving config: %s\n", res.Response.Err)
	} else {
		fmt.Printf(util.Info + "Saved console config\n")
	}
	return
}

// PromptServer - Modify the right-side prompt
type PromptServer struct {
	Positional struct {
		Prompt string `description:"prompt string" required:"yes"`
	} `positional-args:"yes" required:"yes"`
	Options struct {
		Right bool `long:"right" short:"r" description:"apply changes to the right-side prompt"`
		Left  bool `long:"left" short:"l" description:"apply changes to the left-side prompt"`
	} `group:"prompt options"`
}

// Execute - Modify the right-side prompt
func (c *PromptServer) Execute(args []string) (err error) {
	if len(args) > 0 {
		fmt.Printf(util.Warn+"Detected undesired remaining arguments: %s\n", tui.Bold(strings.Join(args, " ")))
		fmt.Printf("    Please use \\ dashes for each space in prompt string (input readline doesn't detect them)\n")
		fmt.Printf(tui.Yellow("    The current value has therefore not been saved.\n"))
		return
	}

	// Which prompt side did we set
	var side string
	var prompt *string
	if c.Options.Right {
		side = "(right)"
		prompt = &cctx.Config.ServerPrompt.Right
	}
	if c.Options.Left {
		side = "(left)"
		prompt = &cctx.Config.ServerPrompt.Left
	}
	if !c.Options.Left && !c.Options.Right {
		side = "(left)"
		prompt = &cctx.Config.ServerPrompt.Left
	}

	if c.Positional.Prompt == "\"\"" || c.Positional.Prompt == "''" {
		*prompt = "" // Set the prompt string
		fmt.Printf(util.Info + "Detected empty prompt string: deactivating the corresponding prompt.\n")
		return
	}

	*prompt = c.Positional.Prompt // Set the prompt string
	fmt.Printf(util.Info+"Server prompt %s : %s\n", side, tui.Bold(c.Positional.Prompt))

	return
}

// PromptSliver - Modify the left-side prompt
type PromptSliver struct {
	Positional struct {
		Prompt string `description:"prompt string" required:"yes"`
	} `positional-args:"yes" required:"yes"`
	Options struct {
		Right bool `long:"right" short:"r" description:"apply changes to the right-side prompt"`
		Left  bool `long:"left" short:"l" description:"apply changes to the left-side prompt"`
	} `group:"prompt options"`
}

// Execute - Modify the right-side prompt
func (c *PromptSliver) Execute(args []string) (err error) {
	if len(args) > 0 {
		fmt.Printf(util.Warn+"Detected undesired remaining arguments: %s\n", tui.Bold(strings.Join(args, " ")))
		fmt.Printf("    Please use \\ dashes for each space in prompt string (input readline doesn't detect them)\n")
		fmt.Printf(tui.Yellow("    The current value has therefore not been saved.\n"))
		return
	}

	// Which prompt side did we set
	var side string
	var prompt *string
	if c.Options.Right {
		side = "(right)"
		prompt = &cctx.Config.SliverPrompt.Right
	}
	if c.Options.Left {
		side = "(left)"
		prompt = &cctx.Config.SliverPrompt.Left
	}
	if !c.Options.Left && !c.Options.Right {
		side = "(left)"
		prompt = &cctx.Config.SliverPrompt.Left
	}

	if c.Positional.Prompt == "" {
		*prompt = "" // Set the prompt string
		fmt.Printf(util.Info + "Detected empty prompt string: deactivating the corresponding prompt.\n")
		return
	}

	*prompt = c.Positional.Prompt // Set the prompt string
	fmt.Printf(util.Info+"Sliver prompt %s : %s\n", side, tui.Bold(c.Positional.Prompt))

	return
}

// Hints - Turn the hints on/off
type Hints struct {
	Positional struct {
		Display string `description:"show / hide command hints" required:"yes"`
	} `positional-args:"yes" required:"yes"`
}

// Execute - Turn the hints on/off
func (c *Hints) Execute(args []string) (err error) {

	switch c.Positional.Display {
	case "show", "on":
		cctx.Config.Hints = true
		fmt.Printf(util.Info+"Console hints: %s\n", tui.Yellow(c.Positional.Display))
	case "hide", "off":
		cctx.Config.Hints = false
		fmt.Printf(util.Info+"Console hints: %s\n", tui.Yellow(c.Positional.Display))
	default:
		fmt.Printf(util.Error+"Invalid argument: %s (must be 'hide' or 'show')\n", c.Positional.Display)
		return nil
	}
	return
}

// Vim - Set the console input mode to Vim
type Vim struct{}

// Execute - Set the console input mode to Vim
func (c *Vim) Execute(args []string) (err error) {
	cctx.Config.Vim = true
	fmt.Printf(util.Info+"Console input mode: %s\n", tui.Yellow("Vim"))
	return
}

// Emacs - Set the console input mode to Emacs
type Emacs struct{}

// Execute - Set the console input mode to Emacs
func (c *Emacs) Execute(args []string) (err error) {
	cctx.Config.Vim = false
	fmt.Printf(util.Info+"Console input mode: %s\n", tui.Yellow("Emacs"))
	return
}
