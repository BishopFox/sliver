package rest

import (
	"net/url"

	"github.com/RocketChat/Rocket.Chat.Go.SDK/models"
)

type GroupsResponse struct {
	Status
	models.Pagination
	Groups []models.Channel `json:"groups"`
}

type GroupResponse struct {
	Status
	Group models.Channel `json:"group"`
}

// GetPrivateGroups returns all channels that can be seen by the logged in user.
//
// https://rocket.chat/docs/developer-guides/rest-api/groups/list
func (c *Client) GetPrivateGroups() (*GroupsResponse, error) {
	response := new(GroupsResponse)
	if err := c.Get("groups.list", nil, response); err != nil {
		return nil, err
	}

	return response, nil
}

// GetGroupInfo get information about a group. That might be useful to update the usernames.
//
// https://rocket.chat/docs/developer-guides/rest-api/groups/info
func (c *Client) GetGroupInfo(channel *models.Channel) (*models.Channel, error) {
	response := new(GroupResponse)
	switch {
	case channel.Name != "" && channel.ID == "":
		if err := c.Get("groups.info", url.Values{"roomName": []string{channel.Name}}, response); err != nil {
			return nil, err
		}
	default:
		if err := c.Get("groups.info", url.Values{"roomId": []string{channel.ID}}, response); err != nil {
			return nil, err
		}
	}

	return &response.Group, nil
}
