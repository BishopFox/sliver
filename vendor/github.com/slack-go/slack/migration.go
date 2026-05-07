package slack

import (
	"context"
	"net/url"
)

type migrationExchangeResponseFull struct {
	TeamID         string            `json:"team_id"`
	ToOld          bool              `json:"to_old"`
	EnterpriseID   string            `json:"enterprise_id"`
	UserIDMap      map[string]string `json:"user_id_map"`
	InvalidUserIDs []string          `json:"invalid_user_ids"`
	SlackResponse
}

// MigrationExchange for Enterprise Grid workspaces, map local user IDs to global user IDs
func (api *Client) MigrationExchange(ctx context.Context, teamID string, toOld bool, users []string) (map[string]string, []string, error) {
	values := url.Values{
		"users": users,
	}
	if teamID != "" {
		values.Add("team_id", teamID)
	}
	if toOld {
		values.Add("to_old", "true")
	}

	response := &migrationExchangeResponseFull{}
	err := api.getMethod(ctx, "migration.exchange", api.token, values, response)
	if err != nil {
		return nil, nil, err
	}

	if err := response.Err(); err != nil {
		return nil, nil, err
	}

	return response.UserIDMap, response.InvalidUserIDs, nil
}
