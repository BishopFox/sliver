package mtypes

import (
	"github.com/mailgun/mailgun-go/v5/internal/types/inboxready"
)

const (
	InboxPlacementVersion       = 4
	InboxPlacementEndpoint      = "inbox"
	InboxPlacementTestsEndpoint = InboxPlacementEndpoint + "/tests"
)

// CreateInboxPlacementTestOptions request for running an inbox placement test.
// TODO(vtopc): add account_template_name string:
type CreateInboxPlacementTestOptions = inboxready.POSTV4InboxTestsJSONRequestBody

type (
	CreateInboxPlacementTestResponse      = inboxready.InboxPlacementTestingGithubComMailgunSpyInternalAPICreateTestResp
	CreateInboxPlacementTestResponseLinks = inboxready.InboxPlacementTestingGithubComMailgunSpyInternalAPICreateTestRespLinks
)
