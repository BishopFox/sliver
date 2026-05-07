package express

import (
	"context"
	"fmt"

	"github.com/silenceper/wechat/v2/util"
)

const (
	// 传运单接口，商户使用此接口向微信提供某交易单号对应的运单号。微信后台会跟踪运单的状态变化
	openMsgTraceWaybillURL = "https://api.weixin.qq.com/cgi-bin/express/delivery/open_msg/trace_waybill?access_token=%s"

	// 查询运单接口，商户在调用完trace_waybill接口后，可以使用本接口查询到对应运单的详情信息
	openMsgQueryTraceURL = "https://api.weixin.qq.com/cgi-bin/express/delivery/open_msg/query_trace?access_token=%s"

	// 更新物流信息，更新物品信息
	openMsgUpdateWaybillGoodsURL = "https://api.weixin.qq.com/cgi-bin/express/delivery/open_msg/update_waybill_goods?access_token=%s"

	// 传运单接口，商户使用此接口向微信提供某交易单号对应的运单号。微信后台会跟踪运单的状态变化，在关键物流节点给下单用户推送消息通知
	openMsgFollowWaybillURL = "https://api.weixin.qq.com/cgi-bin/express/delivery/open_msg/follow_waybill?access_token=%s"

	// 查运单接口，商户在调用完follow_waybill接口后，可以使用本接口查询到对应运单的详情信息
	openMsgQueryFollowTraceURL = "https://api.weixin.qq.com/cgi-bin/express/delivery/open_msg/query_follow_trace?access_token=%s"

	// 更新物品信息接口
	openMsgUpdateFollowWaybillGoodsURL = "https://api.weixin.qq.com/cgi-bin/express/delivery/open_msg/update_follow_waybill_goods?access_token=%s"

	// 获取运力id列表
	openMsgGetDeliveryListURL = "https://api.weixin.qq.com/cgi-bin/express/delivery/open_msg/get_delivery_list?access_token=%s"
)

// TraceWaybill 传运单
// https://developers.weixin.qq.com/miniprogram/dev/platform-capabilities/industry/express/business/express_search.html#_2-%E6%8E%A5%E5%8F%A3%E5%88%97%E8%A1%A8
func (express *Express) TraceWaybill(ctx context.Context, in *TraceWaybillRequest) (res TraceWaybillResponse, err error) {
	accessToken, err := express.GetAccessToken()
	if err != nil {
		return
	}

	uri := fmt.Sprintf(openMsgTraceWaybillURL, accessToken)
	response, err := util.PostJSONContext(ctx, uri, in)
	if err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &res, "TraceWaybill")
	return
}

// QueryTrace 查询运单详情信息
// https://developers.weixin.qq.com/miniprogram/dev/platform-capabilities/industry/express/business/express_search.html#_2-%E6%8E%A5%E5%8F%A3%E5%88%97%E8%A1%A8
func (express *Express) QueryTrace(ctx context.Context, in *QueryTraceRequest) (res QueryTraceResponse, err error) {
	accessToken, err := express.GetAccessToken()
	if err != nil {
		return
	}

	uri := fmt.Sprintf(openMsgQueryTraceURL, accessToken)
	response, err := util.PostJSONContext(ctx, uri, in)
	if err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &res, "QueryTrace")
	return
}

// UpdateWaybillGoods 更新物品信息
// https://developers.weixin.qq.com/miniprogram/dev/platform-capabilities/industry/express/business/express_search.html#_2-%E6%8E%A5%E5%8F%A3%E5%88%97%E8%A1%A8
func (express *Express) UpdateWaybillGoods(ctx context.Context, in *UpdateWaybillGoodsRequest) (err error) {
	accessToken, err := express.GetAccessToken()
	if err != nil {
		return
	}

	uri := fmt.Sprintf(openMsgUpdateWaybillGoodsURL, accessToken)
	response, err := util.PostJSONContext(ctx, uri, in)
	if err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithCommonError(response, "UpdateWaybillGoods")
	return
}

// FollowWaybill 传运单
// https://developers.weixin.qq.com/miniprogram/dev/platform-capabilities/industry/express/business/express_open_msg.html#_4-1%E3%80%81%E4%BC%A0%E8%BF%90%E5%8D%95%E6%8E%A5%E5%8F%A3-follow-waybill
func (express *Express) FollowWaybill(ctx context.Context, in *FollowWaybillRequest) (res FollowWaybillResponse, err error) {
	accessToken, err := express.GetAccessToken()
	if err != nil {
		return
	}

	uri := fmt.Sprintf(openMsgFollowWaybillURL, accessToken)
	response, err := util.PostJSONContext(ctx, uri, in)
	if err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &res, "FollowWaybill")
	return
}

// QueryFollowTrace 查询运单详情信息
// https://developers.weixin.qq.com/miniprogram/dev/platform-capabilities/industry/express/business/express_open_msg.html#_4-2%E3%80%81%E6%9F%A5%E8%BF%90%E5%8D%95%E6%8E%A5%E5%8F%A3-query-follow-trace
func (express *Express) QueryFollowTrace(ctx context.Context, in *QueryFollowTraceRequest) (res QueryFollowTraceResponse, err error) {
	accessToken, err := express.GetAccessToken()
	if err != nil {
		return
	}

	uri := fmt.Sprintf(openMsgQueryFollowTraceURL, accessToken)
	response, err := util.PostJSONContext(ctx, uri, in)
	if err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &res, "QueryFollowTrace")
	return
}

// UpdateFollowWaybillGoods 更新物品信息
// https://developers.weixin.qq.com/miniprogram/dev/platform-capabilities/industry/express/business/express_open_msg.html#_4-3%E3%80%81%E6%9B%B4%E6%96%B0%E7%89%A9%E5%93%81%E4%BF%A1%E6%81%AF%E6%8E%A5%E5%8F%A3-update-follow-waybill-goods
func (express *Express) UpdateFollowWaybillGoods(ctx context.Context, in *UpdateFollowWaybillGoodsRequest) (err error) {
	accessToken, err := express.GetAccessToken()
	if err != nil {
		return
	}

	uri := fmt.Sprintf(openMsgUpdateFollowWaybillGoodsURL, accessToken)
	response, err := util.PostJSONContext(ctx, uri, in)
	if err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithCommonError(response, "UpdateFollowWaybillGoods")
	return
}

// GetDeliveryList 获取运力id列表
// https://developers.weixin.qq.com/miniprogram/dev/platform-capabilities/industry/express/business/express_open_msg.html#_4-4%E8%8E%B7%E5%8F%96%E8%BF%90%E5%8A%9Bid%E5%88%97%E8%A1%A8get-delivery-list
func (express *Express) GetDeliveryList(ctx context.Context) (res GetDeliveryListResponse, err error) {
	accessToken, err := express.GetAccessToken()
	if err != nil {
		return
	}

	uri := fmt.Sprintf(openMsgGetDeliveryListURL, accessToken)
	response, err := util.PostJSONContext(ctx, uri, map[string]interface{}{})
	if err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &res, "GetDeliveryList")
	return
}

// TraceWaybillRequest 传运单接口请求参数
type TraceWaybillRequest struct {
	GoodsInfo       FollowWaybillGoodsInfo `json:"goods_info"`        // 必选，商品信息
	Openid          string                 `json:"openid"`            // 必选，用户openid
	SenderPhone     string                 `json:"sender_phone"`      // 寄件人手机号
	ReceiverPhone   string                 `json:"receiver_phone"`    // 必选，收件人手机号，部分运力需要用户手机号作为查单依据
	DeliveryID      string                 `json:"delivery_id"`       // 运力id（运单号所属运力公司id）
	WaybillID       string                 `json:"waybill_id"`        // 必选，运单号
	TransID         string                 `json:"trans_id"`          // 必选，交易单号（微信支付生成的交易单号，一般以420开头）
	OrderDetailPath string                 `json:"order_detail_path"` // 订单详情页地址

}

// TraceWaybillResponse 传运单接口返回参数
type TraceWaybillResponse struct {
	util.CommonError
	WaybillToken string `json:"waybill_token"` // 查询id
}

// QueryTraceRequest 查询运单详情接口请求参数
type QueryTraceRequest struct {
	WaybillToken string `json:"waybill_token"` // 必选，查询id
}

// QueryTraceResponse 查询运单详情接口返回参数
type QueryTraceResponse struct {
	util.CommonError
	WaybillInfo  FlowWaybillInfo         `json:"waybill_info"`  // 运单信息
	ShopInfo     FollowWaybillShopInfo   `json:"shop_info"`     // 商品信息
	DeliveryInfo FlowWaybillDeliveryInfo `json:"delivery_info"` // 运力信息
}

// UpdateWaybillGoodsRequest 更新物品信息接口请求参数
type UpdateWaybillGoodsRequest struct {
	WaybillToken string                 `json:"waybill_token"` // 必选，查询id
	GoodsInfo    FollowWaybillGoodsInfo `json:"goods_info"`    // 必选，商品信息
}

// FollowWaybillRequest 传运单接口请求参数
type FollowWaybillRequest struct {
	GoodsInfo       FollowWaybillGoodsInfo `json:"goods_info"`        // 必选，商品信息
	Openid          string                 `json:"openid"`            // 必选，用户openid
	SenderPhone     string                 `json:"sender_phone"`      // 寄件人手机号
	ReceiverPhone   string                 `json:"receiver_phone"`    // 必选，收件人手机号，部分运力需要用户手机号作为查单依据
	DeliveryID      string                 `json:"delivery_id"`       // 运力id（运单号所属运力公司id）
	WaybillID       string                 `json:"waybill_id"`        // 必选，运单号
	TransID         string                 `json:"trans_id"`          // 必选，交易单号（微信支付生成的交易单号，一般以420开头）
	OrderDetailPath string                 `json:"order_detail_path"` // 订单详情页地址
}

// FollowWaybillGoodsInfo 商品信息
type FollowWaybillGoodsInfo struct {
	DetailList []FollowWaybillGoodsInfoItem `json:"detail_list"`
}

// FollowWaybillShopInfo 商品信息
type FollowWaybillShopInfo struct {
	GoodsInfo FollowWaybillGoodsInfo `json:"goods_info"` // 商品信息
}

// FollowWaybillGoodsInfoItem 商品信息详情
type FollowWaybillGoodsInfoItem struct {
	GoodsName   string `json:"goods_name"`           // 必选，商品名称(最大长度为utf-8编码下的60个字符）
	GoodsImgURL string `json:"goods_img_url"`        // 必选，商品图片url
	GoodsDesc   string `json:"goods_desc,omitempty"` // 商品详情描述，不传默认取“商品名称”值，最多40汉字
}

// FollowWaybillResponse 传运单接口返回参数
type FollowWaybillResponse struct {
	util.CommonError
	WaybillToken string `json:"waybill_token"` // 查询id
}

// QueryFollowTraceRequest 查询运单详情信息请求参数
type QueryFollowTraceRequest struct {
	WaybillToken string `json:"waybill_token"` // 必选，查询id
}

// QueryFollowTraceResponse 查询运单详情信息返回参数
type QueryFollowTraceResponse struct {
	util.CommonError
	WaybillInfo  FlowWaybillInfo         `json:"waybill_info"`  // 运单信息
	ShopInfo     FollowWaybillShopInfo   `json:"shop_info"`     // 商品信息
	DeliveryInfo FlowWaybillDeliveryInfo `json:"delivery_info"` // 运力信息
}

// FlowWaybillInfo 运单信息
type FlowWaybillInfo struct {
	WaybillID string        `json:"waybill_id"` // 运单号
	Status    WaybillStatus `json:"status"`     // 运单状态
}

// UpdateFollowWaybillGoodsRequest 修改运单商品信息请求参数
type UpdateFollowWaybillGoodsRequest struct {
	WaybillToken string                 `json:"waybill_token"` // 必选，查询id
	GoodsInfo    FollowWaybillGoodsInfo `json:"goods_info"`    // 必选，商品信息
}

// GetDeliveryListResponse 获取运力id列表返回参数
type GetDeliveryListResponse struct {
	util.CommonError
	DeliveryList []FlowWaybillDeliveryInfo `json:"delivery_list"` // 运力公司列表
	Count        int                       `json:"count"`         // 运力公司个数
}

// FlowWaybillDeliveryInfo 运力公司信息
type FlowWaybillDeliveryInfo struct {
	DeliveryID   string `json:"delivery_id"`   // 运力公司id
	DeliveryName string `json:"delivery_name"` // 运力公司名称
}

// WaybillStatus 运单状态
type WaybillStatus int

const (
	// WaybillStatusNotExist 运单不存在或者未揽收
	WaybillStatusNotExist WaybillStatus = iota
	// WaybillStatusPicked 已揽件
	WaybillStatusPicked
	// WaybillStatusTransporting 运输中
	WaybillStatusTransporting
	// WaybillStatusDispatching 派件中
	WaybillStatusDispatching
	// WaybillStatusSigned 已签收
	WaybillStatusSigned
	// WaybillStatusException 异常
	WaybillStatusException
	// WaybillStatusSignedByOthers 代签收
	WaybillStatusSignedByOthers
)
