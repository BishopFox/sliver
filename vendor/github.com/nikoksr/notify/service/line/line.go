package line

import (
	"context"
	"fmt"

	"github.com/line/line-bot-sdk-go/linebot"
)

// Line struct holds info about client and destination ID for communicating with line API.
type Line struct {
	client      *linebot.Client
	receiverIDs []string
}

// New creates a new instance of Line notifier service
// For more info about line api credential:
// -> https://github.com/line/line-bot-sdk-go
func New(channelSecret, channelAccessToken string) (*Line, error) {
	bot, err := linebot.New(channelSecret, channelAccessToken)
	if err != nil {
		return nil, err
	}

	l := &Line{
		client: bot,
	}

	return l, nil
}

// AddReceivers receives user, group or room IDs then add them to internal receivers list.
func (l *Line) AddReceivers(receiverIDs ...string) {
	l.receiverIDs = append(l.receiverIDs, receiverIDs...)
}

// Send receives message subject and body then sends it to all receivers set previously
// Subject will be on the first line followed by message on the next line.
func (l *Line) Send(ctx context.Context, subject, message string) error {
	lineMessage := &linebot.TextMessage{
		Text: subject + "\n" + message,
	}

	for _, receiverID := range l.receiverIDs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			_, err := l.client.PushMessage(receiverID, lineMessage).WithContext(ctx).Do()
			if err != nil {
				return fmt.Errorf("push message to %q: %w", receiverID, err)
			}
		}
	}

	return nil
}
