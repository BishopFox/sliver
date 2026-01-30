package mattermost

import (
	"context"
	"errors"
	"fmt"
	"io"
	stdhttp "net/http"

	"github.com/nikoksr/notify/service/http"
)

type httpClient interface {
	AddReceivers(wh ...*http.Webhook)
	PreSend(prefn http.PreSendHookFn)
	Send(ctx context.Context, subject, message string) error
	PostSend(postfn http.PostSendHookFn)
}

// Service encapsulates the notify httpService client and contains mattermost channel ids.
type Service struct {
	loginClient   httpClient
	messageClient httpClient
	channelIDs    map[string]bool
}

// New returns a new instance of a Mattermost notification service.
func New(url string) *Service {
	httpService := setupMsgService(url)
	return &Service{
		setupLoginService(url, httpService),
		httpService,
		make(map[string]bool),
	}
}

// LoginWithCredentials provides helper for authentication using Mattermost user/admin credentials.
func (s *Service) LoginWithCredentials(ctx context.Context, loginID, password string) error {
	return s.loginClient.Send(ctx, loginID, password)
}

// AddReceivers takes Mattermost channel IDs or Chat IDs and adds them to the internal channel ID list.
// The Send method will send a given message to all these channels.
func (s *Service) AddReceivers(channelIDs ...string) {
	for i := range channelIDs {
		s.channelIDs[channelIDs[i]] = true
	}
}

// Send takes a message subject and a message body and send them to added channel ids.
// you will need a 'create_post' permission for your username.
// refer https://api.mattermost.com/ for more info.
func (s *Service) Send(ctx context.Context, subject, message string) error {
	for id := range s.channelIDs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// create post
			if err := s.messageClient.Send(ctx, id, subject+"\n"+message); err != nil {
				return fmt.Errorf("send message to channel %q: %w", id, err)
			}
		}
	}

	return nil
}

// PreSend adds a pre-send hook to the service. The hook will be executed before sending a request to a receiver.
func (s *Service) PreSend(hook http.PreSendHookFn) {
	s.messageClient.PreSend(hook)
}

// PostSend adds a post-send hook to the service. The hook will be executed after sending a request to a receiver.
func (s *Service) PostSend(hook http.PostSendHookFn) {
	s.messageClient.PostSend(hook)
}

// setups main message service for creating posts.
func setupMsgService(url string) *http.Service {
	// create new http client for sending messages/notifications
	httpService := http.New()

	// add custom payload builder
	httpService.AddReceivers(&http.Webhook{
		URL:         url + "/api/v4/posts",
		Header:      stdhttp.Header{},
		ContentType: "application/json",
		Method:      stdhttp.MethodPost,
		BuildPayload: func(channelID, subjectAndMessage string) any {
			return map[string]string{
				"channel_id": channelID,
				"message":    subjectAndMessage,
			}
		},
	})

	// add post-send hook for error checks
	httpService.PostSend(func(_ *stdhttp.Request, resp *stdhttp.Response) error {
		if resp.StatusCode != stdhttp.StatusCreated {
			b, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("create post failed with status: %s body: %s", resp.Status, string(b))
		}
		return nil
	})

	return httpService
}

// setups login service to get token.
func setupLoginService(url string, msgService *http.Service) *http.Service {
	// create another new http client for login request call.
	httpService := http.New()

	// append login path for the given mattermost server with custom payload builder.
	httpService.AddReceivers(&http.Webhook{
		URL:         url + "/api/v4/users/login",
		Header:      stdhttp.Header{},
		ContentType: "application/json",
		Method:      stdhttp.MethodPost,
		BuildPayload: func(loginID, password string) any {
			return map[string]string{
				"login_id": loginID,
				"password": password,
			}
		},
	})

	// Add post-send hook to do error checks and log the response after it is received.
	// Also extract token from response header and set it as part of pre-send hook of main http client for further
	// requests.
	httpService.PostSend(func(_ *stdhttp.Request, resp *stdhttp.Response) error {
		if resp.StatusCode != stdhttp.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("login failed with status: %s body: %s", resp.Status, string(b))
		}

		// get token from header
		token := resp.Header.Get("Token")
		if token == "" {
			return errors.New("received empty token")
		}

		// set token as pre-send hook
		msgService.PreSend(func(req *stdhttp.Request) error {
			req.Header.Set("Authorization", "Bearer "+token)
			return nil
		})
		return nil
	})
	return httpService
}
