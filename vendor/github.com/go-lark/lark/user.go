package lark

// UID types
const (
	UIDEmail   = "email"
	UIDUserID  = "user_id"
	UIDOpenID  = "open_id"
	UIDChatID  = "chat_id"
	UIDUnionID = "union_id"
)

// OptionalUserID to contain openID, chatID, userID, email
type OptionalUserID struct {
	UIDType string
	RealID  string
}

func withOneID(uidType, realID string) *OptionalUserID {
	return &OptionalUserID{
		UIDType: uidType,
		RealID:  realID,
	}
}

// WithEmail uses email as userID
func WithEmail(email string) *OptionalUserID {
	return withOneID(UIDEmail, email)
}

// WithUserID uses userID as userID
func WithUserID(userID string) *OptionalUserID {
	return withOneID(UIDUserID, userID)
}

// WithOpenID uses openID as userID
func WithOpenID(openID string) *OptionalUserID {
	return withOneID(UIDOpenID, openID)
}

// WithChatID uses chatID as userID
func WithChatID(chatID string) *OptionalUserID {
	return withOneID(UIDChatID, chatID)
}

// WithUnionID uses chatID as userID
func WithUnionID(unionID string) *OptionalUserID {
	return withOneID(UIDUnionID, unionID)
}
