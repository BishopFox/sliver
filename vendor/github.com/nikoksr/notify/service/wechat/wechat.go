package wechat

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"sync"
	"time"

	"github.com/silenceper/wechat/v2"
	"github.com/silenceper/wechat/v2/cache"
	"github.com/silenceper/wechat/v2/officialaccount/config"
	"github.com/silenceper/wechat/v2/officialaccount/message"
	"github.com/silenceper/wechat/v2/util"
)

const defaultTimeout = 20 * time.Second

type verificationCallbackFunc func(r *http.Request, verified bool)

// Config is the Service configuration.
type Config struct {
	AppID          string
	AppSecret      string
	Token          string
	EncodingAESKey string
	Cache          cache.Cache
}

// wechatMessageManager abstracts go-wechat's message.Manager for writing unit tests.
type wechatMessageManager interface {
	Send(msg *message.CustomerMessage) error
}

// Service encapsulates the WeChat client along with internal state for storing users.
type Service struct {
	config         *Config
	messageManager wechatMessageManager
	userIDs        []string
}

// New returns a new instance of a WeChat notification service.
func New(cfg *Config) *Service {
	wc := wechat.NewWechat()
	wcCfg := &config.Config{
		AppID:          cfg.AppID,
		AppSecret:      cfg.AppSecret,
		Token:          cfg.Token,
		EncodingAESKey: cfg.EncodingAESKey,
		Cache:          cfg.Cache,
	}

	oa := wc.GetOfficialAccount(wcCfg)

	return &Service{
		config:         cfg,
		messageManager: oa.GetCustomerMessageManager(),
	}
}

// waitForOneOffVerification waits for the verification call from the WeChat backend.
//
// Should be running when (re-)applying settings in wechat configuration.
//
// Set devMode to true when using the sandbox.
//
// See https://developers.weixin.qq.com/doc/offiaccount/en/Basic_Information/Access_Overview.html
func (s *Service) waitForOneOffVerification(
	server *http.Server,
	devMode bool,
	callback verificationCallbackFunc,
) error {
	verificationDone := &sync.WaitGroup{}
	verificationDone.Add(1)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()

		echoStr := html.EscapeString(query.Get("echostr"))
		if devMode {
			if callback != nil {
				callback(r, true)
			}
			_, _ = w.Write([]byte(echoStr))
			// verification done; dev mode
			verificationDone.Done()
			return
		}

		// perform signature check
		timestamp := query.Get("timestamp")
		nonce := query.Get("nonce")
		suppliedSignature := query.Get("signature")
		computedSignature := util.Signature(s.config.Token, timestamp, nonce)
		if suppliedSignature == computedSignature {
			if callback != nil {
				callback(r, true)
			}
			_, _ = w.Write([]byte(echoStr))
			// verification done; prod mode
			verificationDone.Done()
			return
		}

		// verification not done (keep waiting)
		if callback != nil {
			callback(r, false)
		}
	})

	var err error
	go func() {
		if innerErr := server.ListenAndServe(); innerErr != http.ErrServerClosed {
			err = fmt.Errorf("start verification listener: %w", innerErr)
		}
	}()

	// wait until verification is done and shutdown the server
	verificationDone.Wait()

	if serr := server.Shutdown(context.TODO()); serr != nil {
		err = fmt.Errorf("shutdown verification listener: %w", serr)
	}

	return err
}

// WaitForOneOffVerificationWithServer allows you to use WaitForOneOffVerification with a fully custom HTTP server.
//
// Should be running when (re-)applying settings in wechat configuration.
//
// Set devMode to true when using the sandbox.
//
// See https://developers.weixin.qq.com/doc/offiaccount/en/Basic_Information/Access_Overview.html
func (s *Service) WaitForOneOffVerificationWithServer(
	server *http.Server,
	devMode bool,
	callback verificationCallbackFunc,
) error {
	return s.waitForOneOffVerification(server, devMode, callback)
}

// WaitForOneOffVerification waits for the verification call from the WeChat backend. It uses an internal
// ReadHeaderTimeout of 20 seconds to avoid blocking the caller for too long (potential slow loris attack). In case
// that you want to use a different timeout, you can use the WaitForOneOffVerificationWithServer method instead. It
// allows you to specify a custom server.
//
// Should be running when (re-)applying settings in wechat configuration.
//
// Set devMode to true when using the sandbox.
//
// See https://developers.weixin.qq.com/doc/offiaccount/en/Basic_Information/Access_Overview.html
func (s *Service) WaitForOneOffVerification(serverURL string, devMode bool, callback verificationCallbackFunc) error {
	server := &http.Server{
		Addr:              serverURL,
		ReadHeaderTimeout: defaultTimeout,
	}

	return s.WaitForOneOffVerificationWithServer(server, devMode, callback)
}

// AddReceivers takes user ids and adds them to the internal users list. The Send method will send
// a given message to all those users.
func (s *Service) AddReceivers(userIDs ...string) {
	s.userIDs = append(s.userIDs, userIDs...)
}

// Send takes a message subject and a message content and sends them to all previously set users.
func (s *Service) Send(ctx context.Context, subject, content string) error {
	for _, userID := range s.userIDs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			text := fmt.Sprintf("%s\n%s", subject, content)
			err := s.messageManager.Send(message.NewCustomerTextMessage(userID, text))
			if err != nil {
				return fmt.Errorf("send message to user %q: %w", userID, err)
			}
		}
	}

	return nil
}
