package matrix

import (
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

// ServiceOptions allow you to configure the Matrix client options.
type ServiceOptions struct {
	homeServer  string
	accessToken string
	userID      id.UserID
	roomID      id.RoomID
}

// Message structure that reassembles the SendMessageEvent.
type Message struct {
	Body          string            `json:"body"`
	Format        string            `json:"format,omitempty"`
	FormattedBody string            `json:"formatted_body,omitempty"`
	Msgtype       event.MessageType `json:"msgtype"`
}

// Matrix struct that holds necessary data to communicate with the Matrix API.
type Matrix struct {
	client  matrixClient
	options ServiceOptions
}
