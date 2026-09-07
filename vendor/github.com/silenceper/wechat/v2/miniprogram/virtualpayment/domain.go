/*
 *   Copyright silenceper/wechat Author(https://silenceper.com/wechat/). All Rights Reserved.
 *
 *   Licensed under the Apache License, Version 2.0 (the "License");
 *   you may not use this file except in compliance with the License.
 *   You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 *   Unless required by applicable law or agreed to in writing, software
 *   distributed under the License is distributed on an "AS IS" BASIS,
 *   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *   See the License for the specific language governing permissions and
 *   limitations under the License.
 *
 *   You can obtain one at https://github.com/silenceper/wechat.
 *
 */

package virtualpayment

import (
	"github.com/silenceper/wechat/v2/miniprogram/context"
	"github.com/silenceper/wechat/v2/util"
)

// VirtualPayment mini program virtual payment
// https://developers.weixin.qq.com/miniprogram/dev/platform-capabilities/business-capabilities/virtual-payment.html
type VirtualPayment struct {
	ctx        *context.Context
	sessionKey string
}

// Env Environment 0 - Production environment 1 - Sandbox environment
type Env int

// ErrCode error code
type ErrCode int

// OrderStatus 订单状态
type OrderStatus int

// CommonRequest common request parameters
type CommonRequest struct {
	OpenID string `json:"openid"` // The user's openID
	Env    Env    `json:"env"`    // Environment 0 - Production environment 1 - Sandbox environment
}

// PaymentRequest payment request parameters
type PaymentRequest struct {
	SignData  string `json:"sign_data"` // 具体支付参数见 signData, 该参数需以 string 形式传递，例如 signData: '{"offerId":"123","buyQuantity":1,"env":0,"currencyType":"CNY","platform":"android","productId":"testproductId","goodsPrice":10,"outTradeNo":"xxxxxx","attach":"testdata"}'
	Mode      string `json:"mode"`      // 支付模式，枚举值：short_series_goods: 道具直购，short_series_coin: 代币充值
	PaySig    string `json:"pay_sig"`   // 支付签名，具体生成方式见下方说明
	Signature string `json:"signature"` // 用户态签名，具体生成方式见下方说明
}

// SignData 签名数据
type SignData struct {
	OfferID      string `json:"offerId"`             // 在米大师侧申请的应用 id, mp-支付基础配置中的 offerid
	BuyQuantity  int    `json:"buyQuantity"`         // 购买数量
	Env          Env    `json:"env"`                 // 环境 0-正式环境 1-沙箱环境
	CurrencyType string `json:"currencyType"`        // 币种 默认值：CNY 人民币
	Platform     string `json:"platform,omitempty"`  // 申请接入时的平台，platform 与应用 id 有关 默认值：android 安卓平台
	ProductID    string `json:"productId,omitempty"` // 道具 ID, **该字段仅 mode=short_series_goods 时可用**
	GoodsPrice   int    `json:"goodsPrice"`          // 道具单价 (分), **该字段仅 mode=short_series_goods 时可用**, 用来校验价格与后台道具价格是否一致，避免用户在业务商城页看到的价格与实际价格不一致导致投诉
	OutTradeNo   string `json:"outTradeNo"`          // 业务订单号，每个订单号只能使用一次，重复使用会失败 (极端情况不保证唯一，不建议业务强依赖唯一性). 要求 8-32 个字符内，只能是数字、大小写字母、符号 _-|*@组成，不能以下划线 (_) 开头
	Attach       string `json:"attach"`              // 透传数据，发货通知时会透传给开发者
}

// QueryUserBalanceRequest 查询用户代币余额，请求参数
// 1. 需要用户态签名与支付签名
// POST，请求参数为 json 字符串，Content-Type 为 application/json
type QueryUserBalanceRequest struct {
	CommonRequest
	UserIP string `json:"user_ip"` // 用户 ip，例如:1.1.1.1
}

// QueryUserBalanceResponse 查询虚拟支付余额 响应参数
type QueryUserBalanceResponse struct {
	util.CommonError
	Balance        int `json:"balance"`         // 代币总余额，包括有价和赠送部分
	PresentBalance int `json:"present_balance"` // 赠送账户的代币余额
	SumSave        int `json:"sum_save"`        // 累计有价货币充值数量
	SumPresent     int `json:"sum_present"`     // 累计赠送无价货币数量
	SumBalance     int `json:"sum_balance"`     // 历史总增加的代币金额
	SumCost        int `json:"sum_cost"`        // 历史总消耗代币金额
	FirstSaveFlag  int `json:"first_save_flag"` // 是否满足首充活动标记。0:不满足。1:满足
}

// CurrencyPayRequest 扣减代币（一般用于代币支付）
// 1. 需要用户态签名与支付签名
// POST，请求参数为 json 字符串，Content-Type 为 application/json
type CurrencyPayRequest struct {
	CommonRequest
	UserIP     string `json:"user_ip"`     // 用户 ip，例如：1.1.1.1
	Amount     int    `json:"amount"`      // 支付的代币数量
	OrderID    string `json:"order_id"`    // 商户订单号，需要保证唯一性
	PayItem    string `json:"payitem"`     // 物品信息。记录到账户流水中。如:[{"productid":"物品 id", "unit_price": 单价，"quantity": 数量}]
	Remark     string `json:"remark"`      // 备注信息。需要在账单中展示
	DeviceType string `json:"device_type"` // 平台类型 1-安卓 2-苹果
}

// PayItem 物品信息
type PayItem struct {
	ProductID string `json:"productid"`  // 物品 id
	UnitPrice int    `json:"unit_price"` // 单价
	Quantity  int    `json:"quantity"`   // 数量
}

// CurrencyPayResponse 扣减代币（一般用于代币支付）响应参数
type CurrencyPayResponse struct {
	util.CommonError
	OrderID           string `json:"order_id"`            // 商户订单号
	Balance           int    `json:"balance"`             // 总余额，包括有价和赠送部分
	UsedPresentAmount int    `json:"used_present_amount"` // 使用赠送部分的代币数量
}

// QueryOrderRequest 查询创建的订单（现金单，非代币单），请求参数
// 1. 使用支付签名
// POST，请求参数为 json 字符串，Content-Type 为 application/json
type QueryOrderRequest struct {
	CommonRequest
	OrderID   string `json:"order_id,omitempty"`    // 商户订单号 创建的订单号
	WxOrderID string `json:"wx_order_id,omitempty"` // 微信内部单号 (与 order_id 二选一)
}

// OrderItem 订单信息
type OrderItem struct {
	OrderID        string      `json:"order_id"`         // 商户订单号
	CreateTime     int64       `json:"create_time"`      // 订单创建时间
	UpdateTime     int64       `json:"update_time"`      // 订单更新时间
	Status         OrderStatus `json:"status"`           // 订单状态 当前状态 0-订单初始化（未创建成功，不可用于支付）1-订单创建成功 2-订单已经支付，待发货 3-订单发货中 4-订单已发货 5-订单已经退款 6-订单已经关闭（不可再使用）7-订单退款失败
	BizType        int         `json:"biz_type"`         // 业务类型 0-短剧
	OrderFee       int         `json:"order_fee"`        // 订单金额，单位：分
	CouponFee      int         `json:"coupon_fee"`       // 优惠金额，单位：分
	PaidFee        int         `json:"paid_fee"`         // 用户支付金额，单位：分
	OrderType      int         `json:"order_type"`       // 订单类型 0-支付单 1-退款单
	RefundFee      int         `json:"refund_fee"`       // 当类型为退款单时表示退款金额，单位分
	PaidTime       int64       `json:"paid_time"`        // 支付/退款时间，unix秒级时间戳
	ProvideTime    int64       `json:"provide_time"`     // 发货时间，unix 秒级时间戳
	BizMeta        string      `json:"biz_meta"`         // 业务自定义数据 订单创建时传的信息
	EnvType        int         `json:"env_type"`         // 环境类型 1-现网 2-沙箱
	Token          string      `json:"token"`            // 下单时米大师返回的 token
	LeftFee        int         `json:"left_fee"`         // 支付单类型时表示此单经过退款还剩余的金额，单位：分
	WxOrderID      string      `json:"wx_order_id"`      // 微信内部单号
	ChannelOrderID string      `json:"channel_order_id"` // 渠道订单号，为用户微信支付详情页面上的商户单号
	WxPayOrderID   string      `json:"wxpay_order_id"`   // 微信支付交易单号，为用户微信支付详情页面上的交易单号
}

// QueryOrderResponse 查询创建的订单（现金单，非代币单）响应参数
type QueryOrderResponse struct {
	util.CommonError
	Order *OrderItem `json:"order"` // 订单信息
}

// CancelCurrencyPayRequest 取消订单（现金单，非代币单），请求参数
// 1. 使用支付签名
// POST，请求参数为 json 字符串，Content-Type 为 application/json
type CancelCurrencyPayRequest struct {
	CommonRequest
	UserIP     string `json:"user_ip"`      // 用户 ip，例如：1.1.1.1
	PayOrderID string `json:"pay_order_id"` // 支付单号 代币支付 (调用 currency_pay 接口时) 时传的 order_id
	OrderID    string `json:"order_id"`     // 本次退款单的单号
	Amount     int    `json:"amount"`       // 退款金额
	DeviceType int    `json:"device_type"`  // 平台类型 1-安卓 2-苹果
}

// CancelCurrencyPayResponse 取消订单（现金单，非代币单）响应参数
type CancelCurrencyPayResponse struct {
	util.CommonError
	OrderID string `json:"order_id"` // 退款订单号
}

// NotifyProvideGoodsRequest 通知发货，请求参数
// 通知已经发货完成（只能通知现金单）,正常通过 xpay_goods_deliver_notify 消息推送返回成功就不需要调用这个 api 接口。这个接口用于异常情况推送不成功时手动将单改成已发货状态
// 1. 使用支付签名
// POST，请求参数为 json 字符串，Content-Type 为 application/json
type NotifyProvideGoodsRequest struct {
	OrderID   string `json:"order_id,omitempty"`    // 商户订单号 下单时传的单号
	WxOrderID string `json:"wx_order_id,omitempty"` // 微信内部单号 (与 order_id 二选一)
	Env       Env    `json:"env"`                   // 环境 0-正式环境 1-沙箱环境
}

// NotifyProvideGoodsResponse 通知发货响应参数
type NotifyProvideGoodsResponse struct {
	util.CommonError
}

// PresentCurrencyRequest 赠送代币，请求参数
// 代币赠送接口，由于目前不支付按单号查赠送单的功能，所以当需要赠送的时候可以一直重试到返回 0 或者返回 268490004（重复操作）为止
// 1. 需要用户态签名与支付签名
// POST，请求参数为 json 字符串，Content-Type 为 application/json
type PresentCurrencyRequest struct {
	CommonRequest
	OrderID    string `json:"order_id"`    // 赠送单号，商户订单号，需要保证唯一性
	Amount     int    `json:"amount"`      // 赠送的代币数量
	DeviceType string `json:"device_type"` // 平台类型 1-安卓 2-苹果
}

// PresentCurrencyResponse 赠送代币响应参数
type PresentCurrencyResponse struct {
	util.CommonError
	Balance        int    `json:"balance"`         // 赠送后用户的代币余额
	OrderID        string `json:"order_id"`        // 赠送单号
	PresentBalance int    `json:"present_balance"` // 用户收到的总赠送金额
}

// DownloadBillRequest 下载账单，请求参数
// 用于下载小程序账单，第一次调用触发生成下载 url，可以间隔轮训来获取最终生成的下载 url。账单中金额相关字段是以分为单位。
// 1. 使用支付签名
// POST，请求参数为 json 字符串，Content-Type 为 application/json
type DownloadBillRequest struct {
	BeginDs string `json:"begin_ds"` // 账单开始日期，格式为 yyyymmdd 起始时间（如 20230801）
	EndDs   string `json:"end_ds"`   // 账单结束日期，格式为 yyyymmdd 结束时间（如 20230801）
}

// DownloadBillResponse 下载账单响应参数
type DownloadBillResponse struct {
	util.CommonError
	URL string `json:"url"` // 账单下载地址
}

// RefundOrderRequest 退款，请求参数
// 对使用 jsapi 接口下的单进行退款
// 1. 使用支付签名
// POST，请求参数为 json 字符串，Content-Type 为 application/json
type RefundOrderRequest struct {
	CommonRequest
	OrderID       string `json:"order_id"`        // 商户订单号，需要保证唯一性
	WxOrderID     string `json:"wx_order_id"`     // 微信内部单号 (与 order_id 二选一)
	RefundOrderID string `json:"refund_order_id"` // 退款单号，本次退款时需要传的单号，长度为 [8,32]，字符只允许使用字母、数字、'_'、'-'
	LeftFee       int    `json:"left_fee"`        // 退款金额，单位：分 当前单剩余可退金额，单位分，可以通过调用 query_order 接口查到
	RefundFee     int    `json:"refund_fee"`      // 退款金额，单位：分 需要 (0,left_fee] 之间
	BizMeta       string `json:"biz_meta"`        // 商家自定义数据，传入后可在 query_order 接口查询时原样返回，长度需要 [0,1024]
	RefundReason  string `json:"refund_reason"`   // 退款原因，当前仅支持以下值 0-暂无描述 1-产品问题，影响使用或效果不佳 2-售后问题，无法满足需求 3-意愿问题，用户主动退款 4-价格问题 5:其他原因
	ReqFrom       string `json:"req_from"`        // 退款来源，当前仅支持以下值 1-人工客服退款，即用户电话给客服，由客服发起退款流程 2-用户自己发起退款流程 3-其它
}

// RefundOrderResponse 退款响应参数
type RefundOrderResponse struct {
	util.CommonError
	RefundOrderID   string `json:"refund_order_id"`    // 退款单号
	RefundWxOrderID string `json:"refund_wx_order_id"` // 退款单的微信侧单号
	PayOrderID      string `json:"pay_order_id"`       // 该退款单对应的支付单单号
	PayWxOrderID    string `json:"pay_wx_order_id"`    // 该退款单对应的支付单微信侧单号
}

// CreateWithdrawOrderRequest 创建提现单，请求参数
// 1. 使用支付签名
// POST，请求参数为 json 字符串，Content-Type 为 application/json
type CreateWithdrawOrderRequest struct {
	WithdrawNO     string `json:"withdraw_no"`     // 提现单单号，长度为 [8,32]，字符只允许使用字母、数字、'_'、'-'
	WithdrawAmount string `json:"withdraw_amount"` // 提现的金额，单位元，例如提现 1 分钱请使用 0.01
	Env            Env    `json:"env"`             // 环境 0-正式环境 1-沙箱环境
}

// CreateWithdrawOrderResponse 创建提现单响应参数
type CreateWithdrawOrderResponse struct {
	util.CommonError
	WithdrawNO   string `json:"withdraw_no"`    // 提现单单号
	WxWithdrawNO string `json:"wx_withdraw_no"` // 提现单的微信侧单号
}

// QueryWithdrawOrderRequest 查询提现单，请求参数
// 1. 使用支付签名
// POST，请求参数为 json 字符串，Content-Type 为 application/json
type QueryWithdrawOrderRequest struct {
	WithdrawNO string `json:"withdraw_no"` // 提现单单号，长度为 [8,32]，字符只允许使用字母、数字、'_'、'-' (与 wx_withdraw_no 二选一)
	Env        Env    `json:"env"`         // 环境 0-正式环境 1-沙箱环境
}

// QueryWithdrawOrderResponse 查询提现单响应参数
type QueryWithdrawOrderResponse struct {
	util.CommonError
	WithdrawNO               string `json:"withdraw_no"`                // 提现单单号
	Status                   int    `json:"status"`                     // 提现单的微信侧单号 1-创建成功，提现中 2-提现成功 3-提现失败
	WithdrawAmount           string `json:"withdraw_amount"`            // 提现的金额，单位元，例如提现 1 分钱请使用 0.01
	WxWithdrawNo             string `json:"wx_withdraw_no"`             // 提现单的微信侧单号
	WithdrawSuccessTimestamp int64  `json:"withdraw_success_timestamp"` // 提现单成功的秒级时间戳，unix 秒级时间戳
	CreateTime               string `json:"create_time"`                // 提现单创建时间
	FailReason               string `json:"failReason"`                 // 提现失败的原因
}

// StartUploadGoodsRequest 启动批量上传道具任务，请求参数
// 1. 使用支付签名
// POST，请求参数为 json 字符串，Content-Type 为 application/json
type StartUploadGoodsRequest struct {
	UploadItem []*UploadItem `json:"upload_item"` // 道具信息
	Env        Env           `json:"env"`         // 环境 0-正式环境 1-沙箱环境
}

// UploadItem 道具信息
type UploadItem struct {
	ID           string `json:"id"`                      // 道具 id，长度 (0,64]，字符只允许使用字母、数字、'_'、'-'
	Name         string `json:"name"`                    // 道具名称，长度 (0,1024]
	Price        int    `json:"price"`                   // 道具单价，单位分，需要大于 0
	Remark       string `json:"remark"`                  // 道具备注，长度 (0,1024]
	ItemURL      string `json:"item_url"`                // 道具图片的 url 地址，当前仅支持 jpg,png 等格式
	UploadStatus int    `json:"upload_status,omitempty"` // 上传状态 0-上传中 1-id 已经存在 2-上传成功 3-上传失败
	ErrMsg       string `json:"errmsg,omitempty"`        // 上传失败的原因
}

// StartUploadGoodsResponse 启动批量上传道具任务响应参数
type StartUploadGoodsResponse struct {
	util.CommonError
}

// QueryUploadGoodsRequest 查询批量上传道具任务，请求参数
// 1. 使用支付签名
// POST，请求参数为 json 字符串，Content-Type 为 application/json
type QueryUploadGoodsRequest struct {
	Env Env `json:"env"` // 环境 0-正式环境 1-沙箱环境
}

// QueryUploadGoodsResponse 查询批量上传道具任务响应参数
type QueryUploadGoodsResponse struct {
	util.CommonError
	UploadItem []*UploadItem `json:"upload_item"` // 道具信息列表
	Status     int           `json:"status"`      // 任务状态 0-无任务在运行 1-任务运行中 2-上传失败或部分失败（上传任务已经完成）3-上传成功
}

// StartPublishGoodsRequest 启动批量发布道具任务，请求参数
// 1. 使用支付签名
// POST，请求参数为 json 字符串，Content-Type 为 application/json
type StartPublishGoodsRequest struct {
	Env         Env            `json:"env"`          // 环境 0-正式环境 1-沙箱环境
	PublishItem []*PublishItem `json:"publish_item"` // 道具信息 发布的商品列表
}

// PublishItem 道具信息
type PublishItem struct {
	ID            string `json:"id"`                       // 道具 id，添加到开发环境时传的道具 id，长度 (0,64]，字符只允许使用字母、数字、'_'、'-'
	PublishStatus int    `json:"publish_status,omitempty"` // 发布状态 0-上传中 1-id 已经存在 2-发布成功 3-发布失败
	ErrMsg        string `json:"errmsg,omitempty"`         // 发布失败的原因
}

// StartPublishGoodsResponse 启动批量发布道具任务响应参数
type StartPublishGoodsResponse struct {
	util.CommonError
}

// QueryPublishGoodsRequest 查询批量发布道具任务，请求参数
// 1. 使用支付签名
// POST，请求参数为 json 字符串，Content-Type 为 application/json
type QueryPublishGoodsRequest struct {
	Env Env `json:"env"` // 环境 0-正式环境 1-沙箱环境
}

// QueryPublishGoodsResponse 查询批量发布道具任务响应参数
type QueryPublishGoodsResponse struct {
	util.CommonError
	PublishItem []*PublishItem `json:"publish_item"` // 道具信息列表
	Status      int            `json:"status"`       // 任务状态 0-无任务在运行 1-任务运行中 2-上传失败或部分失败（上传任务已经完成）3-上传成功

}

// AsyncXPayGoodsDeliverNotifyRequest 异步通知发货，请求参数
// 1. 使用支付签名
// POST，请求参数为 json 字符串，Content-Type 为 application/json
type AsyncXPayGoodsDeliverNotifyRequest struct {
	ToUserName    string         `json:"ToUserName"`    // 小程序的原始 ID
	FromUserName  string         `json:"FromUserName"`  // 发送方帐号（一个 OpenID）该事件消息的 openid，道具发货场景固定为微信官方的 openid
	CreateTime    int            `json:"CreateTime"`    // 消息发送时间（整型）
	MsgType       string         `json:"MsgType"`       // 消息类型，此时固定为：event
	Event         string         `json:"Event"`         // 事件类型，此时固定为：xpay_goods_deliver_notify
	Openid        string         `json:"openid"`        // 用户 openid
	OutTradeNo    string         `json:"OutTradeNo"`    // 业务订单号
	Env           Env            `json:"env"`           // 环境 0-正式环境 1-沙箱环境
	WechatPayInfo *WeChatPayInfo `json:"WechatPayInfo"` // 微信支付订单信息
	GoodsInfo     *GoodsInfo     `json:"GoodsInfo"`     // 道具信息
	TeamInfo      *TeamInfo      `json:"TeamInfo"`      // 拼团信息
}

// WeChatPayInfo 微信支付信息 非微信支付渠道可能没有
type WeChatPayInfo struct {
	MchOrderNo    string `json:"MchOrderNo"`    // 商户订单号
	TransactionID string `json:"TransactionId"` // 微信支付订单号
	PaidTime      int64  `json:"PaidTime"`      // 用户支付时间，Linux 秒级时间戳
}

// GoodsInfo 道具参数信息
type GoodsInfo struct {
	ProductID   string `json:"ProductId"`   // 道具 ID
	Quantity    int    `json:"Quantity"`    // 数量
	OrigPrice   int    `json:"OrigPrice"`   // 物品原始价格（单位：分）
	ActualPrice int    `json:"ActualPrice"` // 物品实际支付价格（单位：分）
	Attach      string `json:"Attach"`      // 透传信息
}

// TeamInfo 拼团信息
type TeamInfo struct {
	ActivityID string `json:"ActivityId"` // 活动 id
	TeamID     string `json:"TeamId"`     // 团 id
	TeamType   int    `json:"TeamType"`   // 团类型 1-支付全部，拼成退款
	TeamAction int    `json:"TeamAction"` // 0-创团 1-参团
}

// AsyncXPayGoodsDeliverNotifyResponse 异步通知发货响应参数
type AsyncXPayGoodsDeliverNotifyResponse struct {
	util.CommonError
}

// AsyncXPayCoinPayNotifyRequest 异步通知代币支付推送，请求参数
type AsyncXPayCoinPayNotifyRequest struct {
	ToUserName    string         `json:"ToUserName"`    // 小程序的原始 ID
	FromUserName  string         `json:"FromUserName"`  // 发送方帐号（一个 OpenID）该事件消息的 openid，道具发货场景固定为微信官方的 openid
	CreateTime    int            `json:"CreateTime"`    // 消息发送时间（整型）
	MsgType       string         `json:"MsgType"`       // 消息类型，此时固定为：event
	Event         string         `json:"Event"`         // 事件类型，此时固定为：xpay_goods_deliver_notify
	Openid        string         `json:"openid"`        // 用户 openid
	OutTradeNo    string         `json:"OutTradeNo"`    // 业务订单号
	Env           Env            `json:"env"`           // 环境 0-正式环境 1-沙箱环境
	WechatPayInfo *WeChatPayInfo `json:"WechatPayInfo"` // 微信支付订单信息
	CoinInfo      *CoinInfo      `json:"GoodsInfo"`     // 道具信息
}

// CoinInfo 代币信息
type CoinInfo struct {
	Quantity    int    `json:"Quantity"`    // 数量
	OrigPrice   int    `json:"OrigPrice"`   // 物品原始价格（单位：分）
	ActualPrice int    `json:"ActualPrice"` // 物品实际支付价格（单位：分）
	Attach      string `json:"Attach"`      // 透传信息
}

// AsyncXPayCoinPayNotifyResponse 异步通知代币支付推送响应参数
type AsyncXPayCoinPayNotifyResponse struct {
	util.CommonError
}

// AsyncXPayRefundNotifyRequest 异步通知退款推送，请求参数
type AsyncXPayRefundNotifyRequest struct {
	ToUserName               string    `json:"ToUserName"`               // 小程序的原始 ID
	FromUserName             string    `json:"FromUserName"`             // 发送方帐号（一个 OpenID）
	CreateTime               int       `json:"CreateTime"`               // 消息发送时间（整型）
	MsgType                  string    `json:"MsgType"`                  // 消息类型，此时固定为：event
	Event                    string    `json:"Event"`                    // 事件类型，此时固定为：xpay_refund_notify
	Openid                   string    `json:"openid"`                   // 用户 openid
	WxRefundID               string    `json:"WxRefundId"`               // 微信退款单号
	MchRefundID              string    `json:"MchRefundId"`              // 商户退款单号
	WxOrderID                string    `json:"WxOrderId"`                // 退款单对应支付单的微信单号
	MchOrderID               string    `json:"MchOrderId"`               // 退款单对应支付单的商户单号
	RefundFee                int       `json:"RefundFee"`                // 退款金额，单位分
	RetCode                  int       `json:"RetCode"`                  // 退款结果，0 为成功，非0 为失败
	RetMsg                   string    `json:"RetMsg"`                   // 退款结果详情
	RefundStartTimestamp     int64     `json:"RefundStartTimestamp"`     // 开始退款时间，秒级时间戳
	RefundSuccTimestamp      int64     `json:"RefundSuccTimestamp"`      // 结束退款时间，秒级时间戳
	WxpayRefundTransactionID string    `json:"WxpayRefundTransactionId"` // 退款单的微信支付单号
	RetryTimes               int       `json:"RetryTimes"`               // 重试次数，从 0 开始
	TeamInfo                 *TeamInfo `json:"TeamInfo"`                 // 拼团信息
}

// AsyncXPayRefundNotifyResponse 异步通知退款推送响应参数
type AsyncXPayRefundNotifyResponse struct {
	util.CommonError
}

// AsyncXPaySubscribeIosRefundQueryNotifyRequest iOS Apple 支付退款问询消息
// 文档：https://developers.weixin.qq.com/miniprogram/dev/platform-capabilities/business-capabilities/virtual-payment/ios.html
type AsyncXPaySubscribeIosRefundQueryNotifyRequest struct {
	ToUserName   string `json:"ToUserName"`   // 小程序的原始 ID
	FromUserName string `json:"FromUserName"` // 发送方帐号
	CreateTime   int    `json:"CreateTime"`   // 消息发送时间
	MsgType      string `json:"MsgType"`      // 消息类型，此时固定为 event
	Event        string `json:"Event"`        // 事件类型，此时固定为 xpay_subscribe_ios_refund_query_notify

	RefundTime          string `json:"refund_time"`           // 问询时间，Unix 时间戳
	OrderTime           string `json:"order_time"`            // 该笔退款的订单时间，Unix 时间戳
	ChannelBill         string `json:"channel_bill"`          // Apple 支付票据号
	BundleID            string `json:"bundleid"`              // 应用的 Apple bundleid
	ProductID           string `json:"product_id"`            // 道具 id
	PCount              string `json:"p_count"`               // 道具/代币数量
	RefundRequestReason string `json:"refund_request_reason"` // 用户请求退款的原因
	ProvideStatus       string `json:"provide_status"`        // 发货状态，0：未发货 1：已发货 2：发货中
	PayOrderID          string `json:"pay_order_id"`          // 退款对应支付订单号
}

// AsyncXPaySubscribeIosRefundQueryNotifyResponse iOS Apple 支付退款问询应答
type AsyncXPaySubscribeIosRefundQueryNotifyResponse struct {
	ResultCode int32  `json:"result_code"` // 结果码，0-放过，建议退款；1-拦截，拒绝退款
	ResultInfo string `json:"result_info"` // 结果描述
	Evidence   string `json:"evidence"`    // 决策凭据，必填，用于退款审计
}

// AsyncXPayComplaintNotifyRequest 异步通知用户投诉推送，请求参数
type AsyncXPayComplaintNotifyRequest struct {
	ToUserName      string `json:"ToUserName"`      // 小程序的原始 ID
	FromUserName    string `json:"FromUserName"`    // 发送方帐号（一个 OpenID）
	CreateTime      int    `json:"CreateTime"`      // 消息发送时间（整型）
	MsgType         string `json:"MsgType"`         // 消息类型，此时固定为：event
	Event           string `json:"Event"`           // 事件类型，此时固定为：xpay_complaint_notify
	Openid          string `json:"openid"`          // 用户 openid
	WxOrderID       string `json:"WxOrderId"`       // 微信单号
	MchOrderID      string `json:"MchOrderId"`      // 商户单号
	TransactionID   string `json:"TransactionId"`   // 微信支付交易单号
	ComplaintID     string `json:"ComplaintId"`     // 投诉单号
	ComplaintDetail string `json:"ComplaintDetail"` // 投诉详情
	ComplaintTime   int64  `json:"ComplaintTime"`   // 投诉时间，秒级时间戳
	RetryTimes      int    `json:"RetryTimes"`      // 重试次数，从 0 开始
	RequestID       string `json:"RequestId"`       // 请求编号
}

// AsyncXPayComplaintNotifyResponse 异步通知用户投诉推送响应参数
type AsyncXPayComplaintNotifyResponse struct {
	util.CommonError
}

// ==================== 下载订单 ====================

// StartDownloadOrderRequest 发起下载小程序订单明细任务，请求参数
type StartDownloadOrderRequest struct {
	Env     Env    `json:"env"`      // 环境 0-正式环境 1-沙箱环境
	BeginDs string `json:"begin_ds"` // 账单开始日期，格式为 yyyymmdd
	EndDs   string `json:"end_ds"`   // 账单结束日期，格式为 yyyymmdd
}

// StartDownloadOrderResponse 发起下载小程序订单明细任务 响应参数
type StartDownloadOrderResponse struct {
	util.CommonError
}

// QueryDownloadOrderRequest 查询下载订单任务结果，请求参数
type QueryDownloadOrderRequest struct {
	Env     Env    `json:"env"`      // 环境 0-正式环境 1-沙箱环境
	BeginDs string `json:"begin_ds"` // 账单开始日期，格式为 yyyymmdd
	EndDs   string `json:"end_ds"`   // 账单结束日期，格式为 yyyymmdd
}

// QueryDownloadOrderResponse 查询下载订单任务结果 响应参数
type QueryDownloadOrderResponse struct {
	util.CommonError
	URL string `json:"url"` // 订单下载地址
}

// ==================== 资金管理 ====================

// QueryBizBalanceRequest 查询商家账户可提现余额，请求参数
type QueryBizBalanceRequest struct {
	Env Env `json:"env"` // 环境 0-正式环境 1-沙箱环境（仅作为签名校验，查询结果都是正式环境的）
}

// BizBalanceAvailable 可提现余额
type BizBalanceAvailable struct {
	Amount       string `json:"amount"`        // 可提现余额，单位元
	CurrencyCode string `json:"currency_code"` // 币种（一般为 CNY）
}

// QueryBizBalanceResponse 查询商家账户可提现余额 响应参数
type QueryBizBalanceResponse struct {
	util.CommonError
	BalanceAvailable *BizBalanceAvailable `json:"balance_available"` // 可提现余额
}

// ==================== 广告金 ====================

// TransferAccount 广告金充值账户信息
type TransferAccount struct {
	TransferAccountName       string `json:"transfer_account_name"`        // 账户名称
	TransferAccountUID        int64  `json:"transfer_account_uid"`         // 账户 UID
	TransferAccountAgencyID   int64  `json:"transfer_account_agency_id"`   // 代理商 ID
	TransferAccountAgencyName string `json:"transfer_account_agency_name"` // 代理商名称
	State                     int    `json:"state"`                        // 审核状态 0-待审核 1-已通过 2-已驳回
	BindResult                int    `json:"bind_result"`                  // 绑定结果 1-绑定成功 2-绑定失败
	ErrorMsg                  string `json:"error_msg"`                    // 错误信息
}

// QueryTransferAccountRequest 查询广告金充值账户，请求参数
type QueryTransferAccountRequest struct {
	Env Env `json:"env"` // 环境 0-正式环境 1-沙箱环境
}

// QueryTransferAccountResponse 查询广告金充值账户 响应参数
type QueryTransferAccountResponse struct {
	util.CommonError
	AcctList []*TransferAccount `json:"acct_list"` // 充值账户列表
}

// AdverFundsFilter 广告金发放记录查询过滤条件
type AdverFundsFilter struct {
	SettleBegin int64 `json:"settle_begin,omitempty"` // 结算周期开始时间，unix 秒级时间戳
	SettleEnd   int64 `json:"settle_end,omitempty"`   // 结算周期结束时间，unix 秒级时间戳
	FundType    int   `json:"fund_type,omitempty"`    // 资金类型 0-普通赠送 1-广告激励 2-定向激励
}

// AdverFundsRecord 广告金发放记录
type AdverFundsRecord struct {
	SettleBegin  int64  `json:"settle_begin"`  // 结算周期开始时间，unix 秒级时间戳
	SettleEnd    int64  `json:"settle_end"`    // 结算周期结束时间，unix 秒级时间戳
	TotalAmount  int64  `json:"total_amount"`  // 发放总金额，单位分
	RemainAmount int64  `json:"remain_amount"` // 剩余可用金额，单位分
	ExpireTime   int64  `json:"expire_time"`   // 过期时间，unix 秒级时间戳
	FundType     int    `json:"fund_type"`     // 资金类型 0-普通赠送 1-广告激励 2-定向激励
	FundID       string `json:"fund_id"`       // 广告金发放 ID
}

// QueryAdverFundsRequest 查询广告金发放记录，请求参数
type QueryAdverFundsRequest struct {
	Env      Env               `json:"env"`                 // 环境 0-正式环境 1-沙箱环境
	Page     int               `json:"page,omitempty"`      // 页码，>= 1
	PageSize int               `json:"page_size,omitempty"` // 每页记录数
	Filter   *AdverFundsFilter `json:"filter,omitempty"`    // 查询过滤条件
}

// QueryAdverFundsResponse 查询广告金发放记录 响应参数
type QueryAdverFundsResponse struct {
	util.CommonError
	AdverFundsList []*AdverFundsRecord `json:"adver_funds_list"` // 发放记录列表
	TotalPage      int                 `json:"total_page"`       // 总页数
}

// CreateFundsBillRequest 充值广告金，请求参数
type CreateFundsBillRequest struct {
	Env                     Env    `json:"env"`                        // 环境 0-正式环境 1-沙箱环境
	TransferAmount          int64  `json:"transfer_amount"`            // 充值金额，单位分
	TransferAccountUID      int64  `json:"transfer_account_uid"`       // 充值账户 UID
	TransferAccountName     string `json:"transfer_account_name"`      // 充值账户名称
	TransferAccountAgencyID int64  `json:"transfer_account_agency_id"` // 代理商 ID
	RequestID               string `json:"request_id"`                 // 幂等请求 ID，最长 1024 字符
	SettleBegin             int64  `json:"settle_begin"`               // 结算周期开始时间，unix 秒级时间戳
	SettleEnd               int64  `json:"settle_end"`                 // 结算周期结束时间，unix 秒级时间戳
	AuthorizeAdvertise      int    `json:"authorize_advertise"`        // 是否授权广告数据 0-否 1-是
	FundType                int    `json:"fund_type"`                  // 资金类型 0-普通赠送 1-广告激励 2-定向激励
}

// CreateFundsBillResponse 充值广告金 响应参数
type CreateFundsBillResponse struct {
	util.CommonError
	BillID string `json:"bill_id"` // 充值订单号
}

// BindTransferAccountRequest 绑定广告金充值账户，请求参数
type BindTransferAccountRequest struct {
	Env                    Env    `json:"env"`                                 // 环境 0-正式环境 1-沙箱环境
	TransferAccountUID     int64  `json:"transfer_account_uid,omitempty"`      // 充值账户 UID
	TransferAccountOrgName string `json:"transfer_account_org_name,omitempty"` // 充值账户主体名称
}

// BindTransferAccountResponse 绑定广告金充值账户 响应参数
type BindTransferAccountResponse struct {
	util.CommonError
}

// FundsBillFilter 广告金充值记录查询过滤条件
type FundsBillFilter struct {
	OperTimeBegin int64  `json:"oper_time_begin"`      // 充值开始时间，unix 秒级时间戳
	OperTimeEnd   int64  `json:"oper_time_end"`        // 充值结束时间，unix 秒级时间戳
	BillID        string `json:"bill_id,omitempty"`    // 充值订单号
	RequestID     string `json:"request_id,omitempty"` // 幂等请求 ID
}

// FundsBillRecord 广告金充值记录
type FundsBillRecord struct {
	BillID              string `json:"bill_id"`               // 充值订单号
	OperTime            int64  `json:"oper_time"`             // 充值时间，unix 秒级时间戳
	SettleBegin         int64  `json:"settle_begin"`          // 对应广告金结算开始时间，unix 秒级时间戳
	SettleEnd           int64  `json:"settle_end"`            // 对应广告金结算结束时间，unix 秒级时间戳
	FundID              string `json:"fund_id"`               // 对应广告金发放 ID
	TransferAccountName string `json:"transfer_account_name"` // 充值账户名称
	TransferAccountUID  int64  `json:"transfer_account_uid"`  // 充值账户 UID
	TransferAmount      int64  `json:"transfer_amount"`       // 充值金额，单位分
	Status              int    `json:"status"`                // 状态 0-充值中 1-成功 2-失败
	RequestID           string `json:"request_id"`            // 幂等请求 ID
}

// QueryFundsBillRequest 查询广告金充值记录，请求参数
type QueryFundsBillRequest struct {
	Env      Env             `json:"env"`       // 环境 0-正式环境 1-沙箱环境
	Page     int             `json:"page"`      // 页码，>= 1
	PageSize int             `json:"page_size"` // 每页记录数
	Filter   FundsBillFilter `json:"filter"`    // 查询过滤条件
}

// QueryFundsBillResponse 查询广告金充值记录 响应参数
type QueryFundsBillResponse struct {
	util.CommonError
	BillList  []*FundsBillRecord `json:"bill_list"`  // 充值记录列表
	TotalPage int                `json:"total_page"` // 总页数
}

// RecoverBillFilter 广告金回收记录查询过滤条件
type RecoverBillFilter struct {
	RecoverTimeBegin int64  `json:"recover_time_begin"` // 回收开始时间，unix 秒级时间戳
	RecoverTimeEnd   int64  `json:"recover_time_end"`   // 回收结束时间，unix 秒级时间戳
	BillID           string `json:"bill_id,omitempty"`  // 回收订单号
}

// RecoverBillRecord 广告金回收记录
type RecoverBillRecord struct {
	BillID             string   `json:"bill_id"`              // 回收订单号
	RecoverTime        int64    `json:"recover_time"`         // 回收时间，unix 秒级时间戳
	SettleBegin        int64    `json:"settle_begin"`         // 结算周期开始时间，unix 秒级时间戳
	SettleEnd          int64    `json:"settle_end"`           // 结算周期结束时间，unix 秒级时间戳
	FundID             string   `json:"fund_id"`              // 对应广告金发放 ID
	RecoverAccountName string   `json:"recover_account_name"` // 回收账户
	RecoverAmount      int64    `json:"recover_amount"`       // 回收金额，单位分
	RefundOrderList    []string `json:"refund_order_list"`    // 对应的退款订单号列表
}

// QueryRecoverBillRequest 查询广告金回收记录，请求参数
type QueryRecoverBillRequest struct {
	Env      Env               `json:"env"`       // 环境 0-正式环境 1-沙箱环境
	Page     int               `json:"page"`      // 页码，>= 1
	PageSize int               `json:"page_size"` // 每页记录数
	Filter   RecoverBillFilter `json:"filter"`    // 查询过滤条件
}

// QueryRecoverBillResponse 查询广告金回收记录 响应参数
type QueryRecoverBillResponse struct {
	util.CommonError
	BillList  []*RecoverBillRecord `json:"bill_list"`  // 回收记录列表
	TotalPage int                  `json:"total_page"` // 总页数
}

// DownloadAdverFundsOrderRequest 下载广告金对应的商户订单信息，请求参数
type DownloadAdverFundsOrderRequest struct {
	Env    Env    `json:"env"`     // 环境 0-正式环境 1-沙箱环境
	FundID string `json:"fund_id"` // 广告金发放 ID
}

// DownloadAdverFundsOrderResponse 下载广告金对应的商户订单信息 响应参数
type DownloadAdverFundsOrderResponse struct {
	util.CommonError
	URL string `json:"url"` // 订单下载地址
}

// ==================== 投诉处理 ====================

// ComplaintOrderInfo 投诉关联订单信息
type ComplaintOrderInfo struct {
	TransactionID string `json:"transaction_id"` // 微信支付交易单号
	MchOrderNo    string `json:"mch_order_no"`   // 商户订单号
	RefundID      string `json:"refund_id"`      // 退款订单号
}

// ComplaintMedia 投诉媒体信息
type ComplaintMedia struct {
	MediaType int      `json:"media_type"` // 媒体类型
	MediaURL  string   `json:"media_url"`  // 媒体 URL
	MediaTags []string `json:"media_tags"` // 媒体标签
}

// ComplaintItem 投诉详情
type ComplaintItem struct {
	ComplaintID           string                `json:"complaint_id"`            // 投诉单号
	ComplaintTime         string                `json:"complaint_time"`          // 投诉时间
	ComplaintDetail       string                `json:"complaint_detail"`        // 投诉内容
	ComplaintState        string                `json:"complaint_state"`         // 投诉状态 PENDING/PROCESSING/PROCESSED
	PayerPhone            string                `json:"payer_phone"`             // 投诉人联系方式
	PayerOpenID           string                `json:"payer_openid"`            // 投诉人 OpenID
	ComplaintOrderInfo    []*ComplaintOrderInfo `json:"complaint_order_info"`    // 关联订单信息
	ComplaintFullRefunded bool                  `json:"complaint_full_refunded"` // 是否全部退款
	IncomingUserResponse  bool                  `json:"incoming_user_response"`  // 是否有待处理的用户消息
	UserComplaintTimes    int                   `json:"user_complaint_times"`    // 用户投诉次数
	ComplaintMediaList    []*ComplaintMedia     `json:"complaint_media_list"`    // 用户上传的证据
}

// GetComplaintListRequest 获取投诉列表，请求参数
type GetComplaintListRequest struct {
	Env       Env    `json:"env"`        // 环境 0-正式环境 1-沙箱环境
	BeginDate string `json:"begin_date"` // 查询开始日期，格式 yyyy-mm-dd
	EndDate   string `json:"end_date"`   // 查询结束日期，格式 yyyy-mm-dd
	Offset    int    `json:"offset"`     // 偏移量，从 0 开始
	Limit     int    `json:"limit"`      // 最大返回记录数
}

// GetComplaintListResponse 获取投诉列表 响应参数
type GetComplaintListResponse struct {
	util.CommonError
	Total      int              `json:"total"`      // 总数
	Complaints []*ComplaintItem `json:"complaints"` // 投诉列表
}

// GetComplaintDetailRequest 获取投诉详情，请求参数
type GetComplaintDetailRequest struct {
	Env         Env    `json:"env"`          // 环境 0-正式环境 1-沙箱环境
	ComplaintID string `json:"complaint_id"` // 投诉单号
}

// GetComplaintDetailResponse 获取投诉详情 响应参数
type GetComplaintDetailResponse struct {
	util.CommonError
	Complaint *ComplaintItem `json:"complaint"` // 投诉详情
}

// NegotiationHistory 协商历史记录
type NegotiationHistory struct {
	LogID              string            `json:"log_id"`               // 操作流水号
	Operator           string            `json:"operator"`             // 操作人
	OperateTime        string            `json:"operate_time"`         // 操作时间
	OperateType        string            `json:"operate_type"`         // 操作类型
	OperateDetails     string            `json:"operate_details"`      // 操作详情
	ComplaintMediaList []*ComplaintMedia `json:"complaint_media_list"` // 上传的证据
}

// GetNegotiationHistoryRequest 获取协商历史，请求参数
type GetNegotiationHistoryRequest struct {
	Env         Env    `json:"env"`          // 环境 0-正式环境 1-沙箱环境
	ComplaintID string `json:"complaint_id"` // 投诉单号
	Offset      int    `json:"offset"`       // 偏移量，从 0 开始
	Limit       int    `json:"limit"`        // 最大返回记录数
}

// GetNegotiationHistoryResponse 获取协商历史 响应参数
type GetNegotiationHistoryResponse struct {
	util.CommonError
	Total   int                   `json:"total"`   // 总数
	History []*NegotiationHistory `json:"history"` // 协商历史列表
}

// ResponseComplaintRequest 回复用户投诉，请求参数
type ResponseComplaintRequest struct {
	Env             Env      `json:"env"`              // 环境 0-正式环境 1-沙箱环境
	ComplaintID     string   `json:"complaint_id"`     // 投诉单号
	ResponseContent string   `json:"response_content"` // 回复内容
	ResponseImages  []string `json:"response_images"`  // 图片文件 ID 列表（来自 upload_vp_file）
}

// ResponseComplaintResponse 回复用户投诉 响应参数
type ResponseComplaintResponse struct {
	util.CommonError
}

// CompleteComplaintRequest 完成投诉处理，请求参数
type CompleteComplaintRequest struct {
	Env         Env    `json:"env"`          // 环境 0-正式环境 1-沙箱环境
	ComplaintID string `json:"complaint_id"` // 投诉单号
}

// CompleteComplaintResponse 完成投诉处理 响应参数
type CompleteComplaintResponse struct {
	util.CommonError
}

// UploadVPFileRequest 上传媒体文件，请求参数
type UploadVPFileRequest struct {
	Env       Env    `json:"env"`                  // 环境 0-正式环境 1-沙箱环境
	Base64Img string `json:"base64_img,omitempty"` // Base64 编码的图片，最大 1MB
	ImgURL    string `json:"img_url,omitempty"`    // 图片 URL（可直接下载，不支持 302 跳转），最大 2MB，优先使用此字段
	FileName  string `json:"file_name"`            // 图片名称
}

// UploadVPFileResponse 上传媒体文件 响应参数
type UploadVPFileResponse struct {
	util.CommonError
	FileID string `json:"file_id"` // 返回的文件 ID（用于 response_complaint）
}

// GetUploadFileSignRequest 获取微信支付反馈投诉图片的签名头部，请求参数
type GetUploadFileSignRequest struct {
	Env         Env    `json:"env"`          // 环境 0-正式环境 1-沙箱环境
	WxpayURL    string `json:"wxpay_url"`    // 微信支付图片 URL，格式 "https://api.mch.weixin.qq.com/v3/merchant-service/images/{xxxxxx}"
	ConvertCOS  bool   `json:"convert_cos"`  // 是否转换为 COS，获取临时下载链接（有效期 30 分钟）
	ComplaintID string `json:"complaint_id"` // 对应的投诉单号
}

// GetUploadFileSignResponse 获取微信支付反馈投诉图片的签名头部 响应参数
type GetUploadFileSignResponse struct {
	util.CommonError
	Sign   string `json:"sign"`    // Authorization 头部值
	CosURL string `json:"cos_url"` // 当 convert_cos=true 时返回的 COS URL，有效期 30 分钟
}

// URLParams url parameter
type URLParams struct {
	Path        string `json:"path"`
	AccessToken string `json:"access_token"`
	PaySign     string `json:"paySign"`
	Signature   string `json:"signature"`
	Content     string `json:"content"`
}
