package viber

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// Sender structure
type Sender struct {
	Name   string `json:"name"`
	Avatar string `json:"avatar,omitempty"`
}

type event struct {
	Event        string    `json:"event"`
	Timestamp    Timestamp `json:"timestamp"`
	MessageToken uint64    `json:"message_token,omitempty"`
	UserID       string    `json:"user_id,omitempty"`

	// failed event
	Descr string `json:"descr,omitempty"`

	//conversation_started event
	Type       string          `json:"type,omitempty"`
	Context    string          `json:"context,omitempty"`
	Subscribed bool            `json:"subscribed,omitempty"`
	User       json.RawMessage `json:"user,omitempty"`

	// message event
	Sender  json.RawMessage `json:"sender,omitempty"`
	Message json.RawMessage `json:"message,omitempty"`
}

// Viber app
type Viber struct {
	AppKey string
	Sender Sender

	// event methods
	ConversationStarted func(v *Viber, u User, conversationType, context string, subscribed bool, token uint64, t time.Time) Message
	Message             func(v *Viber, u User, m Message, token uint64, t time.Time)
	Subscribed          func(v *Viber, u User, token uint64, t time.Time)
	Unsubscribed        func(v *Viber, userID string, token uint64, t time.Time)
	Delivered           func(v *Viber, userID string, token uint64, t time.Time)
	Seen                func(v *Viber, userID string, token uint64, t time.Time)
	Failed              func(v *Viber, userID string, token uint64, descr string, t time.Time)

	// client for sending messages
	client *http.Client
}

var (
	// Log errors, set to logger if you want to log package activities and errors
	Log               = log.New(ioutil.Discard, "Viber >>", 0)
	regexpPeekMsgType = regexp.MustCompile("\"type\":\\s*\"([^\"]+)\"")
)

// New returns Viber app with specified app key and default sender
// You can also create *VIber{} struct directly
func New(appKey, senderName, senderAvatar string) *Viber {
	return &Viber{
		AppKey: appKey,
		Sender: Sender{
			Name:   senderName,
			Avatar: senderAvatar,
		},
		client: &http.Client{},
	}
}

// ServeHTTP
// https://developers.viber.com/docs/api/rest-bot-api/#callbacks
func (v *Viber) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	body, err := ioutil.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		Log.Println(err)
		return
	}

	Log.Println("Received from Viber:", string(body))

	if !v.checkHMAC(body, r.Header.Get("X-Viber-Content-Signature")) {
		Log.Println("X-Viber-Content-Signature doesn't match")
		return
	}

	var e event
	if err := json.Unmarshal(body, &e); err != nil {
		Log.Println(err)
		return
	}

	switch e.Event {
	case "subscribed":
		if v.Subscribed != nil {
			var u User
			if err := json.Unmarshal(e.User, &u); err != nil {
				Log.Println(err)
				return
			}
			go v.Subscribed(v, u, e.MessageToken, e.Timestamp.Time)
		}

	case "unsubscribed":
		if v.Unsubscribed != nil {
			go v.Unsubscribed(v, e.UserID, e.MessageToken, e.Timestamp.Time)
		}

	case "conversation_started":
		if v.ConversationStarted != nil {
			var u User
			if err := json.Unmarshal(e.User, &u); err != nil {
				Log.Println(err)
				return
			}
			if msg := v.ConversationStarted(v, u, e.Type, e.Context, e.Subscribed, e.MessageToken, e.Timestamp.Time); msg != nil {
				msg.SetReceiver("")
				msg.SetFrom("")
				b, _ := json.Marshal(msg)
				w.Write(b)
			}
		}

	case "delivered":
		if v.Delivered != nil {
			go v.Delivered(v, e.UserID, e.MessageToken, e.Timestamp.Time)
		}

	case "seen":
		if v.Seen != nil {
			go v.Seen(v, e.UserID, e.MessageToken, e.Timestamp.Time)
		}

	case "failed":
		if v.Failed != nil {
			go v.Failed(v, e.UserID, e.MessageToken, e.Descr, e.Timestamp.Time)
		}

	case "message":
		if v.Message != nil {
			var u User
			if err := json.Unmarshal(e.Sender, &u); err != nil {
				Log.Println(err)
				return
			}

			msgType := peakMessageType(e.Message)
			switch msgType {
			case "text":
				var m TextMessage
				if err := json.Unmarshal(e.Message, &m); err != nil {
					Log.Println(err)
					return
				}
				go v.Message(v, u, &m, e.MessageToken, e.Timestamp.Time)

			case "picture":
				var m PictureMessage
				if err := json.Unmarshal(e.Message, &m); err != nil {
					Log.Println(err)
					return
				}
				go v.Message(v, u, &m, e.MessageToken, e.Timestamp.Time)

			case "video":
				var m VideoMessage
				if err := json.Unmarshal(e.Message, &m); err != nil {
					Log.Println(err)
					return
				}
				go v.Message(v, u, &m, e.MessageToken, e.Timestamp.Time)

			case "url":
				var m URLMessage
				if err := json.Unmarshal(e.Message, &m); err != nil {
					Log.Println(err)
					return
				}
				go v.Message(v, u, &m, e.MessageToken, e.Timestamp.Time)

			case "contact":
				// TODO
			case "location":
				// TODO
			default:
				return
			}
		}
	}
}

// checkHMAC reports whether messageMAC is a valid HMAC tag for message.
func (v *Viber) checkHMAC(message []byte, messageMAC string) bool {
	hmac := hmac.New(sha256.New, []byte(v.AppKey))
	hmac.Write(message)
	return messageMAC == hex.EncodeToString(hmac.Sum(nil))
}

// peakMessageType uses regexp to determin message type for unmarshaling
func peakMessageType(b []byte) string {
	matches := regexpPeekMsgType.FindAllSubmatch(b, -1)
	if len(matches) == 0 {
		return ""
	}

	return strings.ToLower(string(matches[0][1]))
}

// SetRequestTimeout for sending messages to viber server
func (v *Viber) SetRequestTimeout(t time.Duration) {
	if v.client == nil {
		v.client = &http.Client{}
	}
	v.client.Timeout = t
}
