package viber

import "encoding/json"

// User struct as part of UserDetails
type User struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Avatar          string `json:"avatar"`
	Country         string `json:"country"`
	Language        string `json:"language"`
	PrimaryDeviceOs string `json:"primary_device_os"`
	APIVersion      int    `json:"api_version"`
	ViberVersion    string `json:"viber_version"`
	Mcc             int    `json:"mcc"`
	Mnc             int    `json:"mnc"`
	DeviceType      string `json:"device_type"`
}

// UserDetails for Viber user
type UserDetails struct {
	Status        int    `json:"status"`
	StatusMessage string `json:"status_message"`
	MessageToken  int64  `json:"message_token"`
	User          `json:"user"`
}

// Online status struct
type online struct {
	Status        int          `json:"status"`
	StatusMessage string       `json:"status_message"`
	Users         []UserOnline `json:"users"`
}

// UserOnline response struct
type UserOnline struct {
	ID                  string `json:"id"`
	OnlineStatus        int    `json:"online_status"`
	OnlineStatusMessage string `json:"online_status_message"`
	LastOnline          int64  `json:"last_online,omitempty"`
}

// UserDetails of user id
func (v *Viber) UserDetails(id string) (UserDetails, error) {
	/*
				b := []byte(`{
				"status": 0,
				"status_message": "ok",
				"message_token": 4912661846655238145,
				"user": {
					"id": "01234567890A=",
					"name": "John McClane",
					"avatar": "http://avatar.example.com",
					"country": "UK",
					"language": "en",
					"primary_device_os": "android 7.1",
					"api_version": 1,
					"viber_version": "6.5.0",
					"mcc": 1,
					"mnc": 1
				}
			}`)

		var u UserDetails
		err := json.Unmarshal(b, &u)
		return u, err
	*/

	var u UserDetails
	s := struct {
		ID string `json:"id"`
	}{
		ID: id,
	}

	b, err := v.PostData("https://chatapi.viber.com/pa/get_user_details", s)
	if err != nil {
		return u, err
	}

	if err := json.Unmarshal(b, &u); err != nil {
		return u, err
	}

	// viber error returned
	if u.Status != 0 {
		return u, Error{Status: u.Status, StatusMessage: u.StatusMessage}
	}

	return u, err

}

// UserOnline status
func (v *Viber) UserOnline(ids []string) ([]UserOnline, error) {
	var uo online
	req := struct {
		IDs []string `json:"ids"`
	}{
		IDs: ids,
	}
	b, err := v.PostData("https://chatapi.viber.com/pa/get_online", req)
	if err != nil {
		return []UserOnline{}, err
	}

	if err := json.Unmarshal(b, &uo); err != nil {
		return []UserOnline{}, err
	}

	// viber error
	if uo.Status != 0 {
		return []UserOnline{}, Error{Status: uo.Status, StatusMessage: uo.StatusMessage}
	}

	return uo.Users, nil
}
