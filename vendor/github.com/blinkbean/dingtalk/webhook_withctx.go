package dingtalk

import "context"

func (d *DingTalk) SendTextMessageWithCtx(ctx context.Context, content string, opt ...atOption) error {
	content = content + d.keyWord
	return d.sendMessage(ctx, NewTextMsg(content, opt...))
}

func (d *DingTalk) SendMarkDownMessageWithCtx(ctx context.Context, title, text string, opts ...atOption) error {
	title = title + d.keyWord
	return d.sendMessage(ctx, NewMarkDownMsg(title, text, opts...))
}

// SendDTMDMessageWithCtx 利用dtmd发送点击消息
func (d *DingTalk) SendDTMDMessageWithCtx(ctx context.Context, title string, dtmdMap *dingMap, opt ...atOption) error {
	title = title + d.keyWord
	return d.sendMessage(ctx, NewDTMDMsg(title, dtmdMap, opt...))
}

func (d *DingTalk) SendMarkDownMessageBySliceWithCtx(ctx context.Context, title string, textList []string, opts ...atOption) error {
	title = title + d.keyWord
	text := ""
	for _, t := range textList {
		text = text + "\n" + t
	}
	return d.sendMessage(ctx, NewMarkDownMsg(title, text, opts...))
}

func (d *DingTalk) SendLinkMessageWithCtx(ctx context.Context, title, text, picUrl, msgUrl string) error {
	title = title + d.keyWord
	return d.sendMessage(ctx, NewLinkMsg(title, text, picUrl, msgUrl))
}

func (d *DingTalk) SendActionCardMessageWithCtx(ctx context.Context, title, text string, opts ...actionCardOption) error {
	title = title + d.keyWord
	return d.sendMessage(ctx, NewActionCardMsg(title, text, opts...))
}

func (d *DingTalk) SendActionCardMessageBySliceWithCtx(ctx context.Context, title string, textList []string, opts ...actionCardOption) error {
	title = title + d.keyWord
	text := ""
	for _, t := range textList {
		text = text + "\n" + t
	}
	return d.sendMessage(ctx, NewActionCardMsg(title, text, opts...))
}

func (d *DingTalk) SendFeedCardMessageWithCtx(ctx context.Context, feedCard []FeedCardLinkModel) error {
	if len(feedCard) > 0 {
		feedCard[0].Title = feedCard[0].Title + d.keyWord
	}
	return d.sendMessage(ctx, NewFeedCardMsg(feedCard))
}
