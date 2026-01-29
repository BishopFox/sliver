package slack

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

// SlashCommand contains information about a request of the slash command
type SlashCommand struct {
	Token               string `json:"token"`
	TeamID              string `json:"team_id"`
	TeamDomain          string `json:"team_domain"`
	EnterpriseID        string `json:"enterprise_id,omitempty"`
	EnterpriseName      string `json:"enterprise_name,omitempty"`
	IsEnterpriseInstall bool   `json:"is_enterprise_install"`
	ChannelID           string `json:"channel_id"`
	ChannelName         string `json:"channel_name"`
	UserID              string `json:"user_id"`
	UserName            string `json:"user_name"`
	Command             string `json:"command"`
	Text                string `json:"text"`
	ResponseURL         string `json:"response_url"`
	TriggerID           string `json:"trigger_id"`
	APIAppID            string `json:"api_app_id"`
}

// SlashCommandParse will parse the request of the slash command
func SlashCommandParse(r *http.Request) (s SlashCommand, err error) {
	if err = r.ParseForm(); err != nil {
		return s, err
	}
	s.Token = r.PostForm.Get("token")
	s.TeamID = r.PostForm.Get("team_id")
	s.TeamDomain = r.PostForm.Get("team_domain")
	s.EnterpriseID = r.PostForm.Get("enterprise_id")
	s.EnterpriseName = r.PostForm.Get("enterprise_name")
	s.IsEnterpriseInstall = r.PostForm.Get("is_enterprise_install") == "true"
	s.ChannelID = r.PostForm.Get("channel_id")
	s.ChannelName = r.PostForm.Get("channel_name")
	s.UserID = r.PostForm.Get("user_id")
	s.UserName = r.PostForm.Get("user_name")
	s.Command = r.PostForm.Get("command")
	s.Text = r.PostForm.Get("text")
	s.ResponseURL = r.PostForm.Get("response_url")
	s.TriggerID = r.PostForm.Get("trigger_id")
	s.APIAppID = r.PostForm.Get("api_app_id")
	return s, nil
}

// ValidateToken validates verificationTokens
func (s SlashCommand) ValidateToken(verificationTokens ...string) bool {
	for _, token := range verificationTokens {
		if s.Token == token {
			return true
		}
	}
	return false
}

// UnmarshalJSON handles is_enterprise_install being either a boolean or a
// string when parsing JSON from various payloads
func (s *SlashCommand) UnmarshalJSON(data []byte) error {
	type SlashCommandCopy SlashCommand
	scopy := &struct {
		*SlashCommandCopy
		IsEnterpriseInstall interface{} `json:"is_enterprise_install"`
	}{
		SlashCommandCopy: (*SlashCommandCopy)(s),
	}

	if err := json.Unmarshal(data, scopy); err != nil {
		return err
	}

	switch rawValue := scopy.IsEnterpriseInstall.(type) {
	case string:
		b, err := strconv.ParseBool(rawValue)
		if err != nil {
			return fmt.Errorf("parsing boolean for is_enterprise_install: %w", err)
		}
		s.IsEnterpriseInstall = b
	case bool:
		s.IsEnterpriseInstall = rawValue
	default:
		return fmt.Errorf("wrong data type for is_enterprise_install: %T", scopy.IsEnterpriseInstall)
	}

	return nil
}
