package lark

// PostNotificationResp response of PostNotification
type PostNotificationResp struct {
	Ok bool `json:"ok,omitempty"`
}

// PostNotificationV2Resp response of PostNotificationV2
type PostNotificationV2Resp struct {
	Code          int    `json:"code"`
	Msg           string `json:"msg"`
	StatusCode    int    `json:"StatusCode"`
	StatusMessage string `json:"StatusMessage"`
}

// PostNotification send message to a webhook
// deprecated: legacy version, please use PostNotificationV2 instead
func (bot *Bot) PostNotification(title, text string) (*PostNotificationResp, error) {
	if !bot.requireType(NotificationBot) {
		return nil, ErrBotTypeError
	}

	params := map[string]interface{}{
		"title": title,
		"text":  text,
	}
	var respData PostNotificationResp
	err := bot.PostAPIRequest("PostNotification", bot.webhook, false, params, &respData)
	return &respData, err
}

// PostNotificationV2 support v2 format
func (bot *Bot) PostNotificationV2(om OutcomingMessage) (*PostNotificationV2Resp, error) {
	if !bot.requireType(NotificationBot) {
		return nil, ErrBotTypeError
	}

	params := buildOutcomingNotification(om)
	var respData PostNotificationV2Resp
	err := bot.PostAPIRequest("PostNotificationV2", bot.webhook, false, params, &respData)
	return &respData, err
}
