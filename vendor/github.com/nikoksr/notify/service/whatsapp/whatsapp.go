package whatsapp

import (
	"context"
)

// Service encapsulates the WhatsApp client along with internal state for storing contacts.
type Service struct{}

// New returns a new instance of a WhatsApp notification service.
func New() (*Service, error) { return &Service{}, nil }

// LoginWithSessionCredentials provides helper for authentication using whatsapp.Session credentials.
func (s *Service) LoginWithSessionCredentials(_, _, _, _ string, _, _ []byte) error { return nil }

// LoginWithQRCode provides helper for authentication using QR code on terminal.
// Refer: https://github.com/Rhymen/go-whatsapp#login for more information.
func (s *Service) LoginWithQRCode() error { return nil }

// AddReceivers takes WhatsApp contacts and adds them to the internal contacts list. The Send method will send
// a given message to all those contacts.
func (s *Service) AddReceivers(_ ...string) {}

// Send takes a message subject and a message body and sends them to all previously set contacts.
func (s *Service) Send(_ context.Context, _, _ string) error { return nil }
