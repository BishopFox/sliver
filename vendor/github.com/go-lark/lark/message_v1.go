package lark

import "strconv"

// BuildOutcomingMessageReq for msg builder
func BuildOutcomingMessageReq(om OutcomingMessage) map[string]interface{} {
	params := map[string]interface{}{
		"msg_type": om.MsgType,
		"chat_id":  om.ChatID, // request must contain chat_id, even if it is empty
	}
	params[om.UIDType] = buildReceiveID(om)
	if len(om.RootID) > 0 {
		params["root_id"] = om.RootID
	}
	content := make(map[string]interface{})
	if om.Content.Text != nil {
		content["text"] = om.Content.Text.Text
	}
	if om.Content.Image != nil {
		content["image_key"] = om.Content.Image.ImageKey
	}
	if om.Content.ShareChat != nil {
		content["share_open_chat_id"] = om.Content.ShareChat.ChatID
	}
	if om.Content.Post != nil {
		content["post"] = *om.Content.Post
	}
	if om.MsgType == MsgInteractive && om.Content.Card != nil {
		params["card"] = *om.Content.Card
	}
	if len(om.Sign) > 0 {
		params["sign"] = om.Sign
		params["timestamp"] = strconv.FormatInt(om.Timestamp, 10)
	}
	params["content"] = content
	return params
}

func buildOutcomingNotification(om OutcomingMessage) map[string]interface{} {
	params := map[string]interface{}{
		"msg_type": om.MsgType,
	}
	if len(om.RootID) > 0 {
		params["root_id"] = om.RootID
	}
	content := make(map[string]interface{})
	if om.Content.Text != nil {
		content["text"] = om.Content.Text.Text
	}
	if om.Content.Image != nil {
		content["image_key"] = om.Content.Image.ImageKey
	}
	if om.Content.ShareChat != nil {
		content["share_open_chat_id"] = om.Content.ShareChat.ChatID
	}
	if om.Content.Post != nil {
		content["post"] = *om.Content.Post
	}
	if om.MsgType == MsgInteractive && om.Content.Card != nil {
		params["card"] = *om.Content.Card
	}
	if len(om.Sign) > 0 {
		params["sign"] = om.Sign
		params["timestamp"] = strconv.FormatInt(om.Timestamp, 10)
	}
	params["content"] = content
	return params
}
