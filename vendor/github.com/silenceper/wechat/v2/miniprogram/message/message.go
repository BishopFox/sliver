package message

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/tidwall/gjson"

	"github.com/silenceper/wechat/v2/miniprogram/context"
	"github.com/silenceper/wechat/v2/miniprogram/security"
	"github.com/silenceper/wechat/v2/util"
)

// ConfirmReceiveMethod 确认收货方式
type ConfirmReceiveMethod int8

const (
	// EventTypeTradeManageRemindAccessAPI 提醒接入发货信息管理服务 API
	// 小程序完成账期授权时/小程序产生第一笔交易时/已产生交易但从未发货的小程序，每天一次
	EventTypeTradeManageRemindAccessAPI EventType = "trade_manage_remind_access_api"
	// EventTypeTradeManageRemindShipping 提醒需要上传发货信息
	// 曾经发过货的小程序，订单超过 48 小时未发货时
	EventTypeTradeManageRemindShipping EventType = "trade_manage_remind_shipping"
	// EventTypeTradeManageOrderSettlement 订单将要结算或已经结算
	// 订单完成发货时/订单结算时
	EventTypeTradeManageOrderSettlement EventType = "trade_manage_order_settlement"
	// EventTypeAddExpressPath 运单轨迹更新事件
	EventTypeAddExpressPath EventType = "add_express_path"
	// EventTypeSecvodUpload 短剧媒资上传完成事件
	EventTypeSecvodUpload EventType = "secvod_upload_event"
	// EventTypeSecvodAudit 短剧媒资审核状态事件
	EventTypeSecvodAudit EventType = "secvod_audit_event"
	// EventTypeWxaMediaCheck 媒体内容安全异步审查结果通知
	EventTypeWxaMediaCheck EventType = "wxa_media_check"
	// EventTypeXpayGoodsDeliverNotify 道具发货推送事件
	EventTypeXpayGoodsDeliverNotify EventType = "xpay_goods_deliver_notify"
	// EventTypeXpayCoinPayNotify 代币支付推送事件
	EventTypeXpayCoinPayNotify EventType = "xpay_coin_pay_notify"
	// EventSubscribePopup 用户操作订阅通知弹窗事件推送，用户在图文等场景内订阅通知的操作
	EventSubscribePopup EventType = "subscribe_msg_popup_event"
	// EventSubscribeMsgChange 用户管理订阅通知，用户在服务通知管理页面做通知管理时的操作
	EventSubscribeMsgChange EventType = "subscribe_msg_change_event"
	// EventSubscribeMsgSent 发送订阅通知，调用 bizsend 接口发送通知
	EventSubscribeMsgSent EventType = "subscribe_msg_sent_event"
	// ConfirmReceiveMethodAuto 自动确认收货
	ConfirmReceiveMethodAuto ConfirmReceiveMethod = 1
	// ConfirmReceiveMethodManual 手动确认收货
	ConfirmReceiveMethodManual ConfirmReceiveMethod = 2
)

const (
	// InfoTypeAcceptSubscribeMessage 接受订阅通知
	InfoTypeAcceptSubscribeMessage InfoType = "accept"
	// InfoTypeRejectSubscribeMessage 拒绝订阅通知
	InfoTypeRejectSubscribeMessage InfoType = "reject"
)

// PushReceiver 接收消息推送
// 暂仅支付 Aes 加密方式
type PushReceiver struct {
	*context.Context
}

// NewPushReceiver 实例化
func NewPushReceiver(ctx *context.Context) *PushReceiver {
	return &PushReceiver{
		Context: ctx,
	}
}

// GetMsg 获取接收到的消息 (如果是加密的返回解密数据)
func (receiver *PushReceiver) GetMsg(r *http.Request) (string, []byte, error) {
	// 判断请求格式
	var dataType string
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "text/xml") {
		// xml 格式
		dataType = DataTypeXML
	} else {
		// json 格式
		dataType = DataTypeJSON
	}

	// 读取参数，验证签名
	signature := r.FormValue("signature")
	timestamp := r.FormValue("timestamp")
	nonce := r.FormValue("nonce")
	encryptType := r.FormValue("encrypt_type")
	// 验证签名
	tmpArr := []string{
		receiver.Token,
		timestamp,
		nonce,
	}
	sort.Strings(tmpArr)
	tmpSignature := util.Signature(tmpArr...)
	if tmpSignature != signature {
		return dataType, nil, errors.New("signature error")
	}

	if encryptType == "aes" {
		// 解密
		var reqData DataReceived
		if dataType == DataTypeXML {
			if err := xml.NewDecoder(r.Body).Decode(&reqData); err != nil {
				return dataType, nil, err
			}
		} else {
			if err := json.NewDecoder(r.Body).Decode(&reqData); err != nil {
				return dataType, nil, err
			}
		}
		_, rawMsgBytes, err := util.DecryptMsg(receiver.AppID, reqData.Encrypt, receiver.EncodingAESKey)
		return dataType, rawMsgBytes, err
	}
	// 不加密
	byteData, err := io.ReadAll(r.Body)
	return dataType, byteData, err
}

// GetMsgData 获取接收到的消息 (解密数据)
func (receiver *PushReceiver) GetMsgData(r *http.Request) (MsgType, EventType, PushData, error) {
	dataType, decryptMsg, err := receiver.GetMsg(r)
	if err != nil {
		return "", "", nil, err
	}
	var (
		msgType   MsgType
		eventType EventType
	)
	if dataType == DataTypeXML {
		var commonToken CommonPushData
		if err := xml.Unmarshal(decryptMsg, &commonToken); err != nil {
			return "", "", nil, err
		}
		msgType, eventType = commonToken.MsgType, commonToken.Event
	} else {
		var commonToken CommonPushData
		if err := json.Unmarshal(decryptMsg, &commonToken); err != nil {
			return "", "", nil, err
		}
		msgType, eventType = commonToken.MsgType, commonToken.Event
	}
	if msgType == MsgTypeEvent {
		pushData, err := receiver.getEvent(dataType, eventType, decryptMsg)
		// 暂不支持其他事件类型
		return msgType, eventType, pushData, err
	}
	// 暂不支持其他消息类型
	return msgType, eventType, decryptMsg, nil
}

// getEvent 获取事件推送的数据
func (receiver *PushReceiver) getEvent(dataType string, eventType EventType, decryptMsg []byte) (PushData, error) {
	switch eventType {
	case EventTypeTradeManageRemindAccessAPI:
		// 提醒接入发货信息管理服务 API
		var pushData PushDataRemindAccessAPI
		err := receiver.unmarshal(dataType, decryptMsg, &pushData)
		return &pushData, err
	case EventTypeTradeManageRemindShipping:
		// 提醒需要上传发货信息
		var pushData PushDataRemindShipping
		err := receiver.unmarshal(dataType, decryptMsg, &pushData)
		return &pushData, err
	case EventTypeTradeManageOrderSettlement:
		// 订单将要结算或已经结算
		var pushData PushDataOrderSettlement
		err := receiver.unmarshal(dataType, decryptMsg, &pushData)
		return &pushData, err
	case EventTypeWxaMediaCheck:
		// 媒体内容安全异步审查结果通知
		var pushData MediaCheckAsyncData
		err := receiver.unmarshal(dataType, decryptMsg, &pushData)
		return &pushData, err
	case EventTypeAddExpressPath:
		// 运单轨迹更新
		var pushData PushDataAddExpressPath
		err := receiver.unmarshal(dataType, decryptMsg, &pushData)
		return &pushData, err
	case EventTypeSecvodUpload:
		// 短剧媒资上传完成
		var pushData PushDataSecVodUpload
		err := receiver.unmarshal(dataType, decryptMsg, &pushData)
		return &pushData, err
	case EventTypeSecvodAudit:
		// 短剧媒资审核状态
		var pushData PushDataSecVodAudit
		err := receiver.unmarshal(dataType, decryptMsg, &pushData)
		return &pushData, err
	case EventTypeXpayGoodsDeliverNotify:
		// 道具发货推送事件
		var pushData PushDataXpayGoodsDeliverNotify
		err := receiver.unmarshal(dataType, decryptMsg, &pushData)
		return &pushData, err
	case EventTypeXpayCoinPayNotify:
		// 代币支付推送事件
		var pushData PushDataXpayCoinPayNotify
		err := receiver.unmarshal(dataType, decryptMsg, &pushData)
		return &pushData, err
	case EventSubscribePopup:
		// 用户操作订阅通知弹窗事件推送
		return receiver.unmarshalSubscribePopup(dataType, decryptMsg)
	case EventSubscribeMsgChange:
		// 用户管理订阅通知事件推送
		return receiver.unmarshalSubscribeMsgChange(dataType, decryptMsg)
	case EventSubscribeMsgSent:
		// 用户发送订阅通知事件推送
		return receiver.unmarshalSubscribeMsgSent(dataType, decryptMsg)
	}
	// 暂不支持其他事件类型，直接返回解密后的数据，由调用方处理
	return decryptMsg, nil
}

// unmarshal 解析推送的数据
func (receiver *PushReceiver) unmarshal(dataType string, decryptMsg []byte, pushData interface{}) error {
	if dataType == DataTypeXML {
		return xml.Unmarshal(decryptMsg, pushData)
	}
	return json.Unmarshal(decryptMsg, pushData)
}

// unmarshalSubscribePopup
func (receiver *PushReceiver) unmarshalSubscribePopup(dataType string, decryptMsg []byte) (PushData, error) {
	var pushData PushDataSubscribePopup
	err := receiver.unmarshal(dataType, decryptMsg, &pushData)
	if err == nil {
		listData := gjson.Get(string(decryptMsg), "List")
		if listData.IsObject() {
			listItem := SubscribeMsgPopupEventList{}
			if parseErr := json.Unmarshal([]byte(listData.Raw), &listItem); parseErr != nil {
				return &pushData, parseErr
			}
			pushData.SetSubscribeMsgPopupEvents([]SubscribeMsgPopupEventList{listItem})
		} else if listData.IsArray() {
			listItems := make([]SubscribeMsgPopupEventList, 0)
			if parseErr := json.Unmarshal([]byte(listData.Raw), &listItems); parseErr != nil {
				return &pushData, parseErr
			}
			pushData.SetSubscribeMsgPopupEvents(listItems)
		}
	}

	return &pushData, err
}

// unmarshalSubscribeMsgChange 解析用户管理订阅通知事件推送
func (receiver *PushReceiver) unmarshalSubscribeMsgChange(dataType string, decryptMsg []byte) (PushData, error) {
	var pushData PushDataSubscribeMsgChange
	err := receiver.unmarshal(dataType, decryptMsg, &pushData)
	if err == nil {
		listData := gjson.Get(string(decryptMsg), "List")
		if listData.IsObject() {
			listItem := SubscribeMsgChangeList{}
			if parseErr := json.Unmarshal([]byte(listData.Raw), &listItem); parseErr != nil {
				return &pushData, parseErr
			}
			pushData.SetSubscribeMsgChangeEvents([]SubscribeMsgChangeList{listItem})
		} else if listData.IsArray() {
			listItems := make([]SubscribeMsgChangeList, 0)
			if parseErr := json.Unmarshal([]byte(listData.Raw), &listItems); parseErr != nil {
				return &pushData, parseErr
			}
			pushData.SetSubscribeMsgChangeEvents(listItems)
		}
	}
	return &pushData, err
}

// unmarshalSubscribeMsgSent 解析用户发送订阅通知事件推送
func (receiver *PushReceiver) unmarshalSubscribeMsgSent(dataType string, decryptMsg []byte) (PushData, error) {
	var pushData PushDataSubscribeMsgSent
	err := receiver.unmarshal(dataType, decryptMsg, &pushData)
	if err == nil {
		listData := gjson.Get(string(decryptMsg), "List")
		if listData.IsObject() {
			listItem := SubscribeMsgSentList{}
			if parseErr := json.Unmarshal([]byte(listData.Raw), &listItem); parseErr != nil {
				return &pushData, parseErr
			}
			pushData.SetSubscribeMsgSentEvents([]SubscribeMsgSentList{listItem})
		} else if listData.IsArray() {
			listItems := make([]SubscribeMsgSentList, 0)
			if parseErr := json.Unmarshal([]byte(listData.Raw), &listItems); parseErr != nil {
				return &pushData, parseErr
			}
			pushData.SetSubscribeMsgSentEvents(listItems)
		}
	}
	return &pushData, err
}

// DataReceived 接收到的数据
type DataReceived struct {
	Encrypt string `json:"Encrypt" xml:"Encrypt"` // 加密的消息体
}

// PushData 推送的数据 (已转对应的结构体)
type PushData interface{}

// CommonPushData 推送数据通用部分
type CommonPushData struct {
	XMLName      xml.Name  `json:"-" xml:"xml"`
	MsgType      MsgType   `json:"MsgType" xml:"MsgType"`           // 消息类型，为固定值 "event"
	Event        EventType `json:"Event" xml:"Event"`               // 事件类型
	ToUserName   string    `json:"ToUserName" xml:"ToUserName"`     // 小程序的原始 ID
	FromUserName string    `json:"FromUserName" xml:"FromUserName"` // 发送方账号（一个 OpenID，此时发送方是系统账号）
	CreateTime   int64     `json:"CreateTime" xml:"CreateTime"`     // 消息创建时间（整型），时间戳
}

// MediaCheckAsyncData 媒体内容安全异步审查结果通知
type MediaCheckAsyncData struct {
	CommonPushData
	Appid   string                `json:"appid" xml:"appid"`
	TraceID string                `json:"trace_id" xml:"trace_id"`
	Version int                   `json:"version" xml:"version"`
	Detail  []*MediaCheckDetail   `json:"detail" xml:"detail"`
	Errcode int                   `json:"errcode" xml:"errcode"`
	Errmsg  string                `json:"errmsg" xml:"errmsg"`
	Result  MediaCheckAsyncResult `json:"result" xml:"result"`
}

// MediaCheckDetail 检测结果详情
type MediaCheckDetail struct {
	Strategy string                `json:"strategy" xml:"strategy"`
	Errcode  int                   `json:"errcode" xml:"errcode"`
	Suggest  security.CheckSuggest `json:"suggest" xml:"suggest"`
	Label    int                   `json:"label" xml:"label"`
	Prob     int                   `json:"prob" xml:"prob"`
}

// MediaCheckAsyncResult 检测结果
type MediaCheckAsyncResult struct {
	Suggest security.CheckSuggest `json:"suggest" xml:"suggest"`
	Label   security.CheckLabel   `json:"label" xml:"label"`
}

// PushDataOrderSettlement 订单将要结算或已经结算通知
type PushDataOrderSettlement struct {
	CommonPushData
	TransactionID           string               `json:"transaction_id" xml:"transaction_id"`                       // 支付订单号
	MerchantID              string               `json:"merchant_id" xml:"merchant_id"`                             // 商户号
	SubMerchantID           string               `json:"sub_merchant_id" xml:"sub_merchant_id"`                     // 子商户号
	MerchantTradeNo         string               `json:"merchant_trade_no" xml:"merchant_trade_no"`                 // 商户订单号
	PayTime                 int64                `json:"pay_time" xml:"pay_time"`                                   // 支付成功时间，秒级时间戳
	ShippedTime             int64                `json:"shipped_time" xml:"shipped_time"`                           // 发货时间，秒级时间戳
	EstimatedSettlementTime int64                `json:"estimated_settlement_time" xml:"estimated_settlement_time"` // 预计结算时间，秒级时间戳。发货时推送才有该字段
	ConfirmReceiveMethod    ConfirmReceiveMethod `json:"confirm_receive_method" xml:"confirm_receive_method"`       // 确认收货方式：1. 自动确认收货；2. 手动确认收货。结算时推送才有该字段
	ConfirmReceiveTime      int64                `json:"confirm_receive_time" xml:"confirm_receive_time"`           // 确认收货时间，秒级时间戳。结算时推送才有该字段
	SettlementTime          int64                `json:"settlement_time" xml:"settlement_time"`                     // 订单结算时间，秒级时间戳。结算时推送才有该字段
}

// PushDataRemindShipping 提醒需要上传发货信息
type PushDataRemindShipping struct {
	CommonPushData
	TransactionID   string `json:"transaction_id" xml:"transaction_id"`       // 微信支付订单号
	MerchantID      string `json:"merchant_id" xml:"merchant_id"`             // 商户号
	SubMerchantID   string `json:"sub_merchant_id" xml:"sub_merchant_id"`     // 子商户号
	MerchantTradeNo string `json:"merchant_trade_no" xml:"merchant_trade_no"` // 商户订单号
	PayTime         int64  `json:"pay_time" xml:"pay_time"`                   // 支付成功时间，秒级时间戳
	Msg             string `json:"msg" xml:"msg"`                             // 消息文本内容
}

// PushDataRemindAccessAPI 提醒接入发货信息管理服务 API 信息
type PushDataRemindAccessAPI struct {
	CommonPushData
	Msg string `json:"msg" xml:"msg"` // 消息文本内容
}

// PushDataAddExpressPath 运单轨迹更新信息
type PushDataAddExpressPath struct {
	CommonPushData
	DeliveryID string                          `json:"DeliveryID" xml:"DeliveryID"` // 快递公司 ID
	WayBillID  string                          `json:"WaybillId" xml:"WaybillId"`   // 运单 ID
	OrderID    string                          `json:"OrderId" xml:"OrderId"`       // 订单 ID
	Version    int                             `json:"Version" xml:"Version"`       // 轨迹版本号（整型）
	Count      int                             `json:"Count" xml:"Count"`           // 轨迹节点数（整型）
	Actions    []*PushDataAddExpressPathAction `json:"Actions" xml:"Actions"`       // 轨迹节点列表
}

// PushDataAddExpressPathAction 轨迹节点
type PushDataAddExpressPathAction struct {
	ActionTime int64  `json:"ActionTime" xml:"ActionTime"` // 轨迹节点 Unix 时间戳
	ActionType int    `json:"ActionType" xml:"ActionType"` // 轨迹节点类型
	ActionMsg  string `json:"ActionMsg" xml:"ActionMsg"`   // 轨迹节点详情
}

// PushDataSecVodUpload 短剧媒资上传完成
type PushDataSecVodUpload struct {
	CommonPushData
	UploadEvent SecVodUploadEvent `json:"upload_event" xml:"upload_event"` // 上传完成事件
}

// SecVodUploadEvent 短剧媒资上传完成事件
type SecVodUploadEvent struct {
	MediaID       int64  `json:"media_id" xml:"media_id"`             // 媒资 id
	SourceContext string `json:"source_context" xml:"source_context"` // 透传上传接口中开发者设置的值。
	ErrCode       int    `json:"errcode" xml:"errcode"`               // 错误码，上传失败时该值非
	ErrMsg        string `json:"errmsg" xml:"errmsg"`                 // 错误提示
}

// PushDataSecVodAudit 短剧媒资审核状态
type PushDataSecVodAudit struct {
	CommonPushData
	AuditEvent SecVodAuditEvent `json:"audit_event" xml:"audit_event"` // 审核状态事件
}

// SecVodAuditEvent 短剧媒资审核状态事件
type SecVodAuditEvent struct {
	DramaID       int64            `json:"drama_id" xml:"drama_id"`             // 剧目 id
	SourceContext string           `json:"source_context" xml:"source_context"` // 透传上传接口中开发者设置的值
	AuditDetail   DramaAuditDetail `json:"audit_detail" xml:"audit_detail"`     // 剧目审核结果，单独每一集的审核结果可以根据 drama_id 查询剧集详情得到
}

// DramaAuditDetail 剧目审核结果
type DramaAuditDetail struct {
	Status     int   `json:"status" xml:"status"`           // 审核状态，0 为无效值；1 为审核中；2 为最终失败；3 为审核通过；4 为驳回重填
	CreateTime int64 `json:"create_time" xml:"create_time"` // 提审时间戳
	AuditTime  int64 `json:"audit_time" xml:"audit_time"`   // 审核时间戳
}

// PushDataXpayGoodsDeliverNotify 道具发货推送
type PushDataXpayGoodsDeliverNotify struct {
	CommonPushData
	OpenID        string        `json:"OpenId" xml:"OpenId"`               // 用户 openid
	OutTradeNo    string        `json:"OutTradeNo" xml:"OutTradeNo"`       // 业务订单号
	Env           int           `json:"Env" xml:"Env"`                     // ，环境配置 0：现网环境（也叫正式环境）1：沙箱环境
	WeChatPayInfo WeChatPayInfo `json:"WeChatPayInfo" xml:"WeChatPayInfo"` // 微信支付信息 非微信支付渠道可能没有
	GoodsInfo     GoodsInfo     `json:"GoodsInfo" xml:"GoodsInfo"`         // 道具参数信息
}

// WeChatPayInfo 微信支付信息
type WeChatPayInfo struct {
	MchOrderNo    string `json:"MchOrderNo" xml:"MchOrderNo"`       // 微信支付商户单号
	TransactionID string `json:"TransactionId" xml:"TransactionId"` // 交易单号（微信支付订单号）
	PaidTime      int64  `json:"PaidTime" xml:"PaidTime"`           // 用户支付时间，Linux 秒级时间戳
}

// GoodsInfo 道具参数信息
type GoodsInfo struct {
	ProductID   string `json:"ProductId" xml:"ProductId"`     // 道具 ID
	Quantity    int    `json:"Quantity" xml:"Quantity"`       // 数量
	OrigPrice   int64  `json:"OrigPrice" xml:"OrigPrice"`     // 物品原始价格（单位：分）
	ActualPrice int64  `json:"ActualPrice" xml:"ActualPrice"` // 物品实际支付价格（单位：分）
	Attach      string `json:"Attach" xml:"Attach"`           // 透传信息
}

// PushDataXpayCoinPayNotify 代币支付推送
type PushDataXpayCoinPayNotify struct {
	CommonPushData
	OpenID        string        `json:"OpenId" xml:"OpenId"`               // 用户 openid
	OutTradeNo    string        `json:"OutTradeNo" xml:"OutTradeNo"`       // 业务订单号
	Env           int           `json:"Env" xml:"Env"`                     // ，环境配置 0：现网环境（也叫正式环境）1：沙箱环境
	WeChatPayInfo WeChatPayInfo `json:"WeChatPayInfo" xml:"WeChatPayInfo"` // 微信支付信息 非微信支付渠道可能没有
	CoinInfo      CoinInfo      `json:"CoinInfo" xml:"CoinInfo"`           // 代币参数信息
}

// CoinInfo 代币参数信息
type CoinInfo struct {
	Quantity    int    `json:"Quantity" xml:"Quantity"`       // 数量
	OrigPrice   int64  `json:"OrigPrice" xml:"OrigPrice"`     // 物品原始价格（单位：分）
	ActualPrice int64  `json:"ActualPrice" xml:"ActualPrice"` // 物品实际支付价格（单位：分）
	Attach      string `json:"Attach" xml:"Attach"`           // 透传信息
}

// PushDataSubscribePopup 用户操作订阅通知弹窗事件推送
type PushDataSubscribePopup struct {
	CommonPushData
	subscribeMsgPopupEventList []SubscribeMsgPopupEventList `json:"-"`
	SubscribeMsgPopupEvent     SubscribeMsgPopupEvent       `xml:"SubscribeMsgPopupEvent"`
}

// SubscribeMsgPopupEvent 用户操作订阅通知弹窗消息回调
type SubscribeMsgPopupEvent struct {
	List []SubscribeMsgPopupEventList `xml:"List"`
}

// SubscribeMsgPopupEventList 订阅消息事件列表
type SubscribeMsgPopupEventList struct {
	TemplateID            string `xml:"TemplateId" json:"TemplateId"`
	SubscribeStatusString string `xml:"SubscribeStatusString" json:"SubscribeStatusString"`
	PopupScene            string `xml:"PopupScene" json:"PopupScene"`
}

// SetSubscribeMsgPopupEvents 设置订阅消息事件
func (s *PushDataSubscribePopup) SetSubscribeMsgPopupEvents(list []SubscribeMsgPopupEventList) {
	s.subscribeMsgPopupEventList = list
}

// GetSubscribeMsgPopupEvents 获取订阅消息事件数据
func (s *PushDataSubscribePopup) GetSubscribeMsgPopupEvents() []SubscribeMsgPopupEventList {
	if s.subscribeMsgPopupEventList != nil {
		return s.subscribeMsgPopupEventList
	}

	if s.SubscribeMsgPopupEvent.List == nil || len(s.SubscribeMsgPopupEvent.List) < 1 {
		return nil
	}
	return s.SubscribeMsgPopupEvent.List
}

// PushDataSubscribeMsgChange 用户管理订阅通知事件推送
type PushDataSubscribeMsgChange struct {
	CommonPushData
	SubscribeMsgChangeEvent SubscribeMsgChangeEvent  `xml:"SubscribeMsgChangeEvent"`
	subscribeMsgChangeList  []SubscribeMsgChangeList `json:"-"`
}

// SubscribeMsgChangeEvent 用户管理订阅通知回调
type SubscribeMsgChangeEvent struct {
	List []SubscribeMsgChangeList `xml:"List" json:"List"`
}

// SubscribeMsgChangeList 订阅消息事件列表
type SubscribeMsgChangeList struct {
	TemplateID            string `xml:"TemplateId" json:"TemplateId"`
	SubscribeStatusString string `xml:"SubscribeStatusString" json:"SubscribeStatusString"`
}

// SetSubscribeMsgChangeEvents 设置订阅消息事件
func (s *PushDataSubscribeMsgChange) SetSubscribeMsgChangeEvents(list []SubscribeMsgChangeList) {
	s.subscribeMsgChangeList = list
}

// GetSubscribeMsgChangeEvents 获取订阅消息事件数据
func (s *PushDataSubscribeMsgChange) GetSubscribeMsgChangeEvents() []SubscribeMsgChangeList {
	if s.subscribeMsgChangeList != nil {
		return s.subscribeMsgChangeList
	}

	if s.SubscribeMsgChangeEvent.List == nil || len(s.SubscribeMsgChangeEvent.List) < 1 {
		return nil
	}

	return s.SubscribeMsgChangeEvent.List
}

// PushDataSubscribeMsgSent 用户发送订阅通知事件推送
type PushDataSubscribeMsgSent struct {
	CommonPushData
	SubscribeMsgSentEvent     SubscribeMsgSentEvent  `xml:"SubscribeMsgSentEvent"`
	subscribeMsgSentEventList []SubscribeMsgSentList `json:"-"`
}

// SubscribeMsgSentEvent 用户发送订阅通知回调
type SubscribeMsgSentEvent struct {
	List []SubscribeMsgSentList `xml:"List" json:"List"`
}

// SubscribeMsgSentList 订阅消息事件列表
type SubscribeMsgSentList struct {
	TemplateID  string `xml:"TemplateId" json:"TemplateId"`
	MsgID       string `xml:"MsgID" json:"MsgID"`
	ErrorCode   string `xml:"ErrorCode" json:"ErrorCode"`
	ErrorStatus string `xml:"ErrorStatus" json:"ErrorStatus"`
}

// SetSubscribeMsgSentEvents 设置订阅消息事件
func (s *PushDataSubscribeMsgSent) SetSubscribeMsgSentEvents(list []SubscribeMsgSentList) {
	s.subscribeMsgSentEventList = list
}

// GetSubscribeMsgSentEvents 获取订阅消息事件数据
func (s *PushDataSubscribeMsgSent) GetSubscribeMsgSentEvents() []SubscribeMsgSentList {
	if s.subscribeMsgSentEventList != nil {
		return s.subscribeMsgSentEventList
	}

	if s.SubscribeMsgSentEvent.List == nil || len(s.SubscribeMsgSentEvent.List) < 1 {
		return nil
	}

	return s.SubscribeMsgSentEvent.List
}
