package pagerduty

import (
	"context"
	"errors"
	"fmt"

	"github.com/PagerDuty/go-pagerduty"

	"github.com/nikoksr/notify"
)

type Client interface {
	CreateIncidentWithContext(
		ctx context.Context,
		from string,
		options *pagerduty.CreateIncidentOptions,
	) (*pagerduty.Incident, error)
}

// Compile-time check to verify that the PagerDuty type implements the notifier.Notifier interface.
var _ notify.Notifier = &PagerDuty{}

type PagerDuty struct {
	*Config

	Client Client
}

func New(token string, clientOptions ...pagerduty.ClientOptions) (*PagerDuty, error) {
	if token == "" {
		return nil, errors.New("access token is required")
	}

	pagerDuty := &PagerDuty{
		Config: NewConfig(),
		Client: pagerduty.NewClient(token, clientOptions...),
	}

	return pagerDuty, nil
}

func (s *PagerDuty) Send(ctx context.Context, subject, message string) error {
	if err := s.Config.OK(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	incident := s.IncidentOptions(subject, message)

	for _, receiver := range s.Config.Receivers {
		// set the service ID to the receiver
		incident.Service.ID = receiver

		_, err := s.Client.CreateIncidentWithContext(ctx, s.Config.FromAddress, incident)
		if err != nil {
			return fmt.Errorf("create pager duty incident: %w", err)
		}
	}

	return nil
}

func (s *PagerDuty) IncidentOptions(subject, message string) *pagerduty.CreateIncidentOptions {
	return &pagerduty.CreateIncidentOptions{
		Title: subject,
		Service: &pagerduty.APIReference{
			ID:   "", // service ID will be set per receiver
			Type: APIReferenceType,
		},
		Body: &pagerduty.APIDetails{
			Type:    s.Config.NotificationType,
			Details: message,
		},
		Priority: s.Config.PriorityReference(),
		Urgency:  s.Config.Urgency,
	}
}
