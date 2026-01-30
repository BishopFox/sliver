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
	if response, err = util.PostJSONContext(ctx, address, in); err != nil {
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
	if response, err = util.PostJSONContext(ctx, address, in); err != nil {
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
	if response, err = util.PostJSONContext(ctx, address, in); err != nil {
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
	if response, err = util.PostJSONContext(ctx, address, in); err != nil {
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
	if response, err = util.PostJSONContext(ctx, address, in); err != nil {
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
	if response, err = util.PostJSONContext(ctx, address, in); err != nil {
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
	if response, err = util.PostJSONContext(ctx, address, in); err != nil {
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
	if response, err = util.PostJSONContext(ctx, address, in); err != nil {
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
	if response, err = util.PostJSONContext(ctx, address, in); err != nil {
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
	if response, err = util.PostJSONContext(ctx, address, in); err != nil {
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
	if response, err = util.PostJSONContext(ctx, address, in); err != nil {
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
	if response, err = util.PostJSONContext(ctx, address, in); err != nil {
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
	if response, err = util.PostJSONContext(ctx, address, in); err != nil {
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
	if response, err = util.PostJSONContext(ctx, address, in); err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &out, "QueryPublishGoods")
	return
}

// hmacSha256 hmac sha256
func (s *VirtualPayment) hmacSha256(key, data string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
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
	case queryUserBalance:
	case currencyPay:
	case cancelCurrencyPay:
		if params.PaySign, params.Signature, err = s.PaySignature(params.Path, params.Content); err != nil {
			return
		}
	case queryOrder:
	case notifyProvideGoods:
	case presentCurrency:
	case downloadBill:
	case refundOrder:
	case createWithdrawOrder:
	case queryWithdrawOrder:
	case startUploadGoods:
	case queryUploadGoods:
	case startPublishGoods:
	case queryPublishGoods:
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
