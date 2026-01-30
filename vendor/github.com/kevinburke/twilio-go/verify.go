package twilio

import (
	"context"
	"net/url"

	"github.com/kevinburke/go-types"
)

const servicesPathPart = "Services"
const verificationsPathPart = "Verifications"
const verificationCheckPart = "VerificationCheck"
const accessTokensPart = "AccessTokens"
const entitiesPathPart = "Entities"
const challengesPart = "Challenges"

type VerifyPhoneNumberService struct {
	client *Client
}

type VerifyAccessTokenService struct {
	client *Client
}

type VerifyChallengeService struct {
	client *Client
}

type VerifyPhoneNumber struct {
	Sid         string           `json:"sid"`
	ServiceSid  string           `json:"service_sid"`
	AccountSid  string           `json:"account_sid"`
	To          PhoneNumber      `json:"to"`
	Channel     string           `json:"channel"`
	Status      string           `json:"status"`
	Valid       bool             `json:"valid"`
	Lookup      PhoneLookup      `json:"lookup"`
	Amount      types.NullString `json:"amount"`
	Payee       types.NullString `json:"payee"`
	DateCreated TwilioTime       `json:"date_created"`
	DateUpdated TwilioTime       `json:"date_updated"`
	URL         string           `json:"url"`
}

type CheckPhoneNumber struct {
	Sid         string           `json:"sid"`
	ServiceSid  string           `json:"service_sid"`
	AccountSid  string           `json:"account_sid"`
	To          string           `json:"to"`
	Channel     string           `json:"channel"`
	Status      string           `json:"status"`
	Valid       bool             `json:"valid"`
	Amount      types.NullString `json:"amount"`
	Payee       types.NullString `json:"payee"`
	DateCreated TwilioTime       `json:"date_created"`
	DateUpdated TwilioTime       `json:"date_updated"`
}

type VerifyAccessToken struct {
	Token string `json:"token"`
}

type ChallengeLinks struct {
	Notifications string `json:"notifications"`
}

type VerifyChallenge struct {
	Sid             string                 `json:"sid"`
	AccountSid      string                 `json:"account_sid"`
	ServiceSid      string                 `json:"service_sid"`
	EntitySid       string                 `json:"entity_sid"`
	Identity        string                 `json:"identity"`
	FactorSid       string                 `json:"factor_sid"`
	DateCreated     TwilioTime             `json:"date_created"`
	DateUpdated     TwilioTime             `json:"date_updated"`
	DateResponded   TwilioTime             `json:"date_responded"`
	ExpirationDate  TwilioTime             `json:"expiration_date"`
	Status          string                 `json:"status"`
	RespondedReason string                 `json:"responded_reason"`
	Details         map[string]interface{} `json:"details"`
	HiddenDetails   map[string]interface{} `json:"hidden_details"`
	FactorType      string                 `json:"factor_type"`
	Url             string                 `json:"url"`
	Links           ChallengeLinks         `json:"links"`
}

// Create calls the Verify API to start a new verification.
// https://www.twilio.com/docs/verify/api-beta/verification-beta#start-new-verification
func (v *VerifyPhoneNumberService) Create(ctx context.Context, verifyServiceID string, data url.Values) (*VerifyPhoneNumber, error) {
	verify := new(VerifyPhoneNumber)
	err := v.client.CreateResource(ctx, servicesPathPart+"/"+verifyServiceID+"/"+verificationsPathPart, data, verify)
	return verify, err
}

// Get calls the Verify API to retrieve information about a verification.
// https://www.twilio.com/docs/verify/api-beta/verification-beta#fetch-a-verification-1
func (v *VerifyPhoneNumberService) Get(ctx context.Context, verifyServiceID string, sid string) (*VerifyPhoneNumber, error) {
	verify := new(VerifyPhoneNumber)
	err := v.client.GetResource(ctx, servicesPathPart+"/"+verifyServiceID+"/"+verificationsPathPart, sid, verify)
	return verify, err
}

// Check calls the Verify API to check if a user-provided token is correct.
// https://www.twilio.com/docs/verify/api-beta/verification-check-beta#check-a-verification-1
func (v *VerifyPhoneNumberService) Check(ctx context.Context, verifyServiceID string, data url.Values) (*CheckPhoneNumber, error) {
	check := new(CheckPhoneNumber)
	err := v.client.CreateResource(ctx, servicesPathPart+"/"+verifyServiceID+"/"+verificationCheckPart, data, check)
	return check, err
}

// Create calls the Verify API to create an access token
// https://www.twilio.com/docs/verify/api/access-token#
func (v *VerifyAccessTokenService) Create(ctx context.Context, verifyServiceID string, data url.Values) (*VerifyAccessToken, error) {
	accessToken := new(VerifyAccessToken)
	err := v.client.CreateResource(ctx, servicesPathPart+"/"+verifyServiceID+"/"+accessTokensPart, data, accessToken)
	return accessToken, err
}

// Create calls the Verify API to create a challenge
// https://www.twilio.com/docs/verify/api/challenge#create-a-challenge-resource
func (v *VerifyChallengeService) Create(ctx context.Context, verifyServiceID string, identity string, data url.Values) (*VerifyChallenge, error) {
	challenge := new(VerifyChallenge)
	err := v.client.CreateResource(ctx, servicesPathPart+"/"+verifyServiceID+"/"+entitiesPathPart+"/"+identity+"/"+challengesPart, data, challenge)
	return challenge, err
}

// Get calls the Verify API to get a challenge
// https://www.twilio.com/docs/verify/api/challenge#fetch-a-challenge-resource
func (v *VerifyChallengeService) Get(ctx context.Context, verifyServiceID string, identity string, sid string) (*VerifyChallenge, error) {
	challenge := new(VerifyChallenge)
	err := v.client.GetResource(ctx, servicesPathPart+"/"+verifyServiceID+"/"+entitiesPathPart+"/"+identity+"/"+challengesPart, sid, challenge)
	return challenge, err
}
