package twitter

// StatusDeletion indicates that a given Tweet has been deleted.
// https://dev.twitter.com/streaming/overview/messages-types#status_deletion_notices_delete
type StatusDeletion struct {
	ID        int64  `json:"id"`
	IDStr     string `json:"id_str"`
	UserID    int64  `json:"user_id"`
	UserIDStr string `json:"user_id_str"`
}

type statusDeletionNotice struct {
	Delete struct {
		StatusDeletion *StatusDeletion `json:"status"`
	} `json:"delete"`
}

// LocationDeletion indicates geolocation data must be stripped from a range
// of Tweets.
// https://dev.twitter.com/streaming/overview/messages-types#Location_deletion_notices_scrub_geo
type LocationDeletion struct {
	UserID          int64  `json:"user_id"`
	UserIDStr       string `json:"user_id_str"`
	UpToStatusID    int64  `json:"up_to_status_id"`
	UpToStatusIDStr string `json:"up_to_status_id_str"`
}

type locationDeletionNotice struct {
	ScrubGeo *LocationDeletion `json:"scrub_geo"`
}

// StreamLimit indicates a stream matched more statuses than its rate limit
// allowed. The track number is the number of undelivered matches.
// https://dev.twitter.com/streaming/overview/messages-types#limit_notices
type StreamLimit struct {
	Track int64 `json:"track"`
}

type streamLimitNotice struct {
	Limit *StreamLimit `json:"limit"`
}

// StatusWithheld indicates a Tweet with the given ID, belonging to UserId,
// has been withheld in certain countries.
// https://dev.twitter.com/streaming/overview/messages-types#withheld_content_notices
type StatusWithheld struct {
	ID                  int64    `json:"id"`
	UserID              int64    `json:"user_id"`
	WithheldInCountries []string `json:"withheld_in_countries"`
}

type statusWithheldNotice struct {
	StatusWithheld *StatusWithheld `json:"status_withheld"`
}

// UserWithheld indicates a User with the given ID has been withheld in
// certain countries.
// https://dev.twitter.com/streaming/overview/messages-types#withheld_content_notices
type UserWithheld struct {
	ID                  int64    `json:"id"`
	WithheldInCountries []string `json:"withheld_in_countries"`
}
type userWithheldNotice struct {
	UserWithheld *UserWithheld `json:"user_withheld"`
}

// StreamDisconnect indicates the stream has been shutdown for some reason.
// https://dev.twitter.com/streaming/overview/messages-types#disconnect_messages
type StreamDisconnect struct {
	Code       int64  `json:"code"`
	StreamName string `json:"stream_name"`
	Reason     string `json:"reason"`
}

type streamDisconnectNotice struct {
	StreamDisconnect *StreamDisconnect `json:"disconnect"`
}

// StallWarning indicates the client is falling behind in the stream.
// https://dev.twitter.com/streaming/overview/messages-types#stall_warnings
type StallWarning struct {
	Code        string `json:"code"`
	Message     string `json:"message"`
	PercentFull int    `json:"percent_full"`
}

type stallWarningNotice struct {
	StallWarning *StallWarning `json:"warning"`
}

// FriendsList is a list of some of a user's friends.
// https://dev.twitter.com/streaming/overview/messages-types#friends_list_friends
type FriendsList struct {
	Friends []int64 `json:"friends"`
}

type directMessageNotice struct {
	DirectMessage *DirectMessage `json:"direct_message"`
}

// Event is a non-Tweet notification message (e.g. like, retweet, follow).
// https://dev.twitter.com/streaming/overview/messages-types#Events_event
type Event struct {
	Event     string `json:"event"`
	CreatedAt string `json:"created_at"`
	Target    *User  `json:"target"`
	Source    *User  `json:"source"`
	// TODO: add List or deprecate it
	TargetObject *Tweet `json:"target_object"`
}
