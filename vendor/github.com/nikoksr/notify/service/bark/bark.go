package bark

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Service allow you to configure Bark service.
type Service struct {
	deviceKey  string
	client     *http.Client
	serverURLs []string
}

func defaultHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 5 * time.Second, //nolint: mnd // 5 seconds is a reasonable timeout for a push notification
	}
}

// DefaultServerURL is the default server to use for the bark service.
const DefaultServerURL = "https://api.day.app/"

// normalizeServerURL normalizes the server URL. It prefixes it with https:// if it's not already and appends a slash
// if it's not already there. If the serverURL is empty, the DefaultServerURL is used. We're not validating the url here
// on purpose, we leave that to the http client.
func normalizeServerURL(serverURL string) string {
	if serverURL == "" {
		return DefaultServerURL
	}

	// Normalize the url
	if !strings.HasPrefix(serverURL, "http") {
		serverURL = "https://" + serverURL
	}
	if !strings.HasSuffix(serverURL, "/") {
		serverURL += "/"
	}

	return serverURL
}

// AddReceivers adds server URLs to the list of servers to use for sending messages. We call it Receivers and not
// servers because strictly speaking, the server is still receiving the message, and additionally we're following the
// naming convention of the other services.
func (s *Service) AddReceivers(serverURLs ...string) {
	for _, serverURL := range serverURLs {
		serverURL = normalizeServerURL(serverURL)
		s.serverURLs = append(s.serverURLs, serverURL)
	}
}

// NewWithServers returns a new instance of Bark service. You can use this service to send messages to bark. You can
// specify the servers to send the messages to. By default, the service will use the default server
// (https://api.day.app/) if you don't specify any servers.
func NewWithServers(deviceKey string, serverURLs ...string) *Service {
	s := &Service{
		deviceKey:  deviceKey,
		client:     defaultHTTPClient(),
		serverURLs: make([]string, 0),
	}

	if len(serverURLs) == 0 {
		serverURLs = append(serverURLs, DefaultServerURL)
	}

	// Calling service.AddReceivers() instead of directly setting the serverURLs because we want to normalize the URLs.
	s.AddReceivers(serverURLs...)

	return s
}

// New returns a new instance of Bark service. You can use this service to send messages to bark. By default, the
// service will use the default server (https://api.day.app/).
func New(deviceKey string) *Service {
	return NewWithServers(deviceKey)
}

// postData is the data to send to the bark server.
type postData struct {
	DeviceKey string `json:"device_key"`
	Title     string `json:"title"`
	Body      string `json:"body,omitempty"`
	Badge     int    `json:"badge,omitempty"`
	Sound     string `json:"sound,omitempty"`
	Icon      string `json:"icon,omitempty"`
	Group     string `json:"group,omitempty"`
	URL       string `json:"pushURL,omitempty"`
}

func (s *Service) send(ctx context.Context, serverURL, subject, content string) error {
	if serverURL == "" {
		return errors.New("server url is empty")
	}

	// Marshal the message to post
	message := &postData{
		DeviceKey: s.deviceKey,
		Title:     subject,
		Body:      content,
		Sound:     "alarm.caf",
	}

	messageJSON, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	pushURL := serverURL + "push"

	// Create new request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, pushURL, bytes.NewBuffer(messageJSON))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response and verify success
	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bark returned status code %d: %s", resp.StatusCode, string(result))
	}

	return nil
}

// Send takes a message subject and a message content and sends them to bark application.
func (s *Service) Send(ctx context.Context, subject, content string) error {
	if s.client == nil {
		return errors.New("client is nil")
	}

	for _, serverURL := range s.serverURLs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			err := s.send(ctx, serverURL, subject, content)
			if err != nil {
				return fmt.Errorf("send message to bark server %q: %w", serverURL, err)
			}
		}
	}

	return nil
}
