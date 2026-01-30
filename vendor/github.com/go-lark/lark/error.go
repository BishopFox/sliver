package lark

import "errors"

// Errors
var (
	ErrBotTypeError           = errors.New("Bot type error")
	ErrParamUserID            = errors.New("Param error: UserID")
	ErrParamMessageID         = errors.New("Param error: Message ID")
	ErrParamExceedInputLimit  = errors.New("Param error: Exceed input limit")
	ErrMessageTypeNotSuppored = errors.New("Message type not supported")
	ErrEncryptionNotEnabled   = errors.New("Encryption is not enabled")
	ErrCustomHTTPClientNotSet = errors.New("Custom HTTP client not set")
	ErrMessageNotBuild        = errors.New("Message not build")
	ErrUnsupportedUIDType     = errors.New("Unsupported UID type")
	ErrInvalidReceiveID       = errors.New("Invalid receive ID")
	ErrEventTypeNotMatch      = errors.New("Event type not match")
	ErrMessageType            = errors.New("Message type error")
)
