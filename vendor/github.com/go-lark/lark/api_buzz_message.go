package lark

import "fmt"

const (
	buzzInAppURL = "/open-apis/im/v1/messages/%s/urgent_app?user_id_type=%s"
	buzzSMSURL   = "/open-apis/im/v1/messages/%s/urgent_sms?user_id_type=%s"
	buzzPhoneURL = "/open-apis/im/v1/messages/%s/urgent_phone?user_id_type=%s"
)

// Buzz types
const (
	BuzzTypeInApp = "buzz_inapp"
	BuzzTypeSMS   = "buzz_sms"
	BuzzTypePhone = "buzz_phone"
)

// BuzzMessageResponse .
type BuzzMessageResponse struct {
	BaseResponse

	Data struct {
		InvalidUserIDList []string `json:"invalid_user_id_list,omitempty"`
	} `json:"data,omitempty"`
}

// BuzzMessage .
func (bot Bot) BuzzMessage(buzzType string, messageID string, userIDList ...string) (*BuzzMessageResponse, error) {
	var respData BuzzMessageResponse
	url := buzzInAppURL
	switch buzzType {
	case BuzzTypeInApp:
		url = buzzInAppURL
	case BuzzTypeSMS:
		url = buzzSMSURL
	case BuzzTypePhone:
		url = buzzPhoneURL
	}
	req := map[string][]string{
		"user_id_list": userIDList,
	}
	err := bot.PatchAPIRequest("BuzzMessage", fmt.Sprintf(url, messageID, bot.userIDType), true, req, &respData)
	return &respData, err
}
