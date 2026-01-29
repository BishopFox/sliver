package twitter

// A Demux receives interface{} messages individually or from a channel and
// sends those messages to one or more outputs determined by the
// implementation.
type Demux interface {
	Handle(message interface{})
	HandleChan(messages <-chan interface{})
}

// SwitchDemux receives messages and uses a type switch to send each typed
// message to a handler function.
type SwitchDemux struct {
	All              func(message interface{})
	Tweet            func(tweet *Tweet)
	DM               func(dm *DirectMessage)
	StatusDeletion   func(deletion *StatusDeletion)
	LocationDeletion func(LocationDeletion *LocationDeletion)
	StreamLimit      func(limit *StreamLimit)
	StatusWithheld   func(statusWithheld *StatusWithheld)
	UserWithheld     func(userWithheld *UserWithheld)
	StreamDisconnect func(disconnect *StreamDisconnect)
	Warning          func(warning *StallWarning)
	FriendsList      func(friendsList *FriendsList)
	Event            func(event *Event)
	Other            func(message interface{})
}

// NewSwitchDemux returns a new SwitchMux which has NoOp handler functions.
func NewSwitchDemux() SwitchDemux {
	return SwitchDemux{
		All:              func(message interface{}) {},
		Tweet:            func(tweet *Tweet) {},
		DM:               func(dm *DirectMessage) {},
		StatusDeletion:   func(deletion *StatusDeletion) {},
		LocationDeletion: func(LocationDeletion *LocationDeletion) {},
		StreamLimit:      func(limit *StreamLimit) {},
		StatusWithheld:   func(statusWithheld *StatusWithheld) {},
		UserWithheld:     func(userWithheld *UserWithheld) {},
		StreamDisconnect: func(disconnect *StreamDisconnect) {},
		Warning:          func(warning *StallWarning) {},
		FriendsList:      func(friendsList *FriendsList) {},
		Event:            func(event *Event) {},
		Other:            func(message interface{}) {},
	}
}

// Handle determines the type of a message and calls the corresponding receiver
// function with the typed message. All messages are passed to the All func.
// Messages with unmatched types are passed to the Other func.
func (d SwitchDemux) Handle(message interface{}) {
	d.All(message)
	switch msg := message.(type) {
	case *Tweet:
		d.Tweet(msg)
	case *DirectMessage:
		d.DM(msg)
	case *StatusDeletion:
		d.StatusDeletion(msg)
	case *LocationDeletion:
		d.LocationDeletion(msg)
	case *StreamLimit:
		d.StreamLimit(msg)
	case *StatusWithheld:
		d.StatusWithheld(msg)
	case *UserWithheld:
		d.UserWithheld(msg)
	case *StreamDisconnect:
		d.StreamDisconnect(msg)
	case *StallWarning:
		d.Warning(msg)
	case *FriendsList:
		d.FriendsList(msg)
	case *Event:
		d.Event(msg)
	default:
		d.Other(msg)
	}
}

// HandleChan receives messages and calls the corresponding receiver function
// with the typed message. All messages are passed to the All func. Messages
// with unmatched type are passed to the Other func.
func (d SwitchDemux) HandleChan(messages <-chan interface{}) {
	for message := range messages {
		d.Handle(message)
	}
}
