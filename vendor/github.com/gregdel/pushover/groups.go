package pushover

// Group represents a Pushover Delivery Group.
// ref: https://pushover.net/api/groups
type Group struct {
	ID   string `json:"group"`
	Name string `json:"name"`
}

// GroupsListResponse is the response from a group list request.
type GroupsListResponse struct {
	Status    int     `json:"status"`
	RequestID string  `json:"request"`
	Errors    Errors  `json:"errors"`
	Groups    []Group `json:"groups"`
}

// GroupDetailsResponse contains the details of a group that was requested.
type GroupDetailsResponse struct {
	Status    int    `json:"status"`
	RequestID string `json:"request"`
	Name      string `json:"name"`
	Users     []struct {
		User     string  `json:"user"`
		Device   *string `json:"device"`
		Memo     string  `json:"memo"`
		Disabled bool    `json:"disabled"`
	} `json:"users"`
}
