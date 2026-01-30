package rest

import (
	"bytes"
	"encoding/json"
	"github.com/RocketChat/Rocket.Chat.Go.SDK/models"
)

type UpdatePermissionsRequest struct {
	Permissions []models.Permission `json:"permissions"`
}

type UpdatePermissionsResponse struct {
	Status
	Permissions []models.Permission `json:"permissions"`
}

// UpdatePermissions updates permissions
//
// https://rocket.chat/docs/developer-guides/rest-api/permissions/update/
func (c *Client) UpdatePermissions(req *UpdatePermissionsRequest) (*UpdatePermissionsResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	response := new(UpdatePermissionsResponse)
	if err := c.Post("permissions.update", bytes.NewBuffer(body), response); err != nil {
		return nil, err
	}
	return response, nil
}
