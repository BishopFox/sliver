package viber

import "encoding/json"

//
//https://chatapi.viber.com/pa/set_webhook
// {
//    "url": "https://my.host.com",
//    "event_types": ["delivered", "seen", "failed", "subscribed", "unsubscribed", "conversation_started"]
// }

// WebhookReq request
type WebhookReq struct {
	URL        string   `json:"url"`
	EventTypes []string `json:"event_types"`
}

// {
//     "status": 0,
//     "status_message": "ok",
//     "event_types": ["delivered", "seen", "failed", "subscribed",  "unsubscribed", "conversation_started"]
// }

//WebhookResp response
type WebhookResp struct {
	Status        int      `json:"status"`
	StatusMessage string   `json:"status_message"`
	EventTypes    []string `json:"event_types,omitempty"`
}

// WebhookVerify response
type WebhookVerify struct {
	Event        string `json:"event"`
	Timestamp    uint64 `json:"timestamp"`
	MessageToken uint64 `json:"message_token"`
}

// SetWebhook for Viber callbacks
// if eventTypes is nil, all callbacks will be set to webhook
// if eventTypes is empty []string mandatory callbacks will be set
// Mandatory callbacks: "message", "subscribed", "unsubscribed"
// All possible callbacks: "message", "subscribed",  "unsubscribed", "delivered", "seen", "failed", "conversation_started"
func (v *Viber) SetWebhook(url string, eventTypes []string) (WebhookResp, error) {
	var resp WebhookResp

	req := WebhookReq{
		URL:        url,
		EventTypes: eventTypes,
	}
	r, err := v.PostData("https://chatapi.viber.com/pa/set_webhook", req)
	if err != nil {
		return resp, err
	}

	err = json.Unmarshal(r, &resp)
	return resp, err
}
