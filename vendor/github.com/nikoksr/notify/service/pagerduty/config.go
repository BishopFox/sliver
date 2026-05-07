package pagerduty

import (
	"errors"
	"fmt"
	"net/mail"

	"github.com/PagerDuty/go-pagerduty"
)

const (
	APIReferenceType        = "service_reference"
	APIPriorityReference    = "priority_reference"
	DefaultNotificationType = "incident"
)

// Config contains the configuration for the PagerDuty service.
type Config struct {
	FromAddress      string
	Receivers        []string
	NotificationType string
	Urgency          string
	PriorityID       string
}

func NewConfig() *Config {
	return &Config{
		NotificationType: DefaultNotificationType,
		Receivers:        make([]string, 0, 1),
	}
}

// OK checks if the configuration is valid.
// It returns an error if the configuration is invalid.
func (c *Config) OK() error {
	if c.FromAddress == "" {
		return errors.New("from address is required")
	}

	_, err := mail.ParseAddress(c.FromAddress)
	if err != nil {
		return fmt.Errorf("from address is invalid: %w", err)
	}

	if len(c.Receivers) == 0 {
		return errors.New("at least one receiver is required")
	}

	if c.NotificationType == "" {
		return errors.New("notification type is required")
	}

	return nil
}

// PriorityReference returns the PriorityID reference if it is set, otherwise it returns nil.
func (c *Config) PriorityReference() *pagerduty.APIReference {
	if c.PriorityID == "" {
		return nil
	}

	return &pagerduty.APIReference{
		ID:   c.PriorityID,
		Type: APIPriorityReference,
	}
}

// SetFromAddress sets the from address in the configuration.
func (c *Config) SetFromAddress(fromAddress string) {
	c.FromAddress = fromAddress
}

// AddReceivers appends the receivers to the configuration.
func (c *Config) AddReceivers(receivers ...string) {
	c.Receivers = append(c.Receivers, receivers...)
}

// SetPriorityID sets the PriorityID in the configuration.
func (c *Config) SetPriorityID(priorityID string) {
	c.PriorityID = priorityID
}

// SetUrgency sets the urgency in the configuration.
func (c *Config) SetUrgency(urgency string) {
	c.Urgency = urgency
}

// SetNotificationType sets the notification type in the configuration.
// If the notification type is empty, it will be set to the default value "incident".
func (c *Config) SetNotificationType(notificationType string) {
	if notificationType == "" {
		notificationType = DefaultNotificationType
	}

	c.NotificationType = notificationType
}
