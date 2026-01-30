package mtypes

import (
	"github.com/mailgun/mailgun-go/v5/internal/types/inboxready"
)

const (
	InboxreadyDomainsVersion  = 1
	InboxreadyDomainsEndpoint = "inboxready/domains"
)

type MonitoredDomain = inboxready.InboxReadyGithubComMailgunInboxreadyModelDomain

type AddDomainToMonitoringOptions = inboxready.POSTV1InboxreadyDomainsParams
type DeleteMonitoredDomainOptions inboxready.DELETEV1InboxreadyDomainsParams

type AddDomainToMonitoringResponse = inboxready.InboxReadyGithubComMailgunInboxreadyAPIAddDomainResponse
