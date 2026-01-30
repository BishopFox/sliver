package invoice

import (
	"fmt"

	"github.com/silenceper/wechat/v2/util"
)

const (
	// getInvoiceInfoURL 查询电子发票
	getInvoiceInfoURL = "https://qyapi.weixin.qq.com/cgi-bin/card/invoice/reimburse/getinvoiceinfo?access_token=%s"
	// updateInvoiceStatusURL 更新发票状态
	updateInvoiceStatusURL = "https://qyapi.weixin.qq.com/cgi-bin/card/invoice/reimburse/updateinvoicestatus?access_token=%s"
	// updateStatusBatchURL 批量更新发票状态
	updateStatusBatchURL = "https://qyapi.weixin.qq.com/cgi-bin/card/invoice/reimburse/updatestatusbatch?access_token=%s"
	// getInvoiceInfoBatchURL 批量查询电子发票
	getInvoiceInfoBatchURL = "https://qyapi.weixin.qq.com/cgi-bin/card/invoice/reimburse/getinvoiceinfobatch?access_token=%s"
)

// GetInvoiceInfoRequest 查询电子发票请求
type GetInvoiceInfoRequest struct {
	CardID      string `json:"card_id"`
	EncryptCode string `json:"encrypt_code"`
}

// GetInvoiceInfoResponse 查询电子发票响应
type GetInvoiceInfoResponse struct {
	util.CommonError
	CardID    string   `json:"card_id"`
	BeginTime int64    `json:"begin_time"`
	EndTime   int64    `json:"end_time"`
	OpenID    string   `json:"openid"`
	Type      string   `json:"type"`
	Payee     string   `json:"payee"`
	Detail    string   `json:"detail"`
	UserInfo  UserInfo `json:"user_info"`
}

// UserInfo 发票的用户信息
type UserInfo struct {
	Fee                   int64  `json:"fee"`
	Title                 string `json:"title"`
	BillingTime           int64  `json:"billing_time"`
	BillingNo             string `json:"billing_no"`
	BillingCode           string `json:"billing_code"`
	Info                  []Info `json:"info"`
	FeeWithoutTax         int64  `json:"fee_without_tax"`
	Tax                   int64  `json:"tax"`
	Detail                string `json:"detail"`
	PdfURL                string `json:"pdf_url"`
	TripPdfURL            string `json:"trip_pdf_url"`
	ReimburseStatus       string `json:"reimburse_status"`
	CheckCode             string `json:"check_code"`
	BuyerNumber           string `json:"buyer_number"`
	BuyerAddressAndPhone  string `json:"buyer_address_and_phone"`
	BuyerBankAccount      string `json:"buyer_bank_account"`
	SellerNumber          string `json:"seller_number"`
	SellerAddressAndPhone string `json:"seller_address_and_phone"`
	SellerBankAccount     string `json:"seller_bank_account"`
	Remarks               string `json:"remarks"`
	Cashier               string `json:"cashier"`
	Maker                 string `json:"maker"`
}

// Info 商品信息结构
type Info struct {
	Name  string `json:"name"`
	Num   int64  `json:"num"`
	Unit  string `json:"unit"`
	Fee   int64  `json:"fee"`
	Price int64  `json:"price"`
}

// GetInvoiceInfo 查询电子发票
// see https://developer.work.weixin.qq.com/document/path/90284
func (r *Client) GetInvoiceInfo(req *GetInvoiceInfoRequest) (*GetInvoiceInfoResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(getInvoiceInfoURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &GetInvoiceInfoResponse{}
	err = util.DecodeWithError(response, result, "GetInvoiceInfo")
	return result, err
}

// UpdateInvoiceStatusRequest 更新发票状态请求
type UpdateInvoiceStatusRequest struct {
	CardID          string `json:"card_id"`
	EncryptCode     string `json:"encrypt_code"`
	ReimburseStatus string `json:"reimburse_status"`
}

// UpdateInvoiceStatus 更新发票状态
// see https://developer.work.weixin.qq.com/document/path/90285
func (r *Client) UpdateInvoiceStatus(req *UpdateInvoiceStatusRequest) error {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(updateInvoiceStatusURL, accessToken), req); err != nil {
		return err
	}
	return util.DecodeWithCommonError(response, "UpdateInvoiceStatus")
}

// UpdateStatusBatchRequest 批量更新发票状态
type UpdateStatusBatchRequest struct {
	OpenID          string    `json:"openid"`
	ReimburseStatus string    `json:"reimburse_status"`
	InvoiceList     []Invoice `json:"invoice_list"`
}

// Invoice 发票卡券
type Invoice struct {
	CardID      string `json:"card_id"`
	EncryptCode string `json:"encrypt_code"`
}

// UpdateStatusBatch 批量更新发票状态
// see https://developer.work.weixin.qq.com/document/path/90286
func (r *Client) UpdateStatusBatch(req *UpdateStatusBatchRequest) error {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(updateStatusBatchURL, accessToken), req); err != nil {
		return err
	}
	return util.DecodeWithCommonError(response, "UpdateStatusBatch")
}

// GetInvoiceInfoBatchRequest 批量查询电子发票请求
type GetInvoiceInfoBatchRequest struct {
	ItemList []Invoice `json:"item_list"`
}

// GetInvoiceInfoBatchResponse 批量查询电子发票响应
type GetInvoiceInfoBatchResponse struct {
	util.CommonError
	ItemList []Item `json:"item_list"`
}

// Item 电子发票的结构化信息
type Item struct {
	CardID    string   `json:"card_id"`
	BeginTime int64    `json:"begin_time"`
	EndTime   int64    `json:"end_time"`
	OpenID    string   `json:"openid"`
	Type      string   `json:"type"`
	Payee     string   `json:"payee"`
	Detail    string   `json:"detail"`
	UserInfo  UserInfo `json:"user_info"`
}

// GetInvoiceInfoBatch 批量查询电子发票
// see https://developer.work.weixin.qq.com/document/path/90287
func (r *Client) GetInvoiceInfoBatch(req *GetInvoiceInfoBatchRequest) (*GetInvoiceInfoBatchResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(getInvoiceInfoBatchURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &GetInvoiceInfoBatchResponse{}
	err = util.DecodeWithError(response, result, "GetInvoiceInfoBatch")
	return result, err
}
