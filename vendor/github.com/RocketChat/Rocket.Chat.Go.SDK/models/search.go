package models

type SearchUsers struct {
	ID         string `json:"_id"`
	Status     string `json:"status"`
	Name       string `json:"name"`
	Username   string `json:"username"`
	StatusText string `json:"statustext,omitempty"`
	Nickname   string `json:"nickname,omitempty"`
	Outside    bool   `json:"outside"`
}

type SearchRooms struct {
	RoomId string `json:"_id"`
	Name   string `json:"name"`
	Type   string `json:"t"`
}
