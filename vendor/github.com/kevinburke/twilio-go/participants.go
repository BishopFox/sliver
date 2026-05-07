package twilio

// It's difficult to work on this API since Twilio doesn't return Participants
// after a conference ends.
//
// https://github.com/saintpete/logrole/issues/4

type ParticipantService struct{}

type Participant struct {
	AccountSid             string     `json:"account_sid"`
	CallSid                string     `json:"call_sid"`
	ConferenceSid          string     `json:"conference_sid"`
	DateCreated            TwilioTime `json:"date_created"`
	DateUpdated            TwilioTime `json:"date_updated"`
	EndConferenceOnExit    bool       `json:"end_conference_on_exit"`
	Hold                   bool       `json:"hold"`
	Muted                  bool       `json:"muted"`
	StartConferenceOnEnter bool       `json:"start_conference_on_enter"`
	URI                    string     `json:"uri"`
}
