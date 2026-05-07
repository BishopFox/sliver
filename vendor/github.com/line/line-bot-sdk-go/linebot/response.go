// Copyright 2016 LINE Corporation
//
// LINE Corporation licenses this file to you under the Apache License,
// version 2.0 (the "License"); you may not use this file except in compliance
// with the License. You may obtain a copy of the License at:
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package linebot

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

// BasicResponse type
type BasicResponse struct {
	RequestID string
}

type errorResponseDetail struct {
	Message  string `json:"message"`
	Property string `json:"property"`
}

// ErrorResponse type
type ErrorResponse struct {
	Message string                `json:"message"`
	Details []errorResponseDetail `json:"details"`
	// OAuth Errors
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// UserProfileResponse type
type UserProfileResponse struct {
	UserID        string `json:"userId"`
	DisplayName   string `json:"displayName"`
	PictureURL    string `json:"pictureUrl"`
	StatusMessage string `json:"statusMessage"`
	Language      string `json:"language"`
}

// GroupSummaryResponse type
type GroupSummaryResponse struct {
	GroupID    string `json:"groupId"`
	GroupName  string `json:"groupName"`
	PictureURL string `json:"pictureUrl"`
}

// MemberIDsResponse type
type MemberIDsResponse struct {
	MemberIDs []string `json:"memberIds"`
	Next      string   `json:"next"`
}

// MemberCountResponse type
type MemberCountResponse struct {
	Count int `json:"count"`
}

// MessageContentResponse type
type MessageContentResponse struct {
	Content       io.ReadCloser
	ContentLength int64
	ContentType   string
}

// MessagesNumberResponse type
type MessagesNumberResponse struct {
	Status  string
	Success int64
}

// MessageQuotaResponse type
type MessageQuotaResponse struct {
	Type       string
	Value      int64
	TotalUsage int64 `json:"totalUsage"`
}

// MessageConsumptionResponse type
type MessageConsumptionResponse struct {
	TotalUsage int64
}

// BotInfoResponse type
type BotInfoResponse struct {
	UserID         string         `json:"userId"`
	BasicID        string         `json:"basicId"`
	PremiumID      string         `json:"premiumId"`
	DisplayName    string         `json:"displayName"`
	PictureURL     string         `json:"pictureUrl"`
	ChatMode       ChatMode       `json:"chatMode"`
	MarkAsReadMode MarkAsReadMode `json:"markAsReadMode"`
}

// MessagesNumberDeliveryResponse type
type MessagesNumberDeliveryResponse struct {
	Status          string `json:"status"`
	Broadcast       int64  `json:"broadcast"`
	Targeting       int64  `json:"targeting"`
	AutoResponse    int64  `json:"autoResponse"`
	WelcomeResponse int64  `json:"welcomeResponse"`
	Chat            int64  `json:"chat"`
	APIBroadcast    int64  `json:"apiBroadcast"`
	APIPush         int64  `json:"apiPush"`
	APIMulticast    int64  `json:"apiMulticast"`
	APINarrowcast   int64  `json:"apiNarrowcast"`
	APIReply        int64  `json:"apiReply"`
}

// MessagesNumberFollowersResponse type
type MessagesNumberFollowersResponse struct {
	Status          string `json:"status"`
	Followers       int64  `json:"followers"`
	TargetedReaches int64  `json:"targetedReaches"`
	Blocks          int64  `json:"blocks"`
}

// MessagesProgressResponse type
type MessagesProgressResponse struct {
	Phase             string `json:"phase"`
	SuccessCount      int64  `json:"successCount"`
	FailureCount      int64  `json:"failureCount"`
	TargetCount       int64  `json:"targetCount"`
	FailedDescription string `json:"failedDescription"`
	ErrorCode         int    `json:"errorCode"`
	AcceptedTime      string `json:"acceptedTime"`
	CompletedTime     string `json:"completedTime,omitempty"`
}

// MessagesFriendDemographicsResponse type
type MessagesFriendDemographicsResponse struct {
	Available           bool                       `json:"available"`
	Genders             []GenderDetail             `json:"genders"`
	Ages                []AgeDetail                `json:"ages"`
	Areas               []AreasDetail              `json:"areas"`
	AppTypes            []AppTypeDetail            `json:"appTypes"`
	SubscriptionPeriods []SubscriptionPeriodDetail `json:"subscriptionPeriods"`
}

// GenderDetail type
type GenderDetail struct {
	Gender     string  `json:"gender"`
	Percentage float64 `json:"percentage"`
}

// AgeDetail type
type AgeDetail struct {
	Age        string  `json:"age"`
	Percentage float64 `json:"percentage"`
}

// AreasDetail type
type AreasDetail struct {
	Area       string  `json:"area"`
	Percentage float64 `json:"percentage"`
}

// AppTypeDetail type
type AppTypeDetail struct {
	AppType    string  `json:"appType"`
	Percentage float64 `json:"percentage"`
}

// SubscriptionPeriodDetail type
type SubscriptionPeriodDetail struct {
	SubscriptionPeriod string  `json:"subscriptionPeriod"`
	Percentage         float64 `json:"percentage"`
}

// MessagesUserInteractionStatsResponse type
type MessagesUserInteractionStatsResponse struct {
	Overview OverviewDetail  `json:"overview"`
	Messages []MessageDetail `json:"messages"`
	Clicks   []ClickDetail   `json:"clicks"`
}

// OverviewDetail type
type OverviewDetail struct {
	RequestID                   string `json:"requestId"`
	Timestamp                   int64  `json:"timestamp"`
	Delivered                   int64  `json:"delivered"`
	UniqueImpression            int64  `json:"uniqueImpression"`
	UniqueClick                 int64  `json:"uniqueClick"`
	UniqueMediaPlayed           int64  `json:"uniqueMediaPlayed"`
	UniqueMediaPlayed100Percent int64  `json:"uniqueMediaPlayed100Percent"`
}

// MessageDetail type
type MessageDetail struct {
	Seq                         int64 `json:"seq"`
	Impression                  int64 `json:"impression"`
	MediaPlayed                 int64 `json:"mediaPlayed"`
	MediaPlayed25Percent        int64 `json:"mediaPlayed25Percent"`
	MediaPlayed50Percent        int64 `json:"mediaPlayed50Percent"`
	MediaPlayed75Percent        int64 `json:"mediaPlayed75Percent"`
	MediaPlayed100Percent       int64 `json:"mediaPlayed100Percent"`
	UniqueMediaPlayed           int64 `json:"uniqueMediaPlayed"`
	UniqueMediaPlayed25Percent  int64 `json:"uniqueMediaPlayed25Percent"`
	UniqueMediaPlayed50Percent  int64 `json:"uniqueMediaPlayed50Percent"`
	UniqueMediaPlayed75Percent  int64 `json:"uniqueMediaPlayed75Percent"`
	UniqueMediaPlayed100Percent int64 `json:"uniqueMediaPlayed100Percent"`
}

// ClickDetail type
type ClickDetail struct {
	Seq                  int64  `json:"seq"`
	URL                  string `json:"url"`
	Click                int64  `json:"click"`
	UniqueClick          int64  `json:"uniqueClick"`
	UniqueClickOfRequest int64  `json:"uniqueClickOfRequest"`
}

// RichMenuIDResponse type
type RichMenuIDResponse struct {
	RichMenuID string `json:"richMenuId"`
}

// RichMenuResponse type
type RichMenuResponse struct {
	RichMenuID  string       `json:"richMenuId"`
	Size        RichMenuSize `json:"size"`
	Selected    bool         `json:"selected"`
	Name        string       `json:"name"`
	ChatBarText string       `json:"chatBarText"`
	Areas       []AreaDetail `json:"areas"`
}

// LIFFAppsResponse type
type LIFFAppsResponse struct {
	Apps []LIFFApp `json:"apps"`
}

// LIFFIDResponse type
type LIFFIDResponse struct {
	LIFFID string `json:"liffId"`
}

// LinkTokenResponse type
type LinkTokenResponse struct {
	LinkToken string `json:"linkToken"`
}

// WebhookInfoResponse type
type WebhookInfoResponse struct {
	Endpoint string `json:"endpoint"`
	Active   string `json:"active"`
}

// isSuccess checks if status code is 2xx: The action was successfully received,
// understood, and accepted.
func isSuccess(code int) bool {
	return code/100 == 2
}

// AccessTokenResponse type
type AccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
	TokenType   string `json:"token_type"`
	KeyID       string `json:"key_id"`
}

// AccessTokensResponse type
type AccessTokensResponse struct {
	KeyIDs []string `json:"kids"`
}

// TestWebhookResponse type
type TestWebhookResponse struct {
	Success    bool      `json:"success"`
	Timestamp  time.Time `json:"timestamp"`
	StatusCode int       `json:"statusCode"`
	Reason     string    `json:"reason"`
	Detail     string    `json:"detail"`
}

func checkResponse(res *http.Response) error {
	if isSuccess(res.StatusCode) {
		return nil
	}
	decoder := json.NewDecoder(res.Body)
	result := ErrorResponse{}
	if err := decoder.Decode(&result); err != nil {
		return &APIError{
			Code: res.StatusCode,
		}
	}
	return &APIError{
		Code:     res.StatusCode,
		Response: &result,
	}
}

func decodeToBasicResponse(res *http.Response) (*BasicResponse, error) {
	if err := checkResponse(res); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(res.Body)
	result := BasicResponse{
		RequestID: res.Header.Get("X-Line-Request-Id"),
	}
	if err := decoder.Decode(&result); err != nil {
		if err == io.EOF {
			return &result, nil
		}
		return nil, err
	}
	return &result, nil
}

func decodeToUserProfileResponse(res *http.Response) (*UserProfileResponse, error) {
	if err := checkResponse(res); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(res.Body)
	result := UserProfileResponse{}
	if err := decoder.Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func decodeToGroupSummaryResponse(res *http.Response) (*GroupSummaryResponse, error) {
	if err := checkResponse(res); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(res.Body)
	result := &GroupSummaryResponse{}
	if err := decoder.Decode(result); err != nil {
		return nil, err
	}
	return result, nil
}

func decodeToMemberIDsResponse(res *http.Response) (*MemberIDsResponse, error) {
	if err := checkResponse(res); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(res.Body)
	result := &MemberIDsResponse{}
	if err := decoder.Decode(result); err != nil {
		return nil, err
	}
	return result, nil
}

func decodeToMemberCountResponse(res *http.Response) (*MemberCountResponse, error) {
	if err := checkResponse(res); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(res.Body)
	result := &MemberCountResponse{}
	if err := decoder.Decode(result); err != nil {
		return nil, err
	}
	return result, nil
}

func decodeToMessageContentResponse(res *http.Response) (*MessageContentResponse, error) {
	if err := checkResponse(res); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, res.Body); err != nil {
		return nil, err
	}
	result := MessageContentResponse{
		Content:       ioutil.NopCloser(&buf),
		ContentType:   res.Header.Get("Content-Type"),
		ContentLength: res.ContentLength,
	}
	return &result, nil
}

func decodeToMessageQuotaResponse(res *http.Response) (*MessageQuotaResponse, error) {
	if err := checkResponse(res); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(res.Body)
	result := &MessageQuotaResponse{}
	if err := decoder.Decode(result); err != nil {
		return nil, err
	}
	return result, nil
}

func decodeToMessageConsumptionResponse(res *http.Response) (*MessageConsumptionResponse, error) {
	if err := checkResponse(res); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(res.Body)
	result := &MessageConsumptionResponse{}
	if err := decoder.Decode(result); err != nil {
		return nil, err
	}
	return result, nil
}

func decodeToBotInfoResponse(res *http.Response) (*BotInfoResponse, error) {
	if err := checkResponse(res); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(res.Body)
	result := &BotInfoResponse{}
	if err := decoder.Decode(result); err != nil {
		return nil, err
	}
	return result, nil
}

func decodeToRichMenuResponse(res *http.Response) (*RichMenuResponse, error) {
	if err := checkResponse(res); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(res.Body)
	result := RichMenuResponse{}
	if err := decoder.Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func decodeToRichMenuListResponse(res *http.Response) ([]*RichMenuResponse, error) {
	if err := checkResponse(res); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(res.Body)
	var result = struct {
		RichMenus []*RichMenuResponse `json:"richmenus"`
	}{}
	if err := decoder.Decode(&result); err != nil {
		return nil, err
	}
	return result.RichMenus, nil
}

func decodeToRichMenuIDResponse(res *http.Response) (*RichMenuIDResponse, error) {
	if err := checkResponse(res); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(res.Body)
	result := RichMenuIDResponse{}
	if err := decoder.Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func decodeToLIFFResponse(res *http.Response) (*LIFFAppsResponse, error) {
	if err := checkResponse(res); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(res.Body)
	result := &LIFFAppsResponse{}
	if err := decoder.Decode(result); err != nil {
		return nil, err
	}
	return result, nil
}

func decodeToLIFFIDResponse(res *http.Response) (*LIFFIDResponse, error) {
	if err := checkResponse(res); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(res.Body)
	result := LIFFIDResponse{}
	if err := decoder.Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func decodeToLinkTokenResponse(res *http.Response) (*LinkTokenResponse, error) {
	if err := checkResponse(res); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(res.Body)
	result := LinkTokenResponse{}
	if err := decoder.Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func decodeToWebhookInfoResponse(res *http.Response) (*WebhookInfoResponse, error) {
	if err := checkResponse(res); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(res.Body)
	result := WebhookInfoResponse{}
	if err := decoder.Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func decodeToMessagesNumberResponse(res *http.Response) (*MessagesNumberResponse, error) {
	if err := checkResponse(res); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(res.Body)
	result := MessagesNumberResponse{}
	if err := decoder.Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func decodeToMessagesNumberDeliveryResponse(res *http.Response) (*MessagesNumberDeliveryResponse, error) {
	if err := checkResponse(res); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(res.Body)
	result := MessagesNumberDeliveryResponse{}
	if err := decoder.Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func decodeToMessagesNumberFollowersResponse(res *http.Response) (*MessagesNumberFollowersResponse, error) {
	if err := checkResponse(res); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(res.Body)
	result := MessagesNumberFollowersResponse{}
	if err := decoder.Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func decodeToMessagesFriendDemographicsResponse(res *http.Response) (*MessagesFriendDemographicsResponse, error) {
	if err := checkResponse(res); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(res.Body)
	result := MessagesFriendDemographicsResponse{}
	if err := decoder.Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func decodeToMessagesUserInteractionStatsResponse(res *http.Response) (*MessagesUserInteractionStatsResponse, error) {
	if err := checkResponse(res); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(res.Body)
	result := MessagesUserInteractionStatsResponse{}
	if err := decoder.Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func decodeToMessagesProgressResponse(res *http.Response) (*MessagesProgressResponse, error) {
	if err := checkResponse(res); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(res.Body)
	result := MessagesProgressResponse{}
	if err := decoder.Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func decodeToAccessTokenResponse(res *http.Response) (*AccessTokenResponse, error) {
	if err := checkResponse(res); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(res.Body)
	result := AccessTokenResponse{}
	if err := decoder.Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func decodeToAccessTokensResponse(res *http.Response) (*AccessTokensResponse, error) {
	if err := checkResponse(res); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(res.Body)
	result := AccessTokensResponse{}
	if err := decoder.Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func decodeToTestWebhookResponsee(res *http.Response) (*TestWebhookResponse, error) {
	if err := checkResponse(res); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(res.Body)
	result := TestWebhookResponse{}
	if err := decoder.Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}
