package viber

import "encoding/json"

// Member of account details
type Member struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
	Role   string `json:"role"`
}

// Account details
type Account struct {
	Status        int    `json:"status"`
	StatusMessage string `json:"status_message"`
	ID            string `json:"id"`
	Name          string `json:"name"`
	URI           string `json:"uri"`
	Icon          string `json:"icon"`
	Background    string `json:"background"`
	Category      string `json:"category"`
	Subcategory   string `json:"subcategory"`
	Location      struct {
		Lon float64 `json:"lon"`
		Lat float64 `json:"lat"`
	} `json:"location"`
	Country          string   `json:"country"`
	Webhook          string   `json:"webhook"`
	EventTypes       []string `json:"event_types"`
	SubscribersCount int      `json:"subscribers_count"`
	Members          []Member `json:"members"`
}

// AccountInfo returns Public chat info
// https://developers.viber.com/docs/api/rest-bot-api/#get-account-info
func (v *Viber) AccountInfo() (Account, error) {
	var a Account
	b, err := v.PostData("https://chatapi.viber.com/pa/get_account_info", struct{}{})
	if err != nil {
		return a, err
	}

	err = json.Unmarshal(b, &a)
	if err != nil {
		return a, err
	}

	if a.Status != 0 {
		return a, Error{Status: a.Status, StatusMessage: a.StatusMessage}
	}

	return a, err
}
