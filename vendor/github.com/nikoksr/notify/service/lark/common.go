package lark

// sender is an interface for sending a message to an already defined receiver.
type sender interface {
	Send(subject, message string) error
}

// sender is an interface for sending a message to a specific receiver ID.
type sendToer interface {
	SendTo(subject, message, id, idType string) error
}

// ReceiverID encapsulates a receiver ID and its type in Lark.
type ReceiverID struct {
	id  string
	typ receiverIDType
}

// OpenID specifies an ID as a Lark Open ID.
func OpenID(s string) *ReceiverID {
	return &ReceiverID{s, openID}
}

// UserID specifies an ID as a Lark User ID.
func UserID(s string) *ReceiverID {
	return &ReceiverID{s, userID}
}

// UnionID specifies an ID as a Lark Union ID.
func UnionID(s string) *ReceiverID {
	return &ReceiverID{s, unionID}
}

// Email specifies a receiver ID as an email.
func Email(s string) *ReceiverID {
	return &ReceiverID{s, email}
}

// ChatID specifies an ID as a Lark Chat ID.
func ChatID(s string) *ReceiverID {
	return &ReceiverID{s, chatID}
}

// receiverIDType represents the different ID types implemented by Lark. This
// information is required when sending a message. More information about the
// different ID types can be found in the "Query parameters" section of
// the https://open.larksuite.com/document/uAjLw4CM/ukTMukTMukTM/reference/im-v1/message/create,
// or on
// https://open.larksuite.com/document/home/user-identity-introduction/introduction.
type receiverIDType string

const (
	openID  receiverIDType = "open_id"
	userID  receiverIDType = "user_id"
	unionID receiverIDType = "union_id"
	email   receiverIDType = "email"
	chatID  receiverIDType = "chat_id"
)
