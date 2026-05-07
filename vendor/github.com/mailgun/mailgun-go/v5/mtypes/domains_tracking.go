package mtypes

type DomainTrackingResponse struct {
	Tracking DomainTracking `json:"tracking"`
}

// Specify the domain tracking options
type DomainTracking struct {
	Click       TrackingStatus `json:"click"`
	Open        TrackingStatus `json:"open"`
	Unsubscribe TrackingStatus `json:"unsubscribe"`
}

// TrackingStatus is the tracking status of a domain
type TrackingStatus struct {
	Active     bool   `json:"active"`
	HTMLFooter string `json:"html_footer"`
	TextFooter string `json:"text_footer"`
}
