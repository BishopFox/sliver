package order

import (
	"fmt"
	"time"

	"github.com/silenceper/wechat/v2/miniprogram/context"
	"github.com/silenceper/wechat/v2/util"
)

const (
	// 发货信息录入
	uploadShippingInfoURL = "https://api.weixin.qq.com/wxa/sec/order/upload_shipping_info?access_token=%s"

	// 查询订单发货状态
	getShippingOrderURL = "https://api.weixin.qq.com/wxa/sec/order/get_order?access_token=%s"

	// 查询订单列表
	getShippingOrderListURL = "https://api.weixin.qq.com/wxa/sec/order/get_order_list?access_token=%s"

	// 确认收货提醒接口
	notifyConfirmReceiveURL = "https://api.weixin.qq.com/wxa/sec/order/notify_confirm_receive?access_token=%s"
)

// Shipping 发货信息管理
type Shipping struct {
	*context.Context
}

// NewShipping init
func NewShipping(ctx *context.Context) *Shipping {
	return &Shipping{ctx}
}

// UploadShippingInfo 发货信息录入
// see https://developers.weixin.qq.com/miniprogram/dev/platform-capabilities/business-capabilities/order-shipping/order-shipping.html
func (shipping *Shipping) UploadShippingInfo(in *UploadShippingInfoRequest) (err error) {
	accessToken, err := shipping.GetAccessToken()
	if err != nil {
		return
	}

	uri := fmt.Sprintf(uploadShippingInfoURL, accessToken)
	response, err := util.PostJSON(uri, in)
	if err != nil {
		return
	}

	// 使用通用方法返回错误
	return util.DecodeWithCommonError(response, "UploadShippingInfo")
}

// GetShippingOrder 查询订单发货状态
func (shipping *Shipping) GetShippingOrder(in *GetShippingOrderRequest) (res ShippingOrderResponse, err error) {
	accessToken, err := shipping.GetAccessToken()
	if err != nil {
		return
	}

	uri := fmt.Sprintf(getShippingOrderURL, accessToken)
	response, err := util.PostJSON(uri, in)
	if err != nil {
		return
	}

	err = util.DecodeWithError(response, &res, "GetShippingOrder")
	return
}

// GetShippingOrderList 查询订单列表
func (shipping *Shipping) GetShippingOrderList(in *GetShippingOrderListRequest) (res GetShippingOrderListResponse, err error) {
	accessToken, err := shipping.GetAccessToken()
	if err != nil {
		return
	}

	uri := fmt.Sprintf(getShippingOrderListURL, accessToken)
	response, err := util.PostJSON(uri, in)
	if err != nil {
		return
	}

	err = util.DecodeWithError(response, &res, "GetShippingOrderList")
	return
}

// NotifyConfirmReceive 确认收货提醒接口
func (shipping *Shipping) NotifyConfirmReceive(in *NotifyConfirmReceiveRequest) (err error) {
	accessToken, err := shipping.GetAccessToken()
	if err != nil {
		return
	}

	uri := fmt.Sprintf(notifyConfirmReceiveURL, accessToken)
	response, err := util.PostJSON(uri, in)
	if err != nil {
		return
	}

	// 使用通用方法返回错误
	return util.DecodeWithCommonError(response, "NotifyConfirmReceive")
}

// UploadShippingInfoRequest 发货信息录入请求参数
type UploadShippingInfoRequest struct {
	OrderKey       *ShippingOrderKey `json:"order_key"`        // 订单，需要上传物流信息的订单
	LogisticsType  LogisticsType     `json:"logistics_type"`   // 物流模式
	DeliveryMode   DeliveryMode      `json:"delivery_mode"`    // 发货模式
	IsAllDelivered bool              `json:"is_all_delivered"` // 分拆发货模式时必填，用于标识分拆发货模式下是否已全部发货完成
	ShippingList   []*ShippingInfo   `json:"shipping_list"`    // 物流信息列表，发货物流单列表，支持统一发货（单个物流单）和分拆发货（多个物流单）两种模式
	UploadTime     *time.Time        `json:"upload_time"`      // 上传时间，用于标识请求的先后顺序
	Payer          *ShippingPayer    `json:"payer"`            // 支付人信息
}

// ShippingOrderKey 订单
type ShippingOrderKey struct {
	OrderNumberType NumberType `json:"order_number_type"` // 订单单号类型，用于确认需要上传详情的订单。枚举值1，使用下单商户号和商户侧单号；枚举值2，使用微信支付单号。
	TransactionID   string     `json:"transaction_id"`    // 原支付交易对应的微信订单号
	Mchid           string     `json:"mchid"`             // 支付下单商户的商户号，由微信支付生成并下发
	OutTradeNo      string     `json:"out_trade_no"`      // 商户系统内部订单号，只能是数字、大小写字母`_-*`且在同一个商户号下唯一
}

// ShippingPayer 支付者信息
type ShippingPayer struct {
	Openid string `json:"openid"` // 用户标识，用户在小程序appid下的唯一标识
}

// ShippingInfo 物流信息
type ShippingInfo struct {
	TrackingNo     string          `json:"tracking_no"`     // 物流单号，物流快递发货时必填
	ExpressCompany string          `json:"express_company"` // 物流公司编码，快递公司ID，物流快递发货时必填；参见「查询物流公司编码列表」
	ItemDesc       string          `json:"item_desc"`       // 商品信息，例如：微信红包抱枕*1个，限120个字以内
	Contact        ShippingContact `json:"contact"`         // 联系方式，当发货的物流公司为顺丰时，联系方式为必填，收件人或寄件人联系方式二选一
}

// ShippingContact 联系方式
type ShippingContact struct {
	ConsignorContact string `json:"consignor_contact"` // 寄件人联系方式，寄件人联系方式，采用掩码传输，最后4位数字不能打掩码
	ReceiverContact  string `json:"receiver_contact"`  // 收件人联系方式，收件人联系方式，采用掩码传输，最后4位数字不能打掩码
}

// DeliveryMode 发货模式
type DeliveryMode uint8

const (
	// DeliveryModeUnifiedDelivery 统一发货
	DeliveryModeUnifiedDelivery DeliveryMode = 1
	// DeliveryModeSplitDelivery 分拆发货
	DeliveryModeSplitDelivery DeliveryMode = 2
)

// LogisticsType 物流模式
type LogisticsType uint8

const (
	// LogisticsTypeExpress 实体物流配送采用快递公司进行实体物流配送形式
	LogisticsTypeExpress LogisticsType = 1
	// LogisticsTypeSameCity 同城配送
	LogisticsTypeSameCity LogisticsType = 2
	// LogisticsTypeVirtual 虚拟商品，虚拟商品，例如话费充值，点卡等，无实体配送形式
	LogisticsTypeVirtual LogisticsType = 3
	// LogisticsTypeSelfPickup 用户自提
	LogisticsTypeSelfPickup LogisticsType = 4
)

// NumberType 订单单号类型
type NumberType uint8

const (
	// NumberTypeOutTradeNo 使用下单商户号和商户侧单号
	NumberTypeOutTradeNo NumberType = 1
	// NumberTypeTransactionID 使用微信支付单号
	NumberTypeTransactionID NumberType = 2
)

// GetShippingOrderRequest 查询订单发货状态参数
type GetShippingOrderRequest struct {
	TransactionID   string `json:"transaction_id"`    // 原支付交易对应的微信订单号
	MerchantID      string `json:"merchant_id"`       // 支付下单商户的商户号，由微信支付生成并下发
	SubMerchantID   string `json:"sub_merchant_id"`   //二级商户号
	MerchantTradeNo string `json:"merchant_trade_no"` //商户系统内部订单号，只能是数字、大小写字母`_-*`且在同一个商户号下唯一。
}

// ShippingItem 物流信息
type ShippingItem struct {
	TrackingNo     string `json:"tracking_no"`     // 物流单号，示例值: "323244567777
	ExpressCompany string `json:"express_company"` // 物流公司编码，快递公司ID，物流快递发货时必填；参见「查询物流公司编码列表」
	UploadTime     int64  `json:"upload_time"`     // 上传物流信息时间，时间戳形式
}

// ShippingDetail 发货信息
type ShippingDetail struct {
	DeliveryMode        DeliveryMode    `json:"delivery_mode"`         // 发货模式
	LogisticsType       LogisticsType   `json:"logistics_type"`        // 物流模式
	FinishShipping      bool            `json:"finish_shipping"`       // 是否已全部发货
	FinishShippingCount int             `json:"finish_shipping_count"` // 已完成全部发货的次数
	GoodsDesc           string          `json:"goods_desc"`            // 在小程序后台发货信息录入页录入的商品描述
	ShippingList        []*ShippingItem `json:"shipping_list"`         // 物流信息列表
}

// ShippingOrder 订单发货状态
type ShippingOrder struct {
	TransactionID   string          `json:"transaction_id"`    // 原支付交易对应的微信订单号
	MerchantTradeNo string          `json:"merchant_trade_no"` // 商户系统内部订单号，只能是数字、大小写字母`_-*`且在同一个商户号下唯一
	MerchantID      string          `json:"merchant_id"`       // 支付下单商户的商户号，由微信支付生成并下发
	SubMerchantID   string          `json:"sub_merchant_id"`   // 二级商户号
	Description     string          `json:"description"`       // 以分号连接的该支付单的所有商品描述，当超过120字时自动截断并以 “...” 结尾
	PaidAmount      int64           `json:"paid_amount"`       // 支付单实际支付金额，整型，单位：分钱
	Openid          string          `json:"openid"`            // 支付者openid
	TradeCreateTime int64           `json:"trade_create_time"` // 交易创建时间，时间戳形式
	PayTime         int64           `json:"pay_time"`          // 支付时间，时间戳形式
	InComplaint     bool            `json:"in_complaint"`      // 是否处在交易纠纷中
	OrderState      State           `json:"order_state"`       // 订单状态枚举：(1) 待发货；(2) 已发货；(3) 确认收货；(4) 交易完成；(5) 已退款
	Shipping        *ShippingDetail `json:"shipping"`          // 订单发货信息
}

// ShippingOrderResponse 查询订单发货状态返回参数
type ShippingOrderResponse struct {
	util.CommonError
	Order ShippingOrder `json:"order"` // 订单发货信息
}

// State 订单状态
type State uint8

const (
	// StateWaitShipment 待发货
	StateWaitShipment State = 1
	// StateShipped 已发货
	StateShipped State = 2
	// StateConfirm 确认收货
	StateConfirm State = 3
	// StateComplete 交易完成
	StateComplete State = 4
	// StateRefund 已退款
	StateRefund State = 5
)

// GetShippingOrderListRequest 查询订单列表请求参数
type GetShippingOrderListRequest struct {
	PayTimeRange *TimeRange `json:"pay_time_range"`        // 支付时间范围
	OrderState   State      `json:"order_state,omitempty"` // 订单状态
	Openid       string     `json:"openid,omitempty"`      // 支付者openid
	LastIndex    string     `json:"last_index,omitempty"`  // 	翻页时使用，获取第一页时不用传入，如果查询结果中 has_more 字段为 true，则传入该次查询结果中返回的 last_index 字段可获取下一页
	PageSize     int64      `json:"page_size"`             // 每页数量，最多50条
}

// TimeRange 时间范围
type TimeRange struct {
	BeginTime int64 `json:"begin_time,omitempty"` // 查询开始时间，时间戳形式
	EndTime   int64 `json:"end_time,omitempty"`   // 查询结束时间，时间戳形式
}

// GetShippingOrderListResponse 查询订单列表返回参数
type GetShippingOrderListResponse struct {
	util.CommonError
	OrderList []*ShippingOrder `json:"order_list"`
	LastIndex string           `json:"last_index"`
	HasMore   bool             `json:"has_more"`
}

// NotifyConfirmReceiveRequest 确认收货提醒接口请求参数
type NotifyConfirmReceiveRequest struct {
	TransactionID   string `json:"transaction_id"`    // 原支付交易对应的微信订单号
	MerchantID      string `json:"merchant_id"`       // 支付下单商户的商户号，由微信支付生成并下发
	SubMerchantID   string `json:"sub_merchant_id"`   // 二级商户号
	MerchantTradeNo string `json:"merchant_trade_no"` // 商户系统内部订单号，只能是数字、大小写字母`_-*`且在同一个商户号下唯一
	ReceivedTime    int64  `json:"received_time"`     // 收货时间，时间戳形式
}
