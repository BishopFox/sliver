package console

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
	"encoding/json"
	"fmt"
	"sort"

	"github.com/AlecAivazis/survey/v2"
	"github.com/maxlandon/gonsole"
	"github.com/maxlandon/readline"
	"google.golang.org/grpc"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
)

var (
	// serverConfig - The prompt and the console both need access to it at times.
	serverConfig *assets.ClientConfig
)

func selectConfig() *assets.ClientConfig {

	configs := assets.GetConfigs()

	if len(configs) == 0 {
		return nil
	}

	if len(configs) == 1 {
		for _, config := range configs {
			return config
		}
	}

	answer := struct{ Config string }{}
	qs := getPromptForConfigs(configs)
	err := survey.Ask(qs, &answer)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}

	// Keep a reference of the config we will use
	serverConfig = configs[answer.Config]

	return configs[answer.Config]
}

func getPromptForConfigs(configs map[string]*assets.ClientConfig) []*survey.Question {

	keys := []string{}
	for k := range configs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	return []*survey.Question{
		{
			Name: "config",
			Prompt: &survey.Select{
				Message: "Select a server:",
				Options: keys,
				Default: keys[0],
			},
		},
	}
}

// loadConsoleConfig - Once the client is connected, it receives a console
// configuration from the server, according to the user profile.
// In case of errors, it loads the builtin console configuration below.
func loadConsoleConfig(rpc rpcpb.SliverRPCClient) (config *gonsole.Config, err error) {

	req := &clientpb.GetConsoleConfigReq{}
	res, err := rpc.LoadConsoleConfig(context.Background(), req, grpc.EmptyCallOption{})
	if err != nil {
		config = loadDefaultConsoleConfig()
		return config, fmt.Errorf("RPC Error: %s", err.Error())
	}
	if res.Response.Err != "" {
		config = loadDefaultConsoleConfig()
		return config, fmt.Errorf("%s", res.Response.Err)
	}

	// The ser has sent us a JSON struct
	config = &gonsole.Config{}
	err = json.Unmarshal(res.Config, config)
	if err != nil {
		fmt.Printf(Warn+"Error unmarshaling config: %s\n", err.Error())
		return
	}

	return
}

// loadDefaultConsoleConfig - When the user has no saved config on the server
// (or the server has not saved itself) we load this default console configuration.
func loadDefaultConsoleConfig() (config *gonsole.Config) {

	// Get a config object with defaults, and initialized maps.
	config = gonsole.NewDefaultConfig()
	config.InputMode = gonsole.InputEmacs
	config.MaxTabCompleterRows = 37 // just half the height of my 13.3" laptop...

	// Make little adjustements to default server prompt, depending on server/client
	var ps string
	ps += readline.RESET
	if serverConfig.LHost == "" {
		ps += "{bddg}{y} server {fw}@{local_ip} {reset}"
	} else {
		ps += "{bddg}@{ly}{server_ip}{reset}"
	}
	// Current working directory
	ps += " {dim}in {reset}{bold}{ly}{cwd}"
	ps += readline.RESET

	// Server context prompt
	config.Prompts[constants.ServerMenu] = &gonsole.PromptConfig{
		Left:            ps,
		Right:           "{dim}[{reset}{y}{jobs}{fw} jobs, {b}{sessions}{fw} sessions{dim}]",
		MultilinePrompt: " > ",
		Multiline:       true,
		NewlineAfter:    true,
		NewlineBefore:   true,
	}

	// Sliver context prompt
	config.Prompts[constants.SliverMenu] = &gonsole.PromptConfig{
		Left:            "{bddg} {fr}{session_name} {reset}{bold} {user}{dim}@{reset}{bold}{host}{reset}{dim} in{reset} {bold}{b}{wd}",
		Right:           "{dim}[{reset}{y}{bold}{platform}{reset}, {bold}{g}{address}{fw}{reset}{dim}]",
		MultilinePrompt: " > ",
		Multiline:       true,
		NewlineAfter:    true,
		NewlineBefore:   true,
	}

	// Special coloring for session environment variables
	config.Highlighting["%"] = readline.RED

	return
}
