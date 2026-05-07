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

const (
	// EnvProduction 环境 0-正式环境 1-沙箱环境
	EnvProduction Env = 0
	// EnvSandbox 环境 0-正式环境 1-沙箱环境
	EnvSandbox Env = 1
)

const (
	// Success 错误码 0、成功
	Success ErrCode = 0
	// SystemError 错误码 -1、系统错误
	SystemError ErrCode = -1
	// OpenIDError 错误码 268490001、openid 错误
	OpenIDError ErrCode = 268490001
	// RequestParamError 错误码 268490002、请求参数字段错误，具体看 errmsg
	RequestParamError ErrCode = 268490002
	// SignError 错误码 268490003、签名错误
	SignError ErrCode = 268490003
	// RepeatOperationError 错误码 268490004、重复操作（赠送和代币支付相关接口会返回，表示之前的操作已经成功）
	RepeatOperationError ErrCode = 268490004
	// OrderRefundedError 错误码 268490005、订单已经通过 cancel_currency_pay 接口退款，不支持再退款
	OrderRefundedError ErrCode = 268490005
	// InsufficientBalanceError 错误码 268490006、代币的退款/支付操作金额不足
	InsufficientBalanceError ErrCode = 268490006
	// SensitiveContentError 错误码 268490007、图片或文字存在敏感内容，禁止使用
	SensitiveContentError ErrCode = 268490007
	// TokenNotPublishedError 错误码 268490008、代币未发布，不允许进行代币操作
	TokenNotPublishedError ErrCode = 268490008
	// SessionKeyExpiredError 错误码 268490009、用户 session_key 不存在或已过期，请重新登录
	SessionKeyExpiredError ErrCode = 268490009
	// BillGeneratingError 错误码 268490011、账单数据生成中，请稍后调用本接口获取
	BillGeneratingError ErrCode = 268490011
)

const (
	// OrderStatusInit 订单状态 当前状态 0-订单初始化（未创建成功，不可用于支付）
	OrderStatusInit OrderStatus = 0
	// OrderStatusCreated 订单状态 当前状态 1-订单创建成功
	OrderStatusCreated OrderStatus = 1
	// OrderStatusPaid 订单状态 当前状态  2-订单已经支付，待发货
	OrderStatusPaid OrderStatus = 2
	// OrderStatusDelivering 订单状态 当前状态 3-订单发货中
	OrderStatusDelivering OrderStatus = 3
	// OrderStatusDelivered 订单状态 当前状态 4-订单已发货
	OrderStatusDelivered OrderStatus = 4
	// OrderStatusRefunded 订单状态 当前状态 5-订单已经退款
	OrderStatusRefunded OrderStatus = 5
	// OrderStatusClosed 订单状态 当前状态  6-订单已经关闭（不可再使用）
	OrderStatusClosed OrderStatus = 6
	// OrderStatusRefundFailed 订单状态 当前状态 7-订单退款失败
	OrderStatusRefundFailed OrderStatus = 7
)

const (
	// baseSite 基础网址
	baseSite = "https://api.weixin.qq.com"

	// queryUserBalance 查询虚拟支付余额
	queryUserBalance = "/xpay/query_user_balance"

	// currencyPay 扣减代币（一般用于代币支付）
	currencyPay = "/xpay/currency_pay"

	// queryOrder 查询创建的订单（现金单，非代币单）
	queryOrder = "/xpay/query_order"

	// cancelCurrencyPay 代币支付退款 (currency_pay 接口的逆操作)
	cancelCurrencyPay = "/xpay/cancel_currency_pay"

	// notifyProvideGoods 通知已经发货完成（只能通知现金单）,正常通过 xpay_goods_deliver_notify 消息推送返回成功就不需要调用这个 api 接口。这个接口用于异常情况推送不成功时手动将单改成已发货状态
	notifyProvideGoods = "/xpay/notify_provide_goods"

	// presentCurrency 代币赠送接口，由于目前不支付按单号查赠送单的功能，所以当需要赠送的时候可以一直重试到返回 0 或者返回 268490004（重复操作）为止
	presentCurrency = "/xpay/present_currency"

	// downloadBill 下载账单
	downloadBill = "/xpay/download_bill"

	// refundOrder 退款 对使用 jsapi 接口下的单进行退款
	refundOrder = "/xpay/refund_order"

	// createWithdrawOrder 创建提现单
	createWithdrawOrder = "/xpay/create_withdraw_order"

	// queryWithdrawOrder 查询提现单
	queryWithdrawOrder = "/xpay/query_withdraw_order"

	// startUploadGoods 启动批量上传道具任务
	startUploadGoods = "/xpay/start_upload_goods"

	// queryUploadGoods 查询批量上传道具任务状态
	queryUploadGoods = "/xpay/query_upload_goods"

	// startPublishGoods 启动批量发布道具任务
	startPublishGoods = "/xpay/start_publish_goods"

	// queryPublishGoods 查询批量发布道具任务状态
	queryPublishGoods = "/xpay/query_publish_goods"
)

const (
	// signature user mode signature
	signature = "signature"

	// paySignature payment signature
	paySignature = "pay_sig"

	// accessToken access_token authorization tokens
	accessToken = "access_token"

	// EmptyString empty string
	EmptyString = ""
)
