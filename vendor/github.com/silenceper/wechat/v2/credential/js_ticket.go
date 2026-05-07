package credential

import context2 "context"

// JsTicketHandle js ticket获取
type JsTicketHandle interface {
	// GetTicket 获取ticket
	GetTicket(accessToken string) (ticket string, err error)
}

// JsTicketContextHandle js ticket获取
type JsTicketContextHandle interface {
	JsTicketHandle
	GetTicketContext(ctx context2.Context, accessToken string) (ticket string, err error)
}
