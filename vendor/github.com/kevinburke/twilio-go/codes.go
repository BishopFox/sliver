package twilio

import (
	"encoding/json"
	"strconv"
)

// A Twilio error code. A full list can be found here:
// https://www.twilio.com/docs/api/errors/reference
type Code int

func (c *Code) convCode(i *int, err error) error {
	if err != nil {
		return err
	}
	if *i == 4107 {
		// Twilio incorrectly sent back 14107 as 4107 in some cases. Not sure
		// how often this happened or if the problem is more general
		*i = 14107
	}
	*c = Code(*i)
	return nil
}

func (c *Code) UnmarshalJSON(b []byte) error {
	s := new(string)
	if err := json.Unmarshal(b, s); err == nil {
		if *s == "" || *s == "null" {
			*c = Code(0)
			return nil
		}
		i, err := strconv.Atoi(*s)
		return c.convCode(&i, err)
	}
	i := new(int)
	err := json.Unmarshal(b, i)
	return c.convCode(i, err)
}

const CodeHTTPRetrievalFailure = 11200
const CodeHTTPConnectionFailure = 11205
const CodeHTTPProtocolViolation = 11206
const CodeReplyLimitExceeded = 14107
const CodeDocumentParseFailure = 12100
const CodeForbiddenPhoneNumber = 13225
const CodeNoInternationalAuthorization = 13227
const CodeSayInvalidText = 13520
const CodeQueueOverflow = 30001
const CodeAccountSuspended = 30002
const CodeUnreachable = 30003
const CodeMessageBlocked = 30004
const CodeUnknownDestination = 30005
const CodeLandline = 30006
const CodeCarrierViolation = 30007
const CodeUnknownError = 30008
const CodeMissingSegment = 30009
const CodeMessagePriceExceedsMaxPrice = 30010
