package telegram

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

const (
	ModeMarkdown = tgbotapi.ModeMarkdown
	ModeHTML     = tgbotapi.ModeHTML
)

// HTML is the default mode.
//
//nolint:gochecknoglobals // I agree with the linter, won't bother fixing this now, will be fixed in v2.
var parseMode = ModeHTML

// Telegram struct holds necessary data to communicate with the Telegram API.
type Telegram struct {
	client  *tgbotapi.BotAPI
	chatIDs []int64
}

// New returns a new instance of a Telegram notification service.
// For more information about telegram api token:
//
//	-> https://pkg.go.dev/github.com/go-telegram-bot-api/telegram-bot-api#NewBotAPI
func New(apiToken string) (*Telegram, error) {
	client, err := tgbotapi.NewBotAPI(apiToken)
	if err != nil {
		return nil, err
	}

	t := &Telegram{
		client:  client,
		chatIDs: []int64{},
	}

	return t, nil
}

// SetClient set a new custom BotAPI instance.
// For example allowing you to use NewBotAPIWithClient:
//
//	-> https://pkg.go.dev/github.com/go-telegram-bot-api/telegram-bot-api#NewBotAPIWithClient
func (t *Telegram) SetClient(client *tgbotapi.BotAPI) {
	t.client = client
}

// SetParseMode sets the parse mode for the message body.
// For more information about telegram constants:
//
//	-> https://pkg.go.dev/github.com/go-telegram-bot-api/telegram-bot-api#pkg-constants
func (t *Telegram) SetParseMode(mode string) {
	parseMode = mode
}

// AddReceivers takes Telegram chat IDs and adds them to the internal chat ID list. The Send method will send
// a given message to all those chats.
func (t *Telegram) AddReceivers(chatIDs ...int64) {
	t.chatIDs = append(t.chatIDs, chatIDs...)
}

// Send takes a message subject and a message body and sends them to all previously set chats. Message body supports
// html as markup language.
func (t Telegram) Send(ctx context.Context, subject, message string) error {
	fullMessage := subject + "\n" + message // Treating subject as message title

	msg := tgbotapi.NewMessage(0, fullMessage)
	msg.ParseMode = parseMode

	for _, chatID := range t.chatIDs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msg.ChatID = chatID
			if _, err := t.client.Send(msg); err != nil {
				return fmt.Errorf("send message to chat %d: %w", chatID, err)
			}
		}
	}

	return nil
}
