package robot

import (
	"encoding/json"
	"fmt"

	"github.com/silenceper/wechat/v2/util"
)

const (
	// webhookSendURL 机器人发送群组消息
	webhookSendURL = "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=%s"
)

// RobotBroadcast 群机器人消息发送
// @see https://developer.work.weixin.qq.com/document/path/91770
func (r *Client) RobotBroadcast(webhookKey string, options interface{}) (info util.CommonError, err error) {
	var data []byte
	if data, err = util.PostJSON(fmt.Sprintf(webhookSendURL, webhookKey), options); err != nil {
		return
	}
	if err = json.Unmarshal(data, &info); err != nil {
		return
	}
	if info.ErrCode != 0 {
		return info, err
	}
	return info, nil
}
