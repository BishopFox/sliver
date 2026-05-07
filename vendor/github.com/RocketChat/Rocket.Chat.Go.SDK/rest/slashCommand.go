package rest

import (
	"bytes"
	"fmt"
	"net/url"

	"github.com/RocketChat/Rocket.Chat.Go.SDK/models"
)


type SlashCommandsResponse struct {
	Status
	Commands []models.SlashCommand `json:"commands"`
}

type ExecuteSlashCommandResponse struct {
	Status
}

// GetSlashCommandsList
// Slash Commands available in the Rocket.Chat server.
// It supports the offset, count and Sort Query Parameters along with just the Fields and Query Parameters.
//
// https://developer.rocket.chat/reference/api/rest-api/endpoints/core-endpoints/commands-endpoints/list
func (c *Client) GetSlashCommandsList(params url.Values) ([]models.SlashCommand, error) {
	response := new(SlashCommandsResponse)
	if err := c.Get("commands.list", params, response); err != nil {
		return nil, err
	}
	return response.Commands, nil
}

// ExecuteSlashCommand
// Execute a slash command in a room in the Rocket.Chat server
// command and roomId are required params is optional it depends upon command
//
// https://developer.rocket.chat/reference/api/rest-api/endpoints/core-endpoints/commands-endpoints/execute-a-slash-command
func (c *Client) ExecuteSlashCommand(channel *models.ChannelSubscription, command string, params string) (ExecuteSlashCommandResponse, error) {
	response := new(ExecuteSlashCommandResponse)
	var body = fmt.Sprintf(`{ "command": "%s", "roomId": "%s", "params": "%s"}`, command, string(channel.RoomId), params)
	if err := c.Post("commands.run", bytes.NewBufferString(body), response); err != nil {
		return *response, err
	}
	return *response, nil
}
