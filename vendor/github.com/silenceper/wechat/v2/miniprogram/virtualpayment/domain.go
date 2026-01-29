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
// https://developers.weixin.qq.com/miniprogram/dev/platform-capabilities/industry/virtual-payment.html#_2-3-%E6%9C%8D%E5%8A%A1%E5%99%A8API
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
}

// WeChatPayInfo 微信支付信息 非微信支付渠道可能没有
type WeChatPayInfo struct {
	MchOrderNo    string `json:"MchOrderNo"`    // 商户订单号
	TransactionID string `json:"TransactionId"` // 微信支付订单号
}

// GoodsInfo 道具参数信息
type GoodsInfo struct {
	ProductID   string `json:"ProductId"`   // 道具 ID
	Quantity    int    `json:"Quantity"`    // 数量
	OrigPrice   int    `json:"OrigPrice"`   // 物品原始价格（单位：分）
	ActualPrice int    `json:"ActualPrice"` // 物品实际支付价格（单位：分）
	Attach      string `json:"Attach"`      // 透传信息
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

// URLParams url parameter
type URLParams struct {
	Path        string `json:"path"`
	AccessToken string `json:"access_token"`
	PaySign     string `json:"paySign"`
	Signature   string `json:"signature"`
	Content     string `json:"content"`
}
