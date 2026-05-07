package lark

import (
	"encoding/json"
)

// BuildMessage .
func BuildMessage(om OutcomingMessage) (*IMMessageRequest, error) {
	req := IMMessageRequest{
		MsgType:   string(om.MsgType),
		Content:   buildContent(om),
		ReceiveID: buildReceiveID(om),
	}
	if req.ReceiveID == "" {
		return nil, ErrInvalidReceiveID
	}
	if req.Content == "" {
		return nil, ErrMessageNotBuild
	}
	if om.UUID != "" {
		req.UUID = om.UUID
	}
	return &req, nil
}

func buildReplyMessage(om OutcomingMessage) (*IMMessageRequest, error) {
	req := IMMessageRequest{
		MsgType:   string(om.MsgType),
		Content:   buildContent(om),
		ReceiveID: buildReceiveID(om),
	}
	if req.Content == "" {
		return nil, ErrMessageNotBuild
	}
	if om.ReplyInThread == true {
		req.ReplyInThread = om.ReplyInThread
	}

	return &req, nil
}

func buildUpdateMessage(om OutcomingMessage) (*IMMessageRequest, error) {
	req := IMMessageRequest{
		Content: buildContent(om),
	}
	if om.MsgType != MsgInteractive {
		req.MsgType = om.MsgType
	}
	if req.Content == "" {
		return nil, ErrMessageNotBuild
	}

	return &req, nil
}

func buildContent(om OutcomingMessage) string {
	var (
		content = ""
		b       []byte
		err     error
	)
	switch om.MsgType {
	case MsgText:
		b, err = json.Marshal(om.Content.Text)
	case MsgImage:
		b, err = json.Marshal(om.Content.Image)
	case MsgFile:
		b, err = json.Marshal(om.Content.File)
	case MsgShareCard:
		b, err = json.Marshal(om.Content.ShareChat)
	case MsgShareUser:
		b, err = json.Marshal(om.Content.ShareUser)
	case MsgPost:
		b, err = json.Marshal(om.Content.Post)
	case MsgInteractive:
		if om.Content.Card != nil {
			b, err = json.Marshal(om.Content.Card)
		} else if om.Content.Template != nil {
			b, err = json.Marshal(om.Content.Template)
		}
	case MsgAudio:
		b, err = json.Marshal(om.Content.Audio)
	case MsgMedia:
		b, err = json.Marshal(om.Content.Media)
	case MsgSticker:
		b, err = json.Marshal(om.Content.Sticker)
	}
	if err != nil {
		return ""
	}
	content = string(b)

	return content
}

func buildReceiveID(om OutcomingMessage) string {
	switch om.UIDType {
	case UIDEmail:
		return om.Email
	case UIDUserID:
		return om.UserID
	case UIDOpenID:
		return om.OpenID
	case UIDChatID:
		return om.ChatID
	case UIDUnionID:
		return om.UnionID
	}
	return ""
}
