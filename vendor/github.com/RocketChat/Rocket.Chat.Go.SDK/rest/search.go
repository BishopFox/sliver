package rest

import (
	"net/url"

	"github.com/RocketChat/Rocket.Chat.Go.SDK/models"
)

type SearchUsersOrRoomsResponse struct {
	Users []models.SearchUsers `json:"users"`
	Rooms []models.SearchRooms `json:"rooms"`
	Status
}

// Searches for users or rooms that are visible to the user.
// WARNING: It will only return rooms that user didn't join yet.
//
// https://developer.rocket.chat/reference/api/rest-api/endpoints/core-endpoints/miscellaneous-endpoints/spotlight
func (c *Client) SearchUsersOrRooms(query string) (*SearchUsersOrRoomsResponse, error) {
	params := url.Values{
		"query": []string{query},
	}

	response := new(SearchUsersOrRoomsResponse)
	err := c.Get("spotlight", params, response)
	return response, err
}
