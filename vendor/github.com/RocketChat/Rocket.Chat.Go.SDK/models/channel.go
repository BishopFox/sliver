package models

import "time"

type Channel struct {
	ID    string `json:"_id"`
	Name  string `json:"name"`
	Fname string `json:"fname,omitempty"`
	Type  string `json:"t"`
	Msgs  int    `json:"msgs"`

	ReadOnly  bool `json:"ro,omitempty"`
	SysMes    bool `json:"sysMes,omitempty"`
	Default   bool `json:"default"`
	Broadcast bool `json:"broadcast,omitempty"`

	Timestamp *time.Time `json:"ts,omitempty"`
	UpdatedAt *time.Time `json:"_updatedAt,omitempty"`

	User        *User    `json:"u,omitempty"`
	LastMessage *Message `json:"lastMessage,omitempty"`

	// Lm          interface{} `json:"lm"`
	// CustomFields struct {
	// } `json:"customFields,omitempty"`
}

type ChannelSubscription struct {
	ID          string   `json:"_id"`
	Alert       bool     `json:"alert"`
	Name        string   `json:"name"`
	DisplayName string   `json:"fname"`
	Open        bool     `json:"open"`
	RoomId      string   `json:"rid"`
	Type        string   `json:"c"`
	User        User     `json:"u"`
	Roles       []string `json:"roles"`
	Unread      float64  `json:"unread"`
}
