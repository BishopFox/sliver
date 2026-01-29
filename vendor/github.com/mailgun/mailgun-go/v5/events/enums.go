package events

const (
	EventAccepted              = "accepted"
	EventRejected              = "rejected"
	EventDelivered             = "delivered"
	EventFailed                = "failed"
	EventOpened                = "opened"
	EventClicked               = "clicked"
	EventUnsubscribed          = "unsubscribed"
	EventComplained            = "complained"
	EventStored                = "stored"
	EventDropped               = "dropped"
	EventListMemberUploaded    = "list_member_uploaded"
	EventListMemberUploadError = "list_member_upload_error"
	EventListUploaded          = "list_uploaded"
)

const (
	TransportHTTP = "http"
	TransportSMTP = "smtp"

	DeviceUnknown       = "unknown"
	DeviceMobileBrowser = "desktop"
	DeviceBrowser       = "mobile"
	DeviceEmail         = "tablet"
	DeviceOther         = "other"

	ClientUnknown       = "unknown"
	ClientMobileBrowser = "mobile browser"
	ClientBrowser       = "browser"
	ClientEmail         = "email client"
	ClientLibrary       = "library"
	ClientRobot         = "robot"
	ClientOther         = "other"

	ReasonUnknown             = "unknown"
	ReasonGeneric             = "generic"
	ReasonBounce              = "bounce"
	ReasonESPBlock            = "espblock"
	ReasonGreylisted          = "greylisted"
	ReasonBlacklisted         = "blacklisted"
	ReasonSuppressBounce      = "suppress-bounce"
	ReasonSuppressComplaint   = "suppress-complaint"
	ReasonSuppressUnsubscribe = "suppress-unsubscribe"
	ReasonOld                 = "old"
	ReasonHardFail            = "hardfail"

	SeverityUnknown   = "unknown"
	SeverityTemporary = "temporary"
	SeverityPermanent = "permanent"
	SeverityInternal  = "internal"

	MethodUnknown = "unknown"
	MethodSMTP    = "smtp"
	MethodHTTP    = "http"
)
