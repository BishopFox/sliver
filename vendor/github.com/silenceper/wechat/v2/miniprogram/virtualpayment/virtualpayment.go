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
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	"github.com/silenceper/wechat/v2/util"
)

// SetSessionKey 设置 sessionKey
func (s *VirtualPayment) SetSessionKey(sessionKey string) {
	s.sessionKey = sessionKey
}

// QueryUserBalance 查询虚拟支付余额
func (s *VirtualPayment) QueryUserBalance(ctx context.Context, in *QueryUserBalanceRequest) (out QueryUserBalanceResponse, err error) {
	var jsonByte []byte
	if jsonByte, err = json.Marshal(in); err != nil {
		return
	}

	var (
		params = URLParams{
			Path:    queryUserBalance,
			Content: string(jsonByte),
		}
		address string
	)
	if address, err = s.requestAddress(params); err != nil {
		return
	}

	var response []byte
	if response, err = postJSONBytesContext(ctx, address, jsonByte); err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &out, "QueryUserBalance")
	return
}

// CurrencyPay currency pay 扣减代币（一般用于代币支付）
func (s *VirtualPayment) CurrencyPay(ctx context.Context, in *CurrencyPayRequest) (out CurrencyPayResponse, err error) {
	var jsonByte []byte
	if jsonByte, err = json.Marshal(in); err != nil {
		return
	}

	var (
		params = URLParams{
			Path:    currencyPay,
			Content: string(jsonByte),
		}
		address string
	)
	if address, err = s.requestAddress(params); err != nil {
		return
	}

	var response []byte
	if response, err = postJSONBytesContext(ctx, address, jsonByte); err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &out, "CurrencyPay")
	return
}

// QueryOrder 查询创建的订单（现金单，非代币单）
func (s *VirtualPayment) QueryOrder(ctx context.Context, in *QueryOrderRequest) (out QueryOrderResponse, err error) {
	var jsonByte []byte
	if jsonByte, err = json.Marshal(in); err != nil {
		return
	}

	var (
		params = URLParams{
			Path:      queryOrder,
			Signature: EmptyString,
			Content:   string(jsonByte),
		}
		address string
	)
	if address, err = s.requestAddress(params); err != nil {
		return
	}
	var response []byte
	if response, err = postJSONBytesContext(ctx, address, jsonByte); err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &out, "QueryOrder")
	return
}

// CancelCurrencyPay 取消订单 代币支付退款 (currency_pay 接口的逆操作)
func (s *VirtualPayment) CancelCurrencyPay(ctx context.Context, in *CancelCurrencyPayRequest) (out CancelCurrencyPayResponse, err error) {
	var jsonByte []byte
	if jsonByte, err = json.Marshal(in); err != nil {
		return
	}

	var (
		params = URLParams{
			Path:    cancelCurrencyPay,
			Content: string(jsonByte),
		}
		address string
	)
	if address, err = s.requestAddress(params); err != nil {
		return
	}

	var response []byte
	if response, err = postJSONBytesContext(ctx, address, jsonByte); err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &out, "CancelCurrencyPay")
	return
}

// NotifyProvideGoods 通知发货
// 通知已经发货完成（只能通知现金单）,正常通过 xpay_goods_deliver_notify 消息推送返回成功就不需要调用这个 api 接口。这个接口用于异常情况推送不成功时手动将单改成已发货状态
func (s *VirtualPayment) NotifyProvideGoods(ctx context.Context, in *NotifyProvideGoodsRequest) (out NotifyProvideGoodsResponse, err error) {
	var jsonByte []byte
	if jsonByte, err = json.Marshal(in); err != nil {
		return
	}

	var (
		params = URLParams{
			Path:      notifyProvideGoods,
			Content:   string(jsonByte),
			Signature: EmptyString,
		}
		address string
	)
	if address, err = s.requestAddress(params); err != nil {
		return
	}

	var response []byte
	if response, err = postJSONBytesContext(ctx, address, jsonByte); err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &out, "NotifyProvideGoods")
	return
}

// PresentCurrency 代币赠送接口，由于目前不支付按单号查赠送单的功能，所以当需要赠送的时候可以一直重试到返回 0 或者返回 268490004（重复操作）为止
func (s *VirtualPayment) PresentCurrency(ctx context.Context, in *PresentCurrencyRequest) (out PresentCurrencyResponse, err error) {
	var jsonByte []byte
	if jsonByte, err = json.Marshal(in); err != nil {
		return
	}

	var (
		params = URLParams{
			Path:      presentCurrency,
			Content:   string(jsonByte),
			Signature: EmptyString,
		}
		address string
	)
	if address, err = s.requestAddress(params); err != nil {
		return
	}

	var response []byte
	if response, err = postJSONBytesContext(ctx, address, jsonByte); err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &out, "PresentCurrency")
	return
}

// DownloadBill 下载订单交易账单
func (s *VirtualPayment) DownloadBill(ctx context.Context, in *DownloadBillRequest) (out DownloadBillResponse, err error) {
	var jsonByte []byte
	if jsonByte, err = json.Marshal(in); err != nil {
		return
	}

	var (
		params = URLParams{
			Path:      downloadBill,
			Content:   string(jsonByte),
			Signature: EmptyString,
		}
		address string
	)
	if address, err = s.requestAddress(params); err != nil {
		return
	}

	var response []byte
	if response, err = postJSONBytesContext(ctx, address, jsonByte); err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &out, "DownloadBill")
	return
}

// RefundOrder 退款 对使用 jsapi 接口下的单进行退款
func (s *VirtualPayment) RefundOrder(ctx context.Context, in *RefundOrderRequest) (out RefundOrderResponse, err error) {
	var jsonByte []byte
	if jsonByte, err = json.Marshal(in); err != nil {
		return
	}

	var (
		params = URLParams{
			Path:      refundOrder,
			Content:   string(jsonByte),
			Signature: EmptyString,
		}
		address string
	)
	if address, err = s.requestAddress(params); err != nil {
		return
	}

	var response []byte
	if response, err = postJSONBytesContext(ctx, address, jsonByte); err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &out, "RefundOrder")
	return
}

// CreateWithdrawOrder 创建提现单
func (s *VirtualPayment) CreateWithdrawOrder(ctx context.Context, in *CreateWithdrawOrderRequest) (out CreateWithdrawOrderResponse, err error) {
	var jsonByte []byte
	if jsonByte, err = json.Marshal(in); err != nil {
		return
	}

	var (
		params = URLParams{
			Path:      createWithdrawOrder,
			Content:   string(jsonByte),
			Signature: EmptyString,
		}
		address string
	)
	if address, err = s.requestAddress(params); err != nil {
		return
	}

	var response []byte
	if response, err = postJSONBytesContext(ctx, address, jsonByte); err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &out, "CreateWithdrawOrder")
	return
}

// QueryWithdrawOrder 查询提现单
func (s *VirtualPayment) QueryWithdrawOrder(ctx context.Context, in *QueryWithdrawOrderRequest) (out QueryWithdrawOrderResponse, err error) {
	var jsonByte []byte
	if jsonByte, err = json.Marshal(in); err != nil {
		return
	}

	var (
		params = URLParams{
			Path:      queryWithdrawOrder,
			Content:   string(jsonByte),
			Signature: EmptyString,
		}
		address string
	)
	if address, err = s.requestAddress(params); err != nil {
		return
	}

	var response []byte
	if response, err = postJSONBytesContext(ctx, address, jsonByte); err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &out, "QueryWithdrawOrder")
	return
}

// StartUploadGoods 开始上传商品
func (s *VirtualPayment) StartUploadGoods(ctx context.Context, in *StartUploadGoodsRequest) (out StartUploadGoodsResponse, err error) {
	var jsonByte []byte
	if jsonByte, err = json.Marshal(in); err != nil {
		return
	}

	var (
		params = URLParams{
			Path:      startUploadGoods,
			Content:   string(jsonByte),
			Signature: EmptyString,
		}
		address string
	)
	if address, err = s.requestAddress(params); err != nil {
		return
	}

	var response []byte
	if response, err = postJSONBytesContext(ctx, address, jsonByte); err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &out, "StartUploadGoods")
	return
}

// QueryUploadGoods 查询上传商品
func (s *VirtualPayment) QueryUploadGoods(ctx context.Context, in *QueryUploadGoodsRequest) (out QueryUploadGoodsResponse, err error) {
	var jsonByte []byte
	if jsonByte, err = json.Marshal(in); err != nil {
		return
	}

	var (
		params = URLParams{
			Path:      queryUploadGoods,
			Content:   string(jsonByte),
			Signature: EmptyString,
		}
		address string
	)
	if address, err = s.requestAddress(params); err != nil {
		return
	}

	var response []byte
	if response, err = postJSONBytesContext(ctx, address, jsonByte); err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &out, "QueryUploadGoods")
	return
}

// StartPublishGoods 开始发布商品
func (s *VirtualPayment) StartPublishGoods(ctx context.Context, in *StartPublishGoodsRequest) (out StartPublishGoodsResponse, err error) {
	var jsonByte []byte
	if jsonByte, err = json.Marshal(in); err != nil {
		return
	}

	var (
		params = URLParams{
			Path:      startPublishGoods,
			Content:   string(jsonByte),
			Signature: EmptyString,
		}
		address string
	)
	if address, err = s.requestAddress(params); err != nil {
		return
	}

	var response []byte
	if response, err = postJSONBytesContext(ctx, address, jsonByte); err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &out, "StartPublishGoods")
	return
}

// QueryPublishGoods 查询发布商品
func (s *VirtualPayment) QueryPublishGoods(ctx context.Context, in *QueryPublishGoodsRequest) (out QueryPublishGoodsResponse, err error) {
	var jsonByte []byte
	if jsonByte, err = json.Marshal(in); err != nil {
		return
	}

	var (
		params = URLParams{
			Path:      queryPublishGoods,
			Content:   string(jsonByte),
			Signature: EmptyString,
		}
		address string
	)
	if address, err = s.requestAddress(params); err != nil {
		return
	}

	var response []byte
	if response, err = postJSONBytesContext(ctx, address, jsonByte); err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &out, "QueryPublishGoods")
	return
}

// StartDownloadOrder 发起下载小程序订单明细任务
func (s *VirtualPayment) StartDownloadOrder(ctx context.Context, in *StartDownloadOrderRequest) (out StartDownloadOrderResponse, err error) {
	var jsonByte []byte
	if jsonByte, err = json.Marshal(in); err != nil {
		return
	}

	var (
		params = URLParams{
			Path:      startDownloadOrder,
			Content:   string(jsonByte),
			Signature: EmptyString,
		}
		address string
	)
	if address, err = s.requestAddress(params); err != nil {
		return
	}

	var response []byte
	if response, err = postJSONBytesContext(ctx, address, jsonByte); err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &out, "StartDownloadOrder")
	return
}

// QueryDownloadOrder 查询下载订单任务结果
func (s *VirtualPayment) QueryDownloadOrder(ctx context.Context, in *QueryDownloadOrderRequest) (out QueryDownloadOrderResponse, err error) {
	var jsonByte []byte
	if jsonByte, err = json.Marshal(in); err != nil {
		return
	}

	var (
		params = URLParams{
			Path:      queryDownloadOrder,
			Content:   string(jsonByte),
			Signature: EmptyString,
		}
		address string
	)
	if address, err = s.requestAddress(params); err != nil {
		return
	}

	var response []byte
	if response, err = postJSONBytesContext(ctx, address, jsonByte); err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &out, "QueryDownloadOrder")
	return
}

// QueryBizBalance 查询商家账户可提现余额
func (s *VirtualPayment) QueryBizBalance(ctx context.Context, in *QueryBizBalanceRequest) (out QueryBizBalanceResponse, err error) {
	var jsonByte []byte
	if jsonByte, err = json.Marshal(in); err != nil {
		return
	}

	var (
		params = URLParams{
			Path:      queryBizBalance,
			Content:   string(jsonByte),
			Signature: EmptyString,
		}
		address string
	)
	if address, err = s.requestAddress(params); err != nil {
		return
	}

	var response []byte
	if response, err = postJSONBytesContext(ctx, address, jsonByte); err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &out, "QueryBizBalance")
	return
}

// QueryTransferAccount 查询广告金充值账户
func (s *VirtualPayment) QueryTransferAccount(ctx context.Context, in *QueryTransferAccountRequest) (out QueryTransferAccountResponse, err error) {
	var jsonByte []byte
	if jsonByte, err = json.Marshal(in); err != nil {
		return
	}

	var (
		params = URLParams{
			Path:      queryTransferAccount,
			Content:   string(jsonByte),
			Signature: EmptyString,
		}
		address string
	)
	if address, err = s.requestAddress(params); err != nil {
		return
	}

	var response []byte
	if response, err = postJSONBytesContext(ctx, address, jsonByte); err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &out, "QueryTransferAccount")
	return
}

// QueryAdverFunds 查询广告金发放记录
func (s *VirtualPayment) QueryAdverFunds(ctx context.Context, in *QueryAdverFundsRequest) (out QueryAdverFundsResponse, err error) {
	var jsonByte []byte
	if jsonByte, err = json.Marshal(in); err != nil {
		return
	}

	var (
		params = URLParams{
			Path:      queryAdverFunds,
			Content:   string(jsonByte),
			Signature: EmptyString,
		}
		address string
	)
	if address, err = s.requestAddress(params); err != nil {
		return
	}

	var response []byte
	if response, err = postJSONBytesContext(ctx, address, jsonByte); err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &out, "QueryAdverFunds")
	return
}

// CreateFundsBill 充值广告金
func (s *VirtualPayment) CreateFundsBill(ctx context.Context, in *CreateFundsBillRequest) (out CreateFundsBillResponse, err error) {
	var jsonByte []byte
	if jsonByte, err = json.Marshal(in); err != nil {
		return
	}

	var (
		params = URLParams{
			Path:      createFundsBill,
			Content:   string(jsonByte),
			Signature: EmptyString,
		}
		address string
	)
	if address, err = s.requestAddress(params); err != nil {
		return
	}

	var response []byte
	if response, err = postJSONBytesContext(ctx, address, jsonByte); err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &out, "CreateFundsBill")
	return
}

// BindTransferAccount 绑定广告金充值账户
func (s *VirtualPayment) BindTransferAccount(ctx context.Context, in *BindTransferAccountRequest) (out BindTransferAccountResponse, err error) {
	var jsonByte []byte
	if jsonByte, err = json.Marshal(in); err != nil {
		return
	}

	var (
		params = URLParams{
			Path:      bindTransferAccount,
			Content:   string(jsonByte),
			Signature: EmptyString,
		}
		address string
	)
	if address, err = s.requestAddress(params); err != nil {
		return
	}

	var response []byte
	if response, err = postJSONBytesContext(ctx, address, jsonByte); err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &out, "BindTransferAccount")
	return
}

// QueryFundsBill 查询广告金充值记录
func (s *VirtualPayment) QueryFundsBill(ctx context.Context, in *QueryFundsBillRequest) (out QueryFundsBillResponse, err error) {
	var jsonByte []byte
	if jsonByte, err = json.Marshal(in); err != nil {
		return
	}

	var (
		params = URLParams{
			Path:      queryFundsBill,
			Content:   string(jsonByte),
			Signature: EmptyString,
		}
		address string
	)
	if address, err = s.requestAddress(params); err != nil {
		return
	}

	var response []byte
	if response, err = postJSONBytesContext(ctx, address, jsonByte); err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &out, "QueryFundsBill")
	return
}

// QueryRecoverBill 查询广告金回收记录
func (s *VirtualPayment) QueryRecoverBill(ctx context.Context, in *QueryRecoverBillRequest) (out QueryRecoverBillResponse, err error) {
	var jsonByte []byte
	if jsonByte, err = json.Marshal(in); err != nil {
		return
	}

	var (
		params = URLParams{
			Path:      queryRecoverBill,
			Content:   string(jsonByte),
			Signature: EmptyString,
		}
		address string
	)
	if address, err = s.requestAddress(params); err != nil {
		return
	}

	var response []byte
	if response, err = postJSONBytesContext(ctx, address, jsonByte); err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &out, "QueryRecoverBill")
	return
}

// DownloadAdverFundsOrder 下载广告金对应的商户订单信息
func (s *VirtualPayment) DownloadAdverFundsOrder(ctx context.Context, in *DownloadAdverFundsOrderRequest) (out DownloadAdverFundsOrderResponse, err error) {
	var jsonByte []byte
	if jsonByte, err = json.Marshal(in); err != nil {
		return
	}

	var (
		params = URLParams{
			Path:      downloadAdverFundsOrder,
			Content:   string(jsonByte),
			Signature: EmptyString,
		}
		address string
	)
	if address, err = s.requestAddress(params); err != nil {
		return
	}

	var response []byte
	if response, err = postJSONBytesContext(ctx, address, jsonByte); err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &out, "DownloadAdverFundsOrder")
	return
}

// GetComplaintList 获取投诉列表
func (s *VirtualPayment) GetComplaintList(ctx context.Context, in *GetComplaintListRequest) (out GetComplaintListResponse, err error) {
	var jsonByte []byte
	if jsonByte, err = json.Marshal(in); err != nil {
		return
	}

	var (
		params = URLParams{
			Path:      getComplaintList,
			Content:   string(jsonByte),
			Signature: EmptyString,
		}
		address string
	)
	if address, err = s.requestAddress(params); err != nil {
		return
	}

	var response []byte
	if response, err = postJSONBytesContext(ctx, address, jsonByte); err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &out, "GetComplaintList")
	return
}

// GetComplaintDetail 获取投诉详情
func (s *VirtualPayment) GetComplaintDetail(ctx context.Context, in *GetComplaintDetailRequest) (out GetComplaintDetailResponse, err error) {
	var jsonByte []byte
	if jsonByte, err = json.Marshal(in); err != nil {
		return
	}

	var (
		params = URLParams{
			Path:      getComplaintDetail,
			Content:   string(jsonByte),
			Signature: EmptyString,
		}
		address string
	)
	if address, err = s.requestAddress(params); err != nil {
		return
	}

	var response []byte
	if response, err = postJSONBytesContext(ctx, address, jsonByte); err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &out, "GetComplaintDetail")
	return
}

// GetNegotiationHistory 获取协商历史
func (s *VirtualPayment) GetNegotiationHistory(ctx context.Context, in *GetNegotiationHistoryRequest) (out GetNegotiationHistoryResponse, err error) {
	var jsonByte []byte
	if jsonByte, err = json.Marshal(in); err != nil {
		return
	}

	var (
		params = URLParams{
			Path:      getNegotiationHistory,
			Content:   string(jsonByte),
			Signature: EmptyString,
		}
		address string
	)
	if address, err = s.requestAddress(params); err != nil {
		return
	}

	var response []byte
	if response, err = postJSONBytesContext(ctx, address, jsonByte); err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &out, "GetNegotiationHistory")
	return
}

// ResponseComplaint 回复用户
func (s *VirtualPayment) ResponseComplaint(ctx context.Context, in *ResponseComplaintRequest) (out ResponseComplaintResponse, err error) {
	var jsonByte []byte
	if jsonByte, err = json.Marshal(in); err != nil {
		return
	}

	var (
		params = URLParams{
			Path:      responseComplaint,
			Content:   string(jsonByte),
			Signature: EmptyString,
		}
		address string
	)
	if address, err = s.requestAddress(params); err != nil {
		return
	}

	var response []byte
	if response, err = postJSONBytesContext(ctx, address, jsonByte); err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &out, "ResponseComplaint")
	return
}

// CompleteComplaint 完成投诉处理
func (s *VirtualPayment) CompleteComplaint(ctx context.Context, in *CompleteComplaintRequest) (out CompleteComplaintResponse, err error) {
	var jsonByte []byte
	if jsonByte, err = json.Marshal(in); err != nil {
		return
	}

	var (
		params = URLParams{
			Path:      completeComplaint,
			Content:   string(jsonByte),
			Signature: EmptyString,
		}
		address string
	)
	if address, err = s.requestAddress(params); err != nil {
		return
	}

	var response []byte
	if response, err = postJSONBytesContext(ctx, address, jsonByte); err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &out, "CompleteComplaint")
	return
}

// UploadVPFile 上传媒体文件
func (s *VirtualPayment) UploadVPFile(ctx context.Context, in *UploadVPFileRequest) (out UploadVPFileResponse, err error) {
	var jsonByte []byte
	if jsonByte, err = json.Marshal(in); err != nil {
		return
	}

	var (
		params = URLParams{
			Path:      uploadVPFile,
			Content:   string(jsonByte),
			Signature: EmptyString,
		}
		address string
	)
	if address, err = s.requestAddress(params); err != nil {
		return
	}

	var response []byte
	if response, err = postJSONBytesContext(ctx, address, jsonByte); err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &out, "UploadVPFile")
	return
}

// GetUploadFileSign 获取微信支付反馈投诉图片的签名头部
func (s *VirtualPayment) GetUploadFileSign(ctx context.Context, in *GetUploadFileSignRequest) (out GetUploadFileSignResponse, err error) {
	var jsonByte []byte
	if jsonByte, err = json.Marshal(in); err != nil {
		return
	}

	var (
		params = URLParams{
			Path:      getUploadFileSign,
			Content:   string(jsonByte),
			Signature: EmptyString,
		}
		address string
	)
	if address, err = s.requestAddress(params); err != nil {
		return
	}

	var response []byte
	if response, err = postJSONBytesContext(ctx, address, jsonByte); err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &out, "GetUploadFileSign")
	return
}

// hmacSha256 hmac sha256
func (s *VirtualPayment) hmacSha256(key, data string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// postJSONBytesContext POST JSON bytes with context
func postJSONBytesContext(ctx context.Context, address string, jsonByte []byte) ([]byte, error) {
	return util.HTTPPostContext(ctx, address, jsonByte, map[string]string{
		"Content-Type": "application/json;charset=utf-8",
	})
}

// PaySign pay sign
func (s *VirtualPayment) PaySign(url, data string) (string, error) {
	if strings.TrimSpace(s.ctx.Config.AppKey) == "" {
		return "", errors.New("appKey is empty")
	}
	return s.hmacSha256(s.ctx.Config.AppKey, url+"&"+data), nil
}

// Signature user signature
func (s *VirtualPayment) Signature(data string) (string, error) {
	if strings.TrimSpace(s.sessionKey) == "" {
		return "", errors.New("sessionKey is empty")
	}
	return s.hmacSha256(s.sessionKey, data), nil
}

// PaySignature pay sign and signature
func (s *VirtualPayment) PaySignature(url, data string) (paySign, signature string, err error) {
	if paySign, err = s.PaySign(url, data); err != nil {
		return
	}
	if signature, err = s.Signature(data); err != nil {
		return
	}
	return
}

// requestURL .组合 URL
func (s *VirtualPayment) requestAddress(params URLParams) (url string, err error) {
	switch params.Path {
	case queryUserBalance, currencyPay, cancelCurrencyPay:
		if params.PaySign, params.Signature, err = s.PaySignature(params.Path, params.Content); err != nil {
			return
		}
	case queryOrder, notifyProvideGoods, presentCurrency, downloadBill, refundOrder,
		createWithdrawOrder, queryWithdrawOrder, startUploadGoods, queryUploadGoods,
		startPublishGoods, queryPublishGoods, startDownloadOrder, queryDownloadOrder,
		queryBizBalance, queryTransferAccount, queryAdverFunds, createFundsBill,
		bindTransferAccount, queryFundsBill, queryRecoverBill, downloadAdverFundsOrder,
		getComplaintList, getComplaintDetail, getNegotiationHistory, responseComplaint,
		completeComplaint, uploadVPFile, getUploadFileSign:
		if params.PaySign, err = s.PaySign(params.Path, params.Content); err != nil {
			return
		}
	default:
		err = errors.New("path is not exist")
		return
	}

	if params.AccessToken, err = s.ctx.GetAccessToken(); err != nil {
		return
	}

	url = baseSite + params.Path + "?" + accessToken + "=" + params.AccessToken
	if params.PaySign != EmptyString {
		url += "&" + paySignature + "=" + params.PaySign
	}
	if params.Signature != EmptyString {
		url += "&" + signature + "=" + params.Signature
	}
	return
}
