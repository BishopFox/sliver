package viber

import "encoding/json"

/*
{
	"receiver": "01234567890A=",
	"min_api_version": 1,
	"sender": {
		"name": "John McClane",
		"avatar": "http://avatar.example.com"
	},
	"tracking_data": "tracking data",
	"type": "text",
	"text": "a message from pa"
}
*/

type messageResponse struct {
	Status        int    `json:"status"`
	StatusMessage string `json:"status_message"`
	MessageToken  uint64 `json:"message_token"`
}

// Message interface for all types of viber messages
type Message interface {
	SetReceiver(r string)
	SetFrom(from string)
	SetKeyboard(k *Keyboard)
}

// TextMessage for Viber
type TextMessage struct {
	Receiver      string      `json:"receiver,omitempty"`
	From          string      `json:"from,omitempty"`
	MinAPIVersion uint        `json:"min_api_version,omitempty"`
	Sender        Sender      `json:"sender"`
	Type          MessageType `json:"type"`
	TrackingData  string      `json:"tracking_data,omitempty"`
	Text          string      `json:"text"`
	Keyboars      *Keyboard   `json:"keyboard,omitempty"`
	//    "media": "http://www.images.com/img.jpg",
	//    "thumbnail": "http://www.images.com/thumb.jpg"
	// 	"size": 10000,
	// 	"duration": 10
}

// URLMessage structure
type URLMessage struct {
	TextMessage
	Media string `json:"media"`
}

// PictureMessage structure
type PictureMessage struct {
	TextMessage
	Media     string `json:"media"`
	Thumbnail string `json:"thumbnail,omitempty"`
}

// VideoMessage structure
type VideoMessage struct {
	TextMessage
	Media     string `json:"media"`
	Thumbnail string `json:"thumbnail,omitempty"`
	Size      uint   `json:"size"`
	Duration  uint   `json:"duration,omitempty"`
}

// MessageType for viber messaging
type MessageType string

// Message types
const (
	TypeTextMessage      = MessageType("text")
	TypeURLMessage       = MessageType("url")
	TypePictureMessage   = MessageType("picture")
	TypeVideoMessage     = MessageType("video")
	TypeFileMessage      = MessageType("file")
	TypeLocationMessage  = MessageType("location")
	TypeContactMessage   = MessageType("contact")
	TypeStickerMessage   = MessageType("sticker")
	TypeRichMediaMessage = MessageType("rich_media")
)

//video, file, location, contact, sticker, carousel content

func parseMsgResponse(b []byte) (msgToken uint64, err error) {
	var resp messageResponse
	if err := json.Unmarshal(b, &resp); err != nil {
		return 0, err
	}

	if resp.Status != 0 {
		return resp.MessageToken, Error{Status: resp.Status, StatusMessage: resp.StatusMessage}
	}

	return resp.MessageToken, nil
}

func (v *Viber) sendMessage(url string, m interface{}) (msgToken uint64, err error) {
	b, err := v.PostData(url, m)
	if err != nil {
		return 0, err
	}
	return parseMsgResponse(b)
}

// NewTextMessage viber
func (v *Viber) NewTextMessage(msg string) *TextMessage {
	return &TextMessage{
		Sender: v.Sender,
		Type:   TypeTextMessage,
		Text:   msg,
	}
}

// NewURLMessage creates new message with global sender and common params set
func (v *Viber) NewURLMessage(msg string, url string) *URLMessage {
	return &URLMessage{
		TextMessage: TextMessage{
			Sender: v.Sender,
			Type:   TypeURLMessage,
			Text:   msg,
		},
		Media: url,
	}
}

// NewPictureMessage for viber
func (v *Viber) NewPictureMessage(msg string, url string, thumbURL string) *PictureMessage {
	return &PictureMessage{
		TextMessage: TextMessage{
			Sender: v.Sender,
			Type:   TypePictureMessage,
			Text:   msg,
		},
		Media:     url,
		Thumbnail: thumbURL,
	}
}

// SendTextMessage to reciever, returns message token
func (v *Viber) SendTextMessage(receiver string, msg string) (msgToken uint64, err error) {
	return v.SendMessage(receiver, v.NewTextMessage(msg))
}

// SendURLMessage to easily send url messages as global sender
func (v *Viber) SendURLMessage(receiver string, msg string, url string) (msgToken uint64, err error) {
	return v.SendMessage(receiver, v.NewURLMessage(msg, url))
}

// SendPictureMessage to receiver, returns message token
func (v *Viber) SendPictureMessage(receiver string, msg string, url string, thumbURL string) (token uint64, err error) {
	return v.SendMessage(receiver, v.NewPictureMessage(msg, url, thumbURL))
}

// SendPublicMessage from public account
func (v *Viber) SendPublicMessage(from string, m Message) (msgToken uint64, err error) {
	// text, picture, video, file, location, contact, sticker and url
	m.SetFrom(from)
	return v.sendMessage("https://chatapi.viber.com/pa/post", m)
}

// SendMessage to receiver
func (v *Viber) SendMessage(to string, m Message) (msgToken uint64, err error) {
	m.SetReceiver(to)
	return v.sendMessage("https://chatapi.viber.com/pa/send_message", m)
}

// SetReceiver for text message
func (m *TextMessage) SetReceiver(r string) {
	m.Receiver = r
}

// SetFrom to text message for public account message
func (m *TextMessage) SetFrom(from string) {
	m.From = from
}

// SetKeyboard for text message
func (m *TextMessage) SetKeyboard(k *Keyboard) {
	m.Keyboars = k
}
