package express

import (
	"context"
	"fmt"

	"github.com/silenceper/wechat/v2/util"
)

// 同城配送 API URL
// https://developers.weixin.qq.com/miniprogram/dev/platform-capabilities/industry/express/business/intracity_service.html
const (
	// 开通门店权限
	intracityApplyURL = "https://api.weixin.qq.com/cgi-bin/express/intracity/apply?access_token=%s"
	// 创建门店
	intracityCreateStoreURL = "https://api.weixin.qq.com/cgi-bin/express/intracity/createstore?access_token=%s"
	// 查询门店
	intracityQueryStoreURL = "https://api.weixin.qq.com/cgi-bin/express/intracity/querystore?access_token=%s"
	// 更新门店
	intracityUpdateStoreURL = "https://api.weixin.qq.com/cgi-bin/express/intracity/updatestore?access_token=%s"
	// 门店运费充值
	intracityStoreChargeURL = "https://api.weixin.qq.com/cgi-bin/express/intracity/storecharge?access_token=%s"
	// 门店运费退款
	intracityStoreRefundURL = "https://api.weixin.qq.com/cgi-bin/express/intracity/storerefund?access_token=%s"
	// 门店运费流水查询
	intracityQueryFlowURL = "https://api.weixin.qq.com/cgi-bin/express/intracity/queryflow?access_token=%s"
	// 门店余额查询
	intracityBalanceQueryURL = "https://api.weixin.qq.com/cgi-bin/express/intracity/balancequery?access_token=%s"
	// 预下配送单（查询运费）
	intracityPreAddOrderURL = "https://api.weixin.qq.com/cgi-bin/express/intracity/preaddorder?access_token=%s"
	// 创建配送单
	intracityAddOrderURL = "https://api.weixin.qq.com/cgi-bin/express/intracity/addorder?access_token=%s"
	// 查询配送单
	intracityQueryOrderURL = "https://api.weixin.qq.com/cgi-bin/express/intracity/queryorder?access_token=%s"
	// 取消配送单
	intracityCancelOrderURL = "https://api.weixin.qq.com/cgi-bin/express/intracity/cancelorder?access_token=%s"
	// 模拟配送回调（仅用于测试）
	intracityMockNotifyURL = "https://api.weixin.qq.com/cgi-bin/express/intracity/mocknotify?access_token=%s"
)

// PayMode 充值/扣费主体
type PayMode string

const (
	// PayModeStore 门店
	PayModeStore PayMode = "PAY_MODE_STORE"
	// PayModeApp 小程序
	PayModeApp PayMode = "PAY_MODE_APP"
	// PayModeComponent 服务商
	PayModeComponent PayMode = "PAY_MODE_COMPONENT"
)

// OrderPattern 运力偏好
type OrderPattern uint32

const (
	// OrderPatternPriceFirst 价格优先
	OrderPatternPriceFirst OrderPattern = 1
	// OrderPatternTransFirst 运力优先
	OrderPatternTransFirst OrderPattern = 2
)

// FlowType 流水类型
type FlowType uint32

const (
	// FlowTypeCharge 充值流水
	FlowTypeCharge FlowType = 1
	// FlowTypeConsume 消费流水
	FlowTypeConsume FlowType = 2
	// FlowTypeRefund 退款流水
	FlowTypeRefund FlowType = 3
)

// IntracityDeliveryStatus 配送单状态
type IntracityDeliveryStatus int32

const (
	// IntracityDeliveryStatusReady 配送单待接单
	IntracityDeliveryStatusReady IntracityDeliveryStatus = 100
	// IntracityDeliveryStatusPickedUp 配送单待取货
	IntracityDeliveryStatusPickedUp IntracityDeliveryStatus = 101
	// IntracityDeliveryStatusOngoing 配送单配送中
	IntracityDeliveryStatusOngoing IntracityDeliveryStatus = 102
	// IntracityDeliveryStatusFinished 配送单已送达
	IntracityDeliveryStatusFinished IntracityDeliveryStatus = 200
	// IntracityDeliveryStatusCancelled 配送单已取消
	IntracityDeliveryStatusCancelled IntracityDeliveryStatus = 300
	// IntracityDeliveryStatusAbnormal 配送单异常
	IntracityDeliveryStatusAbnormal IntracityDeliveryStatus = 400
)

// IntracityAddressInfo 门店地址信息
type IntracityAddressInfo struct {
	Province string  `json:"province"`       // 省/自治区/直辖市
	City     string  `json:"city"`           // 地级市
	Area     string  `json:"area"`           // 县/县级市/区
	Street   string  `json:"street"`         // 街道
	House    string  `json:"house"`          // 具体门牌号或详细地址
	Lat      float64 `json:"lat"`            // 门店所在地纬度
	Lng      float64 `json:"lng"`            // 门店所在地经度
	Phone    string  `json:"phone"`          // 门店联系电话
	Name     string  `json:"name,omitempty"` // 联系人姓名（收货地址时使用）
}

// IntracityStoreInfo 门店信息
type IntracityStoreInfo struct {
	WxStoreID          string               `json:"wx_store_id"`          // 微信门店编号
	OutStoreID         string               `json:"out_store_id"`         // 自定义门店编号
	CityID             string               `json:"city_id"`              // 门店所在城市ID
	StoreName          string               `json:"store_name"`           // 门店名称
	OrderPattern       OrderPattern         `json:"order_pattern"`        // 运力偏好
	ServiceTransPrefer string               `json:"service_trans_prefer"` // 优先使用的运力ID
	AddressInfo        IntracityAddressInfo `json:"address_info"`         // 门店地址信息
}

// ============ 门店管理接口 ============

// IntracityApply 开通门店权限
// https://developers.weixin.qq.com/miniprogram/dev/platform-capabilities/industry/express/business/intracity_service.html
func (express *Express) IntracityApply(ctx context.Context) error {
	accessToken, err := express.GetAccessToken()
	if err != nil {
		return err
	}

	uri := fmt.Sprintf(intracityApplyURL, accessToken)
	response, err := util.PostJSONContext(ctx, uri, map[string]interface{}{})
	if err != nil {
		return err
	}

	return util.DecodeWithCommonError(response, "IntracityApply")
}

// CreateStoreRequest 创建门店请求参数
type CreateStoreRequest struct {
	OutStoreID         string               `json:"out_store_id"`                   // 自定义门店编号
	StoreName          string               `json:"store_name"`                     // 门店名称
	OrderPattern       OrderPattern         `json:"order_pattern,omitempty"`        // 运力偏好：1-价格优先，2-运力优先
	ServiceTransPrefer string               `json:"service_trans_prefer,omitempty"` // 优先使用的运力ID，order_pattern=2时必填
	AddressInfo        IntracityAddressInfo `json:"address_info"`                   // 门店地址信息
}

// CreateStoreResponse 创建门店返回参数
type CreateStoreResponse struct {
	util.CommonError
	WxStoreID  string `json:"wx_store_id"`  // 微信门店编号
	AppID      string `json:"appid"`        // 小程序appid
	OutStoreID string `json:"out_store_id"` // 自定义门店ID
}

// IntracityCreateStore 创建门店
func (express *Express) IntracityCreateStore(ctx context.Context, req *CreateStoreRequest) (res CreateStoreResponse, err error) {
	accessToken, err := express.GetAccessToken()
	if err != nil {
		return
	}

	uri := fmt.Sprintf(intracityCreateStoreURL, accessToken)
	response, err := util.PostJSONContext(ctx, uri, req)
	if err != nil {
		return
	}

	err = util.DecodeWithError(response, &res, "IntracityCreateStore")
	return
}

// QueryStoreRequest 查询门店请求参数
type QueryStoreRequest struct {
	WxStoreID  string `json:"wx_store_id,omitempty"`  // 微信门店编号
	OutStoreID string `json:"out_store_id,omitempty"` // 自定义门店编号
}

// QueryStoreResponse 查询门店返回参数
type QueryStoreResponse struct {
	util.CommonError
	Total     uint32               `json:"total"`      // 符合条件的门店总数
	AppID     string               `json:"appid"`      // 小程序appid
	StoreList []IntracityStoreInfo `json:"store_list"` // 门店信息列表
}

// IntracityQueryStore 查询门店
func (express *Express) IntracityQueryStore(ctx context.Context, req *QueryStoreRequest) (res QueryStoreResponse, err error) {
	accessToken, err := express.GetAccessToken()
	if err != nil {
		return
	}

	uri := fmt.Sprintf(intracityQueryStoreURL, accessToken)
	response, err := util.PostJSONContext(ctx, uri, req)
	if err != nil {
		return
	}

	err = util.DecodeWithError(response, &res, "IntracityQueryStore")
	return
}

// UpdateStoreKeyInfo 更新门店的key信息
type UpdateStoreKeyInfo struct {
	WxStoreID  string `json:"wx_store_id,omitempty"`  // 微信门店编号
	OutStoreID string `json:"out_store_id,omitempty"` // 自定义门店编号，二选一
}

// UpdateStoreContent 更新门店的内容
type UpdateStoreContent struct {
	StoreName          string                `json:"store_name,omitempty"`           // 门店名称
	OrderPattern       OrderPattern          `json:"order_pattern,omitempty"`        // 运力偏好
	ServiceTransPrefer string                `json:"service_trans_prefer,omitempty"` // 优先使用的运力ID
	AddressInfo        *IntracityAddressInfo `json:"address_info,omitempty"`         // 门店地址信息
}

// UpdateStoreRequest 更新门店请求参数
type UpdateStoreRequest struct {
	Keys    UpdateStoreKeyInfo `json:"keys"`    // 门店编号
	Content UpdateStoreContent `json:"content"` // 更新内容
}

// IntracityUpdateStore 更新门店
func (express *Express) IntracityUpdateStore(ctx context.Context, req *UpdateStoreRequest) error {
	accessToken, err := express.GetAccessToken()
	if err != nil {
		return err
	}

	uri := fmt.Sprintf(intracityUpdateStoreURL, accessToken)
	response, err := util.PostJSONContext(ctx, uri, req)
	if err != nil {
		return err
	}

	return util.DecodeWithCommonError(response, "IntracityUpdateStore")
}

// ============ 充值退款接口 ============

// StoreChargeRequest 门店运费充值请求参数
type StoreChargeRequest struct {
	WxStoreID      string  `json:"wx_store_id,omitempty"` // 微信门店编号，pay_mode=PAY_MODE_STORE时必传
	ServiceTransID string  `json:"service_trans_id"`      // 运力ID
	Amount         uint32  `json:"amount"`                // 充值金额，单位：分，50元起充
	PayMode        PayMode `json:"pay_mode,omitempty"`    // 充值主体
}

// StoreChargeResponse 门店运费充值返回参数
type StoreChargeResponse struct {
	util.CommonError
	PayURL    string `json:"payurl"`      // 充值页面地址
	AppID     string `json:"appid"`       // 小程序appid
	WxStoreID string `json:"wx_store_id"` // 微信门店编号
}

// IntracityStoreCharge 门店运费充值
func (express *Express) IntracityStoreCharge(ctx context.Context, req *StoreChargeRequest) (res StoreChargeResponse, err error) {
	accessToken, err := express.GetAccessToken()
	if err != nil {
		return
	}

	uri := fmt.Sprintf(intracityStoreChargeURL, accessToken)
	response, err := util.PostJSONContext(ctx, uri, req)
	if err != nil {
		return
	}

	err = util.DecodeWithError(response, &res, "IntracityStoreCharge")
	return
}

// StoreRefundRequest 门店运费退款请求参数
type StoreRefundRequest struct {
	WxStoreID      string  `json:"wx_store_id,omitempty"` // 微信门店编号
	PayMode        PayMode `json:"pay_mode,omitempty"`    // 充值/扣费主体
	ServiceTransID string  `json:"service_trans_id"`      // 运力ID
}

// StoreRefundResponse 门店运费退款返回参数
type StoreRefundResponse struct {
	util.CommonError
	AppID        string `json:"appid"`         // 小程序appid
	WxStoreID    string `json:"wx_store_id"`   // 微信门店编号
	RefundAmount uint32 `json:"refund_amount"` // 退款金额，单位：分
}

// IntracityStoreRefund 门店运费退款
func (express *Express) IntracityStoreRefund(ctx context.Context, req *StoreRefundRequest) (res StoreRefundResponse, err error) {
	accessToken, err := express.GetAccessToken()
	if err != nil {
		return
	}

	uri := fmt.Sprintf(intracityStoreRefundURL, accessToken)
	response, err := util.PostJSONContext(ctx, uri, req)
	if err != nil {
		return
	}

	err = util.DecodeWithError(response, &res, "IntracityStoreRefund")
	return
}

// QueryFlowRequest 门店运费流水查询请求参数
type QueryFlowRequest struct {
	WxStoreID      string   `json:"wx_store_id"`                // 微信门店编号
	FlowType       FlowType `json:"flow_type"`                  // 流水类型：1-充值，2-消费，3-退款
	ServiceTransID string   `json:"service_trans_id,omitempty"` // 运力ID
	BeginTime      uint32   `json:"begin_time,omitempty"`       // 开始时间戳
	EndTime        uint32   `json:"end_time,omitempty"`         // 结束时间戳
	PayMode        PayMode  `json:"pay_mode"`                   // 扣费主体
}

// FlowRecordInfo 流水记录信息
type FlowRecordInfo struct {
	FlowType             FlowType `json:"flow_type"`                        // 流水类型
	AppID                string   `json:"appid"`                            // appid
	WxStoreID            string   `json:"wx_store_id"`                      // 微信门店ID
	PayOrderID           uint64   `json:"pay_order_id,omitempty"`           // 充值订单号
	WxOrderID            string   `json:"wx_order_id,omitempty"`            // 订单ID（消费流水）
	ServiceTransID       string   `json:"service_trans_id"`                 // 运力ID
	OpenID               string   `json:"openid,omitempty"`                 // 用户openid（消费流水）
	DeliveryStatus       int32    `json:"delivery_status,omitempty"`        // 运单状态（消费流水）
	PayAmount            int32    `json:"pay_amount"`                       // 支付金额，单位：分
	PayTime              uint32   `json:"pay_time,omitempty"`               // 支付时间
	PayStatus            string   `json:"pay_status,omitempty"`             // 支付状态
	RefundStatus         string   `json:"refund_status,omitempty"`          // 退款状态
	RefundAmount         int32    `json:"refund_amount,omitempty"`          // 退款金额
	RefundTime           uint32   `json:"refund_time,omitempty"`            // 退款时间
	DeductAmount         int32    `json:"deduct_amount,omitempty"`          // 扣除违约金
	CreateTime           uint32   `json:"create_time"`                      // 创建时间
	ConsumeDeadline      uint32   `json:"consume_deadline,omitempty"`       // 有效截止日期
	BillID               string   `json:"bill_id,omitempty"`                // 运单ID
	DeliveryFinishedTime uint32   `json:"delivery_finished_time,omitempty"` // 运单完成配送的时间
}

// QueryFlowResponse 门店运费流水查询返回参数
type QueryFlowResponse struct {
	util.CommonError
	Total          uint32           `json:"total"`            // 总数
	FlowList       []FlowRecordInfo `json:"flow_list"`        // 流水数组
	TotalPayAmt    int              `json:"total_pay_amt"`    // 总支付金额
	TotalRefundAmt int              `json:"total_refund_amt"` // 总退款金额
	TotalDeductAmt int              `json:"total_deduct_amt"` // 总违约金（消费流水返回）
}

// IntracityQueryFlow 门店运费流水查询
func (express *Express) IntracityQueryFlow(ctx context.Context, req *QueryFlowRequest) (res QueryFlowResponse, err error) {
	accessToken, err := express.GetAccessToken()
	if err != nil {
		return
	}

	uri := fmt.Sprintf(intracityQueryFlowURL, accessToken)
	response, err := util.PostJSONContext(ctx, uri, req)
	if err != nil {
		return
	}

	err = util.DecodeWithError(response, &res, "IntracityQueryFlow")
	return
}

// BalanceQueryRequest 门店余额查询请求参数
type BalanceQueryRequest struct {
	WxStoreID      string  `json:"wx_store_id,omitempty"`      // 微信门店编号
	ServiceTransID string  `json:"service_trans_id,omitempty"` // 运力ID
	PayMode        PayMode `json:"pay_mode,omitempty"`         // 充值/扣费主体
}

// BalanceInfo 余额信息
type BalanceInfo struct {
	ServiceTransID string `json:"service_trans_id"` // 运力ID
	Balance        int32  `json:"balance"`          // 余额，单位：分
}

// BalanceQueryResponse 门店余额查询返回参数
type BalanceQueryResponse struct {
	util.CommonError
	AppID       string        `json:"appid"`        // 小程序appid
	WxStoreID   string        `json:"wx_store_id"`  // 微信门店编号
	BalanceList []BalanceInfo `json:"balance_list"` // 余额列表
}

// IntracityBalanceQuery 门店余额查询
func (express *Express) IntracityBalanceQuery(ctx context.Context, req *BalanceQueryRequest) (res BalanceQueryResponse, err error) {
	accessToken, err := express.GetAccessToken()
	if err != nil {
		return
	}

	uri := fmt.Sprintf(intracityBalanceQueryURL, accessToken)
	response, err := util.PostJSONContext(ctx, uri, req)
	if err != nil {
		return
	}

	err = util.DecodeWithError(response, &res, "IntracityBalanceQuery")
	return
}

// ============ 配送订单接口 ============

// CargoInfo 货物信息
type CargoInfo struct {
	Name   string `json:"name"`             // 货物名称
	Num    uint32 `json:"num,omitempty"`    // 货物数量
	Price  uint32 `json:"price,omitempty"`  // 货物价格，单位：分
	Weight uint32 `json:"weight,omitempty"` // 货物重量，单位：克
}

// PreAddOrderRequest 预下配送单请求参数
type PreAddOrderRequest struct {
	WxStoreID      string     `json:"wx_store_id,omitempty"`      // 微信门店编号，二选一
	OutStoreID     string     `json:"out_store_id,omitempty"`     // 自定义门店编号，二选一
	UserOpenID     string     `json:"user_openid"`                // 用户openid
	UserPhone      string     `json:"user_phone,omitempty"`       // 用户联系电话
	UserName       string     `json:"user_name,omitempty"`        // 用户姓名
	UserLat        float64    `json:"user_lat"`                   // 用户地址纬度
	UserLng        float64    `json:"user_lng"`                   // 用户地址经度
	UserAddress    string     `json:"user_address"`               // 用户详细地址
	ServiceTransID string     `json:"service_trans_id,omitempty"` // 运力ID，不传则查询所有运力
	PayMode        PayMode    `json:"pay_mode,omitempty"`         // 充值/扣费主体
	CargoInfo      *CargoInfo `json:"cargo_info,omitempty"`       // 货物信息
}

// TransInfo 运力信息
type TransInfo struct {
	ServiceTransID string `json:"service_trans_id"` // 运力ID
	ServiceName    string `json:"service_name"`     // 运力名称
	Price          uint32 `json:"price"`            // 配送费用，单位：分
	Distance       uint32 `json:"distance"`         // 配送距离，单位：米
	Errcode        int    `json:"errcode"`          // 错误码，0表示成功
	Errmsg         string `json:"errmsg"`           // 错误信息
}

// PreAddOrderResponse 预下配送单返回参数
type PreAddOrderResponse struct {
	util.CommonError
	WxStoreID string      `json:"wx_store_id"` // 微信门店编号
	AppID     string      `json:"appid"`       // 小程序appid
	TransList []TransInfo `json:"trans_list"`  // 运力列表
}

// IntracityPreAddOrder 预下配送单（查询运费）
func (express *Express) IntracityPreAddOrder(ctx context.Context, req *PreAddOrderRequest) (res PreAddOrderResponse, err error) {
	accessToken, err := express.GetAccessToken()
	if err != nil {
		return
	}

	uri := fmt.Sprintf(intracityPreAddOrderURL, accessToken)
	response, err := util.PostJSONContext(ctx, uri, req)
	if err != nil {
		return
	}

	err = util.DecodeWithError(response, &res, "IntracityPreAddOrder")
	return
}

// AddOrderRequest 创建配送单请求参数
type AddOrderRequest struct {
	WxStoreID       string     `json:"wx_store_id,omitempty"`       // 微信门店编号，二选一
	OutStoreID      string     `json:"out_store_id,omitempty"`      // 自定义门店编号，二选一
	OutOrderID      string     `json:"out_order_id"`                // 自定义订单号，需唯一
	ServiceTransID  string     `json:"service_trans_id,omitempty"`  // 运力ID
	UserOpenID      string     `json:"user_openid"`                 // 用户openid
	UserPhone       string     `json:"user_phone"`                  // 用户联系电话
	UserName        string     `json:"user_name"`                   // 用户姓名
	UserLat         float64    `json:"user_lat"`                    // 用户地址纬度
	UserLng         float64    `json:"user_lng"`                    // 用户地址经度
	UserAddress     string     `json:"user_address"`                // 用户详细地址
	PayMode         PayMode    `json:"pay_mode,omitempty"`          // 充值/扣费主体
	CargoInfo       *CargoInfo `json:"cargo_info,omitempty"`        // 货物信息
	OrderDetailPath string     `json:"order_detail_path,omitempty"` // 订单详情页路径
	CallbackURL     string     `json:"callback_url,omitempty"`      // 配送状态回调URL
	UseInsurance    uint32     `json:"use_insurance,omitempty"`     // 是否使用保价：0-不使用，1-使用
	InsuranceValue  uint32     `json:"insurance_value,omitempty"`   // 保价金额，单位：分
	ExpectTime      uint32     `json:"expect_time,omitempty"`       // 期望送达时间戳
	Remark          string     `json:"remark,omitempty"`            // 备注
}

// AddOrderResponse 创建配送单返回参数
type AddOrderResponse struct {
	util.CommonError
	WxOrderID      string `json:"wx_order_id"`      // 微信订单号
	AppID          string `json:"appid"`            // 小程序appid
	WxStoreID      string `json:"wx_store_id"`      // 微信门店编号
	OutOrderID     string `json:"out_order_id"`     // 自定义订单号
	ServiceTransID string `json:"service_trans_id"` // 运力ID
	BillID         string `json:"bill_id"`          // 运力订单号
	Price          uint32 `json:"price"`            // 配送费用，单位：分
	Distance       uint32 `json:"distance"`         // 配送距离，单位：米
}

// IntracityAddOrder 创建配送单
func (express *Express) IntracityAddOrder(ctx context.Context, req *AddOrderRequest) (res AddOrderResponse, err error) {
	accessToken, err := express.GetAccessToken()
	if err != nil {
		return
	}

	uri := fmt.Sprintf(intracityAddOrderURL, accessToken)
	response, err := util.PostJSONContext(ctx, uri, req)
	if err != nil {
		return
	}

	err = util.DecodeWithError(response, &res, "IntracityAddOrder")
	return
}

// QueryOrderRequest 查询配送单请求参数
type QueryOrderRequest struct {
	WxOrderID  string `json:"wx_order_id,omitempty"`  // 微信订单号，二选一
	OutOrderID string `json:"out_order_id,omitempty"` // 自定义订单号，二选一
	WxStoreID  string `json:"wx_store_id,omitempty"`  // 微信门店编号
	OutStoreID string `json:"out_store_id,omitempty"` // 自定义门店编号
}

// RiderInfo 骑手信息
type RiderInfo struct {
	Name        string `json:"name"`          // 骑手姓名
	Phone       string `json:"phone"`         // 骑手电话
	RiderCode   string `json:"rider_code"`    // 骑手编号
	RiderImgURL string `json:"rider_img_url"` // 骑手头像URL
}

// QueryOrderResponse 查询配送单返回参数
type QueryOrderResponse struct {
	util.CommonError
	WxOrderID      string                  `json:"wx_order_id"`      // 微信订单号
	AppID          string                  `json:"appid"`            // 小程序appid
	WxStoreID      string                  `json:"wx_store_id"`      // 微信门店编号
	OutOrderID     string                  `json:"out_order_id"`     // 自定义订单号
	ServiceTransID string                  `json:"service_trans_id"` // 运力ID
	BillID         string                  `json:"bill_id"`          // 运力订单号
	DeliveryStatus IntracityDeliveryStatus `json:"delivery_status"`  // 配送状态
	Price          uint32                  `json:"price"`            // 配送费用，单位：分
	Distance       uint32                  `json:"distance"`         // 配送距离，单位：米
	CreateTime     uint32                  `json:"create_time"`      // 订单创建时间
	RiderInfo      *RiderInfo              `json:"rider_info"`       // 骑手信息
	FinishTime     uint32                  `json:"finish_time"`      // 订单完成时间
}

// IntracityQueryOrder 查询配送单
func (express *Express) IntracityQueryOrder(ctx context.Context, req *QueryOrderRequest) (res QueryOrderResponse, err error) {
	accessToken, err := express.GetAccessToken()
	if err != nil {
		return
	}

	uri := fmt.Sprintf(intracityQueryOrderURL, accessToken)
	response, err := util.PostJSONContext(ctx, uri, req)
	if err != nil {
		return
	}

	err = util.DecodeWithError(response, &res, "IntracityQueryOrder")
	return
}

// CancelOrderRequest 取消配送单请求参数
type CancelOrderRequest struct {
	WxOrderID    string `json:"wx_order_id,omitempty"`   // 微信订单号，二选一
	OutOrderID   string `json:"out_order_id,omitempty"`  // 自定义订单号，二选一
	WxStoreID    string `json:"wx_store_id,omitempty"`   // 微信门店编号
	OutStoreID   string `json:"out_store_id,omitempty"`  // 自定义门店编号
	CancelReason string `json:"cancel_reason,omitempty"` // 取消原因
}

// CancelOrderResponse 取消配送单返回参数
type CancelOrderResponse struct {
	util.CommonError
	WxOrderID    string `json:"wx_order_id"`   // 微信订单号
	RefundAmount int32  `json:"refund_amount"` // 退款金额，单位：分
	DeductAmount int32  `json:"deduct_amount"` // 扣除违约金，单位：分
}

// IntracityCancelOrder 取消配送单
func (express *Express) IntracityCancelOrder(ctx context.Context, req *CancelOrderRequest) (res CancelOrderResponse, err error) {
	accessToken, err := express.GetAccessToken()
	if err != nil {
		return
	}

	uri := fmt.Sprintf(intracityCancelOrderURL, accessToken)
	response, err := util.PostJSONContext(ctx, uri, req)
	if err != nil {
		return
	}

	err = util.DecodeWithError(response, &res, "IntracityCancelOrder")
	return
}

// MockNotifyRequest 模拟配送回调请求参数（仅用于测试）
type MockNotifyRequest struct {
	WxOrderID      string                  `json:"wx_order_id"`     // 微信订单号
	DeliveryStatus IntracityDeliveryStatus `json:"delivery_status"` // 配送状态
}

// IntracityMockNotify 模拟配送回调（仅用于测试）
func (express *Express) IntracityMockNotify(ctx context.Context, req *MockNotifyRequest) error {
	accessToken, err := express.GetAccessToken()
	if err != nil {
		return err
	}

	uri := fmt.Sprintf(intracityMockNotifyURL, accessToken)
	response, err := util.PostJSONContext(ctx, uri, req)
	if err != nil {
		return err
	}

	return util.DecodeWithCommonError(response, "IntracityMockNotify")
}
