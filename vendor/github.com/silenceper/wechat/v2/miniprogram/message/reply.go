package message

import "errors"

// ErrInvalidReply 无效的回复
var ErrInvalidReply = errors.New("无效的回复信息")

// ErrUnsupportedReply 不支持的回复类型
var ErrUnsupportedReply = errors.New("不支持的回复消息")

// Reply 消息回复
type Reply struct {
	MsgType MsgType
	MsgData interface{}
}
