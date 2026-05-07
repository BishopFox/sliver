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
	"encoding/hex"
	"encoding/json"
	"time"
)

// EventType type
type EventType string

// EventType constants
const (
	EventTypeMessage           EventType = "message"
	EventTypeFollow            EventType = "follow"
	EventTypeUnfollow          EventType = "unfollow"
	EventTypeJoin              EventType = "join"
	EventTypeLeave             EventType = "leave"
	EventTypeMemberJoined      EventType = "memberJoined"
	EventTypeMemberLeft        EventType = "memberLeft"
	EventTypePostback          EventType = "postback"
	EventTypeBeacon            EventType = "beacon"
	EventTypeAccountLink       EventType = "accountLink"
	EventTypeThings            EventType = "things"
	EventTypeUnsend            EventType = "unsend"
	EventTypeVideoPlayComplete EventType = "videoPlayComplete"
)

// EventMode type
type EventMode string

// EventMode constants
const (
	EventModeActive  EventMode = "active"
	EventModeStandby EventMode = "standby"
)

// EventSourceType type
type EventSourceType string

// EventSourceType constants
const (
	EventSourceTypeUser  EventSourceType = "user"
	EventSourceTypeGroup EventSourceType = "group"
	EventSourceTypeRoom  EventSourceType = "room"
)

// EventSource type
type EventSource struct {
	Type    EventSourceType `json:"type"`
	UserID  string          `json:"userId,omitempty"`
	GroupID string          `json:"groupId,omitempty"`
	RoomID  string          `json:"roomId,omitempty"`
}

// Params type
type Params struct {
	Date     string `json:"date,omitempty"`
	Time     string `json:"time,omitempty"`
	Datetime string `json:"datetime,omitempty"`
}

// Members type
type Members struct {
	Members []EventSource `json:"members"`
}

// Postback type
type Postback struct {
	Data   string  `json:"data"`
	Params *Params `json:"params,omitempty"`
}

// BeaconEventType type
type BeaconEventType string

// BeaconEventType constants
const (
	BeaconEventTypeEnter  BeaconEventType = "enter"
	BeaconEventTypeLeave  BeaconEventType = "leave"
	BeaconEventTypeBanner BeaconEventType = "banner"
	BeaconEventTypeStay   BeaconEventType = "stay"
)

// Beacon type
type Beacon struct {
	Hwid          string
	Type          BeaconEventType
	DeviceMessage []byte
}

// AccountLinkResult type
type AccountLinkResult string

// AccountLinkResult constants
const (
	AccountLinkResultOK     AccountLinkResult = "ok"
	AccountLinkResultFailed AccountLinkResult = "failed"
)

// AccountLink type
type AccountLink struct {
	Result AccountLinkResult
	Nonce  string
}

// ThingsResult type
type ThingsResult struct {
	ScenarioID             string
	Revision               int
	StartTime              int64
	EndTime                int64
	ResultCode             ThingsResultCode
	ActionResults          []*ThingsActionResult
	BLENotificationPayload []byte
	ErrorReason            string
}

// ThingsResultCode type
type ThingsResultCode string

// ThingsResultCode constsnts
const (
	ThingsResultCodeSuccess      ThingsResultCode = "success"
	ThingsResultCodeGattError    ThingsResultCode = "gatt_error"
	ThingsResultCodeRuntimeError ThingsResultCode = "runtime_error"
)

// ThingsActionResult type
type ThingsActionResult struct {
	Type ThingsActionResultType
	Data []byte
}

// ThingsActionResultType type
type ThingsActionResultType string

// ThingsActionResultType contants
const (
	ThingsActionResultTypeBinary ThingsActionResultType = "binary"
	ThingsActionResultTypeVoid   ThingsActionResultType = "void"
)

// Things type
type Things struct {
	DeviceID string
	Type     string
	Result   *ThingsResult
}

// Unsend type
type Unsend struct {
	MessageID string `json:"messageId"`
}

// VideoPlayComplete type
type VideoPlayComplete struct {
	TrackingID string `json:"trackingId"`
}

// StickerResourceType type
type StickerResourceType string

// StickerResourceType constants
const (
	StickerResourceTypeStatic         StickerResourceType = "STATIC"
	StickerResourceTypeAnimation      StickerResourceType = "ANIMATION"
	StickerResourceTypeSound          StickerResourceType = "SOUND"
	StickerResourceTypeAnimationSound StickerResourceType = "ANIMATION_SOUND"
	StickerResourceTypePerStickerText StickerResourceType = "PER_STICKER_TEXT"
	StickerResourceTypePopup          StickerResourceType = "POPUP"
	StickerResourceTypePopupSound     StickerResourceType = "POPUP_SOUND"
	StickerResourceTypeNameText       StickerResourceType = "NAME_TEXT"
)

// Event type
type Event struct {
	ReplyToken        string
	Type              EventType
	Mode              EventMode
	Timestamp         time.Time
	Source            *EventSource
	Message           Message
	Joined            *Members
	Left              *Members
	Postback          *Postback
	Beacon            *Beacon
	AccountLink       *AccountLink
	Things            *Things
	Members           []*EventSource
	Unsend            *Unsend
	VideoPlayComplete *VideoPlayComplete
}

type rawEvent struct {
	ReplyToken        string               `json:"replyToken,omitempty"`
	Type              EventType            `json:"type"`
	Mode              EventMode            `json:"mode"`
	Timestamp         int64                `json:"timestamp"`
	Source            *EventSource         `json:"source"`
	Message           *rawEventMessage     `json:"message,omitempty"`
	Postback          *Postback            `json:"postback,omitempty"`
	Beacon            *rawBeaconEvent      `json:"beacon,omitempty"`
	AccountLink       *rawAccountLinkEvent `json:"link,omitempty"`
	Joined            *rawMemberEvent      `json:"joined,omitempty"`
	Left              *rawMemberEvent      `json:"left,omitempty"`
	Things            *rawThingsEvent      `json:"things,omitempty"`
	Unsend            *Unsend              `json:"unsend,omitempty"`
	VideoPlayComplete *VideoPlayComplete   `json:"videoPlayComplete,omitempty"`
}

type rawMemberEvent struct {
	Members []*EventSource `json:"members"`
}

type rawEventMessage struct {
	ID                  string              `json:"id"`
	Type                MessageType         `json:"type"`
	Text                string              `json:"text,omitempty"`
	Duration            int                 `json:"duration,omitempty"`
	Title               string              `json:"title,omitempty"`
	Address             string              `json:"address,omitempty"`
	FileName            string              `json:"fileName,omitempty"`
	FileSize            int                 `json:"fileSize,omitempty"`
	Latitude            float64             `json:"latitude,omitempty"`
	Longitude           float64             `json:"longitude,omitempty"`
	PackageID           string              `json:"packageId,omitempty"`
	StickerID           string              `json:"stickerId,omitempty"`
	StickerResourceType StickerResourceType `json:"stickerResourceType,omitempty"`
	Keywords            []string            `json:"keywords,omitempty"`
	Emojis              []*Emoji            `json:"emojis,omitempty"`
	Mention             *Mention            `json:"mention,omitempty"`
}

type rawBeaconEvent struct {
	Hwid string          `json:"hwid"`
	Type BeaconEventType `json:"type"`
	DM   string          `json:"dm,omitempty"`
}

type rawAccountLinkEvent struct {
	Result AccountLinkResult `json:"result"`
	Nonce  string            `json:"nonce"`
}

type rawThingsResult struct {
	ScenarioID             string                   `json:"scenarioId"`
	Revision               int                      `json:"revision"`
	StartTime              int64                    `json:"startTime"`
	EndTime                int64                    `json:"endTime"`
	ResultCode             ThingsResultCode         `json:"resultCode"`
	ActionResults          []*rawThingsActionResult `json:"actionResults"`
	BLENotificationPayload string                   `json:"bleNotificationPayload,omitempty"`
	ErrorReason            string                   `json:"errorReason,omitempty"`
}

type rawThingsActionResult struct {
	Type ThingsActionResultType `json:"type,omitempty"`
	Data string                 `json:"data,omitempty"`
}

type rawThingsEvent struct {
	DeviceID string           `json:"deviceId"`
	Type     string           `json:"type"`
	Result   *rawThingsResult `json:"result,omitempty"`
}

const (
	millisecPerSec     = int64(time.Second / time.Millisecond)
	nanosecPerMillisec = int64(time.Millisecond / time.Nanosecond)
)

// MarshalJSON method of Event
func (e *Event) MarshalJSON() ([]byte, error) {
	raw := rawEvent{
		ReplyToken:        e.ReplyToken,
		Type:              e.Type,
		Mode:              e.Mode,
		Timestamp:         e.Timestamp.Unix()*millisecPerSec + int64(e.Timestamp.Nanosecond())/int64(time.Millisecond),
		Source:            e.Source,
		Postback:          e.Postback,
		Unsend:            e.Unsend,
		VideoPlayComplete: e.VideoPlayComplete,
	}
	if e.Beacon != nil {
		raw.Beacon = &rawBeaconEvent{
			Hwid: e.Beacon.Hwid,
			Type: e.Beacon.Type,
			DM:   hex.EncodeToString(e.Beacon.DeviceMessage),
		}
	}
	if e.AccountLink != nil {
		raw.AccountLink = &rawAccountLinkEvent{
			Result: e.AccountLink.Result,
			Nonce:  e.AccountLink.Nonce,
		}
	}

	switch e.Type {
	case EventTypeMemberJoined:
		raw.Joined = &rawMemberEvent{
			Members: e.Members,
		}
	case EventTypeMemberLeft:
		raw.Left = &rawMemberEvent{
			Members: e.Members,
		}
	case EventTypeThings:
		raw.Things = &rawThingsEvent{
			DeviceID: e.Things.DeviceID,
			Type:     e.Things.Type,
		}
		if e.Things.Result != nil {
			raw.Things.Result = &rawThingsResult{
				ScenarioID: e.Things.Result.ScenarioID,
				Revision:   e.Things.Result.Revision,
				StartTime:  e.Things.Result.StartTime,
				EndTime:    e.Things.Result.EndTime,
				ResultCode: e.Things.Result.ResultCode,

				BLENotificationPayload: string(e.Things.Result.BLENotificationPayload),
				ErrorReason:            e.Things.Result.ErrorReason,
			}
			if e.Things.Result.ActionResults != nil {
				raw.Things.Result.ActionResults = make([]*rawThingsActionResult, len(e.Things.Result.ActionResults))
			}
			for i := range e.Things.Result.ActionResults {
				raw.Things.Result.ActionResults[i] = &rawThingsActionResult{
					Type: e.Things.Result.ActionResults[i].Type,
					Data: string(e.Things.Result.ActionResults[i].Data),
				}
			}
		}
	}

	switch m := e.Message.(type) {
	case *TextMessage:
		raw.Message = &rawEventMessage{
			Type:    MessageTypeText,
			ID:      m.ID,
			Text:    m.Text,
			Emojis:  m.Emojis,
			Mention: m.Mention,
		}
	case *ImageMessage:
		raw.Message = &rawEventMessage{
			Type: MessageTypeImage,
			ID:   m.ID,
		}
	case *VideoMessage:
		raw.Message = &rawEventMessage{
			Type: MessageTypeVideo,
			ID:   m.ID,
		}
	case *AudioMessage:
		raw.Message = &rawEventMessage{
			Type:     MessageTypeAudio,
			ID:       m.ID,
			Duration: m.Duration,
		}
	case *FileMessage:
		raw.Message = &rawEventMessage{
			Type:     MessageTypeFile,
			ID:       m.ID,
			FileName: m.FileName,
			FileSize: m.FileSize,
		}
	case *LocationMessage:
		raw.Message = &rawEventMessage{
			Type:      MessageTypeLocation,
			ID:        m.ID,
			Title:     m.Title,
			Address:   m.Address,
			Latitude:  m.Latitude,
			Longitude: m.Longitude,
		}
	case *StickerMessage:
		raw.Message = &rawEventMessage{
			Type:                MessageTypeSticker,
			ID:                  m.ID,
			PackageID:           m.PackageID,
			StickerID:           m.StickerID,
			StickerResourceType: m.StickerResourceType,
			Keywords:            m.Keywords,
		}
	}
	return json.Marshal(&raw)
}

// UnmarshalJSON method of Event
func (e *Event) UnmarshalJSON(body []byte) (err error) {
	rawEvent := rawEvent{}
	if err = json.Unmarshal(body, &rawEvent); err != nil {
		return
	}

	e.ReplyToken = rawEvent.ReplyToken
	e.Type = rawEvent.Type
	e.Mode = rawEvent.Mode
	e.Timestamp = time.Unix(rawEvent.Timestamp/millisecPerSec, (rawEvent.Timestamp%millisecPerSec)*nanosecPerMillisec).UTC()
	e.Source = rawEvent.Source

	switch rawEvent.Type {
	case EventTypeMessage:
		switch rawEvent.Message.Type {
		case MessageTypeText:
			e.Message = &TextMessage{
				ID:      rawEvent.Message.ID,
				Text:    rawEvent.Message.Text,
				Emojis:  rawEvent.Message.Emojis,
				Mention: rawEvent.Message.Mention,
			}
		case MessageTypeImage:
			e.Message = &ImageMessage{
				ID: rawEvent.Message.ID,
			}
		case MessageTypeVideo:
			e.Message = &VideoMessage{
				ID: rawEvent.Message.ID,
			}
		case MessageTypeAudio:
			e.Message = &AudioMessage{
				ID:       rawEvent.Message.ID,
				Duration: rawEvent.Message.Duration,
			}
		case MessageTypeFile:
			e.Message = &FileMessage{
				ID:       rawEvent.Message.ID,
				FileName: rawEvent.Message.FileName,
				FileSize: rawEvent.Message.FileSize,
			}
		case MessageTypeLocation:
			e.Message = &LocationMessage{
				ID:        rawEvent.Message.ID,
				Title:     rawEvent.Message.Title,
				Address:   rawEvent.Message.Address,
				Latitude:  rawEvent.Message.Latitude,
				Longitude: rawEvent.Message.Longitude,
			}
		case MessageTypeSticker:
			e.Message = &StickerMessage{
				ID:                  rawEvent.Message.ID,
				PackageID:           rawEvent.Message.PackageID,
				StickerID:           rawEvent.Message.StickerID,
				StickerResourceType: rawEvent.Message.StickerResourceType,
				Keywords:            rawEvent.Message.Keywords,
			}
		}
	case EventTypePostback:
		e.Postback = rawEvent.Postback
	case EventTypeBeacon:
		var deviceMessage []byte
		deviceMessage, err = hex.DecodeString(rawEvent.Beacon.DM)
		if err != nil {
			return
		}
		e.Beacon = &Beacon{
			Hwid:          rawEvent.Beacon.Hwid,
			Type:          rawEvent.Beacon.Type,
			DeviceMessage: deviceMessage,
		}
	case EventTypeAccountLink:
		e.AccountLink = &AccountLink{
			Result: rawEvent.AccountLink.Result,
			Nonce:  rawEvent.AccountLink.Nonce,
		}
	case EventTypeMemberJoined:
		e.Members = rawEvent.Joined.Members
	case EventTypeMemberLeft:
		e.Members = rawEvent.Left.Members
	case EventTypeThings:
		e.Things = &Things{
			Type:     rawEvent.Things.Type,
			DeviceID: rawEvent.Things.DeviceID,
		}
		if rawEvent.Things.Result != nil {
			rawResult := rawEvent.Things.Result
			e.Things.Result = &ThingsResult{
				ScenarioID:             rawResult.ScenarioID,
				Revision:               rawResult.Revision,
				StartTime:              rawResult.StartTime,
				EndTime:                rawResult.EndTime,
				ResultCode:             rawResult.ResultCode,
				ActionResults:          make([]*ThingsActionResult, len(rawResult.ActionResults)),
				BLENotificationPayload: []byte(rawResult.BLENotificationPayload),
				ErrorReason:            rawResult.ErrorReason,
			}
			for i := range rawResult.ActionResults {
				e.Things.Result.ActionResults[i] = &ThingsActionResult{
					Type: rawResult.ActionResults[i].Type,
					Data: []byte(rawResult.ActionResults[i].Data),
				}
			}
		}
	case EventTypeUnsend:
		e.Unsend = rawEvent.Unsend
	case EventTypeVideoPlayComplete:
		e.VideoPlayComplete = rawEvent.VideoPlayComplete
	}
	return
}
