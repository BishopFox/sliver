package rest

import (
	"net/url"
	"strconv"
	"time"

	"github.com/RocketChat/Rocket.Chat.Go.SDK/models"
)

type GroupMembersResponse struct {
	Status
	models.Pagination
	Members []models.User `json:"members"`
}

// Get messages from a dm. The channel id has to be not nil. Optionally a
// count can be specified to limit the size of the returned messages.
//
// https://rocket.chat/docs/developer-guides/rest-api/channels/history
func (c *Client) GroupHistory(channel *models.Channel, inclusive bool, fromDate time.Time, page *models.Pagination) ([]models.Message, error) {
	params := url.Values{
		"roomId": []string{channel.ID},
	}

	if page != nil {
		params.Add("count", strconv.Itoa(page.Count))
	}

	latestTime := fromDate.Format(time.RFC3339)
	params.Add("latest", latestTime)
	params.Add("inclusive", "true")

	response := new(MessagesResponse)
	if err := c.Get("groups.history", params, response); err != nil {
		return nil, err
	}

	return response.Messages, nil
}

// GetChannelMembers lists all channel users.
//
// https://developer.rocket.chat/reference/api/rest-api/endpoints/core-endpoints/channels-endpoints/members
func (c *Client) GetGroupMembers(channel *models.Channel) ([]models.User, error) {
	response := new(GroupMembersResponse)
	switch {
	case channel.Name != "" && channel.ID == "":
		if err := c.Get("groups.members", url.Values{"roomName": []string{channel.Name}}, response); err != nil {
			return nil, err
		}
	default:
		if err := c.Get("groups.members", url.Values{"roomId": []string{channel.ID}}, response); err != nil {
			return nil, err
		}
	}

	return response.Members, nil
}
