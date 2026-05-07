package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/RocketChat/Rocket.Chat.Go.SDK/models"
)

type ChannelsResponse struct {
	Status
	models.Pagination
	Channels []models.Channel `json:"channels"`
}

type ChannelResponse struct {
	Status
	Channel models.Channel `json:"channel"`
}

type channelMembersResponse struct {
	Status
	models.Pagination
	Members []*models.User `json:"members"`
}

// CreateChannel is payload for channels.create
//
// https://developer.rocket.chat/reference/api/rest-api/endpoints/core-endpoints/channels-endpoints/create
type createChannel struct {
	ChannelName string   `json:"name"`
	Members     []string `json:"members"`
	ReadOnly    bool     `json:"readOnly"`
}

type inviteChannel struct {
	RoomId  string   `json:"roomId"`
	UserIds []string `json:"userIds"`
}

type joinChannel struct {
	RoomId   string `json:"roomId"`
	JoinCode string `json:"joinCode,omitempty"`
}

// GetPublicChannels returns all channels that can be seen by the logged in user.
//

// https://rocket.chat/docs/developer-guides/rest-api/channels/list
func (c *Client) GetPublicChannels() (*ChannelsResponse, error) {
	response := new(ChannelsResponse)
	if err := c.Get("channels.list", nil, response); err != nil {
		return nil, err
	}

	return response, nil
}

// GetJoinedChannels returns all channels that the user has joined.
//
// https://rocket.chat/docs/developer-guides/rest-api/channels/list-joined
func (c *Client) GetJoinedChannels(params url.Values) (*ChannelsResponse, error) {
	response := new(ChannelsResponse)
	if err := c.Get("channels.list.joined", params, response); err != nil {
		return nil, err
	}

	return response, nil
}

// LeaveChannel leaves a channel. The id of the channel has to be not nil.
//
// https://rocket.chat/docs/developer-guides/rest-api/channels/leave
func (c *Client) LeaveChannel(channel *models.Channel) error {
	var body = fmt.Sprintf(`{ "roomId": "%s"}`, channel.ID)
	return c.Post("channels.leave", bytes.NewBufferString(body), new(ChannelResponse))
}

// GetChannelInfo get information about a channel. That might be useful to update the usernames.
//
// https://rocket.chat/docs/developer-guides/rest-api/channels/info
func (c *Client) GetChannelInfo(channel *models.Channel) (*models.Channel, error) {
	response := new(ChannelResponse)
	switch {
	case channel.Name != "" && channel.ID == "":
		if err := c.Get("channels.info", url.Values{"roomName": []string{channel.Name}}, response); err != nil {
			return nil, err
		}
	default:
		if err := c.Get("channels.info", url.Values{"roomId": []string{channel.ID}}, response); err != nil {
			return nil, err
		}
	}

	return &response.Channel, nil
}

// CreateChannel creates a new public channel, optionally including specified users.
// The channel creator is always included.
//
// https://developer.rocket.chat/reference/api/rest-api/endpoints/core-endpoints/channels-endpoints/create
func (c *Client) CreateChannel(channelName string, users []*models.User, readOnly bool) (*models.Channel, error) {
	var usernames []string
	for _, user := range users {
		usernames = append(usernames, user.Name)
	}
	createChannel := &createChannel{ChannelName: channelName, Members: usernames, ReadOnly: readOnly}
	body, err := json.Marshal(createChannel)
	if err != nil {
		return nil, err
	}
	response := new(ChannelResponse)
	err = c.Post("channels.create", bytes.NewBuffer(body), response)

	return &response.Channel, err
}

// InviteChannel adds users to the channel.
//
// https://developer.rocket.chat/reference/api/rest-api/endpoints/core-endpoints/channels-endpoints/invite
func (c *Client) InviteChannel(channel *models.Channel, users []*models.User) (*models.Channel, error) {
	var ids []string
	for _, user := range users {
		ids = append(ids, user.ID)
	}
	invite := &inviteChannel{RoomId: channel.ID, UserIds: ids}
	body, err := json.Marshal(invite)
	if err != nil {
		return nil, err
	}
	response := new(ChannelResponse)
	err = c.Post("channels.invite", bytes.NewBuffer(body), response)

	return &response.Channel, err
}

// JoinChannel joins yourself to the channel.
// joinCode isn't needed if the user has the permission "join-without-join-code"
// An empty string should be passed if not required
//
// https://developer.rocket.chat/reference/api/rest-api/endpoints/core-endpoints/channels-endpoints/join
func (c *Client) JoinChannel(channel *models.Channel, joinCode string) (*models.Channel, error) {
	join := &joinChannel{RoomId: channel.ID}
	body, err := json.Marshal(join)
	if err != nil {
		return nil, err
	}
	response := new(ChannelResponse)
	err = c.Post("channels.join", bytes.NewBuffer(body), response)

	return &response.Channel, err
}

// GetChannelMembers lists all channel users.
//
// https://developer.rocket.chat/reference/api/rest-api/endpoints/core-endpoints/channels-endpoints/members
func (c *Client) GetChannelMembers(channel *models.Channel) ([]*models.User, error) {
	response := new(channelMembersResponse)
	switch {
	case channel.Name != "" && channel.ID == "":
		if err := c.Get("channels.members", url.Values{"roomName": []string{channel.Name}}, response); err != nil {
			return nil, err
		}
	default:
		if err := c.Get("channels.members", url.Values{"roomId": []string{channel.ID}}, response); err != nil {
			return nil, err
		}
	}

	return response.Members, nil
}

// Get messages from a channel. The channel id has to be not nil. Optionally a
// count can be specified to limit the size of the returned messages.
//
// https://rocket.chat/docs/developer-guides/rest-api/channels/history
func (c *Client) ChannelHistory(channel *models.Channel, inclusive bool, fromDate time.Time, page *models.Pagination) ([]models.Message, error) {
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
	if err := c.Get("channels.history", params, response); err != nil {
		return nil, err
	}

	return response.Messages, nil
}
