package matrix

import (
	"context"
	"errors"

	matrix "maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

type matrixClient interface {
	SendMessageEvent(
		ctx context.Context,
		roomID id.RoomID,
		eventType event.Type,
		contentJSON any,
		extra ...matrix.ReqSendEvent,
	) (resp *matrix.RespSendEvent, err error)
}

// Compile time check to ensure that matrix.Client implements the matrixClient interface.
var _ matrixClient = new(matrix.Client)

// New returns a new instance of a Matrix notification service.
// For more information about the Matrix api specs:
//
// -> https://spec.matrix.org/v1.2/client-server-api
func New(userID id.UserID, roomID id.RoomID, homeServer, accessToken string) (*Matrix, error) {
	client, err := matrix.NewClient(homeServer, userID, accessToken)
	if err != nil {
		return nil, err
	}

	s := &Matrix{
		client: client,
		options: ServiceOptions{
			homeServer:  homeServer,
			accessToken: accessToken,
			userID:      userID,
			roomID:      roomID,
		},
	}
	return s, nil
}

// Send takes a message body and sends them to the previously set channel.
// you will need an account, access token and roomID
// see https://matrix.org
func (s *Matrix) Send(ctx context.Context, _, message string) error {
	messageBody := createMessage(message)

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		_, err := s.client.SendMessageEvent(ctx, s.options.roomID, event.EventMessage, &messageBody)
		if err != nil {
			return errors.New("failed to send message to the room using Matrix")
		}
	}
	return nil
}

func createMessage(message string) Message {
	return Message{
		Body:    message,
		Msgtype: event.MsgText,
	}
}
