package pushbullet

import (
	"context"
	"fmt"

	"github.com/cschomburg/go-pushbullet"
)

// Pushbullet struct holds necessary data to communicate with the Pushbullet API.
type Pushbullet struct {
	client          *pushbullet.Client
	deviceNicknames []string
}

// New returns a new instance of a Pushbullet notification service.
// For more information about Pushbullet api token:
//
//	-> https://docs.pushbullet.com/#api-overview
func New(apiToken string) *Pushbullet {
	client := pushbullet.New(apiToken)

	pb := &Pushbullet{
		client:          client,
		deviceNicknames: []string{},
	}

	return pb
}

// AddReceivers takes Pushbullet device nicknames and adds them to the internal deviceNicknames list.
// The Send method will send a given message to all those devices.
func (pb *Pushbullet) AddReceivers(deviceNicknames ...string) {
	pb.deviceNicknames = append(pb.deviceNicknames, deviceNicknames...)
}

// Send takes a message subject and a message body and sends them to all valid devices.
// you will need Pushbullet installed on the relevant devices
// (android, chrome, firefox, windows)
// see https://www.pushbullet.com/apps
func (pb Pushbullet) Send(ctx context.Context, subject, message string) error {
	for _, deviceNickname := range pb.deviceNicknames {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			dev, err := pb.client.Device(deviceNickname)
			if err != nil {
				return fmt.Errorf("get device with nickname %q: %w", deviceNickname, err)
			}

			if err = dev.PushNote(subject, message); err != nil {
				return fmt.Errorf("send push to %q: %w", deviceNickname, err)
			}
		}
	}

	return nil
}
