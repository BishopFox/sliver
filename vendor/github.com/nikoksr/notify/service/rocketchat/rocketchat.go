package rocketchat

import (
	"context"
	"fmt"
	"net/url"

	"github.com/RocketChat/Rocket.Chat.Go.SDK/models"
	"github.com/RocketChat/Rocket.Chat.Go.SDK/rest"
)

// RocketChat struct holds necessary data to communicate with the RocketChat API.
type RocketChat struct {
	client       *rest.Client
	channelNames []string
}

// New returns a new instance of a RocketChat notification service.
// serverURL is the endpoint of server i.e "localhost" , scheme is protocol i.e "http/https"
// userID and token of the user sending the message.
func New(serverURL, scheme, userID, token string) (*RocketChat, error) {
	u := url.URL{
		Scheme: scheme,
		Host:   serverURL,
	}
	authInfo := models.UserCredentials{
		ID:    userID,
		Token: token,
	}

	c := rest.NewClient(&u, false)
	if err := c.Login(&authInfo); err != nil {
		return nil, err
	}

	rc := RocketChat{
		client:       c,
		channelNames: []string{},
	}

	return &rc, nil
}

// AddReceivers takes rocketchat channel names and adds them to the internal channel list. The Send method will send
// a given message to all channels in the list.
func (r *RocketChat) AddReceivers(channelNames ...string) {
	r.channelNames = append(r.channelNames, channelNames...)
}

// Send takes a message subject and a message body and sends them to all previously set channels.
// user used for sending the message has to be a member of the channel.
// https://docs.rocket.chat/api/rest-api/methods/chat/postmessage
func (r *RocketChat) Send(ctx context.Context, subject, message string) error {
	fullMessage := subject + "\n" + message // Treating subject as message title

	for _, channelName := range r.channelNames {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msg := models.PostMessage{
				Channel: channelName,
				Text:    fullMessage,
			}
			_, err := r.client.PostMessage(&msg)
			if err != nil {
				return fmt.Errorf("send message to channel %q: %w", channelName, err)
			}
		}
	}

	return nil
}
