package rest

import (
	"bytes"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"time"

	"github.com/RocketChat/Rocket.Chat.Go.SDK/models"
)

type DirectMessageResponse struct {
	Status
	Room Room `json:"room"`
}

type Room struct {
	ID        string   `json:"_id"`
	Rid       string   `json:"rid"`
	Type      string   `json:"t"`
	Usernames []string `json:"usernames"`
}

// Creates a DirectMessage
//
// https://developer.rocket.chat/api/rest-api/methods/im/create
func (c *Client) CreateDirectMessage(username string) (*Room, error) {
	body := fmt.Sprintf(`{ "username": "%s" }`, username)
	resp := new(DirectMessageResponse)

	if err := c.Post("im.create", bytes.NewBufferString(body), resp); err != nil {
		return nil, err
	}

	log.Println(resp)

	return &resp.Room, nil
}

// Get messages from a dm. The channel id has to be not nil. Optionally a
// count can be specified to limit the size of the returned messages.
//
// https://rocket.chat/docs/developer-guides/rest-api/channels/history
func (c *Client) DMHistory(channel *models.Channel, inclusive bool, fromDate time.Time, page *models.Pagination) ([]models.Message, error) {
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
	if err := c.Get("dm.history", params, response); err != nil {
		return nil, err
	}

	return response.Messages, nil
}
