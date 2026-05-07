package lark

import (
	"context"
	"fmt"

	"github.com/go-lark/lark"

	"github.com/nikoksr/notify"
)

// WebhookService is a Notify service that uses a Lark webhook to send messages.
type WebhookService struct {
	cli sender
}

// Compile time check that larkCustomAppService implements notify.Notifer.
var _ notify.Notifier = &WebhookService{}

// NewWebhookService returns a new instance of a Lark notify service using a
// Lark group chat webhook. Note that this service does not take any
// notification receivers because it can only push messages to the group chat
// it belongs to.
func NewWebhookService(webhookURL string) *WebhookService {
	bot := lark.NewNotificationBot(webhookURL)
	return &WebhookService{
		cli: &larkClientGoLarkNotificationBot{
			bot: bot,
		},
	}
}

// Send sends the message subject and body to the group chat.
func (w *WebhookService) Send(_ context.Context, subject, message string) error {
	return w.cli.Send(subject, message)
}

// larkClientGoLarkNotificationBot is a wrapper around go-lark/lark's Bot, to
// be used for notifications via webhooks only.
type larkClientGoLarkNotificationBot struct {
	bot *lark.Bot
}

// Send implements the sender interface using a go-lark/lark notification bot.
func (w *larkClientGoLarkNotificationBot) Send(subject, message string) error {
	content := lark.NewPostBuilder().
		Title(subject).
		TextTag(message, 1, false).
		Render()
	msg := lark.NewMsgBuffer(lark.MsgPost).Post(content)
	res, err := w.bot.PostNotificationV2(msg.Build())
	if err != nil {
		return fmt.Errorf("post webhook message: %w", err)
	}
	if res.Code != 0 {
		return fmt.Errorf(
			"send failed with error code %d, please see "+
				"https://open.larksuite.com/document/ukTMukTMukTM/ugjM14COyUjL4ITN for details",
			res.Code,
		)
	}
	return nil
}
