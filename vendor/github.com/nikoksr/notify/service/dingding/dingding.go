package dingding

import (
	"context"
	"fmt"

	"github.com/blinkbean/dingtalk"
)

// Service encapsulates the DingTalk client.
type Service struct {
	config Config
	client *dingtalk.DingTalk
}

// Config is the Service configuration.
type Config struct {
	Token  string
	Secret string
}

// New returns a new instance of a DingTalk notification service.
func New(cfg *Config) *Service {
	dt := dingtalk.InitDingTalkWithSecret(cfg.Token, cfg.Secret)
	s := Service{
		config: *cfg,
		client: dt,
	}

	return &s
}

// Send takes a message subject and a message content and sends them to all previously set users.
func (s *Service) Send(ctx context.Context, subject, content string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		text := subject + "\n" + content
		err := s.client.SendTextMessage(text)
		if err != nil {
			return fmt.Errorf("send message: %w", err)
		}
	}

	return nil
}
