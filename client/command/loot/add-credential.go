package loot

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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

	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/desertbit/grumble"
)

// LootAddCredentialCmd - Add a credential type loot
func LootAddCredentialCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	prompt := &survey.Select{
		Message: "Choose a credential type:",
		Options: []string{
			clientpb.CredentialType_API_KEY.String(),
			clientpb.CredentialType_USER_PASSWORD.String(),
		},
	}
	credType := ""
	survey.AskOne(prompt, &credType, survey.WithValidator(survey.Required))
	name := ctx.Flags.String("name")
	if name == "" {
		namePrompt := &survey.Input{Message: "Credential Name: "}
		con.Println()
		survey.AskOne(namePrompt, &name)
		con.Println()
	}

	loot := &clientpb.Loot{
		Type:       clientpb.LootType_LOOT_CREDENTIAL,
		Name:       name,
		Credential: &clientpb.Credential{},
	}

	switch credType {
	case clientpb.CredentialType_USER_PASSWORD.String():
		loot.CredentialType = clientpb.CredentialType_USER_PASSWORD
		for loot.Credential.User == "" {
			usernamePrompt := &survey.Input{Message: "Username: "}
			survey.AskOne(usernamePrompt, &loot.Credential.User)
			if loot.Credential.User == "" {
				con.Println("Username is required")
			}
		}
		for loot.Credential.Password == "" {
			passwordPrompt := &survey.Input{Message: "Password: "}
			survey.AskOne(passwordPrompt, &loot.Credential.Password)
			if loot.Credential.Password == "" {
				con.Println("Password is required")
			}
		}
	case clientpb.CredentialType_API_KEY.String():
		loot.CredentialType = clientpb.CredentialType_API_KEY
		usernamePrompt := &survey.Input{Message: "API Key: "}
		survey.AskOne(usernamePrompt, &loot.Credential.APIKey)
	}

	loot, err := con.Rpc.LootAdd(context.Background(), loot)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	con.PrintInfof("Successfully added loot to server (%s)\n", loot.LootID)
}
